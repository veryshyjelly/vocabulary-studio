package vocab

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type authTransportFunc func(*http.Request) (*http.Response, error)

func (f authTransportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRefreshRejectsInvalidatedAccount(t *testing.T) {
	for _, transition := range []string{"logout", "import", "login"} {
		t.Run(transition, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			started, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			store := NewMemorySessionStore()
			transport := authTransportFunc(func(r *http.Request) (*http.Response, error) {
				var payload any = map[string]bool{}
				header := http.Header{"Content-Type": {"application/json"}}
				switch r.URL.Path {
				case "/auth/me.json":
					payload = LoginData{ValidUser: true, GUID: "new", Auth: AuthDetails{LoggedIn: true, UID: "new-user"}}
				case "/1.0/auth/token":
					_ = r.ParseForm()
					guid := r.FormValue("refresh_token")
					if guid == "old" {
						header.Add("Set-Cookie", "guid=old; Path=/")
						close(started)
						select {
						case <-release:
						case <-r.Context().Done():
							return nil, r.Context().Err()
						}
					}
					payload = TokenInfo{AccessToken: guid + "-token", RefreshToken: guid, ExpiresIn: 3600}
				}
				body, _ := json.Marshal(payload)
				return &http.Response{StatusCode: 200, Header: header, Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
			})
			client, err := NewClient(WithTransport(transport), WithSessionStore(store), WithRateLimit(0, 0))
			if err != nil {
				t.Fatal(err)
			}
			client.ImportSession(&SessionSnapshot{GUID: "old"})
			type result struct {
				token *TokenInfo
				err   error
			}
			done := make(chan result, 1)
			go func() { tok, err := client.Auth.EnsureToken(ctx, false); done <- result{tok, err} }()
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal("old refresh did not start")
			}
			if transition == "logout" {
				if err := client.Auth.Logout(ctx); err != nil {
					t.Fatal(err)
				}
			} else {
				newDone := make(chan error, 1)
				go func() {
					if transition == "login" {
						_, err := client.Auth.Login(ctx, "new-user", "password")
						newDone <- err
					} else {
						client.ImportSession(&SessionSnapshot{GUID: "new", UserUID: "new-user"})
						_, err := client.Auth.EnsureToken(ctx, false)
						newDone <- err
					}
				}()
				select {
				case err := <-newDone:
					if err != nil {
						t.Fatal(err)
					}
				case <-ctx.Done():
					t.Fatal("new account joined the blocked old refresh")
				}
			}
			unblock()
			select {
			case got := <-done:
				if got.token != nil || !errors.Is(got.err, ErrSessionExpired) {
					t.Fatal("obsolete refresh returned credentials or failed to report session expiry")
				}
			case <-ctx.Done():
				t.Fatal("old refresh did not finish")
			}
			snapshot := client.ExportSession()
			saved, _ := store.Load(ctx)
			for _, candidate := range []*SessionSnapshot{snapshot, saved} {
				if candidate == nil {
					continue
				}
				for _, cookie := range candidate.Cookies {
					if cookie.Name == "guid" && cookie.Value == "old" {
						t.Fatal("obsolete refresh restored a credential cookie in memory or storage")
					}
				}
			}
			if transition == "logout" {
				if snapshot.GUID != "" || snapshot.Token != nil || saved != nil {
					t.Fatal("logout credentials were restored")
				}
			} else {
				if snapshot.GUID != "new" || snapshot.Token == nil || snapshot.Token.AccessToken != "new-token" {
					t.Fatal("new account memory was overwritten")
				}
				if saved == nil || saved.GUID != "new" || saved.Token == nil || saved.Token.AccessToken != "new-token" {
					t.Fatal("new account persistence was overwritten")
				}
			}
		})
	}
}

type blockedAuthStore struct {
	*MemorySessionStore
	started chan struct{}
	release chan struct{}
}

func (s *blockedAuthStore) Save(ctx context.Context, snap *SessionSnapshot) error {
	close(s.started)
	select {
	case <-s.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	return s.MemorySessionStore.Save(ctx, snap)
}

func TestLogoutOrdersInFlightTokenSave(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	store := &blockedAuthStore{NewMemorySessionStore(), make(chan struct{}), make(chan struct{})}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(store.release) }) }
	defer release()
	logoutStarted := make(chan struct{})
	transport := authTransportFunc(func(r *http.Request) (*http.Response, error) {
		body := `{}`
		if r.URL.Path == "/1.0/auth/token" {
			body = `{"access_token":"old-token","refresh_token":"old","expires_in":3600}`
		} else if r.URL.Path == "/auth/logout.json" {
			close(logoutStarted)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	client, err := NewClient(WithTransport(transport), WithSessionStore(store), WithRateLimit(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	client.ImportSession(&SessionSnapshot{GUID: "old"})
	refreshDone := make(chan error, 1)
	go func() { _, err := client.Auth.EnsureToken(ctx, false); refreshDone <- err }()
	select {
	case <-store.started:
	case <-ctx.Done():
		t.Fatal("token save did not start")
	}
	logoutDone := make(chan error, 1)
	go func() { logoutDone <- client.Auth.Logout(ctx) }()
	select {
	case <-logoutStarted:
	case <-ctx.Done():
		t.Fatal("logout did not start")
	}
	release()
	select {
	case err := <-refreshDone:
		if err != nil && !errors.Is(err, ErrSessionExpired) {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("refresh did not finish")
	}
	select {
	case err := <-logoutDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("logout did not finish")
	}
	saved, err := store.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if saved != nil || client.ExportSession().Token != nil || client.ExportSession().GUID != "" {
		t.Fatal("in-flight token persistence survived completed logout")
	}
}
