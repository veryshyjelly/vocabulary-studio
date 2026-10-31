package vocab

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSessionStoresAndRedaction(t *testing.T) {
	now := time.Date(2026, 10, 31, 15, 0, 0, 0, time.UTC)
	snap := &SessionSnapshot{
		GUID:     "secret_guid_123",
		UserUID:  "u_42",
		Username: "alice",
		Timezone: "America/New_York",
		Token: &TokenInfo{
			TokenType:    "bearer",
			AccessToken:  "secret_access_jwt",
			RefreshToken: "secret_refresh_tok",
			ExpiresAt:    now.Add(time.Hour),
		},
		Cookies: []SerializableCookie{
			{
				RawURL:   "https://www.vocabulary.com",
				Name:     "SESSION",
				Value:    "secret_session_cookie",
				Path:     "/",
				Secure:   true,
				HttpOnly: true,
			},
		},
		UpdatedAt: now,
	}

	// 1. Verify String(), GoString(), and slog redaction
	for _, formatted := range []string{
		fmt.Sprintf("%v", snap),
		fmt.Sprintf("%+v", snap),
		fmt.Sprintf("%#v", snap),
		fmt.Sprintf("%v", snap.Token),
		fmt.Sprintf("%#v", snap.Cookies[0]),
	} {
		for _, secret := range []string{"secret_guid_123", "secret_access_jwt", "secret_refresh_tok", "secret_session_cookie"} {
			if strings.Contains(formatted, secret) {
				t.Errorf("secret %q leaked in formatted output: %s", secret, formatted)
			}
		}
	}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
	logger.Info("dump session", slog.Any("snap", *snap), slog.Any("tok", *snap.Token), slog.Any("cookie", snap.Cookies[0]))
	logOut := logBuf.String()
	for _, secret := range []string{"secret_guid_123", "secret_access_jwt", "secret_refresh_tok", "secret_session_cookie"} {
		if strings.Contains(logOut, secret) {
			t.Errorf("secret %q leaked in slog output: %s", secret, logOut)
		}
	}

	// 2. Verify Clone immutability
	cloned := snap.Clone()
	cloned.Cookies[0].Value = "mutated"
	cloned.Token.AccessToken = "mutated"
	if snap.Cookies[0].Value == "mutated" || snap.Token.AccessToken == "mutated" {
		t.Fatal("Clone() did not perform a deep copy")
	}

	// 3. Verify FileSessionStore atomic write, permissions, and corruption detection
	ctx := context.Background()
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "nested", "session.json")
	store := NewFileSessionStore(sessionFile)

	if err := store.Save(ctx, snap); err != nil {
		t.Fatalf("FileSessionStore.Save: %v", err)
	}
	info, err := os.Stat(sessionFile)
	if err != nil {
		t.Fatalf("stat session file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("session file permissions = %o, want 0600", perm)
	}
	loaded, err := store.Load(ctx)
	if err != nil || loaded == nil || loaded.GUID != "secret_guid_123" || len(loaded.Cookies) != 1 {
		t.Fatalf("FileSessionStore.Load: %+v, err=%v", loaded, err)
	}

	// Write corrupted file and verify ErrCorruptedSession
	if err := os.WriteFile(sessionFile, []byte("{corrupted json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(ctx); !errors.Is(err, ErrCorruptedSession) {
		t.Errorf("expected ErrCorruptedSession for invalid JSON, got %v", err)
	}
}
