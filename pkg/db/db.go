package db

import (
	"database/sql"
	"errors"
	"os"
)

func Connect() (*sql.DB, error) {
	os.Unsetenv("PGSERVICEFILE")
	os.Unsetenv("PGSERVICE")
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, errors.New("empty $DATABASE_URL")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
