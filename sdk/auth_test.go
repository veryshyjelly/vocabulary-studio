package vocab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAuthRollbackSingleflightAnd401Retry(t *testing.T) {
	var tokenHits atomic.Int32
	var apiHits atomic.Int32
	tokenBarrier := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/login/", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch r.FormValue("username") {
		case "good":
			http.SetCookie(w, &http.Cookie{Name: "SESSION", Value: "good_sess", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "guid", Value: "good_guid", Path: "/"})
		case "throttled":
			_, _ = w.Write([]byte(`{"reason":"maxattempts"}`))
		default:
			http.SetCookie(w, &http.Cookie{Name: "SESSION", Value: "bad_sess", Path: "/"})
		}
	})
	mux.HandleFunc("/auth/me.json", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("SESSION")
		if err == nil && c.Value == "good_sess" {
			_ = json.NewEncoder(w).Encode(LoginData{
				ValidUser: true,
				GUID:      "good_guid",
				Auth:      AuthDetails{LoggedIn: true, UID: "u1", Nickname: "GoodUser"},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(LoginData{ValidUser: false})
	})
	mux.HandleFunc("/1.0/auth/token", func(w http.ResponseWriter, r *http.Request) {
		hit := tokenHits.Add(1)
		if hit == 2 {
			// Wait briefly on the 2nd token call to test concurrent singleflight with one canceled caller
			<-tokenBarrier
		}
		_ = json.NewEncoder(w).Encode(TokenInfo{
			TokenType:    "bearer",
			AccessToken:  fmt.Sprintf("jwt_tok_%d", hit),
			RefreshToken: "good_guid",
			ExpiresIn:    3600,
		})
	})
	mux.HandleFunc("/1.0/me/effort", func(w http.ResponseWriter, r *http.Request) {
		n := apiHits.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid_token"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(EffortStats{Points: 900})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(
		WithBaseURLs(srv.URL, srv.URL+"/1.0", srv.URL, srv.URL),
		WithRateLimit(0, 0),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	if _, err := client.Auth.Login(ctx, "good", "pw"); err != nil {
		t.Fatalf("Login(good): %v", err)
	}

	// Attempt failed login with wrong credentials — must return ErrInvalidCredentials AND preserve "good_sess"
	if _, err := client.Auth.Login(ctx, "bad", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	meAfterFail, err := client.Auth.Me(ctx)
	if err != nil || !meAfterFail.ValidUser || meAfterFail.Auth.UID != "u1" {
		t.Fatalf("expected previous session to be preserved after failed Login, got %+v, err=%v", meAfterFail, err)
	}

	// Test concurrent EnsureToken with 1 canceled caller and 5 valid callers
	client.Auth.InvalidateToken()
	var wg sync.WaitGroup
	errs := make([]error, 6)
	tokens := make([]*TokenInfo, 6)

	cancelCtx, cancel := context.WithCancel(ctx)
	wg.Add(6)
	go func() {
		defer wg.Done()
		tokens[0], errs[0] = client.Auth.EnsureToken(cancelCtx, false)
	}()
	for i := 1; i < 6; i++ {
		idx := i
		go func() {
			defer wg.Done()
			tokens[idx], errs[idx] = client.Auth.EnsureToken(ctx, false)
		}()
	}
	time.Sleep(15 * time.Millisecond)
	cancel() // Cancel caller 0 while singleflight is in-flight!
	close(tokenBarrier)
	wg.Wait()

	if !errors.Is(errs[0], context.Canceled) {
		t.Errorf("expected caller 0 to receive context.Canceled, got %v", errs[0])
	}
	for i := 1; i < 6; i++ {
		if errs[i] != nil || tokens[i] == nil || tokens[i].AccessToken != "jwt_tok_2" {
			t.Errorf("caller %d failed: tok=%+v err=%v", i, tokens[i], errs[i])
		}
	}
	if tokenHits.Load() != 2 {
		t.Errorf("expected exactly 2 total token hits, got %d", tokenHits.Load())
	}

	// Test 401 auto-refresh in doAPI
	effort, err := client.Progress.GetEffort(ctx, "", "")
	if err != nil || effort.Points != 900 {
		t.Fatalf("expected doAPI to auto-refresh token on 401 and succeed, got %+v, err=%v", effort, err)
	}
}
