package desktop

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	vocab "github.com/veryshyjelly/vocabulary_go"
	"github.com/veryshyjelly/vocabulary_go/internal/htmlparse"
)

// AuthService exposes authentication and session lifecycle methods to the Wails frontend.
// It never returns passwords, cookies, GUIDs, or JWTs.
type AuthService struct {
	hub *Hub
}

// GetStatus checks the current session via POST /auth/me.json and returns non-sensitive account info.
func (s *AuthService) GetStatus(ctx context.Context) (*AccountStatusDTO, error) {
	ctx, cancel := s.hub.BeginOperation(ctx, "auth:status", 20*time.Second)
	defer cancel()

	me, err := s.hub.Client().Auth.Me(ctx)
	if err != nil {
		return s.unauthenticatedStatus(), err
	}
	return s.mapLoginData(me), nil
}

// RestoreSession attempts to restore a persisted session from the configured SessionStore
// and verifies it against POST /auth/me.json.
func (s *AuthService) RestoreSession(ctx context.Context) (*AccountStatusDTO, error) {
	ctx, cancel := s.hub.BeginOperation(ctx, "auth:restore", 20*time.Second)
	defer cancel()

	client := s.hub.Client()
	if err := client.RestoreSession(ctx); err != nil {
		// If keyring isn't populated or returns not found, return unauthenticated status cleanly
		return s.unauthenticatedStatus(), nil
	}
	me, err := client.Auth.Me(ctx)
	if err != nil {
		return s.unauthenticatedStatus(), nil
	}
	status := s.mapLoginData(me)
	s.hub.Emit("auth:changed", status)
	return status, nil
}

// Login authenticates with Vocabulary.com using username/email and password,
// optionally setting the preferred SessionStoreMode ("keyring", "file", or "memory") first.
func (s *AuthService) Login(ctx context.Context, username, password, storeMode string) (*AccountStatusDTO, error) {
	ctx, cancel := s.hub.BeginOperation(ctx, "auth:login", 30*time.Second)
	defer cancel()

	if mode := SessionStoreMode(strings.TrimSpace(storeMode)); mode != "" && mode.IsValid() {
		if err := s.hub.SwitchSessionStoreMode(ctx, mode); err != nil {
			return nil, err
		}
	}

	me, err := s.hub.Client().Auth.Login(ctx, username, password)
	if err != nil {
		return nil, err
	}
	status := s.mapLoginData(me)
	s.hub.Emit("auth:changed", status)
	return status, nil
}

// Logout terminates the session on Vocabulary.com and clears local cookies, tokens, and persistent storage.
func (s *AuthService) Logout(ctx context.Context) (*AccountStatusDTO, error) {
	ctx, cancel := s.hub.BeginOperation(ctx, "auth:logout", 20*time.Second)
	defer cancel()

	if err := s.hub.Client().Auth.Logout(ctx); err != nil {
		return nil, err
	}
	s.hub.Trainer.EndSession()
	status := s.unauthenticatedStatus()
	s.hub.Emit("auth:changed", status)
	return status, nil
}

func (s *AuthService) unauthenticatedStatus() *AccountStatusDTO {
	s.hub.mu.RLock()
	mode := s.hub.prefs.SessionStoreMode
	s.hub.mu.RUnlock()
	return &AccountStatusDTO{
		Authenticated:    false,
		ValidUser:        false,
		SessionStoreMode: mode,
		Level:            vocab.UserLevel{ID: "L01", Name: "Novice"},
	}
}

func (s *AuthService) mapLoginData(me *vocab.LoginData) *AccountStatusDTO {
	if me == nil {
		return s.unauthenticatedStatus()
	}
	s.hub.mu.RLock()
	mode := s.hub.prefs.SessionStoreMode
	s.hub.mu.RUnlock()

	return &AccountStatusDTO{
		Authenticated:    me.ValidUser && me.Auth.LoggedIn,
		ValidUser:        me.ValidUser,
		UID:              me.Auth.UID,
		Nickname:         me.Auth.Nickname,
		FullName:         me.Auth.FullName,
		Email:            me.Auth.Email,
		Role:             me.Auth.Role,
		Points:           me.Points,
		Level:            me.Level,
		Paid:             me.Paid,
		FullVocabTrainer: me.Perms.FullVocabTrainer,
		SessionStoreMode: mode,
		Bowl:             me.Bowl,
	}
}

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

