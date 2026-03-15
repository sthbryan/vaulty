package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/password"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/storage"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var localMode bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Vaulty with a GitHub repository or local storage",
	Long: `Initialize Vaulty by creating a new vault.

This command will guide you through:
  • Setting up a secure master password
  • Choosing storage mode (GitHub or local)
  • Creating a recovery seed phrase

Use --local for local-only storage without GitHub.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println()
	ui.PrintAnimatedLogo()

	if localMode {
		fmt.Println(ui.TitleStyle.Render("✨ Welcome to Vaulty (Local Mode)!"))
		fmt.Println(ui.MutedStyle.Render("  Secure secret management on your local filesystem"))
	} else {
		fmt.Println(ui.TitleStyle.Render("✨ Welcome to Vaulty!"))
		fmt.Println(ui.MutedStyle.Render("  Secure secret management powered by GitHub"))
	}
	fmt.Println()

	cfg, err := config.Load("")
	if err != nil {
		cfg = &config.Config{}
	}

	if localMode {
		return runInitLocal(cfg)
	}

	return runInitGitHub(cfg)
}

func runInitGitHub(cfg *config.Config) error {
	var repoInput string

	if cfg.Repo != "" {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Existing vault found: %s", cfg.Repo)))

		useExisting, err := ui.AskConfirm("Use this repository?", true)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}

		if useExisting {
			repoInput = cfg.Repo
		}
	}

	if repoInput == "" {
		var vaultOption string
		err := huh.NewSelect[string]().
			Title("Vault name").
			Description("Choose a name for your vault repository").
			Options(
				huh.NewOption("my-vault (default)", "my-vault"),
				huh.NewOption("Custom name", "custom"),
			).
			Value(&vaultOption).
			Run()
		if err != nil {
			return fmt.Errorf("form cancelled")
		}

		if vaultOption == "custom" {
			err := huh.NewInput().
				Title("Enter vault name").
				Placeholder("my-secrets").
				Value(&vaultOption).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("vault name is required")
					}
					if strings.Contains(s, " ") {
						return fmt.Errorf("vault name cannot contain spaces")
					}
					if !regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`).MatchString(s) {
						return fmt.Errorf("vault name can only contain letters, numbers, hyphens and underscores")
					}
					return nil
				}).
				Run()
			if err != nil {
				return fmt.Errorf("form cancelled")
			}
		}

		var ownerInput string
		err = huh.NewInput().
			Title("GitHub owner/organization").
			Placeholder("your-username or org name").
			Value(&ownerInput).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("owner is required")
				}
				return nil
			}).
			Run()
		if err != nil {
			return fmt.Errorf("form cancelled")
		}

		repoInput = ownerInput + "/" + vaultOption
	}

	owner, repo, _ := github.ParseRepo(repoInput)
	repoFull := fmt.Sprintf("%s/%s", owner, repo)

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication: %w", err)
	}

	client := github.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err = client.GetContent(ctx, owner, repo, ".vaulty/metadata.vty")
	if err == nil {
		return fmt.Errorf("vault already exists at %s - use 'vty link' to connect to an existing vault", repoFull)
	}

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Initializing vault in %s...", repoFull)))

	cfg.SetRepo(repoFull)

	passStorage, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("password storage: %w", err)
	}

	if err := initializeNewRepo(ctx, client, owner, repo, cfg, passStorage); err != nil {
		return err
	}

	return nil
}

