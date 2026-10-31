package vocab

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// TrainerService implements Vocabulary.com's VocabTrainer (`v=4`) and List Practice (`v=3`)
// endpoints (`/challenge/start.json`, `/challenge/nextquestion.json`, `/challenge/saveanswer.json`,
// `/challenge/hint.json`, `/challenge/reset.json`, and spoken audio URLs).
type TrainerService struct {
	client *Client
}

// StartChallengeRequest configures a call to `POST /challenge/start.json`.
type StartChallengeRequest struct {
	ActivityType ActivityType  // "c" (VocabTrainer) or "p" (List Practice)
	Version      ClientVersion // 4 for VocabTrainer, 3 for List Practice
	WordlistID   int64         // Optional list ID for list-scoped training or practice
	ActivityID   string        // Optional activity ID (?aid=...)
	Secret       string        // Optional previous turn secret to resume
}

// ChallengePlayerData represents `pdata` returned across `/challenge/*.json` responses.
type ChallengePlayerData struct {
	Points      int                 `json:"points"`
	NumPlayed   int                 `json:"numplayed"`
	NumMastered int                 `json:"nummastered"`
	Level       UserLevel           `json:"level"`
	Lists       []ChallengeListInfo `json:"lists,omitempty"`
}

// ChallengeListInfo represents an active learning list inside `pdata.lists`.
type ChallengeListInfo struct {
	Current    bool    `json:"current"`
	ListID     int64   `json:"listId"`
	Name       string  `json:"name"`
	Priority   int     `json:"priority"`
	Progress   float64 `json:"progress"`
	WordCount  int     `json:"wordcount"`
	TestPrepID string  `json:"testPrepId,omitempty"`
}

// ChallengeRound represents `round` metadata in `/challenge/*.json`.
type ChallengeRound struct {
	Number      int `json:"number"`
	PlayedCount int `json:"played_count"`
	Streak      int `json:"streak"`
}

// ChallengeGame represents `game` metadata when practicing a specific list (`activitytype="p"`).
type ChallengeGame struct {
	Type            string   `json:"type"`
	WordlistID      int64    `json:"wordlistid,omitempty"`
	Name            string   `json:"name,omitempty"`
	Progress        float64  `json:"progress"`
	Played          int      `json:"played,omitempty"`
	Correct         int      `json:"correct,omitempty"`
	Points          int      `json:"points,omitempty"`
	TroubleWords    []string `json:"trouble_words,omitempty"`
	TestPrepProgram any      `json:"testPrepProgram,omitempty"`
}

// ChallengeStateResponse is the response returned by `/challenge/start.json` and `/challenge/nextquestion.json`.
type ChallengeStateResponse struct {
	Version        int                  `json:"v,omitempty"`
	Action         string               `json:"action,omitempty"` // "newround", "next", "resume", "trialover"
	Secret         string               `json:"secret"`
	PlayerData     *ChallengePlayerData `json:"pdata,omitempty"`
	Round          *ChallengeRound      `json:"round,omitempty"`
	Game           *ChallengeGame       `json:"game,omitempty"`
	Question       *RawQuestion         `json:"question,omitempty"`
	ParsedQuestion *ParsedQuestion      `json:"parsed_question,omitempty"`
}

