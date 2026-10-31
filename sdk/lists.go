package vocab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Root Category Unit IDs used by Vocabulary.com's curated collections (`/lists/*`).
const (
	UnitCategoryLiterature          = "gfpldfb3" // LIT - /lists/literature/
	UnitCategoryNonFiction          = "igonyd6p" // NON - /lists/non-fiction/
	UnitCategoryTextbooks           = "5gi4cws6" // TXT - /lists/textbooks-and-curricula/
	UnitCategoryTestPrep            = "kecpw3dy" // TST - /lists/test-prep/
	UnitCategoryCurrentEvents       = "hehvvpjm" // NWS - /lists/current-events/
	UnitCategoryRootsAndAffixes     = "lnin7xdp" // ROOTS - /lists/roots-and-affixes/
	UnitCategoryJustForFun          = "alhahx3a" // FUN - /lists/just-for-fun/
	UnitCategorySpeeches            = "z2pwznqw" // SPH - /lists/speeches/
	UnitCategoryHistoricalDocuments = "aoqd4lh6" // HIST - /lists/historical-documents/
)

// ListsService manages Vocabulary.com wordlists and curated units/collections.
type ListsService struct {
	client *Client
}

// VocabularyList represents a full wordlist returned by `POST /lists/load.json`.
type VocabularyList struct {
	Version      int        `json:"ver,omitempty"`
	WordlistID   int64      `json:"wordlistid"`
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	Shared       bool       `json:"shared"`
	CreateDate   string     `json:"createdate,omitempty"`
	ModifiedDate string     `json:"modifieddate,omitempty"`
	Words        []ListWord `json:"words,omitempty"`
}

// ListWord represents a word entry inside a VocabularyList.
type ListWord struct {
	Word            string           `json:"word"`
	Lang            string           `json:"lang,omitempty"`
	Definition      string           `json:"definition,omitempty"`
	ShortDefinition string           `json:"shortdefinition,omitempty"`
	Description     string           `json:"description,omitempty"`
	Audio           []string         `json:"audio,omitempty"`
	FFreq           float64          `json:"ffreq,omitempty"`
	Learnable       bool             `json:"learnable"`
	Difficulty      int              `json:"difficulty,omitempty"`
	Diff            float64          `json:"diff,omitempty"`
	Example         *ListWordExample `json:"example,omitempty"`
}

// ListWordExample represents an example sentence attached to a ListWord.
type ListWordExample struct {
	Text    string `json:"text"`
	Offsets []int  `json:"offsets,omitempty"`
	Source  *struct {
		ID       int64  `json:"id,omitempty"`
		Name     string `json:"name,omitempty"`
		CorpusID string `json:"corpusId,omitempty"`
	} `json:"source,omitempty"`
}

// FlashcardListResponse represents the rich wordlist returned by `GET https://api.vocabulary.com/1.0/lists/{listId}`
// (used by Vocabulary.com's modern Flashcards view with full senses, IPA variants, images, and pronunciation videos).
type FlashcardListResponse struct {
	Wordlist struct {
		ID          int64               `json:"id,omitempty"`
		Name        string              `json:"name,omitempty"`
		Description string              `json:"description,omitempty"`
		Words       []FlashcardListWord `json:"words"`
	} `json:"wordlist"`
	DictionaryVideoHost string `json:"dictionary_video_host,omitempty"`
	ImageRoot           string `json:"image_root,omitempty"`
}

// FlashcardListWord represents a single word inside FlashcardListResponse.
type FlashcardListWord struct {
	Word       string           `json:"word"`
	Lang       string           `json:"lang,omitempty"`
	Difficulty int              `json:"difficulty,omitempty"`
	Frequency  float64          `json:"frequency,omitempty"`
	Learnable  bool             `json:"learnable"`
	Example    *ListWordExample `json:"example,omitempty"`
	Sense      *FlashcardSense  `json:"sense,omitempty"`
}

// FlashcardSense contains the sense, image, audio, and IPA video metadata for a flashcard word.
type FlashcardSense struct {
	ID           int            `json:"id"`
	Ordinal      string         `json:"ordinal,omitempty"`
	Definition   string         `json:"definition"`
	PartOfSpeech string         `json:"part_of_speech"`
	ImageURL     string         `json:"image_url,omitempty"`
	Audio        []string       `json:"audio,omitempty"`
	IPAs         []FlashcardIPA `json:"ipas,omitempty"`
}

