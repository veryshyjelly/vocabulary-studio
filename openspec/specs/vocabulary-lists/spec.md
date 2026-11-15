# Vocabulary Lists

## Purpose

Find and manage vocabulary lists, extract words from text, and study list content.

Implementation: `sdk/lists.go`, `internal/desktop/lists_service.go`,
`frontend/src/features/lists/ListsView.tsx`.
Existing coverage: `sdk/integration_test.go`, `internal/desktop/desktop_test.go`,
`frontend/src/app/App.test.tsx`.

## Requirements

### Requirement: Browse personal and public lists
The application SHALL retrieve a user's lists, search public lists with pagination, and retrieve list details by ID.

#### Scenario: Open a search result
- WHEN a user selects a vocabulary list from public search results
- THEN the application loads the selected list's details and words

#### Scenario: Personal list filter matches nothing
- WHEN a user's personal-list filter has no matching lists
- THEN the view presents an empty result and a way to clear the filter

### Requirement: Create and manage lists
The desktop service SHALL expose SDK operations to create lists, add words, delete lists, set favorites, stop learning a list, and reset list practice.

#### Scenario: Create a list
- WHEN a caller creates a list with its name, description, sharing setting, and words
- THEN the service returns the created list ID after upstream creation succeeds

#### Scenario: Mutation fails upstream
- WHEN a list mutation fails in the SDK
- THEN the desktop service reports the error to its caller

### Requirement: Vocabulary extraction and word validation
The desktop service SHALL expose text-based vocabulary extraction, word validation, and learnable sense lookup to prepare list content.

#### Scenario: Extract words from a passage
- WHEN a user supplies a passage to InstaList
- THEN the application requests vocabulary extraction and presents the returned candidate words

### Requirement: Flashcard study
The application SHALL retrieve flashcards for a selected list and support revealing a card's answer through pointer and keyboard interaction.

#### Scenario: Flip a focused flashcard
- WHEN a user activates a focused flashcard with the keyboard
- THEN the card flips once
- AND flashcard shortcuts do not intercept typing in a search input
