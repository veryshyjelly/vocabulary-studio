# Proposal

## Why

Login and token refresh discard FlushSession errors, so successful sign-in can conceal failure to save the selected persistent session. Users cannot tell whether a restart will restore their account.

## What Changes

- Report persistence failures separately from authentication through the bounded approach in design.md.
- Add focused regression and characterization coverage before changing implementation.
- Add acceptance scenarios for the affected existing capability.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `authentication`: Observable persistence failure and recovery.

## Impact

SDK diagnostics, CLI exit/reporting, desktop safe DTOs and UI. Preserve SDK source compatibility and session format; regenerate bindings for additive bridge changes.

## Evidence and Coverage Gaps

- `sdk/auth.go:181,251,427` (repository-relative source locations).
- `internal/desktop/auth_service.go:61` (repository-relative source locations).

Successful web login with token endpoint failure; successful durable and memory-only login; credential-free status and CLI JSON.

The review established source-level failure paths or maintenance gaps. Existing Go tests including race runs and 13 frontend tests passed; targeted regression interleavings have not yet been added. Each task list requires those tests before the associated fix/refactor.

## Priority and Dependencies

High. Independent; coordinate status/error vocabulary with define-storage-migration-failure-semantics.

## Out of Scope

No silent backend fallback; token acquisition failure is not invalid web credentials.