// FlashcardIPA holds regional IPA and pronunciation video metadata.
type FlashcardIPA struct {
	AudioID            string `json:"audio_id,omitempty"`
	VideoURL           string `json:"video_url,omitempty"`
	Region             string `json:"region,omitempty"`
	FormattedIPAString string `json:"formatted_ipastring,omitempty"`
	PlaceholderURL     string `json:"placeholder_url,omitempty"`
}

// ProfileListSummary represents a user's list item from `POST /lists/byprofile.json`.
type ProfileListSummary struct {
	WordlistID   int64  `json:"wordlistid"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	WordCount    int    `json:"wordcount"`
	Owner        bool   `json:"owner"`
	Shared       bool   `json:"shared,omitempty"`
	ActivityDate string `json:"activitydate,omitempty"`
}

// SmartSearchResponse represents the JSON returned by `GET /lists/smart-search.json`.
type SmartSearchResponse struct {
	TotalResults int               `json:"totalResults"`
	HasMore      bool              `json:"hasMore"`
	Results      []SmartSearchItem `json:"results"`
}

// SmartSearchItem represents one vocabulary list result in SmartSearchResponse.
type SmartSearchItem struct {
	WordlistID  int64    `json:"wordlistid"`
	Name        string   `json:"name"`
	WordCount   int      `json:"wordcount"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	ParentUnit  string   `json:"parentunit,omitempty"`
	Unit        *struct {
		WordCount int    `json:"wordcount"`
		Title     string `json:"title"`
		HTMLURL   string `json:"html_url"`
		Type      string `json:"type"`
		Author    string `json:"author,omitempty"`
		Img       string `json:"img,omitempty"`
	} `json:"unit,omitempty"`
}

// Unit represents a curated Collection/Unit (e.g. a book, test-prep pack, or category) from `/1.0/units`.
type Unit struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	FullTitle        string     `json:"full_title,omitempty"`
	Subtitle         string     `json:"subtitle,omitempty"`
	Type             string     `json:"type"` // "work", "author", "collection", "unit", "publisher", "subject"
	Author           string     `json:"author,omitempty"`
	ISBN             string     `json:"isbn,omitempty"`
	ShortDescription string     `json:"short_description,omitempty"`
	Description      string     `json:"description,omitempty"`
	OwnerUID         string     `json:"owner_uid,omitempty"`
	ParentUnits      []string   `json:"parent_units,omitempty"`
	CreatedAt        string     `json:"created_at,omitempty"`
	ModifiedAt       string     `json:"modified_at,omitempty"`
	URL              string     `json:"url,omitempty"`
	HTMLURL          string     `json:"html_url,omitempty"`
	Items            []UnitItem `json:"items,omitempty"`
	Assets           []struct {
		ID          string `json:"id,omitempty"`
		Name        string `json:"name"`
		ContentType string `json:"content_type,omitempty"`
	} `json:"assets,omitempty"`
}

// UnitItem represents a child wordlist or sub-unit inside a Unit.
type UnitItem struct {
	ID        any    `json:"id"` // Can be numeric wordlist ID or string unit ID
	Name      string `json:"name"`
	Title     string `json:"title,omitempty"`
	Type      string `json:"type"` // "list" or "unit"
	WordCount int    `json:"word_count,omitempty"`
	URL       string `json:"url,omitempty"`
	HTMLURL   string `json:"html_url,omitempty"`
}

// UnitsResponse wraps responses from `GET /1.0/units` and `GET /1.0/units/{unitId}`.
type UnitsResponse struct {
	AssetBase    string `json:"asset_base,omitempty"`
	Unit         *Unit  `json:"unit,omitempty"`
	Units        []Unit `json:"units,omitempty"`
	RelatedUnits []Unit `json:"related_units,omitempty"`
	TotalCount   int    `json:"total_count,omitempty"`
	NextPageURI  string `json:"next_page_uri,omitempty"`
}

// SearchUnitsOptions configures `GET /1.0/units`.
type SearchUnitsOptions struct {
	Query            string
	IDs              []string
	Types            []string // "work", "author", "collection", "unit", "publisher", "subject"
	TaggedWithUnitID string   // e.g. UnitCategoryLiterature ("gfpldfb3")
	ContainsListID   int64
	Featured         *bool
	SortBy           string // e.g. "featured_at"
	SortDirection    string
	IncludeRelated   bool
	Skip             int
	Limit            int
}

// LearnableSense represents a selectable word sense returned by `GET /lists/vocabgrabber/senses.json`.
type LearnableSense struct {
	ID         int    `json:"id"`
	Learnable  bool   `json:"learnable"`
	POS        string `json:"pos"`
	Group      int    `json:"group"`
	Ordinal    int    `json:"ordinal"`
	SubOrdinal int    `json:"subordinal"`
	Def        string `json:"def"`
}

