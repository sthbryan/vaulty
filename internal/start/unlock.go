package start

import (
	"fmt"

	"github.com/sthbryan/vaulty/v2/internal/auth"
	"github.com/sthbryan/vaulty/v2/internal/ui"
	"github.com/sthbryan/vaulty/v2/pkg/models"
)

func UnlockWithPassword(config *models.VaultConfig, meta *models.VaultMeta) error {
	ui.PrintInfo("Enter your master password")

	password, err := ui.Password("Master password", "••••••••")
	if err != nil {
		return fmt.Errorf("wizard cancelled")
	}

	authSvc := auth.New()
	_, err = authSvc.DecryptVaultKey(meta.EncryptedKey, password)
	if err != nil {
		ui.PrintError("Invalid password")
		return fmt.Errorf("invalid password")
	}

	return nil
}
