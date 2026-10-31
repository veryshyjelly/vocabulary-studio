package vocab

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/veryshyjelly/vocabulary_go/internal/htmlparse"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// QuestionType enumerates all 16 question type codes used by Vocabulary.com.
type QuestionType string

const (
	QuestionTypeAntonym               QuestionType = "A" // ANTONYM
	QuestionTypeDefinition            QuestionType = "D" // DEFINITION
	QuestionTypeCustom                QuestionType = "P" // CUSTOM
	QuestionTypeFillIn                QuestionType = "F" // FILLIN
	QuestionTypeImage4                QuestionType = "I" // IMAGE (4-image choices)
	QuestionTypeSingleImage           QuestionType = "G" // SINGLE_IMAGE
	QuestionTypeSentenceDefinition    QuestionType = "L" // SENTDEFINITION
	QuestionTypeSentenceSynonym       QuestionType = "H" // SENTSYNONYM
	QuestionTypeSpelling              QuestionType = "T" // SPELLING (type-in)
	QuestionTypeSynonym               QuestionType = "S" // SYNONYM
	QuestionTypeHyponymTypeOf         QuestionType = "Y" // HYPONYM_TYPE_OF
	QuestionTypeMeronymPartOf         QuestionType = "R" // MERONYM_PART_OF
	QuestionTypeMatchWordToDefinition QuestionType = "C" // MATCH_WORD_TO_DEFINITION
	QuestionTypeMultipleMeaning       QuestionType = "M" // MULTIPLE_MEANING
	QuestionTypeRelatedWord           QuestionType = "E" // RELATED_WORD
	QuestionTypeWordUsage             QuestionType = "U" // WORD_USAGE
)

// IsKnown reports whether the QuestionType is one of the 16 known Vocabulary.com question types.
func (qt QuestionType) IsKnown() bool {
	switch qt {
	case QuestionTypeAntonym,
		QuestionTypeDefinition,
		QuestionTypeCustom,
		QuestionTypeFillIn,
		QuestionTypeImage4,
		QuestionTypeSingleImage,
		QuestionTypeSentenceDefinition,
		QuestionTypeSentenceSynonym,
		QuestionTypeSpelling,
		QuestionTypeSynonym,
		QuestionTypeHyponymTypeOf,
		QuestionTypeMeronymPartOf,
		QuestionTypeMatchWordToDefinition,
		QuestionTypeMultipleMeaning,
		QuestionTypeRelatedWord,
		QuestionTypeWordUsage:
		return true
	default:
		return false
	}
}

// ActivityType identifies the gameplay mode in `/challenge/start.json`.
type ActivityType string

const (
	ActivityVocabTrainer ActivityType = "c" // General VocabTrainer / Challenge
	ActivityListPractice ActivityType = "p" // List Practice
)

// ClientVersion identifies the protocol version sent as `v` to `/challenge/*.json`.
type ClientVersion int

const (
	ClientVersionMobile       ClientVersion = 1
	ClientVersionClassicWeb   ClientVersion = 3 // Used by List Practice (/lists/{id}/practice)
	ClientVersionVocabTrainer ClientVersion = 4 // Used by modern /vocabtrainer/
)

// HintType identifies a lifeline/hint code in `/challenge/hint.json`.
type HintType string

const (
	HintFiftyFifty HintType = "F" // Eliminates 2 wrong choices (returns `nonces`)
	HintDefinition HintType = "D" // Shows word definition (returns `def`)
	HintExample    HintType = "E" // Shows "Word in the Wild" example (returns `word`)
)

// IsValid reports whether ht is a supported HintType ("F", "D", or "E").
func (ht HintType) IsValid() bool {
	switch ht {
	case HintFiftyFifty, HintDefinition, HintExample:
		return true
	default:
		return false
	}
}

// ChoiceOption is an alias for htmlparse.ChoiceOption representing one multiple-choice answer.
type ChoiceOption = htmlparse.ChoiceOption

// ParsedQuestion is the decoded, strongly-typed representation of a challenge or quiz question slide.
type ParsedQuestion struct {
	Turn             int            `json:"turn,omitempty"`
	Type             QuestionType   `json:"type"`
	Difficulty       float64        `json:"difficulty,omitempty"`
	Hints            []HintType     `json:"hints,omitempty"`
	AnswerStats      *AnswerStats   `json:"answer_stats,omitempty"`
	Mode             string         `json:"mode,omitempty"`
	PointValue       int            `json:"point_value,omitempty"`
	AudioID          string         `json:"audio_id,omitempty"`
	AudioURL         string         `json:"audio_url,omitempty"`
	TargetWord       string         `json:"target_word,omitempty"`
	InstructionsText string         `json:"instructions_text,omitempty"`
	InstructionsHTML string         `json:"instructions_html,omitempty"`
	SentenceText     string         `json:"sentence_text,omitempty"`
	SentenceHTML     string         `json:"sentence_html,omitempty"`
	BlankedWord      string         `json:"blanked_word,omitempty"`
	SourceText       string         `json:"source_text,omitempty"`
	PromptImageURL   string         `json:"prompt_image_url,omitempty"`
	IsSpelling       bool           `json:"is_spelling"`
	Choices          []ChoiceOption `json:"choices,omitempty"`
	RawHTML          string         `json:"raw_html,omitempty"`
}

