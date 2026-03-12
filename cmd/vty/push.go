package main

import (
	"github.com/spf13/cobra"
)

var (
	pushForce bool
	pushEnv   string
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push secrets to Vaulty",
	Long:  `Push environment files or SSH keys to your Vaulty repository.`,
}

var pushEnvCmd = &cobra.Command{
	Use:   "env <name> <path>",
	Short: "Push an environment file to Vaulty",
	Long: `Compress, encrypt, and upload an environment file to your Vaulty repository.

The file will be:
  1. Compressed using gzip for efficiency
  2. Encrypted using AES-256-GCM with your password
  3. Uploaded to your GitHub repository in the envs/ directory

Examples:
  vty push env production .env.production
  vty push env staging .env.staging --force`,
	Args: cobra.ExactArgs(2),
	RunE: runPushEnv,
}

var pushSSHCmd = &cobra.Command{
	Use:   "ssh <name> <path>",
	Short: "Push an SSH key to Vaulty",
	Long: `Compress, encrypt, and upload an SSH private key to your Vaulty repository.

The file will be:
  1. Compressed using gzip for efficiency
  2. Encrypted using AES-256-GCM with your password
  3. Uploaded to ssh/{username}/{name}.vty in your repository

Only owners and editors can push SSH keys to their own directory.
Viewers cannot push any secrets.

Examples:
  vty push ssh laptop ~/.ssh/id_rsa
  vty push ssh server ~/.ssh/server_key --force`,
	Args: cobra.ExactArgs(2),
	RunE: runPushSSH,
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.AddCommand(pushEnvCmd)
	pushCmd.AddCommand(pushSSHCmd)

	pushEnvCmd.Flags().BoolVarP(&pushForce, "force", "f", false, "Overwrite without prompting")
	pushEnvCmd.Flags().StringVarP(&pushEnv, "env", "e", "", "Target environment (optional: production, staging, development)")
	pushSSHCmd.Flags().BoolVarP(&pushForce, "force", "f", false, "Overwrite without prompting")
	pushSSHCmd.Flags().StringVarP(&pushEnv, "env", "e", "", "Target environment (optional, for env files only)")
}
