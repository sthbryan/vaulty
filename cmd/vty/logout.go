package main

import (
	"fmt"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/spf13/cobra"
)

var logoutForce bool

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored master password and log out",
	RunE:  runLogout,
}

func runLogout(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.CurrentUser == "" {
		return fmt.Errorf("no active session - run 'vty login' first")
	}

	sm := session.GetManager()
	sess := sm.Get(cfg.CurrentUser)

	if sess != nil && sess.IsActive() {
		if !logoutForce {
			confirmed, err := ui.AskConfirm(fmt.Sprintf("You will be logged out from %s", cfg.CurrentUser), false)
			if err != nil {
				return fmt.Errorf("confirmation failed: %w", err)
			}
			if !confirmed {
				fmt.Println("Logout cancelled")
				return nil
			}
		}

		sm.Delete(cfg.CurrentUser)
	}

	// Clear current user from config
	cfg.ClearCurrentUser()
	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	fmt.Println("✅ Logged out. Cache and config kept. Run 'vty login' to access vault again.")

	return nil
}

func init() {
	logoutCmd.Flags().BoolVarP(&logoutForce, "force", "f", false, "Skip confirmation")
	rootCmd.AddCommand(logoutCmd)
}