// StartBee starts or resumes a Spelling Bee session for `listID`.
func (t *TrainerService) StartBee(ctx context.Context, listID int64, activityID string) (*BeeSessionDTO, error) {
	ctx, cancel := t.hub.BeginOperation(ctx, "bee:start", 30*time.Second)
	defer cancel()

	resp, err := t.hub.Client().Bee.Start(ctx, listID, strings.TrimSpace(activityID), "")
	if err != nil {
		return nil, err
	}

	t.hub.beeMu.Lock()
	t.hub.beeListID = listID
	t.hub.beeActID = strings.TrimSpace(activityID)
	t.hub.beeShownAt = time.Now()
	t.hub.beeCurrent = resp
	t.hub.beePending = nil
	dto := t.buildBeeSessionLocked(false, false, 0, "")
	t.hub.beeMu.Unlock()

	return dto, nil
}

// GuessBee submits a spelling guess in the active Spelling Bee session.
func (t *TrainerService) GuessBee(ctx context.Context, guess string) (*BeeSessionDTO, error) {
	ctx, cancel := t.hub.BeginOperation(ctx, "bee:guess", 30*time.Second)
	defer cancel()

	t.hub.beeMu.Lock()
	cur := t.hub.beeCurrent
	listID := t.hub.beeListID
	actID := t.hub.beeActID
	shownAt := t.hub.beeShownAt
	t.hub.beeMu.Unlock()

	if cur == nil || listID <= 0 {
		return nil, fmt.Errorf("vocab: no active Spelling Bee session")
	}

	rt := time.Since(shownAt).Milliseconds()
	resp, err := t.hub.Client().Bee.Guess(ctx, listID, cur.Secret, guess, rt, actID)
	if err != nil {
		return nil, err
	}

	t.hub.beeMu.Lock()
	cur.Secret = resp.Secret
	if resp.Score != nil {
		cur.Score = resp.Score
	}
	if resp.PlayerData != nil {
		cur.PlayerData = resp.PlayerData
	}
	cur.Guesses = resp.Guesses
	t.hub.beeShownAt = time.Now()
	if resp.Correct {
		t.hub.beePending = resp.NextData
		if t.hub.beePending != nil && t.hub.beePending.Secret == "" {
			t.hub.beePending.Secret = resp.Secret
		}
	}
	dto := t.buildBeeSessionLocked(resp.Correct, resp.Correct, resp.Distance, resp.Word)
	t.hub.beeMu.Unlock()

	return dto, nil
}

// SurrenderBee gives up on the current Spelling Bee word (available after 3 incorrect guesses).
func (t *TrainerService) SurrenderBee(ctx context.Context) (*BeeSessionDTO, error) {
	ctx, cancel := t.hub.BeginOperation(ctx, "bee:surrender", 30*time.Second)
	defer cancel()

	t.hub.beeMu.Lock()
	cur := t.hub.beeCurrent
	listID := t.hub.beeListID
	actID := t.hub.beeActID
	shownAt := t.hub.beeShownAt
	t.hub.beeMu.Unlock()

	if cur == nil || listID <= 0 {
		return nil, fmt.Errorf("vocab: no active Spelling Bee session")
	}

	rt := time.Since(shownAt).Milliseconds()
	resp, err := t.hub.Client().Bee.Surrender(ctx, listID, cur.Secret, rt, actID)
	if err != nil {
		return nil, err
	}

	t.hub.beeMu.Lock()
	cur.Secret = resp.Secret
	if resp.Score != nil {
		cur.Score = resp.Score
	}
	t.hub.beePending = resp.NextData
	if t.hub.beePending != nil && t.hub.beePending.Secret == "" {
		t.hub.beePending.Secret = resp.Secret
	}
	dto := t.buildBeeSessionLocked(true, false, 0, resp.Word)
	t.hub.beeMu.Unlock()

	return dto, nil
}

