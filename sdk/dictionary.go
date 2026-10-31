package vocab

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/veryshyjelly/vocabulary_go/sdk/internal/htmlparse"
)

// Corpus domain filter codes used by `https://corpus.vocabulary.com/api/1.0/examples`.
type CorpusDomain string

const (
	CorpusDomainAll        CorpusDomain = ""  // All sources (uses /examples/random.json)
	CorpusDomainLiterature CorpusDomain = "F" // Literature / Fiction
	CorpusDomainArts       CorpusDomain = "A" // Arts / Culture
	CorpusDomainNews       CorpusDomain = "N" // News
	CorpusDomainBusiness   CorpusDomain = "B" // Business
	CorpusDomainSports     CorpusDomain = "S" // Sports
	CorpusDomainScienceMed CorpusDomain = "M" // Science / Medicine
	CorpusDomainTechnology CorpusDomain = "T" // Technology
)

// DictionaryEntry is an alias for htmlparse.DictionaryEntry representing a full dictionary word page.
type DictionaryEntry = htmlparse.DictionaryEntry

// AutocompleteSuggestion is an alias for htmlparse.AutocompleteSuggestion.
type AutocompleteSuggestion = htmlparse.AutocompleteSuggestion

// WordOfTheDayEntry is an alias for htmlparse.WordOfTheDayEntry.
type WordOfTheDayEntry = htmlparse.WordOfTheDayEntry

// DictionaryService provides dictionary lookup, autocomplete, WordFinder search,
// corpus example sentences ("Word in the Wild"), audio pronunciation resolution, and Word of the Day.
type DictionaryService struct {
	client *Client
}

// WordFinderOptions configures `GET /wordfinder/search.json`.
type WordFinderOptions struct {
	SearchText     string
	SearchLocation string // "startingwith", "endingwith", "containinginorder", "including"
	NumLetters     int
	MaxResults     int
}

// WordFinderResponse represents the JSON returned by `GET /wordfinder/search.json`.
type WordFinderResponse struct {
	TotalResults int              `json:"totalResults"`
	Words        []WordFinderWord `json:"words"`
}

// WordFinderWord represents one word and its senses in WordFinderResponse.
type WordFinderWord struct {
	Word   string            `json:"word"`
	Senses []WordFinderSense `json:"senses"`
}

// WordFinderSense represents one sense inside WordFinderWord.
type WordFinderSense struct {
	POS             string   `json:"pos"`
	SynsetID        int      `json:"synsetid"`
	Definition      string   `json:"definition"`
	ShortDefinition string   `json:"shortdefinition,omitempty"`
	Synonyms        []string `json:"synonyms,omitempty"`
}

// CorpusExamplesResponse represents the JSON returned by `https://corpus.vocabulary.com/api/1.0/examples`.
type CorpusExamplesResponse struct {
	TotalHits int              `json:"totalHits"`
	Sentences []CorpusSentence `json:"sentences"`
}

// CorpusSentence represents a single real-world usage example from the Vocabulary.com Corpus API.
type CorpusSentence struct {
	CorpusID     string        `json:"corpusId"`
	Offsets      []int         `json:"offsets"` // [start, end] char offsets of the target word
	Sentence     string        `json:"sentence"`
	VolumeID     int64         `json:"volumeId"`
	VolumeOffset int           `json:"volumeOffset"`
	Volume       *CorpusVolume `json:"volume,omitempty"`
}

