package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/v2/internal/ui"
	"github.com/sthbryan/vaulty/v2/internal/vault"
)

// <--- Main --->

func runLogout(cmd *cobra.Command, args []string) error {
	if !vault.SessionExists() {
		ui.PrintError("No active session")
		ui.PrintInfo("Vault is already locked")
		return fmt.Errorf("no active session")
	}

	session, err := vault.LoadSession()
	if err != nil {
		ui.PrintError("Failed to load session")
		return fmt.Errorf("failed to load session")
	}

	if session.IsExpired() {
		ui.PrintError("Session already expired")
		ui.PrintInfo("Vault is already locked")
		return nil
	}

	ui.PrintBold("-- Logout --")
	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Session expires: %s", session.ExpiresAt.Format("2006-01-02 15:04")))
	fmt.Println()

	ok, err := ui.Confirm("Lock vault now?")
	if err != nil || !ok {
		return fmt.Errorf("Cancelled")
	}

	fmt.Println()

	if err := vault.DeleteSession(); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to lock vault: %v", err))
		return fmt.Errorf("failed to lock vault")
	}

	ui.PrintSuccess("Vault locked!")
	ui.PrintInfo("Run 'login' to unlock")

	return nil
}

// <--- Cobra --->

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Lock the vault",
	Long:  "Terminate the current session and lock the vault.",
	RunE:  runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}