package desktop

import (
	"context"
	"fmt"
	"strings"
	"time"

	vocab "github.com/veryshyjelly/vocabulary_go/sdk"
)

// TrainerService exposes interactive VocabTrainer (v=4), List Practice (v=3),
// Spelling Bee, and Quiz workflows to the Wails frontend.
type TrainerService struct {
	hub *Hub
}

// StartSession initializes or resumes a VocabTrainer (listID == 0) or List Practice (listID > 0) session.
func (t *TrainerService) StartSession(ctx context.Context, listID int64, activityID string) (*TrainerSessionStateDTO, error) {
	ctx, cancel := t.hub.BeginOperation(ctx, "trainer:start", 30*time.Second)
	defer cancel()

	actType := vocab.ActivityVocabTrainer
	ver := vocab.ClientVersionVocabTrainer
	if listID > 0 {
		actType = vocab.ActivityListPractice
		ver = vocab.ClientVersionClassicWeb
	}

	req := vocab.StartChallengeRequest{
		ActivityType: actType,
		Version:      ver,
		WordlistID:   listID,
		ActivityID:   strings.TrimSpace(activityID),
	}

	session, state, err := t.hub.Client().Trainer.NewSession(ctx, req)
	if err != nil && session == nil {
		return nil, err
	}

	t.hub.trainerMu.Lock()
	t.hub.session = session
	t.hub.sessionActType = string(actType)
	t.hub.sessionListID = listID
	t.hub.lastState = state
	t.hub.lastOutcome = nil
	dto := t.buildSessionStateLocked()
	t.hub.trainerMu.Unlock()

	return dto, err
}

// GetSessionState returns the current TrainerSessionStateDTO without making a network call.
func (t *TrainerService) GetSessionState() *TrainerSessionStateDTO {
	t.hub.trainerMu.Lock()
	defer t.hub.trainerMu.Unlock()
	return t.buildSessionStateLocked()
}

// SubmitChoice submits a multiple-choice selection (0..3) to the active TrainerSession.
// All grading and pacing rules are enforced by the underlying Go SDK.
func (t *TrainerService) SubmitChoice(ctx context.Context, choiceIndex int) (*TrainerSessionStateDTO, error) {
	ctx, cancel := t.hub.BeginOperation(ctx, "trainer:submit", 45*time.Second)
	defer cancel()

	t.hub.trainerMu.Lock()
	session := t.hub.session
	t.hub.trainerMu.Unlock()

	if session == nil {
		return nil, fmt.Errorf("vocab: no active trainer session")
	}

	t.hub.Emit("trainer:pacing", map[string]any{"status": "submitting"})
	outcome, _, err := session.ChooseIndex(ctx, choiceIndex)
	if err != nil {
		return nil, err
	}

	fb := t.mapAnswerOutcome(outcome)

	t.hub.trainerMu.Lock()
	t.hub.lastOutcome = fb
	dto := t.buildSessionStateLocked()
	t.hub.trainerMu.Unlock()

	return dto, nil
}

// SubmitSpelling submits a spelling guess (`type == "T"`) to the active TrainerSession.
func (t *TrainerService) SubmitSpelling(ctx context.Context, rawGuess string) (*TrainerSessionStateDTO, error) {
	ctx, cancel := t.hub.BeginOperation(ctx, "trainer:submit", 45*time.Second)
	defer cancel()

	t.hub.trainerMu.Lock()
	session := t.hub.session
	t.hub.trainerMu.Unlock()

	if session == nil {
		return nil, fmt.Errorf("vocab: no active trainer session")
	}

	t.hub.Emit("trainer:pacing", map[string]any{"status": "submitting"})
	outcome, _, err := session.Spell(ctx, rawGuess)
	if err != nil {
		return nil, err
	}

	fb := t.mapAnswerOutcome(outcome)

	t.hub.trainerMu.Lock()
	t.hub.lastOutcome = fb
	dto := t.buildSessionStateLocked()
	t.hub.trainerMu.Unlock()

	return dto, nil
}

