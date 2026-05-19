package main

import (
	"testing"
	"time"

	"github.com/sthbryan/vaulty/v2/pkg/models"
)

func TestSessionIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "expired",
			expiresAt: time.Now().Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "not_expired",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &models.Session{
				ExpiresAt: tt.expiresAt,
			}
			if got := session.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommandErrorLogout(t *testing.T) {
	t.Run("error_message", func(t *testing.T) {
		err := &CommandError{
			Message: "no active session",
			Hint:    "Run 'vty login'",
		}
		if err.Error() != "no active session" {
			t.Errorf("Error() = %v, want %v", err.Error(), "no active session")
		}
	})
}