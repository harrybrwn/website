package migrations

import (
	"embed"
	"fmt"
	"io/fs"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sirupsen/logrus"
)

var (
	//go:embed api hooks data-analyst-roadmap_*
	all    embed.FS
	logger logrus.FieldLogger = logrus.StandardLogger()
)

func RunAll(logger logrus.FieldLogger, u *url.URL) error {
	entries, err := all.ReadDir(".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub, err := fs.Sub(all, e.Name())
		if err != nil {
			return err
		}
		err = up(logger, u, sub, e.Name())
		if err != nil {
			return err
		}
	}
	return nil
}

func Run(logger logrus.FieldLogger, name string, u *url.URL) error {
	return up(logger, u, all, name)
}

func Get(name string, db *url.URL) (*migrate.Migrate, error) {
	m, err := newMigration(db, all, name)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func List(db *url.URL) ([]*migrate.Migrate, error) {
	entries, err := all.ReadDir(".")
	if err != nil {
		return nil, err
	}
	res := make([]*migrate.Migrate, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		fmt.Println(e.Name())
		m, err := Get(e.Name(), db)
		if err != nil {
			return nil, err
		}
		res = append(res, m)
	}
	return res, nil
}

func up(logger logrus.FieldLogger, db *url.URL, fsys fs.FS, dir string) error {
	m, err := newMigration(db, fsys, dir)
	if err != nil {
		return err
	}
	SetLogger(m, logger)
	return m.Up()
}

func newMigration(db *url.URL, fsys fs.FS, dir string) (*migrate.Migrate, error) {
	source, err := iofs.New(fsys, dir)
	if err != nil {
		return nil, err
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, db.String())
	if err != nil {
		return nil, err
	}
	SetLogger(m, logger) // use this package's logger by default
	return m, nil
}

func SetLogger(m *migrate.Migrate, logger logrus.FieldLogger) {
	m.Log = &migrateLogger{FieldLogger: logger, verbose: true}
}

type migrateLogger struct {
	logrus.FieldLogger
	verbose bool
}

func (l *migrateLogger) Verbose() bool {
	return l.verbose
}