// NextBeeWord advances to the next word in the Spelling Bee after completing or surrendering the current word.
func (t *TrainerService) NextBeeWord() (*BeeSessionDTO, error) {
	t.hub.beeMu.Lock()
	defer t.hub.beeMu.Unlock()
	if t.hub.beePending == nil {
		return t.buildBeeSessionLocked(true, false, 0, ""), nil
	}
	next := t.hub.beePending
	if next.Score == nil && t.hub.beeCurrent != nil {
		next.Score = t.hub.beeCurrent.Score
	}
	if next.Secret == "" && t.hub.beeCurrent != nil {
		next.Secret = t.hub.beeCurrent.Secret
	}
	t.hub.beeCurrent = next
	t.hub.beePending = nil
	t.hub.beeShownAt = time.Now()
	return t.buildBeeSessionLocked(false, false, 0, ""), nil
}

// ResetBee resets Spelling Bee progress for `listID`.
func (t *TrainerService) ResetBee(ctx context.Context, listID int64) error {
	ctx, cancel := t.hub.BeginOperation(ctx, "bee:reset", 20*time.Second)
	defer cancel()
	return t.hub.Client().Bee.Reset(ctx, listID)
}

func (t *TrainerService) buildBeeSessionLocked(completed, correct bool, distance int, word string) *BeeSessionDTO {
	cur := t.hub.beeCurrent
	if cur == nil {
		return &BeeSessionDTO{Active: false}
	}
	defs := make([]BeeDefDTO, len(cur.Defs))
	for i, d := range cur.Defs {
		defs[i] = BeeDefDTO{
			POS:  d.POS,
			Text: strings.Join(d.Def, "______"),
		}
	}
	return &BeeSessionDTO{
		Active:           true,
		ListID:           t.hub.beeListID,
		ActivityID:       t.hub.beeActID,
		AudioURL:         cur.AudioURL(vocab.DefaultAudioBaseURL),
		FormattedExample: cur.FormattedExample("______"),
		Definitions:      defs,
		Score:            cur.Score,
		Guesses:          cur.Guesses,
		CanSurrender:     cur.Guesses >= 3 && !completed,
		CompletedWord:    completed,
		Correct:          correct,
		Distance:         distance,
		RevealedWord:     word,
		HasNextWord:      t.hub.beePending != nil || (cur.Score != nil && cur.Score.Remaining > 0),
		PlayerData:       cur.PlayerData,
	}
}

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
func (t *TrainerService) GetQuizResults(ctx context.Context, activityID string) ([]htmlparse.QuizQuestionResult, error) {
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

// DictionaryService exposes dictionary lookup, debounced autocomplete, corpus examples,
// audio resolution, and search history to the Wails frontend.
type DictionaryService struct {
	hub *Hub
}

// Lookup fetches the full dictionary entry, user mastery progress, and corpus usage examples for `word`.
// Previous in-flight Lookup calls are automatically canceled.
func (d *DictionaryService) Lookup(ctx context.Context, word string) (*WordDetailDTO, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "dict:lookup", 25*time.Second)
	defer cancel()

	word = strings.TrimSpace(word)
	if word == "" {
		return nil, fmt.Errorf("vocab: word is required")
	}

	client := d.hub.Client()
	entry, err := client.Dictionary.Lookup(ctx, word)
	if err != nil {
		return nil, err
	}

	d.hub.recordSearchWord(entry.Word)

	var (
		wg       sync.WaitGroup
		progress *vocab.WordProgressDetail
		examples []CorpusSentenceDTO
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		if wp, pErr := client.Progress.GetWordProgress(ctx, entry.Word, true); pErr == nil {
			progress = wp
		}
	}()
	go func() {
		defer wg.Done()
		if ex, eErr := client.Dictionary.Examples(ctx, entry.Word, vocab.CorpusDomainAll, 0, 8); eErr == nil && ex != nil {
			examples = mapCorpusSentences(ex.Sentences)
		}
	}()
	wg.Wait()

	return &WordDetailDTO{
		Entry:    entry,
		Progress: progress,
		Examples: examples,
	}, nil
}

// Autocomplete returns word suggestions for `query`, canceling any obsolete in-flight autocomplete request.
func (d *DictionaryService) Autocomplete(ctx context.Context, query string, maxResults int) ([]vocab.AutocompleteSuggestion, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "dict:autocomplete", 10*time.Second)
	defer cancel()

	return d.hub.Client().Dictionary.Autocomplete(ctx, query, maxResults)
}

