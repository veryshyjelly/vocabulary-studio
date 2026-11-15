# Dictionary

## Purpose

Explore words through structured dictionary entries, pronunciation, examples,
related words, and local lookup history.

Implementation: `sdk/dictionary.go`, `sdk/internal/htmlparse/dictionary.go`,
`internal/desktop/dictionary_service.go`, `frontend/src/features/dictionary/DictionaryView.tsx`.
Existing coverage: `sdk/integration_test.go`, `sdk/internal/htmlparse/htmlparse_test.go`,
`internal/desktop/desktop_test.go`, `frontend/src/app/App.test.tsx`.

## Requirements

### Requirement: Structured word lookup
The desktop dictionary service SHALL trim lookup input, reject empty words, and return the SDK's structured dictionary entry with available definitions, pronunciations, relations, and word family information.

#### Scenario: Look up a word
- WHEN a user looks up a nonempty word and the dictionary request succeeds
- THEN the response contains the structured entry
- AND the application supports navigating from that entry to a word family member

#### Scenario: Empty lookup
- WHEN lookup input is empty or only whitespace
- THEN the service returns a word-required error without issuing a dictionary lookup

### Requirement: Optional lookup enrichment
The desktop dictionary service SHALL preserve a successful dictionary entry when supplementary progress or example requests fail.

#### Scenario: Examples are unavailable
- WHEN the dictionary entry loads but its supplementary examples request fails
- THEN lookup still returns the entry with any other enrichment that succeeded

### Requirement: Examples and pronunciation
The application SHALL support source-filtered, paginated corpus examples and audio URL resolution for word pronunciation.

#### Scenario: Example source changes during loading
- WHEN the user selects a newer example source before an older request finishes
- THEN the dictionary view keeps the newer source's result when responses finish out of order

#### Scenario: Play a pronunciation
- WHEN a word has available audio and the user requests playback
- THEN the application plays the selected pronunciation's audio URL or resolves a fallback word audio URL

### Requirement: Local search history
The desktop SHALL record successful word lookups in local search history and allow that history to be read and cleared.

#### Scenario: Successful lookup is recorded
- WHEN a dictionary entry is successfully retrieved
- THEN its canonical word is recorded in search history

#### Scenario: Clear history
- WHEN a user clears search history
- THEN subsequent history retrieval contains no prior recorded words
