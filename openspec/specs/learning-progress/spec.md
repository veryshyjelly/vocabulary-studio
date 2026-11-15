# Learning Progress

## Purpose

Track word learning and study activity and bring available learning context
into the home dashboard.

Implementation: `sdk/progress.go`, `internal/desktop/progress_service.go`,
`frontend/src/features/home/HomeView.tsx`, `frontend/src/features/progress/ProgressView.tsx`.
Existing coverage: `sdk/integration_test.go`, `internal/desktop/desktop_test.go`,
`frontend/src/app/App.test.tsx`.

## Requirements

### Requirement: Home learning dashboard
The desktop SHALL compose a home dashboard from available account status, daily word, local search history, recent lists, learning words, and study activity.

#### Scenario: Open the home view
- WHEN the application loads the home dashboard
- THEN it displays available learning context and entry points to continue studying

### Requirement: Word learning controls
The desktop SHALL expose per-word progress, learning start and stop operations, and priority updates through the SDK.

#### Scenario: Start learning a word
- WHEN a caller starts learning a word and the upstream operation succeeds
- THEN the service completes successfully and the word's progress can be requested

### Requirement: Filtered and paginated progress
The desktop SHALL retrieve learning words using filter, sort, and offset parameters and retrieve activity history using its continuation cursor.

#### Scenario: Continue activity history
- WHEN a caller requests more activities with a next-page URI
- THEN the desktop uses that continuation to retrieve the next activity page

### Requirement: Study statistics and achievements
The desktop SHALL expose list progress, effort statistics, achievements, and leaderboards using the requested supported metrics and periods.

#### Scenario: View a leaderboard
- WHEN a user requests a leaderboard metric and period
- THEN the application presents the returned leaderboard entries or a loading error
