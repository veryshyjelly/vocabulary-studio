package vocab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newPersistenceStatusClient(t *testing.T, store SessionStore, tokenOutage bool) *Client {
	t.Helper()
	transport := authTransportFunc(func(r *http.Request) (*http.Response, error) {
		body, status := `{}`, http.StatusOK
		switch r.URL.Path {
		case "/auth/me.json":
			body = `{"validUser":true,"guid":"account","auth":{"loggedIn":true,"uid":"user"}}`
		case "/1.0/auth/token":
			body = `{"access_token":"token","refresh_token":"account","expires_in":3600}`
			if tokenOutage {
				status, body = http.StatusServiceUnavailable, `{"error":"unavailable"}`
			}
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	client, err := NewClient(WithSessionStore(store), WithTransport(transport), WithRateLimit(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestWebLoginSurvivesTokenOutage(t *testing.T) {
	ctx := context.Background()
	store := NewMemorySessionStore()
	client := newPersistenceStatusClient(t, store, true)
	me, err := client.Auth.Login(ctx, "user", "password")
	if err != nil || me == nil || !me.ValidUser || !me.Auth.LoggedIn {
		t.Fatal("token endpoint outage invalidated web authentication")
	}
	snapshot := client.ExportSession()
	saved, err := store.Load(ctx)
	if err != nil || saved == nil || saved.GUID != "account" || saved.Token != nil || snapshot.GUID != "account" || snapshot.Token != nil {
		t.Fatal("web session was not preserved without an API token")
	}
}

type persistenceFaultStore struct {
	*MemorySessionStore
	fail    atomic.Bool
	saves   atomic.Int32
	inspect func()
}

func (s *persistenceFaultStore) Save(ctx context.Context, snapshot *SessionSnapshot) error {
	s.saves.Add(1)
	if s.inspect != nil {
		s.inspect()
	}
	if s.fail.Load() {
		return errors.New("backend error containing private-account-data")
	}
	return s.MemorySessionStore.Save(ctx, snapshot)
}

func TestPersistenceStatusFailureAndRetry(t *testing.T) {
	for _, operation := range []string{"login", "refresh"} {
		t.Run(operation, func(t *testing.T) {
			ctx := context.Background()
			store := &persistenceFaultStore{MemorySessionStore: NewMemorySessionStore()}
			store.fail.Store(true)
			client := newPersistenceStatusClient(t, store, false)
			if client.PersistenceStatus().State != "unknown" {
				t.Fatal("new client's persistence status is not unknown")
			}
			// A store may inspect safe health while Save is executing under lifecycle ordering.
			store.inspect = func() { _ = client.PersistenceStatus() }
			if operation == "login" {
				me, err := client.Auth.Login(ctx, "user", "password")
				if err != nil || me == nil || !me.ValidUser || !me.Auth.LoggedIn {
					t.Fatal("save failure invalidated successful web login")
				}
			} else {
				client.ImportSession(&SessionSnapshot{GUID: "account"})
				token, err := client.Auth.EnsureToken(ctx, false)
				if err != nil || token == nil || token.AccessToken != "token" {
					t.Fatal("save failure invalidated successful token acquisition")
				}
			}
			if snapshot := client.ExportSession(); snapshot.GUID != "account" || snapshot.Token == nil {
				t.Fatal("save failure cleared authenticated memory state")
			}
			status := client.PersistenceStatus()
			encoded, err := json.Marshal(status)
			if err != nil || string(encoded) != `{"state":"failed"}` || fmt.Sprint(status) != "{failed}" {
				t.Fatal("persistence status did not safely expose failure")
			}
			status.State = "modified"
			if client.PersistenceStatus().State != "failed" {
				t.Fatal("returned status was not independent")
			}
			if err := client.FlushSession(ctx); err == nil || client.PersistenceStatus().State != "failed" {
				t.Fatal("failed explicit retry was not reported")
			}
			store.fail.Store(false)
			if err := client.FlushSession(ctx); err != nil || client.PersistenceStatus().State != "saved" {
				t.Fatal("successful retry did not clear warning")
			}
			saved, err := store.Load(ctx)
			if err != nil || saved == nil || saved.Token == nil || saved.GUID != "account" {
				t.Fatal("retry did not save the active session")
			}
		})
	}
}

func TestPersistenceStatusStoreSelection(t *testing.T) {
	ctx := context.Background()
	client := newPersistenceStatusClient(t, NewMemorySessionStore(), false)
	client.ImportSession(&SessionSnapshot{GUID: "account"})
	if err := client.FlushSession(ctx); err != nil {
		t.Fatal(err)
	}
	broken := &persistenceFaultStore{MemorySessionStore: NewMemorySessionStore()}
	broken.fail.Store(true)
	if err := client.SetSessionStore(ctx, broken); err == nil || client.PersistenceStatus().State != "saved" {
		t.Fatal("failed destination changed active-backend health")
	}
	broken.fail.Store(false)
	if err := client.SetSessionStore(ctx, broken); err != nil {
		t.Fatal(err)
	}
	broken.fail.Store(true)
	if err := client.FlushSession(ctx); err == nil || client.PersistenceStatus().State != "failed" {
		t.Fatal("selected backend failure was not recorded")
	}
	if err := client.SetSessionStore(ctx, NewMemorySessionStore()); err != nil || client.PersistenceStatus().State != "saved" {
		t.Fatal("successful backend switch did not clear warning")
	}
	client.ImportSession(&SessionSnapshot{GUID: "another"})
	if client.PersistenceStatus().State != "unknown" {
		t.Fatal("import retained previous account's save status")
	}
	if err := client.Auth.Logout(ctx); err != nil || client.PersistenceStatus().State != "unknown" {
		t.Fatal("logout retained account save status")
	}
}

func TestObsoleteAuthenticationCannotChangePersistenceHealth(t *testing.T) {
	for _, operation := range []string{"login", "refresh"} {
		t.Run(operation, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			started, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			transport := authTransportFunc(func(r *http.Request) (*http.Response, error) {
				body := `{}`
				if r.URL.Path == "/auth/me.json" {
					body = `{"validUser":true,"guid":"old","auth":{"loggedIn":true}}`
				}
				if r.URL.Path == "/1.0/auth/token" {
					close(started)
					select {
					case <-release:
					case <-r.Context().Done():
						return nil, r.Context().Err()
					}
					body = `{"access_token":"old-token","refresh_token":"old","expires_in":3600}`
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})
			store := &persistenceFaultStore{MemorySessionStore: NewMemorySessionStore()}
			client, err := NewClient(WithSessionStore(store), WithTransport(transport), WithRateLimit(0, 0))
			if err != nil {
				t.Fatal(err)
			}
			client.ImportSession(&SessionSnapshot{GUID: "old"})
			done := make(chan error, 1)
			go func() {
				if operation == "login" {
					_, err := client.Auth.Login(ctx, "old", "password")
					done <- err
				} else {
					_, err := client.Auth.EnsureToken(ctx, false)
					done <- err
				}
			}()
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal("old authentication did not start")
			}
			client.ImportSession(&SessionSnapshot{GUID: "new"})
			store.fail.Store(true)
			if err := client.FlushSession(ctx); err == nil {
				t.Fatal("expected new-account save failure")
			}
			saves := store.saves.Load()
			unblock()
			select {
			case err := <-done:
				if operation == "refresh" && !errors.Is(err, ErrSessionExpired) {
					t.Fatal("obsolete refresh did not expire")
				}
			case <-ctx.Done():
				t.Fatal("old authentication did not finish")
			}
			if client.PersistenceStatus().State != "failed" || store.saves.Load() != saves {
				t.Fatal("obsolete authentication changed current persistence health or triggered a save")
			}
		})
	}
}

func TestFailedLoginRollbackPreservesPersistenceHealth(t *testing.T) {
	for _, trainer := range []bool{false, true} {
		t.Run(fmt.Sprintf("trainer=%t", trainer), func(t *testing.T) {
			ctx := context.Background()
			store := &persistenceFaultStore{MemorySessionStore: NewMemorySessionStore()}
			store.fail.Store(true)
			transport := authTransportFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"validUser":false,"reason":"invalid"}`)), Request: r}, nil
			})
			client, err := NewClient(WithTransport(transport), WithSessionStore(store), WithRateLimit(0, 0))
			if err != nil {
				t.Fatal(err)
			}
			client.ImportSession(&SessionSnapshot{GUID: "existing"})
			if err := client.FlushSession(ctx); err == nil {
				t.Fatal("expected save failure")
			}
			saves := store.saves.Load()
			if trainer {
				_, err = client.Auth.LoginVocabTrainer(ctx, "wrong", "credentials")
			} else {
				_, err = client.Auth.Login(ctx, "wrong", "credentials")
			}
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatal("expected invalid credentials")
			}
			if client.PersistenceStatus().State != "failed" || client.ExportSession().GUID != "existing" || store.saves.Load() != saves {
				t.Fatal("failed login hid prior persistence failure or changed the restored account")
			}
		})
	}
}

func TestInvalidatedLoginCannotStartReplacementRefresh(t *testing.T) {
	ctx := context.Background()
	store := &persistenceFaultStore{MemorySessionStore: NewMemorySessionStore()}
	client := newPersistenceStatusClient(t, store, false)
	client.ImportSession(&SessionSnapshot{GUID: "old"})
	loginGeneration := client.authGeneration
	client.ImportSession(&SessionSnapshot{GUID: "new"})
	store.fail.Store(true)
	if err := client.FlushSession(ctx); err == nil {
		t.Fatal("expected replacement account's save failure")
	}
	saves := store.saves.Load()
	// Model a login paused after publishing its account but before initiating
	// token acquisition: it must not start work for the replacement generation.
	token, err := client.Auth.ensureToken(ctx, true, &loginGeneration)
	if token != nil || !errors.Is(err, ErrSessionExpired) || store.saves.Load() != saves || client.PersistenceStatus().State != "failed" {
		t.Fatal("obsolete login initiated token work for replacement account")
	}
}

func TestFailedLoginRetainsConcurrentPersistenceFailure(t *testing.T) {
	for _, trainer := range []bool{false, true} {
		for _, replace := range []bool{false, true} {
			t.Run(fmt.Sprintf("trainer=%t/replace=%t", trainer, replace), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				started, release := make(chan struct{}), make(chan struct{})
				var once sync.Once
				unblock := func() { once.Do(func() { close(release) }) }
				defer unblock()
				transport := authTransportFunc(func(r *http.Request) (*http.Response, error) {
					if r.URL.Path != "/auth/me.json" {
						close(started)
						select {
						case <-release:
						case <-r.Context().Done():
							return nil, r.Context().Err()
						}
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"validUser":false,"reason":"invalid"}`)), Request: r}, nil
				})
				store := &persistenceFaultStore{MemorySessionStore: NewMemorySessionStore()}
				client, err := NewClient(WithTransport(transport), WithSessionStore(store), WithRateLimit(0, 0))
				if err != nil {
					t.Fatal(err)
				}
				client.ImportSession(&SessionSnapshot{GUID: "existing"})
				if err := client.FlushSession(ctx); err != nil {
					t.Fatal(err)
				}
				done := make(chan error, 1)
				go func() {
					if trainer {
						_, err := client.Auth.LoginVocabTrainer(ctx, "wrong", "credentials")
						done <- err
					} else {
						_, err := client.Auth.Login(ctx, "wrong", "credentials")
						done <- err
					}
				}()
				select {
				case <-started:
				case <-ctx.Done():
					t.Fatal("login did not start")
				}
				wantGUID := "existing"
				if replace {
					wantGUID = "replacement"
					client.ImportSession(&SessionSnapshot{GUID: wantGUID})
				}
				store.fail.Store(true)
				if err := client.FlushSession(ctx); err == nil {
					t.Fatal("expected concurrent save failure")
				}
				saves := store.saves.Load()
				unblock()
				select {
				case err := <-done:
					if !errors.Is(err, ErrInvalidCredentials) {
						t.Fatal("expected invalid credentials")
					}
				case <-ctx.Done():
					t.Fatal("login did not finish")
				}
				if client.PersistenceStatus().State != "failed" || client.ExportSession().GUID != wantGUID || store.saves.Load() != saves {
					t.Fatal("failed login overwrote current persistence health or replacement account")
				}
			})
		}
	}
}
