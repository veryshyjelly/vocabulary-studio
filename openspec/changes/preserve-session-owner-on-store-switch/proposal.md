# Proposal

## Why

Storage switching replaces the Hub client while active TrainerSession objects retain the old service/client. Continuing practice can refresh and persist through the previous backend, recreating a cleared session file after switching to memory.

## What Changes

- Preserve session ownership during storage changes through the bounded approach in design.md.
- Add focused regression and characterization coverage before changing implementation.
- Add acceptance scenarios for the affected existing capability.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `authentication`: Storage switches preserve active session ownership.

## Impact

SDK store ownership and desktop switching; retain active trainer state and snapshot format. Prefer an additive SDK operation without Wails signature changes.

## Evidence and Coverage Gaps

- `internal/desktop/hub.go:317` (repository-relative source locations).
- `sdk/trainer_session.go:70,217` (repository-relative source locations).
- `internal/desktop/desktop_test.go:335` (repository-relative source locations).

Practice state, attempts and feedback across a successful switch; same-mode no-op; failed destination Save leaves old mode usable.

The review established source-level failure paths or maintenance gaps. Existing Go tests including race runs and 13 frontend tests passed; targeted regression interleavings have not yet been added. Each task list requires those tests before the associated fix/refactor.

## Priority and Dependencies

High. Coordinate with prevent-stale-auth-refresh-commit and reuse its lifecycle synchronization. Establishes ownership for migration failure handling.

## Out of Scope

Do not end practice as a shortcut or solve every migration failure case here.
