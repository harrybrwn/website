package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/term"

	"harrybrown.com/app"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/db"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		cancel    context.CancelFunc
		ctx       = context.Background()
		out       = os.Stdout
		in        = bufio.NewScanner(os.Stdin)
		env       = ".env"
		username  string
		email     string
		rolesflag string
		yes       bool
	)
	flag.StringVar(&username, "u", username, "username of new user")
	flag.StringVar(&email, "e", email, "email of new user")
	flag.StringVar(&rolesflag, "roles", rolesflag, "roles for new user")
	flag.StringVar(&env, "env", env, "environment file")
	flag.BoolVar(&yes, "y", yes, "skip verification prompts")
	flag.Parse()
	ctx, cancel = signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	if err := godotenv.Load(env); err != nil {
		return err
	}

	roles := make([]auth.Role, 0)
	for _, r := range strings.Split(rolesflag, ",") {
		switch auth.Role(strings.ToLower(strings.Trim(r, "\n\t "))) {
		case auth.RoleAdmin:
			roles = append(roles, auth.RoleAdmin)
		case auth.RoleDefault:
			roles = append(roles, auth.RoleDefault)
		case auth.RoleTanya:
			roles = append(roles, auth.RoleTanya)
		default:
			return fmt.Errorf("unknown role: %s", r)
		}
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
		fmt.Fprint(out, "Create User (y/N): ")
		in.Scan()
		switch strings.ToLower(in.Text()) {
		case "y", "yes":
		default:
			fmt.Fprintln(out)
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

	db, err := db.Connect(logrus.New())
	if err != nil {
		return err
	}
	defer db.Close()
	store := app.NewUserStore(db)
	u, err := store.Create(ctx, string(pw), &user)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "User created:\n%s\n", b)
	return nil
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
