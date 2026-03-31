package main

import (
	"testing"

	"github.com/DeadBryam/vaulty/internal/config"
)

func TestPrepareTransferMetadata_UpdatesOwnerAndRoles(t *testing.T) {
	meta := &config.Metadata{
		Owner: "alice",
		Users: []config.UserEntry{
			{Username: "alice", Role: "owner"},
			{Username: "juan", Role: "editor"},
		},
	}

	updated, err := prepareTransferMetadata(meta, "alice", "juan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Owner != "juan" {
		t.Fatalf("expected owner juan, got %s", updated.Owner)
	}

	var aliceRole, juanRole string
	for _, u := range updated.Users {
		if u.Username == "alice" {
			aliceRole = u.Role
		}
		if u.Username == "juan" {
			juanRole = u.Role
		}
	}

	if aliceRole != "editor" {
		t.Fatalf("expected alice role editor, got %s", aliceRole)
	}
	if juanRole != "owner" {
		t.Fatalf("expected juan role owner, got %s", juanRole)
	}
}

func TestPrepareTransferMetadata_FailsWhenNewOwnerMissing(t *testing.T) {
	meta := &config.Metadata{
		Owner: "alice",
		Users: []config.UserEntry{{Username: "alice", Role: "owner"}},
	}

	_, err := prepareTransferMetadata(meta, "alice", "juan")
	if err == nil {
		t.Fatalf("expected error when new owner is missing")
	}
}
