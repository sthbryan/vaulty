package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/password"
	"github.com/sthbryan/vaulty/internal/ui"
)

var logoutForce bool

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored master password",
	RunE:  runLogout,
}

func runLogout(cmd *cobra.Command, args []string) error {
	if !logoutForce {
		confirmed, err := ui.AskConfirm("Clear stored master password?", false)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !confirmed {
			fmt.Println("Logout cancelled")
			return nil
		}
	}

	storage, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("password storage: %w", err)
	}

	if err := storage.Delete(); err != nil {
		return fmt.Errorf("failed to clear password: %w", err)
	}

	fmt.Println("Password cleared from storage")
	fmt.Println("Run 'vty init' to link again or 'vty recover' if you forgot your password")

	return nil
}

func init() {
	logoutCmd.Flags().BoolVarP(&logoutForce, "force", "f", false, "Skip confirmation")
	rootCmd.AddCommand(logoutCmd)
}
