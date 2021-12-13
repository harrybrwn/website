package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"
)

func Connect() (*sql.DB, error) {
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
			err = db.Ping()
			if err == nil {
				return db, nil
			}
		case <-ctx.Done():
			return nil, errors.New("database ping timeout")
		}
	}
}
