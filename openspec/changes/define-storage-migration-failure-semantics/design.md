# Design

## Context

See [proposal.md](proposal.md) for source evidence and motivation. Existing baseline specs remain the record of implemented behavior; the planned acceptance scenarios belong only to this change until implementation.

## Goals / Non-Goals

**Goals:** define consistent storage migration failure outcomes with focused, deterministic coverage.

**Non-Goals:** No distributed transaction engine or guarantee of atomic recovery across process crashes.

## Decisions

Serialize migrations. Stage destination and preference data before publishing selection; before commit retain old ownership on failure and report failed cleanup of staged data. After durable preference commit publish new mode, then clean old store; cleanup failure is partial success with retryable warning. Save candidate ordinary preferences before publishing and emitting settings:changed. Avoid holding general Hub mutex across external IO.

### Alternatives considered

A broad subsystem rewrite would expand compatibility risk without being required by the evidence. Use the smallest lifecycle/status/request-ownership change described above. Cancellation alone, ignored persistence errors, or blanket failure of otherwise usable data do not satisfy the acceptance scenarios.

## Characterization Before Changes

- Same-mode no-op
- successful file/memory switches
- invalid enum behavior
- preference events
- destination-save failure preserves old mode
- unrelated preferences survive.

Add regression tests for the incorrect cases separately from characterization of behavior that must be preserved. A defect reproduction may fail before the fix; preservation tests must pass before refactoring. Use barriers, deferred promises, fake clocks, and failing stores rather than sleeps or live accounts.

## Dependencies and Sequencing

After preserve-session-owner-on-store-switch; reuse persistence diagnostics from surface-session-persistence-failures.

Dependencies here are an implementation ordering contract; OpenSpec artifact readiness does not automatically enforce cross-change sequencing. Re-read prerequisites and current source before applying.

## Risks / Trade-offs

- Concurrent operations or partial failures may expose untested ordering → reproduce the specific scenario deterministically and keep IO/lock ordering explicit.
- Contract drift → preserve the compatibility boundaries in proposal.md and regenerate bindings instead of editing generated output.
- Over-scoping → stop at the acceptance scenarios; capture additional defects as separate changes.
- Cross-backend crash atomicity is unavailable → document pre-commit failure versus post-commit cleanup warning; do not claim distributed transactional guarantees.

## Migration Plan

Desktop migration, preference persistence and safe diagnostics. Preserve snapshot/preferences formats; additive bridge changes may require regeneration.

Land preservation/regression tests with the implementation that satisfies them. No persistent data-format migration is planned. Revert the bounded implementation if regressions appear; do not delete user session data as a rollback technique. Reverting a security fix restores its known risk and should not be treated as remediation.

## Validation

Desktop race tests, SDK persistence tests and settings UI tests

Also run `openspec validate define-storage-migration-failure-semantics --strict --no-interactive` and confirm all tasks are checked only after their verification succeeds.
