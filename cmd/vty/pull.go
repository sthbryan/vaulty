package main

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	pullOutput      string
	pullInteractive bool
	pullUser        string
	pullEnv         string
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull and decrypt secrets from Vault",
	Long: `Pull encrypted secrets from your GitHub repository.

This command downloads, decrypts and saves secrets from your Vaulty repository.

Examples:
  vty pull env myapp-prod          # Pull environment secrets
  vty pull ssh my-key              # Pull your own SSH key
  vty pull ssh my-key -u other     # Owner: pull another user's SSH key`,
}

var pullEnvCmd = &cobra.Command{
	Use:   "env <name>",
	Short: "Pull environment secrets from Vault",
	Long: `Download and decrypt environment secrets from the envs/ directory.

Examples:
  vty pull env myapp-prod
  vty pull env myapp-prod -o .env.production`,
	Args: cobra.ExactArgs(1),
	RunE: runPullEnv,
}

var pullSSHCmd = &cobra.Command{
	Use:   "ssh <name>",
	Short: "Pull SSH key from Vault",
	Long: `Download and decrypt SSH key from the ssh/ directory.

Users can only pull their own SSH keys unless they are the owner.

Examples:
  vty pull ssh my-key              # Pull current user's SSH key
  vty pull ssh team-key -u other   # Owner: pull another user's key`,
	Args: cobra.ExactArgs(1),
	RunE: runPullSSH,
}

func init() {
	pullEnvCmd.Flags().StringVarP(&pullOutput, "output", "o", "", "Output filename (default: .env, use - for stdout)")
	pullEnvCmd.Flags().BoolVarP(&pullInteractive, "interactive", "i", false, "Interactive mode (prompt for filename)")
	pullEnvCmd.Flags().StringVarP(&pullEnv, "env", "e", "", "Source environment (optional: production, staging, development)")

	pullSSHCmd.Flags().StringVarP(&pullOutput, "output", "o", "", "Output filename (default: <name>, use - for stdout)")
	pullSSHCmd.Flags().BoolVarP(&pullInteractive, "interactive", "i", false, "Interactive mode (prompt for filename)")
	pullSSHCmd.Flags().StringVarP(&pullUser, "user", "u", "", "Target user (owner only)")

	pullCmd.AddCommand(pullEnvCmd)
	pullCmd.AddCommand(pullSSHCmd)
	rootCmd.AddCommand(pullCmd)
}

func SetLogger(l *log.Logger) {
	logger = l
}