// GetExamples fetches real-world usage sentences ("Word in the Wild") with optional domain filtering
// ("" All, "F" Fiction, "A" Arts, "N" News, "B" Business, "S" Sports, "M" Science/Med, "T" Tech).
func (d *DictionaryService) GetExamples(ctx context.Context, word, domain string, offset, limit int) ([]CorpusSentenceDTO, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "dict:examples", 20*time.Second)
	defer cancel()

	resp, err := d.hub.Client().Dictionary.Examples(ctx, word, vocab.CorpusDomain(strings.TrimSpace(domain)), offset, limit)
	if err != nil {
		return nil, err
	}
	return mapCorpusSentences(resp.Sentences), nil
}

// GetRandomWord fetches a random word from Vocabulary.com's dictionary.
func (d *DictionaryService) GetRandomWord(ctx context.Context) (*WordDetailDTO, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "dict:random", 25*time.Second)
	defer cancel()

	entry, err := d.hub.Client().Dictionary.RandomWord(ctx)
	if err != nil {
		return nil, err
	}
	d.hub.recordSearchWord(entry.Word)
	return &WordDetailDTO{Entry: entry}, nil
}

// ResolveAudioURL resolves the MP3 pronunciation stream URL for `word`.
func (d *DictionaryService) ResolveAudioURL(ctx context.Context, word string) (string, error) {
	return d.hub.Client().Dictionary.ResolveWordAudioURL(ctx, word, 0)
}

// GetSearchHistory returns the user's recent dictionary lookups.
func (d *DictionaryService) GetSearchHistory() []string {
	d.hub.mu.RLock()
	defer d.hub.mu.RUnlock()
	out := make([]string, len(d.hub.prefs.SearchHistory))
	copy(out, d.hub.prefs.SearchHistory)
	return out
}

// ClearSearchHistory clears the saved dictionary lookup history.
func (d *DictionaryService) ClearSearchHistory() error {
	d.hub.mu.Lock()
	defer d.hub.mu.Unlock()
	d.hub.prefs.SearchHistory = []string{}
	return d.hub.savePreferencesLocked()
}

func mapCorpusSentences(sentences []vocab.CorpusSentence) []CorpusSentenceDTO {
	out := make([]CorpusSentenceDTO, 0, len(sentences))
	for _, s := range sentences {
		dto := CorpusSentenceDTO{
			Sentence:  s.Sentence,
			Offsets:   s.Offsets,
			CorpusID:  s.CorpusID,
			SourceURL: s.SourceURL(),
		}
		if s.Volume != nil {
			dto.SourceTitle = s.Volume.Title
			dto.Author = s.Volume.Author
			if s.Volume.Corpus != nil {
				dto.SourceName = s.Volume.Corpus.Name
			}
		}
		out = append(out, dto)
	}
	return out
}

// ListsService exposes vocabulary list browsing, search, flashcards, creation, editing,
// and InstaList text extraction to the Wails frontend.
type ListsService struct {
	hub *Hub
}

// GetUserLists fetches the authenticated user's lists (`display` = `"created"` or `""`).
func (l *ListsService) GetUserLists(ctx context.Context, display string) ([]vocab.ProfileListSummary, error) {
	ctx, cancel := l.hub.BeginOperation(ctx, "lists:user", 20*time.Second)
	defer cancel()
	return l.hub.Client().Lists.ListUserLists(ctx, display, "activitydate", 100)
}

// SearchLists searches public vocabulary lists (`GET /lists/smart-search.json`).
func (l *ListsService) SearchLists(ctx context.Context, query string, skip, limit int) (*vocab.SmartSearchResponse, error) {
	ctx, cancel := l.hub.BeginOperation(ctx, "lists:search", 20*time.Second)
	defer cancel()
	return l.hub.Client().Lists.Search(ctx, query, skip, limit, false)
}

// GetListDetail loads a full vocabulary list by ID (`POST /lists/load.json`).
func (l *ListsService) GetListDetail(ctx context.Context, listID int64) (*vocab.VocabularyList, error) {
	ctx, cancel := l.hub.BeginOperation(ctx, "lists:detail", 20*time.Second)
	defer cancel()
	return l.hub.Client().Lists.Get(ctx, listID)
}

