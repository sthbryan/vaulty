package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/password"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/spf13/cobra"
)

var removeUserCmd = &cobra.Command{
	Use:   "remove-user <username>",
	Short: "Remove a user from the vault and rotate the master key",
	Long: `Remove a user from Vaulty and rotate the master key.

This command will:
  1. Verify you are the vault owner
  2. Decrypt the current vault with the old master key
  3. Generate a new master key
  4. Re-encrypt the vault with the new key
  5. Re-encrypt the new key for all remaining users
  6. Upload all changes to GitHub
  7. Delete the removed user's key file

This action is irreversible. The removed user will no longer have access to the vault.`,
	Args: cobra.ExactArgs(1),
	RunE: runRemoveUser,
}

func runRemoveUser(cmd *cobra.Command, args []string) error {
	username := args[0]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	if !cfg.IsOwner() {
		return fmt.Errorf("only the vault owner can remove users")
	}

	if username == cfg.CurrentUser {
		return fmt.Errorf("cannot remove yourself from the vault")
	}

	fmt.Println()
	ui.PrintWarning("You are about to remove: %s", username)
	fmt.Println()

	confirmed, err := ui.AskConfirm("Remove "+username+" from vault? This will rotate the masterKey.", false)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}
	if !confirmed {
		ui.PrintInfo("Remove cancelled")
		return nil
	}

	pwd, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("failed to create password storage: %w", err)
	}

	currentPassword, err := pwd.Get()
	if err != nil {
		return fmt.Errorf("password not found, run 'vty init' or 'vty recover'")
	}

	verifyPassword, err := ui.AskPassword("Verify your password")
	if err != nil {
		return fmt.Errorf("password prompt failed: %w", err)
	}

	if verifyPassword != currentPassword {
		return fmt.Errorf("password is incorrect")
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("failed to get GitHub token: %w", err)
	}

	client := github.NewClient(token)
	ctx := context.Background()

	owner, repoName, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("invalid repo format: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Downloading metadata..."))

	metadataBytes, err := client.GetMetadata(ctx, owner, repoName)
	if err != nil {
		return fmt.Errorf("failed to download metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return fmt.Errorf("parsing metadata: %w", err)
	}

	var ownerEntry *config.UserEntry
	for i := range metadata.Users {
		if metadata.Users[i].Username == cfg.CurrentUser {
			ownerEntry = &metadata.Users[i]
			break
		}
	}

	if ownerEntry != nil && ownerEntry.PasswordChallenge != nil {
		if !crypto.ValidatePasswordWithChallenge(verifyPassword, ownerEntry.PasswordChallenge.Salt, ownerEntry.PasswordChallenge.Challenge) {
			fmt.Println()
			fmt.Println(ui.ErrorStyle.Render("❌ Incorrect password"))
			fmt.Println()
			return fmt.Errorf("password validation failed")
		}
	}

	fmt.Println()
	ui.PrintInfo("Downloading old master key...")

	oldKeyResp, err := client.GetUserKeys(ctx, owner, repoName, cfg.Metadata.Owner)
	if err != nil {
		return fmt.Errorf("failed to get old master key: %w", err)
	}

	oldKeyData, err := client.DecodeContent(oldKeyResp)
	if err != nil {
		return fmt.Errorf("failed to decode old master key: %w", err)
	}

	oldEncryptedData := &crypto.EncryptedData{}
	if err := json.Unmarshal(oldKeyData, oldEncryptedData); err != nil {
		return fmt.Errorf("failed to parse old master key: %w", err)
	}

	oldMasterKey, err := crypto.DecryptMasterKeyWithPassword(oldEncryptedData, currentPassword)
	if err != nil {
		return fmt.Errorf("failed to decrypt old master key: %w", err)
	}

	ui.PrintInfo("Downloading vault...")

	vaultResp, err := client.GetVault(ctx, owner, repoName)
	if err != nil {
		return fmt.Errorf("failed to get vault: %w", err)
	}

	vaultData, err := client.DecodeContent(vaultResp)
	if err != nil {
		return fmt.Errorf("failed to decode vault: %w", err)
	}

	vaultEncryptedData := &crypto.EncryptedData{}
	if err := json.Unmarshal(vaultData, vaultEncryptedData); err != nil {
		return fmt.Errorf("failed to parse vault: %w", err)
	}

	ui.PrintInfo("Decrypting vault...")

	plaintextSecrets, err := crypto.DecryptVaultData(vaultEncryptedData, oldMasterKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt vault: %w", err)
	}

	ui.PrintInfo("Generating new master key...")

	newMasterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		return fmt.Errorf("failed to generate new master key: %w", err)
	}

	ui.PrintInfo("Re-encrypting vault with new master key...")

	newVaultEncrypted, err := crypto.EncryptVaultData(plaintextSecrets, newMasterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt vault: %w", err)
	}

	ui.PrintInfo("Removing user from metadata...")

	oldUserCount := len(metadata.Users)

	var newUsers []config.UserEntry
	for _, u := range metadata.Users {
		if u.Username != username {
			newUsers = append(newUsers, u)
		}
	}

	if len(newUsers) == oldUserCount {
		return fmt.Errorf("user %q not found in metadata", username)
	}

	metadata.Users = newUsers

	updatedMetadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	ui.PrintInfo("Re-encrypting new master key for remaining users...")

	keysToUpload := make(map[string][]byte)

	for _, u := range metadata.Users {
		var userPassword string

		if u.Username == cfg.CurrentUser {
			userPassword = currentPassword
		} else {
			var err error
			userPassword, err = ui.AskPassword(fmt.Sprintf("Enter password for user %s", u.Username))
			if err != nil {
				return fmt.Errorf("password prompt failed for %s: %w", u.Username, err)
			}
		}

		encryptedNewKey, err := crypto.EncryptMasterKeyWithPassword(newMasterKey, userPassword)
		if err != nil {
			return fmt.Errorf("failed to encrypt master key for %s: %w", u.Username, err)
		}

		keyJSON, err := json.Marshal(encryptedNewKey)
		if err != nil {
			return fmt.Errorf("failed to marshal encrypted key: %w", err)
		}

		keysToUpload[u.Username] = keyJSON
	}

	fmt.Println()
	ui.PrintInfo("Uploading changes to GitHub...")

	newVaultJSON, err := json.Marshal(newVaultEncrypted)
	if err != nil {
		return fmt.Errorf("failed to marshal vault: %w", err)
	}

	err = client.PutVault(ctx, owner, repoName, newVaultJSON)
	if err != nil {
		return fmt.Errorf("failed to upload vault: %w", err)
	}

	err = client.PutMetadata(ctx, owner, repoName, updatedMetadataJSON)
	if err != nil {
		return fmt.Errorf("failed to upload metadata: %w", err)
	}

	for username, keyJSON := range keysToUpload {
		err = client.PutUserKeys(ctx, owner, repoName, username, keyJSON)
		if err != nil {
			return fmt.Errorf("failed to upload keys for %s: %w", username, err)
		}
	}

	ui.PrintInfo("Deleting removed user's key file...")

	removedUserKeyPath := fmt.Sprintf("keys/%s.enc", username)
	removedUserKeyResp, err := client.GetContent(ctx, owner, repoName, removedUserKeyPath)
	if err == nil && removedUserKeyResp != nil {
		err = client.DeleteContent(ctx, owner, repoName, removedUserKeyPath, removedUserKeyResp.Sha)
		if err != nil {
			return fmt.Errorf("failed to delete removed user's key file: %w", err)
		}
	}

	fmt.Println()
	ui.PrintSuccess("✅ %s removed, masterKey rotated, all users re-encrypted", username)
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(removeUserCmd)
}
