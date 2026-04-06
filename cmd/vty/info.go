package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/crypto"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/storage"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/models"
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

func runInfo(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.IsLocalMode() {
		factory := storage.NewFactory(cfg)
		s, err := factory.CreateStorage()
		if err != nil {
			return err
		}
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
		for _, r := range resources {
			secrets = append(secrets, models.SecretInfo{
				Name: r.Name,
				Type: r.Type,
				Size: r.Size,
			})
		}
	}

	configs, err := listResources(ctx, client, owner, repo, "config")
	if err == nil {
		logger.Info("Listed configs", "count", len(configs))
		for _, c := range configs {
			secrets = append(secrets, models.SecretInfo{
				Name: c.Name,
				Type: c.Type,
				Size: c.Size,
			})
		}
	}

	sshKeys, err = client.ListSSHKeys(ctx, owner, repo, "")
	if err == nil {
		logger.Info("Listed SSH keys", "count", len(sshKeys))
		for _, k := range sshKeys {
			secrets = append(secrets, models.SecretInfo{
				Name: k.KeyName,
				Type: models.SecretTypeSSH,
				Size: int64(k.Size),
			})
		}
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

	renderDetailedVaultInfo(cfg, sess, secrets, sshKeys, cfg.UpdatedAt)
	return nil
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().StringVarP(&infoEnv, "env", "e", "", "Filter by environment")
}