func initializeNewRepo(ctx context.Context, client *github.Client, owner, repo string, cfg *config.Config, passStorage password.Storage) error {
	fmt.Println(ui.InfoStyle.Render("📦 Creating repository..."))

	_, err := client.ListDirectory(ctx, owner, repo, "")
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Repository %s/%s does not exist, creating...", owner, repo)))
			if err := createGitHubRepo(repo); err != nil {
				return fmt.Errorf("creating repository: %w", err)
			}
			time.Sleep(2 * time.Second)
		} else {
			return fmt.Errorf("checking repository: %w", err)
		}
	}

	fmt.Println(ui.InfoStyle.Render("👤 Set your username"))
	fmt.Println()

	var username string
	err = huh.NewInput().
		Title("Username").
		Placeholder("your-username").
		Value(&username).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("username cannot be empty")
			}
			if len(s) < 3 {
				return fmt.Errorf("username must be at least 3 characters")
			}
			if strings.Contains(s, " ") {
				return fmt.Errorf("username cannot contain spaces")
			}
			if !regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`).MatchString(s) {
				return fmt.Errorf("username can only contain letters, numbers, hyphens and underscores")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("🌍 Define your environments"))
	fmt.Println()

	var selectedEnvs []string
	err = huh.NewMultiSelect[string]().
		Title("Select environments (select multiple)").
		Options(
			huh.NewOption("production", "production"),
			huh.NewOption("staging", "staging"),
			huh.NewOption("development", "development"),
			huh.NewOption("test", "test"),
			huh.NewOption("local", "local"),
		).
		Value(&selectedEnvs).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	var customOption string
	err = huh.NewSelect[string]().
		Title("Add custom environments?").
		Options(
			huh.NewOption("No", "no"),
			huh.NewOption("Yes", "yes"),
		).
		Value(&customOption).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	if customOption == "yes" {
		var customInput string
		err = huh.NewInput().
			Title("Enter custom environments (comma-separated)").
			Placeholder("qa, uat, demo").
			Value(&customInput).
			Run()
		if err != nil {
			return fmt.Errorf("form cancelled")
		}

		envParts := strings.Split(customInput, ",")
		for _, e := range envParts {
			e = strings.TrimSpace(e)
			if e != "" {
				exists := false
				for _, existing := range selectedEnvs {
					if existing == e {
						exists = true
						break
					}
				}
				if !exists {
					selectedEnvs = append(selectedEnvs, e)
				}
			}
		}
	}

	if len(selectedEnvs) == 0 {
		selectedEnvs = []string{"production"}
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("🔐 Create your master password"))
	fmt.Println()

	var password1, password2 string

	err = huh.NewInput().
		Title("Master password").
		Placeholder("Enter a strong password").
		EchoMode(huh.EchoModePassword).
		Value(&password1).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("password cannot be empty")
			}
			if strings.Contains(s, " ") {
				return fmt.Errorf("password cannot contain spaces")
			}
			if len(s) < 8 {
				return fmt.Errorf("password must be at least 8 characters")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	err = huh.NewInput().
		Title("Confirm password").
		Placeholder("Re-enter your password").
		EchoMode(huh.EchoModePassword).
		Value(&password2).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("please confirm your password")
			}
			if s != password1 {
				return fmt.Errorf("passwords do not match")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	masterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		return fmt.Errorf("generating master key: %w", err)
	}

	encryptedMasterKey, err := crypto.EncryptMasterKeyWithPassword(masterKey, password1)
	if err != nil {
		return fmt.Errorf("encrypting master key: %w", err)
	}

	salt, challenge, err := crypto.GeneratePasswordChallengeStruct(password1)
	if err != nil {
		return fmt.Errorf("generating password challenge: %w", err)
	}

	passwordChallenge := &config.PasswordChallenge{
		Salt:      salt,
		Challenge: challenge,
	}

	masterKeyJSON, err := json.Marshal(encryptedMasterKey)
	if err != nil {
		return fmt.Errorf("marshaling master key: %w", err)
	}

	masterKeyHex, err := crypto.CompressHex(masterKeyJSON)
	if err != nil {
		return fmt.Errorf("compressing master key: %w", err)
	}

	err = client.PutUserKeys(ctx, owner, repo, username, []byte(masterKeyHex))
	if err != nil {
		return fmt.Errorf("uploading master key: %w", err)
	}

	emptyVault, err := crypto.EncryptVaultData([]byte{}, masterKey)
	if err != nil {
		return fmt.Errorf("creating vault: %w", err)
	}

	vaultJSON, err := json.Marshal(emptyVault)
	if err != nil {
		return fmt.Errorf("marshaling vault: %w", err)
	}

	vaultHex, err := crypto.CompressHex(vaultJSON)
	if err != nil {
		return fmt.Errorf("compressing vault: %w", err)
	}

	err = client.PutVault(ctx, owner, repo, []byte(vaultHex))
	if err != nil {
		return fmt.Errorf("uploading vault: %w", err)
	}

	metadata := &config.Metadata{
		Repo:    fmt.Sprintf("%s/%s", owner, repo),
		Owner:   username,
		Version: "2.1",
		Users: []config.UserEntry{
			{
				Username:          username,
				Role:              "owner",
				CreatedAt:         time.Now(),
				PasswordChallenge: passwordChallenge,
			},
		},
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	metadataHex, err := crypto.CompressHex(metadataJSON)
	if err != nil {
		return fmt.Errorf("compressing metadata: %w", err)
	}

	metadataContent := base64.StdEncoding.EncodeToString([]byte(metadataHex))
	err = client.PutContent(ctx, owner, repo, ".vaulty/metadata.vty", github.ContentRequest{
		Message: "Add metadata",
		Content: metadataContent,
	})
	if err != nil {
		return fmt.Errorf("uploading metadata: %w", err)
	}

	seedPhrase, err := crypto.GenerateRecoverySeed()
	if err != nil {
		return fmt.Errorf("generating recovery seed: %w", err)
	}

	encryptedSeed, err := crypto.EncryptRecoverySeed(seedPhrase, password1)
	if err != nil {
		return fmt.Errorf("encrypting recovery seed: %w", err)
	}

	encryptedSeedJSON, err := json.Marshal(encryptedSeed)
	if err != nil {
		return fmt.Errorf("marshaling encrypted seed: %w", err)
	}

	recoveryHex, err := crypto.CompressHex(encryptedSeedJSON)
	if err != nil {
		return fmt.Errorf("compressing recovery: %w", err)
	}

	recoveryContent := base64.StdEncoding.EncodeToString([]byte(recoveryHex))
	err = client.PutContent(ctx, owner, repo, ".vaulty/recovery/"+username+".recovery.vty", github.ContentRequest{
		Message: "Add recovery seed",
		Content: recoveryContent,
	})
	if err != nil {
		return fmt.Errorf("uploading recovery seed: %w", err)
	}

	if err := passStorage.Set(password1); err != nil {
		return fmt.Errorf("storing password: %w", err)
	}

	cfg.SetCurrentUser(username, "owner")

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✅ Repository initialized successfully!"))
	fmt.Println()
	fmt.Println(ui.WarningStyle.Render("⚠️  IMPORTANT: Save your recovery seed phrase"))
	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("Recovery seed phrase:"))
	fmt.Println(ui.WarningStyle.Render(seedPhrase))
	fmt.Println()

	saveToFile, err := ui.AskConfirm("Save seed phrase to a file?", true)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}

	if saveToFile {
		var filePath string
		err = huh.NewInput().
			Title("File path").
			Placeholder("vaulty-recovery-seed.txt").
			Value(&filePath).
			Run()
		if err != nil {
			return fmt.Errorf("form cancelled")
		}

		if filePath == "" {
			filePath = "vaulty-recovery-seed.txt"
		}

		err = os.WriteFile(filePath, []byte(seedPhrase), 0600)
		if err != nil {
			return fmt.Errorf("saving seed file: %w", err)
		}

		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Seed phrase saved to: %s", filePath)))
		fmt.Println(ui.MutedStyle.Render("Store this file in a secure location (e.g., password manager)."))
	} else {
		fmt.Println()
		fmt.Println(ui.MutedStyle.Render("Write this down and store it securely. You will need it to recover"))
		fmt.Println(ui.MutedStyle.Render("your vault if you forget your master password."))
	}

	fmt.Println()

	cfg.Metadata = metadata
	cfg.Environments = selectedEnvs

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("🔐 Creating your session..."))
	fmt.Println()

	sess, err := authenticateUser(cfg, password1)
	if err != nil {
		logger.Warn("auto-login failed, you can run 'vty login' manually", "error", err)
	} else {
		mgr := session.GetManager()
		mgr.Create(sess)
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Logged in as %s (%s)", sess.Username, sess.Role)))
	}

	return nil
}

func createGitHubRepo(repoName string) error {
	cmd := exec.Command("gh", "repo", "create", repoName, "--private", "--description", "Vaulty secrets repository", "--confirm")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}
	return nil
}

func runInitLocal(cfg *config.Config) error {
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Initializing local vault in ~/.vty/vault/..."))

	existingStorage, err := storage.NewLocalStorage()
	if err == nil {
		_, err := existingStorage.GetMetadata(context.Background())
		if err == nil {
			return fmt.Errorf("local vault already exists - use 'vty link --local' to connect to an existing local vault")
		}
	}

	cfg.SetLocalMode()

	passStorage, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("password storage: %w", err)
	}

	ctx := context.Background()

	if err := initializeNewLocalVault(ctx, cfg, passStorage); err != nil {
		return err
	}

	return nil
}

func initializeNewLocalVault(ctx context.Context, cfg *config.Config, passStorage password.Storage) error {
	localStorage, err := storage.NewLocalStorage()
	if err != nil {
		return fmt.Errorf("creating local storage: %w", err)
	}

	fmt.Println(ui.InfoStyle.Render("👤 Set your username"))
	fmt.Println()

	var username string
	err = huh.NewInput().
		Title("Username").
		Placeholder("your-username").
		Value(&username).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("username cannot be empty")
			}
			if len(s) < 3 {
				return fmt.Errorf("username must be at least 3 characters")
			}
			if strings.Contains(s, " ") {
				return fmt.Errorf("username cannot contain spaces")
			}
			if !regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`).MatchString(s) {
				return fmt.Errorf("username can only contain letters, numbers, hyphens and underscores")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("🌍 Define your environments"))
	fmt.Println()

	var selectedEnvs []string
	err = huh.NewMultiSelect[string]().
		Title("Select environments (select multiple)").
		Options(
			huh.NewOption("production", "production"),
			huh.NewOption("staging", "staging"),
			huh.NewOption("development", "development"),
			huh.NewOption("test", "test"),
			huh.NewOption("local", "local"),
		).
		Value(&selectedEnvs).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	var customOption string
	err = huh.NewSelect[string]().
		Title("Add custom environments?").
		Options(
			huh.NewOption("No", "no"),
			huh.NewOption("Yes", "yes"),
		).
		Value(&customOption).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	if customOption == "yes" {
		var customInput string
		err = huh.NewInput().
			Title("Enter custom environments (comma-separated)").
			Placeholder("qa, uat, demo").
			Value(&customInput).
			Run()
		if err != nil {
			return fmt.Errorf("form cancelled")
		}

		envParts := strings.Split(customInput, ",")
		for _, e := range envParts {
			e = strings.TrimSpace(e)
			if e != "" {
				exists := false
				for _, existing := range selectedEnvs {
					if existing == e {
						exists = true
						break
					}
				}
				if !exists {
					selectedEnvs = append(selectedEnvs, e)
				}
			}
		}
	}

	if len(selectedEnvs) == 0 {
		selectedEnvs = []string{"production"}
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("🔐 Create your master password"))
	fmt.Println()

	var password1, password2 string

	err = huh.NewInput().
		Title("Master password").
		Placeholder("Enter a strong password").
		EchoMode(huh.EchoModePassword).
		Value(&password1).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("password cannot be empty")
			}
			if strings.Contains(s, " ") {
				return fmt.Errorf("password cannot contain spaces")
			}
			if len(s) < 8 {
				return fmt.Errorf("password must be at least 8 characters")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	err = huh.NewInput().
		Title("Confirm password").
		Placeholder("Re-enter your password").
		EchoMode(huh.EchoModePassword).
		Value(&password2).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("please confirm your password")
			}
			if s != password1 {
				return fmt.Errorf("passwords do not match")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	masterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		return fmt.Errorf("generating master key: %w", err)
	}

	encryptedMasterKey, err := crypto.EncryptMasterKeyWithPassword(masterKey, password1)
	if err != nil {
		return fmt.Errorf("encrypting master key: %w", err)
	}

	salt, challenge, err := crypto.GeneratePasswordChallengeStruct(password1)
	if err != nil {
		return fmt.Errorf("generating password challenge: %w", err)
	}

	passwordChallenge := &config.PasswordChallenge{
		Salt:      salt,
		Challenge: challenge,
	}

	masterKeyJSON, err := json.Marshal(encryptedMasterKey)
	if err != nil {
		return fmt.Errorf("marshaling master key: %w", err)
	}

	masterKeyHex, err := crypto.CompressHex(masterKeyJSON)
	if err != nil {
		return fmt.Errorf("compressing master key: %w", err)
	}

	err = localStorage.PutUserKeys(ctx, username, []byte(masterKeyHex))
	if err != nil {
		return fmt.Errorf("saving master key: %w", err)
	}

	emptyVault, err := crypto.EncryptVaultData([]byte{}, masterKey)
	if err != nil {
		return fmt.Errorf("creating vault: %w", err)
	}

	vaultJSON, err := json.Marshal(emptyVault)
	if err != nil {
		return fmt.Errorf("marshaling vault: %w", err)
	}

	vaultHex, err := crypto.CompressHex(vaultJSON)
	if err != nil {
		return fmt.Errorf("compressing vault: %w", err)
	}

	err = localStorage.PutVault(ctx, []byte(vaultHex))
	if err != nil {
		return fmt.Errorf("saving vault: %w", err)
	}

	metadata := &config.Metadata{
		Repo:    "local://",
		Owner:   username,
		Version: "2.1",
		Users: []config.UserEntry{
			{
				Username:          username,
				Role:              "owner",
				CreatedAt:         time.Now(),
				PasswordChallenge: passwordChallenge,
			},
		},
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	err = localStorage.PutMetadata(ctx, metadataJSON)
	if err != nil {
		return fmt.Errorf("saving metadata: %w", err)
	}

	seedPhrase, err := crypto.GenerateRecoverySeed()
	if err != nil {
		return fmt.Errorf("generating recovery seed: %w", err)
	}

	encryptedSeed, err := crypto.EncryptRecoverySeed(seedPhrase, password1)
	if err != nil {
		return fmt.Errorf("encrypting recovery seed: %w", err)
	}

	encryptedSeedJSON, err := json.Marshal(encryptedSeed)
	if err != nil {
		return fmt.Errorf("marshaling encrypted seed: %w", err)
	}

	recoveryHex, err := crypto.CompressHex(encryptedSeedJSON)
	if err != nil {
		return fmt.Errorf("compressing recovery: %w", err)
	}

	err = localStorage.PutRecoverySeed(ctx, username, []byte(recoveryHex))
	if err != nil {
		return fmt.Errorf("saving recovery seed: %w", err)
	}

	if err := passStorage.Set(password1); err != nil {
		return fmt.Errorf("storing password: %w", err)
	}

	cfg.SetCurrentUser(username, "owner")

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✅ Local vault initialized successfully!"))
	fmt.Println()
	fmt.Println(ui.WarningStyle.Render("⚠️  IMPORTANT: Save your recovery seed phrase"))
	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("Recovery seed phrase:"))
	fmt.Println(ui.WarningStyle.Render(seedPhrase))
	fmt.Println()

	saveToFile, err := ui.AskConfirm("Save seed phrase to a file?", true)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}

	if saveToFile {
		var filePath string
		err = huh.NewInput().
			Title("File path").
			Placeholder("vaulty-recovery-seed.txt").
			Value(&filePath).
			Run()
		if err != nil {
			return fmt.Errorf("form cancelled")
		}

		if filePath == "" {
			filePath = "vaulty-recovery-seed.txt"
		}

		err = os.WriteFile(filePath, []byte(seedPhrase), 0600)
		if err != nil {
			return fmt.Errorf("saving seed file: %w", err)
		}

		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Seed phrase saved to: %s", filePath)))
		fmt.Println(ui.MutedStyle.Render("Store this file in a secure location (e.g., password manager)."))
	} else {
		fmt.Println()
		fmt.Println(ui.MutedStyle.Render("Write this down and store it securely. You will need it to recover"))
		fmt.Println(ui.MutedStyle.Render("your vault if you forget your master password."))
	}

	fmt.Println()

	cfg.Metadata = metadata
	cfg.Environments = selectedEnvs

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("🔐 Creating your session..."))
	fmt.Println()

	sess, err := authenticateUser(cfg, password1)
	if err != nil {
		logger.Warn("auto-login failed, you can run 'vty login' manually", "error", err)
	} else {
		mgr := session.GetManager()
		mgr.Create(sess)
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Logged in as %s (%s)", sess.Username, sess.Role)))
	}

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&localMode, "local", "l", false, "Use local storage instead of GitHub")
}
