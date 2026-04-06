package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/session"
	"github.com/sthbryan/vaulty/internal/ui"
)

var logoutForce bool

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored password and log out",
	Long: `Clear stored master password and end your active session.

This command will:
  • Remove your session from the session manager
  • Clear the current user from configuration
  • Keep cache and configuration files on disk

You can re-login anytime with 'vty login'.`,
	RunE:  runLogout,
}

func runLogout(cmd *cobra.Command, args []string) error {
	fmt.Println()
	ui.PrintAnimatedLogo()
	fmt.Println()

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
