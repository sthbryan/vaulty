package main

import (
	"github.com/spf13/cobra"
)

var showEnv string

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Display secret contents",
	Long: `Display the contents of a secret with formatting.

Examples:
  vty show env api -e production
  vty show ssh laptop
  vty show resource docker-compose
  vty show config zellij`,
}

var showEnvCmd = &cobra.Command{
	Use:   "env <name>",
	Short: "Show environment secrets",
	Long: `Download and display environment secrets.

Examples:
  vty show env api
  vty show env api -e production`,
	Args: cobra.ExactArgs(1),
	RunE: runShowEnv,
}

var showSSHCmd = &cobra.Command{
	Use:   "ssh <name>",
	Short: "Show SSH key info",
	Long: `Display SSH key information (fingerprint + preview).

Examples:
  vty show ssh laptop`,
	Args: cobra.ExactArgs(1),
	RunE: runShowSSH,
}

var showResourceCmd = &cobra.Command{
	Use:   "resource <name>",
	Short: "Show resource file",
	Long: `Download and display a resource file.

Examples:
  vty show resource docker-compose`,
	Args: cobra.ExactArgs(1),
	RunE: runShowResource,
}

var showConfigCmd = &cobra.Command{
	Use:   "config <name>",
	Short: "Show config file",
	Long: `Download and display a config file.

Examples:
  vty show config zellij`,
	Args: cobra.ExactArgs(1),
	RunE: runShowConfig,
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.AddCommand(showEnvCmd)
	showCmd.AddCommand(showSSHCmd)
	showCmd.AddCommand(showResourceCmd)
	showCmd.AddCommand(showConfigCmd)

	showEnvCmd.Flags().StringVarP(&showEnv, "env", "e", "", "Target environment (optional)")
}
