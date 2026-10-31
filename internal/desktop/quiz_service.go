package desktop

import (
	"context"
	"fmt"
	"strings"
	"time"

	vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
)

// StartQuiz begins or resumes a Quiz (`POST /quiz/start.json`).
func (t *TrainerService) StartQuiz(ctx context.Context, quizID, assignmentID, activityID string) (*QuizSessionDTO, error) {
	ctx, cancel := t.hub.BeginOperation(ctx, "quiz:start", 30*time.Second)
	defer cancel()

	state, err := t.hub.Client().Quiz.Start(ctx, vocab.StartQuizRequest{
		QuizID:       strings.TrimSpace(quizID),
		AssignmentID: strings.TrimSpace(assignmentID),
		ActivityID:   strings.TrimSpace(activityID),
	})
	if err != nil {
		return nil, err
	}

	t.hub.quizMu.Lock()
	t.hub.quizState = state
	dto := t.buildQuizSessionLocked()
	t.hub.quizMu.Unlock()

	return dto, nil
}

// SubmitQuizAnswer submits an answer in the active Quiz session (enforcing the >= 1550ms submit delay).
func (t *TrainerService) SubmitQuizAnswer(ctx context.Context, answerNonceOrSpelling string) (*QuizSessionDTO, error) {
	ctx, cancel := t.hub.BeginOperation(ctx, "quiz:submit", 30*time.Second)
	defer cancel()

	t.hub.quizMu.Lock()
	qs := t.hub.quizState
	t.hub.quizMu.Unlock()

	if qs == nil {
		return nil, fmt.Errorf("vocab: no active quiz session")
	}

	t.hub.Emit("trainer:pacing", map[string]any{"status": "submitting"})
	nextState, err := t.hub.Client().Quiz.SubmitAnswer(ctx, qs.Secret, answerNonceOrSpelling, qs.ShownAt)
	if err != nil {
		return nil, err
	}

	t.hub.quizMu.Lock()
	t.hub.quizState = nextState
	dto := t.buildQuizSessionLocked()
	t.hub.quizMu.Unlock()

	return dto, nil
}

// GetQuizResults fetches the per-question breakdown for a completed quiz.
func (t *TrainerService) GetQuizResults(ctx context.Context, activityID string) ([]vocab.QuizQuestionResult, error) {
	ctx, cancel := t.hub.BeginOperation(ctx, "quiz:results", 20*time.Second)
	defer cancel()
	return t.hub.Client().Quiz.GetResults(ctx, activityID)
}

func (t *TrainerService) buildQuizSessionLocked() *QuizSessionDTO {
	qs := t.hub.quizState
	if qs == nil {
		return &QuizSessionDTO{Active: false}
	}
	var qDTO *QuestionSlideDTO
	if qs.ParsedSlide != nil && !qs.IsComplete {
		pq := &vocab.ParsedQuestion{
			Type:             vocab.QuestionType(qs.ParsedSlide.QuestionType),
			Mode:             qs.ParsedSlide.Mode,
			PointValue:       qs.ParsedSlide.PointValue,
			AudioID:          qs.ParsedSlide.AudioID,
			AudioURL:         t.hub.Client().Dictionary.AudioURLForID(qs.ParsedSlide.AudioID),
			TargetWord:       qs.ParsedSlide.TargetWord,
			InstructionsText: qs.ParsedSlide.InstructionsText,
			InstructionsHTML: qs.ParsedSlide.InstructionsHTML,
			SentenceText:     qs.ParsedSlide.SentenceText,
			SentenceHTML:     qs.ParsedSlide.SentenceHTML,
			BlankedWord:      qs.ParsedSlide.BlankedWord,
			SourceText:       qs.ParsedSlide.SourceText,
			PromptImageURL:   qs.ParsedSlide.PromptImageURL,
			IsSpelling:       qs.ParsedSlide.IsSpelling,
			Choices:          qs.ParsedSlide.Choices,
		}
		qDTO = mapParsedQuestionToDTO(pq, nil)
	}
	return &QuizSessionDTO{
		Active:         true,
		QuestionNumber: qs.QuestionNumber,
		QuestionTotal:  qs.QuestionTotal,
		IsComplete:     qs.IsComplete,
		ActivityID:     qs.ActivityID,
		Question:       qDTO,
		Game:           qs.Game,
		PlayerData:     qs.PlayerData,
	}
}
