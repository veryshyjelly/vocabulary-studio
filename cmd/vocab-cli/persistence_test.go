package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
)

type cliTransportFunc func(*http.Request) (*http.Response, error)

func (f cliTransportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type cliSaveStore struct {
	*vocab.MemorySessionStore
	fail bool
}

func (s *cliSaveStore) Save(ctx context.Context, snap *vocab.SessionSnapshot) error {
	if s.fail {
		return errors.New("private backend detail: credential-secret")
	}
	return s.MemorySessionStore.Save(ctx, snap)
}

func cliAccountOptions(store vocab.SessionStore, tokenOutage bool) []vocab.Option {
	transport := cliTransportFunc(func(r *http.Request) (*http.Response, error) {
		status, body := 200, `{}`
		switch r.URL.Path {
		case "/auth/me.json":
			body = `{"validUser":true,"guid":"credential-secret","points":42,"level":{"id":"L01","name":"Novice"},"auth":{"loggedin":true,"uid":"user1","nickname":"Learner"}}`
		case "/1.0/auth/token":
			body = `{"access_token":"bearer-secret","refresh_token":"credential-secret","expires_in":3600}`
			if tokenOutage {
				status, body = 503, `{"error":"unavailable"}`
			}
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	return []vocab.Option{vocab.WithSessionStore(store), vocab.WithTransport(transport), vocab.WithRateLimit(0, 0)}
}

func TestLoginPersistenceReporting(t *testing.T) {
	for _, tc := range []struct {
		name        string
		failSave    bool
		tokenOutage bool
	}{
		{name: "saved"},
		{name: "save failure", failSave: true},
		{name: "token outage preserves web login", tokenOutage: true},
	} {
		for _, jsonOutput := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "/text", true: "/json"}[jsonOutput], func(t *testing.T) {
				store := &cliSaveStore{MemorySessionStore: vocab.NewMemorySessionStore(), fail: tc.failSave}
				args := []string{"--session", "unused-session-path"}
				if jsonOutput {
					args = append(args, "--json")
				}
				args = append(args, "login", "user", "password-secret")
				var out, errOut bytes.Buffer
				code := run(context.Background(), args, strings.NewReader(""), &out, &errOut, cliAccountOptions(store, tc.tokenOutage))
				wantCode := 0
				if tc.failSave {
					wantCode = 1
				}
				if code != wantCode {
					t.Fatalf("exit = %d, want %d", code, wantCode)
				}
				if tc.failSave {
					if !strings.Contains(errOut.String(), "signed in, but the session could not be saved") {
						t.Fatal("missing safe persistence warning")
					}
					if strings.Contains(out.String(), "Session saved") {
						t.Fatal("failed login save claimed persistence")
					}
				} else if errOut.Len() != 0 {
					t.Fatalf("unexpected stderr: %s", errOut.String())
				}
				if jsonOutput {
					var account map[string]any
					if err := json.Unmarshal(out.Bytes(), &account); err != nil {
						t.Fatal("stdout is not valid JSON:", err)
					}
					if _, exists := account["guid"]; exists {
						t.Fatal("account JSON includes refresh credential field")
					}
					if account["points"] != float64(42) || account["validUser"] != true {
						t.Fatal("safe account fields were lost")
					}
				} else if !strings.Contains(out.String(), "Logged in as Learner") {
					t.Fatal("missing successful web-login output")
				}
				for _, secret := range []string{"credential-secret", "bearer-secret", "password-secret", "private backend detail"} {
					if strings.Contains(out.String()+errOut.String(), secret) {
						t.Fatal("output exposed sensitive data")
					}
				}
			})
		}
	}
}

func TestMeJSONOmitsRefreshCredential(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"--json", "me"}, strings.NewReader(""), &out, &errOut, cliAccountOptions(vocab.NewMemorySessionStore(), false))
	if code != 0 || errOut.Len() != 0 {
		t.Fatalf("me failed: exit=%d stderr=%s", code, errOut.String())
	}
	var account map[string]any
	if err := json.Unmarshal(out.Bytes(), &account); err != nil {
		t.Fatal(err)
	}
	if _, exists := account["guid"]; exists {
		t.Fatal("me JSON includes refresh credential field")
	}
	if account["points"] != float64(42) || strings.Contains(out.String(), "credential-secret") {
		t.Fatal("unexpected account JSON")
	}
}
