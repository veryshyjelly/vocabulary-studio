# Proposal

## Why

ChooseIndex and Spell duplicate first-answer state commits and local-retry reward clearing. This creates maintenance drift risk; duplication alone is not evidence of incorrect behavior.

## What Changes

- Simplify trainer answer bookkeeping through the bounded approach in design.md.
- Add focused regression and characterization coverage before changing implementation.
- Preserve product behavior; this change declares `skip_specs: true`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None; behavior-preserving refactoring only.

## Impact

Private SDK implementation and tests only. No behavior/spec changes or public API migration.

## Evidence and Coverage Gaps

- `sdk/trainer_session.go:229,343,283,403` (repository-relative source locations).
- `sdk/integration_test.go:273` (repository-relative source locations).
- `sdk/trainer_test.go:96` (repository-relative source locations).

First success saves once; retries stay local with zero points/bonus/achievements; canceled/failed/missing-answer responses preserve attempts; secret rotation/hints/completion/trial-over/pacing; choice versus spelling semantics and clone isolation.

The review established source-level failure paths or maintenance gaps. Existing Go tests including race runs and 13 frontend tests passed; targeted regression interleavings have not yet been added. Each task list requires those tests before the associated fix/refactor.

## Priority and Dependencies

Low. Characterization passes before extraction. Schedule after lifecycle fixes settle; otherwise logically independent.

## Out of Scope

No generic gameplay engine, naming sweep, broad file moves or changed lock model.
