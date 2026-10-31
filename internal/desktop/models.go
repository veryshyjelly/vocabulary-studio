package desktop

import (
	"strings"

	vocab "github.com/veryshyjelly/vocabulary_go"
	"golang.org/x/net/html"
)

const AppVersion = "1.0.0"

// SessionStoreMode identifies how authenticated sessions are persisted on desktop.
type SessionStoreMode string

const (
	SessionStoreKeyring SessionStoreMode = "keyring"
	SessionStoreFile    SessionStoreMode = "file"
	SessionStoreMemory  SessionStoreMode = "memory"
)

// IsValid reports whether the mode is one of "keyring", "file", or "memory".
func (m SessionStoreMode) IsValid() bool {
	return m == SessionStoreKeyring || m == SessionStoreFile || m == SessionStoreMemory
}

// AppPreferences stores non-sensitive user preferences for the desktop app.
type AppPreferences struct {
	Theme               string           `json:"theme"`                // "light", "dark", "system"
	PronunciationAccent string           `json:"pronunciation_accent"` // "US", "UK"
	DefaultView         string           `json:"default_view"`         // "home", "practice", "dictionary", "lists", "discover", "progress"
	SessionStoreMode    SessionStoreMode `json:"session_store_mode"`   // "keyring", "file", "memory"
	SearchHistory       []string         `json:"search_history"`
	AppVersion          string           `json:"app_version"`
}

// DefaultPreferences returns the default desktop application preferences.
func DefaultPreferences() AppPreferences {
	return AppPreferences{
		Theme:               "system",
		PronunciationAccent: "US",
		DefaultView:         "home",
		SessionStoreMode:    SessionStoreKeyring,
		SearchHistory:       []string{},
		AppVersion:          AppVersion,
	}
}

// RichSpan is a safe, structured text span extracted from HTML fragments (`<strong>`, `<i>`, `<em>`)
// so the React frontend can render rich typography without ever using dangerouslySetInnerHTML.
type RichSpan struct {
	Text   string `json:"text"`
	Bold   bool   `json:"bold,omitempty"`
	Italic bool   `json:"italic,omitempty"`
}

// ParseSafeRichSpans parses an HTML fragment and extracts plain text + safe Bold/Italic spans,
// stripping all scripts, styles, event handlers, and unsafe tags.
func ParseSafeRichSpans(rawHTML string) (plainText string, spans []RichSpan) {
	trimmed := strings.TrimSpace(rawHTML)
	if trimmed == "" {
		return "", nil
	}
	doc, err := html.Parse(strings.NewReader(trimmed))
	if err != nil {
		return trimmed, []RichSpan{{Text: trimmed}}
	}

	var rawSpans []RichSpan
	var walk func(n *html.Node, bold, italic bool)
	walk = func(n *html.Node, bold, italic bool) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			if tag == "script" || tag == "style" || tag == "noscript" {
				return
			}
			switch tag {
			case "strong", "b":
				bold = true
			case "em", "i", "dfn":
				italic = true
			}
		}
		if n.Type == html.TextNode {
			txt := n.Data
			if txt != "" {
				rawSpans = append(rawSpans, RichSpan{
					Text:   txt,
					Bold:   bold,
					Italic: italic,
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, bold, italic)
		}
	}
	walk(doc, false, false)

	// Coalesce adjacent spans with identical formatting
	var merged []RichSpan
	var plainBuilder strings.Builder
	for _, sp := range rawSpans {
		plainBuilder.WriteString(sp.Text)
		if len(merged) > 0 && merged[len(merged)-1].Bold == sp.Bold && merged[len(merged)-1].Italic == sp.Italic {
			merged[len(merged)-1].Text += sp.Text
		} else {
			merged = append(merged, sp)
		}
	}
	plain := strings.Join(strings.Fields(plainBuilder.String()), " ")
	return plain, merged
}

// AccountStatusDTO contains strictly non-sensitive user identity and subscription state.
// Secrets (passwords, cookies, GUIDs, Bearer JWTs) are never included.
type AccountStatusDTO struct {
	Authenticated    bool              `json:"authenticated"`
	ValidUser        bool              `json:"valid_user"`
	UID              string            `json:"uid,omitempty"`
	Nickname         string            `json:"nickname,omitempty"`
	FullName         string            `json:"full_name,omitempty"`
	Email            string            `json:"email,omitempty"`
	Role             string            `json:"role,omitempty"`
	Points           int               `json:"points"`
	Level            vocab.UserLevel   `json:"level"`
	Paid             bool              `json:"paid"`
	FullVocabTrainer bool              `json:"full_vocab_trainer"`
	SessionStoreMode SessionStoreMode  `json:"session_store_mode"`
	Bowl             *vocab.BowlSeason `json:"bowl,omitempty"`
}

