package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/cobra"
	"harrybrown.com/pkg/app"
)

func main() {
	var (
		baseURL = "https://api.hrry.me-local"
		config  Config
	)
	root := cobra.Command{
		Use: "hrry",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			config.URL, err = url.Parse(baseURL)
			if err != nil {
				return err
			}
			return nil
		},
	}
	root.AddCommand(
		newLoginCmd(),
		newTokenCmd(&config),
	)
	root.PersistentFlags().StringVar(&baseURL, "url", baseURL, "base api url")
	err := root.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

type Config struct {
	URL *url.URL
}

func newLoginCmd() *cobra.Command {
	var (
		username string
		password string
	)
	c := cobra.Command{
		Use: "login",
	}
	c.Flags().StringVarP(&username, "username", "u", "", "username of the user")
	c.Flags().StringVarP(&password, "password", "p", "", "password of the user")
	return &c
}

func newTokenCmd(cfg *Config) *cobra.Command {
	var (
		username string
		password string
	)
	c := cobra.Command{
		Use: "token",
		RunE: func(cmd *cobra.Command, args []string) error {
			var u = *cfg.URL
			u.Path = "/api/token"
			req := http.Request{
				Method: "POST",
				URL:    &u,
			}
			res, err := http.DefaultClient.Do(&req)
			if err != nil {
				return err
			}
			defer res.Body.Close()
			var login app.Login
			err = json.NewDecoder(res.Body).Decode(&login)
			if err != nil {
				return err
			}
			return nil
		},
	}
	c.Flags().StringVarP(&username, "username", "u", "", "username of the user")
	c.Flags().StringVarP(&password, "password", "p", "", "password of the user")
	return &c
}
