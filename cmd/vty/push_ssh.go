package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/DeadBryam/vaulty/pkg/models"
	"github.com/spf13/cobra"
)

func runPushSSH(cmd *cobra.Command, args []string) error {
	name := args[0]
	path := args[1]

	if err := validateName(name); err != nil {
		return err
	}

	cfg, client, err := loadConfigAndClient()
	if err != nil {
		return err
	}

	sess, err := ensureAuthenticated(cfg)
	if err != nil {
		return err
	}

	if err := checkPushPermissions(sess.Role); err != nil {
		return err
	}

	if err := validateFile(path); err != nil {
		return err
	}

	vaultFile, originalSize, err := encryptAndPrepareFileWithSession(path, name, models.SecretTypeSSH, sess)
	if err != nil {
		return err
	}

	remotePath := fmt.Sprintf("ssh/%s/%s.vty", sess.Username, name)

	ctx := context.Background()
	owner, repoName, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("invalid repo format: %w", err)
	}

	ui.PrintCloud("Ensuring SSH directory exists for user: %s", sess.Username)
	if err := ensureSSHUserDir(ctx, client, owner, repoName, sess.Username); err != nil {
		return fmt.Errorf("failed to ensure SSH user directory: %w", err)
	}

	encryptedSize, err := encryptAndUploadBinary(client, cfg, remotePath, vaultFile, sess.MasterKey, name)
	if err != nil {
		return err
	}

	ui.PrintSuccess("Pushed SSH key successfully!")
	fmt.Println()
	fmt.Printf("  Name:    %s\n", name)
	fmt.Printf("  User:    %s\n", sess.Username)
	fmt.Printf("  Path:    %s\n", remotePath)
	fmt.Printf("  Size:    %s → %s\n",
		ui.FormatBytes(originalSize),
		ui.FormatBytes(int64(encryptedSize)))
	fmt.Printf("  Repo:    %s\n", cfg.Repo)

	return nil
}

func ensureSSHUserDir(ctx context.Context, client *github.Client, owner, repo, username string) error {
	userDir := fmt.Sprintf("ssh/%s", username)
	placeholderPath := fmt.Sprintf("%s/.gitkeep", userDir)

	_, err := client.GetContent(ctx, owner, repo, userDir)
	if err == nil {
		return nil
	}

	_, err = client.GetContent(ctx, owner, repo, placeholderPath)
	if err == nil {
		return nil
	}

	emptyContent := base64.StdEncoding.EncodeToString([]byte{})
	req := github.ContentRequest{
		Message: fmt.Sprintf("Create SSH directory for user: %s", username),
		Content: emptyContent,
	}

	if err := client.PutContent(ctx, owner, repo, placeholderPath, req); err != nil {
		if !strings.Contains(err.Error(), "422") {
			return err
		}
	}

	return nil
}