// HomeDashboardDTO aggregates real data for the Home screen.
type HomeDashboardDTO struct {
	Account              AccountStatusDTO            `json:"account"`
	WordOfTheDay         *vocab.WordOfTheDayEntry    `json:"word_of_the_day,omitempty"`
	Effort               *vocab.EffortStats          `json:"effort,omitempty"`
	InProgressActivities []vocab.ActivityItem        `json:"in_progress_activities,omitempty"`
	LearningWords        []vocab.AccountProgressWord `json:"learning_words,omitempty"`
	TroubleWords         []vocab.AccountProgressWord `json:"trouble_words,omitempty"`
	RecentLists          []vocab.ProfileListSummary  `json:"recent_lists,omitempty"`
	SearchHistory        []string                    `json:"search_history"`
}

// ChoiceDTO represents a multiple-choice option in a question slide.
type ChoiceDTO struct {
	Index      int    `json:"index"`
	Nonce      string `json:"nonce"`
	AccessKey  string `json:"access_key,omitempty"`
	Text       string `json:"text,omitempty"`
	ImageURL   string `json:"image_url,omitempty"`
	Eliminated bool   `json:"eliminated,omitempty"`
}

// QuestionSlideDTO is the sanitized, frontend-ready representation of an active question.
type QuestionSlideDTO struct {
	Turn              int                `json:"turn"`
	Type              string             `json:"type"`
	TypeLabel         string             `json:"type_label"`
	Mode              string             `json:"mode"`
	PointValue        int                `json:"point_value"`
	AudioURL          string             `json:"audio_url,omitempty"`
	TargetWord        string             `json:"target_word,omitempty"`
	InstructionsText  string             `json:"instructions_text"`
	InstructionsSpans []RichSpan         `json:"instructions_spans,omitempty"`
	SentenceText      string             `json:"sentence_text,omitempty"`
	SentenceSpans     []RichSpan         `json:"sentence_spans,omitempty"`
	BlankedWord       string             `json:"blanked_word,omitempty"`
	SourceText        string             `json:"source_text,omitempty"`
	PromptImageURL    string             `json:"prompt_image_url,omitempty"`
	IsSpelling        bool               `json:"is_spelling"`
	IsImageGrid       bool               `json:"is_image_grid"`
	AvailableHints    []string           `json:"available_hints,omitempty"`
	Choices           []ChoiceDTO        `json:"choices,omitempty"`
	AnswerStats       *vocab.AnswerStats `json:"answer_stats,omitempty"`
}

// AnswerFeedbackDTO represents the result of submitting a multiple-choice or spelling attempt.
type AnswerFeedbackDTO struct {
	Correct           bool                      `json:"correct"`
	CorrectChoice     int                       `json:"correct_choice"`
	UserChoice        int                       `json:"user_choice"`
	AcceptedAnswers   []string                  `json:"accepted_answers,omitempty"`
	Word              string                    `json:"word"`
	PointsEarned      int                       `json:"points_earned"`
	BonusPoints       int                       `json:"bonus_points,omitempty"`
	Attempt           int                       `json:"attempt"`
	EvaluatedLocally  bool                      `json:"evaluated_locally"`
	ShortBlurbText    string                    `json:"short_blurb_text,omitempty"`
	ShortBlurbSpans   []RichSpan                `json:"short_blurb_spans,omitempty"`
	LongBlurbText     string                    `json:"long_blurb_text,omitempty"`
	LongBlurbSpans    []RichSpan                `json:"long_blurb_spans,omitempty"`
	PartOfSpeech      string                    `json:"part_of_speech,omitempty"`
	SenseDefinition   string                    `json:"sense_definition,omitempty"`
	AudioURL          string                    `json:"audio_url,omitempty"`
	UnblankedSentence string                    `json:"unblanked_sentence,omitempty"`
	WordProgress      float64                   `json:"word_progress,omitempty"`
	Achievements      []vocab.EarnedAchievement `json:"achievements,omitempty"`
}

// HintResultDTO represents the result of using a lifeline hint on the active question.
type HintResultDTO struct {
	HintType         string     `json:"hint_type"`
	DefinitionText   string     `json:"definition_text,omitempty"`
	DefinitionSpans  []RichSpan `json:"definition_spans,omitempty"`
	EliminatedNonces []string   `json:"eliminated_nonces,omitempty"`
	Word             string     `json:"word,omitempty"`
	ExampleSentence  string     `json:"example_sentence,omitempty"`
	ExampleSource    string     `json:"example_source,omitempty"`
}

// TrainerSessionStateDTO represents the complete UI state of the active VocabTrainer or List Practice session.
type TrainerSessionStateDTO struct {
	Active       bool                       `json:"active"`
	SessionState vocab.TrainerSessionState  `json:"session_state"`
	ActivityType string                     `json:"activity_type"` // "c" or "p"
	WordlistID   int64                      `json:"wordlist_id,omitempty"`
	Tries        int                        `json:"tries"`
	Question     *QuestionSlideDTO          `json:"question,omitempty"`
	LastFeedback *AnswerFeedbackDTO         `json:"last_feedback,omitempty"`
	Round        *vocab.ChallengeRound      `json:"round,omitempty"`
	Game         *vocab.ChallengeGame       `json:"game,omitempty"`
	PlayerData   *vocab.ChallengePlayerData `json:"player_data,omitempty"`
}

