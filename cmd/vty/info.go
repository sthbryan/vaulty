package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/storage"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/DeadBryam/vaulty/pkg/models"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

var (
	infoCmd = &cobra.Command{
		Use:   "info",
		Short: "Show vault contents and metadata",
		Long: `Display all secrets stored in your Vaulty vault.

Shows name, type, size, and when each secret was last updated.
Requires an active session (use 'vty login' first).`,
		RunE: runInfo,
	}

	infoEnv string
)

func fetchAndDecryptVtyFile(ctx context.Context, client *github.Client, owner, repo, path string, masterKey []byte) ([]byte, error) {
	content, err := client.GetContent(ctx, owner, repo, path)
	if err != nil {
		return nil, fmt.Errorf("fetching content: %w", err)
	}

	encodedData, err := client.DecodeContent(content)
	if err != nil {
		return nil, fmt.Errorf("decoding content: %w", err)
	}

	hexData := string(encodedData)
	plaintext, err := crypto.DecryptBinary(hexData, masterKey)
	if err != nil {
		if err == crypto.ErrDecryptionFailed {
			return nil, fmt.Errorf("decryption failed: invalid password")
		}
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	return plaintext, nil
}

func listEnvSecrets(ctx context.Context, client *github.Client, owner, repo, env string, masterKey []byte) ([]models.SecretInfo, error) {
	var path string
	if env == "shared" {
		path = "envs"
	} else {
		path = fmt.Sprintf("envs/%s", env)
	}

	items, err := client.ListDirectory(ctx, owner, repo, path)
	if err != nil {
		return nil, err
	}

	var secrets []models.SecretInfo
	for _, item := range items {
		if strings.HasSuffix(item.Name, ".vty") {
			name := strings.TrimSuffix(item.Name, ".vty")
			filePath := fmt.Sprintf("%s/%s", path, item.Name)

			decryptedContent, decryptErr := fetchAndDecryptVtyFile(ctx, client, owner, repo, filePath, masterKey)
			var size int64
			if decryptErr == nil {
				size = int64(len(decryptedContent))
			} else {
				size = int64(item.Size)
				logger.Debug("Could not decrypt env file", "name", name, "error", decryptErr)
			}

			secrets = append(secrets, models.SecretInfo{
				Name:        name,
				Type:        models.SecretTypeEnv,
				Environment: env,
				CreatedAt:   time.Time{},
				UpdatedAt:   time.Time{},
				Size:        size,
			})
		}
	}

	return secrets, nil
}

type ResourceInfo struct {
	Name        string
	Type        models.SecretType
	Tag         string
	IsEncrypted bool
	IsDirectory bool
	Size        int64
}

func listResources(ctx context.Context, client *github.Client, owner, repo, baseDir string) ([]ResourceInfo, error) {
	resources, err := listResourcesInDir(ctx, client, owner, repo, baseDir)
	if err != nil {
		return nil, err
	}

	subdirs, err := client.ListDirectory(ctx, owner, repo, baseDir)
	if err != nil {
		return resources, nil
	}

	for _, subdir := range subdirs {
		if subdir.Type == "dir" {
			tagResources, err := listResourcesInDir(ctx, client, owner, repo, fmt.Sprintf("%s/%s", baseDir, subdir.Name))
			if err != nil {
				continue
			}
			for i := range tagResources {
				tagResources[i].Tag = subdir.Name
			}
			resources = append(resources, tagResources...)
		}
	}

	return resources, nil
}

func listResourcesInDir(ctx context.Context, client *github.Client, owner, repo, dir string) ([]ResourceInfo, error) {
	items, err := client.ListDirectory(ctx, owner, repo, dir)
	if err != nil {
		return nil, err
	}

	var resources []ResourceInfo
	for _, item := range items {
		if strings.HasSuffix(item.Name, ".vty") {
			name := strings.TrimSuffix(item.Name, ".vty")
			var secretType models.SecretType
			if strings.HasPrefix(dir, "resources") {
				secretType = models.SecretTypeResource
			} else {
				secretType = models.SecretTypeConfig
			}

			resources = append(resources, ResourceInfo{
				Name:        name,
				Type:        secretType,
				IsEncrypted: false,
				Size:        int64(item.Size),
			})
		}
	}

	return resources, nil
}

func listSecretsByEnvironment(ctx context.Context, client *github.Client, cfg *config.Config, owner, repo string, masterKey []byte) ([]models.SecretInfo, error) {
	var allSecrets []models.SecretInfo

	if infoEnv != "" {
		if infoEnv == "shared" {

			secrets, err := listEnvSecrets(ctx, client, owner, repo, "shared", masterKey)
			if err != nil {
				return nil, err
			}
			return secrets, nil
		}

		if !cfg.HasEnvironment(infoEnv) {
			return nil, fmt.Errorf("environment %q not defined in config. Defined: %v", infoEnv, cfg.GetEnvironments())
		}

		secrets, err := listEnvSecrets(ctx, client, owner, repo, infoEnv, masterKey)
		if err != nil {
			return nil, err
		}
		return secrets, nil
	}

	sharedSecrets, err := listEnvSecrets(ctx, client, owner, repo, "shared", masterKey)
	if err != nil {
		logger.Debug("Could not list shared secrets", "error", err)
	}
	allSecrets = append(allSecrets, sharedSecrets...)

	for _, env := range cfg.GetEnvironments() {
		envSecrets, err := listEnvSecrets(ctx, client, owner, repo, env, masterKey)
		if err != nil {
			logger.Debug("Could not list secrets for environment", "env", env, "error", err)
			continue
		}
		allSecrets = append(allSecrets, envSecrets...)
	}

	return allSecrets, nil
}

func runInfoLocal(cmd *cobra.Command, args []string, cfg *config.Config, s storage.Storage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Loading local vault contents..."))

	var secrets []models.SecretInfo
	var sshKeys []github.SSHKeyInfo
	var resources []ResourceInfo
	var configs []ResourceInfo

	metadataData, err := s.GetMetadata(ctx)
	var metadata *config.Metadata
	if err == nil && len(metadataData) > 0 {
		decompressed, err := crypto.DecompressHex(string(metadataData))
		if err == nil {
			metadata = &config.Metadata{}
			if err := json.Unmarshal(decompressed, metadata); err != nil {
				logger.Debug("Could not parse metadata", "error", err)
			}
		}
	}
	if metadata == nil {
		metadata = &config.Metadata{}
	}

	envs, err := s.ListEnvs(ctx)
	if err == nil {
		for _, env := range envs {
			var envName string
			if env == "." {
				envName = "shared"
			} else {
				envName = env
			}

			if infoEnv != "" && infoEnv != "shared" && infoEnv != envName {
				continue
			}

			homeDir, _ := os.UserHomeDir()
			entries, err := os.ReadDir(filepath.Join(homeDir, ".vty", "vault", "envs", env))
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if strings.HasSuffix(name, ".vty") {
					secretName := strings.TrimSuffix(name, ".vty")
					info, _ := entry.Info()
					secrets = append(secrets, models.SecretInfo{
						Name:        secretName,
						Type:        models.SecretTypeEnv,
						Environment: envName,
						CreatedAt:   time.Time{},
						UpdatedAt:   info.ModTime(),
						Size:        info.Size(),
					})
				}
			}
		}
	}

	resourcesList, err := s.ListResources(ctx)
	if err == nil {
		for _, path := range resourcesList {
			if strings.HasSuffix(path, ".vty") {
				name := strings.TrimSuffix(filepath.Base(path), ".vty")
				dir := filepath.Dir(path)
				tag := ""
				if dir != "." {
					tag = dir
				}

				homeDir, _ := os.UserHomeDir()
				absPath := filepath.Join(homeDir, ".vty", "vault", path)
				info, _ := os.Stat(absPath)
				size := int64(0)
				if info != nil {
					size = info.Size()
				}

				if strings.HasPrefix(path, "resources/") || strings.HasPrefix(path, "resources\\") {
					resources = append(resources, ResourceInfo{
						Name:        name,
						Type:        models.SecretTypeResource,
						Tag:         tag,
						IsEncrypted: false,
						Size:        size,
					})
				} else if strings.HasPrefix(path, "config/") || strings.HasPrefix(path, "config\\") {
					configs = append(configs, ResourceInfo{
						Name:        name,
						Type:        models.SecretTypeConfig,
						Tag:         tag,
						IsEncrypted: false,
						Size:        size,
					})
				}
			}
		}
	}

	if len(secrets) == 0 && len(resources) == 0 && len(configs) == 0 && len(metadata.Users) == 0 {
		fmt.Println()
		fmt.Println(ui.InfoStyle.Render("No secrets found in local vault"))
		return nil
	}

	sort.Slice(secrets, func(i, j int) bool {
		if secrets[i].Type == secrets[j].Type {
			return secrets[i].Name < secrets[j].Name
		}
		return secrets[i].Type < secrets[j].Type
	})

	currentUser := cfg.CurrentUser
	if currentUser == "" {
		currentUser = "local"
	}

	sess := &session.Session{
		Username:  currentUser,
		Role:      cfg.CurrentUserRole,
		MasterKey: nil,
	}

	renderDetailedVaultInfo(cfg, sess, secrets, sshKeys, resources, configs, cfg.UpdatedAt)
	return nil
}

