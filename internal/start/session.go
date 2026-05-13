package start

import (
	"time"

	"github.com/sthbryan/vaulty/v2/internal/vault"
	"github.com/sthbryan/vaulty/v2/pkg/models"
)

func SessionExists() bool {
	return vault.SessionExists()
}

func LoadSession() (*models.Session, error) {
	return vault.LoadSession()
}

func CreateSession(username, vaultID, storageType string, hours int64) error {
	return vault.CreateSession(username, vaultID, storageType, int(hours))
}

func ExtendSession(session *models.Session, hours int) error {
	session.ExpiresAt = session.ExpiresAt.Add(time.Duration(hours) * time.Hour)
	return vault.SaveSession(session)
}