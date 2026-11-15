# CLI

## Purpose

Expose authentication, dictionary and list retrieval, and interactive practice
through a command-line consumer of the shared SDK.

Implementation: `cmd/vocab-cli/main.go`. Existing coverage: `cmd/vocab-cli/main_test.go`.

## Requirements

### Requirement: Supported command surface
The CLI SHALL provide `login`, `logout`, `me`, `lookup`, `examples`, `wod`, `list`, `search-lists`, and `play` commands through the SDK.

#### Scenario: Retrieve a dictionary word
- WHEN a caller runs `vocab-cli lookup <word>` with a successful upstream response
- THEN the CLI writes the word's dictionary result to standard output

#### Scenario: Required argument is missing
- WHEN a caller invokes a command without its required argument
- THEN the CLI reports the usage error instead of issuing an incomplete operation

### Requirement: Session and timeout options
The CLI SHALL accept a configurable session file and operation timeout and SHALL use the SDK for authentication and session persistence.

#### Scenario: Reuse a saved session
- WHEN an authenticated CLI command uses a valid session at the selected session path
- THEN it restores the saved SDK session for that invocation

### Requirement: Structured result output
The CLI SHALL support the `--json` flag for structured command results.

#### Scenario: Request a JSON lookup result
- WHEN a caller runs a successful lookup with `--json`
- THEN standard output contains the structured dictionary result encoded as JSON

### Requirement: Interactive trainer play
The CLI SHALL run interactive VocabTrainer or list practice through the shared trainer session, including choice and spelling answers and local retry feedback.

#### Scenario: Practice a selected list
- WHEN a caller starts `play` with a list ID
- THEN the CLI starts practice for that list and reads answers from its input stream
- AND the SDK owns answer evaluation, session state, and pacing
