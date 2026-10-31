package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
)

func TestCLICommandsAndInteractivePlay(t *testing.T) {
	slideHTML := `<div class="challenge-slide" data-audio="C/1XB5QC9VRN0HU">
		<div class="question typeD" data-template="multiple-choice">
			<div class="mode"><div class="progress"></div><div class="pointValue">100</div></div>
			<div class="instructions">In this sentence, <strong>callow</strong> means:</div>
			<div class="choices">
				<a data-nonce="n0" accesskey="1A">wealthy</a>
				<a data-nonce="n1" accesskey="2B">young and inexperienced</a>
				<a data-nonce="n2" accesskey="3C">secretive</a>
				<a data-nonce="n3" accesskey="4D">exhausted</a>
			</div>
		</div>
	</div>`
	encodedSlide := base64.StdEncoding.EncodeToString([]byte(slideHTML))

	spellHTML := `<div class="challenge-slide" data-audio="C/1XB5QC9VRN0HU">
		<div class="question typeT">
			<div class="mode"><div class="mastery"></div><div class="pointValue">150</div></div>
			<div class="questionContent"><div class="sentence">A <span class="blanked">______</span> youth.</div></div>
			<div class="spelltheword"><input type="text" class="wordspelling" /></div>
		</div>
	</div>`
	encodedSpell := base64.StdEncoding.EncodeToString([]byte(spellHTML))

	mux := http.NewServeMux()
	mux.HandleFunc("/login/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "SESSION", Value: "cli_sess", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "guid", Value: "cli_guid", Path: "/"})
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/auth/me.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.LoginData{
			ValidUser: true,
			GUID:      "cli_guid",
			Points:    2500,
			Level:     vocab.UserLevel{ID: "L02", Name: "Hotshot"},
			Auth:      vocab.AuthDetails{LoggedIn: true, UID: "u_cli", Nickname: "CLIUser"},
			Perms:     vocab.UserPerms{FullVocabTrainer: true},
		})
	})
	mux.HandleFunc("/auth/logout.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/1.0/auth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.TokenInfo{
			TokenType:   "bearer",
			AccessToken: "tok_cli",
			ExpiresIn:   3600,
		})
	})
	mux.HandleFunc("/dictionary/callow", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<div class="wordPage" data-word="callow" data-learnable="true">
			<div class="word-area"><h1>callow</h1><p class="short">Young and inexperienced.</p><p class="long">Long explanation.</p>
			<div class="ipa-section"><div class="ipa-with-audio"><span class="us-flag-icon"></span><span class="span-replace-h3">/ˈkæloʊ/</span><a class="audio" data-audio="C/1XB5QC9VRN0HU"></a></div></div></div>
			<div class="word-definitions"><ol><li class="sense pos_a"><div class="definition"><div class="pos-icon">adjective</div>young and inexperienced</div></li></ol></div>
		</div>`))
	})
	mux.HandleFunc("/corpus/1.0/examples/random.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.CorpusExamplesResponse{
			TotalHits: 1,
			Sentences: []vocab.CorpusSentence{{Sentence: "A callow youth.", Volume: &vocab.CorpusVolume{Title: "Book"}}},
		})
	})
	mux.HandleFunc("/word-of-the-day/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<div>
			<div class="box-yesterday-today-tomorrow">October 31, 2026</div>
			<div class="box-lks-audio-wod"><a class="word-of-the-day" href="/dictionary/enchantment">enchantment</a></div>
			<p class="txt-wod-usage">Short usage.</p><p class="txt-wod-desc">Long desc.</p>
		</div>`))
	})
	mux.HandleFunc("/lists/load.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": 0,
			"result": vocab.VocabularyList{
				WordlistID:  123,
				Name:        "Test List",
				Description: "Desc",
				Words:       []vocab.ListWord{{Word: "callow", Definition: "young"}},
			},
		})
	})
	mux.HandleFunc("/lists/smart-search.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.SmartSearchResponse{
			TotalResults: 1,
			Results:      []vocab.SmartSearchItem{{WordlistID: 123, Name: "Test List", WordCount: 1}},
		})
	})
	mux.HandleFunc("/challenge/start.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.ChallengeStateResponse{
			Version:  4,
			Action:   "newround",
			Secret:   "s1",
			Question: &vocab.RawQuestion{Turn: 1, Type: vocab.QuestionTypeDefinition, Code: encodedSlide},
		})
	})
	mux.HandleFunc("/challenge/hint.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.HintResponse{
			Secret: "s1_h",
			Nonces: []string{"n0", "n3"},
		})
	})
	mux.HandleFunc("/challenge/saveanswer.json", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		a := r.FormValue("a")
		_ = json.NewEncoder(w).Encode(vocab.SaveAnswerResponse{
			Secret: "s2",
			Answer: &vocab.AnswerOutcome{
				Correct:         a == "n1" || a == "callow",
				CorrectChoice:   1,
				AcceptedAnswers: []string{"callow"},
				Word:            "callow",
				Points:          100,
				Blurb:           &vocab.WordBlurb{Short: "Short blurb"},
			},
		})
	})
	mux.HandleFunc("/challenge/nextquestion.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.ChallengeStateResponse{
			Version:  4,
			Action:   "next",
			Secret:   "s3",
			Question: &vocab.RawQuestion{Turn: 2, Type: vocab.QuestionTypeSpelling, Code: encodedSpell},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	sessionFile := filepath.Join(t.TempDir(), "session.json")
	opts := []vocab.Option{
		vocab.WithBaseURLs(srv.URL, srv.URL+"/1.0", srv.URL+"/corpus/1.0", srv.URL+"/audio/1.0/us"),
		vocab.WithPacingGuard(false),
		vocab.WithRateLimit(0, 0),
	}

	runCmd := func(input string, args ...string) (int, string, string) {
		var stdout, stderr bytes.Buffer
		fullArgs := append([]string{"--session", sessionFile}, args...)
		code := run(context.Background(), fullArgs, strings.NewReader(input), &stdout, &stderr, opts)
		return code, stdout.String(), stderr.String()
	}

	// 1. login & me
	if code, out, errOut := runCmd("", "login", "cli@example.com", "pw123"); code != 0 || !strings.Contains(out, "Logged in as CLIUser") {
		t.Fatalf("login failed: code=%d out=%q err=%q", code, out, errOut)
	}
	if code, out, errOut := runCmd("", "me"); code != 0 || !strings.Contains(out, "CLIUser") {
		t.Fatalf("me failed: code=%d out=%q err=%q", code, out, errOut)
	}

	// 2. lookup (text & json)
	if code, out, _ := runCmd("", "lookup", "callow"); code != 0 || !strings.Contains(out, "young and inexperienced") {
		t.Fatalf("lookup text failed: code=%d out=%q", code, out)
	}
	if code, out, _ := runCmd("", "--json", "lookup", "callow"); code != 0 || !strings.Contains(out, `"word": "callow"`) {
		t.Fatalf("lookup json failed: code=%d out=%q", code, out)
	}

	// 3. examples, wod, list, search-lists
	if code, out, _ := runCmd("", "examples", "callow"); code != 0 || !strings.Contains(out, "A callow youth.") {
		t.Fatalf("examples failed: code=%d out=%q", code, out)
	}
	if code, out, _ := runCmd("", "wod"); code != 0 || !strings.Contains(out, "enchantment") {
		t.Fatalf("wod failed: code=%d out=%q", code, out)
	}
	if code, out, _ := runCmd("", "list", "123"); code != 0 || !strings.Contains(out, "Test List") {
		t.Fatalf("list failed: code=%d out=%q", code, out)
	}
	if code, out, _ := runCmd("", "search-lists", "test"); code != 0 || !strings.Contains(out, "Test List") {
		t.Fatalf("search-lists failed: code=%d out=%q", code, out)
	}

	// 4. interactive play: Q1 (hint 'h' -> wrong choice '1' -> right choice '2') -> Q2 spelling (wrong 'calow' -> right 'callow') -> Q3 quit 'q'
	playInput := "h\n1\n2\ncalow\ncallow\nq\n"
	if code, out, errOut := runCmd(playInput, "play", "123"); code != 0 {
		t.Fatalf("play failed: code=%d out=%q err=%q", code, out, errOut)
	} else if !strings.Contains(out, "Eliminated nonces") || !strings.Contains(out, "Incorrect — try spelling it again!") {
		t.Errorf("unexpected play output: %s", out)
	}

	// 5. logout, help, and error exit codes
	if code, out, _ := runCmd("", "logout"); code != 0 || !strings.Contains(out, "Logged out") {
		t.Fatalf("logout failed: code=%d out=%q", code, out)
	}
	if code, out, _ := runCmd("", "help"); code != 0 || !strings.Contains(out, "Usage: vocab-cli") {
		t.Errorf("help failed: code=%d out=%s", code, out)
	}
	if code, _, _ := runCmd("", "unknown-cmd"); code != 2 {
		t.Errorf("expected exit code 2 for unknown command, got %d", code)
	}
}
