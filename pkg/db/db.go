package db

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"time"
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

type logger interface {
	Info(...interface{})
}

func Connect(loggers ...logger) (DB, error) {
	var logger logger = new(lg)
	if len(loggers) > 0 {
		logger = loggers[0]
	}
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
			logger.Info("connecting to database")
			err = db.Ping()
			if err == nil {
				logger.Info("database connected")
				return &database{DB: db}, nil
			}
		case <-ctx.Done():
			return nil, errors.New("database ping timeout")
		}
	}
}

type lg struct{}

func (*lg) Info(...interface{}) {}
