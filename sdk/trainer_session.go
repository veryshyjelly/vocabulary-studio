package vocab

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// TrainerSessionState represents the explicit lifecycle state of a TrainerSession.
type TrainerSessionState string

const (
	// SessionStateIdle indicates the session has not yet loaded an active question.
	SessionStateIdle TrainerSessionState = "idle"
	// SessionStateActiveQuestion indicates a question is active and accepting answers.
	SessionStateActiveQuestion TrainerSessionState = "active_question"
	// SessionStateQuestionCompleted indicates the current question has been answered correctly
	// and the session is waiting for Next() to advance to the next slide.
	SessionStateQuestionCompleted TrainerSessionState = "question_completed"
	// SessionStateTrialOver indicates the unauthenticated/free question limit has been reached.
	SessionStateTrialOver TrainerSessionState = "trial_over"
)

// TrainerSession is a high-level stateful gameplay helper that tracks `secret`, `shownAt`,
// explicit session state (`TrainerSessionState`), current question choices, and local multi-try
// evaluation (since Vocabulary.com only calls `saveanswer.json` on the 1st try and evaluates
// 2nd/3rd/4th tries locally against `correct_choice` / `accepted_answers`).
//
// TrainerSession serializes turn operations and is safe for concurrent use.
type TrainerSession struct {
	opMu             sync.Mutex // Serializes state-mutating operations (ChooseIndex, Spell, Hint, Next)
	mu               sync.RWMutex
	service          *TrainerService
	version          ClientVersion
	actType          ActivityType
	wordlistID       int64
	activityID       string
	state            TrainerSessionState
	secret           string
	shownAt          time.Time
	tries            int
	currentQ         *ParsedQuestion
	firstTryOutcome  *AnswerOutcome
	eliminatedNonces map[string]bool
	playerData       *ChallengePlayerData
	round            *ChallengeRound
	game             *ChallengeGame
}

// NewSession starts or resumes a stateful TrainerSession.
func (t *TrainerService) NewSession(ctx context.Context, req StartChallengeRequest) (*TrainerSession, *ChallengeStateResponse, error) {
	state, err := t.Start(ctx, req)
	if err != nil && state == nil {
		return nil, nil, err
	}
	ver := req.Version
	if ver == 0 {
		if req.ActivityType == ActivityListPractice {
			ver = ClientVersionClassicWeb
		} else {
			ver = ClientVersionVocabTrainer
		}
	}
	act := req.ActivityType
	if act == "" {
		act = ActivityVocabTrainer
	}
	s := &TrainerSession{
		service:          t,
		version:          ver,
		actType:          act,
		wordlistID:       req.WordlistID,
		activityID:       req.ActivityID,
		state:            SessionStateIdle,
		eliminatedNonces: make(map[string]bool),
	}
	s.applyStateLocked(state)
	return s, state, err
}

func (s *TrainerSession) applyStateLocked(state *ChallengeStateResponse) {
	if state == nil {
		return
	}
	if state.Secret != "" {
		s.secret = state.Secret
	}
	if state.PlayerData != nil {
		s.playerData = state.PlayerData
	}
	if state.Round != nil {
		s.round = state.Round
	}
	if state.Game != nil {
		s.game = state.Game
	}
	s.currentQ = state.ParsedQuestion.Clone()
	s.shownAt = s.service.client.clock.Now()
	s.tries = 0
	s.firstTryOutcome = nil
	s.eliminatedNonces = make(map[string]bool)
	switch {
	case state.Action == "trialover":
		s.state = SessionStateTrialOver
	case s.currentQ != nil:
		s.state = SessionStateActiveQuestion
	default:
		s.state = SessionStateIdle
	}
}

// State returns the current TrainerSessionState of the session.
func (s *TrainerSession) State() TrainerSessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// CurrentQuestion returns a copy of the active ParsedQuestion in the session.
func (s *TrainerSession) CurrentQuestion() *ParsedQuestion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentQ.Clone()
}

// Secret returns the current turn secret token.
func (s *TrainerSession) Secret() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.secret
}

// Tries returns the number of answer attempts made on the current question so far.
func (s *TrainerSession) Tries() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tries
}

// IsNonceEliminated reports whether a choice nonce was eliminated by a 50/50 hint on the current question.
func (s *TrainerSession) IsNonceEliminated(nonce string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.eliminatedNonces[nonce]
}

// Next fetches the next question in the session and resets the per-question attempt counter.
func (s *TrainerSession) Next(ctx context.Context) (*ChallengeStateResponse, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.RLock()
	if s.state == SessionStateTrialOver {
		s.mu.RUnlock()
		return nil, ErrTrialOver
	}
	secret := s.secret
	ver := s.version
	s.mu.RUnlock()

	state, err := s.service.NextQuestion(ctx, secret, ver)
	if err != nil && state == nil {
		return nil, err
	}
	s.mu.Lock()
	s.applyStateLocked(state)
	s.mu.Unlock()
	return state, err
}