// UseHint invokes a lifeline ("F" 50/50, "D" Definition, or "E" Word in the Wild) on the active question.
func (t *TrainerService) UseHint(ctx context.Context, hintType string) (*HintResultDTO, error) {
	ctx, cancel := t.hub.BeginOperation(ctx, "trainer:hint", 20*time.Second)
	defer cancel()

	t.hub.trainerMu.Lock()
	session := t.hub.session
	t.hub.trainerMu.Unlock()

	if session == nil {
		return nil, fmt.Errorf("vocab: no active trainer session")
	}

	ht := vocab.HintType(strings.ToUpper(strings.TrimSpace(hintType)))
	resp, err := session.Hint(ctx, ht)
	if err != nil {
		return nil, err
	}

	defPlain, defSpans := ParseSafeRichSpans(resp.Definition)
	out := &HintResultDTO{
		HintType:         string(ht),
		DefinitionText:   defPlain,
		DefinitionSpans:  defSpans,
		EliminatedNonces: resp.Nonces,
		Word:             resp.Word,
	}

	// When HintType == "E" (Word in the Wild), fetch 1 real corpus example sentence for the word
	if ht == vocab.HintExample && resp.Word != "" {
		if exResp, exErr := t.hub.Client().Dictionary.Examples(ctx, resp.Word, vocab.CorpusDomainAll, 0, 1); exErr == nil && len(exResp.Sentences) > 0 {
			out.ExampleSentence = exResp.Sentences[0].Sentence
			if exResp.Sentences[0].Volume != nil {
				out.ExampleSource = exResp.Sentences[0].Volume.Title
			}
		}
	}

	return out, nil
}

// NextQuestion advances the active TrainerSession to the next question slide.
func (t *TrainerService) NextQuestion(ctx context.Context) (*TrainerSessionStateDTO, error) {
	ctx, cancel := t.hub.BeginOperation(ctx, "trainer:next", 30*time.Second)
	defer cancel()

	t.hub.trainerMu.Lock()
	session := t.hub.session
	t.hub.trainerMu.Unlock()

	if session == nil {
		return nil, fmt.Errorf("vocab: no active trainer session")
	}

	state, err := session.Next(ctx)
	if err != nil && state == nil {
		return nil, err
	}

	t.hub.trainerMu.Lock()
	t.hub.lastState = state
	t.hub.lastOutcome = nil
	dto := t.buildSessionStateLocked()
	t.hub.trainerMu.Unlock()

	return dto, err
}

// ResetChallenge resets an out-of-sync challenge state on the server.
func (t *TrainerService) ResetChallenge(ctx context.Context) error {
	ctx, cancel := t.hub.BeginOperation(ctx, "trainer:reset", 20*time.Second)
	defer cancel()

	err := t.hub.Client().Trainer.Reset(ctx)
	t.EndSession()
	return err
}

// EndSession clears the local TrainerSession state.
func (t *TrainerService) EndSession() {
	t.hub.CancelOperation("trainer:submit")
	t.hub.trainerMu.Lock()
	t.hub.session = nil
	t.hub.lastState = nil
	t.hub.lastOutcome = nil
	t.hub.trainerMu.Unlock()
}

func (t *TrainerService) buildSessionStateLocked() *TrainerSessionStateDTO {
	if t.hub.session == nil {
		return &TrainerSessionStateDTO{
			Active:       false,
			SessionState: vocab.SessionStateIdle,
		}
	}
	pq := t.hub.session.CurrentQuestion()
	var slideDTO *QuestionSlideDTO
	if pq != nil {
		slideDTO = mapParsedQuestionToDTO(pq, t.hub.session.IsNonceEliminated)
	}
	dto := &TrainerSessionStateDTO{
		Active:       true,
		SessionState: t.hub.session.State(),
		ActivityType: t.hub.sessionActType,
		WordlistID:   t.hub.sessionListID,
		Tries:        t.hub.session.Tries(),
		Question:     slideDTO,
		LastFeedback: t.hub.lastOutcome,
	}
	if t.hub.lastState != nil {
		dto.Round = t.hub.lastState.Round
		dto.Game = t.hub.lastState.Game
		dto.PlayerData = t.hub.lastState.PlayerData
	}
	return dto
}

