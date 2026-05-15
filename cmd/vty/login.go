// <--- Main --->

package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/v2/internal/auth"
	"github.com/sthbryan/vaulty/v2/internal/ui"
	"github.com/sthbryan/vaulty/v2/internal/vault"
	"github.com/sthbryan/vaulty/v2/internal/vault/providers"
	"github.com/sthbryan/vaulty/v2/pkg/models"
)

func runLogin(cmd *cobra.Command, args []string) error {
	config, err := vault.LoadConfig()
	session, sessionErr := vault.LoadSession()

	if err != nil && sessionErr != nil {
		ui.PrintError("No vault configured. Use 'start' to create a vault.")
		return nil
	}

	var username, vaultID, storageType string
	if err == nil {
		username = config.Username
		vaultID = config.VaultID
		storageType = config.StorageType
	} else if sessionErr == nil {
		username = session.Username
		vaultID = session.VaultID
		storageType = session.StorageType
	}

	if sessionErr == nil && !session.IsExpired() {
		ui.PrintBold("-- Session active --")
		fmt.Println()
		ui.PrintInfo(fmt.Sprintf("Username: %s", username))
		ui.PrintInfo(fmt.Sprintf("Vault: %s", vaultID))
		ui.PrintInfo(fmt.Sprintf("Expires: %s", session.ExpiresAt.Format("2006-01-02 15:04")))
		fmt.Println()

		duration, err := ui.Select("Extend session by?", []ui.SelectOption{
			{ID: "8h", Label: "8 hours"},
			{ID: "24h", Label: "24 hours"},
			{ID: "7d", Label: "7 days"},
			{ID: "30d", Label: "30 days"},
		})
		if err != nil {
			return fmt.Errorf("cancelled")
		}

		return runExtendSession(session, duration)
	}

	configForUnlock := &models.VaultConfig{
		Username:    username,
		VaultID:     vaultID,
		StorageType: storageType,
	}
	return runLoginWithConfig(configForUnlock)
}

func runExtendSession(session *models.Session, duration string) error {
	authSvc := auth.New()
	hours, err := authSvc.GenerateSessionDuration(duration)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Invalid duration: %v", err))
		return fmt.Errorf("invalid duration")
	}

	session.ExpiresAt = session.ExpiresAt.Add(time.Duration(hours) * time.Second)

	if err := vault.SaveSession(session); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to save session: %v", err))
		return fmt.Errorf("failed to save session")
	}

	fmt.Println()
	ui.PrintSuccess(fmt.Sprintf("Session extended until %s", session.ExpiresAt.Format("2006-01-02 15:04")))
	return nil
}

func runLoginWithConfig(config *models.VaultConfig) error {
	ui.PrintBold("-- Session expired --")
	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Username: %s", config.Username))
	ui.PrintInfo(fmt.Sprintf("Vault: %s", config.VaultID))
	ui.PrintInfo(fmt.Sprintf("Storage: %s", config.StorageType))
	fmt.Println()

	var meta *models.VaultMeta
	var err error

	if config.StorageType == "github" {
		ui.PrintInfo("Loading from GitHub...")
		meta, err = vault.LoadMetaFromStorage(vault.VaultInfo{
			Username:    config.Username,
			VaultID:     config.VaultID,
			StorageType: config.StorageType,
		})
		if err != nil {
			if providers.IsFileNotFound(err) {
				ui.PrintError("Vault not found in repository")
				return fmt.Errorf("vault not found")
			}
			ui.PrintError(fmt.Sprintf("Failed to load vault: %v", err))
			return fmt.Errorf("failed to load vault")
		}
	} else {
		ui.PrintInfo("Loading from local...")
		meta, err = vault.LoadMeta()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to load vault meta: %v", err))
			return fmt.Errorf("failed to load vault meta")
		}
	}

	return loginWithPassword(config, meta)
}

func loginWithPassword(config *models.VaultConfig, meta *models.VaultMeta) error {
	ui.PrintInfo("Enter your master password")

	password, err := ui.Password("Master password", "••••••••")
	if err != nil {
		return fmt.Errorf("wizard cancelled")
	}

	authSvc := auth.New()
	_, err = authSvc.DecryptVaultKey(meta.EncryptedKey, password)
	if err != nil {
		ui.PrintError("Invalid password")
		return fmt.Errorf("invalid password")
	}

	duration := "8h"
	authSvc2 := auth.New()
	hours, _ := authSvc2.GenerateSessionDuration(duration)

	session := &models.Session{
		Username:    config.Username,
		VaultID:     config.VaultID,
		StorageType: config.StorageType,
		ExpiresAt:   time.Now().Add(time.Duration(hours) * time.Second),
	}

	if err := vault.SaveSession(session); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to save session: %v", err))
		return fmt.Errorf("failed to save session")
	}

	fmt.Println()
	ui.PrintSuccess("Vault unlocked!")

	return nil
}

// <--- Cobra --->

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Unlock vault or extend session",
	Long:  "Unlock a linked vault or extend the current session.",
	RunE:  runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
