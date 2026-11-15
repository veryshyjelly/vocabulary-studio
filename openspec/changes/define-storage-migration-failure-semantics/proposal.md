# Proposal

## Why

Migration clears the old store and changes live state before persisting preferences. Preference-write failure leaves inconsistent live and durable selection; ignored old-store cleanup failure can leave credentials behind. Ordinary preference updates also mutate before saving.

## What Changes

- Define consistent storage migration failure outcomes through the bounded approach in design.md.
- Add focused regression and characterization coverage before changing implementation.
- Add acceptance scenarios for the affected existing capability.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `authentication`: Consistent storage migration outcomes.
- `desktop-experience`: publish preference changes only after persistence succeeds.

## Impact

Desktop migration, preference persistence and safe diagnostics. Preserve snapshot/preferences formats; additive bridge changes may require regeneration.

## Evidence and Coverage Gaps

- `internal/desktop/hub.go:325,329` (repository-relative source locations).
- `internal/desktop/settings_service.go:34` (repository-relative source locations).
- `internal/desktop/desktop_test.go:335` (repository-relative source locations).

Same-mode no-op; successful file/memory switches; invalid enum behavior; preference events; destination-save failure preserves old mode; unrelated preferences survive.

The review established source-level failure paths or maintenance gaps. Existing Go tests including race runs and 13 frontend tests passed; targeted regression interleavings have not yet been added. Each task list requires those tests before the associated fix/refactor.

## Priority and Dependencies

Medium. After preserve-session-owner-on-store-switch; reuse persistence diagnostics from surface-session-persistence-failures.

## Out of Scope

No distributed transaction engine or guarantee of atomic recovery across process crashes.