func runInfo(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.IsLocalMode() {
		localStorage, err := storage.NewLocalStorage()
		if err != nil {
			return err
		}
		var s storage.Storage
		s = localStorage
		return runInfoLocal(cmd, args, cfg, s)
	}

	sess, err := ensureAuthenticated(cfg)
	if err != nil {
		return err
	}

	owner, repo, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("invalid repo in config: %w", err)
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication: %w", err)
	}

	client := github.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Fetching vault contents..."))

	var secrets []models.SecretInfo
	var sshKeys []github.SSHKeyInfo

	logger.Info("🔓 Loading vault...")
	vaultResp, err := client.GetContent(ctx, owner, repo, ".vaulty/vault.vty")
	if err != nil {
		logger.Warn("Could not fetch vault", "error", err)
	} else {
		vaultData, err := client.DecodeContent(vaultResp)
		if err != nil {
			logger.Warn("Could not decode vault", "error", err)
		} else {
			vaultJSON, err := crypto.DecompressHex(string(vaultData))
			if err != nil {
				logger.Warn("Could not decompress vault", "error", err)
			} else {
				encryptedVault := &crypto.EncryptedData{}
				if err := json.Unmarshal(vaultJSON, encryptedVault); err != nil {
					logger.Warn("Could not parse vault JSON", "error", err)
				}
			}
		}
	}

	envSecrets, err := listSecretsByEnvironment(ctx, client, cfg, owner, repo, sess.MasterKey)
	if err != nil {
		logger.Warn("Could not list environment secrets", "error", err)
	}
	secrets = append(secrets, envSecrets...)

	resources, err := listResources(ctx, client, owner, repo, "resources")
	if err == nil {
		logger.Info("Listed resources", "count", len(resources))
	}

	configs, err := listResources(ctx, client, owner, repo, "config")
	if err == nil {
		logger.Info("Listed configs", "count", len(configs))
	}

	if len(resources) == 0 && len(configs) == 0 && len(secrets) == 0 {
		fmt.Println()
		fmt.Println(ui.InfoStyle.Render("No secrets found in vault"))
		return nil
	}

	sort.Slice(secrets, func(i, j int) bool {
		if secrets[i].Type == secrets[j].Type {
			return secrets[i].Name < secrets[j].Name
		}
		return secrets[i].Type < secrets[j].Type
	})

	renderDetailedVaultInfo(cfg, sess, secrets, sshKeys, resources, configs, cfg.UpdatedAt)
	return nil
}

