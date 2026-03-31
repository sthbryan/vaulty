package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DeadBryam/vaulty/internal/compress"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/spf13/cobra"
)

type BackupEntry struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Env     string `json:"env,omitempty"`
	Path    string `json:"path"`
	Content []byte `json:"content"`
}

type BackupManifest struct {
	Version    string    `json:"version"`
	Exported   time.Time `json:"exported"`
	Owner      string    `json:"owner"`
	Repo       string    `json:"repo"`
	EntryCount int       `json:"entry_count"`
}

func runExport(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	sess, err := ensureAuthenticated(cfg)
	if err != nil {
		return err
	}

	s, err := getStorage(cfg)
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	owner, repo, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("invalid repo in config: %w", err)
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication: %w", err)
	}

	client := github.NewClient(token)

	logger.Info("Starting vault export...")

	entries := []BackupEntry{}

	envs, err := s.ListEnvs(ctx)
	if err == nil {
		for _, env := range envs {
			secrets, err := s.ListEnvSecrets(ctx, env)
			if err != nil {
				logger.Warn("failed to list secrets in env", "env", env)
				continue
			}
			for _, name := range secrets {
				data, err := s.GetEnv(ctx, env, name)
				if err != nil {
					logger.Warn("failed to get env", "env", env, "name", name)
					continue
				}
				entries = append(entries, BackupEntry{
					Type:    "env",
					Name:    name,
					Env:     env,
					Content: data,
				})
			}
		}
	}

	sharedSecrets, err := s.ListEnvSecrets(ctx, "")
	if err == nil {
		for _, name := range sharedSecrets {
			data, err := s.GetEnv(ctx, "", name)
			if err != nil {
				logger.Warn("failed to get shared env", "name", name)
				continue
			}
			entries = append(entries, BackupEntry{
				Type:    "env",
				Name:    name,
				Env:     "",
				Content: data,
			})
		}
	}

	metadata, err := s.ListMetadata(ctx)
	if err == nil {
		for _, name := range metadata {
			data, err := s.GetContent(ctx, name)
			if err != nil {
				continue
			}
			content, err := s.DecodeContent(data)
			if err != nil {
				continue
			}
			entries = append(entries, BackupEntry{
				Type:    "metadata",
				Name:    name,
				Path:    name,
				Content: content,
			})
		}
	}

	resourceInfos, err := listResources(ctx, client, owner, repo, "resources")
	if err == nil {
		logger.Info("Exporting resources", "count", len(resourceInfos))
		for _, r := range resourceInfos {
			path := fmt.Sprintf("resources/%s.vty", r.Name)
			data, err := s.GetResource(ctx, path)
			if err != nil {
				logger.Warn("failed to get resource", "name", r.Name)
				continue
			}
			entries = append(entries, BackupEntry{
				Type:    "resource",
				Name:    r.Name,
				Path:    path,
				Content: data,
			})
		}
	}

	configInfos, err := listResources(ctx, client, owner, repo, "config")
	if err == nil {
		logger.Info("Exporting configs", "count", len(configInfos))
		for _, c := range configInfos {
			path := fmt.Sprintf("config/%s.vty", c.Name)
			data, err := s.GetResource(ctx, path)
			if err != nil {
				logger.Warn("failed to get config", "name", c.Name)
				continue
			}
			entries = append(entries, BackupEntry{
				Type:    "config",
				Name:    c.Name,
				Path:    path,
				Content: data,
			})
		}
	}

	manifest := BackupManifest{
		Version:    "1.0",
		Exported:   time.Now(),
		Owner:      cfg.CurrentUser,
		Repo:       cfg.Repo,
		EntryCount: len(entries),
	}

	manifestJSON, _ := json.Marshal(manifest)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	manifestWriter, err := zw.Create("manifest.json")
	if err == nil {
		manifestWriter.Write(manifestJSON)
	}

	for _, entry := range entries {
		filename := fmt.Sprintf("%s/%s", entry.Type, entry.Name)
		if entry.Env != "" {
			filename = fmt.Sprintf("%s/%s/%s", entry.Type, entry.Env, entry.Name)
		}
		w, err := zw.Create(filename)
		if err != nil {
			continue
		}
		w.Write(entry.Content)
	}

	zw.Close()

	compressed, err := compress.Compress(buf.Bytes())
	if err != nil {
		return fmt.Errorf("compressing backup: %w", err)
	}

	encrypted, err := crypto.EncryptBinary(compressed, sess.MasterKey)
	if err != nil {
		return fmt.Errorf("encrypting backup: %w", err)
	}

	if err := os.WriteFile(exportOutput, []byte(encrypted), 0600); err != nil {
		return fmt.Errorf("writing backup file: %w", err)
	}

	fmt.Printf("✓ Vault exported successfully!\n\n")
	fmt.Printf("  File:    %s\n", exportOutput)
	fmt.Printf("  Size:    %d bytes\n", len(encrypted))
	fmt.Printf("  Entries: %d\n", len(entries))
	fmt.Printf("  Owner:   %s\n", cfg.CurrentUser)
	fmt.Printf("  Repo:    %s\n", cfg.Repo)

	return nil
}