// GetFlashcards fetches the rich flashcard representation of a list (`GET /1.0/lists/{listId}`).
func (l *ListsService) GetFlashcards(ctx context.Context, listID int64) (*vocab.FlashcardListResponse, error) {
	ctx, cancel := l.hub.BeginOperation(ctx, "lists:flashcards", 20*time.Second)
	defer cancel()
	return l.hub.Client().Lists.GetFlashcardList(ctx, listID)
}

// CreateList creates a new vocabulary list and returns its wordlist ID.
func (l *ListsService) CreateList(ctx context.Context, name, description string, shared bool, words []vocab.CreateListWordInput) (int64, error) {
	return l.hub.Client().Lists.Create(ctx, vocab.CreateListRequest{
		Name:        name,
		Description: description,
		Action:      "create",
		Shared:      shared,
		Words:       words,
	})
}

// AddWordsToList appends words to an existing vocabulary list.
func (l *ListsService) AddWordsToList(ctx context.Context, listID int64, words []vocab.CreateListWordInput) (int64, error) {
	return l.hub.Client().Lists.AddWords(ctx, listID, words)
}

// DeleteList permanently deletes a user's vocabulary list.
func (l *ListsService) DeleteList(ctx context.Context, listID int64) error {
	return l.hub.Client().Lists.Delete(ctx, listID)
}

// SetFavorite bookmarks or unbookmarks a vocabulary list.
func (l *ListsService) SetFavorite(ctx context.Context, listID int64, favorite bool) error {
	return l.hub.Client().Lists.SetFavorite(ctx, listID, favorite)
}

// StopLearningList removes a list from the user's active learning program.
func (l *ListsService) StopLearningList(ctx context.Context, listID int64) error {
	return l.hub.Client().Lists.StopLearning(ctx, listID)
}

// ResetListPractice resets the user's Practice session on a list.
func (l *ListsService) ResetListPractice(ctx context.Context, listID int64) error {
	return l.hub.Client().Lists.ResetPractice(ctx, listID)
}

// ValidateWords checks whether words exist in the dictionary and are learnable.
func (l *ListsService) ValidateWords(ctx context.Context, words []string) ([]vocab.ValidatedWord, error) {
	validated, _, err := l.hub.Client().Lists.ValidateWords(ctx, words)
	return validated, err
}

// GrabVocabulary extracts learnable vocabulary words from pasted text ("InstaList").
func (l *ListsService) GrabVocabulary(ctx context.Context, rawText string) (*vocab.VocabGrabberResponse, error) {
	return l.hub.Client().Lists.GrabVocabulary(ctx, rawText)
}

// GetWordSenses returns selectable learnable senses for a word when editing a list.
func (l *ListsService) GetWordSenses(ctx context.Context, word string) ([]vocab.LearnableSense, error) {
	return l.hub.Client().Lists.GetWordSenses(ctx, word)
}

// DiscoveryService exposes Word of the Day, WOD archives, Curated Units/Collections,
// and Advanced WordFinder search to the Wails frontend.
type DiscoveryService struct {
	hub *Hub
}

// GetWordOfTheDay fetches the Word of the Day for `date` (`YYYY-MM-DD`, or `""` for today).
func (d *DiscoveryService) GetWordOfTheDay(ctx context.Context, date string) (*vocab.WordOfTheDayEntry, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "discover:wod", 20*time.Second)
	defer cancel()
	return d.hub.Client().Dictionary.WordOfTheDay(ctx, date)
}

// GetWordOfTheDayArchive fetches the monthly Word of the Day archive (`YYYYMM`, e.g. `202610`).
func (d *DiscoveryService) GetWordOfTheDayArchive(ctx context.Context, startYearMonth, numMonths int) (*vocab.WODArchiveResponse, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "discover:wod-archive", 20*time.Second)
	defer cancel()
	return d.hub.Client().Dictionary.WordOfTheDayArchive(ctx, startYearMonth, numMonths)
}

