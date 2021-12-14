package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"
)

type logger interface {
	Info(...interface{})
}

func Connect(loggers ...logger) (*sql.DB, error) {
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
		return db, nil
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
				return db, nil
			}
		case <-ctx.Done():
			return nil, errors.New("database ping timeout")
		}
	}
}

type lg struct{}

func (*lg) Info(...interface{}) {}