func runImport(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	sess, err := ensureAuthenticated(cfg)
	if err != nil {
		return err
	}

	s, err := getStorage(cfg)
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}

	data, err := os.ReadFile(importInput)
	if err != nil {
		return fmt.Errorf("reading backup file: %w", err)
	}

	decrypted, err := crypto.DecryptBinary(string(data), sess.MasterKey)
	if err != nil {
		if err == crypto.ErrDecryptionFailed {
			return fmt.Errorf("decryption failed: invalid password")
		}
		return fmt.Errorf("decrypting backup: %w", err)
	}

	decompressed, err := compress.Decompress(decrypted)
	if err != nil {
		return fmt.Errorf("decompressing backup: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(decompressed), int64(len(decompressed)))
	if err != nil {
		return fmt.Errorf("reading backup archive: %w", err)
	}

	var manifest BackupManifest
	var entries []BackupEntry

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		if f.Name == "manifest.json" {
			json.Unmarshal(content, &manifest)
			continue
		}

		parts := strings.Split(f.Name, "/")
		if len(parts) < 2 {
			continue
		}

		entry := BackupEntry{
			Type:    parts[0],
			Path:    f.Name,
			Content: content,
		}

		if entry.Type == "env" && len(parts) >= 3 {
			entry.Env = parts[1]
			entry.Name = strings.TrimSuffix(parts[2], ".vty")
		} else {
			entry.Name = strings.TrimSuffix(filepath.Base(f.Name), ".vty")
		}

		entries = append(entries, entry)
	}

	fmt.Printf("Backup file: %s\n", importInput)
	fmt.Printf("  Exported:  %s\n", manifest.Exported.Format(time.RFC3339))
	fmt.Printf("  Entries:   %d\n", manifest.EntryCount)
	fmt.Println()
	fmt.Println(ui.WarningStyle.Render("⚠️  This will WIPE your current vault and replace it with this backup."))
	fmt.Println()

	fmt.Print("Type 'yes' to confirm: ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "yes" {
		fmt.Println("Import cancelled.")
		return nil
	}

	ctx := context.Background()

	fmt.Println("\nWiping current vault...")

	owner, repo, _ := github.ParseRepo(cfg.Repo)
	token, _ := github.GetGitHubToken()
	client := github.NewClient(token)

	wipeEnvs := []string{""}
	wipeEnvs = append(wipeEnvs, cfg.Environments...)
	for _, env := range wipeEnvs {
		var path string
		if env == "" {
			path = "envs"
		} else {
			path = fmt.Sprintf("envs/%s", env)
		}
		items, _ := client.ListDirectory(ctx, owner, repo, path)
		for _, item := range items {
			if strings.HasSuffix(item.Name, ".vty") {
				client.DeleteContent(ctx, owner, repo, fmt.Sprintf("%s/%s", path, item.Name), item.Sha)
			}
		}
	}

	wipeDirs := []string{"resources", "config"}
	for _, dir := range wipeDirs {
		items, _ := client.ListDirectory(ctx, owner, repo, dir)
		for _, item := range items {
			if strings.HasSuffix(item.Name, ".vty") {
				client.DeleteContent(ctx, owner, repo, fmt.Sprintf("%s/%s", dir, item.Name), item.Sha)
			}
		}
	}

	fmt.Println("Vault wiped successfully.")

	fmt.Println("Importing backup...")

	imported := 0
	skipped := 0

	for _, entry := range entries {
		switch entry.Type {
		case "env":
			err := s.PutEnv(ctx, entry.Env, entry.Name, entry.Content)
			if err != nil {
				logger.Warn("failed to import env", "name", entry.Name, "env", entry.Env)
				skipped++
				continue
			}
			imported++

		case "metadata":
			err := s.PutContent(ctx, entry.Path, string(entry.Content))
			if err != nil {
				logger.Warn("failed to import metadata", "path", entry.Path)
				skipped++
				continue
			}
			imported++

		case "resource":
			err := s.PutResource(ctx, entry.Path, entry.Content)
			if err != nil {
				logger.Warn("failed to import resource", "path", entry.Path)
				skipped++
				continue
			}
			imported++

		case "config":
			err := s.PutResource(ctx, entry.Path, entry.Content)
			if err != nil {
				logger.Warn("failed to import config", "path", entry.Path)
				skipped++
				continue
			}
			imported++
		}
	}

	fmt.Printf("\n✓ Import complete!\n\n")
	fmt.Printf("  Imported: %d\n", imported)
	fmt.Printf("  Skipped:  %d\n", skipped)

	return nil
}
