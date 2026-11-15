package desktop

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
)

type storeSwitchTransport func(*http.Request) (*http.Response, error)

func (f storeSwitchTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newStoreSwitchHub(t *testing.T, mode SessionStoreMode) *Hub {
	t.Helper()
	dir := t.TempDir()
	prefs := DefaultPreferences()
	prefs.SessionStoreMode = mode
	data, _ := json.Marshal(prefs)
	if err := os.WriteFile(filepath.Join(dir, "preferences.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	slide := base64.StdEncoding.EncodeToString([]byte(`<div class="challenge-slide"><div class="question typeD"><div class="choices"><a data-nonce="wrong">wrong</a><a data-nonce="right">right</a></div></div></div>`))
	transport := storeSwitchTransport(func(r *http.Request) (*http.Response, error) {
		var payload any
		switch r.URL.Path {
		case "/challenge/start.json":
			payload = vocab.ChallengeStateResponse{Secret: "turn", Question: &vocab.RawQuestion{Type: vocab.QuestionTypeDefinition, Code: slide}}
		case "/challenge/saveanswer.json":
			payload = vocab.SaveAnswerResponse{Secret: "saved", Answer: &vocab.AnswerOutcome{CorrectChoice: 1, Word: "callow", Points: 100}}
		case "/1.0/auth/token":
			payload = vocab.TokenInfo{AccessToken: "refreshed", RefreshToken: "account", ExpiresIn: 3600}
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
		body, _ := json.Marshal(payload)
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})
	hub, err := NewHub(HubOptions{ConfigDir: dir, ClientOpts: []vocab.Option{vocab.WithTransport(transport), vocab.WithRateLimit(0, 0), vocab.WithPacingGuard(false)}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(hub.Shutdown)
	hub.Client().ImportSession(&vocab.SessionSnapshot{GUID: "account"})
	if err := hub.Client().FlushSession(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := hub.Trainer.StartSession(context.Background(), 0, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := hub.Trainer.SubmitChoice(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	return hub
}

func TestStoreSwitchPreservesPracticeState(t *testing.T) {
	for _, target := range []SessionStoreMode{SessionStoreFile, SessionStoreMemory} {
		t.Run(string(target), func(t *testing.T) {
			hub := newStoreSwitchHub(t, SessionStoreFile)
			before := hub.Trainer.GetSessionState()
			if err := hub.SwitchSessionStoreMode(context.Background(), target); err != nil {
				t.Fatal(err)
			}
			if after := hub.Trainer.GetSessionState(); !reflect.DeepEqual(before, after) {
				t.Fatal("switch changed active question, attempts or feedback")
			}
			after, err := hub.Trainer.SubmitChoice(context.Background(), 1)
			if err != nil || after.LastFeedback == nil || !after.LastFeedback.Correct || !after.LastFeedback.EvaluatedLocally {
				t.Fatal("practice did not continue with a local retry")
			}
		})
	}
}

func TestStoreSwitchFailedDestinationPreservesOwner(t *testing.T) {
	hub := newStoreSwitchHub(t, SessionStoreMemory)
	owner := hub.Client()
	before := hub.Trainer.GetSessionState()
	if err := os.Mkdir(filepath.Join(hub.configDir, "session.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := hub.SwitchSessionStoreMode(context.Background(), SessionStoreFile); err == nil {
		t.Fatal("expected destination save failure")
	}
	if hub.Client() != owner || hub.Settings.GetPreferences().SessionStoreMode != SessionStoreMemory {
		t.Fatal("failed switch changed owner or mode")
	}
	if !reflect.DeepEqual(before, hub.Trainer.GetSessionState()) {
		t.Fatal("failed switch changed practice state")
	}
}

func TestActiveTrainerUsesSelectedStoreAfterSwitch(t *testing.T) {
	hub := newStoreSwitchHub(t, SessionStoreFile)
	owner := hub.Client()
	session := hub.session
	if err := hub.SwitchSessionStoreMode(context.Background(), SessionStoreMemory); err != nil {
		t.Fatal(err)
	}
	owner.Auth.InvalidateToken()
	if _, err := owner.Auth.EnsureToken(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(hub.configDir, "session.json")); !os.IsNotExist(err) {
		t.Fatal("refresh through existing practice owner recreated old session file")
	}
	if hub.Client() != owner || hub.session != session {
		t.Fatal("storage switch replaced active client or trainer")
	}
	after, err := hub.Trainer.SubmitChoice(context.Background(), 1)
	if err != nil || after.LastFeedback == nil || !after.LastFeedback.EvaluatedLocally {
		t.Fatal("existing practice did not continue")
	}
}
