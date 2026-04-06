package users

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/storage"
)

type RemoveUserInput struct {
	Username      string
	OwnerPassword string
}

type RemoveUserOutput struct {
	RemovedUser string
}

type RemoveUserUseCase struct {
	storageFactory *storage.Factory
}

func NewRemoveUserUseCase(factory *storage.Factory) *RemoveUserUseCase {
	return &RemoveUserUseCase{storageFactory: factory}
}

func parseEncryptedDataFromStorage(data []byte) (*crypto.EncryptedData, error) {
	parsed := &crypto.EncryptedData{}
	if err := json.Unmarshal(data, parsed); err == nil {
		return parsed, nil
	}

	decompressed, err := crypto.DecompressHex(string(data))
	if err != nil {
		return nil, fmt.Errorf("unsupported key format: %w", err)
	}

	if err := json.Unmarshal(decompressed, parsed); err != nil {
		return nil, fmt.Errorf("invalid encrypted key payload: %w", err)
	}

	return parsed, nil
}

func reencryptBinaryPayload(data, oldMasterKey, newMasterKey []byte) ([]byte, error) {
	plaintext, err := crypto.DecryptBinary(string(data), oldMasterKey)
	if err != nil {
		return nil, fmt.Errorf("decrypting old payload: %w", err)
	}

	reencryptedHex, err := crypto.EncryptBinary(plaintext, newMasterKey)
	if err != nil {
		return nil, fmt.Errorf("encrypting payload with new key: %w", err)
	}

	return []byte(reencryptedHex), nil
}

func normalizeResourcePath(path string) string {
	clean := filepath.ToSlash(path)
	if strings.HasPrefix(clean, "resources/") || strings.HasPrefix(clean, "config/") {
		return clean
	}
	return "resources/" + clean
}

func migrateResources(ctx context.Context, s storage.Storage, oldMasterKey, newMasterKey []byte) error {
	paths, err := s.ListResources(ctx)
	if err != nil {
		return fmt.Errorf("listing resources: %w", err)
	}

	for _, path := range paths {
		normalizedPath := normalizeResourcePath(path)
		data, err := s.GetResource(ctx, normalizedPath)
		if err != nil {
			return fmt.Errorf("reading resource %s: %w", normalizedPath, err)
		}

		reencrypted, err := reencryptBinaryPayload(data, oldMasterKey, newMasterKey)
		if err != nil {
			return fmt.Errorf("re-encrypting resource %s: %w", normalizedPath, err)
		}

		if err := s.PutResource(ctx, normalizedPath, reencrypted); err != nil {
			return fmt.Errorf("writing resource %s: %w", normalizedPath, err)
		}
	}

	return nil
}

func migrateEnvs(ctx context.Context, s storage.Storage, oldMasterKey, newMasterKey []byte) error {
	envs, err := s.ListEnvs(ctx)
	if err != nil {
		return fmt.Errorf("listing envs: %w", err)
	}

	for _, env := range envs {
		secrets, err := s.ListEnvSecrets(ctx, env)
		if err != nil {
			return fmt.Errorf("listing env secrets for %s: %w", env, err)
		}

		for _, name := range secrets {
			data, err := s.GetEnv(ctx, env, name)
			if err != nil {
				return fmt.Errorf("reading env secret %s/%s: %w", env, name, err)
			}

			reencrypted, err := reencryptBinaryPayload(data, oldMasterKey, newMasterKey)
			if err != nil {
				return fmt.Errorf("re-encrypting env secret %s/%s: %w", env, name, err)
			}

			if err := s.PutEnv(ctx, env, name, reencrypted); err != nil {
				return fmt.Errorf("writing env secret %s/%s: %w", env, name, err)
			}
		}
	}

	return nil
}