// Clone returns a deep copy of ParsedQuestion so callers cannot mutate session state.
func (pq *ParsedQuestion) Clone() *ParsedQuestion {
	if pq == nil {
		return nil
	}
	cp := *pq
	if pq.Hints != nil {
		cp.Hints = make([]HintType, len(pq.Hints))
		copy(cp.Hints, pq.Hints)
	}
	if pq.AnswerStats != nil {
		as := *pq.AnswerStats
		cp.AnswerStats = &as
	}
	if pq.Choices != nil {
		cp.Choices = make([]ChoiceOption, len(pq.Choices))
		copy(cp.Choices, pq.Choices)
	}
	return &cp
}

// AnswerStats holds historical community accuracy for a question.
type AnswerStats struct {
	Correct int `json:"correct"`
	Total   int `json:"total"`
}

// RawQuestion represents the `question` object inside `/challenge/start.json` and `/challenge/nextquestion.json`.
type RawQuestion struct {
	Turn        int          `json:"turn"`
	Type        QuestionType `json:"type"`
	Code        string       `json:"code"` // Base64-encoded HTML slide
	Difficulty  float64      `json:"difficulty,omitempty"`
	AnswerStats *AnswerStats `json:"answerstats,omitempty"`
	Hints       []HintType   `json:"hints,omitempty"`
}

// Decode decodes the Base64 `Code` field and parses its HTML structure into a ParsedQuestion.
func (rq *RawQuestion) Decode(audioBaseURL string) (*ParsedQuestion, error) {
	if rq == nil || strings.TrimSpace(rq.Code) == "" {
		return nil, fmt.Errorf("vocab: question code is empty")
	}
	slide, err := htmlparse.DecodeAndParseSlide(rq.Code, string(rq.Type))
	if err != nil {
		return nil, err
	}
	qType := rq.Type
	if qType == "" && slide.QuestionType != "" {
		qType = QuestionType(slide.QuestionType)
	}
	var audioURL string
	if slide.AudioID != "" && audioBaseURL != "" {
		audioURL = strings.TrimRight(audioBaseURL, "/") + "/" + slide.AudioID + ".mp3"
	}
	var hintsCopy []HintType
	if rq.Hints != nil {
		hintsCopy = make([]HintType, len(rq.Hints))
		copy(hintsCopy, rq.Hints)
	}
	var statsCopy *AnswerStats
	if rq.AnswerStats != nil {
		sc := *rq.AnswerStats
		statsCopy = &sc
	}
	return &ParsedQuestion{
		Turn:             rq.Turn,
		Type:             qType,
		Difficulty:       rq.Difficulty,
		Hints:            hintsCopy,
		AnswerStats:      statsCopy,
		Mode:             slide.Mode,
		PointValue:       slide.PointValue,
		AudioID:          slide.AudioID,
		AudioURL:         audioURL,
		TargetWord:       slide.TargetWord,
		InstructionsText: slide.InstructionsText,
		InstructionsHTML: slide.InstructionsHTML,
		SentenceText:     slide.SentenceText,
		SentenceHTML:     slide.SentenceHTML,
		BlankedWord:      slide.BlankedWord,
		SourceText:       slide.SourceText,
		PromptImageURL:   slide.PromptImageURL,
		IsSpelling:       slide.IsSpelling,
		Choices:          slide.Choices,
		RawHTML:          slide.RawHTML,
	}, nil
}

// NormalizeSpellingGuess normalizes a user's spelling answer in the exact same way as Vocabulary.com's frontend:
// `input.trim().toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "")`.
func NormalizeSpellingGuess(guess string) string {
	trimmed := strings.ToLower(strings.TrimSpace(guess))
	if trimmed == "" {
		return ""
	}
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	res, _, err := transform.String(t, trimmed)
	if err != nil {
		return trimmed
	}
	return res
}

// ProficiencyScoreFromAbility converts a raw ability float (typically -11 to +10) into
// Vocabulary.com's 0–2100 Proficiency Score: round(100 * (ability + 11)).
func ProficiencyScoreFromAbility(ability float64) int {
	val := 100.0 * (ability + 11.0)
	if val < 0 {
		return int(val - 0.5)
	}
	return int(val + 0.5)
}
