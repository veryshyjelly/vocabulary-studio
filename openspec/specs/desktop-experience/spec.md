# Desktop Experience

## Purpose

Provide a native desktop shell with consistent navigation, persisted preferences,
keyboard interaction, and safe presentation of upstream content.

Implementation: `cmd/vocab-desktop/main.go`, `internal/desktop/hub.go`,
`internal/desktop/settings_service.go`, `internal/desktop/richtext.go`,
`frontend/src/app/App.tsx`, `frontend/src/lib/bridge.ts`, `frontend/src/components/ui/dialog.tsx`.
Existing coverage: `internal/desktop/desktop_test.go`, `frontend/src/app/App.test.tsx`.

## Requirements

### Requirement: Application navigation and startup
The desktop SHALL provide Home, Practice, Dictionary, My Lists, Discover, Progress, and Settings views and SHALL load preferences and attempt session restoration at startup.

#### Scenario: Navigate to dictionary lookup
- WHEN the user invokes the platform modifier plus K shortcut outside an open modal
- THEN the application opens the dictionary view and focuses lookup

### Requirement: Persisted valid preferences
The desktop SHALL persist supported light, dark, or system appearance, US or UK pronunciation, default view, and session storage settings and SHALL retain current enum preferences when unsupported values are supplied.

#### Scenario: Update appearance
- WHEN a user saves a supported theme preference
- THEN the service persists it and emits `settings:changed`
- AND the application applies the selected appearance

#### Scenario: Invalid theme value
- WHEN a preference update contains an unsupported theme
- THEN the existing theme remains selected

### Requirement: Keyboard and modal interaction
The application SHALL support practice answer and advance shortcuts, keyboard tool-tab navigation, and modal focus restoration while preventing plain shortcuts from interfering with text entry.

#### Scenario: Dismiss sign-in
- WHEN a user dismisses the sign-in modal with Escape
- THEN focus returns to the control that opened the modal
- AND reopening sign-in does not retain the entered password

#### Scenario: Change a tool tab with arrow keys
- WHEN a focused tool tab receives the supported arrow-key navigation
- THEN selection moves to the adjacent tab and the selected tab remains in the tab order

### Requirement: Safe upstream rich text
Desktop adapters SHALL convert upstream HTML into plain text and structured safe spans, and the frontend SHALL render those spans without injecting upstream HTML.

#### Scenario: Upstream text contains unsafe markup
- WHEN a desktop adapter processes rich text containing scripts or unsafe links
- THEN the UI receives sanitized text and permitted span data without executable markup

### Requirement: Superseded operations are canceled
The desktop operation manager SHALL cancel an earlier operation when a new operation starts with the same key and SHALL apply operation timeouts.

#### Scenario: A newer lookup supersedes an earlier lookup
- WHEN a second dictionary lookup starts while the previous lookup is active
- THEN the previous lookup's context is canceled

### Requirement: Bundled production assets
The native desktop build SHALL embed production frontend assets and bundled fonts so an installed application can open without a frontend development server.

#### Scenario: Launch an installed build
- WHEN a user launches a packaged application
- THEN its interface loads from bundled assets without requiring Go, Bun, or a Vite server on the user's machine
