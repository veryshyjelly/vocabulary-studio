package desktop

import (
	vocab "github.com/veryshyjelly/vocabulary_go/sdk"
)

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
