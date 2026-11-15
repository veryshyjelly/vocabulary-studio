# Discovery

## Purpose

Discover words and study material through daily entries, curated units, and word patterns.

Implementation: `sdk/dictionary.go`, `sdk/lists.go`,
`internal/desktop/discovery_service.go`, `frontend/src/features/discovery/DiscoverView.tsx`.
Existing coverage: `sdk/integration_test.go`, `internal/desktop/desktop_test.go`.

## Requirements

### Requirement: Word of the Day and archive
The desktop SHALL retrieve Word of the Day for the current or requested date and retrieve its archive for a requested month range.

#### Scenario: Open a dated daily entry
- WHEN a user requests Word of the Day for a supported date
- THEN the service returns the upstream daily entry for that request

#### Scenario: Daily entry request fails
- WHEN the SDK cannot retrieve the daily entry
- THEN the desktop service reports the error to its caller

### Requirement: Curated collection discovery
The desktop SHALL expose root collection categories, unit details, and paginated unit search optionally scoped to a category.

#### Scenario: Search within a category
- WHEN a user submits a collection query with a category unit ID
- THEN the desktop requests matching units within that category using the requested pagination

### Requirement: WordFinder search
The desktop SHALL expose WordFinder searches using the supplied text, search location, letter count, and result limit.

#### Scenario: Find matching words
- WHEN a caller provides WordFinder search criteria
- THEN the desktop returns the SDK's matching word results or its error
