package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	version = ""
	commit  = ""
	date    = ""
)

var logger *log.Logger

var rootCmd = &cobra.Command{
	Use:   "vty",
	Short: "🔐 Vaulty - Secure environment and SSH key vault",
	Long: `Vaulty is a secure vault for managing environment variables, SSH keys, and team resources.
	
	Use "vty [command] --help" for more information about a command.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
}

func init() {
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix:          "vty",
		ReportTimestamp: false,
		Level:           log.InfoLevel,
	})

	rootCmd.AddGroup(
		&cobra.Group{ID: "secrets", Title: "Secret Management"},
		&cobra.Group{ID: "vault", Title: "Vault Operations"},
		&cobra.Group{ID: "config", Title: "Configuration"},
		&cobra.Group{ID: "account", Title: "Account"},
		&cobra.Group{ID: "system", Title: "System"},
		&cobra.Group{ID: "team", Title: "Team Management (owner only)"},
	)

	configCmd.GroupID = "config"

	pushCmd.GroupID = "secrets"
	pullCmd.GroupID = "secrets"
	showCmd.GroupID = "secrets"
	runCmd.GroupID = "secrets"
	deleteCmd.GroupID = "secrets"

	exportCmd.GroupID = "vault"
	importCmd.GroupID = "vault"
	infoCmd.GroupID = "vault"

	loginCmd.GroupID = "account"
	logoutCmd.GroupID = "account"
	initCmd.GroupID = "account"
	linkCmd.GroupID = "account"
	unlinkCmd.GroupID = "account"

	updateCmd.GroupID = "system"

	addUserCmd.GroupID = "team"
	removeUserCmd.GroupID = "team"
	transferOwnerCmd.GroupID = "team"
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}