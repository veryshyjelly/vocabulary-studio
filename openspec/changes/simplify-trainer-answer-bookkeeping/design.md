# Design

## Context

See [proposal.md](proposal.md) for source evidence and motivation. Existing baseline specs remain the record of implemented behavior; the planned acceptance scenarios belong only to this change until implementation.

## Goals / Non-Goals

**Goals:** simplify trainer answer bookkeeping with focused, deterministic coverage.

**Non-Goals:** No generic gameplay engine, naming sweep, broad file moves or changed lock model.

## Decisions

Characterize outputs and HTTP request counts, then extract small private helpers for genuinely shared state commits/reward clearing. Keep choice correctness and spelling normalization explicit. Preserve locking, exported API, upstream formats and scoring; do not assert helper internals in tests.

### Alternatives considered

Keeping both implementations avoids immediate refactor risk, but small helpers reduce drift once tests establish equivalence. A generic answer engine would hide the mode differences and is rejected.

## Characterization Before Changes

- First success saves once
- retries stay local with zero points/bonus/achievements
- canceled/failed/missing-answer responses preserve attempts
- secret rotation/hints/completion/trial-over/pacing
- choice versus spelling semantics and clone isolation.

Add regression tests for the incorrect cases separately from characterization of behavior that must be preserved. A defect reproduction may fail before the fix; preservation tests must pass before refactoring. Use barriers, deferred promises, fake clocks, and failing stores rather than sleeps or live accounts.

## Dependencies and Sequencing

Characterization passes before extraction. Schedule after lifecycle fixes settle; otherwise logically independent.

Dependencies here are an implementation ordering contract; OpenSpec artifact readiness does not automatically enforce cross-change sequencing. Re-read prerequisites and current source before applying.

## Risks / Trade-offs

- Concurrent operations or partial failures may expose untested ordering → reproduce the specific scenario deterministically and keep IO/lock ordering explicit.
- Contract drift → preserve the compatibility boundaries in proposal.md and regenerate bindings instead of editing generated output.
- Over-scoping → stop at the acceptance scenarios; capture additional defects as separate changes.

## Migration Plan

Private SDK implementation and tests only. No behavior/spec changes or public API migration.

Land preservation/regression tests with the implementation that satisfies them. No persistent data-format migration is planned. Revert the bounded implementation if regressions appear; do not delete user session data as a rollback technique. Reverting a security fix restores its known risk and should not be treated as remediation.

## Validation

cd sdk && CGO_ENABLED=1 go test -race ./...

Also run `openspec validate simplify-trainer-answer-bookkeeping --strict --no-interactive` and confirm all tasks are checked only after their verification succeeds.