func renderDetailedVaultInfo(cfg *config.Config, sess *session.Session, secrets []models.SecretInfo, sshKeys []github.SSHKeyInfo, resources []ResourceInfo, configs []ResourceInfo, lastSync time.Time) {
	fmt.Println()

	fmt.Println(ui.MutedStyle.Render("User: " + sess.Username + " (" + sess.Role + ")"))
	fmt.Println()

	if sess.Role == "owner" && cfg.Metadata != nil && len(cfg.Metadata.Users) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== USERS ==="))
		renderUsersTable(cfg.Metadata.Users)
		fmt.Println()
	}

	envSecretsByEnv := make(map[string][]models.SecretInfo)
	var envOrder []string
	var sshSecrets []models.SecretInfo
	for _, s := range secrets {
		if s.Type == models.SecretTypeSSH {
			sshSecrets = append(sshSecrets, s)
		} else {
			env := s.Environment
			if env == "" {
				env = "shared"
			}
			if _, exists := envSecretsByEnv[env]; !exists {
				envOrder = append(envOrder, env)
			}
			envSecretsByEnv[env] = append(envSecretsByEnv[env], s)
		}
	}

	sort.Slice(envOrder, func(i, j int) bool {
		if envOrder[i] == "shared" {
			return true
		}
		if envOrder[j] == "shared" {
			return false
		}
		return envOrder[i] < envOrder[j]
	})

	for _, env := range envOrder {
		sort.Slice(envSecretsByEnv[env], func(i, j int) bool {
			return envSecretsByEnv[env][i].Name < envSecretsByEnv[env][j].Name
		})
	}

	if len(envSecretsByEnv) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== ENVIRONMENT VARIABLES ==="))
		for _, env := range envOrder {
			envSecrets := envSecretsByEnv[env]
			fmt.Printf("\n[%s]\n", ui.HighlightStyle.Render(env))
			renderSecretsTable(envSecrets)
		}
		fmt.Println()
	}

	if len(sshKeys) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== SSH KEYS ==="))
		renderSSHKeysTable(sshKeys, sess.Role)
		fmt.Println()
	}

	if len(resources) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== RESOURCES ==="))
		renderResourcesTable(resources)
		fmt.Println()
	}

	if len(configs) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== CONFIG ==="))
		renderResourcesTable(configs)
		fmt.Println()
	}

	totalSize := int64(0)
	envCount := 0
	sshCount := len(sshKeys)
	for _, s := range secrets {
		totalSize += s.Size
		if s.Type == models.SecretTypeEnv {
			envCount++
		}
	}

	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== SUMMARY ==="))
	fmt.Println()

	fmt.Printf("  Total Secrets: %s (", ui.HighlightStyle.Render(fmt.Sprintf("%d", len(secrets))))
	fmt.Printf("%s env + ", ui.HighlightStyle.Render(fmt.Sprintf("%d", envCount)))
	fmt.Printf("%s ssh)\n", ui.HighlightStyle.Render(fmt.Sprintf("%d", sshCount)))
	fmt.Printf("  Total Size:    %s (", ui.HighlightStyle.Render(formatSize(totalSize)))
	fmt.Printf("%s env + ", ui.HighlightStyle.Render(formatSize(calculateTypeSize(secrets, models.SecretTypeEnv))))
	fmt.Printf("%s ssh)\n", ui.HighlightStyle.Render(formatSize(calculateTypeSize(secrets, models.SecretTypeSSH))))
	fmt.Println()

	if sess.Role == "owner" {
		fmt.Println("  SSH Breakdown:")
		userKeyCounts := make(map[string]int)
		userKeySizes := make(map[string]int64)
		for _, key := range sshKeys {
			userKeyCounts[key.Username]++
			userKeySizes[key.Username] += int64(key.Size)
		}
		for username, count := range userKeyCounts {
			fmt.Printf("    %s: %s (%s)\n",
				ui.HighlightStyle.Render(username),
				ui.HighlightStyle.Render(fmt.Sprintf("%d keys", count)),
				ui.HighlightStyle.Render(formatSize(userKeySizes[username])))
		}
		fmt.Println()
	} else {
		fmt.Printf("  My SSH Keys:   %s\n", ui.HighlightStyle.Render(fmt.Sprintf("%d", sshCount)))
		fmt.Println()
	}

	if sess.Role == "owner" && cfg.Metadata != nil {
		ownerCount := 0
		editorCount := 0
		viewerCount := 0
		for _, u := range cfg.Metadata.Users {
			switch u.Role {
			case "owner":
				ownerCount++
			case "editor":
				editorCount++
			case "viewer":
				viewerCount++
			}
		}
		fmt.Printf("  Users: %s total", ui.HighlightStyle.Render(fmt.Sprintf("%d", len(cfg.Metadata.Users))))
		if len(cfg.Metadata.Users) > 0 {
			fmt.Printf(" (%d owner, %d editor, %d viewer)", ownerCount, editorCount, viewerCount)
		}
		fmt.Println()
		fmt.Println()
	}

	fmt.Printf("  Repository:    %s\n", cfg.Repo)
	fmt.Printf("  Last Sync:     %s\n", ui.HighlightStyle.Render(formatTime(lastSync)))
	fmt.Printf("  Last Updated:  %s\n", ui.HighlightStyle.Render(formatTime(cfg.UpdatedAt)))
	fmt.Printf("  Created:       %s\n", ui.HighlightStyle.Render(formatTime(cfg.CreatedAt)))
	fmt.Println()
}

