package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

func main() {
	_ = vocab.LoadDotEnvFile(".env.local")
	_ = vocab.LoadDotEnvFile(".env")

	var outDir string
	var sessionFile string
	flag.StringVar(&outDir, "out", "sdk/testdata/cassettes", "Output directory for scrubbed go-vcr cassettes")
	flag.StringVar(&sessionFile, "session", ".scratch/session.json", "Path to cached session file")
	flag.Parse()

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatalf("mkdir %s: %v", outDir, err)
	}

	cassettePath := filepath.Join(outDir, "live_smoke")
	rec, err := recorder.New(
		cassettePath,
		recorder.WithMode(recorder.ModeRecordOnce),
		recorder.WithSkipRequestLatency(true),
		recorder.WithHook(scrubSensitiveDataHook, recorder.BeforeSaveHook),
	)
	if err != nil {
		fatalf("init go-vcr recorder: %v", err)
	}
	defer func() {
		if err := rec.Stop(); err != nil {
			fatalf("stop recorder: %v", err)
		}
		fmt.Printf("Saved scrubbed cassette to %s.yaml\n", cassettePath)
	}()

	client, err := vocab.NewClient(
		vocab.WithSessionStore(vocab.NewFileSessionStore(sessionFile)),
		vocab.WithTransport(rec),
	)
	if err != nil {
		fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 1. Optional login if credentials are provided and session is not already valid
	me, err := client.Auth.Me(ctx)
	if err != nil {
		fatalf("auth/me.json failed: %v", err)
	}
	if !me.ValidUser {
		username := os.Getenv("VOCAB_USERNAME")
		password := os.Getenv("VOCAB_PASSWORD")
		if username != "" && password != "" {
			fmt.Println("Logging in with VOCAB_USERNAME from environment...")
			me, err = client.Auth.Login(ctx, username, password)
			if err != nil {
				fatalf("login failed: %v", err)
			}
			fmt.Printf("Authenticated as %s (FullVocabTrainer=%v)\n", me.Auth.Nickname, me.Perms.FullVocabTrainer)
		} else {
			fmt.Println("No VOCAB_USERNAME set in .env.local; recording public + anonymous Bearer endpoints...")
		}
	} else {
		fmt.Printf("Reusing existing authenticated session from %s (user=%s)\n", sessionFile, me.Auth.Nickname)
	}

	// 2. Dictionary Lookup & Autocomplete
	entry, err := client.Dictionary.Lookup(ctx, "callow")
	if err != nil {
		fatalf("Lookup(callow): %v", err)
	}
	fmt.Printf("Recorded Dictionary Lookup: %s (%d senses, %d family words)\n",
		entry.Word, len(entry.Senses), len(entry.WordFamily))

	ac, err := client.Dictionary.Autocomplete(ctx, "callo", 5)
	if err != nil {
		fatalf("Autocomplete(callo): %v", err)
	}
	fmt.Printf("Recorded Autocomplete: %d suggestions\n", len(ac))

	// 3. Corpus Examples
	ex, err := client.Dictionary.Examples(ctx, "callow", vocab.CorpusDomainAll, 0, 5)
	if err != nil {
		fatalf("Examples(callow): %v", err)
	}
	fmt.Printf("Recorded Corpus Examples: %d hits\n", ex.TotalHits)

	// 4. Word of the Day & Archive
	wod, err := client.Dictionary.WordOfTheDay(ctx, "")
	if err != nil {
		fatalf("WordOfTheDay: %v", err)
	}
	fmt.Printf("Recorded Word of the Day: %s (%s)\n", wod.Word, wod.Date)

	// 5. Lists & Units (Public + Bearer JWT)
	searchRes, err := client.Lists.Search(ctx, "gre", 0, 5, false)
	if err != nil {
		fatalf("Lists.Search: %v", err)
	}
	fmt.Printf("Recorded Lists.Search: %d total results\n", searchRes.TotalResults)
	if len(searchRes.Results) > 0 {
		firstID := searchRes.Results[0].WordlistID
		wl, err := client.Lists.Get(ctx, firstID)
		if err != nil {
			fatalf("Lists.Get(%d): %v", firstID, err)
		}
		fmt.Printf("Recorded Lists.Get(%d): %s (%d words)\n", wl.WordlistID, wl.Name, len(wl.Words))
	}

	wp, err := client.Progress.GetWordProgress(ctx, "callow", true)
	if err != nil {
		fatalf("Progress.GetWordProgress: %v", err)
	}
	fmt.Printf("Recorded Progress.GetWordProgress(callow): learnable=%v\n", wp.Learnable)
}

func scrubSensitiveDataHook(i *cassette.Interaction) error {
	sensitiveReqHeaders := []string{"Cookie", "Authorization"}
	for _, h := range sensitiveReqHeaders {
		if len(i.Request.Headers[h]) > 0 {
			i.Request.Headers[h] = []string{"REDACTED"}
		}
	}
	if len(i.Response.Headers["Set-Cookie"]) > 0 {
		i.Response.Headers["Set-Cookie"] = []string{"REDACTED"}
	}
	if i.Request.Form != nil {
		for _, field := range []string{"password", "username", "refresh_token"} {
			if _, ok := i.Request.Form[field]; ok {
				i.Request.Form[field] = []string{"REDACTED"}
			}
		}
	}
	if i.Request.Method == http.MethodPost && filepath.Base(i.Request.URL) == "login" {
		i.Request.Body = "REDACTED"
	}
	if i.Request.Method == http.MethodPost && filepath.Base(i.Request.URL) == "token" && i.Request.Body != "" {
		i.Request.Body = "REDACTED"
	}

	// Scrub sensitive tokens/identifiers inside JSON response bodies (/1.0/auth/token, /auth/me.json).
	trimmedBody := strings.TrimSpace(i.Response.Body)
	if strings.HasPrefix(trimmedBody, "{") && strings.HasSuffix(trimmedBody, "}") {
		var m map[string]any
		if json.Unmarshal([]byte(trimmedBody), &m) == nil {
			modified := false
			if _, ok := m["access_token"]; ok {
				m["access_token"] = "REDACTED_ACCESS_TOKEN"
				modified = true
			}
			if _, ok := m["refresh_token"]; ok {
				m["refresh_token"] = "REDACTED_REFRESH_TOKEN"
				modified = true
			}
			if _, ok := m["guid"]; ok {
				m["guid"] = "REDACTED_GUID"
				modified = true
			}
			if _, ok := m["visitid"]; ok {
				m["visitid"] = "REDACTED_VISITID"
				modified = true
			}
			if authMap, ok := m["auth"].(map[string]any); ok {
				if _, hasEmail := authMap["email"]; hasEmail {
					authMap["email"] = "REDACTED_EMAIL"
					modified = true
				}
			}
			if modified {
				if scrubbedBytes, err := json.Marshal(m); err == nil {
					i.Response.Body = string(scrubbedBytes)
				}
			}
		}
	}
	return nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
