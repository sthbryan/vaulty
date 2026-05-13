package ui

import "charm.land/huh/v2"

type WizardState struct {
	StorageType     string
	Username        string
	VaultID         string
	Password        string
	ConfirmPassword string
}

type Wizard struct {
	state *WizardState
}

func NewWizard() *Wizard {
	return &Wizard{state: &WizardState{}}
}

func (w *Wizard) Run() (*WizardState, error) {
	form := huh.NewForm(
		SelectGroup("Select storage type", "storage", []SelectOption{
			{ID: "github", Label: "GitHub (encrypted, synced)"},
			{ID: "local", Label: "Local (encrypted, your machine)"},
		}, &w.state.StorageType),
		InputGroup("Username", "username", "your-username", &w.state.Username),
		InputGroup("Vault name", "vault", "my-vault", &w.state.VaultID),
		PasswordGroup("Master password (min 8 chars)", "password", "••••••••", &w.state.Password),
		PasswordGroup("Confirm password", "confirm", "••••••••", &w.state.ConfirmPassword),
	).WithTheme(Theme)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return w.state, nil
}