// ValidatedWord represents a word validation item from `POST /lists/validate.json`.
type ValidatedWord struct {
	Word       string  `json:"word"`
	Lang       string  `json:"lang"`
	Found      bool    `json:"found"`
	SynsetID   int     `json:"synsetid,omitempty"`
	Definition string  `json:"definition,omitempty"`
	Freq       float64 `json:"freq,omitempty"`
	Learnable  bool    `json:"learnable"`
}

// VocabGrabberResponse represents the extracted vocabulary from `POST /lists/vocabgrabber/grab.json`.
type VocabGrabberResponse struct {
	Format       string        `json:"format"`
	Words        []GrabbedWord `json:"words"`
	Sentences    [][]int       `json:"sentences,omitempty"`
	NotLearnable []string      `json:"notlearnable,omitempty"`
	NotFound     []string      `json:"notfound,omitempty"`
}

// GrabbedWord represents one word extracted from text by VocabGrabber.
type GrabbedWord struct {
	Word string  `json:"word"`
	Uses []int   `json:"uses,omitempty"`
	Def  string  `json:"def,omitempty"`
	Diff int     `json:"diff,omitempty"`
	Freq float64 `json:"freq,omitempty"`
	Rel  float64 `json:"rel,omitempty"`
}

// CreateListWordInput represents a word entry when creating a new list or adding words to an existing list.
type CreateListWordInput struct {
	Word        string `json:"word"`
	Lang        string `json:"lang,omitempty"`
	Description string `json:"description,omitempty"`
	SynsetID    int    `json:"synsetid,omitempty"`
	Synsets     []int  `json:"synsets,omitempty"`
	Example     any    `json:"example,omitempty"`
}

// CreateListRequest configures `POST /lists/save.json` when creating a new vocabulary list.
type CreateListRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Action      string                `json:"action,omitempty"` // "create", "learn", or "assign"
	Shared      bool                  `json:"shared"`
	Words       []CreateListWordInput `json:"words"`
}

// Get loads a vocabulary list by ID via `POST /lists/load.json`.
func (l *ListsService) Get(ctx context.Context, listID int64) (*VocabularyList, error) {
	if listID <= 0 {
		return nil, fmt.Errorf("vocab: listID must be > 0")
	}
	form := url.Values{
		"id": {strconv.FormatInt(listID, 10)},
	}
	var envelope struct {
		Status int             `json:"status"`
		Result *VocabularyList `json:"result"`
	}
	if err := l.client.doWebForm(ctx, "/lists/load.json", form, &envelope); err != nil {
		return nil, err
	}
	if envelope.Result == nil {
		return nil, ErrNotFound
	}
	return envelope.Result, nil
}

