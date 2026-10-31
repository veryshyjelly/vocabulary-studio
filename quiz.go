package vocab

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/veryshyjelly/vocabulary_go/internal/htmlparse"
)

// QuizService implements Vocabulary.com's Quiz endpoints (`/quiz/start.json`, `/quiz/answer.json`, `/quiz/results/{aid}`).
type QuizService struct {
	client *Client
}

// StartQuizRequest configures `POST /quiz/start.json`.
type StartQuizRequest struct {
	QuizID       string // data-quizid (qid)
	AssignmentID string // data-assignmentid (asn)
	ActivityID   string // data-activityid (act)
}

// QuizStateResponse represents the JSON returned by `/quiz/start.json` and `/quiz/answer.json`.
type QuizStateResponse struct {
	Secret         string               `json:"secret"`
	QuestionNumber int                  `json:"qn"`
	QuestionTotal  int                  `json:"qt"`
	QuestionHTML   string               `json:"qs"` // Raw HTML string for the question slide or div.results
	ActivityID     string               `json:"aid,omitempty"`
	Game           *ChallengeGame       `json:"game,omitempty"`
	PlayerData     *ChallengePlayerData `json:"pdata,omitempty"`
	ParsedSlide    *htmlparse.SlideData `json:"parsed_slide,omitempty"`
	IsComplete     bool                 `json:"is_complete"`
	ShownAt        time.Time            `json:"-"`
}

// Start begins or resumes a Quiz (`POST /quiz/start.json`).
func (q *QuizService) Start(ctx context.Context, req StartQuizRequest) (*QuizStateResponse, error) {
	form := url.Values{
		"v": {"3"},
	}
	if req.ActivityID != "" {
		form.Set("act", req.ActivityID)
	}
	if req.AssignmentID != "" {
		form.Set("asn", req.AssignmentID)
	}
	if req.QuizID != "" {
		form.Set("qid", req.QuizID)
	}

	var resp QuizStateResponse
	if err := q.client.doWebForm(ctx, "/quiz/start.json", form, &resp); err != nil {
		return nil, err
	}
	if err := q.populateQuizSlide(&resp); err != nil {
		return &resp, err
	}
	return &resp, nil
}

// SubmitAnswer submits a single-attempt Quiz answer (`POST /quiz/answer.json`),
// enforcing the >= 1550ms minimum submit delay required by `vcom/quiz/quiz.js`.
func (q *QuizService) SubmitAnswer(ctx context.Context, secret, answerNonceOrSpelling string, shownAt time.Time) (*QuizStateResponse, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("vocab: secret is required for Quiz.SubmitAnswer")
	}
	if strings.TrimSpace(answerNonceOrSpelling) == "" {
		return nil, fmt.Errorf("vocab: answer is required for Quiz.SubmitAnswer")
	}
	if _, err := q.client.pacingGuard.EnsureMinResponseTime(ctx, shownAt, MinQuizSubmitMS); err != nil {
		return nil, err
	}
	form := url.Values{
		"secret": {secret},
		"a":      {answerNonceOrSpelling},
		"v":      {"3"},
	}
	var resp QuizStateResponse
	if err := q.client.doWebForm(ctx, "/quiz/answer.json", form, &resp); err != nil {
		return nil, err
	}
	if err := q.populateQuizSlide(&resp); err != nil {
		return &resp, err
	}
	return &resp, nil
}

// GetResults fetches and parses the detailed breakdown for a completed quiz (`GET /quiz/results/{aid}`).
func (q *QuizService) GetResults(ctx context.Context, activityID string) ([]htmlparse.QuizQuestionResult, error) {
	if strings.TrimSpace(activityID) == "" {
		return nil, fmt.Errorf("vocab: activityID is required for Quiz.GetResults")
	}
	rawBytes, _, err := q.client.doWebGet(ctx, "/quiz/results/"+url.PathEscape(activityID), nil, "text/html")
	if err != nil {
		return nil, err
	}
	return htmlparse.ParseQuizResultsHTML(string(rawBytes))
}

func (q *QuizService) populateQuizSlide(resp *QuizStateResponse) error {
	if resp == nil {
		return nil
	}
	resp.ShownAt = q.client.clock.Now()
	if strings.TrimSpace(resp.QuestionHTML) == "" {
		return nil
	}
	slide, err := htmlparse.ParseSlideHTML(resp.QuestionHTML, "")
	if err != nil {
		return err
	}
	resp.ParsedSlide = slide
	resp.IsComplete = slide.IsResultsSlide
	return nil
}
