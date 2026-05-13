package session

import "errors"

var (
	ErrSessionLocked   = errors.New("session is locked")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrInvalidRole     = errors.New("invalid role")
	ErrEmptyUsername   = errors.New("username cannot be empty")
)
