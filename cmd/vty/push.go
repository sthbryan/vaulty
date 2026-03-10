package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DeadBryam/vaulty/internal/compress"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/DeadBryam/vaulty/pkg/models"
	"github.com/spf13/cobra"
)

// BinaryVaultFile represents the structure stored in binary .vty files
type BinaryVaultFile struct {
	Metadata models.SecretMetadata `json:"metadata"`
	Data     []byte                `json:"data"`
}

var (
	pushForce bool
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push secrets to Vaulty",
	Long:  `Push environment files or SSH keys to your Vaulty repository.`,
}

var pushEnvCmd = &cobra.Command{
	Use:   "env <name> <path>",
	Short: "Push an environment file to Vaulty",
	Long: `Compress, encrypt, and upload an environment file to your Vaulty repository.

The file will be:
  1. Compressed using gzip for efficiency
  2. Encrypted using AES-256-GCM with your password
  3. Uploaded to your GitHub repository in the envs/ directory

Examples:
  vty push env production .env.production
  vty push env staging .env.staging --force`,
	Args: cobra.ExactArgs(2),
	RunE: runPushEnv,
}

var pushSSHCmd = &cobra.Command{
	Use:   "ssh <name> <path>",
	Short: "Push an SSH key to Vaulty",
	Long: `Compress, encrypt, and upload an SSH private key to your Vaulty repository.

The file will be:
  1. Compressed using gzip for efficiency
  2. Encrypted using AES-256-GCM with your password
  3. Uploaded to ssh/{username}/{name}.vty in your repository

Only owners and editors can push SSH keys to their own directory.
Viewers cannot push any secrets.

Examples:
  vty push ssh laptop ~/.ssh/id_rsa
  vty push ssh server ~/.ssh/server_key --force`,
	Args: cobra.ExactArgs(2),
	RunE: runPushSSH,
}

func checkPushPermissions(role string) error {
	if role == "" {
		return fmt.Errorf("no active session. Run 'vty login' first")
	}
	if role == "viewer" {
		return fmt.Errorf("viewers cannot push secrets. Contact the repository owner for access")
	}
	return nil
}

func validateName(name string) error {
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("name cannot contain path separators")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("name cannot start with a dot")
	}
	return nil
}

func validateFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path must be a file, not a directory: %s", path)
	}
	return nil
}

func loadConfigAndClient() (*config.Config, *github.Client, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, fmt.Errorf("configuration error: %w", err)
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}
	client := github.NewClient(token)

	return cfg, client, nil
}

func encryptAndPrepareFileWithSession(path, name string, secretType models.SecretType, sess *session.Session) (*BinaryVaultFile, int64, error) {
	ui.PrintInfo("Reading file: %s", path)

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read file: %w", err)
	}

	originalSize := int64(len(content))
	ui.PrintStats("Original size: %s", ui.FormatBytes(originalSize))

	hash := sha256.Sum256(content)
	checksum := fmt.Sprintf("%x", hash)

	ui.PrintInfo("Compressing...")
	compressed, err := compress.Compress(content)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to compress: %w", err)
	}

	compressedSize := int64(len(compressed))
	ui.PrintStats("Compressed size: %s (%.1f%% reduction)",
		ui.FormatBytes(compressedSize),
		float64(originalSize-compressedSize)/float64(originalSize)*100)

	vaultFile := &BinaryVaultFile{
		Metadata: models.SecretMetadata{
			Name:      name,
			Type:      secretType,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Size:      originalSize,
			Checksum:  checksum,
		},
		Data: compressed,
	}

	return vaultFile, originalSize, nil
}

func encryptAndUploadBinary(client *github.Client, cfg *config.Config, remotePath string, vaultFile *BinaryVaultFile, masterKey []byte, name string) (int, error) {
	ui.PrintLock("Encrypting as binary...")

	vaultData, err := json.Marshal(vaultFile)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal vault file: %w", err)
	}

	hexEncrypted, err := crypto.EncryptBinary(vaultData, masterKey)
	if err != nil {
		return 0, fmt.Errorf("failed to encrypt binary: %w", err)
	}

	if err := uploadToGitHub(client, cfg, remotePath, []byte(hexEncrypted), name); err != nil {
		return 0, err
	}

	return len(hexEncrypted), nil
}

func uploadToGitHub(client *github.Client, cfg *config.Config, remotePath string, vaultData []byte, name string) error {
	ctx := context.Background()
	owner, repoName, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("invalid repo format: %w", err)
	}

	ui.PrintCloud("Checking remote: %s/%s/%s", owner, repoName, remotePath)

	var existingSha string
	existingContent, err := client.GetContent(ctx, owner, repoName, remotePath)
	if err == nil && existingContent != nil {
		if !pushForce {
			ui.PrintWarning("File already exists on remote")
			confirmed, confirmErr := ui.AskConfirm("Overwrite existing file?", false)
			if confirmErr != nil {
				return fmt.Errorf("confirmation failed: %w", confirmErr)
			}
			if !confirmed {
				ui.PrintInfo("Push cancelled")
				return nil
			}
		}
		existingSha = existingContent.Sha
		ui.PrintInfo("Will overwrite existing file")
	}

	ui.PrintCloud("Uploading to GitHub...")

	encodedContent := base64.StdEncoding.EncodeToString(vaultData)
	commitMsg := fmt.Sprintf("Update %s via Vaulty push", name)
	if existingSha == "" {
		commitMsg = fmt.Sprintf("Add %s via Vaulty push", name)
	}

	req := github.ContentRequest{
		Message: commitMsg,
		Content: encodedContent,
		Sha:     existingSha,
	}

	if err := client.PutContent(ctx, owner, repoName, remotePath, req); err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	return nil
}

func runPushEnv(cmd *cobra.Command, args []string) error {
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

	vaultFile, originalSize, err := encryptAndPrepareFileWithSession(path, name, models.SecretTypeEnv, sess)
	if err != nil {
		return err
	}

	remotePath := fmt.Sprintf("envs/%s.vty", name)
	encryptedSize, err := encryptAndUploadBinary(client, cfg, remotePath, vaultFile, sess.MasterKey, name)
	if err != nil {
		return err
	}

	ui.PrintSuccess("Pushed successfully!")
	fmt.Println()
	fmt.Printf("  Name:    %s\n", name)
	fmt.Printf("  Path:    %s\n", remotePath)
	fmt.Printf("  Size:    %s → %s\n",
		ui.FormatBytes(originalSize),
		ui.FormatBytes(int64(encryptedSize)))
	fmt.Printf("  Repo:    %s\n", cfg.Repo)

	return nil
}

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

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.AddCommand(pushEnvCmd)
	pushCmd.AddCommand(pushSSHCmd)

	pushEnvCmd.Flags().BoolVarP(&pushForce, "force", "f", false, "Overwrite without prompting")
	pushSSHCmd.Flags().BoolVarP(&pushForce, "force", "f", false, "Overwrite without prompting")
}
