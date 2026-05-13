package session

import (
	"bytes"
	"sync"
	"time"
)

const (
	RoleOwner  = "owner"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

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

var (
	sessionManager = &SessionManager{
		sessions: make(map[string]*Session),
	}
)

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

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

func (s *Session) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.isLocked && s.MasterKey != nil
}

func (s *Session) IsExpired(ttl time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.LastAccessed) > ttl
}

func (s *Session) Lock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isLocked = true
}

func (s *Session) HasExpired() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.isLocked {
		return true, ErrSessionLocked
	}

	return time.Since(s.LastAccessed) > time.Hour, nil
}

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

func (s *Session) UpdateLastAccessed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastAccessed = time.Now()
}

func GetManager() *SessionManager {
	return sessionManager
}

func (sm *SessionManager) Create(session *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[session.Username] = session
}

func (sm *SessionManager) Get(username string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[username]
}

func (sm *SessionManager) Delete(username string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[username]; exists {
		session.ClearSensitiveData()
		delete(sm.sessions, username)
	}
}

func (sm *SessionManager) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, session := range sm.sessions {
		session.ClearSensitiveData()
	}

	sm.sessions = make(map[string]*Session)
}

func (sm *SessionManager) All() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	usernames := make([]string, 0, len(sm.sessions))
	for username := range sm.sessions {
		usernames = append(usernames, username)
	}
	return usernames
}
