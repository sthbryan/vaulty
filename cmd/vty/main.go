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
  • Multiple storage backends
  • Easy integration with shell environments

Use "vty [command] --help" for more information about a command.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := checkConfig(); err != nil {
			logger.Error("Configuration check failed", "error", err)
			return err
		}
		return nil
	},
}

func init() {
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix:          "vty",
		ReportTimestamp: false,
		Level:           log.InfoLevel,
	})

	// Subcommands will be initialized here
	// Example: rootCmd.AddCommand(envCmd)
	// Example: rootCmd.AddCommand(sshCmd)
}

func checkConfig() error {
	// Placeholder: verify config file exists and is valid
	// This will be implemented when config package is ready
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}