// ChooseIndex submits a multiple-choice selection by 0-based index (0..3).
//
// On the 1st try (`Attempt == 1`), it submits the choice's nonce to `/challenge/saveanswer.json`
// (`EvaluatedLocally == false`).
// On subsequent tries (`Attempt > 1`), it evaluates locally against the 1st try's `CorrectChoice`
// (`EvaluatedLocally == true`, `Points == 0`) just like Vocabulary.com's frontend JS.
func (s *TrainerSession) ChooseIndex(ctx context.Context, choiceIndex int) (*AnswerOutcome, bool, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.RLock()
	state := s.state
	q := s.currentQ
	shownAt := s.shownAt
	secret := s.secret
	ver := s.version
	currentTries := s.tries
	cachedOutcome := s.firstTryOutcome
	s.mu.RUnlock()

	switch state {
	case SessionStateTrialOver:
		return nil, false, ErrTrialOver
	case SessionStateQuestionCompleted:
		return nil, false, fmt.Errorf("%w: question is already completed; call Next to load the next question", ErrInvalidStateTransition)
	case SessionStateIdle:
		return nil, false, fmt.Errorf("%w: no active question in session", ErrInvalidStateTransition)
	}
	if q == nil {
		return nil, false, fmt.Errorf("%w: no active question in session", ErrInvalidStateTransition)
	}
	if q.IsSpelling {
		return nil, false, fmt.Errorf("%w: cannot call ChooseIndex on spelling question (type %s); use Spell", ErrInvalidStateTransition, q.Type)
	}
	if choiceIndex < 0 || choiceIndex >= len(q.Choices) {
		return nil, false, fmt.Errorf("vocab: choiceIndex %d out of range (0..%d)", choiceIndex, len(q.Choices)-1)
	}

	nonce := q.Choices[choiceIndex].Nonce
	if strings.TrimSpace(nonce) == "" {
		return nil, false, fmt.Errorf("vocab: choice %d has an empty nonce", choiceIndex)
	}

	if currentTries == 0 {
		rt, err := s.service.client.pacingGuard.EnsureMinResponseTime(ctx, shownAt, MinChallengeFirstTryMS)
		if err != nil {
			return nil, false, err
		}
		resp, err := s.service.SubmitAnswer(ctx, secret, nonce, rt, ver)
		if err != nil {
			return nil, false, err
		}
		if resp.Answer == nil {
			return nil, false, fmt.Errorf("%w: /challenge/saveanswer.json returned nil answer object", ErrUnexpectedResponse)
		}

		isCorrect := resp.Answer.Correct || resp.Answer.CorrectChoice == choiceIndex
		now := s.service.client.clock.Now()

		s.mu.Lock()
		s.tries = 1
		if resp.Secret != "" {
			s.secret = resp.Secret
		}
		if resp.PlayerData != nil {
			s.playerData = resp.PlayerData
		}
		if resp.Round != nil {
			s.round = resp.Round
		}
		if resp.Game != nil {
			s.game = resp.Game
		}
		s.firstTryOutcome = resp.Answer.Clone()
		s.shownAt = now
		if isCorrect {
			s.state = SessionStateQuestionCompleted
		}
		s.mu.Unlock()

		out := resp.Answer.Clone()
		out.Attempt = 1
		out.EvaluatedLocally = false
		out.UserChoice = choiceIndex
		out.Correct = isCorrect
		return out, isCorrect, nil
	}

	// Subsequent try: enforce >= 475ms retry pacing and evaluate locally against firstTryOutcome.
	if cachedOutcome == nil {
		return nil, false, fmt.Errorf("%w: missing first-try outcome for local evaluation", ErrInvalidStateTransition)
	}
	if _, err := s.service.client.pacingGuard.EnsureMinResponseTime(ctx, shownAt, MinChallengeRetryMS); err != nil {
		return nil, false, err
	}

	isCorrect := cachedOutcome.CorrectChoice == choiceIndex
	now := s.service.client.clock.Now()

	s.mu.Lock()
	s.tries++
	tryNum := s.tries
	s.shownAt = now
	if isCorrect {
		s.state = SessionStateQuestionCompleted
	}
	s.mu.Unlock()

	localOut := cachedOutcome.Clone()
	localOut.Attempt = tryNum
	localOut.EvaluatedLocally = true
	localOut.Correct = isCorrect
	localOut.UserChoice = choiceIndex
	localOut.Points = 0
	localOut.Bonus = 0
	localOut.Achievements = nil
	return localOut, isCorrect, nil
}

