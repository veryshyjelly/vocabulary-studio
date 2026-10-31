package vocab

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// BeeService implements Vocabulary.com's Spelling Bee endpoints for word lists
// (`/lists/{listid}/bee/start.json`, `guess.json`, `surrender.json`, `reset`).
// Notice we never send the `f` localStorage bot-honeypot parameter.
type BeeService struct {
	client *Client
}

// BeeScore represents the `score` object in Spelling Bee responses.
type BeeScore struct {
	Remaining     int     `json:"remaining"`
	Played        int     `json:"played"`
	Correct       int     `json:"correct"`
	Pct           float64 `json:"pct"`
	Progress      float64 `json:"progress"`
	Elapsed       int64   `json:"elapsed"`
	LongestStreak int     `json:"longestStreak"`
	Surrendered   int     `json:"surrendered"`
}

// BeeDef represents a definition with blanked word segments in Spelling Bee.
type BeeDef struct {
	POS string   `json:"pos"`
	Def []string `json:"def"` // Joined with "______" when displaying
}

// BeeStateResponse represents the JSON returned by `/lists/{listid}/bee/start.json`,
// `guess.json`, and `surrender.json`.
type BeeStateResponse struct {
	Action       string               `json:"action,omitempty"`
	Secret       string               `json:"secret"`
	Score        *BeeScore            `json:"score,omitempty"`
	Audio        []string             `json:"audio,omitempty"`
	Example      []string             `json:"example,omitempty"` // Joined with "______"
	Defs         []BeeDef             `json:"defs,omitempty"`
	PlayerData   *ChallengePlayerData `json:"pdata,omitempty"`
	Achievements []EarnedAchievement  `json:"achievements,omitempty"`

	// Fields present on guess.json / surrender.json responses:
	Guesses  int               `json:"guesses,omitempty"`
	Correct  bool              `json:"correct,omitempty"`
	Distance int               `json:"distance,omitempty"` // Edit distance hint when guesses > 1
	Word     string            `json:"word,omitempty"`     // Revealed target word on correct or surrender
	Subs     []string          `json:"subs,omitempty"`     // Replacement strings for blanks
	NextData *BeeStateResponse `json:"data,omitempty"`     // Next word payload when current word completes
}

// AudioURL returns the full MP3 URL for the current Spelling Bee word.
func (b *BeeStateResponse) AudioURL(audioBaseURL string) string {
	if b == nil || len(b.Audio) == 0 || b.Audio[0] == "" {
		return ""
	}
	if audioBaseURL == "" {
		audioBaseURL = DefaultAudioBaseURL
	}
	return strings.TrimRight(audioBaseURL, "/") + "/" + b.Audio[0] + ".mp3"
}

// FormattedExample joins the blanked example segments with `blankPlaceholder` (e.g. `"______"`).
func (b *BeeStateResponse) FormattedExample(blankPlaceholder string) string {
	if b == nil || len(b.Example) == 0 {
		return ""
	}
	if blankPlaceholder == "" {
		blankPlaceholder = "______"
	}
	return strings.Join(b.Example, blankPlaceholder)
}

// Start begins or resumes a Spelling Bee session for `listID` (`POST /lists/{listid}/bee/start.json`).
func (b *BeeService) Start(ctx context.Context, listID int64, activityID, prevSecret string) (*BeeStateResponse, error) {
	if listID <= 0 {
		return nil, fmt.Errorf("vocab: listID must be > 0")
	}
	form := url.Values{
		"v": {"3"},
	}
	if prevSecret != "" {
		form.Set("secret", prevSecret)
	}
	if activityID != "" {
		form.Set("aid", activityID)
	}
	path := fmt.Sprintf("/lists/%d/bee/start.json", listID)
	var resp BeeStateResponse
	if err := b.client.doWebForm(ctx, path, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Guess submits a spelling guess (`POST /lists/{listid}/bee/guess.json`).
func (b *BeeService) Guess(ctx context.Context, listID int64, secret, guess string, responseTimeMS int64, activityID string) (*BeeStateResponse, error) {
	if listID <= 0 {
		return nil, fmt.Errorf("vocab: listID must be > 0")
	}
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("vocab: secret is required for Bee.Guess")
	}
	normalized := NormalizeSpellingGuess(guess)
	if normalized == "" {
		return nil, fmt.Errorf("vocab: guess is required for Bee.Guess")
	}
	if responseTimeMS < MinBeeSubmitMS {
		responseTimeMS = 1200
	}
	form := url.Values{
		"secret": {secret},
		"v":      {"3"},
		"guess":  {normalized},
		"rt":     {strconv.FormatInt(responseTimeMS, 10)},
	}
	if activityID != "" {
		form.Set("aid", activityID)
	}
	path := fmt.Sprintf("/lists/%d/bee/guess.json", listID)
	var resp BeeStateResponse
	if err := b.client.doWebForm(ctx, path, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Surrender gives up on the current word (`POST /lists/{listid}/bee/surrender.json`, available after 3 incorrect guesses).
func (b *BeeService) Surrender(ctx context.Context, listID int64, secret string, responseTimeMS int64, activityID string) (*BeeStateResponse, error) {
	if listID <= 0 {
		return nil, fmt.Errorf("vocab: listID must be > 0")
	}
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("vocab: secret is required for Bee.Surrender")
	}
	if responseTimeMS < MinBeeSubmitMS {
		responseTimeMS = 1000
	}
	form := url.Values{
		"secret": {secret},
		"v":      {"3"},
		"rt":     {strconv.FormatInt(responseTimeMS, 10)},
	}
	if activityID != "" {
		form.Set("aid", activityID)
	}
	path := fmt.Sprintf("/lists/%d/bee/surrender.json", listID)
	var resp BeeStateResponse
	if err := b.client.doWebForm(ctx, path, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Reset resets the Spelling Bee progress for `listID` (`GET /lists/{listid}/bee/reset`).
func (b *BeeService) Reset(ctx context.Context, listID int64) error {
	if listID <= 0 {
		return fmt.Errorf("vocab: listID must be > 0")
	}
	path := fmt.Sprintf("/lists/%d/bee/reset", listID)
	_, _, err := b.client.doWebGet(ctx, path, nil, "text/html,*/*")
	return err
}
