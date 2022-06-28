package main

import (
	"errors"
	"fmt"
	"os"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/aws_s3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"

	"github.com/golang-migrate/migrate/v4"
	flag "github.com/spf13/pflag"
)

func main() {
	var (
		source   string
		database string
		up, down bool
	)
	flag.StringVar(&source, "source", source, "migrations source")
	flag.StringVar(&database, "database", database, "database uri")
	flag.BoolVarP(&up, "up", "u", up, "migrate up")
	flag.BoolVarP(&down, "down", "d", down, "migrate down")
	flag.Parse()

	m, err := migrate.New(source, database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if up {
		err = m.Up()
	} else if down {
		err = m.Down()
	} else {
		err = errors.New("use --up or --down")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	serr, derr := m.Close()
	if serr != nil {
		fmt.Fprintf(os.Stderr, "Source Error: %v\n", serr)
		os.Exit(1)
	} else if derr != nil {
		fmt.Fprintf(os.Stderr, "Database Error: %v\n", derr)
		os.Exit(1)
	}
}
