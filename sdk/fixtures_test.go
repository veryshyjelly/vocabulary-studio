package vocab

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

func TestLiveSmokeCassetteReplay(t *testing.T) {
	cassettePath := filepath.Join("testdata", "cassettes", "live_smoke")
	matcher := func(r *http.Request, i cassette.Request) bool {
		return r.Method == i.Method && r.URL.String() == i.URL
	}
	rec, err := recorder.New(
		cassettePath,
		recorder.WithMode(recorder.ModeReplayOnly),
		recorder.WithSkipRequestLatency(true),
		recorder.WithMatcher(matcher),
	)
	if err != nil {
		t.Skipf("skipping cassette replay test (%v)", err)
	}
	defer func() { _ = rec.Stop() }()

	client, err := NewClient(
		WithTransport(rec),
		WithRateLimit(0, 0),
		WithPacingGuard(false),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := client.Dictionary.Lookup(ctx, "callow")
	if err != nil {
		t.Fatalf("Cassette Dictionary.Lookup(callow): %v", err)
	}
	if entry.Word != "callow" || len(entry.Senses) == 0 {
		t.Errorf("unexpected replayed entry: %+v", entry)
	}

	ac, err := client.Dictionary.Autocomplete(ctx, "callo", 5)
	if err != nil || len(ac) == 0 {
		t.Fatalf("Cassette Dictionary.Autocomplete(callo): %v (len=%d)", err, len(ac))
	}

	ex, err := client.Dictionary.Examples(ctx, "callow", CorpusDomainAll, 0, 5)
	if err != nil || len(ex.Sentences) == 0 {
		t.Fatalf("Cassette Dictionary.Examples(callow): %v", err)
	}

	wod, err := client.Dictionary.WordOfTheDay(ctx, "")
	if err != nil || wod.Word != "enchantment" {
		t.Fatalf("Cassette Dictionary.WordOfTheDay: %+v, err=%v", wod, err)
	}

	searchRes, err := client.Lists.Search(ctx, "gre", 0, 5, false)
	if err != nil || len(searchRes.Results) == 0 {
		t.Fatalf("Cassette Lists.Search(gre): %v", err)
	}
	wl, err := client.Lists.Get(ctx, searchRes.Results[0].WordlistID)
	if err != nil || len(wl.Words) == 0 {
		t.Fatalf("Cassette Lists.Get: %v", err)
	}

	wp, err := client.Progress.GetWordProgress(ctx, "callow", true)
	if err != nil || !wp.Learnable {
		t.Fatalf("Cassette Progress.GetWordProgress(callow): %+v, err=%v", wp, err)
	}
}

func TestCassetteFilesContainNoLeakedSecrets(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "cassettes", "*.yaml"))
	if err != nil {
		t.Fatalf("glob cassettes: %v", err)
	}
	jwtRegex := regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)
	cookieLeakRegex := regexp.MustCompile(`(?i)(SESSION|__cf_bm)=[A-Za-z0-9_-]{10,}`)

	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		content := string(raw)
		if m := jwtRegex.FindString(content); m != "" {
			t.Errorf("cassette %s contains unredacted JWT: %s", f, m)
		}
		if m := cookieLeakRegex.FindString(content); m != "" {
			t.Errorf("cassette %s contains unredacted session cookie: %s", f, m)
		}
	}
}
