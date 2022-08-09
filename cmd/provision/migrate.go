package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"harrybrown.com/pkg/provision"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/aws_s3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
)

func NewMigrateCmd(cli *Cli) *cobra.Command {
	c := cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
	}
	c.AddCommand(
		NewMigrateUpCmd(cli),
		NewMigrateDownCmd(cli),
		NewMigrateNewCmd(cli),
	)
	return &c
}

func NewMigrateUpCmd(cli *Cli) *cobra.Command {
	os.Unsetenv("PGSERVICEFILE")
	os.Unsetenv("PGSERVICE")
	c := cobra.Command{
		Use:   "up <migration>",
		Short: "Run migrations up",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO add --all flag to loop through migrations
			m, n, err := getMigration(&cli.config.DB, args)
			if err != nil {
				return errors.Wrap(err, "failed to create new migration")
			}
			if n < 0 {
				err = m.Up()
			} else {
				err = m.Steps(int(n))
			}
			if err == migrate.ErrNoChange {
				return nil
			}
			return err
		},
	}
	return &c
}

func NewMigrateDownCmd(cli *Cli) *cobra.Command {
	c := cobra.Command{
		Use:   "down <migration>",
		Short: "Run migrations down",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, n, err := getMigration(&cli.config.DB, args)
			if err != nil {
				return errors.Wrap(err, "failed to create new migration")
			}
			if n < 0 {
				err = m.Down()
			} else {
				err = m.Steps(-int(n))
			}
			if err == migrate.ErrNoChange {
				return nil
			}
			return err
		},
	}
	return &c
}

const (
	defaultMigrationDigits = 6
	defaultExt             = ".sql"
)

func NewMigrateNewCmd(cli *Cli) *cobra.Command {
	c := cobra.Command{
		Use:   "new <migration> <name>",
		Short: "Create a new migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("not enough args")
			}
			m, ok := cli.config.DB.Migrations[args[0]]
			if !ok {
				return fmt.Errorf("could not find migration named %q", args[0])
			}
			name := args[1]

			u, err := url.Parse(m.Source)
			if err != nil {
				return err
			}
			dir := filepath.Join(u.Host, u.Path)
			matches, err := filepath.Glob(filepath.Join(dir, "*"+defaultExt))
			if err != nil {
				return err
			}
			version, err := nextSeqVersion(matches, defaultMigrationDigits)
			if err != nil {
				return err
			}
			versionGlob := filepath.Join(dir, version+"_*"+defaultExt)
			matches, err = filepath.Glob(versionGlob)
			if err != nil {
				return err
			}
			if len(matches) > 0 {
				return fmt.Errorf("duplicate migration version: %s", version)
			}
			if err = os.MkdirAll(dir, os.ModePerm); err != nil {
				return err
			}

			for _, direction := range []string{"up", "down"} {
				basename := fmt.Sprintf("%s_%s.%s%s", version, name, direction, defaultExt)
				filename := filepath.Join(dir, basename)
				if err = createFile(filename); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return &c
}

func getMigrationConfig(conf *provision.DBConfig, args []string) (*provision.Migration, int64, error) {
	var (
		n   int64
		err error
	)
	if len(args) == 0 {
		return nil, 0, errors.New("cannot run migrations: no name given")
	} else if len(args) >= 2 {
		n, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return nil, 0, errors.Wrap(err, "invalid migration count")
		}
	} else {
		n = -1
	}

	m, ok := conf.Migrations[args[0]]
	if !ok {
		return nil, 0, fmt.Errorf("cannot find migration named %q", args[0])
	}
	return &m, n, nil
}

func getMigration(conf *provision.DBConfig, args []string) (*migrate.Migrate, int64, error) {
	m, n, err := getMigrationConfig(conf, args)
	if err != nil {
		return nil, 0, err
	}
	migration, err := migrate.New(m.Source, conf.URI(m.Database).String())
	if err != nil {
		return nil, 0, err
	}
	migration.Log = &Logger{Logger: logger}
	return migration, n, nil
}

type Logger struct {
	*logrus.Logger
	verbose bool
}

func (l *Logger) Verbose() bool {
	return l.verbose
}

var (
	errInvalidSequenceWidth     = errors.New("Digits must be positive")
	errIncompatibleSeqAndFormat = errors.New("The seq and format options are mutually exclusive")
	errInvalidTimeFormat        = errors.New("Time format may not be empty")
)

func nextSeqVersion(matches []string, seqDigits int) (string, error) {
	if seqDigits <= 0 {
		return "", errInvalidSequenceWidth
	}
	nextSeq := uint64(1)

	if len(matches) > 0 {
		filename := matches[len(matches)-1]
		matchSeqStr := filepath.Base(filename)
		idx := strings.Index(matchSeqStr, "_")
		if idx < 1 { // Using 1 instead of 0 since there should be at least 1 digit
			return "", fmt.Errorf("Malformed migration filename: %s", filename)
		}

		var err error
		matchSeqStr = matchSeqStr[0:idx]
		nextSeq, err = strconv.ParseUint(matchSeqStr, 10, 64)
		if err != nil {
			return "", err
		}
		nextSeq++
	}

	version := fmt.Sprintf("%0[2]*[1]d", nextSeq, seqDigits)
	if len(version) > seqDigits {
		return "", fmt.Errorf("Next sequence number %s too large. At most %d digits are allowed", version, seqDigits)
	}
	return version, nil
}

func createFile(filename string) error {
	// create exclusive (fails if file already exists)
	// os.Create() specifies 0666 as the FileMode, so we're doing the same
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)

	if err != nil {
		return err
	}
	return f.Close()
}
