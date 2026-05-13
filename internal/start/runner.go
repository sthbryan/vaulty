package start

import (
	"fmt"

	"github.com/sthbryan/vaulty/v2/internal/ui"
	"github.com/sthbryan/vaulty/v2/pkg/models"
)

type Runner struct {
	SessionDuration string
}

func NewRunner(sessionDuration string) *Runner {
	return &Runner{SessionDuration: sessionDuration}
}

func (r *Runner) Run() error {
	if SessionExists() {
		session, err := LoadSession()
		if err == nil && !session.IsExpired() {
			ui.PrintBold("-- Already unlocked --")
			fmt.Println()
			ui.PrintInfo(fmt.Sprintf("Username: %s", session.Username))
			ui.PrintInfo(fmt.Sprintf("Vault: %s", session.VaultID))
			ui.PrintInfo(fmt.Sprintf("Expires: %s", session.ExpiresAt.Format("2006-01-02 15:04")))
			fmt.Println()
			return nil
		}
	}

	config, err := LoadConfig()
	if err == nil {
		return r.runUnlockLinked(config)
	}

	return r.runIdentify()
}

func (r *Runner) runUnlockLinked(config *models.VaultConfig) error {
	ui.PrintBold("-- Vault linked, unlock required --")
	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Username: %s", config.Username))
	ui.PrintInfo(fmt.Sprintf("Vault: %s", config.VaultID))
	ui.PrintInfo(fmt.Sprintf("Storage: %s", config.StorageType))
	fmt.Println()

	var token string
	var meta *models.VaultMeta
	var err error

	if config.StorageType == "github" {
		token, err = GetGitHubToken()
		if err != nil {
			return fmt.Errorf("cancelled")
		}

		ui.PrintInfo("Loading from GitHub...")
		meta, err = LoadMetaFromGitHubForRunner(token, config.Username, config.VaultID)
		if err != nil {
			if IsFileNotFound(err) {
				return r.runRecreate(config.Username, config.VaultID, token)
			}
			ui.PrintError(fmt.Sprintf("Failed to load vault: %v", err))
			return fmt.Errorf("failed to load vault")
		}
	} else {
		meta, err = LoadMeta()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to load vault meta: %v", err))
			return fmt.Errorf("failed to load vault meta")
		}
	}

	return r.unlockWithPassword(config, meta)
}

func (r *Runner) runUnlock(detect *ui.DetectState, storageType string, token string) error {
	ui.PrintBold("-- Vault found --")
	fmt.Println()

	config := &models.VaultConfig{
		StorageType: storageType,
		StoragePath: GetStoragePath(detect.Username, detect.VaultID),
		Username:    detect.Username,
		VaultID:     detect.VaultID,
	}

	var meta *models.VaultMeta
	var err error

	if storageType == "github" {
		ui.PrintInfo("Loading from GitHub...")
		meta, err = LoadMetaFromGitHubForRunner(token, detect.Username, detect.VaultID)
		if err != nil {
			if IsFileNotFound(err) {
				return r.runRecreate(detect.Username, detect.VaultID, token)
			}
			ui.PrintError(fmt.Sprintf("Failed to load vault: %v", err))
			return fmt.Errorf("failed to load vault")
		}
	} else {
		ui.PrintInfo("Loading from local...")
		meta, err = LoadMeta()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to load vault meta: %v", err))
			return fmt.Errorf("failed to load vault meta")
		}
	}

	return r.unlockWithPassword(config, meta)
}

func (r *Runner) unlockWithPassword(config *models.VaultConfig, meta *models.VaultMeta) error {
	if err := UnlockWithPassword(config, meta); err != nil {
		return err
	}

	if err := CreateSessionFromRunner(config.Username, config.VaultID, config.StorageType, r.SessionDuration); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create session: %v", err))
		return fmt.Errorf("failed to create session")
	}

	fmt.Println()
	ui.PrintSuccess("Vault unlocked!")

	return nil
}

func (r *Runner) runRecreate(username, vaultID, token string) error {
	ui.PrintError("Vault not found in repository")
	fmt.Println()

	createRepo, err := ui.Confirm("Create new vault in this repository?")
	if err != nil {
		return fmt.Errorf("cancelled")
	}

	if !createRepo {
		return fmt.Errorf("operation cancelled")
	}

	ui.PrintInfo("Creating new vault...")
	fmt.Println()

	create, err := ui.NewCreate().Run()
	if err != nil {
		return fmt.Errorf("cancelled")
	}

	if err := ValidateCreate(create); err != nil {
		return err
	}

	stopSpinner := ui.PrintSpinner("Uploading vault...")
	err = CreateVault(username, vaultID, "github", GetStoragePath(username, vaultID), create.Password, token)
	stopSpinner()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create vault: %v", err))
		return fmt.Errorf("failed to create vault")
	}

	if err := CreateSessionFromRunner(username, vaultID, "github", r.SessionDuration); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create session: %v", err))
		return fmt.Errorf("failed to create session")
	}

	fmt.Println()
	ui.PrintSuccess("Vault created!")
	fmt.Printf("   Storage: %s\n", ui.InfoStyle.Render("github"))
	fmt.Printf("   Path: %s\n", ui.InfoStyle.Render(GetStoragePath(username, vaultID)))

	return nil
}

func (r *Runner) runIdentify() error {
	ui.PrintBold("-- Identify vault --")
	fmt.Println()

	detect, err := ui.Detect()
	if err != nil {
		return fmt.Errorf("cancelled")
	}

	if detect.Username == "" {
		detect.Username = GetCurrentUser()
	}
	if detect.VaultID == "" {
		detect.VaultID = "my-vault"
	}

	fmt.Println()
	identify, err := ui.Identify()
	if err != nil {
		return fmt.Errorf("cancelled")
	}

	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Checking vault in %s...", identify.StorageType))

	exists, token := CheckInStorage(detect, identify.StorageType)

	if exists {
		return r.runUnlock(detect, identify.StorageType, token)
	}
	return r.runCreate(detect, identify.StorageType, token)
}

func (r *Runner) runCreate(detect *ui.DetectState, storageType string, token string) error {
	ui.PrintBold("-- Vault not found --")
	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Creating new %s vault...", storageType))
	fmt.Println()

	create, err := ui.NewCreate().Run()
	if err != nil {
		return fmt.Errorf("cancelled")
	}

	if err := ValidateCreate(create); err != nil {
		return err
	}

	var storagePath string

	if storageType == "github" {
		storagePath, err = SetupGitHubStorage(detect.Username, detect.VaultID, token)
		if err != nil {
			return err
		}
	} else {
		storagePath = GetVaultPath(detect.VaultID)
	}

	fmt.Println()
	stopSpinner := ui.PrintSpinner("Creating vault...")
	err = CreateVault(detect.Username, detect.VaultID, storageType, storagePath, create.Password, token)
	stopSpinner()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create vault: %v", err))
		return fmt.Errorf("failed to create vault")
	}

	if err := CreateSessionFromRunner(detect.Username, detect.VaultID, storageType, r.SessionDuration); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create session: %v", err))
		return fmt.Errorf("failed to create session")
	}

	fmt.Println()
	ui.PrintSuccess("Vault created!")
	fmt.Printf("   Storage: %s\n", ui.InfoStyle.Render(storageType))
	fmt.Printf("   Path: %s\n", ui.InfoStyle.Render(storagePath))

	return nil
}