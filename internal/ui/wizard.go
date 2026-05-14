package ui

import (
	"os/user"

	"charm.land/huh/v2"
)

// --- Detect ---

type DetectState struct {
	Username string
	VaultID  string
}

func Detect() (*DetectState, error) {
	state := &DetectState{}

	currentUser, _ := user.Current()
	usernamePlaceholder := ""
	if currentUser != nil {
		usernamePlaceholder = currentUser.Username
	}

	form := huh.NewForm(
		InputGroup("Username", "username", usernamePlaceholder, &state.Username),
		InputGroup("Vault name", "vault", "my-vault", &state.VaultID),
	).WithTheme(Theme)

	if err := form.Run(); err != nil {
		return nil, err
	}

	if state.VaultID == "" {
		state.VaultID = "my-vault"
	}
	if state.Username == "" && currentUser != nil {
		state.Username = currentUser.Username
	}

	return state, nil
}

// --- Identify ---

type IdentifyState struct {
	StorageType string
}

func Identify() (*IdentifyState, error) {
	state := &IdentifyState{}

	form := huh.NewForm(
		SelectGroup("Select storage type", "storage", []SelectOption{
			{ID: "github", Label: "GitHub (encrypted, synced)"},
			{ID: "local", Label: "Local (encrypted, your machine)"},
		}, &state.StorageType),
	).WithTheme(Theme)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return state, nil
}

// --- Create ---

type CreateState struct {
	Password        string
	ConfirmPassword string
}

func RunCreate() (*CreateState, error) {
	state := &CreateState{}
	form := huh.NewForm(
		PasswordGroup("Master password (min 8 chars)", "password", "••••••••", &state.Password),
		PasswordGroup("Confirm password", "confirm", "••••••••", &state.ConfirmPassword),
	).WithTheme(Theme)


	if err := form.Run(); err != nil {
		return nil, err
	}

	return state, nil
}