func renderUsersTable(users []config.UserEntry) {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary))
			}
			if row%2 == 0 {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
			}
			return lipgloss.NewStyle()
		}).
		Headers("USERNAME", "ROLE", "CREATED")

	for _, user := range users {
		t.Row(
			user.Username,
			user.Role,
			formatTime(user.CreatedAt),
		)
	}

	fmt.Println(t.Render())
}

func renderSecretsTable(secrets []models.SecretInfo) {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary))
			}
			if row%2 == 0 {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
			}
			return lipgloss.NewStyle()
		}).
		Headers("NAME", "TYPE", "SIZE", "UPDATED")

	for _, secret := range secrets {
		t.Row(
			secret.Name,
			string(secret.Type),
			formatSize(secret.Size),
			formatTime(secret.UpdatedAt),
		)
	}

	fmt.Println(t.Render())
}

func renderSSHKeysTable(keys []github.SSHKeyInfo, role string) {
	if role == "owner" {
		t := table.New().
			Border(lipgloss.NormalBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == 0 {
					return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary))
				}
				if row%2 == 0 {
					return lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
				}
				return lipgloss.NewStyle()
			}).
			Headers("USERNAME", "KEYNAME", "SIZE")

		for _, key := range keys {
			t.Row(
				key.Username,
				key.KeyName,
				formatSize(int64(key.Size)),
			)
		}

		fmt.Println(t.Render())
	} else {
		t := table.New().
			Border(lipgloss.NormalBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == 0 {
					return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary))
				}
				if row%2 == 0 {
					return lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
				}
				return lipgloss.NewStyle()
			}).
			Headers("KEYNAME", "SIZE")

		for _, key := range keys {
			t.Row(
				key.KeyName,
				formatSize(int64(key.Size)),
			)
		}

		fmt.Println(t.Render())
	}
}

