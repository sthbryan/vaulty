package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/v2/internal/auth"
	"github.com/sthbryan/vaulty/v2/internal/ui"
	"github.com/sthbryan/vaulty/v2/internal/vault"
	"github.com/sthbryan/vaulty/v2/pkg/models"
)

// <--- Main --->

func runStart(cmd *cobra.Command, args []string) error {
	if vault.SessionExists() {
		session, err := vault.LoadSession()
		if err == nil && !session.IsExpired() {
			ui.PrintError("Vault is already unlocked")
			ui.PrintInfo(fmt.Sprintf("Session expires: %s", session.ExpiresAt.Format("2006-01-02 15:04")))
			return fmt.Errorf("vault already unlocked")
		}
	}

	if vault.ConfigExists() {
		ui.PrintError("Vault is already configured")
		config, _ := vault.LoadConfig()
		ui.PrintInfo(fmt.Sprintf("Username: %s", config.Username))
		ui.PrintInfo(fmt.Sprintf("Vault: %s", config.VaultID))
		ui.PrintInfo(fmt.Sprintf("Storage: %s", config.StorageType))
		ui.PrintInfo("Use 'login' command to unlock")
		return fmt.Errorf("vault already configured")
	}

	return runStartWizard()
}

func runStartWizard() error {
	ui.PrintBold("-- Create vault --")
	fmt.Println()

	identify, err := ui.Identify()
	if err != nil {
		return fmt.Errorf("Cancelled")
	}

	ui.PrintBold("-- Vault info --")
	fmt.Println()

	detect, err := ui.Detect()
	if err != nil {
		return fmt.Errorf("Cancelled")
	}

	if detect.Username == "" {
		detect.Username = vault.CurrentUser()
	}
	if detect.VaultID == "" {
		detect.VaultID = "my-vault"
	}

	info := vault.VaultInfo{
		Username:    detect.Username,
		VaultID:     detect.VaultID,
		StorageType: identify.StorageType,
	}

	fmt.Println()

	exists, err := vault.CheckVaultAvailability(info)
	if err != nil {
		return err
	}

	if exists {
		return runLinkExisting(info)
	}

	return runCreateNew(info)
}

func runLinkExisting(info vault.VaultInfo) error {
	ui.PrintBold("-- Vault exists --")
	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Found vault on %s", info.StorageType))
	ui.PrintInfo(fmt.Sprintf("Repository: %s/%s", info.Username, info.VaultID))
	fmt.Println()

	ok, err := ui.Confirm("Link to existing vault?")
	if err != nil || !ok {
		return fmt.Errorf("Cancelled")
	}

	fmt.Println()
	ui.PrintInfo("Enter master password to verify")

	password, err := ui.Password("Master password", "••••••••")
	if err != nil {
		return fmt.Errorf("Cancelled")
	}

	ui.PrintInfo("Verifying password...")

	meta, err := vault.LoadMetaFromStorage(info)
	if err != nil {
		ui.PrintError("Failed to load vault. Wrong password?")
		return fmt.Errorf("failed to load vault")
	}

	authSvc := auth.New()
	if _, err := authSvc.DecryptVaultKey(meta.EncryptedKey, password); err != nil {
		ui.PrintError("Invalid password")
		return fmt.Errorf("invalid password")
	}

	now := time.Now()
	config := &models.VaultConfig{
		Username:    info.Username,
		VaultID:     info.VaultID,
		StorageType: info.StorageType,
		StoragePath: vault.StoragePath(info.Username, info.VaultID),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := vault.SaveConfig(config); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to save config: %v", err))
		return fmt.Errorf("failed to save config")
	}

	fmt.Println()
	ui.PrintSuccess("Vault linked!")
	fmt.Printf("   Username: %s\n", ui.InfoStyle.Render(info.Username))
	fmt.Printf("   Vault: %s\n", ui.InfoStyle.Render(info.VaultID))
	fmt.Printf("   Storage: %s\n", ui.InfoStyle.Render(info.StorageType))
	fmt.Println()
	ui.PrintInfo("Run 'login' to authenticate")

	return nil
}

func runCreateNew(info vault.VaultInfo) error {
	ui.PrintBold("-- Create new vault --")
	fmt.Println()

	create, err := ui.RunCreate()
	if err != nil {
		return fmt.Errorf("Cancelled")
	}

	if err := startValidateCreate(create); err != nil {
		return err
	}

	if err := vault.SetupStorage(info); err != nil {
		return err
	}

	fmt.Println()
	stopSpinner := ui.PrintSpinner("Creating vault...")

	infoWithPass := vault.VaultInfoWithPassword{
		VaultInfo: info,
		Password:  create.Password,
	}
	err = vault.CreateVault(infoWithPass)
	stopSpinner()

	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create vault: %v", err))
		return fmt.Errorf("failed to create vault")
	}

	fmt.Println()
	ui.PrintSuccess("Vault created!")
	fmt.Printf("   Username: %s\n", ui.InfoStyle.Render(info.Username))
	fmt.Printf("   Vault: %s\n", ui.InfoStyle.Render(info.VaultID))
	fmt.Printf("   Storage: %s\n", ui.InfoStyle.Render(info.StorageType))
	fmt.Println()
	ui.PrintInfo("Run 'login' to authenticate")

	return nil
}

func startValidateCreate(state *ui.CreateState) error {
	if state.Password != state.ConfirmPassword {
		return fmt.Errorf("passwords do not match")
	}
	return ui.ValidatePassword(state.Password)
}

// <--- Cobra --->

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Create a new vault",
	Long:  "Launch the wizard to create a new vault or link to an existing one.",
	RunE:  runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}
