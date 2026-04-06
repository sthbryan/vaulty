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

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/crypto"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/session"
	"github.com/sthbryan/vaulty/internal/storage"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/models"
)

type ResourceInfo struct {
	Name        string
	Type        models.SecretType
	Tag         string
	IsEncrypted bool
	IsDirectory bool
	Size        int64
}

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

func runInfoLocal(_ *cobra.Command, _ []string, cfg *config.Config, s storage.Storage) error {
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
		metadata = &config.Metadata{}
		if err := json.Unmarshal(metadataData, metadata); err != nil {
			logger.Debug("Could not parse metadata", "error", err)
		}
	}
	if metadata == nil {
		metadata = &config.Metadata{}
	}

	envs, err := s.ListEnvs(ctx)
	if err == nil {
		for _, env := range envs {
			var envName string
			isSharedEnv := strings.HasSuffix(env, ".vty")
			if isSharedEnv {
				envName = strings.TrimSuffix(env, ".vty")
			} else if env == "." || env == "shared" {
				envName = "shared"
			} else {
				envName = env
			}

			if infoEnv != "" && infoEnv != "shared" && infoEnv != envName {
				continue
			}

			envSecrets, err := s.ListEnvSecrets(ctx, env)
			if err != nil {
				logger.Debug("Failed to list env secrets", "env", env, "error", err)
				continue
			}

			var size int64
			if isSharedEnv {
				homeDir, _ := os.UserHomeDir()
				filePath := filepath.Join(homeDir, ".vty", "vault", "envs", env)
				info, _ := os.Stat(filePath)
				if info != nil {
					size = info.Size()
				}
			}

			for _, secretName := range envSecrets {
				secrets = append(secrets, models.SecretInfo{
					Name:        secretName,
					Type:        models.SecretTypeEnv,
					Environment: envName,
					CreatedAt:   time.Time{},
					UpdatedAt:   time.Time{},
				})
			}

			if isSharedEnv && size > 0 {
				secrets = append(secrets, models.SecretInfo{
					Name:        envName,
					Type:        models.SecretTypeEnv,
					Environment: "shared",
					CreatedAt:   time.Time{},
					UpdatedAt:   time.Time{},
					Size:        size,
				})
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

	for _, r := range resources {
		secrets = append(secrets, models.SecretInfo{
			Name:        r.Name,
			Type:        r.Type,
			Environment: r.Tag,
			Size:        r.Size,
		})
	}

	for _, c := range configs {
		secrets = append(secrets, models.SecretInfo{
			Name:        c.Name,
			Type:        c.Type,
			Environment: c.Tag,
			Size:        c.Size,
		})
	}

	sshList, err := s.ListSSHKeys(ctx, cfg.CurrentUser)
	if err == nil {
		for _, key := range sshList {
			sshKeys = append(sshKeys, github.SSHKeyInfo{
				Username: key.Username,
				KeyName:  key.KeyName,
				Size:     key.Size,
			})
		}
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

	renderDetailedVaultInfo(cfg, sess, secrets, sshKeys, cfg.UpdatedAt)
	return nil
}
