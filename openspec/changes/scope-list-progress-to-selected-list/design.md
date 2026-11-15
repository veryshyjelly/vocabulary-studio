# Design

## Context

See [proposal.md](proposal.md) for source evidence and motivation. Existing baseline specs remain the record of implemented behavior; the planned acceptance scenarios belong only to this change until implementation.

## Goals / Non-Goals

**Goals:** scope progress to the selected list with focused, deterministic coverage.

**Non-Goals:** No global request framework or broad frontend loading refactor.

## Decisions

Use a local request generation plus list/account identity for details and enrichment. Invalidate on back navigation, account changes and unmount; clear old progress immediately on selection. Guard success, error and loading updates. Preserve useful details when current progress fails.

### Alternatives considered

A broad subsystem rewrite would expand compatibility risk without being required by the evidence. Use the smallest lifecycle/status/request-ownership change described above. Cancellation alone, ignored persistence errors, or blanket failure of otherwise usable data do not satisfy the acceptance scenarios.

## Characterization Before Changes

- Details/back navigation
- flashcards and keyboard behavior
- progress failure preserves valid details.

Add regression tests for the incorrect cases separately from characterization of behavior that must be preserved. A defect reproduction may fail before the fix; preservation tests must pass before refactoring. Use barriers, deferred promises, fake clocks, and failing stores rather than sleeps or live accounts.

## Dependencies and Sequencing

None; independently implementable.

Dependencies here are an implementation ordering contract; OpenSpec artifact readiness does not automatically enforce cross-change sequencing. Re-read prerequisites and current source before applying.

## Risks / Trade-offs

- Concurrent operations or partial failures may expose untested ordering → reproduce the specific scenario deterministically and keep IO/lock ordering explicit.
- Contract drift → preserve the compatibility boundaries in proposal.md and regenerate bindings instead of editing generated output.
- Over-scoping → stop at the acceptance scenarios; capture additional defects as separate changes.

## Migration Plan

List feature and UI tests only; no SDK, bridge or persistence API changes.

Land preservation/regression tests with the implementation that satisfies them. No persistent data-format migration is planned. Revert the bounded implementation if regressions appear; do not delete user session data as a rollback technique. Reverting a security fix restores its known risk and should not be treated as remediation.

## Validation

### Implementation evidence

- Local request generations guard detail, progress, and flashcard responses, including success, failure, and detail-loading completion. Account UID/authentication changes and navigation/unmount invalidate outstanding requests.
- Previously displayed progress is cleared on selection changes. A separate detail-loading flag prevents unrelated overview requests from clearing active detail loading.
- Two preservation tests passed before implementation. Six targeted stale-response regressions failed before the fix; the expanded focused suite now passes all 12 tests.
- `bun run typecheck` and the complete `bun run test` frontend suite passed. No SDK or bridge contract was changed.

cd frontend && bun run typecheck && bun run test

Also run `openspec validate scope-list-progress-to-selected-list --strict --no-interactive` and confirm all tasks are checked only after their verification succeeds.
