package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"harrybrown.com/app"
	"harrybrown.com/pkg/db"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func run() error {
	var (
		ctx       = context.Background()
		in        = bufio.NewScanner(os.Stdin)
		username  string
		email     string
		rolesflag string
	)
	flag.StringVar(&username, "u", username, "username of new user")
	flag.StringVar(&email, "e", email, "email of new user")
	flag.StringVar(&rolesflag, "roles", rolesflag, "roles for new user")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	roles := strings.Split(rolesflag, ",")
	if !strings.Contains(rolesflag, ",") {
		roles = []string{"default"}
	}

	fmt.Print("Password: ")
	pw, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	}
	pwhash, err := bcrypt.GenerateFromPassword(pw, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	fmt.Printf("\nusername: %q\nemail: %q\nroles: %v\n", username, email, roles)
	fmt.Print("Create User (y/N): ")
	in.Scan()
	switch strings.ToLower(in.Text()) {
	case "y", "yes":
	default:
		return nil
	}

	db, err := db.Connect()
	if err != nil {
		return err
	}
	defer db.Close()
	store := app.NewUserStore(db)
	u, err := store.Put(ctx, &app.User{
		Username: username,
		Email:    email,
		PWHash:   pwhash,
	})
	if err != nil {
		return err
	}
	b, err := json.Marshal(u)
	if err != nil {
		return err
	}
	fmt.Printf("User created:\n%s\n", b)
	return nil
}