// AnswerOutcome represents the `answer` object inside `POST /challenge/saveanswer.json`,
// augmented with `Attempt` and `EvaluatedLocally` so callers can distinguish server-persisted
// first attempts from locally-graded subsequent retries.
type AnswerOutcome struct {
	Correct          bool                `json:"correct"`
	CorrectChoice    int                 `json:"correct_choice"` // 0-based choice index (0..3)
	UserChoice       int                 `json:"user_choice,omitempty"`
	AcceptedAnswers  []string            `json:"accepted_answers,omitempty"`
	Word             string              `json:"word"`
	Points           int                 `json:"points"`
	Bonus            int                 `json:"bonus,omitempty"`
	Hints            bool                `json:"hints,omitempty"`
	QuestionMode     string              `json:"question_mode,omitempty"`
	QuestionType     QuestionType        `json:"question_type,omitempty"`
	ResponseTime     int64               `json:"response_time,omitempty"`
	RoundStreak      int                 `json:"round_streak,omitempty"`
	RoundTurn        int                 `json:"round_turn,omitempty"`
	SessionTime      int64               `json:"session_time,omitempty"`
	Streak           int                 `json:"streak,omitempty"`
	Turn             int                 `json:"turn,omitempty"`
	PlayedAt         string              `json:"played_at,omitempty"`
	PageFreq         float64             `json:"pagefreq,omitempty"`
	Frequency        *WordFrequency      `json:"frequency,omitempty"`
	ListsMastered    []int64             `json:"lists_mastered,omitempty"`
	ListsProgress    []int64             `json:"lists_progress,omitempty"`
	Achievements     []EarnedAchievement `json:"achievements,omitempty"`
	Progress         *AnswerWordProgress `json:"progress,omitempty"`
	Blurb            *WordBlurb          `json:"blurb,omitempty"`
	Sense            *AnswerWordSense    `json:"sense,omitempty"`
	SentenceHTML     string              `json:"sentence_html,omitempty"`
	Attempt          int                 `json:"attempt,omitempty"`
	EvaluatedLocally bool                `json:"evaluated_locally,omitempty"`
}

// Clone returns a deep copy of AnswerOutcome.
func (ao *AnswerOutcome) Clone() *AnswerOutcome {
	if ao == nil {
		return nil
	}
	cp := *ao
	if ao.AcceptedAnswers != nil {
		cp.AcceptedAnswers = make([]string, len(ao.AcceptedAnswers))
		copy(cp.AcceptedAnswers, ao.AcceptedAnswers)
	}
	if ao.Frequency != nil {
		f := *ao.Frequency
		cp.Frequency = &f
	}
	if ao.ListsMastered != nil {
		cp.ListsMastered = make([]int64, len(ao.ListsMastered))
		copy(cp.ListsMastered, ao.ListsMastered)
	}
	if ao.ListsProgress != nil {
		cp.ListsProgress = make([]int64, len(ao.ListsProgress))
		copy(cp.ListsProgress, ao.ListsProgress)
	}
	if ao.Achievements != nil {
		cp.Achievements = make([]EarnedAchievement, len(ao.Achievements))
		copy(cp.Achievements, ao.Achievements)
	}
	if ao.Progress != nil {
		pr := *ao.Progress
		cp.Progress = &pr
	}
	if ao.Blurb != nil {
		bl := *ao.Blurb
		cp.Blurb = &bl
	}
	if ao.Sense != nil {
		sn := *ao.Sense
		if ao.Sense.Audio != nil {
			sn.Audio = make([]string, len(ao.Sense.Audio))
			copy(sn.Audio, ao.Sense.Audio)
		}
		if ao.Sense.PronunciationVariants != nil {
			sn.PronunciationVariants = make([]AnswerPronunciation, len(ao.Sense.PronunciationVariants))
			copy(sn.PronunciationVariants, ao.Sense.PronunciationVariants)
		}
		cp.Sense = &sn
	}
	return &cp
}

// WordFrequency holds frequency band and rank from `saveanswer.json`.
type WordFrequency struct {
	Band int `json:"band"`
	Rank int `json:"rank"`
}

// EarnedAchievement represents an achievement unlocked or incremented on an answer.
type EarnedAchievement struct {
	ID           string  `json:"id"`
	Type         string  `json:"type,omitempty"`
	Name         string  `json:"name,omitempty"`
	Count        int     `json:"count,omitempty"`
	Milestone    int     `json:"milestone,omitempty"`
	NewMilestone bool    `json:"new_milestone,omitempty"`
	Bonus        int     `json:"bonus,omitempty"`
	Progress     float64 `json:"progress,omitempty"`
}

// AnswerWordProgress holds the updated mastery progress for the answered word.
type AnswerWordProgress struct {
	PlayedAt    string  `json:"played_at,omitempty"`
	Priority    int     `json:"priority"`
	Progress    float64 `json:"progress"`
	ScheduledAt string  `json:"scheduled_at,omitempty"`
	Value       float64 `json:"value"`
}