// CorpusVolume holds publication metadata for a CorpusSentence.
type CorpusVolume struct {
	ID            int64    `json:"id"`
	Author        string   `json:"author,omitempty"`
	Title         string   `json:"title,omitempty"`
	DateAdded     int64    `json:"dateAdded,omitempty"`
	DatePublished int64    `json:"datePublished,omitempty"`
	Domain        string   `json:"domain,omitempty"`
	Domains       []string `json:"domains,omitempty"`
	Locator       string   `json:"locator,omitempty"`
	SentenceCount int      `json:"sentenceCount,omitempty"`
	WordCount     int      `json:"wordCount,omitempty"`
	ASIN          string   `json:"asin,omitempty"`
	ISBN          string   `json:"isbn,omitempty"`
	Corpus        *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"corpus,omitempty"`
}

// SourceURL resolves the canonical external link for a CorpusSentence
// (following `vcom/examplebrowser.js` rules for Project Gutenberg, HTTP locators, and corpus fallback).
func (cs *CorpusSentence) SourceURL() string {
	if cs == nil || cs.Volume == nil {
		return ""
	}
	if cs.Volume.Corpus != nil && cs.Volume.Corpus.ID == "GUT" && cs.Volume.Locator != "" {
		return "https://www.gutenberg.org/ebooks/" + cs.Volume.Locator
	}
	if strings.HasPrefix(cs.Volume.Locator, "http://") || strings.HasPrefix(cs.Volume.Locator, "https://") {
		return cs.Volume.Locator
	}
	if cs.VolumeID > 0 {
		return fmt.Sprintf("https://corpus.vocabulary.com/go/%d", cs.VolumeID)
	}
	return ""
}

// WODArchiveResponse represents the JSON returned by `GET /word-of-the-day/archive/by-month.json`.
type WODArchiveResponse struct {
	HasMore           bool              `json:"hasMore"`
	NextPageYearMonth int               `json:"nextPageYearMonth"`
	Results           []WODArchiveMonth `json:"results"`
}

// WODArchiveMonth represents a monthly group of Word of the Day entries.
type WODArchiveMonth struct {
	FirstOfMonth  string           `json:"firstOfMonth"`
	YearMonth     int              `json:"yearMonth"`
	WordOfTheDays []WODArchiveItem `json:"wordOfTheDays"`
}

// WODArchiveItem represents a single day's entry inside WODArchiveMonth.
type WODArchiveItem struct {
	Date       string `json:"date"`
	QuestionID int64  `json:"questionId"`
	YYYYMMDD   int    `json:"yyyymmdd"`
	Word       string `json:"word"`
}

// PreviewQuestionResponse represents the response from `POST /challenge/preview.json`.
type PreviewQuestionResponse struct {
	Word           string          `json:"word"`
	Secret         string          `json:"secret"`
	Code           string          `json:"code"`
	ParsedQuestion *ParsedQuestion `json:"parsed_question,omitempty"`
}

// Lookup fetches and parses the full dictionary entry for `word` (`GET /dictionary/{word}`).
func (d *DictionaryService) Lookup(ctx context.Context, word string) (*DictionaryEntry, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, fmt.Errorf("vocab: word is required")
	}
	path := "/dictionary/" + url.PathEscape(word)
	rawBytes, _, err := d.client.doWebGet(ctx, path, nil, "text/html")
	if err != nil {
		return nil, err
	}
	return htmlparse.ParseDictionaryHTML(string(rawBytes), d.client.audioBaseURL)
}

// Autocomplete queries `GET /dictionary/autocomplete-ss` and parses the returned suggestions.
// Note: Also supports Vocabulary.com's advanced Lucene-like syntax (e.g. `pos:(noun OR verb) +synonym:happy`).
func (d *DictionaryService) Autocomplete(ctx context.Context, search string, maxResults int) ([]AutocompleteSuggestion, error) {
	search = strings.TrimSpace(search)
	if search == "" {
		return nil, nil
	}
	if maxResults <= 0 {
		maxResults = 10
	}
	q := url.Values{
		"search":      {search},
		"maxResults":  {strconv.Itoa(maxResults)},
		"filterLevel": {"0"},
	}
	rawBytes, _, err := d.client.doWebGet(ctx, "/dictionary/autocomplete-ss", q, "text/html")
	if err != nil {
		return nil, err
	}
	return htmlparse.ParseAutocompleteHTML(string(rawBytes))
}

// WordFinder searches words by pattern/length via `GET /wordfinder/search.json`.
func (d *DictionaryService) WordFinder(ctx context.Context, opts WordFinderOptions) (*WordFinderResponse, error) {
	loc := opts.SearchLocation
	if loc == "" {
		loc = "startingwith"
	}
	maxRes := opts.MaxResults
	if maxRes <= 0 {
		maxRes = 25
	}
	q := url.Values{
		"searchText":     {opts.SearchText},
		"searchLocation": {loc},
		"maxResults":     {strconv.Itoa(maxRes)},
	}
	if opts.NumLetters > 0 {
		q.Set("numLetters", strconv.Itoa(opts.NumLetters))
	} else {
		q.Set("numLetters", "")
	}
	var resp WordFinderResponse
	if err := d.client.doWebJSONGet(ctx, "/wordfinder/search.json", q, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RandomWord fetches a random dictionary entry via `GET /randomword`.
func (d *DictionaryService) RandomWord(ctx context.Context) (*DictionaryEntry, error) {
	rawBytes, _, err := d.client.doWebGet(ctx, "/randomword", nil, "text/html")
	if err != nil {
		return nil, err
	}
	return htmlparse.ParseDictionaryHTML(string(rawBytes), d.client.audioBaseURL)
}

// PreviewQuestion fetches a sample multiple-choice question for `word` (`POST /challenge/preview.json`).
func (d *DictionaryService) PreviewQuestion(ctx context.Context, word string) (*PreviewQuestionResponse, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, fmt.Errorf("vocab: word is required")
	}
	form := url.Values{
		"word": {word},
	}
	var resp PreviewQuestionResponse
	if err := d.client.doWebForm(ctx, "/challenge/preview.json", form, &resp); err != nil {
		return nil, err
	}
	if resp.Code != "" {
		rq := &RawQuestion{Code: resp.Code}
		pq, err := rq.Decode(d.client.audioBaseURL)
		if err != nil {
			return &resp, err
		}
		resp.ParsedQuestion = pq
	}
	return &resp, nil
}

// Examples fetches real-world usage sentences ("Word in the Wild") from `https://corpus.vocabulary.com/api/1.0/examples`.
func (d *DictionaryService) Examples(ctx context.Context, query string, domain CorpusDomain, startOffset, maxResults int) (*CorpusExamplesResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("vocab: query is required")
	}
	if maxResults <= 0 {
		maxResults = 24
	}
	if startOffset < 0 {
		startOffset = 0
	}
	q := url.Values{
		"query":       {query},
		"startOffset": {strconv.Itoa(startOffset)},
		"maxResults":  {strconv.Itoa(maxResults)},
	}
	endpoint := "/examples/random.json"
	if domain != CorpusDomainAll {
		endpoint = "/examples"
		q.Set("domain", string(domain))
	}

	var out CorpusExamplesResponse
	if err := d.client.doCorpusGet(ctx, endpoint, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AudioURLForID constructs the direct MP3 pronunciation stream URL for an audio ID (e.g. "C/1XB5QC9VRN0HU").
func (d *DictionaryService) AudioURLForID(audioID string) string {
	audioID = strings.TrimPrefix(strings.TrimSpace(audioID), "/")
	audioID = strings.TrimSuffix(audioID, ".mp3")
	if audioID == "" {
		return ""
	}
	return strings.TrimRight(d.client.audioBaseURL, "/") + "/" + audioID + ".mp3"
}

// ResolveWordAudioURL resolves the MP3 pronunciation URL for `word` using `POST /dictionary/audio.json`
// (when authenticated) or falling back to `GET /1.0/progress/words/{word}` (which works anonymously).
func (d *DictionaryService) ResolveWordAudioURL(ctx context.Context, word string, index int) (string, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return "", fmt.Errorf("vocab: word is required")
	}
	if index < 0 {
		index = 0
	}
	form := url.Values{
		"word": {word},
		"idx":  {strconv.Itoa(index)},
	}
	var audioResp struct {
		Result string `json:"result"`
	}
	if err := d.client.doWebForm(ctx, "/dictionary/audio.json", form, &audioResp); err == nil && audioResp.Result != "" {
		return d.AudioURLForID(audioResp.Result), nil
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	// Fallback: /1.0/progress/words/{word} returns sense.audio even with an anonymous Bearer token.
	wp, err := d.client.Progress.GetWordProgress(ctx, word, false)
	if err == nil && wp != nil && wp.Sense != nil && len(wp.Sense.Audio) > 0 {
		idx := index
		if idx >= len(wp.Sense.Audio) {
			idx = 0
		}
		return d.AudioURLForID(wp.Sense.Audio[idx]), nil
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	// Second fallback: parse dictionary page
	entry, err := d.Lookup(ctx, word)
	if err != nil {
		return "", err
	}
	for _, p := range entry.Pronunciations {
		if p.AudioURL != "" {
			return p.AudioURL, nil
		}
	}
	return "", ErrNotFound
}

// WordOfTheDay fetches and parses the Word of the Day for `date` (`YYYY-MM-DD`, or `""` for today).
func (d *DictionaryService) WordOfTheDay(ctx context.Context, date string) (*WordOfTheDayEntry, error) {
	path := "/word-of-the-day/"
	if trimmed := strings.TrimSpace(date); trimmed != "" {
		if _, err := time.Parse("2006-01-02", trimmed); err != nil {
			return nil, fmt.Errorf("vocab: invalid WordOfTheDay date %q (expected YYYY-MM-DD): %w", trimmed, err)
		}
		path = "/word-of-the-day/" + url.PathEscape(trimmed)
	}
	rawBytes, _, err := d.client.doWebGet(ctx, path, nil, "text/html")
	if err != nil {
		return nil, err
	}
	return htmlparse.ParseWordOfTheDayHTML(string(rawBytes), d.client.audioBaseURL)
}

// WordOfTheDayArchive fetches the monthly Word of the Day archive (`GET /word-of-the-day/archive/by-month.json`).
// `startYearMonth` is formatted as `YYYYMM` (e.g., `202610`).
func (d *DictionaryService) WordOfTheDayArchive(ctx context.Context, startYearMonth, numMonths int) (*WODArchiveResponse, error) {
	if numMonths <= 0 {
		numMonths = 3
	}
	q := url.Values{
		"sort":      {"descending"},
		"numMonths": {strconv.Itoa(numMonths)},
	}
	if startYearMonth > 0 {
		q.Set("startYearMonth", strconv.Itoa(startYearMonth))
	}
	var resp WODArchiveResponse
	if err := d.client.doWebJSONGet(ctx, "/word-of-the-day/archive/by-month.json", q, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
