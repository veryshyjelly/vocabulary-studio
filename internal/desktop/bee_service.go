package desktop

import (
	"context"
	"fmt"
	"strings"
	"time"

	vocab "github.com/veryshyjelly/vocabulary_go/sdk"
)

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