// WordBlurb contains the short and long conversational explanations of a word.
type WordBlurb struct {
	Short string `json:"short"`
	Long  string `json:"long"`
}

// AnswerWordSense contains the sense definition and pronunciation data returned on answer completion.
type AnswerWordSense struct {
	ID                    int                   `json:"id"`
	Ordinal               string                `json:"ordinal,omitempty"`
	POS                   string                `json:"pos,omitempty"`
	PartOfSpeech          string                `json:"part_of_speech,omitempty"`
	Definition            string                `json:"definition"`
	Audio                 []string              `json:"audio,omitempty"`
	VideoHost             string                `json:"videoHost,omitempty"`
	ImageQuestionRoot     string                `json:"imageQuestionRoot,omitempty"`
	PronunciationVariants []AnswerPronunciation `json:"pronunciationVariants,omitempty"`
}

// AnswerPronunciation holds US/UK pronunciation variants inside AnswerWordSense.
type AnswerPronunciation struct {
	AudioID string `json:"audioId,omitempty"`
	IPA     string `json:"ipa,omitempty"`
	Region  string `json:"region,omitempty"`
	Video   *struct {
		URL          string `json:"url,omitempty"`
		ThumbnailURL string `json:"thumbnailURL,omitempty"`
	} `json:"video,omitempty"`
}

// SaveAnswerResponse is the response returned by `POST /challenge/saveanswer.json`.
type SaveAnswerResponse struct {
	Secret     string               `json:"secret"`
	PlayerData *ChallengePlayerData `json:"pdata,omitempty"`
	Round      *ChallengeRound      `json:"round,omitempty"`
	Game       *ChallengeGame       `json:"game,omitempty"`
	Answer     *AnswerOutcome       `json:"answer"`
}

// HintResponse is the response returned by `POST /challenge/hint.json`.
type HintResponse struct {
	Secret     string               `json:"secret"`
	PlayerData *ChallengePlayerData `json:"pdata,omitempty"`
	Definition string               `json:"def,omitempty"`    // Populated when HintType == "D"
	Nonces     []string             `json:"nonces,omitempty"` // 2 wrong nonces when HintType == "F" (50/50)
	Word       string               `json:"word,omitempty"`   // Target word when HintType == "E" (Word in the Wild)
}

// Start initializes or resumes a VocabTrainer (`activitytype="c"`, `v=4`) or List Practice (`activitytype="p"`, `v=3`) session.
func (t *TrainerService) Start(ctx context.Context, req StartChallengeRequest) (*ChallengeStateResponse, error) {
	actType := req.ActivityType
	if actType == "" {
		actType = ActivityVocabTrainer
	}
	ver := req.Version
	if ver == 0 {
		if actType == ActivityListPractice {
			ver = ClientVersionClassicWeb
		} else {
			ver = ClientVersionVocabTrainer
		}
	}

	form := url.Values{
		"activitytype": {string(actType)},
		"v":            {strconv.Itoa(int(ver))},
	}
	if req.WordlistID > 0 {
		form.Set("wordlistid", strconv.FormatInt(req.WordlistID, 10))
	}
	if req.ActivityID != "" {
		form.Set("activityid", req.ActivityID)
	}
	if req.Secret != "" {
		form.Set("secret", req.Secret)
	}

	var resp ChallengeStateResponse
	if err := t.client.doWebForm(ctx, "/challenge/start.json", form, &resp); err != nil {
		return nil, err
	}
	if err := t.populateParsedQuestion(&resp); err != nil {
		return &resp, err
	}
	if resp.Action == "trialover" {
		return &resp, ErrTrialOver
	}
	return &resp, nil
}

