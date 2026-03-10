package password

import (
	"errors"
	"sync"
	"time"
)

const defaultTTL = 15 * time.Minute

type MemoryStorage struct {
	mu        sync.RWMutex
	password  string
	expiresAt time.Time
	cleanupCh chan struct{}
}

func NewMemoryStorage() *MemoryStorage {
	m := &MemoryStorage{
		cleanupCh: make(chan struct{}),
	}
	go m.cleanupLoop()
	return m
}

func (m *MemoryStorage) Get() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.password == "" {
		return "", errors.New("no password stored")
	}

	if time.Now().After(m.expiresAt) {
		return "", errors.New("password expired")
	}

	return m.password, nil
}

func (m *MemoryStorage) Set(password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.password = password
	m.expiresAt = time.Now().Add(defaultTTL)

	return nil
}

func (m *MemoryStorage) Delete() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.password = ""
	m.expiresAt = time.Time{}

	return nil
}

func (m *MemoryStorage) Type() string {
	return "memory"
}

func (m *MemoryStorage) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			if m.password != "" && time.Now().After(m.expiresAt) {
				m.password = ""
				m.expiresAt = time.Time{}
			}
			m.mu.Unlock()
		case <-m.cleanupCh:
			return
		}
	}
}

func (m *MemoryStorage) Stop() {
	close(m.cleanupCh)
}
