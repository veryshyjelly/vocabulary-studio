# Design

## Context

See [proposal.md](proposal.md) for source evidence and motivation. Existing baseline specs remain the record of implemented behavior; the planned acceptance scenarios belong only to this change until implementation.

## Goals / Non-Goals

**Goals:** report partial dashboard failures with focused, deterministic coverage.

**Non-Goals:** No telemetry platform, new result limits or all-or-nothing aggregation.

## Decisions

Add safe availability metadata alongside existing values; preserve successful sections and classify empty success separately. Represent failed account verification as unavailable rather than proof of sign-out, without claiming authentication success. Offer a dashboard retry; separate section retry endpoints are not required. Do not pass raw HTTP bodies into UI diagnostics.

### Alternatives considered

A broad subsystem rewrite would expand compatibility risk without being required by the evidence. Use the smallest lifecycle/status/request-ownership change described above. Cancellation alone, ignored persistence errors, or blanket failure of otherwise usable data do not satisfy the acceptance scenarios.

## Characterization Before Changes

- Success and anonymous dashboard
- current result limits/history
- partial success retains available sections.

Add regression tests for the incorrect cases separately from characterization of behavior that must be preserved. A defect reproduction may fail before the fix; preservation tests must pass before refactoring. Use barriers, deferred promises, fake clocks, and failing stores rather than sleeps or live accounts.

## Dependencies and Sequencing

None.

Dependencies here are an implementation ordering contract; OpenSpec artifact readiness does not automatically enforce cross-change sequencing. Re-read prerequisites and current source before applying.

## Risks / Trade-offs

- Concurrent operations or partial failures may expose untested ordering → reproduce the specific scenario deterministically and keep IO/lock ordering explicit.
- Contract drift → preserve the compatibility boundaries in proposal.md and regenerate bindings instead of editing generated output.
- Over-scoping → stop at the acceptance scenarios; capture additional defects as separate changes.

## Migration Plan

Dashboard DTO, generated bindings, home UI and focused tests; additive fields preserve successful data and SDK endpoints.

Land preservation/regression tests with the implementation that satisfies them. No persistent data-format migration is planned. Revert the bounded implementation if regressions appear; do not delete user session data as a rollback technique. Reverting a security fix restores its known risk and should not be treated as remediation.

## Validation

Desktop tests, binding regeneration and frontend typecheck/build/tests

Also run `openspec validate report-partial-dashboard-failures --strict --no-interactive` and confirm all tasks are checked only after their verification succeeds.
