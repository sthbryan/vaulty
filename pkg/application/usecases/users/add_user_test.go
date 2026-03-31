package users

import "testing"

func TestOwnerUsernameForKeyLookup_PrefersMetadataOwner(t *testing.T) {
	got := ownerUsernameForKeyLookup("vault-owner", "repo-owner")
	if got != "vault-owner" {
		t.Fatalf("expected metadata owner, got %q", got)
	}
}

func TestOwnerUsernameForKeyLookup_FallsBackToStorageOwner(t *testing.T) {
	got := ownerUsernameForKeyLookup("", "repo-owner")
	if got != "repo-owner" {
		t.Fatalf("expected storage owner fallback, got %q", got)
	}
}
