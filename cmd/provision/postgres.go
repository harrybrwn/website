package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

type DBConfig struct {
	Host      string `json:"host" yaml:"host"`
	Port      string `json:"port" yaml:"port"`
	RootUser  string `json:"root_user" yaml:"root_user"`
	Password  string `json:"password" yaml:"password"`
	Users     []*DBUser
	Databases []*struct {
		Name  string
		Owner string
	}
}

type DBUser struct {
	Name       string
	Password   string
	SuperUser  bool
	CreateDB   bool
	CreateRole bool
}

func (db *DBConfig) init() {
	if db.Host == "" {
		db.Host = os.Getenv("POSTGRES_HOST")
	}
	if db.Port == "" {
		db.Port = os.Getenv("POSTGRES_PORT")
	}
	if db.Port == "" {
		db.Port = "5432"
	}
	if db.RootUser == "" {
		db.RootUser = os.Getenv("POSTGRES_USER")
	}
	if db.Password == "" {
		db.Password = os.Getenv("POSTGRES_PASSWORD")
	}
}

const (
	pqDuplicateObject   = "42710"
	pqDuplicateDatabase = "42P04"
)

func (db *DBConfig) Provision(ctx context.Context) error {
	os.Unsetenv("PGSERVICEFILE")
	os.Unsetenv("PGSERVICE")
	d, err := sql.Open("postgres", db.uri())
	if err != nil {
		return err
	}
	defer d.Close()

	for _, user := range db.Users {
		query := fmt.Sprintf(
			`CREATE ROLE "%s" WITH PASSWORD '%s' LOGIN `,
			user.Name,
			user.Password)
		if user.SuperUser {
			query += "SUPERUSER "
		}
		if user.CreateDB {
			query += "CREATEDB "
		}
		if user.CreateRole {
			query += "CREATEROLE "
		}
		_, err = d.ExecContext(ctx, query)
		switch e := err.(type) {
		case nil:
			continue
		case *pq.Error:
			if e.Code == pqDuplicateObject {
				continue
			}
			return e
		default:
			return err
		}
	}

	for _, database := range db.Databases {
		if database.Owner == "" {
			return errors.New("each database must have an owner")
		}
		query := fmt.Sprintf(
			`CREATE DATABASE "%s" OWNER '%s'`,
			database.Name,
			database.Owner)
		_, err := d.ExecContext(ctx, query)
		switch e := err.(type) {
		case nil:
			continue
		case *pq.Error:
			if e.Code == pqDuplicateDatabase {
				continue
			}
			return err
		default:
			return err
		}
	}
	return nil
}

func (db *DBConfig) uri() string {
	u := url.URL{
		Scheme:   "postgres",
		Host:     db.Host,
		User:     url.UserPassword(db.RootUser, db.Password),
		Path:     "/",
		RawQuery: "sslmode=disable",
	}
	return u.String()
}
