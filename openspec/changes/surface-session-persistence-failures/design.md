# Design

## Context

See [proposal.md](proposal.md) for source evidence and motivation. Existing baseline specs remain the record of implemented behavior; the planned acceptance scenarios belong only to this change until implementation.

## Goals / Non-Goals

**Goals:** report persistence failures separately from authentication with focused, deterministic coverage.

**Non-Goals:** No silent backend fallback; token acquisition failure is not invalid web credentials.

## Decisions

Preserve successful in-memory web authentication and expose additive safe persistence health separately from invalid credentials and API-token availability. Provide retry to the selected backend. Desktop shows an unsaved-session warning; CLI explicit persistent login reports failure on stderr and exits nonzero while preserving valid JSON stdout. Clear warning only on successful save or deliberate memory selection.

### Alternatives considered

A broad subsystem rewrite would expand compatibility risk without being required by the evidence. Use the smallest lifecycle/status/request-ownership change described above. Cancellation alone, ignored persistence errors, or blanket failure of otherwise usable data do not satisfy the acceptance scenarios.

## Characterization Before Changes

- Successful web login with token endpoint failure
- successful durable and memory-only login
- credential-free status and CLI JSON.

Add regression tests for the incorrect cases separately from characterization of behavior that must be preserved. A defect reproduction may fail before the fix; preservation tests must pass before refactoring. Use barriers, deferred promises, fake clocks, and failing stores rather than sleeps or live accounts.

## Dependencies and Sequencing

Independent; coordinate status/error vocabulary with define-storage-migration-failure-semantics.

Dependencies here are an implementation ordering contract; OpenSpec artifact readiness does not automatically enforce cross-change sequencing. Re-read prerequisites and current source before applying.

## Risks / Trade-offs

- Concurrent operations or partial failures may expose untested ordering → reproduce the specific scenario deterministically and keep IO/lock ordering explicit.
- Contract drift → preserve the compatibility boundaries in proposal.md and regenerate bindings instead of editing generated output.
- Over-scoping → stop at the acceptance scenarios; capture additional defects as separate changes.

## Migration Plan

SDK diagnostics, CLI exit/reporting, desktop safe DTOs and UI. Preserve SDK source compatibility and session format; regenerate bindings for additive bridge changes.

Land preservation/regression tests with the implementation that satisfies them. No persistent data-format migration is planned. Revert the bounded implementation if regressions appear; do not delete user session data as a rollback technique. Reverting a security fix restores its known risk and should not be treated as remediation.

## Validation

Run focused tests appropriate to the affected layers and preserve the characterized behavior.

Run `openspec validate surface-session-persistence-failures --strict --no-interactive` before implementation is considered complete.
