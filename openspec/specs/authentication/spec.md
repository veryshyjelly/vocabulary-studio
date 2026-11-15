# Authentication

## Purpose

Authenticate Vocabulary.com accounts and preserve sessions using an explicitly
selected storage backend without exposing credentials to the frontend.

Implementation: `sdk/auth.go`, `sdk/session_store.go`,
`internal/desktop/auth_service.go`, `internal/desktop/hub.go`.
Existing coverage: `sdk/auth_test.go`, `sdk/session_store_test.go`,
`internal/desktop/desktop_test.go`, `frontend/src/app/App.test.tsx`.

## Requirements

### Requirement: Account sign-in and safe status
The desktop authentication service SHALL sign in through the SDK and return account status without passwords, session cookies, or bearer tokens.

#### Scenario: Successful sign-in
- WHEN a user supplies valid Vocabulary.com credentials
- THEN the service returns authenticated account information
- AND it emits `auth:changed` with safe account status

#### Scenario: Invalid credentials
- WHEN Vocabulary.com rejects the submitted credentials
- THEN sign-in reports an authentication error
- AND the SDK does not commit the rejected login as a new authenticated session

### Requirement: Session restoration and logout
The desktop service SHALL restore saved sessions when possible, treat failed restoration as unauthenticated status, and clear local authentication on logout.

#### Scenario: Saved session is unavailable
- WHEN session loading or restored account verification fails
- THEN restoration returns unauthenticated status without preventing the application from opening

#### Scenario: User signs out
- WHEN the user logs out and clearing the selected session store succeeds
- THEN the SDK clears its local authentication and selected saved session
- AND the desktop returns unauthenticated account status

### Requirement: Explicit session storage selection
The application SHALL support keyring, restricted file, and memory session stores and SHALL report storage-switch persistence failures without silently selecting a different backend.

#### Scenario: Destination storage cannot save an active session
- GIVEN an authenticated client using one session store
- WHEN switching stores fails while saving the session to the destination
- THEN the switch returns an error and keeps the original active client and storage mode

#### Scenario: Memory-only storage
- WHEN a user chooses memory session storage
- THEN session persistence uses process memory instead of a session file or OS keychain

### Requirement: Restricted file session persistence
The file session store SHALL save snapshots atomically with `0600` file permissions and SHALL identify malformed saved session data as corrupted.

#### Scenario: Save a file session
- WHEN a session snapshot is saved to a file store
- THEN the resulting session file has `0600` permissions
- AND loading it recovers the saved session data

#### Scenario: Saved file contains invalid data
- WHEN the file store loads malformed session JSON
- THEN it returns an error matching `ErrCorruptedSession`
