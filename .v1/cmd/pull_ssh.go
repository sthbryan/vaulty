package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
)

var pullSSHCmd = &cobra.Command{
	Use:   "ssh <name>",
	Short: "Get SSH key",
	Long: `Get SSH key from your vault.
Examples:
  vty pull ssh my-key
  vty pull ssh team-key -u other   # Owner: get another user's key`,
	Args: cobra.ExactArgs(1),
	RunE: runPullSSH,
}

func init() {
	pullCmd.AddCommand(pullSSHCmd)
	pullSSHCmd.Flags().StringVarP(&pullOutput, "output", "o", "", "Output filename")
	pullSSHCmd.Flags().BoolVarP(&pullInteractive, "interactive", "i", false, "Interactive mode")
	pullSSHCmd.Flags().StringVarP(&pullUser, "user", "u", "", "Target user (owner only)")
}

func runPullSSH(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	sess, err := ensureAuthenticated(cfg)
	if err != nil {
		return err
	}

	targetUser := sess.Username
	if pullUser != "" {
		if sess.Role != "owner" {
			return fmt.Errorf("only owner can pull other users' SSH keys")
		}
		targetUser = pullUser
	}

	return pullSecretWithSession(name, "ssh", targetUser, sess)
}
