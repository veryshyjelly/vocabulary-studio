# Spec Delta

## ADDED Requirements

### Requirement: Authentication generation isolation
The SDK SHALL reject token results from a generation invalidated by logout or account replacement without returning, installing, or persisting those credentials.

#### Scenario: Refresh completes after logout
- **WHEN** a refresh captured credentials before logout and its response arrives after logout completes
- **THEN** local credentials and the selected store remain cleared; the old result is rejected

#### Scenario: Account changes during refresh
- **WHEN** an old account refresh completes after a new account is installed
- **THEN** the new credentials remain unchanged and new callers do not consume old-generation results

#### Scenario: One refresh waiter cancels
- **WHEN** one caller cancels while other callers await the same current-generation refresh
- **THEN** only the canceled waiter receives cancellation and the others can receive the valid token
