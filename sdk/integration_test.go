package vocab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFullSDKFlows(t *testing.T) {
	var tokenCalls atomic.Int32

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

	mux := http.NewServeMux()

	mux.HandleFunc("/login/", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.FormValue("username") != "sub@example.com" || r.FormValue("password") != "secret123" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html>login failed</html>`))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "SESSION", Value: "sess_abc", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "guid", Value: "guid_xyz", Path: "/"})
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/auth/me.json", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("SESSION")
		if err != nil || c.Value != "sess_abc" {
			_ = json.NewEncoder(w).Encode(LoginData{ValidUser: false})
			return
		}
		_ = json.NewEncoder(w).Encode(LoginData{
			ValidUser: true,
			GUID:      "guid_xyz",
			Paid:      true,
			Points:    15400,
			Level:     UserLevel{ID: "L04", Name: "Wordsmith", Milestone: 3, Progress: 0.75},
			Auth:      AuthDetails{LoggedIn: true, UID: "u_999", Nickname: "Ayush", Email: "sub@example.com"},
			Perms:     UserPerms{FullVocabTrainer: true, SubscribeEmail: true},
		})
	})

	mux.HandleFunc("/1.0/auth/token", func(w http.ResponseWriter, r *http.Request) {
		tokenCalls.Add(1)
		_ = r.ParseForm()
		rt := r.FormValue("refresh_token")
		if rt == "" {
			rt = "anon_rt"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token_type":    "bearer",
			"access_token":  "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1Xzk5OSIsImV4cCI6MTk5OTk5OTk5OX0.sig",
			"refresh_token": rt,
			"expires_in":    3600,
		})
	})

	mux.HandleFunc("/challenge/start.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ChallengeStateResponse{
			Version: 4,
			Action:  "newround",
			Secret:  "sec_turn_1",
			Round:   &ChallengeRound{Number: 1, PlayedCount: 0, Streak: 0},
			Question: &RawQuestion{
				Turn:  1,
				Type:  QuestionTypeDefinition,
				Code:  encodedSlide,
				Hints: []HintType{HintFiftyFifty, HintDefinition, HintExample},
			},
		})
	})

	mux.HandleFunc("/challenge/hint.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(HintResponse{
			Secret: "sec_turn_1_hinted",
			Nonces: []string{"n0", "n3"},
		})
	})

	mux.HandleFunc("/challenge/saveanswer.json", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		nonce := r.FormValue("a")
		correct := nonce == "n1"
		_ = json.NewEncoder(w).Encode(SaveAnswerResponse{
			Secret: "sec_turn_1_saved",
			Round:  &ChallengeRound{Number: 1, PlayedCount: 1, Streak: 1},
			Answer: &AnswerOutcome{
				Correct:       correct,
				CorrectChoice: 1,
				Word:          "callow",
				Points:        100,
				Blurb:         &WordBlurb{Short: "Callow means young and inexperienced."},
			},
		})
	})

	mux.HandleFunc("/challenge/nextquestion.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ChallengeStateResponse{
			Version: 4,
			Action:  "next",
			Secret:  "sec_turn_2",
			Round:   &ChallengeRound{Number: 1, PlayedCount: 1, Streak: 1},
			Question: &RawQuestion{
				Turn: 2,
				Type: QuestionTypeDefinition,
				Code: encodedSlide,
			},
		})
	})

	mux.HandleFunc("/lists/load.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": 0,
			"result": VocabularyList{
				Version:     2,
				WordlistID:  7813867,
				Name:        "GRE High Frequency",
				Description: "Top GRE words",
				Shared:      true,
				Words: []ListWord{
					{Word: "callow", Lang: "en", Definition: "young and inexperienced", Learnable: true},
				},
			},
		})
	})

	mux.HandleFunc("/lists/save.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"result": 999888})
	})

	mux.HandleFunc("/1.0/units/gfpldfb3", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(UnitsResponse{
			AssetBase: "https://cdn.vocabulary.com/units",
			Unit: &Unit{
				ID:    "gfpldfb3",
				Title: "Literature",
				Type:  "collection",
			},
		})
	})

	mux.HandleFunc("/1.0/progress/words/callow", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(WordProgressDetail{
			Word:      "callow",
			Learnable: true,
			PKV:       82,
		})
	})

	mux.HandleFunc("/1.0/me/effort", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(EffortStats{
			WordsMastered:     12,
			Points:            4500,
			QuestionsAnswered: 50,
			TimeMinutes:       22,
		})
	})

	mux.HandleFunc("/lists/7813867/bee/start.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(BeeStateResponse{
			Action:  "start",
			Secret:  "bee_sec_1",
			Audio:   []string{"C/1XB5QC9VRN0HU"},
			Example: []string{"A ", " youth of seventeen."},
			Score:   &BeeScore{Remaining: 10, Played: 0},
		})
	})

	mux.HandleFunc("/lists/7813867/bee/guess.json", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		g := r.FormValue("guess")
		_ = json.NewEncoder(w).Encode(BeeStateResponse{
			Secret:  "bee_sec_2",
			Guesses: 1,
			Correct: g == "callow",
			Word:    "callow",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	fc := newFakeClock(time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC))
	sessionPath := filepath.Join(t.TempDir(), "session.json")
	client, err := NewClient(
		WithBaseURLs(srv.URL, srv.URL+"/1.0", srv.URL+"/corpus/1.0", srv.URL+"/audio/1.0/us"),
		WithSessionStore(NewFileSessionStore(sessionPath)),
		WithClock(fc),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()

	// 1. Test Login & Session Persistence
	me, err := client.Auth.Login(ctx, "sub@example.com", "secret123")
	if err != nil {
		t.Fatalf("Auth.Login: %v", err)
	}
	if !me.ValidUser || me.Auth.Nickname != "Ayush" || !me.Perms.FullVocabTrainer {
		t.Fatalf("unexpected LoginData: %+v", me)
	}
	if tokenCalls.Load() != 1 {
		t.Errorf("expected 1 token exchange call after login, got %d", tokenCalls.Load())
	}

	// Verify session snapshot reloads cleanly into a second client instance
	client2, err := NewClient(
		WithBaseURLs(srv.URL, srv.URL+"/1.0", srv.URL+"/corpus/1.0", srv.URL+"/audio/1.0/us"),
		WithSessionStore(NewFileSessionStore(sessionPath)),
		WithClock(fc),
	)
	if err != nil {
		t.Fatalf("NewClient 2: %v", err)
	}
	me2, err := client2.Auth.Me(ctx)
	if err != nil || !me2.ValidUser {
		t.Fatalf("restored client Me() failed: %+v, err=%v", me2, err)
	}

	// 2. Test TrainerSession (Start -> Hint -> Wrong Choice -> Local 2nd Try Correct Choice -> Next)
	session, _, err := client.Trainer.NewSession(ctx, StartChallengeRequest{
		ActivityType: ActivityVocabTrainer,
	})
	if err != nil {
		t.Fatalf("Trainer.NewSession: %v", err)
	}
	if session.State() != SessionStateActiveQuestion {
		t.Fatalf("State() = %q, want %q", session.State(), SessionStateActiveQuestion)
	}
	q := session.CurrentQuestion()
	if q == nil || q.TargetWord != "callow" || len(q.Choices) != 4 {
		t.Fatalf("unexpected current question: %+v", q)
	}

	hintResp, err := session.Hint(ctx, HintFiftyFifty)
	if err != nil || len(hintResp.Nonces) != 2 || session.Secret() != "sec_turn_1_hinted" {
		t.Fatalf("session.Hint failed: %+v, err=%v", hintResp, err)
	}
	if !session.IsNonceEliminated("n0") || !session.IsNonceEliminated("n3") {
		t.Errorf("expected n0 and n3 to be marked eliminated")
	}

	// First try: pick wrong choice index 0 -> calls saveanswer.json (EvaluatedLocally=false, Attempt=1)
	outcome1, ok1, err := session.ChooseIndex(ctx, 0)
	if err != nil {
		t.Fatalf("ChooseIndex(0): %v", err)
	}
	if ok1 || outcome1.CorrectChoice != 1 || outcome1.EvaluatedLocally || outcome1.Attempt != 1 {
		t.Fatalf("expected 1st try server-persisted incorrect with CorrectChoice=1, got ok=%v outcome=%+v", ok1, outcome1)
	}
	if session.State() != SessionStateActiveQuestion {
		t.Errorf("State after wrong 1st try = %q, want %q", session.State(), SessionStateActiveQuestion)
	}

	// Using a hint after 1st attempt must fail with ErrInvalidStateTransition
	if _, err := session.Hint(ctx, HintDefinition); !errors.Is(err, ErrInvalidStateTransition) {
		t.Errorf("expected ErrInvalidStateTransition when calling Hint after 1st try, got %v", err)
	}

	// Second try: pick choice index 1 -> evaluated locally (EvaluatedLocally=true, Attempt=2, Points=0)
	outcome2, ok2, err := session.ChooseIndex(ctx, 1)
	if err != nil || !ok2 || outcome2.Word != "callow" {
		t.Fatalf("expected 2nd try local evaluation to succeed, got ok=%v err=%v", ok2, err)
	}
	if !outcome2.EvaluatedLocally || outcome2.Attempt != 2 || outcome2.Points != 0 {
		t.Errorf("expected local evaluation metadata (EvaluatedLocally=true, Attempt=2, Points=0), got %+v", outcome2)
	}
	if session.State() != SessionStateQuestionCompleted {
		t.Errorf("State after correct 2nd try = %q, want %q", session.State(), SessionStateQuestionCompleted)
	}

	// Attempting to answer again after completion must fail with ErrInvalidStateTransition
	if _, _, err := session.ChooseIndex(ctx, 1); !errors.Is(err, ErrInvalidStateTransition) {
		t.Errorf("expected ErrInvalidStateTransition when answering completed question, got %v", err)
	}

	if _, err := session.Next(ctx); err != nil {
		t.Fatalf("session.Next: %v", err)
	}
	if session.Secret() != "sec_turn_2" || session.State() != SessionStateActiveQuestion || session.Tries() != 0 {
		t.Errorf("unexpected state after Next: secret=%q state=%q tries=%d", session.Secret(), session.State(), session.Tries())
	}

	// 3. Test Lists & Units
	wl, err := client.Lists.Get(ctx, 7813867)
	if err != nil || wl.Name != "GRE High Frequency" || len(wl.Words) != 1 {
		t.Fatalf("Lists.Get failed: %+v, err=%v", wl, err)
	}
	newID, err := client.Lists.Create(ctx, CreateListRequest{
		Name: "My Custom List",
		Words: []CreateListWordInput{
			{Word: "callow"},
		},
	})
	if err != nil || newID != 999888 {
		t.Fatalf("Lists.Create failed: id=%d, err=%v", newID, err)
	}
	unitResp, err := client.Lists.GetUnit(ctx, UnitCategoryLiterature)
	if err != nil || unitResp.Unit == nil || unitResp.Unit.Title != "Literature" {
		t.Fatalf("Lists.GetUnit failed: %+v, err=%v", unitResp, err)
	}

	// 4. Test Progress & Bee
	wp, err := client.Progress.GetWordProgress(ctx, "callow", true)
	if err != nil || !wp.Learnable || wp.PKV != 82 {
		t.Fatalf("Progress.GetWordProgress failed: %+v, err=%v", wp, err)
	}
	effort, err := client.Progress.GetEffort(ctx, "2026-10-25", "2026-10-31")
	if err != nil || effort.WordsMastered != 12 {
		t.Fatalf("Progress.GetEffort failed: %+v, err=%v", effort, err)
	}

	beeStart, err := client.Bee.Start(ctx, 7813867, "", "")
	if err != nil || beeStart.FormattedExample("___") != "A ___ youth of seventeen." {
		t.Fatalf("Bee.Start failed: %+v, err=%v", beeStart, err)
	}
	beeGuess, err := client.Bee.Guess(ctx, 7813867, beeStart.Secret, "Callow", 1400, "")
	if err != nil || !beeGuess.Correct {
		t.Fatalf("Bee.Guess failed: %+v, err=%v", beeGuess, err)
	}
}

func TestComprehensiveEndpointsPaginationAndValidation(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/login/vocabtrainer", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "SESSION", Value: "vt_sess", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "guid", Value: "vt_guid", Path: "/"})
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/auth/me.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(LoginData{
			ValidUser: true,
			GUID:      "vt_guid",
			Auth:      AuthDetails{LoggedIn: true, UID: "u_vt", Nickname: "VTUser"},
		})
	})
	mux.HandleFunc("/auth/logout.json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/1.0/auth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(TokenInfo{
			TokenType:    "bearer",
			AccessToken:  "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1X3Z0In0.sig",
			RefreshToken: "vt_guid",
			ExpiresIn:    3600,
		})
	})

	// Lists endpoints
	mux.HandleFunc("/1.0/lists/101", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(FlashcardListResponse{
			DictionaryVideoHost: "https://sd-pronunciation-processed-videos.sdcdns.com",
		})
	})
	mux.HandleFunc("/lists/byprofile.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"wordlists": []ProfileListSummary{{WordlistID: 101, Name: "My List", WordCount: 10, Owner: true}},
			},
		})
	})
	mux.HandleFunc("/1.0/units", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Total-Count", "42")
		w.Header().Set("Link", `<`+r.Host+`/1.0/units?skip=10>; rel="next"`)
		_ = json.NewEncoder(w).Encode(UnitsResponse{
			Units: []Unit{{ID: "u1", Title: "Unit 1", Type: "work"}},
		})
	})
	mux.HandleFunc("/lists/vocabgrabber/senses.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]LearnableSense{{ID: 101016, Learnable: true, POS: "adjective", Def: "young"}})
	})
	mux.HandleFunc("/lists/validate.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": 0,
			"result": map[string]any{
				"valid": true,
				"words": []ValidatedWord{{Word: "callow", Found: true, Learnable: true}},
			},
		})
	})
	mux.HandleFunc("/lists/vocabgrabber/grab.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(VocabGrabberResponse{
			Format: "text",
			Words:  []GrabbedWord{{Word: "callow", Def: "young"}},
		})
	})
	mux.HandleFunc("/lists/save.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"result": 101})
	})
	mux.HandleFunc("/lists/delete.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":0}`))
	})
	mux.HandleFunc("/lists/addfavorite.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":true}`))
	})
	mux.HandleFunc("/lists/deletefavorite.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":true}`))
	})
	mux.HandleFunc("/lists/101/stop.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/lists/101/reset", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>reset</html>`))
	})

	// Dictionary endpoints
	mux.HandleFunc("/wordfinder/search.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(WordFinderResponse{
			TotalResults: 1,
			Words:        []WordFinderWord{{Word: "callow"}},
		})
	})
	mux.HandleFunc("/randomword", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<div class="wordPage" data-word="serendipity" data-learnable="true"><p class="short">Good luck.</p></div>`))
	})
	mux.HandleFunc("/challenge/preview.json", func(w http.ResponseWriter, r *http.Request) {
		slide := base64.StdEncoding.EncodeToString([]byte(`<div class="question typeD"><div class="choices"><a data-nonce="p1">choice</a></div></div>`))
		_ = json.NewEncoder(w).Encode(PreviewQuestionResponse{Word: "callow", Secret: "ps", Code: slide})
	})
	mux.HandleFunc("/dictionary/audio.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"result": "C/1XB5QC9VRN0HU"})
	})
	mux.HandleFunc("/word-of-the-day/archive/by-month.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(WODArchiveResponse{HasMore: false})
	})

	// Progress endpoints
	mux.HandleFunc("/1.0/progress/words", func(w http.ResponseWriter, r *http.Request) {
		words := r.URL.Query()["words"]
		out := make([]WordProgressDetail, len(words))
		for i, wd := range words {
			out[i] = WordProgressDetail{Word: wd, Learnable: true}
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("/1.0/progress/words/callow", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(WordProgressDetail{Word: "callow", Learnable: true})
	})
	mux.HandleFunc("/progress/startlearning.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/progress/stoplearning.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/account/progress/words.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]AccountProgressWord{{Word: "callow", Progress: 0.5}})
	})
	mux.HandleFunc("/1.0/progress/lists/101", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ListProgressStatus{LearnableWordCount: 10, Progress: 0.4})
	})
	mux.HandleFunc("/1.0/me/activities", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<https://api.vocabulary.com/1.0/me/activities?page=2>; rel="next"`)
		_ = json.NewEncoder(w).Encode(ActivitiesResponse{
			Activities: []ActivityItem{{ID: "act1", Progress: 0.5}},
		})
	})
	mux.HandleFunc("/1.0/me/achievements", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"achievements": []AchievementSummary{{ID: "DSTK1", Milestone: 1}},
		})
	})
	mux.HandleFunc("/1.0/profiles/u_vt", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"profile": ProfileSummary{UID: "u_vt", Nickname: "VTUser"},
		})
	})
	mux.HandleFunc("/1.0/leaderboards/individual/points/daily/20261031", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(LeaderboardResponse{Domain: "individuals", Metric: "points"})
	})
	mux.HandleFunc("/lists/101/bee/surrender.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(BeeStateResponse{Secret: "bs", Word: "callow"})
	})
	mux.HandleFunc("/lists/101/bee/reset", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`reset`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(
		WithBaseURLs(srv.URL, srv.URL+"/1.0", srv.URL+"/corpus/1.0", srv.URL+"/audio/1.0/us"),
		WithRateLimit(0, 0),
		WithPacingGuard(false),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// Auth LoginVocabTrainer & ParseJWTClaims
	if _, err := client.Auth.LoginVocabTrainer(ctx, "vtuser", "pw"); err != nil {
		t.Fatalf("LoginVocabTrainer: %v", err)
	}
	claims, err := ParseJWTClaims("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1X3Z0In0.sig")
	if err != nil || claims["sub"] != "u_vt" {
		t.Errorf("ParseJWTClaims: %+v, err=%v", claims, err)
	}

	// Lists methods
	if _, err := client.Lists.GetFlashcardList(ctx, 101); err != nil {
		t.Errorf("GetFlashcardList: %v", err)
	}
	if uLists, err := client.Lists.ListUserLists(ctx, "created", "", 10); err != nil || len(uLists) != 1 {
		t.Errorf("ListUserLists: %+v, err=%v", uLists, err)
	}
	featured := true
	uSearch, err := client.Lists.SearchUnits(ctx, SearchUnitsOptions{
		Query:    "lit",
		Featured: &featured,
		Limit:    10,
	})
	if err != nil || uSearch.TotalCount != 42 || uSearch.NextPageURI == "" {
		t.Errorf("SearchUnits: %+v, err=%v", uSearch, err)
	}
	if senses, err := client.Lists.GetWordSenses(ctx, "callow"); err != nil || len(senses) != 1 {
		t.Errorf("GetWordSenses: %+v, err=%v", senses, err)
	}
	if vWords, valid, err := client.Lists.ValidateWords(ctx, []string{"callow"}); err != nil || !valid || len(vWords) != 1 {
		t.Errorf("ValidateWords: %+v, valid=%v, err=%v", vWords, valid, err)
	}
	if grabbed, err := client.Lists.GrabVocabulary(ctx, "callow youth"); err != nil || len(grabbed.Words) != 1 {
		t.Errorf("GrabVocabulary: %+v, err=%v", grabbed, err)
	}
	if _, err := client.Lists.AddWords(ctx, 101, []CreateListWordInput{{Word: "callow"}}); err != nil {
		t.Errorf("AddWords: %v", err)
	}
	if err := client.Lists.SetFavorite(ctx, 101, true); err != nil {
		t.Errorf("SetFavorite(true): %v", err)
	}
	if err := client.Lists.SetFavorite(ctx, 101, false); err != nil {
		t.Errorf("SetFavorite(false): %v", err)
	}
	if err := client.Lists.StopLearning(ctx, 101); err != nil {
		t.Errorf("StopLearning: %v", err)
	}
	if err := client.Lists.ResetPractice(ctx, 101); err != nil {
		t.Errorf("ResetPractice: %v", err)
	}
	if err := client.Lists.Delete(ctx, 101); err != nil {
		t.Errorf("Delete: %v", err)
	}

	// Dictionary methods
	if wf, err := client.Dictionary.WordFinder(ctx, WordFinderOptions{SearchText: "cal", NumLetters: 6}); err != nil || wf.TotalResults != 1 {
		t.Errorf("WordFinder: %+v, err=%v", wf, err)
	}
	if rw, err := client.Dictionary.RandomWord(ctx); err != nil || rw.Word != "serendipity" {
		t.Errorf("RandomWord: %+v, err=%v", rw, err)
	}
	if pq, err := client.Dictionary.PreviewQuestion(ctx, "callow"); err != nil || pq.ParsedQuestion == nil {
		t.Errorf("PreviewQuestion: %+v, err=%v", pq, err)
	}
	if audioURL, err := client.Dictionary.ResolveWordAudioURL(ctx, "callow", 0); err != nil || !strings.HasSuffix(audioURL, "C/1XB5QC9VRN0HU.mp3") {
		t.Errorf("ResolveWordAudioURL: %q, err=%v", audioURL, err)
	}
	if _, err := client.Dictionary.WordOfTheDay(ctx, "invalid-date"); err == nil {
		t.Error("expected error for invalid WordOfTheDay date format")
	}
	if _, err := client.Dictionary.WordOfTheDayArchive(ctx, 202610, 3); err != nil {
		t.Errorf("WordOfTheDayArchive: %v", err)
	}

	// CorpusSentence SourceURL resolution
	csGutenberg := CorpusSentence{VolumeID: 10, Volume: &CorpusVolume{Locator: "1342", Corpus: &struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: "GUT"}}}
	if csGutenberg.SourceURL() != "https://www.gutenberg.org/ebooks/1342" {
		t.Errorf("unexpected Gutenberg SourceURL: %s", csGutenberg.SourceURL())
	}

	// Progress methods (including >50 words batch chunking)
	manyWords := make([]string, 55)
	for i := range manyWords {
		manyWords[i] = fmt.Sprintf("word_%d", i)
	}
	batchRes, err := client.Progress.GetWordsProgressBatch(ctx, manyWords)
	if err != nil || len(batchRes) != 55 {
		t.Errorf("GetWordsProgressBatch: len=%d, err=%v", len(batchRes), err)
	}
	if _, err := client.Progress.SetWordPriority(ctx, "callow", PriorityHigh); err != nil {
		t.Errorf("SetWordPriority: %v", err)
	}
	if _, err := client.Progress.SetWordPriority(ctx, "callow", WordPriority(99)); err == nil {
		t.Error("expected error for invalid WordPriority")
	}
	if err := client.Progress.StartLearningWord(ctx, "callow"); err != nil {
		t.Errorf("StartLearningWord: %v", err)
	}
	if err := client.Progress.StopLearningWord(ctx, "callow"); err != nil {
		t.Errorf("StopLearningWord: %v", err)
	}
	if lw, err := client.Progress.ListLearningWords(ctx, WordFilterTrouble, 0, "date", "desc"); err != nil || len(lw) != 1 {
		t.Errorf("ListLearningWords: %+v, err=%v", lw, err)
	}
	if _, err := client.Progress.ListLearningWords(ctx, WordFilter("bad"), 0, "", ""); err == nil {
		t.Error("expected error for invalid WordFilter")
	}
	if _, err := client.Progress.GetListProgress(ctx, 101); err != nil {
		t.Errorf("GetListProgress: %v", err)
	}
	actResp, err := client.Progress.GetActivities(ctx, ActivitiesQuery{Limit: 5})
	if err != nil || actResp.NextPageURI == "" {
		t.Errorf("GetActivities: %+v, err=%v", actResp, err)
	}
	if _, err := client.Progress.GetActivitiesByURI(ctx, actResp.NextPageURI); err != nil {
		t.Errorf("GetActivitiesByURI: %v", err)
	}
	if achs, err := client.Progress.GetAchievements(ctx, 10); err != nil || len(achs) != 1 {
		t.Errorf("GetAchievements: %+v, err=%v", achs, err)
	}
	if prof, err := client.Progress.GetProfile(ctx, "u_vt"); err != nil || prof.Nickname != "VTUser" {
		t.Errorf("GetProfile: %+v, err=%v", prof, err)
	}
	if lb, err := client.Progress.GetIndividualLeaderboard(ctx, "points", "daily", "20261031", 10, "u_vt"); err != nil || lb.Metric != "points" {
		t.Errorf("GetIndividualLeaderboard: %+v, err=%v", lb, err)
	}

	// Bee surrender & reset
	if bs, err := client.Bee.Surrender(ctx, 101, "sec", 1000, ""); err != nil || bs.Word != "callow" {
		t.Errorf("Bee.Surrender: %+v, err=%v", bs, err)
	}
	if err := client.Bee.Reset(ctx, 101); err != nil {
		t.Errorf("Bee.Reset: %v", err)
	}

	// Logout
	if err := client.Auth.Logout(ctx); err != nil {
		t.Errorf("Logout: %v", err)
	}

	// DotEnv file loader
	envPath := filepath.Join(t.TempDir(), ".env.test")
	_ = os.WriteFile(envPath, []byte("# comment\nexport VOCAB_TEST_DOTENV_VAR=\"hello_world\"\n"), 0o600)
	if err := LoadDotEnvFile(envPath); err != nil {
		t.Errorf("LoadDotEnvFile: %v", err)
	}
	if os.Getenv("VOCAB_TEST_DOTENV_VAR") != "hello_world" {
		t.Errorf("VOCAB_TEST_DOTENV_VAR = %q, want hello_world", os.Getenv("VOCAB_TEST_DOTENV_VAR"))
	}
	_ = os.Unsetenv("VOCAB_TEST_DOTENV_VAR")
}
