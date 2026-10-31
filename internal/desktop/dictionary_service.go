package desktop

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	vocab "github.com/veryshyjelly/vocabulary_go/sdk"
)

// DictionaryService exposes dictionary lookup, debounced autocomplete, corpus examples,
// audio resolution, and search history to the Wails frontend.
type DictionaryService struct {
	hub *Hub
}

// Lookup fetches the full dictionary entry, user mastery progress, and corpus usage examples for `word`.
// Previous in-flight Lookup calls are automatically canceled.
func (d *DictionaryService) Lookup(ctx context.Context, word string) (*WordDetailDTO, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "dict:lookup", 25*time.Second)
	defer cancel()

	word = strings.TrimSpace(word)
	if word == "" {
		return nil, fmt.Errorf("vocab: word is required")
	}

	client := d.hub.Client()
	entry, err := client.Dictionary.Lookup(ctx, word)
	if err != nil {
		return nil, err
	}

	d.hub.recordSearchWord(entry.Word)

	var (
		wg       sync.WaitGroup
		progress *vocab.WordProgressDetail
		examples []CorpusSentenceDTO
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		if wp, pErr := client.Progress.GetWordProgress(ctx, entry.Word, true); pErr == nil {
			progress = wp
		}
	}()
	go func() {
		defer wg.Done()
		if ex, eErr := client.Dictionary.Examples(ctx, entry.Word, vocab.CorpusDomainAll, 0, 8); eErr == nil && ex != nil {
			examples = mapCorpusSentences(ex.Sentences)
		}
	}()
	wg.Wait()

	return &WordDetailDTO{
		Entry:    entry,
		Progress: progress,
		Examples: examples,
	}, nil
}

// Autocomplete returns word suggestions for `query`, canceling any obsolete in-flight autocomplete request.
func (d *DictionaryService) Autocomplete(ctx context.Context, query string, maxResults int) ([]vocab.AutocompleteSuggestion, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "dict:autocomplete", 10*time.Second)
	defer cancel()

	return d.hub.Client().Dictionary.Autocomplete(ctx, query, maxResults)
}

// GetExamples fetches real-world usage sentences ("Word in the Wild") with optional domain filtering
// ("" All, "F" Fiction, "A" Arts, "N" News, "B" Business, "S" Sports, "M" Science/Med, "T" Tech).
func (d *DictionaryService) GetExamples(ctx context.Context, word, domain string, offset, limit int) ([]CorpusSentenceDTO, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "dict:examples", 20*time.Second)
	defer cancel()

	resp, err := d.hub.Client().Dictionary.Examples(ctx, word, vocab.CorpusDomain(strings.TrimSpace(domain)), offset, limit)
	if err != nil {
		return nil, err
	}
	return mapCorpusSentences(resp.Sentences), nil
}

// GetRandomWord fetches a random word from Vocabulary.com's dictionary.
func (d *DictionaryService) GetRandomWord(ctx context.Context) (*WordDetailDTO, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "dict:random", 25*time.Second)
	defer cancel()

	entry, err := d.hub.Client().Dictionary.RandomWord(ctx)
	if err != nil {
		return nil, err
	}
	d.hub.recordSearchWord(entry.Word)
	return &WordDetailDTO{Entry: entry}, nil
}

// ResolveAudioURL resolves the MP3 pronunciation stream URL for `word`.
func (d *DictionaryService) ResolveAudioURL(ctx context.Context, word string) (string, error) {
	return d.hub.Client().Dictionary.ResolveWordAudioURL(ctx, word, 0)
}

// GetSearchHistory returns the user's recent dictionary lookups.
func (d *DictionaryService) GetSearchHistory() []string {
	d.hub.mu.RLock()
	defer d.hub.mu.RUnlock()
	out := make([]string, len(d.hub.prefs.SearchHistory))
	copy(out, d.hub.prefs.SearchHistory)
	return out
}

// ClearSearchHistory clears the saved dictionary lookup history.
func (d *DictionaryService) ClearSearchHistory() error {
	d.hub.mu.Lock()
	defer d.hub.mu.Unlock()
	d.hub.prefs.SearchHistory = []string{}
	return d.hub.savePreferencesLocked()
}

func mapCorpusSentences(sentences []vocab.CorpusSentence) []CorpusSentenceDTO {
	out := make([]CorpusSentenceDTO, 0, len(sentences))
	for _, s := range sentences {
		dto := CorpusSentenceDTO{
			Sentence:  s.Sentence,
			Offsets:   s.Offsets,
			CorpusID:  s.CorpusID,
			SourceURL: s.SourceURL(),
		}
		if s.Volume != nil {
			dto.SourceTitle = s.Volume.Title
			dto.Author = s.Volume.Author
			if s.Volume.Corpus != nil {
				dto.SourceName = s.Volume.Corpus.Name
			}
		}
		out = append(out, dto)
	}
	return out
}