func mapParsedQuestionToDTO(pq *vocab.ParsedQuestion, isEliminated func(string) bool) *QuestionSlideDTO {
	if pq == nil {
		return nil
	}
	instText, instSpans := ParseSafeRichSpans(pq.InstructionsHTML)
	if instText == "" {
		instText = pq.InstructionsText
	}
	sentText, sentSpans := ParseSafeRichSpans(pq.SentenceHTML)
	if sentText == "" {
		sentText = pq.SentenceText
	}
	choices := make([]ChoiceDTO, len(pq.Choices))
	for i, ch := range pq.Choices {
		choices[i] = ChoiceDTO{
			Index:      ch.Index,
			Nonce:      ch.Nonce,
			AccessKey:  ch.AccessKey,
			Text:       ch.Text,
			ImageURL:   ch.ImageURL,
			Eliminated: isEliminated != nil && isEliminated(ch.Nonce),
		}
	}
	hints := make([]string, len(pq.Hints))
	for i, h := range pq.Hints {
		hints[i] = string(h)
	}
	return &QuestionSlideDTO{
		Turn:              pq.Turn,
		Type:              string(pq.Type),
		TypeLabel:         QuestionTypeLabel(string(pq.Type)),
		Mode:              pq.Mode,
		PointValue:        pq.PointValue,
		AudioURL:          pq.AudioURL,
		TargetWord:        pq.TargetWord,
		InstructionsText:  instText,
		InstructionsSpans: instSpans,
		SentenceText:      sentText,
		SentenceSpans:     sentSpans,
		BlankedWord:       pq.BlankedWord,
		SourceText:        pq.SourceText,
		PromptImageURL:    pq.PromptImageURL,
		IsSpelling:        pq.IsSpelling,
		IsImageGrid:       pq.Type == vocab.QuestionTypeImage4,
		AvailableHints:    hints,
		Choices:           choices,
		AnswerStats:       pq.AnswerStats,
	}
}

func (t *TrainerService) mapAnswerOutcome(ao *vocab.AnswerOutcome) *AnswerFeedbackDTO {
	if ao == nil {
		return nil
	}
	fb := &AnswerFeedbackDTO{
		Correct:          ao.Correct,
		CorrectChoice:    ao.CorrectChoice,
		UserChoice:       ao.UserChoice,
		AcceptedAnswers:  ao.AcceptedAnswers,
		Word:             ao.Word,
		PointsEarned:     ao.Points,
		BonusPoints:      ao.Bonus,
		Attempt:          ao.Attempt,
		EvaluatedLocally: ao.EvaluatedLocally,
		Achievements:     ao.Achievements,
	}
	if ao.Blurb != nil {
		fb.ShortBlurbText, fb.ShortBlurbSpans = ParseSafeRichSpans(ao.Blurb.Short)
		fb.LongBlurbText, fb.LongBlurbSpans = ParseSafeRichSpans(ao.Blurb.Long)
	}
	if ao.Sense != nil {
		fb.PartOfSpeech = ao.Sense.PartOfSpeech
		fb.SenseDefinition = ao.Sense.Definition
		if len(ao.Sense.Audio) > 0 {
			fb.AudioURL = t.hub.Client().Dictionary.AudioURLForID(ao.Sense.Audio[0])
		}
	}
	if ao.SentenceHTML != "" {
		fb.UnblankedSentence, _ = ParseSafeRichSpans(ao.SentenceHTML)
	}
	if ao.Progress != nil {
		fb.WordProgress = ao.Progress.Progress
	}
	return fb
}
