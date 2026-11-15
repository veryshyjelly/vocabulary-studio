# Proposal

## Why

Dashboard aggregation discards component errors, maps account-fetch errors to signed-out status, and returns success. Its DTO cannot distinguish unavailable information from an empty successful result.

## What Changes

- Report partial dashboard failures through the bounded approach in design.md.
- Add focused regression and characterization coverage before changing implementation.
- Add acceptance scenarios for the affected existing capability.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `learning-progress`: Explicit dashboard availability.

## Impact

Dashboard DTO, generated bindings, home UI and focused tests; additive fields preserve successful data and SDK endpoints.

## Evidence and Coverage Gaps

- `internal/desktop/progress_service.go:32,43,107` (repository-relative source locations).
- `internal/desktop/home_types.go:8` (repository-relative source locations).

Success and anonymous dashboard; current result limits/history; partial success retains available sections.

The review established source-level failure paths or maintenance gaps. Existing Go tests including race runs and 13 frontend tests passed; targeted regression interleavings have not yet been added. Each task list requires those tests before the associated fix/refactor.

## Priority and Dependencies

Medium. None.

## Out of Scope

No telemetry platform, new result limits or all-or-nothing aggregation.
