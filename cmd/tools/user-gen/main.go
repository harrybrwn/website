package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"golang.org/x/term"

	"harrybrown.com/app"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/db"
	"harrybrown.com/pkg/log"
)

func init() {
	os.Unsetenv("PGSERVICEFILE")
	os.Unsetenv("PGSERVICE")
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var (
	out = os.Stdout
	in  = bufio.NewScanner(os.Stdin)
	yes bool
)

func run() error {
	var (
		cancel    context.CancelFunc
		ctx       = context.Background()
		env       = []string{".env"}
		username  string
		email     string
		rolesflag string
	)
	flag.StringVarP(&username, "username", "u", username, "username of new user")
	flag.StringVarP(&email, "email", "e", email, "email of new user")
	flag.StringVar(&rolesflag, "roles", rolesflag, "roles for new user")
	flag.StringArrayVar(&env, "env", env, "environment file")
	flag.BoolVarP(&yes, "yes", "y", yes, "skip verification prompts")
	flag.Parse()

	ctx, cancel = signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	if err := godotenv.Load(env...); err != nil {
		return err
	}
	db, err := database()
	if err != nil {
		return err
	}
	defer db.Close()

	roles := make([]auth.Role, 0)
	for _, r := range strings.Split(rolesflag, ",") {
		role := auth.ParseRole(r)
		if role == auth.RoleInvalid {
			return fmt.Errorf("invalid role: %s", r)
		}
		roles = append(roles, role)
	}
	if len(roles) == 0 {
		roles = []auth.Role{auth.RoleDefault}
	}

	if username == "" {
		fmt.Fprintf(out, "username: ")
		fmt.Scanln(&username)
	}
	if email == "" {
		fmt.Fprintf(out, "email: ")
		fmt.Scanln(&email)
	}

	pw, err := readPassword(out, yes)
	if err != nil {
		fmt.Fprintln(out)
		return err
	}
	if len(pw) == 0 {
		return errors.New("no password given")
	}
	if !yes && flag.Arg(0) != "-" {
		fmt.Fprintf(out, "\nusername: %q\nemail: %q\nroles: %v\n", username, email, roles)
		if !ask("Create User?") {
			return errors.New("cancelled")
		}
	}
	user := app.User{
		Username: username,
		Email:    email,
	}
	for _, r := range roles {
		user.Roles = append(user.Roles, auth.Role(r))
	}

	store := app.NewUserStore(db)
	var u *app.User
	existing, err := store.Find(ctx, user.Email)
	switch err {
	case sql.ErrNoRows:
		u, err = store.Create(ctx, string(pw), &user)
		if err != nil {
			return err
		}
	case nil:
		if !ask("User exists. Would you like you Update it?") {
			return errors.New("cancelled")
		}
		existing.Roles = user.Roles
		existing.Username = user.Username
		existing.Email = user.Email
		err = store.UpdateWithPassword(ctx, string(pw), existing)
		if err != nil {
			return err
		}
		u = existing
	default:
		return err
	}
	b, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "User created:\n%s\n", b)
	return nil
}

func ask(question string) bool {
	fmt.Fprint(out, strings.Trim(question, " \n\t")+" (y/N): ")
	if yes {
		fmt.Fprintln(out, "y")
		return true
	}
	in.Scan()
	switch strings.ToLower(in.Text()) {
	case "y", "yes":
		return true
	default:
		fmt.Fprintln(out)
		return false
	}
}

func readPassword(out io.Writer, yes bool) ([]byte, error) {
	var (
		err error
		pw  []byte
	)
	if flag.Arg(0) == "-" {
		var buf bytes.Buffer
		if _, err = io.Copy(&buf, os.Stdin); err != nil {
			return nil, err
		}
		pw = bytes.Trim(buf.Bytes(), "\n\t ")
	} else {
		fmt.Fprint(out, "Password: ")
		pw, err = term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return nil, fmt.Errorf("%w: use \"-\" as an argument to read passwords from stdin", err)
		}
		if !yes {
			fmt.Fprint(out, "\nConfirm Password: ")
			confirmPw, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(confirmPw, pw) {
				return nil, errors.New("passwords were different")
			}
		}
	}
	return pw, nil
}

func database() (db.DB, error) {
	host := getenv("POSTGRES_HOST", "localhost")
	port := getenv("POSTGRES_PORT", "5432")
	u := url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(host, port),
		User: url.UserPassword(
			getenv("POSTGRES_USER", "harrybrwn"),
			getenv("POSTGRES_PASSWORD", ""),
		),
		Path:     filepath.Join("/", getenv("POSTGRES_DB", "harrybrwn_api")),
		RawQuery: "sslmode=disable",
	}
	logger := log.GetLogger()
	if app.Debug {
		logger.Info(u)
	}
	pool, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(); err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}
	return db.New(pool, db.WithLogger(logger)), nil
}

func getenv(key, defaultValue string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		return defaultValue
	}
	return val
}
