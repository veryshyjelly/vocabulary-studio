package desktop

import (
	"context"
	"time"

	vocab "github.com/veryshyjelly/vocabulary_go/sdk"
)

// ListsService exposes vocabulary list browsing, search, flashcards, creation, editing,
// and InstaList text extraction to the Wails frontend.
type ListsService struct {
	hub *Hub
}

// GetUserLists fetches the authenticated user's lists (`display` = `"created"` or `""`).
func (l *ListsService) GetUserLists(ctx context.Context, display string) ([]vocab.ProfileListSummary, error) {
	ctx, cancel := l.hub.BeginOperation(ctx, "lists:user", 20*time.Second)
	defer cancel()
	return l.hub.Client().Lists.ListUserLists(ctx, display, "activitydate", 100)
}

// SearchLists searches public vocabulary lists (`GET /lists/smart-search.json`).
func (l *ListsService) SearchLists(ctx context.Context, query string, skip, limit int) (*vocab.SmartSearchResponse, error) {
	ctx, cancel := l.hub.BeginOperation(ctx, "lists:search", 20*time.Second)
	defer cancel()
	return l.hub.Client().Lists.Search(ctx, query, skip, limit, false)
}

// GetListDetail loads a full vocabulary list by ID (`POST /lists/load.json`).
func (l *ListsService) GetListDetail(ctx context.Context, listID int64) (*vocab.VocabularyList, error) {
	ctx, cancel := l.hub.BeginOperation(ctx, "lists:detail", 20*time.Second)
	defer cancel()
	return l.hub.Client().Lists.Get(ctx, listID)
}

// GetFlashcards fetches the rich flashcard representation of a list (`GET /1.0/lists/{listId}`).
func (l *ListsService) GetFlashcards(ctx context.Context, listID int64) (*vocab.FlashcardListResponse, error) {
	ctx, cancel := l.hub.BeginOperation(ctx, "lists:flashcards", 20*time.Second)
	defer cancel()
	return l.hub.Client().Lists.GetFlashcardList(ctx, listID)
}

// CreateList creates a new vocabulary list and returns its wordlist ID.
func (l *ListsService) CreateList(ctx context.Context, name, description string, shared bool, words []vocab.CreateListWordInput) (int64, error) {
	return l.hub.Client().Lists.Create(ctx, vocab.CreateListRequest{
		Name:        name,
		Description: description,
		Action:      "create",
		Shared:      shared,
		Words:       words,
	})
}

// AddWordsToList appends words to an existing vocabulary list.
func (l *ListsService) AddWordsToList(ctx context.Context, listID int64, words []vocab.CreateListWordInput) (int64, error) {
	return l.hub.Client().Lists.AddWords(ctx, listID, words)
}

// DeleteList permanently deletes a user's vocabulary list.
func (l *ListsService) DeleteList(ctx context.Context, listID int64) error {
	return l.hub.Client().Lists.Delete(ctx, listID)
}

// SetFavorite bookmarks or unbookmarks a vocabulary list.
func (l *ListsService) SetFavorite(ctx context.Context, listID int64, favorite bool) error {
	return l.hub.Client().Lists.SetFavorite(ctx, listID, favorite)
}

// StopLearningList removes a list from the user's active learning program.
func (l *ListsService) StopLearningList(ctx context.Context, listID int64) error {
	return l.hub.Client().Lists.StopLearning(ctx, listID)
}

// ResetListPractice resets the user's Practice session on a list.
func (l *ListsService) ResetListPractice(ctx context.Context, listID int64) error {
	return l.hub.Client().Lists.ResetPractice(ctx, listID)
}

// ValidateWords checks whether words exist in the dictionary and are learnable.
func (l *ListsService) ValidateWords(ctx context.Context, words []string) ([]vocab.ValidatedWord, error) {
	validated, _, err := l.hub.Client().Lists.ValidateWords(ctx, words)
	return validated, err
}

// GrabVocabulary extracts learnable vocabulary words from pasted text ("InstaList").
func (l *ListsService) GrabVocabulary(ctx context.Context, rawText string) (*vocab.VocabGrabberResponse, error) {
	return l.hub.Client().Lists.GrabVocabulary(ctx, rawText)
}

// GetWordSenses returns selectable learnable senses for a word when editing a list.
func (l *ListsService) GetWordSenses(ctx context.Context, word string) ([]vocab.LearnableSense, error) {
	return l.hub.Client().Lists.GetWordSenses(ctx, word)
}
