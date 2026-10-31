package desktop

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	vocab "github.com/veryshyjelly/vocabulary_go/sdk"
)

func TestParseSafeRichSpansSanitizesHTML(t *testing.T) {
	raw := `When someone is <strong>callow</strong>, they are <i>very</i> young.<script>alert("xss")</script>`
	plain, spans := ParseSafeRichSpans(raw)
	if strings.Contains(plain, "alert") {
		t.Fatalf("script content leaked into plainText: %q", plain)
	}
	if plain != "When someone is callow, they are very young." {
		t.Errorf("plain = %q", plain)
	}
	foundBold := false
	foundItalic := false
	for _, sp := range spans {
		if sp.Bold && sp.Text == "callow" {
			foundBold = true
		}
		if sp.Italic && sp.Text == "very" {
			foundItalic = true
		}
		if strings.Contains(sp.Text, "alert") {
			t.Errorf("script content leaked into span: %+v", sp)
		}
	}
	if !foundBold || !foundItalic {
		t.Errorf("expected bold 'callow' and italic 'very' spans, got %+v", spans)
	}
}

func TestDesktopHubAndServicesEndToEnd(t *testing.T) {
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
		http.SetCookie(w, &http.Cookie{Name: "SESSION", Value: "desk_sess", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "guid", Value: "desk_guid", Path: "/"})
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/auth/me.json", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("SESSION")
		if err != nil || c.Value != "desk_sess" {
			_ = json.NewEncoder(w).Encode(vocab.LoginData{ValidUser: false})
			return
		}
		_ = json.NewEncoder(w).Encode(vocab.LoginData{
			ValidUser: true,
			GUID:      "desk_guid",
			Paid:      true,
			Points:    8400,
			Level:     vocab.UserLevel{ID: "L03", Name: "Scholar"},
			Auth:      vocab.AuthDetails{LoggedIn: true, UID: "u_desk", Nickname: "DeskUser", Email: "desk@example.com"},
			Perms:     vocab.UserPerms{FullVocabTrainer: true},
		})
	})
	mux.HandleFunc("/auth/logout.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/1.0/auth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.TokenInfo{
			TokenType:    "bearer",
			AccessToken:  "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1X2Rlc2sifQ.sig",
			RefreshToken: "desk_guid",
			ExpiresIn:    3600,
		})
	})

	// Trainer, Bee, Quiz endpoints
	mux.HandleFunc("/challenge/start.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.ChallengeStateResponse{
			Version: 4,
			Action:  "newround",
			Secret:  "sec_1",
			Round:   &vocab.ChallengeRound{Number: 1, PlayedCount: 0, Streak: 2},
			Question: &vocab.RawQuestion{
				Turn:  1,
				Type:  vocab.QuestionTypeDefinition,
				Code:  encodedSlide,
				Hints: []vocab.HintType{vocab.HintFiftyFifty, vocab.HintDefinition, vocab.HintExample},
			},
		})
	})
	mux.HandleFunc("/challenge/hint.json", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch r.FormValue("type") {
		case "F":
			_ = json.NewEncoder(w).Encode(vocab.HintResponse{Secret: "sec_1f", Nonces: []string{"n0", "n3"}})
		case "E":
			_ = json.NewEncoder(w).Encode(vocab.HintResponse{Secret: "sec_1e", Word: "callow"})
		default:
			_ = json.NewEncoder(w).Encode(vocab.HintResponse{Secret: "sec_1d", Definition: "<strong>young</strong> and inexperienced"})
		}
	})
	mux.HandleFunc("/challenge/saveanswer.json", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		ans := r.FormValue("a")
		_ = json.NewEncoder(w).Encode(vocab.SaveAnswerResponse{
			Secret: "sec_saved",
			Round:  &vocab.ChallengeRound{Number: 1, PlayedCount: 1, Streak: 3},
			Answer: &vocab.AnswerOutcome{
				Correct:         ans == "n1" || ans == "callow",
				CorrectChoice:   1,
				AcceptedAnswers: []string{"callow"},
				Word:            "callow",
				Points:          100,
				Blurb:           &vocab.WordBlurb{Short: "Short <strong>callow</strong> blurb", Long: "Long <i>callow</i> blurb"},
				Sense:           &vocab.AnswerWordSense{PartOfSpeech: "adjective", Definition: "young and inexperienced", Audio: []string{"C/1XB5QC9VRN0HU"}},
			},
		})
	})
	mux.HandleFunc("/challenge/nextquestion.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.ChallengeStateResponse{
			Version:  4,
			Action:   "next",
			Secret:   "sec_2",
			Question: &vocab.RawQuestion{Turn: 2, Type: vocab.QuestionTypeSpelling, Code: encodedSpell},
		})
	})
	mux.HandleFunc("/lists/500/bee/start.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.BeeStateResponse{
			Action:  "start",
			Secret:  "b1",
			Audio:   []string{"C/1XB5QC9VRN0HU"},
			Example: []string{"A ", " youth."},
			Defs:    []vocab.BeeDef{{POS: "adj", Def: []string{"young and ", ""}}},
			Score:   &vocab.BeeScore{Remaining: 5, Played: 0},
		})
	})
	mux.HandleFunc("/lists/500/bee/guess.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.BeeStateResponse{
			Secret:   "b2",
			Guesses:  1,
			Correct:  true,
			Word:     "callow",
			NextData: &vocab.BeeStateResponse{Secret: "b2", Example: []string{"Next ", " word."}},
		})
	})
	mux.HandleFunc("/quiz/start.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.QuizStateResponse{
			Secret:         "q1",
			QuestionNumber: 1,
			QuestionTotal:  1,
			QuestionHTML:   `<div class="question typeD"><div class="instructions">Pick one:</div><div class="choices"><a data-nonce="qn1">ans</a></div></div>`,
		})
	})
	mux.HandleFunc("/quiz/answer.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.QuizStateResponse{
			Secret:       "q2",
			ActivityID:   "qa_1",
			QuestionHTML: `<div class="results">Done</div>`,
		})
	})

	// Dictionary, Corpus, WOD, Lists, Units, Progress
	mux.HandleFunc("/dictionary/callow", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<div class="wordPage" data-word="callow" data-learnable="true">
			<div class="word-area"><h1>callow</h1><p class="short">Young.</p></div>
			<div class="word-definitions"><ol><li class="sense pos_a"><div class="definition"><div class="pos-icon">adjective</div>young and inexperienced</div></li></ol></div>
		</div>`))
	})
	mux.HandleFunc("/dictionary/autocomplete-ss", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<div class="suggestions"><div class="word-result suggestion learnable"><span class="word">callow</span><span class="definition">young</span></div></div>`))
	})
	mux.HandleFunc("/corpus/1.0/examples/random.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.CorpusExamplesResponse{
			TotalHits: 1,
			Sentences: []vocab.CorpusSentence{{Sentence: "A callow youth.", Volume: &vocab.CorpusVolume{Title: "Guardian"}}},
		})
	})
	mux.HandleFunc("/word-of-the-day/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<div><div class="box-yesterday-today-tomorrow">October 31, 2026</div><div class="box-lks-audio-wod"><a class="word-of-the-day">enchantment</a></div></div>`))
	})
	mux.HandleFunc("/1.0/progress/words/callow", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.WordProgressDetail{Word: "callow", Learnable: true})
	})
	mux.HandleFunc("/1.0/me/effort", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.EffortStats{WordsMastered: 7, QuestionsAnswered: 30, Points: 2100, TimeMinutes: 15})
	})
	mux.HandleFunc("/1.0/me/activities", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(vocab.ActivitiesResponse{Activities: []vocab.ActivityItem{{ID: "a1", Name: "GRE List", Progress: 0.5}}})
	})
	mux.HandleFunc("/account/progress/words.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]vocab.AccountProgressWord{{Word: "callow", Definition: "young", Progress: 0.4}})
	})
	mux.HandleFunc("/lists/byprofile.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"wordlists": []vocab.ProfileListSummary{{WordlistID: 500, Name: "My Study List", WordCount: 12}},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	var eventsMu sync.Mutex
	var emittedEvents []string

	cfgDir := t.TempDir()
	hub, err := NewHub(HubOptions{
		ConfigDir: cfgDir,
		ClientOpts: []vocab.Option{
			vocab.WithBaseURLs(srv.URL, srv.URL+"/1.0", srv.URL+"/corpus/1.0", srv.URL+"/audio/1.0/us"),
			vocab.WithPacingGuard(false),
			vocab.WithRateLimit(0, 0),
		},
		EmitEvent: func(name string, _ any) {
			eventsMu.Lock()
			emittedEvents = append(emittedEvents, name)
			eventsMu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("NewHub: %v", err)
	}
	defer hub.Shutdown()

	ctx := context.Background()

	// 1. Login with File SessionStoreMode
	status, err := hub.Auth.Login(ctx, "desk@example.com", "pw", "file")
	if err != nil {
		t.Fatalf("Auth.Login: %v", err)
	}
	if !status.Authenticated || status.Nickname != "DeskUser" || status.SessionStoreMode != SessionStoreFile {
		t.Fatalf("unexpected AccountStatusDTO: %+v", status)
	}

	// 2. Home Dashboard aggregation
	dash, err := hub.Progress.GetHomeDashboard(ctx)
	if err != nil {
		t.Fatalf("GetHomeDashboard: %v", err)
	}
	if !dash.Account.Authenticated || dash.WordOfTheDay == nil || dash.Effort == nil || len(dash.LearningWords) == 0 || len(dash.RecentLists) == 0 {
		t.Errorf("incomplete HomeDashboardDTO: %+v", dash)
	}

	// 3. Dictionary Lookup & Search History
	detail, err := hub.Dictionary.Lookup(ctx, "callow")
	if err != nil || detail.Entry == nil || detail.Entry.Word != "callow" || len(detail.Examples) == 0 {
		t.Fatalf("Dictionary.Lookup: %+v, err=%v", detail, err)
	}
	hist := hub.Dictionary.GetSearchHistory()
	if len(hist) != 1 || hist[0] != "callow" {
		t.Errorf("unexpected search history: %v", hist)
	}

	// 4. TrainerSession workflow (Start -> Hint -> Wrong 1st try -> Right 2nd try -> Next -> Spell)
	trState, err := hub.Trainer.StartSession(ctx, 0, "")
	if err != nil || !trState.Active || trState.Question == nil {
		t.Fatalf("Trainer.StartSession: %+v, err=%v", trState, err)
	}
	hintRes, err := hub.Trainer.UseHint(ctx, "F")
	if err != nil || len(hintRes.EliminatedNonces) != 2 {
		t.Fatalf("Trainer.UseHint(F): %+v, err=%v", hintRes, err)
	}
	// Wrong 1st try (choice 0)
	afterTry1, err := hub.Trainer.SubmitChoice(ctx, 0)
	if err != nil || afterTry1.LastFeedback == nil || afterTry1.LastFeedback.Correct || afterTry1.LastFeedback.EvaluatedLocally {
		t.Fatalf("SubmitChoice(0): %+v, err=%v", afterTry1, err)
	}
	// Right 2nd try (choice 1)
	afterTry2, err := hub.Trainer.SubmitChoice(ctx, 1)
	if err != nil || afterTry2.LastFeedback == nil || !afterTry2.LastFeedback.Correct || !afterTry2.LastFeedback.EvaluatedLocally {
		t.Fatalf("SubmitChoice(1): %+v, err=%v", afterTry2, err)
	}
	// Advance to Spelling Question
	nextSt, err := hub.Trainer.NextQuestion(ctx)
	if err != nil || nextSt.Question == nil || !nextSt.Question.IsSpelling {
		t.Fatalf("NextQuestion: %+v, err=%v", nextSt, err)
	}
	afterSpell, err := hub.Trainer.SubmitSpelling(ctx, "Callow")
	if err != nil || afterSpell.LastFeedback == nil || !afterSpell.LastFeedback.Correct {
		t.Fatalf("SubmitSpelling: %+v, err=%v", afterSpell, err)
	}

	// 5. Spelling Bee & Quiz
	beeSt, err := hub.Trainer.StartBee(ctx, 500, "")
	if err != nil || !beeSt.Active {
		t.Fatalf("StartBee: %+v, err=%v", beeSt, err)
	}
	beeGuessed, err := hub.Trainer.GuessBee(ctx, "callow")
	if err != nil || !beeGuessed.Correct {
		t.Fatalf("GuessBee: %+v, err=%v", beeGuessed, err)
	}
	if _, err := hub.Trainer.NextBeeWord(); err != nil {
		t.Errorf("NextBeeWord: %v", err)
	}

	quizSt, err := hub.Trainer.StartQuiz(ctx, "q1", "", "")
	if err != nil || !quizSt.Active {
		t.Fatalf("StartQuiz: %+v, err=%v", quizSt, err)
	}
	quizDone, err := hub.Trainer.SubmitQuizAnswer(ctx, "qn1")
	if err != nil || !quizDone.IsComplete {
		t.Fatalf("SubmitQuizAnswer: %+v, err=%v", quizDone, err)
	}

	// 6. Settings Preferences & SessionStore migration to Memory
	updatedPrefs, err := hub.Settings.UpdatePreferences(ctx, AppPreferences{
		Theme:               "dark",
		PronunciationAccent: "UK",
		DefaultView:         "practice",
		SessionStoreMode:    SessionStoreMemory,
	})
	if err != nil || updatedPrefs.Theme != "dark" || updatedPrefs.SessionStoreMode != SessionStoreMemory {
		t.Fatalf("UpdatePreferences: %+v, err=%v", updatedPrefs, err)
	}

	// 7. Additional Discovery, Lists, Dictionary, and Progress service methods
	cats := hub.Discovery.GetRootCategories()
	if len(cats) != 9 {
		t.Errorf("GetRootCategories len = %d, want 9", len(cats))
	}
	if wod, err := hub.Discovery.GetWordOfTheDay(ctx, ""); err != nil || wod.Word != "enchantment" {
		t.Errorf("Discovery.GetWordOfTheDay: %+v, err=%v", wod, err)
	}
	if ac, err := hub.Dictionary.Autocomplete(ctx, "callo", 5); err != nil || len(ac) != 1 {
		t.Errorf("Dictionary.Autocomplete: %+v, err=%v", ac, err)
	}
	if exList, err := hub.Dictionary.GetExamples(ctx, "callow", "", 0, 5); err != nil || len(exList) != 1 {
		t.Errorf("Dictionary.GetExamples: %+v, err=%v", exList, err)
	}
	if err := hub.Dictionary.ClearSearchHistory(); err != nil || len(hub.Dictionary.GetSearchHistory()) != 0 {
		t.Errorf("ClearSearchHistory failed: %v", err)
	}
	if uLists, err := hub.Lists.GetUserLists(ctx, ""); err != nil || len(uLists) != 1 {
		t.Errorf("Lists.GetUserLists: %+v, err=%v", uLists, err)
	}
	if lw, err := hub.Progress.ListLearningWords(ctx, "learning", 0, "", "desc"); err != nil || len(lw) != 1 {
		t.Errorf("Progress.ListLearningWords: %+v, err=%v", lw, err)
	}
	if acts, err := hub.Progress.GetActivities(ctx, "false", 10, ""); err != nil || len(acts.Activities) != 1 {
		t.Errorf("Progress.GetActivities: %+v, err=%v", acts, err)
	}
	if eff, err := hub.Progress.GetEffort(ctx, "2026-10-25", "2026-10-31"); err != nil || eff.WordsMastered != 7 {
		t.Errorf("Progress.GetEffort: %+v, err=%v", eff, err)
	}
	if _, err := hub.Auth.Logout(ctx); err != nil {
		t.Errorf("Auth.Logout: %v", err)
	}
}
