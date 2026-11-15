package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
)

type desktopPersistenceStore struct {
	*vocab.MemorySessionStore
	mu   sync.Mutex
	fail bool
}

func (s *desktopPersistenceStore) setFailure(fail bool) {
	s.mu.Lock()
	s.fail = fail
	s.mu.Unlock()
}

func (s *desktopPersistenceStore) Save(ctx context.Context, snap *vocab.SessionSnapshot) error {
	s.mu.Lock()
	fail := s.fail
	s.mu.Unlock()
	if fail {
		return errors.New("unsafe_backend_payload_secret")
	}
	return s.MemorySessionStore.Save(ctx, snap)
}

func newPersistenceTestHub(t *testing.T, store *desktopPersistenceStore) (*Hub, *atomic.Int32) {
	t.Helper()
	dir := t.TempDir()
	prefs := DefaultPreferences()
	prefs.SessionStoreMode = SessionStoreFile
	data, _ := json.Marshal(prefs)
	if err := os.WriteFile(filepath.Join(dir, "preferences.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	calls := &atomic.Int32{}
	transport := storeSwitchTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		var payload any = map[string]bool{}
		switch r.URL.Path {
		case "/auth/me.json":
			payload = vocab.LoginData{ValidUser: true, GUID: "private_guid_value", Auth: vocab.AuthDetails{LoggedIn: true, UID: "user"}}
		case "/1.0/auth/token":
			payload = vocab.TokenInfo{AccessToken: "private_token_value", RefreshToken: "private_guid_value", ExpiresIn: 3600}
		}
		body, _ := json.Marshal(payload)
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})
	hub, err := NewHub(HubOptions{ConfigDir: dir, ClientOpts: []vocab.Option{vocab.WithSessionStore(store), vocab.WithTransport(transport), vocab.WithRateLimit(0, 0)}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(hub.Shutdown)
	return hub, calls
}

func TestDesktopPersistencePartialLoginAndRetry(t *testing.T) {
	ctx := context.Background()
	store := &desktopPersistenceStore{MemorySessionStore: vocab.NewMemorySessionStore(), fail: true}
	hub, calls := newPersistenceTestHub(t, store)
	account, err := hub.Auth.Login(ctx, "user", "password", "")
	if err != nil || !account.Authenticated || account.PersistenceState != "failed" {
		t.Fatal("save failure must preserve authenticated account with failed persistence status")
	}
	before := calls.Load()
	if hub.Auth.GetPersistenceStatus().State != "failed" || calls.Load() != before {
		t.Fatal("status inspection must be local")
	}
	_, err = hub.Auth.RetryPersistence(ctx)
	if err == nil || strings.Contains(err.Error(), "unsafe_backend_payload_secret") {
		t.Fatal("retry failure must use a safe error")
	}
	if hub.Auth.GetPersistenceStatus().State != "failed" {
		t.Fatal("failed retry cleared warning")
	}
	store.setFailure(false)
	status, err := hub.Auth.RetryPersistence(ctx)
	if err != nil || status.State != "saved" {
		t.Fatal("retry did not recover")
	}
	if calls.Load() != before {
		t.Fatal("save retry unexpectedly made network requests")
	}
	if hub.Settings.GetPreferences().SessionStoreMode != SessionStoreFile {
		t.Fatal("retry changed selected storage")
	}
	serialized, _ := json.Marshal(account)
	for _, secret := range []string{"private_guid_value", "private_token_value", "unsafe_backend_payload_secret", "password"} {
		if strings.Contains(string(serialized), secret) {
			t.Fatal("account status leaked sensitive data")
		}
	}
}

func TestDesktopRefreshPersistenceAndMemorySelection(t *testing.T) {
	ctx := context.Background()
	store := &desktopPersistenceStore{MemorySessionStore: vocab.NewMemorySessionStore()}
	hub, _ := newPersistenceTestHub(t, store)
	account, err := hub.Auth.Login(ctx, "user", "password", "")
	if err != nil || account.PersistenceState != "saved" {
		t.Fatal("successful login did not save")
	}
	store.setFailure(true)
	hub.Client().Auth.InvalidateToken()
	if _, err := hub.Client().Auth.EnsureToken(ctx, false); err != nil {
		t.Fatal("failed save invalidated valid token acquisition")
	}
	if hub.Auth.GetPersistenceStatus().State != "failed" {
		t.Fatal("refresh save failure was hidden")
	}
	account, err = hub.Auth.GetStatus(ctx)
	if err != nil || !account.Authenticated || account.PersistenceState != "failed" {
		t.Fatal("status did not preserve authenticated account and persistence failure")
	}
	if err := hub.SwitchSessionStoreMode(ctx, SessionStoreMemory); err != nil {
		t.Fatal(err)
	}
	if hub.Auth.GetPersistenceStatus().State != "saved" {
		t.Fatal("deliberate memory selection did not clear persistence warning")
	}
}
