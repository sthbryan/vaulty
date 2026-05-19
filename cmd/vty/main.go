package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "vty",
	Short:         "Vaulty - Secret management CLI",
	Long:          "A secure CLI for managing secrets, environment variables, configurations and more.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		handleCommandError(err)
		os.Exit(1)
	}
}