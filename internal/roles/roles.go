package roles

import (
	"fmt"
	"time"
)

type Role string

const (
	Owner  Role = "owner"
	Editor Role = "editor"
	Viewer Role = "viewer"
)

func (r Role) CanAddUsers() bool {
	return r == Owner || r == Editor
}

func (r Role) CanRemoveUsers() bool {
	return r == Owner
}

func (r Role) CanTransferOwnership() bool {
	return r == Owner
}

func (r Role) CanEditSecrets() bool {
	return r == Owner || r == Editor
}

func (r Role) CanViewSecrets() bool {
	return r == Owner || r == Editor || r == Viewer
}

func (r Role) CanPullSecrets() bool {
	return r == Owner || r == Editor || r == Viewer
}

func (r Role) String() string {
	return string(r)
}

func ValidateRole(role string) (Role, error) {
	r := Role(role)
	switch r {
	case Owner, Editor, Viewer:
		return r, nil
	default:
		return "", fmt.Errorf("invalid role: %s", role)
	}
}

type UserMetadata struct {
	Username  string
	Role      string
	CreatedAt time.Time
}