// GetRootCategories returns Vocabulary.com's 9 curated root collection categories.
func (d *DiscoveryService) GetRootCategories() []CategoryCardDTO {
	return []CategoryCardDTO{
		{UnitID: vocab.UnitCategoryLiterature, Tag: "LIT", Title: "Literature", Description: "Classic novels, plays, poetry, and short stories."},
		{UnitID: vocab.UnitCategoryNonFiction, Tag: "NON", Title: "Non-Fiction", Description: "Memoirs, journalism, essays, and biographies."},
		{UnitID: vocab.UnitCategoryTestPrep, Tag: "TST", Title: "Test Prep", Description: "Essential vocabulary for SAT, ACT, GRE, and TOEFL."},
		{UnitID: vocab.UnitCategoryTextbooks, Tag: "TXT", Title: "Textbooks & Curricula", Description: "Grade-aligned ELA and academic curricula."},
		{UnitID: vocab.UnitCategoryCurrentEvents, Tag: "NWS", Title: "Current Events", Description: "Words drawn from contemporary news and culture."},
		{UnitID: vocab.UnitCategoryRootsAndAffixes, Tag: "ROOTS", Title: "Morphology & Roots", Description: "Greek and Latin roots, prefixes, and suffixes."},
		{UnitID: vocab.UnitCategoryHistoricalDocuments, Tag: "HIST", Title: "Historical Documents", Description: "Founding charters, treaties, and landmark texts."},
		{UnitID: vocab.UnitCategorySpeeches, Tag: "SPH", Title: "Speeches", Description: "Iconic rhetoric and addresses from world history."},
		{UnitID: vocab.UnitCategoryJustForFun, Tag: "FUN", Title: "Just for Fun", Description: "Witty, unusual, and thematic word collections."},
	}
}

// GetUnit fetches a single curated Unit/Collection and its sub-lists (`GET /1.0/units/{unitId}`).
func (d *DiscoveryService) GetUnit(ctx context.Context, unitID string) (*vocab.UnitsResponse, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "discover:unit", 20*time.Second)
	defer cancel()
	return d.hub.Client().Lists.GetUnit(ctx, unitID)
}

// SearchUnits searches or browses curated Units/Collections (`GET /1.0/units`).
func (d *DiscoveryService) SearchUnits(ctx context.Context, query, categoryUnitID string, skip, limit int) (*vocab.UnitsResponse, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "discover:units", 20*time.Second)
	defer cancel()
	if limit <= 0 {
		limit = 12
	}
	return d.hub.Client().Lists.SearchUnits(ctx, vocab.SearchUnitsOptions{
		Query:            strings.TrimSpace(query),
		TaggedWithUnitID: strings.TrimSpace(categoryUnitID),
		IncludeRelated:   true,
		Skip:             skip,
		Limit:            limit,
	})
}

// WordFinder searches words by pattern (`startingwith`, `endingwith`, `containinginorder`, `including`) and length.
func (d *DiscoveryService) WordFinder(ctx context.Context, searchText, searchLocation string, numLetters, maxResults int) (*vocab.WordFinderResponse, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "discover:wordfinder", 20*time.Second)
	defer cancel()
	return d.hub.Client().Dictionary.WordFinder(ctx, vocab.WordFinderOptions{
		SearchText:     strings.TrimSpace(searchText),
		SearchLocation: strings.TrimSpace(searchLocation),
		NumLetters:     numLetters,
		MaxResults:     maxResults,
	})
}

// ProgressService exposes the Home dashboard aggregator, learning/trouble/mastered word lists,
// priority controls, effort metrics, activities, achievements, and leaderboards.
type ProgressService struct {
	hub *Hub
}

