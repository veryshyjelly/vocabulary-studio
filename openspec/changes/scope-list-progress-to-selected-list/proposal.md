# Proposal

## Why

List progress loads separately and unconditionally updates state. Delayed list A progress can overwrite list B progress, and previous progress is not cleared when selecting a different list.

## What Changes

- Scope progress to the selected list through the bounded approach in design.md.
- Add focused regression and characterization coverage before changing implementation.
- Add acceptance scenarios for the affected existing capability.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `vocabulary-lists`: List enrichment ownership.

## Impact

List feature and UI tests only; no SDK, bridge or persistence API changes.

## Evidence and Coverage Gaps

- `frontend/src/features/lists/ListsView.tsx:89,102` (repository-relative source locations).
- `internal/desktop/progress_service.go:139` (repository-relative source locations).

Details/back navigation; flashcards and keyboard behavior; progress failure preserves valid details.

The review established source-level failure paths or maintenance gaps. Existing Go tests including race runs and 13 frontend tests passed; targeted regression interleavings have not yet been added. Each task list requires those tests before the associated fix/refactor.

## Priority and Dependencies

Medium. None; independently implementable.

## Out of Scope

No global request framework or broad frontend loading refactor.
