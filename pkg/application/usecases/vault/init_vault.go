package vault

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/crypto"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/session"
	"github.com/sthbryan/vaulty/internal/storage"
)

type InitVaultInput struct {
	Username     string
	Password     string
	Environments []string
	IsLocalMode  bool
}

type InitVaultOutput struct {
	Session  *session.Session
	Metadata *config.Metadata
}

type InitVaultUseCase struct {
	storageFactory *storage.Factory
}

func NewInitVaultUseCase(factory *storage.Factory) *InitVaultUseCase {
	return &InitVaultUseCase{storageFactory: factory}
}

func (uc *InitVaultUseCase) ExecuteGitHub(ctx context.Context, input InitVaultInput) (*InitVaultOutput, error) {
	s, err := uc.storageFactory.CreateStorage()
	if err != nil {
		return nil, fmt.Errorf("creating storage: %w", err)
	}

	owner, repo, err := s.GetOwnerAndRepo()
	if err != nil {
		return nil, fmt.Errorf("getting repo info: %w", err)
	}

	client, err := github.GetAuthenticatedClient()
	if err != nil {
		return nil, err
	}
	exists, err := client.RepoExists(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("checking repository: %w", err)
	}
	if !exists {
		fullRepo := fmt.Sprintf("%s/%s", owner, repo)
		if err := github.CreateRepoWithCLI(fullRepo); err != nil {
			return nil, fmt.Errorf("creating repository: %w", err)
		}
	}

	masterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		return nil, fmt.Errorf("generating master key: %w", err)
	}

	encryptedMasterKey, err := crypto.EncryptMasterKeyWithPassword(masterKey, input.Password)
	if err != nil {
		return nil, fmt.Errorf("encrypting master key: %w", err)
	}

	salt, challenge, err := crypto.GeneratePasswordChallengeStruct(input.Password)
	if err != nil {
		return nil, fmt.Errorf("generating password challenge: %w", err)
	}

	passwordChallenge := &config.PasswordChallenge{
		Salt:      salt,
		Challenge: challenge,
	}

	masterKeyJSON, err := json.Marshal(encryptedMasterKey)
	if err != nil {
		return nil, fmt.Errorf("marshaling master key: %w", err)
	}

	masterKeyHex, err := crypto.CompressHex(masterKeyJSON)
	if err != nil {
		return nil, fmt.Errorf("compressing master key: %w", err)
	}

	err = s.PutUserKeys(ctx, input.Username, []byte(masterKeyHex))
	if err != nil {
		return nil, fmt.Errorf("uploading master key: %w", err)
	}

	emptyVault, err := crypto.EncryptVaultData([]byte{}, masterKey)
	if err != nil {
		return nil, fmt.Errorf("creating vault: %w", err)
	}

	vaultJSON, err := json.Marshal(emptyVault)
	if err != nil {
		return nil, fmt.Errorf("marshaling vault: %w", err)
	}

	vaultHex, err := crypto.CompressHex(vaultJSON)
	if err != nil {
		return nil, fmt.Errorf("compressing vault: %w", err)
	}

	err = s.PutVault(ctx, []byte(vaultHex))
	if err != nil {
		return nil, fmt.Errorf("uploading vault: %w", err)
	}

	environments := input.Environments
	if len(environments) == 0 {
		environments = []string{"production"}
	}

	metadata := &config.Metadata{
		Repo:    fmt.Sprintf("%s/%s", owner, repo),
		Owner:   input.Username,
		Version: "2.1",
		Users: []config.UserEntry{
			{
				Username:          input.Username,
				Role:              "owner",
				CreatedAt:         time.Now(),
				PasswordChallenge: passwordChallenge,
			},
		},
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}

	metadataHex, err := crypto.CompressHex(metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("compressing metadata: %w", err)
	}

	metadataContent := base64.StdEncoding.EncodeToString([]byte(metadataHex))
	err = s.PutContent(ctx, ".vaulty/metadata.vty", metadataContent)
	if err != nil {
		return nil, fmt.Errorf("uploading metadata: %w", err)
	}

	sess := session.NewSession(input.Username, "owner", masterKey, []byte{})

	return &InitVaultOutput{
		Session:  sess,
		Metadata: metadata,
	}, nil
}

func (uc *InitVaultUseCase) ExecuteLocal(ctx context.Context, input InitVaultInput) (*InitVaultOutput, error) {
	s, err := uc.storageFactory.CreateStorage()
	if err != nil {
		return nil, fmt.Errorf("creating storage: %w", err)
	}

	masterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		return nil, fmt.Errorf("generating master key: %w", err)
	}

	encryptedMasterKey, err := crypto.EncryptMasterKeyWithPassword(masterKey, input.Password)
	if err != nil {
		return nil, fmt.Errorf("encrypting master key: %w", err)
	}

	salt, challenge, err := crypto.GeneratePasswordChallengeStruct(input.Password)
	if err != nil {
		return nil, fmt.Errorf("generating password challenge: %w", err)
	}

	passwordChallenge := &config.PasswordChallenge{
		Salt:      salt,
		Challenge: challenge,
	}

	masterKeyJSON, err := json.Marshal(encryptedMasterKey)
	if err != nil {
		return nil, fmt.Errorf("marshaling master key: %w", err)
	}

	masterKeyHex, err := crypto.CompressHex(masterKeyJSON)
	if err != nil {
		return nil, fmt.Errorf("compressing master key: %w", err)
	}

	err = s.PutUserKeys(ctx, input.Username, []byte(masterKeyHex))
	if err != nil {
		return nil, fmt.Errorf("saving master key: %w", err)
	}

	emptyVault, err := crypto.EncryptVaultData([]byte{}, masterKey)
	if err != nil {
		return nil, fmt.Errorf("creating vault: %w", err)
	}

	vaultJSON, err := json.Marshal(emptyVault)
	if err != nil {
		return nil, fmt.Errorf("marshaling vault: %w", err)
	}

	vaultHex, err := crypto.CompressHex(vaultJSON)
	if err != nil {
		return nil, fmt.Errorf("compressing vault: %w", err)
	}

	err = s.PutVault(ctx, []byte(vaultHex))
	if err != nil {
		return nil, fmt.Errorf("saving vault: %w", err)
	}

	metadata := &config.Metadata{
		Repo:    "local",
		Owner:   input.Username,
		Version: "2.1",
		Users: []config.UserEntry{
			{
				Username:          input.Username,
				Role:              "owner",
				CreatedAt:         time.Now(),
				PasswordChallenge: passwordChallenge,
			},
		},
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}

	err = s.PutMetadata(ctx, metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("saving metadata: %w", err)
	}

	sess := session.NewSession(input.Username, "owner", masterKey, []byte{})

	return &InitVaultOutput{
		Session:  sess,
		Metadata: metadata,
	}, nil
}
