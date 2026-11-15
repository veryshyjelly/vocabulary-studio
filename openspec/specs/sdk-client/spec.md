# SDK Client

## Purpose

Provide an independent Vocabulary.com client with shared authentication,
transport safeguards, and gameplay pacing for desktop, CLI, and library consumers.

Implementation: `sdk/go.mod`, `sdk/client.go`, `sdk/options.go`, `sdk/auth.go`,
`sdk/pacing.go`. Existing coverage: `sdk/client_test.go`, `sdk/auth_test.go`,
`sdk/pacing_test.go`, `sdk/fixtures_test.go`.

## Requirements

### Requirement: Independent library module
The SDK SHALL be consumable as `github.com/veryshyjelly/vocabulary-studio/sdk` without desktop, Wails, frontend, or CGO dependencies.

#### Scenario: Use the SDK independently
- WHEN a Go application imports the SDK module and creates a client
- THEN it can access SDK services without building the desktop application or frontend assets

### Requirement: Configurable transport with bounded responses
The SDK SHALL accept caller contexts and configurable transport, base URLs, clocks, and response size limits, and SHALL report transport and response decoding failures to the caller.

#### Scenario: Response exceeds the configured limit
- WHEN an upstream response body exceeds the client's configured maximum size
- THEN the operation fails with an error matching `ErrResponseTooLarge`

#### Scenario: Request is canceled
- WHEN a caller cancels the context for an SDK request
- THEN the request terminates with the cancellation error

### Requirement: Shared bearer token refresh
The SDK SHALL reuse valid bearer tokens, deduplicate concurrent token acquisition, and retry an API request once after refreshing a token rejected with HTTP 401 or 403.

#### Scenario: Concurrent callers require a token
- WHEN multiple API requests need a missing or expired token at the same time
- THEN they share token acquisition rather than independently refreshing for each caller

#### Scenario: Refreshed token is also rejected
- WHEN an API request and its refreshed-token retry both receive an authentication rejection
- THEN the SDK returns the authentication error without an unbounded retry loop

### Requirement: SDK-owned gameplay pacing
The SDK SHALL enforce its configured answer timing and burst controls for gameplay submissions through its shared pacing guard.

#### Scenario: A quiz answer is submitted immediately
- WHEN a caller submits an answer before the quiz's minimum submission interval has elapsed
- THEN the SDK waits for the required interval before sending the answer
- AND callers do not need to duplicate the pacing delay in the desktop or CLI

### Requirement: Redirect credential protection
The default SDK redirect policy SHALL remove sensitive authentication headers when a request redirects outside trusted Vocabulary.com or configured hosts.

#### Scenario: Redirect leaves a trusted host
- WHEN the default HTTP client follows a redirect to an untrusted host
- THEN it strips authorization and cookie headers from the redirected request
