package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
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

type PaginationOpts struct {
	Prev   int
	Offset int
	Limit  int
}

// Datastores connects to all the datastores in parallel for faster cold starts.
func Datastores(logger logrus.FieldLogger) (*database, *redis.Client, error) {
	var (
		wg   sync.WaitGroup
		errs = make(chan error)
		db   *database
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

func postgresConnectString() (string, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		return dbURL, nil
	}
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	pw := os.Getenv("POSTGRES_PASSWORD")
	db := os.Getenv("POSTGRES_DB")
	var userinfo *url.Userinfo
	if len(user) > 0 && len(pw) > 0 {
		userinfo = url.UserPassword(user, pw)
	}
	if len(port) == 0 {
		port = "5432"
	}
	if len(host) == 0 {
		return "", errors.New("no database host")
	}
	u := url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(host, port),
		User:     userinfo,
		Path:     filepath.Join("/", db),
		RawQuery: "sslmode=disable",
	}
	return u.String(), nil
}

func Connect(logger logrus.FieldLogger) (*database, error) {
	os.Unsetenv("PGSERVICEFILE")
	os.Unsetenv("PGSERVICE")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	// url := os.ExpandEnv(os.Getenv("DATABASE_URL"))
	url, err := postgresConnectString()
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "invalid database url"))
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, errors.WithStack(err)
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

func redisOpts() (*redis.Options, error) {
	fullurl, ok := lookupAnyOf("REDIS_URL", "REDIS_TLS_URL")
	if ok {
		return redis.ParseURL(fullurl)
	}
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	pw := os.Getenv("REDIS_PASSWORD")
	if len(pw) == 0 {
		return nil, errors.New("no password given for redis")
	}
	return &redis.Options{
		Addr:     net.JoinHostPort(host, port),
		Password: pw,
	}, nil
}

func DialRedis(logger logrus.FieldLogger) (*redis.Client, error) {
	ctx := context.Background()
	opts, err := redisOpts()
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

func S3CredentialsProvider() (credentials.Provider, error) {
	value, err := S3CredentialsValue()
	if err != nil {
		return nil, err
	}
	return &credentials.StaticProvider{Value: *value}, nil
}

func S3CredentialsValue() (*credentials.Value, error) {
	const (
		envAccessKey = "S3_ACCESS_KEY"
		envSecretKey = "S3_SECRET_KEY"
	)
	access := os.Getenv(envAccessKey)
	secret := os.Getenv(envSecretKey)
	if len(access) == 0 {
		return nil, fmt.Errorf("s3 access key not found in %q", envAccessKey)
	}
	if len(secret) == 0 {
		return nil, fmt.Errorf("s3 secret key not found in %q", envSecretKey)
	}
	return &credentials.Value{
		AccessKeyID:     access,
		SecretAccessKey: secret,
		SessionToken:    "",
	}, nil
}