// GetHomeDashboard aggregates real data for the Home screen, gracefully degrading for unauthenticated users.
func (p *ProgressService) GetHomeDashboard(ctx context.Context) (*HomeDashboardDTO, error) {
	ctx, cancel := p.hub.BeginOperation(ctx, "progress:dashboard", 25*time.Second)
	defer cancel()

	client := p.hub.Client()
	out := &HomeDashboardDTO{
		SearchHistory: p.hub.Dictionary.GetSearchHistory(),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// 1. Check auth status
	me, authErr := client.Auth.Me(ctx)
	if authErr == nil && me != nil {
		out.Account = *p.hub.Auth.mapLoginData(me)
	} else {
		out.Account = *p.hub.Auth.unauthenticatedStatus()
	}

	// 2. Fetch Word of the Day (works unauthenticated)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if wod, err := client.Dictionary.WordOfTheDay(ctx, ""); err == nil {
			mu.Lock()
			out.WordOfTheDay = wod
			mu.Unlock()
		}
	}()

	// 3. If authenticated, fetch real learner effort, activities, learning/trouble words, and recent lists in parallel
	if out.Account.Authenticated {
		now := time.Now()
		startDate := now.AddDate(0, 0, -7).Format("2006-01-02")
		endDate := now.Format("2006-01-02")

		wg.Add(5)
		go func() {
			defer wg.Done()
			if eff, err := client.Progress.GetEffort(ctx, startDate, endDate); err == nil {
				mu.Lock()
				out.Effort = eff
				mu.Unlock()
			}
		}()
		go func() {
			defer wg.Done()
			complete := false
			if acts, err := client.Progress.GetActivities(ctx, vocab.ActivitiesQuery{Complete: &complete, Limit: 5, Include: "wordlists"}); err == nil && acts != nil {
				mu.Lock()
				out.InProgressActivities = acts.Activities
				mu.Unlock()
			}
		}()
		go func() {
			defer wg.Done()
			if words, err := client.Progress.ListLearningWords(ctx, vocab.WordFilterLearning, 0, "", ""); err == nil {
				if len(words) > 8 {
					words = words[:8]
				}
				mu.Lock()
				out.LearningWords = words
				mu.Unlock()
			}
		}()
		go func() {
			defer wg.Done()
			if trouble, err := client.Progress.ListLearningWords(ctx, vocab.WordFilterTrouble, 0, "", ""); err == nil {
				if len(trouble) > 8 {
					trouble = trouble[:8]
				}
				mu.Lock()
				out.TroubleWords = trouble
				mu.Unlock()
			}
		}()
		go func() {
			defer wg.Done()
			if lists, err := client.Lists.ListUserLists(ctx, "", "activitydate", 6); err == nil {
				mu.Lock()
				out.RecentLists = lists
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return out, nil
}

// GetWordProgress fetches sense and mastery progress for a single word.
func (p *ProgressService) GetWordProgress(ctx context.Context, word string) (*vocab.WordProgressDetail, error) {
	return p.hub.Client().Progress.GetWordProgress(ctx, word, true)
}

// SetWordPriority updates a word's learning priority (-1 Low, 0 Normal, 1 High).
func (p *ProgressService) SetWordPriority(ctx context.Context, word string, priority int) (*vocab.WordProgressDetail, error) {
	return p.hub.Client().Progress.SetWordPriority(ctx, word, vocab.WordPriority(priority))
}

// StartLearningWord adds `word` to the user's active learning program.
func (p *ProgressService) StartLearningWord(ctx context.Context, word string) error {
	return p.hub.Client().Progress.StartLearningWord(ctx, word)
}

// StopLearningWord removes `word` from the user's active learning program.
func (p *ProgressService) StopLearningWord(ctx context.Context, word string) error {
	return p.hub.Client().Progress.StopLearningWord(ctx, word)
}

// ListLearningWords fetches paginated words from `/account/progress/words.json`
// (`filter` = `"learning"`, `"trouble"`, or `"mastered"`).
func (p *ProgressService) ListLearningWords(ctx context.Context, filter string, skip int, sortBy, sortDir string) ([]vocab.AccountProgressWord, error) {
	ctx, cancel := p.hub.BeginOperation(ctx, "progress:words", 20*time.Second)
	defer cancel()
	return p.hub.Client().Progress.ListLearningWords(ctx, vocab.WordFilter(strings.TrimSpace(filter)), skip, sortBy, sortDir)
}

// GetListProgress fetches the user's mastery status on a vocabulary list.
func (p *ProgressService) GetListProgress(ctx context.Context, listID int64) (*vocab.ListProgressStatus, error) {
	return p.hub.Client().Progress.GetListProgress(ctx, listID)
}

// GetEffort fetches learner effort stats between `startDate` and `endDate` (`YYYY-MM-DD`).
func (p *ProgressService) GetEffort(ctx context.Context, startDate, endDate string) (*vocab.EffortStats, error) {
	return p.hub.Client().Progress.GetEffort(ctx, startDate, endDate)
}

// GetActivities fetches the user's activities (`complete` = `"true"`, `"false"`, or `""` for all).
func (p *ProgressService) GetActivities(ctx context.Context, completeFilter string, limit int, nextPageURI string) (*vocab.ActivitiesResponse, error) {
	ctx, cancel := p.hub.BeginOperation(ctx, "progress:activities", 20*time.Second)
	defer cancel()
	if strings.TrimSpace(nextPageURI) != "" {
		return p.hub.Client().Progress.GetActivitiesByURI(ctx, nextPageURI)
	}
	var compPtr *bool
	switch strings.ToLower(strings.TrimSpace(completeFilter)) {
	case "true":
		t := true
		compPtr = &t
	case "false":
		f := false
		compPtr = &f
	}
	if limit <= 0 {
		limit = 20
	}
	return p.hub.Client().Progress.GetActivities(ctx, vocab.ActivitiesQuery{
		Complete: compPtr,
		Limit:    limit,
		Include:  "wordlists",
	})
}

// GetAchievements fetches the user's earned achievements and milestones.
func (p *ProgressService) GetAchievements(ctx context.Context) ([]vocab.AchievementSummary, error) {
	ctx, cancel := p.hub.BeginOperation(ctx, "progress:achievements", 20*time.Second)
	defer cancel()
	return p.hub.Client().Progress.GetAchievements(ctx, 500)
}

// GetLeaderboard fetches an individual leaderboard (`metric` = `"points"` or `"mastered"`,
// `period` = `""` (all-time), `"daily"`, `"weekly"`, `"monthly"`, or `"season"`, `yyyymmdd` = `"20261031"`).
func (p *ProgressService) GetLeaderboard(ctx context.Context, metric, period, yyyymmdd string, top int) (*vocab.LeaderboardResponse, error) {
	ctx, cancel := p.hub.BeginOperation(ctx, "progress:leaderboard", 20*time.Second)
	defer cancel()
	return p.hub.Client().Progress.GetIndividualLeaderboard(ctx, metric, period, yyyymmdd, top, "")
}

// SettingsService manages non-sensitive desktop preferences and session store backend configuration.
type SettingsService struct {
	hub *Hub
}

// GetPreferences returns the current AppPreferences.
func (s *SettingsService) GetPreferences() AppPreferences {
	s.hub.mu.RLock()
	defer s.hub.mu.RUnlock()
	cp := s.hub.prefs
	cp.AppVersion = AppVersion
	if s.hub.prefs.SearchHistory != nil {
		cp.SearchHistory = make([]string, len(s.hub.prefs.SearchHistory))
		copy(cp.SearchHistory, s.hub.prefs.SearchHistory)
	}
	return cp
}

// UpdatePreferences updates theme, pronunciation accent, default view, and optionally session store mode.
func (s *SettingsService) UpdatePreferences(ctx context.Context, updated AppPreferences) (*AppPreferences, error) {
	if updated.SessionStoreMode != "" && updated.SessionStoreMode.IsValid() {
		if err := s.hub.SwitchSessionStoreMode(ctx, updated.SessionStoreMode); err != nil {
			return nil, err
		}
	}

	s.hub.mu.Lock()
	switch updated.Theme {
	case "light", "dark", "system":
		s.hub.prefs.Theme = updated.Theme
	}
	switch strings.ToUpper(updated.PronunciationAccent) {
	case "US", "UK":
		s.hub.prefs.PronunciationAccent = strings.ToUpper(updated.PronunciationAccent)
	}
	switch updated.DefaultView {
	case "home", "practice", "dictionary", "lists", "discover", "progress":
		s.hub.prefs.DefaultView = updated.DefaultView
	}
	s.hub.prefs.AppVersion = AppVersion
	err := s.hub.savePreferencesLocked()
	prefsCopy := s.hub.prefs
	s.hub.mu.Unlock()

	if err != nil {
		return nil, err
	}
	s.hub.Emit("settings:changed", prefsCopy)
	return &prefsCopy, nil
}
