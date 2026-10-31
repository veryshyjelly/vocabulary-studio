package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
)

func TestScrubSensitiveDataHook(t *testing.T) {
	interaction := &cassette.Interaction{
		Request: cassette.Request{
			Method: http.MethodPost,
			URL:    "https://api.vocabulary.com/1.0/auth/token",
			Body:   "refresh_token=secret_guid_999",
			Headers: http.Header{
				"Cookie":        {"SESSION=secret_sess; guid=secret_guid_999"},
				"Authorization": {"Bearer secret_jwt"},
			},
			Form: url.Values{
				"username":      {"user@example.com"},
				"password":      {"top_secret_pw"},
				"refresh_token": {"secret_guid_999"},
			},
		},
		Response: cassette.Response{
			Headers: http.Header{
				"Set-Cookie": {"SESSION=secret_sess; Path=/; HttpOnly"},
			},
			Body: `{"token_type":"bearer","access_token":"eyJhbGciOiJIUzI1NiJ9.real.sig","refresh_token":"secret_guid_999","guid":"secret_guid_999","visitid":"v_123","auth":{"email":"user@example.com"}}`,
		},
	}

	if err := scrubSensitiveDataHook(interaction); err != nil {
		t.Fatalf("scrubSensitiveDataHook failed: %v", err)
	}

	if interaction.Request.Headers.Get("Cookie") != "REDACTED" {
		t.Errorf("Cookie header not redacted: %v", interaction.Request.Headers["Cookie"])
	}
	if interaction.Request.Headers.Get("Authorization") != "REDACTED" {
		t.Errorf("Authorization header not redacted: %v", interaction.Request.Headers["Authorization"])
	}
	if interaction.Response.Headers.Get("Set-Cookie") != "REDACTED" {
		t.Errorf("Set-Cookie header not redacted: %v", interaction.Response.Headers["Set-Cookie"])
	}
	if interaction.Request.Body != "REDACTED" {
		t.Errorf("Request.Body not redacted: %q", interaction.Request.Body)
	}
	for _, secret := range []string{"secret_guid_999", "top_secret_pw", "user@example.com", "eyJhbGciOiJIUzI1NiJ9.real.sig", "v_123"} {
		if strings.Contains(interaction.Response.Body, secret) {
			t.Errorf("secret %q leaked in scrubbed Response.Body: %s", secret, interaction.Response.Body)
		}
		for _, vals := range interaction.Request.Form {
			for _, v := range vals {
				if strings.Contains(v, secret) {
					t.Errorf("secret %q leaked in scrubbed Request.Form: %v", secret, interaction.Request.Form)
				}
			}
		}
	}
}
