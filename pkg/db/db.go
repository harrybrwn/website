package db

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

type DB interface {
	io.Closer
	QueryContext(context.Context, string, ...interface{}) (Rows, error)
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}

type Scanner interface {
	Scan(...interface{}) error
}

type Rows interface {
	Scanner
	io.Closer
	Next() bool
	Err() error
}

func ScanOne(r Rows, dest ...interface{}) (err error) {
	if !r.Next() {
		if err = r.Err(); err != nil {
			r.Close()
			return err
		}
		r.Close()
		return sql.ErrNoRows
	}
	if err = r.Scan(dest...); err != nil {
		r.Close()
		return err
	}
	return r.Close()
}

type database struct{ *sql.DB }

func (db *database) QueryContext(ctx context.Context, query string, v ...interface{}) (Rows, error) {
	return db.DB.QueryContext(ctx, query, v...)
}

// Datastores connects to all the datastores in parallel for faster cold starts.
func Datastores(logger logrus.FieldLogger) (DB, *redis.Client, error) {
	var (
		wg   sync.WaitGroup
		errs = make(chan error)
		db   DB
		rd   *redis.Client
	)
	wg.Add(2)
	go func() {
		wg.Wait()
		close(errs)
	}()
	go func() {
		defer wg.Done()
		var err error
		db, err = Connect(logger)
		if err != nil {
			errs <- err
		}
	}()
	go func() {
		defer wg.Done()
		var err error
		rd, err = DialRedis(logger)
		if err != nil {
			errs <- err
		}
	}()
	return db, rd, <-errs
}

func Connect(logger logrus.FieldLogger) (DB, error) {
	os.Unsetenv("PGSERVICEFILE")
	os.Unsetenv("PGSERVICE")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	url := os.ExpandEnv(os.Getenv("DATABASE_URL"))
	if url == "" {
		return nil, errors.New("empty $DATABASE_URL")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err == nil {
		return &database{DB: db}, nil
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err = db.Ping()
			if err == nil {
				logger.Info("database connected")
				return &database{DB: db}, nil
			}
			logger.WithError(err).Warn("failed to ping database, retrying...")
		case <-ctx.Done():
			return nil, errors.New("database ping timeout")
		}
	}
}

func lookupAnyOf(keys ...string) (string, bool) {
	for _, k := range keys {
		res, ok := os.LookupEnv(k)
		if ok {
			return res, true
		}
	}
	return "", false
}

func DialRedis(logger logrus.FieldLogger) (*redis.Client, error) {
	ctx := context.Background()
	url, ok := lookupAnyOf("REDIS_URL", "REDIS_TLS_URL")
	if !ok {
		return nil, errors.New("$REDIS_URL not set")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)
	if err = client.Ping(ctx).Err(); err == nil {
		return client, nil
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err = client.Ping(ctx).Err()
			if err == nil || err == redis.Nil {
				logger.Info("redis client connected")
				return client, nil
			}
			logger.WithError(err).Warn("failed to ping redis, retrying...")
		case <-ctx.Done():
			return nil, errors.New("redis client dial timeout")
		}
	}
}
