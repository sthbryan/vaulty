package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var logger *log.Logger

var rootCmd = &cobra.Command{
	Use:   "vty",
	Short: "🔐 Vaulty - Secure environment and SSH key vault",
	Long: `Vaulty is a secure vault for managing environment variables and SSH keys.

It provides a safe way to store, retrieve, and inject sensitive configuration
into your development workflow. Vaulty supports:

  • Secure storage of environment variables
  • SSH key management and injection
  • GitHub as storage backend
  • Easy migration between machines

Use "vty [command] --help" for more information about a command.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
}

func init() {
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix:          "vty",
		ReportTimestamp: false,
		Level:           log.InfoLevel,
	})
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(linkCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}
