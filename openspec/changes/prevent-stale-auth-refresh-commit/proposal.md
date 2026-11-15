# Proposal

## Why

A refresh captures account credentials, outlives logout, then restores the token and GUID and flushes them. Logout must remain a local authentication boundary.

## What Changes

- Prevent stale authentication refresh commits through the bounded approach in design.md.
- Add focused regression and characterization coverage before changing implementation.
- Add acceptance scenarios for the affected existing capability.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `authentication`: Authentication generation isolation.

## Impact

SDK auth internals and tests; preserve exported signatures, failed-login rollback and session format. No bridge change expected.

## Evidence and Coverage Gaps

- `sdk/auth.go:288,326,362,420` (repository-relative source locations).

Existing shared refresh; cancellation of one waiter does not abort others; failed login preserves the previous account; 401 refresh retry.

The review established source-level failure paths or maintenance gaps. Existing Go tests including race runs and 13 frontend tests passed; targeted regression interleavings have not yet been added. Each task list requires those tests before the associated fix/refactor.

## Priority and Dependencies

High. None. First recommended change.

## Out of Scope

No storage migration or gameplay redesign.
