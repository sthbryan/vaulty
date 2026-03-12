package main

import (
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/spf13/cobra"
)

var (
	deleteForce bool
	deleteEnv   string
	deleteUser  string
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete secrets, environments, or vault",
	Long: `Delete secrets, environments, or the entire vault.

Examples:
  vty delete env <name>              # Delete secret from shared
  vty delete env <name> --env=staging # Delete secret from environment
  vty delete envs --env=staging       # Delete all secrets from environment
  vty delete ssh <name>               # Delete SSH key
  vty delete vault                     # Delete entire vault (owner only)`,
}

var deleteEnvCmd = &cobra.Command{
	Use:   "env <name>",
	Short: "Delete a specific environment variable",
	Long: `Delete a specific environment variable from the vault.

If --env is not specified, deletes from shared (envs/{name}.vty).
If --env is specified, deletes from envs/{env}/{name}.vty.`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteEnv,
}

var deleteEnvsCmd = &cobra.Command{
	Use:   "envs",
	Short: "Delete all secrets from an environment",
	Long: `Delete all secrets from a specific environment.

This will permanently remove all secrets from the specified environment.
Use with caution - this action cannot be undone.`,
	RunE: runDeleteEnvs,
}

var deleteSSHCmd = &cobra.Command{
	Use:   "ssh <name>",
	Short: "Delete an SSH key",
	Long: `Delete an SSH key from the vault.

Examples:
  vty delete ssh my-key
  vty delete ssh my-key -u username`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteSSH,
}

var deleteVaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Delete entire vault (DESTRUCTIVE - owner only)",
	Long: `Delete the entire vault including all secrets, SSH keys, and users.

This is a DESTRUCTIVE operation that will:
  - Delete all environment secrets
  - Delete all SSH keys
  - Delete all user keys and recovery files
  - Delete metadata

This action CANNOT be undone. Only the vault owner can perform this action.`,
	RunE: runDeleteVault,
}

func init() {

	deleteCmd.AddCommand(deleteEnvCmd)
	deleteCmd.AddCommand(deleteEnvsCmd)
	deleteCmd.AddCommand(deleteSSHCmd)
	deleteCmd.AddCommand(deleteVaultCmd)

	rootCmd.AddCommand(deleteCmd)

	deleteEnvCmd.Flags().StringVar(&deleteEnv, "env", "", "Environment (optional)")
	deleteEnvCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Force delete without confirmation")

	deleteEnvsCmd.Flags().StringVar(&deleteEnv, "env", "", "Environment (required)")
	deleteEnvsCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Force delete without confirmation")

	deleteSSHCmd.Flags().StringVarP(&deleteUser, "user", "u", "", "User (optional)")
	deleteSSHCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Force delete without confirmation")

	deleteVaultCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Force delete without confirmation")
}

func getConfigAndClient() (*config.Config, *github.Client, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return nil, nil, err
	}

	client := github.NewClient(token)
	return cfg, client, nil
}