func migrateSSH(ctx context.Context, s storage.Storage, users []config.UserEntry, oldMasterKey, newMasterKey []byte) error {
	for _, user := range users {
		keys, err := s.ListSSHKeys(ctx, user.Username)
		if err != nil {
			continue
		}

		for _, key := range keys {
			data, err := s.GetSSHKey(ctx, user.Username, key.KeyName)
			if err != nil {
				return fmt.Errorf("reading ssh key %s/%s: %w", user.Username, key.KeyName, err)
			}

			reencrypted, err := reencryptBinaryPayload(data, oldMasterKey, newMasterKey)
			if err != nil {
				return fmt.Errorf("re-encrypting ssh key %s/%s: %w", user.Username, key.KeyName, err)
			}

			if err := s.PutSSHKey(ctx, user.Username, key.KeyName, reencrypted); err != nil {
				return fmt.Errorf("writing ssh key %s/%s: %w", user.Username, key.KeyName, err)
			}
		}
	}

	return nil
}

func migrateEncryptedArtifacts(ctx context.Context, s storage.Storage, users []config.UserEntry, oldMasterKey, newMasterKey []byte) error {
	if err := migrateResources(ctx, s, oldMasterKey, newMasterKey); err != nil {
		return err
	}
	if err := migrateEnvs(ctx, s, oldMasterKey, newMasterKey); err != nil {
		return err
	}
	if err := migrateSSH(ctx, s, users, oldMasterKey, newMasterKey); err != nil {
		return err
	}
	return nil
}

func resolveRotatedUserPassword(user config.UserEntry, ownerUsername, ownerPassword string, oldMasterKey []byte) (string, error) {
	if user.Username == ownerUsername {
		if strings.TrimSpace(ownerPassword) == "" {
			return "", fmt.Errorf("missing owner password")
		}
		return ownerPassword, nil
	}

	if strings.TrimSpace(user.EncryptedPassword) == "" {
		return "", fmt.Errorf("missing encrypted password for user %q", user.Username)
	}

	passwordBytes, err := crypto.DecryptBinary(user.EncryptedPassword, oldMasterKey)
	if err != nil {
		return "", fmt.Errorf("decrypting encrypted password for user %q: %w", user.Username, err)
	}

	if len(passwordBytes) == 0 {
		return "", fmt.Errorf("decrypted empty password for user %q", user.Username)
	}

	return string(passwordBytes), nil
}

func buildRotatedUserKeys(users []config.UserEntry, ownerUsername, ownerPassword string, oldMasterKey, newMasterKey []byte) (map[string][]byte, error) {
	rotatedKeys := make(map[string][]byte, len(users))

	for _, user := range users {
		password, err := resolveRotatedUserPassword(user, ownerUsername, ownerPassword, oldMasterKey)
		if err != nil {
			return nil, err
		}

		encryptedKey, err := crypto.EncryptMasterKeyWithPassword(newMasterKey, password)
		if err != nil {
			return nil, fmt.Errorf("encrypting rotated key for user %q: %w", user.Username, err)
		}

		keyJSON, err := json.Marshal(encryptedKey)
		if err != nil {
			return nil, fmt.Errorf("marshaling rotated key for user %q: %w", user.Username, err)
		}

		keyHex, err := crypto.CompressHex(keyJSON)
		if err != nil {
			return nil, fmt.Errorf("compressing rotated key for user %q: %w", user.Username, err)
		}

		rotatedKeys[user.Username] = []byte(keyHex)
	}

	return rotatedKeys, nil
}