// GetFlashcardList fetches the rich flashcard representation of a list (`GET https://api.vocabulary.com/1.0/lists/{listId}`),
// including sense definitions, images, audio IDs, and US/UK pronunciation video URLs.
func (l *ListsService) GetFlashcardList(ctx context.Context, listID int64) (*FlashcardListResponse, error) {
	if listID <= 0 {
		return nil, fmt.Errorf("vocab: listID must be > 0")
	}
	var resp FlashcardListResponse
	path := fmt.Sprintf("/lists/%d", listID)
	if _, err := l.client.doAPI(ctx, http.MethodGet, path, nil, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListUserLists fetches the authenticated user's own or recent lists (`POST /lists/byprofile.json`).
// `display` can be `"created"` or `""` (all recent lists); `orderBy` defaults to `"activitydate"`.
func (l *ListsService) ListUserLists(ctx context.Context, display, orderBy string, pageSize int) ([]ProfileListSummary, error) {
	if orderBy == "" {
		orderBy = "activitydate"
	}
	if pageSize <= 0 {
		pageSize = 100
	}
	form := url.Values{
		"orderby":  {orderBy},
		"pageSize": {strconv.Itoa(pageSize)},
	}
	if display != "" {
		form.Set("display", display)
	}
	var envelope struct {
		Result struct {
			Wordlists []ProfileListSummary `json:"wordlists"`
		} `json:"result"`
	}
	if err := l.client.doWebForm(ctx, "/lists/byprofile.json", form, &envelope); err != nil {
		return nil, err
	}
	return envelope.Result.Wordlists, nil
}

// Search searches public vocabulary lists (`GET /lists/smart-search.json`).
func (l *ListsService) Search(ctx context.Context, searchText string, skip, limit int, uniqueCollections bool) (*SmartSearchResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if skip < 0 {
		skip = 0
	}
	q := url.Values{
		"searchText":        {strings.TrimSpace(searchText)},
		"skip":              {strconv.Itoa(skip)},
		"limit":             {strconv.Itoa(limit)},
		"uniqueCollections": {strconv.FormatBool(uniqueCollections)},
	}
	var resp SmartSearchResponse
	if err := l.client.doWebJSONGet(ctx, "/lists/smart-search.json", q, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetUnit fetches a curated Unit/Collection and its sub-lists (`GET https://api.vocabulary.com/1.0/units/{unitId}`).
func (l *ListsService) GetUnit(ctx context.Context, unitID string) (*UnitsResponse, error) {
	unitID = strings.TrimSpace(unitID)
	if unitID == "" {
		return nil, fmt.Errorf("vocab: unitID is required")
	}
	var resp UnitsResponse
	if _, err := l.client.doAPI(ctx, http.MethodGet, "/units/"+url.PathEscape(unitID), nil, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SearchUnits searches or browses curated Units/Collections (`GET https://api.vocabulary.com/1.0/units`).
func (l *ListsService) SearchUnits(ctx context.Context, opts SearchUnitsOptions) (*UnitsResponse, error) {
	q := url.Values{}
	if opts.Query != "" {
		q.Set("q", opts.Query)
	}
	for _, id := range opts.IDs {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			q.Add("ids", trimmed)
		}
	}
	for _, t := range opts.Types {
		if trimmed := strings.TrimSpace(t); trimmed != "" {
			q.Add("type", trimmed)
		}
	}
	if opts.TaggedWithUnitID != "" {
		q.Set("tagged_with_unitid", opts.TaggedWithUnitID)
	}
	if opts.ContainsListID > 0 {
		q.Set("contains_wordlistid", strconv.FormatInt(opts.ContainsListID, 10))
	}
	if opts.Featured != nil {
		q.Set("featured", strconv.FormatBool(*opts.Featured))
	}
	if opts.SortBy != "" {
		q.Set("sort_by", opts.SortBy)
	}
	if opts.SortDirection != "" {
		q.Set("sort_direction", opts.SortDirection)
	}
	if opts.IncludeRelated {
		q.Set("include", "related_units")
	}
	if opts.Skip > 0 {
		q.Set("skip", strconv.Itoa(opts.Skip))
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}

	var resp UnitsResponse
	httpResp, err := l.client.doAPI(ctx, http.MethodGet, "/units", q, nil, nil, &resp)
	if err != nil {
		return nil, err
	}
	resp.TotalCount = parseTotalCountHeader(httpResp)
	if httpResp != nil {
		if links := ParseLinkHeader(httpResp.Header.Get("Link"), l.client.apiBaseURL); links != nil {
			resp.NextPageURI = links["next"]
		}
	}
	return &resp, nil
}

// GetWordSenses returns all selectable learnable senses for a word (`GET /lists/vocabgrabber/senses.json?word={word}`).
func (l *ListsService) GetWordSenses(ctx context.Context, word string) ([]LearnableSense, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, fmt.Errorf("vocab: word is required")
	}
	q := url.Values{"word": {word}}
	var senses []LearnableSense
	if err := l.client.doWebJSONGet(ctx, "/lists/vocabgrabber/senses.json", q, &senses); err != nil {
		return nil, err
	}
	return senses, nil
}

// ValidateWords checks whether words exist in the dictionary and are learnable (`POST /lists/validate.json`).
func (l *ListsService) ValidateWords(ctx context.Context, words []string) ([]ValidatedWord, bool, error) {
	form := url.Values{}
	for _, w := range words {
		if trimmed := strings.TrimSpace(w); trimmed != "" {
			form.Add("words[]", trimmed)
		}
	}
	if len(form["words[]"]) == 0 {
		return nil, true, nil
	}
	var envelope struct {
		Status int `json:"status"`
		Result struct {
			Words []ValidatedWord `json:"words"`
			Valid bool            `json:"valid"`
		} `json:"result"`
	}
	if err := l.client.doWebForm(ctx, "/lists/validate.json", form, &envelope); err != nil {
		return nil, false, err
	}
	return envelope.Result.Words, envelope.Result.Valid, nil
}

// GrabVocabulary extracts learnable vocabulary words from arbitrary text ("InstaList", `POST /lists/vocabgrabber/grab.json`).
func (l *ListsService) GrabVocabulary(ctx context.Context, rawText string) (*VocabGrabberResponse, error) {
	if strings.TrimSpace(rawText) == "" {
		return nil, fmt.Errorf("vocab: rawText is required")
	}
	form := url.Values{"text": {rawText}}
	var resp VocabGrabberResponse
	if err := l.client.doWebForm(ctx, "/lists/vocabgrabber/grab.json", form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Create creates a new vocabulary list via `POST /lists/save.json` and returns the newly created wordlist ID.
func (l *ListsService) Create(ctx context.Context, req CreateListRequest) (int64, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return 0, fmt.Errorf("vocab: list Name is required")
	}
	if req.Action == "" {
		req.Action = "create"
	}
	cleanedWords := make([]CreateListWordInput, 0, len(req.Words))
	for _, w := range req.Words {
		w.Word = strings.TrimSpace(w.Word)
		if w.Word == "" {
			continue
		}
		cleanedWords = append(cleanedWords, w)
	}
	req.Words = cleanedWords
	if len(req.Words) == 0 {
		return 0, fmt.Errorf("vocab: at least one non-empty word is required to create a list")
	}

	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}
	form := url.Values{
		"wordlist": {string(payloadBytes)},
	}
	var envelope struct {
		Result int64 `json:"result"`
	}
	if err := l.client.doWebForm(ctx, "/lists/save.json", form, &envelope); err != nil {
		return 0, err
	}
	return envelope.Result, nil
}

// AddWords appends one or more words to an existing vocabulary list (`POST /lists/save.json`).
func (l *ListsService) AddWords(ctx context.Context, listID int64, words []CreateListWordInput) (int64, error) {
	if listID <= 0 {
		return 0, fmt.Errorf("vocab: listID must be > 0")
	}
	cleaned := make([]CreateListWordInput, 0, len(words))
	for _, w := range words {
		w.Word = strings.TrimSpace(w.Word)
		if w.Word == "" {
			continue
		}
		if w.Lang == "" {
			w.Lang = "en"
		}
		cleaned = append(cleaned, w)
	}
	if len(cleaned) == 0 {
		return listID, nil
	}
	wordsJSON, err := json.Marshal(cleaned)
	if err != nil {
		return 0, err
	}
	form := url.Values{
		"id":       {strconv.FormatInt(listID, 10)},
		"addwords": {string(wordsJSON)},
	}
	var envelope struct {
		Result int64 `json:"result"`
	}
	if err := l.client.doWebForm(ctx, "/lists/save.json", form, &envelope); err != nil {
		return 0, err
	}
	return envelope.Result, nil
}

// Delete permanently deletes a user's vocabulary list (`POST /lists/delete.json`).
func (l *ListsService) Delete(ctx context.Context, listID int64) error {
	if listID <= 0 {
		return fmt.Errorf("vocab: listID must be > 0")
	}
	form := url.Values{"id": {strconv.FormatInt(listID, 10)}}
	return l.client.doWebForm(ctx, "/lists/delete.json", form, nil)
}

// SetFavorite bookmarks (`favorite=true`) or unbookmarks (`favorite=false`) a vocabulary list.
func (l *ListsService) SetFavorite(ctx context.Context, listID int64, favorite bool) error {
	if listID <= 0 {
		return fmt.Errorf("vocab: listID must be > 0")
	}
	path := "/lists/addfavorite.json"
	if !favorite {
		path = "/lists/deletefavorite.json"
	}
	form := url.Values{"wordlistid": {strconv.FormatInt(listID, 10)}}
	return l.client.doWebForm(ctx, path, form, nil)
}

// StopLearning removes a vocabulary list from the user's active learning program (`POST /lists/{listId}/stop.json`).
func (l *ListsService) StopLearning(ctx context.Context, listID int64) error {
	if listID <= 0 {
		return fmt.Errorf("vocab: listID must be > 0")
	}
	path := fmt.Sprintf("/lists/%d/stop.json", listID)
	return l.client.doWebForm(ctx, path, url.Values{}, nil)
}

// ResetPractice resets the user's Practice session on a list (`GET /lists/{listId}/reset`).
func (l *ListsService) ResetPractice(ctx context.Context, listID int64) error {
	if listID <= 0 {
		return fmt.Errorf("vocab: listID must be > 0")
	}
	path := fmt.Sprintf("/lists/%d/reset", listID)
	_, _, err := l.client.doWebGet(ctx, path, nil, "text/html,*/*")
	return err
}