// BeeSessionDTO represents the UI state of an active Spelling Bee session.
type BeeSessionDTO struct {
	Active           bool                       `json:"active"`
	ListID           int64                      `json:"list_id"`
	ActivityID       string                     `json:"activity_id,omitempty"`
	AudioURL         string                     `json:"audio_url,omitempty"`
	FormattedExample string                     `json:"formatted_example,omitempty"`
	Definitions      []BeeDefDTO                `json:"definitions,omitempty"`
	Score            *vocab.BeeScore            `json:"score,omitempty"`
	Guesses          int                        `json:"guesses"`
	CanSurrender     bool                       `json:"can_surrender"`
	CompletedWord    bool                       `json:"completed_word"`
	Correct          bool                       `json:"correct"`
	Distance         int                        `json:"distance,omitempty"`
	RevealedWord     string                     `json:"revealed_word,omitempty"`
	HasNextWord      bool                       `json:"has_next_word"`
	PlayerData       *vocab.ChallengePlayerData `json:"player_data,omitempty"`
}

// BeeDefDTO represents a single blanked definition in Spelling Bee.
type BeeDefDTO struct {
	POS  string `json:"pos"`
	Text string `json:"text"`
}

// QuizSessionDTO represents the UI state of an active Quiz session.
type QuizSessionDTO struct {
	Active         bool                       `json:"active"`
	QuestionNumber int                        `json:"question_number"`
	QuestionTotal  int                        `json:"question_total"`
	IsComplete     bool                       `json:"is_complete"`
	ActivityID     string                     `json:"activity_id,omitempty"`
	Question       *QuestionSlideDTO          `json:"question,omitempty"`
	Game           *vocab.ChallengeGame       `json:"game,omitempty"`
	PlayerData     *vocab.ChallengePlayerData `json:"player_data,omitempty"`
}

// CorpusSentenceDTO wraps a CorpusSentence with its resolved canonical SourceURL.
type CorpusSentenceDTO struct {
	Sentence    string `json:"sentence"`
	Offsets     []int  `json:"offsets,omitempty"`
	SourceTitle string `json:"source_title,omitempty"`
	SourceName  string `json:"source_name,omitempty"`
	Author      string `json:"author,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
	CorpusID    string `json:"corpus_id,omitempty"`
}

// WordDetailDTO combines dictionary lookup, user progress (when available), and corpus examples.
type WordDetailDTO struct {
	Entry    *vocab.DictionaryEntry    `json:"entry"`
	Progress *vocab.WordProgressDetail `json:"progress,omitempty"`
	Examples []CorpusSentenceDTO       `json:"examples,omitempty"`
}

// CategoryCardDTO describes one of Vocabulary.com's 9 root curated unit categories.
type CategoryCardDTO struct {
	UnitID      string `json:"unit_id"`
	Tag         string `json:"tag"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// QuestionTypeLabel returns a human-readable badge label for any of the 16 question type codes.
func QuestionTypeLabel(qType string) string {
	switch vocab.QuestionType(qType) {
	case vocab.QuestionTypeAntonym:
		return "Antonym"
	case vocab.QuestionTypeDefinition:
		return "Definition"
	case vocab.QuestionTypeCustom:
		return "Passage Context"
	case vocab.QuestionTypeFillIn:
		return "Fill in the Blank"
	case vocab.QuestionTypeImage4:
		return "Visual Match (4 Images)"
	case vocab.QuestionTypeSingleImage:
		return "Visual Prompt"
	case vocab.QuestionTypeSentenceDefinition:
		return "Meaning in Context"
	case vocab.QuestionTypeSentenceSynonym:
		return "Synonym in Context"
	case vocab.QuestionTypeSpelling:
		return "Spell the Word"
	case vocab.QuestionTypeSynonym:
		return "Synonym"
	case vocab.QuestionTypeHyponymTypeOf:
		return "Type Of (Hyponym)"
	case vocab.QuestionTypeMeronymPartOf:
		return "Part Of (Meronym)"
	case vocab.QuestionTypeMatchWordToDefinition:
		return "Word Match"
	case vocab.QuestionTypeMultipleMeaning:
		return "Multiple Meanings"
	case vocab.QuestionTypeRelatedWord:
		return "Word in the Wild"
	case vocab.QuestionTypeWordUsage:
		return "Word Usage"
	default:
		if qType != "" {
			return "Question (" + qType + ")"
		}
		return "Vocabulary Question"
	}
}
