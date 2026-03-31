package main

import (
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

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.AddCommand(runEnvCmd)

	runEnvCmd.Flags().StringVarP(&runEnv, "env", "e", "", "Target environment (optional)")
}
