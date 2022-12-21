package provision

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

type DBConfig struct {
	Host      string    `json:"host" yaml:"host" hcl:"host"`
	Port      string    `json:"port" yaml:"port" hcl:"port,optional"`
	RootUser  string    `json:"root_user" yaml:"root_user" hcl:"root_user"`
	Password  string    `json:"password" yaml:"password" hcl:"password"`
	Users     []*DBUser `hcl:"user,block"`
	Databases []*struct {
		Name  string `hcl:"name,label"`
		Owner string `hcl:"owner"`
	} `hcl:"database,block"`
	Migrations map[string]Migration `json:"migrations"`
}

type Migration struct {
	Database string `json:"database"`
	Source   string `json:"source"`
}

func (db *DBConfig) Init() {
	if db.Host == "" {
		db.Host = os.Getenv("POSTGRES_HOST")
	}
	if db.Port == "" {
		db.Port = os.Getenv("POSTGRES_PORT")
	}
	if db.RootUser == "" {
		db.RootUser = os.Getenv("POSTGRES_USER")
	}
	if db.Password == "" {
		db.Password = os.Getenv("POSTGRES_PASSWORD")
	}
}

func (db *DBConfig) Validate() error {
	return validateDBConfig(db)
}

func (db *DBConfig) Defaults() {
	if db.Host == "" {
		db.Host = "localhost"
	}
	if db.Port == "" {
		db.Port = "5432"
	}
}

func (db *DBConfig) URI(path ...string) *url.URL {
	paths := append([]string{"/"}, path...)
	return &url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(db.Host, db.Port),
		User:     url.UserPassword(db.RootUser, db.Password),
		Path:     filepath.Join(paths...),
		RawQuery: "sslmode=disable",
	}
}

type DBUser struct {
	Name       string `hcl:"name,label"`
	Password   string `hcl:"password"`
	SuperUser  bool   `hcl:"superuser,optional"`
	CreateDB   bool   `hcl:"create_db,optional"`
	CreateRole bool   `hcl:"create_role,optional"`
	Grants     struct {
		// Map of database name to grants
		Database map[string][]string `hcl:"database"`
		// Map of table name to grants
		Table map[string][]string `hcl:"table"`
	} `hcl:"grants,block"`
}

const (
	// postgres errors codes
	pqDuplicateObject   = "42710"
	pqDuplicateDatabase = "42P04"
)

func (db *DBConfig) Provision(ctx context.Context) error {
	os.Unsetenv("PGSERVICEFILE")
	os.Unsetenv("PGSERVICE")
	d, err := sql.Open("postgres", db.URI().String())
	if err != nil {
		return err
	}
	defer d.Close()

	if err = db.Validate(); err != nil {
		return err
	}

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

	for _, user := range db.Users {
		for db, grants := range user.Grants.Database {
			for _, grant := range grants {
				query := fmt.Sprintf(
					`GRANT %s ON DATABASE "%s" TO %s`,
					grant,
					db,
					user.Name,
				)
				_, err = d.ExecContext(ctx, query)
				if err != nil {
					return err
				}
			}
		}
		for table, grants := range user.Grants.Table {
			for _, grant := range grants {
				query := fmt.Sprintf(
					`GRANT %s ON TABLE "%s" TO %s`,
					grant,
					table,
					user.Name,
				)
				_, err = d.ExecContext(ctx, query)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

var (
	validTableGrants = map[string]struct{}{
		"SELECT":     {},
		"INSERT":     {},
		"UPDATE":     {},
		"DELETE":     {},
		"TRUNCATE":   {},
		"REFERENCES": {},
		"TRIGGER":    {},
		"ALL":        {},
	}
	validDatabaseGrants = map[string]struct{}{
		"CREATE":    {},
		"CONNECT":   {},
		"TEMPORARY": {},
		"TEMP":      {},
		"ALL":       {},
	}
)

func validateDBConfig(cfg *DBConfig) error {
	var (
		err error
	)

	for _, user := range cfg.Users {
		err = validateDBUser(user)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateDBUser(u *DBUser) error {
	for db, grants := range u.Grants.Database {
		if db == "" {
			return errors.New("empty database name")
		}
		for _, g := range grants {
			g = strings.ToUpper(g)
			_, ok := validDatabaseGrants[g]
			if !ok {
				return fmt.Errorf("grant privilege %q is not valid", g)
			}
		}
	}
	for table, grants := range u.Grants.Table {
		if table == "" {
			return errors.New("empty table name")
		}
		for _, g := range grants {
			g = strings.ToUpper(g)
			_, ok := validTableGrants[g]
			if !ok {
				return fmt.Errorf("grant privilege %q is not valid", g)
			}
		}
	}
	return nil
}
