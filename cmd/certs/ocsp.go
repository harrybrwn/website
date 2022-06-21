package main

import "github.com/spf13/cobra"

func NewOCSPCmd() *cobra.Command {
	c := cobra.Command{Use: "ocsp"}
	c.AddCommand(
		NewOCSPServerCmd(),
		NewOCSPClientCmd(),
	)
	flag := c.PersistentFlags()
	flag.String("responder-cert", "", "ocsp responder certificate")
	flag.String("responder-key", "", "ocsp responder private key")
	return &c
}

func NewOCSPServerCmd() *cobra.Command {
	c := cobra.Command{Use: "server"}
	return &c
}

func NewOCSPClientCmd() *cobra.Command {
	c := cobra.Command{Use: "client"}
	return &c
}
