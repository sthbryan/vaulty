package session

import (
	"bytes"
	"sync"
	"time"
)

// Role constants
const (
	RoleOwner  = "owner"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

// Session represents an authenticated user session with encrypted vault data
type Session struct {
	mu           sync.RWMutex
	Username     string
	Role         string
	MasterKey    []byte
	VaultData    []byte
	CreatedAt    time.Time
	LastAccessed time.Time
	isLocked     bool
}

// sessionManager is a global singleton for session management
var (
	sessionManager = &SessionManager{
		sessions: make(map[string]*Session),
	}
)

// SessionManager manages active sessions
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSession creates a new session with the provided credentials and data
func NewSession(username, role string, masterKey, vaultData []byte) *Session {
	now := time.Now()
	return &Session{
		Username:     username,
		Role:         role,
		MasterKey:    bytes.Clone(masterKey),
		VaultData:    bytes.Clone(vaultData),
		CreatedAt:    now,
		LastAccessed: now,
		isLocked:     false,
	}
}

// IsActive checks if the session is currently active and unlocked
func (s *Session) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.isLocked && s.MasterKey != nil
}

// IsExpired checks if the session has exceeded the given TTL
func (s *Session) IsExpired(ttl time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.LastAccessed) > ttl
}

// Lock locks the session, clearing sensitive data access
func (s *Session) Lock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isLocked = true
}

// HasExpired checks if the session has expired and returns error if locked
func (s *Session) HasExpired() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.isLocked {
		return true, ErrSessionLocked
	}

	// Default 1 hour TTL if not specified
	return time.Since(s.LastAccessed) > time.Hour, nil
}

// ClearSensitiveData securely clears all sensitive data from memory
func (s *Session) ClearSensitiveData() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.MasterKey != nil {
		for i := range s.MasterKey {
			s.MasterKey[i] = 0
		}
		s.MasterKey = nil
	}

	if s.VaultData != nil {
		for i := range s.VaultData {
			s.VaultData[i] = 0
		}
		s.VaultData = nil
	}

	s.isLocked = true
}

// UpdateLastAccessed updates the last accessed timestamp
func (s *Session) UpdateLastAccessed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastAccessed = time.Now()
}

// GetManager returns the global session manager singleton
func GetManager() *SessionManager {
	return sessionManager
}

// Create adds a new session to the manager
func (sm *SessionManager) Create(session *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[session.Username] = session
}

// Get retrieves a session by username
func (sm *SessionManager) Get(username string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[username]
}

// Delete removes a session by username and clears its data
func (sm *SessionManager) Delete(username string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[username]; exists {
		session.ClearSensitiveData()
		delete(sm.sessions, username)
	}
}

// Clear removes all sessions
func (sm *SessionManager) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, session := range sm.sessions {
		session.ClearSensitiveData()
	}

	sm.sessions = make(map[string]*Session)
}

// All returns all active sessions (read-only copy of usernames)
func (sm *SessionManager) All() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	usernames := make([]string, 0, len(sm.sessions))
	for username := range sm.sessions {
		usernames = append(usernames, username)
	}
	return usernames
}