func (uc *RemoveUserUseCase) Execute(ctx context.Context, input RemoveUserInput) (*RemoveUserOutput, error) {
	s, err := uc.storageFactory.CreateStorage()
	if err != nil {
		return nil, fmt.Errorf("creating storage: %w", err)
	}

	metadataBytes, err := s.GetMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to download metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("parsing metadata: %w", err)
	}

	var ownerEntry *config.UserEntry
	for i := range metadata.Users {
		if metadata.Users[i].Username == metadata.Owner {
			ownerEntry = &metadata.Users[i]
			break
		}
	}

	if ownerEntry != nil && ownerEntry.PasswordChallenge != nil {
		if !crypto.ValidatePasswordWithChallenge(input.OwnerPassword, ownerEntry.PasswordChallenge.Salt, ownerEntry.PasswordChallenge.Challenge) {
			return nil, fmt.Errorf("password validation failed")
		}
	}

	oldKeyData, err := s.GetUserKeys(ctx, metadata.Owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get old master key: %w", err)
	}

	oldEncryptedData, err := parseEncryptedDataFromStorage(oldKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse old master key: %w", err)
	}

	oldMasterKey, err := crypto.DecryptMasterKeyWithPassword(oldEncryptedData, input.OwnerPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt old master key: %w", err)
	}

	vaultData, err := s.GetVault(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault: %w", err)
	}

	vaultJSON, err := crypto.DecompressHex(string(vaultData))
	if err != nil {
		return nil, fmt.Errorf("decompressing vault: %w", err)
	}

	vaultEncryptedData := &crypto.EncryptedData{}
	if err := json.Unmarshal(vaultJSON, vaultEncryptedData); err != nil {
		return nil, fmt.Errorf("failed to parse vault: %w", err)
	}

	plaintextSecrets, err := crypto.DecryptVaultData(vaultEncryptedData, oldMasterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt vault: %w", err)
	}

	newMasterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new master key: %w", err)
	}

	newVaultEncrypted, err := crypto.EncryptVaultData(plaintextSecrets, newMasterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt vault: %w", err)
	}

	oldUserCount := len(metadata.Users)

	var newUsers []config.UserEntry
	for _, u := range metadata.Users {
		if u.Username != input.Username {
			newUsers = append(newUsers, u)
		}
	}

	if len(newUsers) == oldUserCount {
		return nil, fmt.Errorf("user %q not found in metadata", input.Username)
	}

	metadata.Users = newUsers

	rotatedUserKeys, err := buildRotatedUserKeys(metadata.Users, metadata.Owner, input.OwnerPassword, oldMasterKey, newMasterKey)
	if err != nil {
		return nil, fmt.Errorf("preparing rotated user keys: %w", err)
	}

	if err := migrateEncryptedArtifacts(ctx, s, metadata.Users, oldMasterKey, newMasterKey); err != nil {
		return nil, fmt.Errorf("failed to re-encrypt encrypted artifacts: %w", err)
	}

	updatedMetadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	newVaultJSON, err := json.Marshal(newVaultEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vault: %w", err)
	}

	newVaultHex, err := crypto.CompressHex(newVaultJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to compress vault: %w", err)
	}

	err = s.PutVault(ctx, []byte(newVaultHex))
	if err != nil {
		return nil, fmt.Errorf("failed to upload vault: %w", err)
	}

	for _, user := range metadata.Users {
		rotatedKey, ok := rotatedUserKeys[user.Username]
		if !ok {
			return nil, fmt.Errorf("missing rotated key payload for user %q", user.Username)
		}

		err = s.PutUserKeys(ctx, user.Username, rotatedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to upload key for user %q: %w", user.Username, err)
		}
	}

	err = s.PutMetadata(ctx, updatedMetadataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to upload metadata: %w", err)
	}

	removedUserKeyPath := fmt.Sprintf(".vaulty/keys/%s.vty", input.Username)
	removedUserKeyResp, err := s.GetContent(ctx, removedUserKeyPath)
	if err == nil && removedUserKeyResp != nil {
		err = s.DeleteContent(ctx, removedUserKeyPath, removedUserKeyResp.Sha)
		if err != nil {
			return nil, fmt.Errorf("failed to delete removed user's key file: %w", err)
		}
	}

	sshDirPath := fmt.Sprintf("ssh/%s", input.Username)
	sshItems, err := s.ListDirectory(ctx, sshDirPath)
	if err == nil {
		for _, item := range sshItems {
			if !strings.HasSuffix(item.Name, ".vty") {
				continue
			}
			itemPath := fmt.Sprintf("%s/%s", sshDirPath, item.Name)
			itemResp, err := s.GetContent(ctx, itemPath)
			if err == nil && itemResp != nil {
				err = s.DeleteContent(ctx, itemPath, itemResp.Sha)
				if err != nil {
					return nil, fmt.Errorf("failed to delete SSH key: %w", err)
				}
			}
		}
	}

	return &RemoveUserOutput{
		RemovedUser: input.Username,
	}, nil
}
