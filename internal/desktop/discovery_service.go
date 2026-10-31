package desktop

import (
	"context"
	"strings"
	"time"

	vocab "github.com/veryshyjelly/vocabulary_go/sdk"
)

// DiscoveryService exposes Word of the Day, WOD archives, Curated Units/Collections,
// and Advanced WordFinder search to the Wails frontend.
type DiscoveryService struct {
	hub *Hub
}

// GetWordOfTheDay fetches the Word of the Day for `date` (`YYYY-MM-DD`, or `""` for today).
func (d *DiscoveryService) GetWordOfTheDay(ctx context.Context, date string) (*vocab.WordOfTheDayEntry, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "discover:wod", 20*time.Second)
	defer cancel()
	return d.hub.Client().Dictionary.WordOfTheDay(ctx, date)
}

// GetWordOfTheDayArchive fetches the monthly Word of the Day archive (`YYYYMM`, e.g. `202610`).
func (d *DiscoveryService) GetWordOfTheDayArchive(ctx context.Context, startYearMonth, numMonths int) (*vocab.WODArchiveResponse, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "discover:wod-archive", 20*time.Second)
	defer cancel()
	return d.hub.Client().Dictionary.WordOfTheDayArchive(ctx, startYearMonth, numMonths)
}

// GetRootCategories returns Vocabulary.com's 9 curated root collection categories.
func (d *DiscoveryService) GetRootCategories() []CategoryCardDTO {
	return []CategoryCardDTO{
		{UnitID: vocab.UnitCategoryLiterature, Tag: "LIT", Title: "Literature", Description: "Classic novels, plays, poetry, and short stories."},
		{UnitID: vocab.UnitCategoryNonFiction, Tag: "NON", Title: "Non-Fiction", Description: "Memoirs, journalism, essays, and biographies."},
		{UnitID: vocab.UnitCategoryTestPrep, Tag: "TST", Title: "Test Prep", Description: "Essential vocabulary for SAT, ACT, GRE, and TOEFL."},
		{UnitID: vocab.UnitCategoryTextbooks, Tag: "TXT", Title: "Textbooks & Curricula", Description: "Grade-aligned ELA and academic curricula."},
		{UnitID: vocab.UnitCategoryCurrentEvents, Tag: "NWS", Title: "Current Events", Description: "Words drawn from contemporary news and culture."},
		{UnitID: vocab.UnitCategoryRootsAndAffixes, Tag: "ROOTS", Title: "Morphology & Roots", Description: "Greek and Latin roots, prefixes, and suffixes."},
		{UnitID: vocab.UnitCategoryHistoricalDocuments, Tag: "HIST", Title: "Historical Documents", Description: "Founding charters, treaties, and landmark texts."},
		{UnitID: vocab.UnitCategorySpeeches, Tag: "SPH", Title: "Speeches", Description: "Iconic rhetoric and addresses from world history."},
		{UnitID: vocab.UnitCategoryJustForFun, Tag: "FUN", Title: "Just for Fun", Description: "Witty, unusual, and thematic word collections."},
	}
}

// GetUnit fetches a single curated Unit/Collection and its sub-lists (`GET /1.0/units/{unitId}`).
func (d *DiscoveryService) GetUnit(ctx context.Context, unitID string) (*vocab.UnitsResponse, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "discover:unit", 20*time.Second)
	defer cancel()
	return d.hub.Client().Lists.GetUnit(ctx, unitID)
}

// SearchUnits searches or browses curated Units/Collections (`GET /1.0/units`).
func (d *DiscoveryService) SearchUnits(ctx context.Context, query, categoryUnitID string, skip, limit int) (*vocab.UnitsResponse, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "discover:units", 20*time.Second)
	defer cancel()
	if limit <= 0 {
		limit = 12
	}
	return d.hub.Client().Lists.SearchUnits(ctx, vocab.SearchUnitsOptions{
		Query:            strings.TrimSpace(query),
		TaggedWithUnitID: strings.TrimSpace(categoryUnitID),
		IncludeRelated:   true,
		Skip:             skip,
		Limit:            limit,
	})
}

// WordFinder searches words by pattern (`startingwith`, `endingwith`, `containinginorder`, `including`) and length.
func (d *DiscoveryService) WordFinder(ctx context.Context, searchText, searchLocation string, numLetters, maxResults int) (*vocab.WordFinderResponse, error) {
	ctx, cancel := d.hub.BeginOperation(ctx, "discover:wordfinder", 20*time.Second)
	defer cancel()
	return d.hub.Client().Dictionary.WordFinder(ctx, vocab.WordFinderOptions{
		SearchText:     strings.TrimSpace(searchText),
		SearchLocation: strings.TrimSpace(searchLocation),
		NumLetters:     numLetters,
		MaxResults:     maxResults,
	})
}
