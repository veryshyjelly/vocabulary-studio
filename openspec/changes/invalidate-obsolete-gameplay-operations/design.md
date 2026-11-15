# Design

## Context

See [proposal.md](proposal.md) for source evidence and motivation. Existing baseline specs remain the record of implemented behavior; the planned acceptance scenarios belong only to this change until implementation.

## Goals / Non-Goals

**Goals:** invalidate obsolete gameplay operations with focused, deterministic coverage.

**Non-Goals:** No generic gameplay framework, automatic retries of mutations, or new mode exclusivity.

## Decisions

Associate work with account and per-mode session identities. Snapshot immutable request inputs under mode locks, do network work outside locks, then check identity before committing. Invalidate before canceling; cancellation alone does not guard completed responses. Clear all local modes even when logout persistence cleanup fails; report that failure separately.

### Alternatives considered

A broad subsystem rewrite would expand compatibility risk without being required by the evidence. Use the smallest lifecycle/status/request-ownership change described above. Cancellation alone, ignored persistence errors, or blanket failure of otherwise usable data do not satisfy the acceptance scenarios.

## Characterization Before Changes

- Trainer first-save/local-retry semantics
- Bee progression/surrender visibility
- quiz completion
- successful logout status/events.

Add regression tests for the incorrect cases separately from characterization of behavior that must be preserved. A defect reproduction may fail before the fix; preservation tests must pass before refactoring. Use barriers, deferred promises, fake clocks, and failing stores rather than sleeps or live accounts.

## Dependencies and Sequencing

Use the ownership decision from preserve-session-owner-on-store-switch; coordinate account invalidation with prevent-stale-auth-refresh-commit.

Dependencies here are an implementation ordering contract; OpenSpec artifact readiness does not automatically enforce cross-change sequencing. Re-read prerequisites and current source before applying.

## Risks / Trade-offs

- Concurrent operations or partial failures may expose untested ordering → reproduce the specific scenario deterministically and keep IO/lock ordering explicit.
- Contract drift → preserve the compatibility boundaries in proposal.md and regenerate bindings instead of editing generated output.
- Over-scoping → stop at the acceptance scenarios; capture additional defects as separate changes.
- The upstream server may accept an obsolete answer → discard its local result without automatic mutation retry or claims of remote rollback.

## Migration Plan

Desktop lifecycle and auth integration. Preserve SDK scoring, pacing, request formats and existing cross-mode behavior. Regenerate bindings only if needed.

Land preservation/regression tests with the implementation that satisfies them. No persistent data-format migration is planned. Revert the bounded implementation if regressions appear; do not delete user session data as a rollback technique. Reverting a security fix restores its known risk and should not be treated as remediation.

## Validation

SDK and desktop race tests plus frontend practice/auth tests

Also run `openspec validate invalidate-obsolete-gameplay-operations --strict --no-interactive` and confirm all tasks are checked only after their verification succeeds.