// Spell submits a spelling guess (`type == "T"`).
//
// On the 1st try (`Attempt == 1`), it calls `/challenge/saveanswer.json` (`EvaluatedLocally == false`).
// On subsequent tries (`Attempt > 1`), it evaluates locally against `AcceptedAnswers` (`EvaluatedLocally == true`).
func (s *TrainerSession) Spell(ctx context.Context, rawGuess string) (*AnswerOutcome, bool, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	normalized := NormalizeSpellingGuess(rawGuess)
	if normalized == "" {
		return nil, false, fmt.Errorf("vocab: spelling guess cannot be empty")
	}

	s.mu.RLock()
	state := s.state
	q := s.currentQ
	shownAt := s.shownAt
	secret := s.secret
	ver := s.version
	currentTries := s.tries
	cachedOutcome := s.firstTryOutcome
	s.mu.RUnlock()

	switch state {
	case SessionStateTrialOver:
		return nil, false, ErrTrialOver
	case SessionStateQuestionCompleted:
		return nil, false, fmt.Errorf("%w: question is already completed; call Next to load the next question", ErrInvalidStateTransition)
	case SessionStateIdle:
		return nil, false, fmt.Errorf("%w: no active question in session", ErrInvalidStateTransition)
	}
	if q == nil {
		return nil, false, fmt.Errorf("%w: no active question in session", ErrInvalidStateTransition)
	}
	if !q.IsSpelling {
		return nil, false, fmt.Errorf("%w: cannot call Spell on multiple-choice question (type %s); use ChooseIndex", ErrInvalidStateTransition, q.Type)
	}

	if currentTries == 0 {
		rt, err := s.service.client.pacingGuard.EnsureMinResponseTime(ctx, shownAt, MinChallengeFirstTryMS)
		if err != nil {
			return nil, false, err
		}
		resp, err := s.service.SubmitAnswer(ctx, secret, normalized, rt, ver)
		if err != nil {
			return nil, false, err
		}
		if resp.Answer == nil {
			return nil, false, fmt.Errorf("%w: /challenge/saveanswer.json returned nil answer object", ErrUnexpectedResponse)
		}

		isCorrect := resp.Answer.Correct
		now := s.service.client.clock.Now()

		s.mu.Lock()
		s.tries = 1
		if resp.Secret != "" {
			s.secret = resp.Secret
		}
		if resp.PlayerData != nil {
			s.playerData = resp.PlayerData
		}
		if resp.Round != nil {
			s.round = resp.Round
		}
		if resp.Game != nil {
			s.game = resp.Game
		}
		s.firstTryOutcome = resp.Answer.Clone()
		s.shownAt = now
		if isCorrect {
			s.state = SessionStateQuestionCompleted
		}
		s.mu.Unlock()

		out := resp.Answer.Clone()
		out.Attempt = 1
		out.EvaluatedLocally = false
		return out, isCorrect, nil
	}

	if cachedOutcome == nil {
		return nil, false, fmt.Errorf("%w: missing first-try outcome for local spelling check", ErrInvalidStateTransition)
	}
	if _, err := s.service.client.pacingGuard.EnsureMinResponseTime(ctx, shownAt, MinChallengeRetryMS); err != nil {
		return nil, false, err
	}

	isCorrect := false
	for _, acc := range cachedOutcome.AcceptedAnswers {
		if NormalizeSpellingGuess(acc) == normalized {
			isCorrect = true
			break
		}
	}
	if !isCorrect && cachedOutcome.Word != "" && NormalizeSpellingGuess(cachedOutcome.Word) == normalized {
		isCorrect = true
	}

	now := s.service.client.clock.Now()
	s.mu.Lock()
	s.tries++
	tryNum := s.tries
	s.shownAt = now
	if isCorrect {
		s.state = SessionStateQuestionCompleted
	}
	s.mu.Unlock()

	localOut := cachedOutcome.Clone()
	localOut.Attempt = tryNum
	localOut.EvaluatedLocally = true
	localOut.Correct = isCorrect
	localOut.Points = 0
	localOut.Bonus = 0
	localOut.Achievements = nil
	return localOut, isCorrect, nil
}

// Hint invokes a lifeline on the active question and updates the session's `secret`.
// In Vocabulary.com's protocol, hints may only be used before the first answer attempt (`Tries() == 0`).
func (s *TrainerSession) Hint(ctx context.Context, hintType HintType) (*HintResponse, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.RLock()
	state := s.state
	tries := s.tries
	secret := s.secret
	ver := s.version
	s.mu.RUnlock()

	if state != SessionStateActiveQuestion {
		return nil, fmt.Errorf("%w: hints can only be used on an active question (current state: %s)", ErrInvalidStateTransition, state)
	}
	if tries > 0 {
		return nil, fmt.Errorf("%w: hints can only be used before the first attempt on a question", ErrInvalidStateTransition)
	}

	resp, err := s.service.UseHint(ctx, secret, hintType, ver)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if resp.Secret != "" {
		s.secret = resp.Secret
	}
	if resp.PlayerData != nil {
		s.playerData = resp.PlayerData
	}
	for _, n := range resp.Nonces {
		if n != "" {
			s.eliminatedNonces[n] = true
		}
	}
	s.mu.Unlock()
	return resp, nil
}
