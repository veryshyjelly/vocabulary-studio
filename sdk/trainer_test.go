package vocab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSpellingTrainerSessionAndQuizService(t *testing.T) {
	spellHTML := `<div class="challenge-slide" data-audio="C/1XB5QC9VRN0HU">
		<div class="question typeT">
			<div class="mode"><div class="mastery"></div><div class="pointValue">150</div></div>
			<div class="questionContent"><div class="sentence">A <span class="blanked">______</span> youth.</div></div>
			<div class="spelltheword"><input type="text" class="wordspelling" /></div>
		</div>
	</div>`
	encodedSpell := base64.StdEncoding.EncodeToString([]byte(spellHTML))

	mux := http.NewServeMux()
	mux.HandleFunc("/challenge/start.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ChallengeStateResponse{
			Version:  4,
			Action:   "newround",
			Secret:   "spell_sec_1",
			Question: &RawQuestion{Turn: 1, Type: QuestionTypeSpelling, Code: encodedSpell},
		})
	})
	mux.HandleFunc("/challenge/saveanswer.json", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		ans := r.FormValue("a")
		_ = json.NewEncoder(w).Encode(SaveAnswerResponse{
			Secret: "spell_sec_2",
			Answer: &AnswerOutcome{
				Correct:         ans == "callow",
				AcceptedAnswers: []string{"callow"},
				Word:            "callow",
				Points:          150,
			},
		})
	})
	mux.HandleFunc("/quiz/start.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(QuizStateResponse{
			Secret:         "quiz_sec_1",
			QuestionNumber: 1,
			QuestionTotal:  2,
			QuestionHTML: `<div class="question typeD"><div class="choices">
				<a data-nonce="qn0">wrong</a><a data-nonce="qn1">right</a>
			</div></div>`,
		})
	})
	mux.HandleFunc("/quiz/answer.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(QuizStateResponse{
			Secret:       "quiz_sec_done",
			QuestionHTML: `<div class="results">Done</div>`,
			ActivityID:   "aid_555",
		})
	})
	mux.HandleFunc("/quiz/results/aid_555", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<ol class="questions"><li data-choice="1" data-ao="2143" data-stats="[80,10,5,5]">Q1</li></ol>`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	fc := newFakeClock(time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC))
	client, err := NewClient(
		WithBaseURLs(srv.URL, srv.URL, srv.URL, srv.URL),
		WithClock(fc),
		WithRateLimit(0, 0),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	session, _, err := client.Trainer.NewSession(ctx, StartChallengeRequest{})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	// Calling ChooseIndex on a spelling question must return ErrInvalidStateTransition
	if _, _, err := session.ChooseIndex(ctx, 0); !errors.Is(err, ErrInvalidStateTransition) {
		t.Errorf("expected ErrInvalidStateTransition when calling ChooseIndex on spelling question, got %v", err)
	}

	// 1st spelling try: wrong
	out1, ok1, err := session.Spell(ctx, "calow")
	if err != nil || ok1 || out1.EvaluatedLocally {
		t.Fatalf("1st spell try unexpected: ok=%v out=%+v err=%v", ok1, out1, err)
	}

	// 2nd spelling try: right (evaluated locally against AcceptedAnswers)
	out2, ok2, err := session.Spell(ctx, "  CALLOW ")
	if err != nil || !ok2 || !out2.EvaluatedLocally || out2.Points != 0 {
		t.Fatalf("2nd spell try unexpected: ok=%v out=%+v err=%v", ok2, out2, err)
	}

	// Quiz flow with fake clock verifying 1550ms wait
	qState, err := client.Quiz.Start(ctx, StartQuizRequest{QuizID: "q1"})
	if err != nil || qState.IsComplete {
		t.Fatalf("Quiz.Start failed: %+v, err=%v", qState, err)
	}
	beforeSubmit := fc.Now()
	qDone, err := client.Quiz.SubmitAnswer(ctx, qState.Secret, "qn1", qState.ShownAt)
	if err != nil || !qDone.IsComplete {
		t.Fatalf("Quiz.SubmitAnswer failed: %+v, err=%v", qDone, err)
	}
	if fc.Now().Sub(beforeSubmit) != time.Duration(MinQuizSubmitMS)*time.Millisecond {
		t.Errorf("Quiz.SubmitAnswer waited %v, want %dms", fc.Now().Sub(beforeSubmit), MinQuizSubmitMS)
	}
	qRes, err := client.Quiz.GetResults(ctx, qDone.ActivityID)
	if err != nil || len(qRes) != 1 || qRes[0].Stats[0] != 80 {
		t.Fatalf("Quiz.GetResults failed: %+v, err=%v", qRes, err)
	}
}
