package vocab_test

import (
	"testing"

	legacy "github.com/veryshyjelly/vocabulary_go"
	sdk "github.com/veryshyjelly/vocabulary_go/sdk"
)

func TestLegacyImportRetainsSDKTypes(t *testing.T) {
	store := legacy.NewMemorySessionStore()
	client, err := legacy.NewClient(legacy.WithSessionStore(store), legacy.WithPacingGuard(false))
	if err != nil {
		t.Fatal(err)
	}
	// Existing callers can pass legacy types directly to new SDK consumers.
	var canonicalClient *sdk.Client = client
	var canonicalStore sdk.SessionStore = store
	if canonicalClient == nil || canonicalStore == nil {
		t.Fatal("legacy constructors must return the canonical SDK types")
	}
	if legacy.ErrSessionExpired != sdk.ErrSessionExpired {
		t.Fatal("legacy error sentinel identity changed")
	}
}
