package vocab

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPTransportErrorsRedirectsAndSizeLimit(t *testing.T) {
	var untrustedAuthHeader string
	var untrustedCookieHeader string

	untrustedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		untrustedAuthHeader = r.Header.Get("Authorization")
		untrustedCookieHeader = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":0}`))
	}))
	defer untrustedSrv.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/rate-limited", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "45")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"too_many_requests","message":"slow down"}`))
	})
	mux.HandleFunc("/html-challenge", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><body>Cloudflare Challenge</body></html>`))
	})
	mux.HandleFunc("/giant", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytes.Repeat([]byte("A"), 2048))
	})
	mux.HandleFunc("/redirect-external", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, untrustedSrv.URL+"/target", http.StatusFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(
		WithBaseURLs(srv.URL, srv.URL, srv.URL, srv.URL),
		WithMaxResponseBytes(1024),
		WithRateLimit(0, 0),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// 1. 429 Rate Limit + Retry-After parsing
	var dummy map[string]any
	err = client.doWebForm(ctx, "/rate-limited", nil, &dummy)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.RetryAfter != 45*time.Second {
		t.Errorf("expected RetryAfter=45s on APIError, got %+v", apiErr)
	}

	// 2. HTML returned where JSON was expected
	err = client.doWebForm(ctx, "/html-challenge", nil, &dummy)
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("expected ErrUnexpectedResponse on HTML challenge page, got %v", err)
	}

	// 3. Max response size limit exceeded
	_, _, err = client.doWebGet(ctx, "/giant", nil, "text/plain")
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("expected ErrResponseTooLarge, got %v", err)
	}

	// 4. Redirect to untrusted host strips Authorization header
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/redirect-external", nil)
	req.Header.Set("Authorization", "Bearer top_secret_token")
	req.Header.Set("Cookie", "SESSION=top_secret_cookie")
	resp, err := client.do(req)
	if err != nil {
		t.Fatalf("redirect request failed: %v", err)
	}
	resp.Body.Close()
	if untrustedAuthHeader != "" || untrustedCookieHeader != "" {
		t.Errorf("leaked headers to untrusted redirect host: Auth=%q Cookie=%q", untrustedAuthHeader, untrustedCookieHeader)
	}
}
