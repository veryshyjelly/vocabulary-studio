package vocab

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type failingSwitchStore struct {
	*MemorySessionStore
	err error
}

func (s failingSwitchStore) Save(context.Context, *SessionSnapshot) error { return s.err }

func TestSetSessionStoreFailurePreservesSelection(t *testing.T) {
	ctx := context.Background()
	old := NewMemorySessionStore()
	client, err := NewClient(WithSessionStore(old))
	if err != nil {
		t.Fatal(err)
	}
	client.ImportSession(&SessionSnapshot{GUID: "account"})
	want := errors.New("save failed")
	if err := client.SetSessionStore(ctx, failingSwitchStore{NewMemorySessionStore(), want}); !errors.Is(err, want) {
		t.Fatal("destination save failure was not returned")
	}
	if err := client.SetSessionStore(ctx, nil); err == nil {
		t.Fatal("nil destination was accepted")
	}
	if err := client.FlushSession(ctx); err != nil {
		t.Fatal(err)
	}
	saved, err := old.Load(ctx)
	if err != nil || saved == nil || saved.GUID != "account" {
		t.Fatal("failed switch replaced the original store")
	}
}

func TestSetSessionStoreRoutesPendingRefreshToDestination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	transport := authTransportFunc(func(r *http.Request) (*http.Response, error) {
		close(started)
		select {
		case <-release:
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"access_token":"fresh","refresh_token":"account","expires_in":3600}`)), Request: r}, nil
	})
	old, destination := NewMemorySessionStore(), NewMemorySessionStore()
	client, err := NewClient(WithSessionStore(old), WithTransport(transport), WithRateLimit(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	client.ImportSession(&SessionSnapshot{GUID: "account"})
	trainer, pacing, generation := client.Trainer, client.pacingGuard, client.authGeneration
	done := make(chan error, 1)
	go func() { _, err := client.Auth.EnsureToken(ctx, false); done <- err }()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("refresh did not start")
	}
	if err := client.SetSessionStore(ctx, destination); err != nil {
		t.Fatal(err)
	}
	if client.Trainer != trainer || client.pacingGuard != pacing || client.authGeneration != generation {
		t.Fatal("store switch changed gameplay ownership or authentication generation")
	}
	unblock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("refresh did not finish")
	}
	saved, err := destination.Load(ctx)
	if err != nil || saved == nil || saved.Token == nil || saved.Token.AccessToken != "fresh" {
		t.Fatal("pending refresh was not persisted to selected destination")
	}
	if stale, err := old.Load(ctx); err != nil || stale != nil {
		t.Fatal("pending refresh persisted to retired backend")
	}
}
