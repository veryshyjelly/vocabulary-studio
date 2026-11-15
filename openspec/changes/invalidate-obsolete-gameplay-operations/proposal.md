# Proposal

## Why

Quiz submission captures state then unconditionally commits after network work, so an older answer can overwrite a newer quiz. Logout clears only trainer state and skips cleanup on persistent-store Clear failure.

## What Changes

- Invalidate obsolete gameplay operations through the bounded approach in design.md.
- Add focused regression and characterization coverage before changing implementation.
- Add acceptance scenarios for the affected existing capability.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `practice`: Gameplay result ownership.

## Impact

Desktop lifecycle and auth integration. Preserve SDK scoring, pacing, request formats and existing cross-mode behavior. Regenerate bindings only if needed.

## Evidence and Coverage Gaps

- `internal/desktop/quiz_service.go:39,53` (repository-relative source locations).
- `internal/desktop/trainer_service.go:199` (repository-relative source locations).
- `internal/desktop/auth_service.go:75` (repository-relative source locations).
- `internal/desktop/bee_service.go:39` (repository-relative source locations).

Trainer first-save/local-retry semantics; Bee progression/surrender visibility; quiz completion; successful logout status/events.

The review established source-level failure paths or maintenance gaps. Existing Go tests including race runs and 13 frontend tests passed; targeted regression interleavings have not yet been added. Each task list requires those tests before the associated fix/refactor.

## Priority and Dependencies

High. Use the ownership decision from preserve-session-owner-on-store-switch; coordinate account invalidation with prevent-stale-auth-refresh-commit.

## Out of Scope

No generic gameplay framework, automatic retries of mutations, or new mode exclusivity.
