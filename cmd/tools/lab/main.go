package main

import (
	"github.com/spf13/cobra"
)

func main() {
	root := NewRootCmd()
	root.Execute()
}

func NewRootCmd() *cobra.Command {
	c := cobra.Command{
		Use:   "lab",
		Short: "hrry.me homelab management tool",
	}
	c.AddCommand(
		NewGenCmd(),
	)
	return &c
}

func NewGenCmd() *cobra.Command {
	c := cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen"},
		Short:   "A collection of code generation tools.",
	}
	c.AddCommand(
		NewGenK8sCmd(),
	)
	return &c
}
