package main

import (
	"fmt"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/spf13/cobra"
)

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
