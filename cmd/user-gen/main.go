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

	"github.com/joho/godotenv"
	"golang.org/x/term"
	"harrybrown.com/app"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/db"
)

func main() {
	godotenv.Load()
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

	roles := make([]auth.Role, 0)
	for _, r := range strings.Split(rolesflag, ",") {
		roles = append(roles, auth.Role(strings.Trim(r, "\t\n ")))
	}

	if !strings.Contains(rolesflag, ",") && len(roles) == 0 {
		roles = []auth.Role{auth.RoleDefault}
	}

	if username == "" {
		fmt.Printf("username: ")
		fmt.Scanln(&username)
	}
	if email == "" {
		fmt.Printf("email: ")
		fmt.Scanln(&email)
	}

	fmt.Print("Password: ")
	pw, err := term.ReadPassword(int(syscall.Stdin))
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
	user := app.User{
		Username: username,
		Email:    email,
		// Roles:    roles,
	}
	for _, r := range roles {
		user.Roles = append(user.Roles, auth.Role(r))
	}

	db, err := db.Connect()
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
	fmt.Printf("User created:\n%s\n", b)
	return nil
}