// NextQuestion advances to the next question in the current session (`POST /challenge/nextquestion.json`).
func (t *TrainerService) NextQuestion(ctx context.Context, secret string, version ClientVersion) (*ChallengeStateResponse, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("vocab: secret is required for NextQuestion")
	}
	if version == 0 {
		version = ClientVersionVocabTrainer
	}
	form := url.Values{
		"secret": {secret},
		"v":      {strconv.Itoa(int(version))},
	}
	var resp ChallengeStateResponse
	if err := t.client.doWebForm(ctx, "/challenge/nextquestion.json", form, &resp); err != nil {
		return nil, err
	}
	if err := t.populateParsedQuestion(&resp); err != nil {
		return &resp, err
	}
	if resp.Action == "trialover" {
		return &resp, ErrTrialOver
	}
	return &resp, nil
}

// SubmitAnswer submits the first-attempt answer (`a` = choice nonce or normalized spelling string)
// to `POST /challenge/saveanswer.json`, applying both minimum response time (`rt`) and rolling 30s cooldown pacing.
func (t *TrainerService) SubmitAnswer(ctx context.Context, secret, answerNonceOrText string, responseTimeMS int64, version ClientVersion) (*SaveAnswerResponse, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("vocab: secret is required for SubmitAnswer")
	}
	if strings.TrimSpace(answerNonceOrText) == "" {
		return nil, fmt.Errorf("vocab: answer nonce or text is required for SubmitAnswer")
	}
	if version == 0 {
		version = ClientVersionVocabTrainer
	}

	commit, release, err := t.client.pacingGuard.ReserveTrainerSlot(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	if responseTimeMS < MinChallengeFirstTryMS {
		responseTimeMS = 1250
	}

	form := url.Values{
		"a":      {answerNonceOrText},
		"rt":     {strconv.FormatInt(responseTimeMS, 10)},
		"secret": {secret},
		"v":      {strconv.Itoa(int(version))},
	}
	var resp SaveAnswerResponse
	if err := t.client.doWebForm(ctx, "/challenge/saveanswer.json", form, &resp); err != nil {
		return nil, err
	}
	commit()
	if resp.Answer != nil {
		resp.Answer.Attempt = 1
		resp.Answer.EvaluatedLocally = false
	}
	return &resp, nil
}

// UseHint requests a lifeline hint (`F` 50/50, `D` Definition, or `E` Example) via `POST /challenge/hint.json`.
// Note: UseHint returns an updated `Secret` which must replace the session's current secret.
func (t *TrainerService) UseHint(ctx context.Context, secret string, hintType HintType, version ClientVersion) (*HintResponse, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("vocab: secret is required for UseHint")
	}
	if !hintType.IsValid() {
		return nil, fmt.Errorf("vocab: invalid hint type %q (expected F, D, or E)", hintType)
	}
	if version == 0 {
		version = ClientVersionVocabTrainer
	}
	form := url.Values{
		"secret": {secret},
		"type":   {string(hintType)},
		"v":      {strconv.Itoa(int(version))},
	}
	var resp HintResponse
	if err := t.client.doWebForm(ctx, "/challenge/hint.json", form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Reset resets an out-of-sync challenge state on the server (`POST /challenge/reset.json`).
func (t *TrainerService) Reset(ctx context.Context) error {
	return t.client.doWebForm(ctx, "/challenge/reset.json", url.Values{}, nil)
}

// QuestionAudioURL returns the spoken audio stream URL for a question prompt, description, or blurb
// (`https://api.vocabulary.com/1.0/challenge/audio/{kind}?v=1&secret={secret}`).
func (t *TrainerService) QuestionAudioURL(kind, secret string) string {
	switch kind {
	case "question", "description", "blurb":
	default:
		kind = "question"
	}
	q := url.Values{
		"v":      {"1"},
		"secret": {secret},
	}
	return appendQuery(t.client.resolveAPIURL("/challenge/audio/"+kind), q)
}

func (t *TrainerService) populateParsedQuestion(resp *ChallengeStateResponse) error {
	if resp == nil || resp.Question == nil || resp.Question.Code == "" {
		return nil
	}
	pq, err := resp.Question.Decode(t.client.audioBaseURL)
	if err != nil {
		return err
	}
	resp.ParsedQuestion = pq
	return nil
}
