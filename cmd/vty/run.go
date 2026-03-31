package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var runEnv string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run commands with secrets injected from Vaulty",
	Long: `Run a command with environment variables injected from your Vaulty vault.

This command downloads, decrypts, and injects secrets directly into the
child process environment without writing any .env files to disk.

Examples:
  vty run env api -- npm run build
  vty run env api -e production -- npm run build`,
}

var runEnvCmd = &cobra.Command{
	Use:   "env <name> [--env <environment>] -- <command> [args...]",
	Short: "Run with environment secrets",
	Long: `Download and decrypt environment secrets, then execute a command with those secrets injected into the environment.

The '--' separator is required to distinguish Vaulty flags from the child command.

Examples:
  vty run env api -- npm run build
  vty run env api -e production -- npm run build
  vty run env api --env staging -- sh -c 'npm run migrate && npm run start'`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRunEnv,
}

func runRunEnv(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("not yet implemented")
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.AddCommand(runEnvCmd)

	runEnvCmd.Flags().StringVarP(&runEnv, "env", "e", "", "Target environment (optional)")
}
