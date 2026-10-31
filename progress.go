package vocab

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// WordPriority represents the learning priority for a word (`-1` Low, `0` Normal/Auto, `1` High).
type WordPriority int

const (
	PriorityLow    WordPriority = -1
	PriorityNormal WordPriority = 0
	PriorityHigh   WordPriority = 1
)

// IsValid reports whether p is a valid WordPriority (-1, 0, or 1).
func (p WordPriority) IsValid() bool {
	return p == PriorityLow || p == PriorityNormal || p == PriorityHigh
}

// WordFilter identifies a filter tab on `/account/progress/words/{learning|trouble|mastered}`.
type WordFilter string

const (
	WordFilterLearning WordFilter = "learning"
	WordFilterTrouble  WordFilter = "trouble"
	WordFilterMastered WordFilter = "mastered"
)

// IsValid reports whether wf is a valid WordFilter ("learning", "trouble", or "mastered").
func (wf WordFilter) IsValid() bool {
	return wf == WordFilterLearning || wf == WordFilterTrouble || wf == WordFilterMastered
}

// ProgressService provides word mastery lookups, priority management, learner effort dashboards,
// activity/assignment tracking, achievements, and leaderboards.
type ProgressService struct {
	client *Client
}

// WordProgressDetail represents the response from `GET https://api.vocabulary.com/1.0/progress/words/{word}`.
type WordProgressDetail struct {
	Word      string          `json:"word"`
	Learnable bool            `json:"learnable"`
	PKV       int             `json:"pkv,omitempty"` // Estimated % probability user knows the word
	Sense     *FlashcardSense `json:"sense,omitempty"`
	Progress  *struct {
		Progress   float64      `json:"progress"` // 0.0 to 1.0
		Priority   WordPriority `json:"priority"` // -1, 0, 1
		PlayedAt   string       `json:"played_at,omitempty"`
		MasteredAt string       `json:"mastered_at,omitempty"`
	} `json:"progress,omitempty"`
	Lists []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"lists,omitempty"`
	Answers []struct {
		At string `json:"at"`
	} `json:"answers,omitempty"`
}

// ListProgressStatus represents the response from `GET https://api.vocabulary.com/1.0/progress/lists/{listId}`.
type ListProgressStatus struct {
	WordlistID         int64   `json:"wordlist_id,omitempty"`
	LearnableWordCount int     `json:"learnable_word_count"`
	Progress           float64 `json:"progress"`
	Learning           bool    `json:"learning"`
	Starred            bool    `json:"starred"`
	LearningProgress   *struct {
		Progress float64 `json:"progress"`
	} `json:"learning_progress,omitempty"`
}

// AccountProgressWord represents an item from `POST /account/progress/words.json`.
type AccountProgressWord struct {
	Word           string  `json:"word"`
	Definition     string  `json:"definition"`
	Progress       float64 `json:"progress"`
	DateMastered   int64   `json:"dateMastered,omitempty"`
	DateLastPlayed int64   `json:"dateLastPlayed,omitempty"`
}

// EffortStats represents the learner dashboard effort metrics from `GET https://api.vocabulary.com/1.0/me/effort`.
type EffortStats struct {
	WordsMastered      int `json:"words_mastered"`
	Points             int `json:"points"`
	QuestionsAnswered  int `json:"questions_answered"`
	QuestionsIncorrect int `json:"questions_incorrect"`
	TimeMinutes        int `json:"time_minutes"`
}

// ActivityItem represents an activity from `GET https://api.vocabulary.com/1.0/me/activities`.
type ActivityItem struct {
	ID           string   `json:"id"`
	Type         string   `json:"type,omitempty"`
	ActivityType string   `json:"activity_type,omitempty"` // "p", "s", "q", "c"
	WordlistID   int64    `json:"wordlist_id,omitempty"`
	ListID       int64    `json:"list_id,omitempty"`
	Name         string   `json:"name,omitempty"`
	Length       int      `json:"length,omitempty"`
	Progress     float64  `json:"progress"`
	StartedAt    string   `json:"started_at,omitempty"`
	LastPlayedAt string   `json:"last_played_at,omitempty"`
	PlayedAt     string   `json:"played_at,omitempty"`
	DueAt        string   `json:"due_at,omitempty"`
	VisibleAt    string   `json:"visible_at,omitempty"`
	HTMLURL      string   `json:"html_url,omitempty"`
	URL          string   `json:"url,omitempty"`
	Assignments  []string `json:"assignments,omitempty"`
}

// ActivitiesResponse wraps `GET /1.0/me/activities`.
type ActivitiesResponse struct {
	Activities  []ActivityItem `json:"activities"`
	Wordlists   []any          `json:"wordlists,omitempty"`
	NextPageURI string         `json:"next_page_uri,omitempty"`
}

// ActivitiesQuery configures `GET /1.0/me/activities`.
type ActivitiesQuery struct {
	Complete     *bool
	Limit        int
	Include      string // "wordlists" or "assignments,classes"
	WordlistID   int64
	ActivityType string // "p", "s", "q", "c"
	StartedAfter string // ISO8601 timestamp
	EndDate      string // ISO8601 timestamp
}

// AchievementSummary represents an achievement from `GET /1.0/me/achievements`.
type AchievementSummary struct {
	ID            string  `json:"id"`
	Count         int     `json:"count,omitempty"`
	Value         int     `json:"value,omitempty"`
	Milestone     int     `json:"milestone"`
	Progress      float64 `json:"progress"`
	StartAt       string  `json:"start_at,omitempty"`
	FirstEarnedAt string  `json:"first_earned_at,omitempty"`
	LastEarnedAt  string  `json:"last_earned_at,omitempty"`
	Milestone1    string  `json:"milestone1,omitempty"`
	Milestone2    string  `json:"milestone2,omitempty"`
	Milestone3    string  `json:"milestone3,omitempty"`
}

// ProfileSummary represents a user profile from `GET /1.0/profiles/{uid}`.
type ProfileSummary struct {
	UID       string    `json:"uid"`
	Nickname  string    `json:"nickname"`
	UserType  string    `json:"user_type,omitempty"`
	Moderator bool      `json:"moderator,omitempty"`
	Points    int       `json:"points"`
	LevelID   string    `json:"level_id,omitempty"`
	LevelName string    `json:"level_name,omitempty"`
	Level     UserLevel `json:"level"`
}

// LeaderboardResponse represents a leaderboard from `GET /1.0/leaderboards/...`.
type LeaderboardResponse struct {
	Domain     string           `json:"domain"`
	Period     string           `json:"period"`
	PeriodName string           `json:"period_name,omitempty"`
	Metric     string           `json:"metric"`
	MetricName string           `json:"metric_name,omitempty"`
	Date       string           `json:"date,omitempty"`
	PrevDate   string           `json:"prev_date,omitempty"`
	HTMLURL    string           `json:"html_url,omitempty"`
	InList     bool             `json:"in_list"`
	Realtime   bool             `json:"realtime"`
	Rows       []LeaderboardRow `json:"rows"`
	Me         *LeaderboardRow  `json:"me,omitempty"`
}

// LeaderboardRow represents one ranked entry in a LeaderboardResponse.
type LeaderboardRow struct {
	Rank    int             `json:"rank"`
	Points  int             `json:"points"`
	When    string          `json:"when,omitempty"`
	Me      bool            `json:"me,omitempty"`
	Profile *ProfileSummary `json:"profile,omitempty"`
}

// GetWordProgress fetches sense and mastery progress for a single word (`GET /1.0/progress/words/{word}`).
func (p *ProgressService) GetWordProgress(ctx context.Context, word string, detail bool) (*WordProgressDetail, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, fmt.Errorf("vocab: word is required")
	}
	var q url.Values
	if detail {
		q = url.Values{"detail": {"true"}}
	}
	var resp WordProgressDetail
	path := "/progress/words/" + url.PathEscape(word)
	if _, err := p.client.doAPI(ctx, http.MethodGet, path, q, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetWordsProgressBatch fetches mastery progress for a slice of words in chunks of 50 (`GET /1.0/progress/words`).
func (p *ProgressService) GetWordsProgressBatch(ctx context.Context, words []string) ([]WordProgressDetail, error) {
	cleaned := make([]string, 0, len(words))
	for _, w := range words {
		if trimmed := strings.TrimSpace(w); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) == 0 {
		return nil, nil
	}

	var all []WordProgressDetail
	const chunkSize = 50
	for start := 0; start < len(cleaned); start += chunkSize {
		end := start + chunkSize
		if end > len(cleaned) {
			end = len(cleaned)
		}
		chunk := cleaned[start:end]
		q := url.Values{
			"limit": {strconv.Itoa(len(chunk))},
		}
		for _, w := range chunk {
			q.Add("words", w)
		}
		var batch []WordProgressDetail
		if _, err := p.client.doAPI(ctx, http.MethodGet, "/progress/words", q, nil, nil, &batch); err != nil {
			return all, err
		}
		all = append(all, batch...)
	}
	return all, nil
}

// SetWordPriority updates a word's learning priority (`PATCH /1.0/progress/words/{word}`).
func (p *ProgressService) SetWordPriority(ctx context.Context, word string, priority WordPriority) (*WordProgressDetail, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, fmt.Errorf("vocab: word is required")
	}
	if !priority.IsValid() {
		return nil, fmt.Errorf("vocab: invalid word priority %d (expected -1, 0, or 1)", priority)
	}
	body := map[string]int{"priority": int(priority)}
	var resp WordProgressDetail
	path := "/progress/words/" + url.PathEscape(word)
	if _, err := p.client.doAPI(ctx, http.MethodPatch, path, nil, body, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StartLearningWord adds `word` to the user's active learning program (`POST /progress/startlearning.json`).
func (p *ProgressService) StartLearningWord(ctx context.Context, word string) error {
	word = strings.TrimSpace(word)
	if word == "" {
		return fmt.Errorf("vocab: word is required")
	}
	form := url.Values{"word": {word}}
	return p.client.doWebForm(ctx, "/progress/startlearning.json", form, nil)
}

// StopLearningWord removes `word` from the user's active learning program (`POST /progress/stoplearning.json`).
func (p *ProgressService) StopLearningWord(ctx context.Context, word string) error {
	word = strings.TrimSpace(word)
	if word == "" {
		return fmt.Errorf("vocab: word is required")
	}
	form := url.Values{"word": {word}}
	return p.client.doWebForm(ctx, "/progress/stoplearning.json", form, nil)
}

// ListLearningWords fetches paginated words from `/account/progress/words.json`
// (`filter` = `"learning"`, `"trouble"`, or `"mastered"`).
func (p *ProgressService) ListLearningWords(ctx context.Context, filter WordFilter, skip int, sortBy, sortDirection string) ([]AccountProgressWord, error) {
	if filter == "" {
		filter = WordFilterLearning
	}
	if !filter.IsValid() {
		return nil, fmt.Errorf("vocab: invalid word filter %q (expected learning, trouble, or mastered)", filter)
	}
	if skip < 0 {
		skip = 0
	}
	form := url.Values{
		"filter": {string(filter)},
		"skip":   {strconv.Itoa(skip)},
	}
	if sortBy != "" {
		form.Set("sortBy", sortBy)
	}
	if sortDirection != "" {
		form.Set("sortDirection", sortDirection)
	}
	var words []AccountProgressWord
	if err := p.client.doWebForm(ctx, "/account/progress/words.json", form, &words); err != nil {
		return nil, err
	}
	return words, nil
}

// GetListProgress fetches the user's mastery status on a vocabulary list (`GET /1.0/progress/lists/{listId}`).
func (p *ProgressService) GetListProgress(ctx context.Context, listID int64) (*ListProgressStatus, error) {
	if listID <= 0 {
		return nil, fmt.Errorf("vocab: listID must be > 0")
	}
	var resp ListProgressStatus
	path := fmt.Sprintf("/progress/lists/%d", listID)
	if _, err := p.client.doAPI(ctx, http.MethodGet, path, nil, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetEffort fetches the learner's effort dashboard metrics (`GET /1.0/me/effort?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD`).
func (p *ProgressService) GetEffort(ctx context.Context, startDate, endDate string) (*EffortStats, error) {
	q := url.Values{}
	if startDate != "" {
		q.Set("start_date", startDate)
	}
	if endDate != "" {
		q.Set("end_date", endDate)
	}
	var resp EffortStats
	if _, err := p.client.doAPI(ctx, http.MethodGet, "/me/effort", q, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetActivities fetches the user's recent or completed activities (`GET /1.0/me/activities`).
func (p *ProgressService) GetActivities(ctx context.Context, opts ActivitiesQuery) (*ActivitiesResponse, error) {
	q := url.Values{}
	if opts.Complete != nil {
		q.Set("complete", strconv.FormatBool(*opts.Complete))
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Include != "" {
		q.Set("include", opts.Include)
	}
	if opts.WordlistID > 0 {
		q.Set("wordlist_id", strconv.FormatInt(opts.WordlistID, 10))
	}
	if opts.ActivityType != "" {
		q.Set("activity_type", opts.ActivityType)
	}
	if opts.StartedAfter != "" {
		q.Set("started_after", opts.StartedAfter)
	}
	if opts.EndDate != "" {
		q.Set("enddate", opts.EndDate)
	}
	var resp ActivitiesResponse
	httpResp, err := p.client.doAPI(ctx, http.MethodGet, "/me/activities", q, nil, nil, &resp)
	if err != nil {
		return nil, err
	}
	if httpResp != nil {
		if links := ParseLinkHeader(httpResp.Header.Get("Link"), p.client.apiBaseURL); links != nil {
			resp.NextPageURI = links["next"]
		}
	}
	return &resp, nil
}

// GetActivitiesByURI fetches the next page of activities using a `NextPageURI` returned by `GetActivities`.
func (p *ProgressService) GetActivitiesByURI(ctx context.Context, nextPageURI string) (*ActivitiesResponse, error) {
	nextPageURI = strings.TrimSpace(nextPageURI)
	if nextPageURI == "" {
		return nil, fmt.Errorf("vocab: nextPageURI is required")
	}
	var resp ActivitiesResponse
	httpResp, err := p.client.doAPI(ctx, http.MethodGet, nextPageURI, nil, nil, nil, &resp)
	if err != nil {
		return nil, err
	}
	if httpResp != nil {
		if links := ParseLinkHeader(httpResp.Header.Get("Link"), p.client.apiBaseURL); links != nil {
			resp.NextPageURI = links["next"]
		}
	}
	return &resp, nil
}

// GetAchievements fetches the authenticated user's earned achievements (`GET /1.0/me/achievements?limit=500`).
func (p *ProgressService) GetAchievements(ctx context.Context, limit int) ([]AchievementSummary, error) {
	if limit <= 0 {
		limit = 500
	}
	q := url.Values{"limit": {strconv.Itoa(limit)}}
	var envelope struct {
		Achievements []AchievementSummary `json:"achievements"`
	}
	if _, err := p.client.doAPI(ctx, http.MethodGet, "/me/achievements", q, nil, nil, &envelope); err != nil {
		return nil, err
	}
	return envelope.Achievements, nil
}

// GetProfile fetches a user's public profile (`GET /1.0/profiles/{uid}`).
func (p *ProgressService) GetProfile(ctx context.Context, uid string) (*ProfileSummary, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return nil, fmt.Errorf("vocab: uid is required")
	}
	var envelope struct {
		Profile *ProfileSummary `json:"profile"`
	}
	if _, err := p.client.doAPI(ctx, http.MethodGet, "/profiles/"+url.PathEscape(uid), nil, nil, nil, &envelope); err != nil {
		return nil, err
	}
	return envelope.Profile, nil
}

// GetIndividualLeaderboard fetches an individual leaderboard (`GET /1.0/leaderboards/individual/{metric}` or `/{metric}/{period}/{yyyymmdd}`).
// `metric` can be `"points"`, `"mastered"`, or an achievement ID; `period` can be `""` (all-time), `"daily"`, `"weekly"`, `"monthly"`, or `"season"`.
func (p *ProgressService) GetIndividualLeaderboard(ctx context.Context, metric, period, yyyymmdd string, top int, uid string) (*LeaderboardResponse, error) {
	if metric == "" {
		metric = "points"
	}
	if top <= 0 {
		top = 20
	}
	path := "/leaderboards/individual/" + url.PathEscape(metric)
	if period != "" && yyyymmdd != "" {
		path = fmt.Sprintf("/leaderboards/individual/%s/%s/%s", url.PathEscape(metric), url.PathEscape(period), url.PathEscape(yyyymmdd))
	}
	q := url.Values{"top": {strconv.Itoa(top)}}
	if uid != "" {
		q.Set("uid", uid)
	}
	var resp LeaderboardResponse
	if _, err := p.client.doAPI(ctx, http.MethodGet, path, q, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