func renderResourcesTable(resources []ResourceInfo) {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary))
			}
			if row%2 == 0 {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
			}
			return lipgloss.NewStyle()
		}).
		Headers("NAME", "TYPE", "TAG", "ENCRYPTED", "DIR", "SIZE")

	for _, r := range resources {
		tag := r.Tag
		if tag == "" {
			tag = "-"
		}
		encrypted := "no"
		if r.IsEncrypted {
			encrypted = "yes"
		}
		dir := "no"
		if r.IsDirectory {
			dir = "yes"
		}
		t.Row(
			r.Name,
			string(r.Type),
			tag,
			encrypted,
			dir,
			formatSize(r.Size),
		)
	}

	fmt.Println(t.Render())
}

func formatSize(bytes int64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
	)
	switch {
	case bytes < KB:
		return fmt.Sprintf("%dB", bytes)
	case bytes < MB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	}
}

func calculateTypeSize(secrets []models.SecretInfo, secretType models.SecretType) int64 {
	var total int64
	for _, s := range secrets {
		if s.Type == secretType {
			total += s.Size
		}
	}
	return total
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	duration := time.Since(t)
	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		return fmt.Sprintf("%d mins ago", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	}
	days := int(duration.Hours()) / 24
	if days == 1 {
		return "1 day ago"
	}
	if days < 7 {
		return fmt.Sprintf("%d days ago", days)
	}
	weeks := days / 7
	if weeks == 1 {
		return "1 week ago"
	}
	if weeks < 4 {
		return fmt.Sprintf("%d weeks ago", weeks)
	}
	return t.Format("2006-01-02")
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().StringVarP(&infoEnv, "env", "e", "", "Filter by environment")
}
