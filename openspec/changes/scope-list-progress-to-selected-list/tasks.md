# Tasks

## 1. Bounded implementation with characterization

Prerequisites: None; independently implementable.

- [x] 1.1 Add or retain passing characterization coverage for: Details/back navigation; flashcards and keyboard behavior; progress failure preserves valid details. Verify preservation tests pass against the pre-change behavior before refactoring; do not preserve the identified defect as an expected success.
- [x] 1.2 Add deferred-promise A/B reverse-order, stale-rejection and leave-details tests; verify only current-list data is shown.
- [x] 1.3 Implement local request ownership and clearing; test current failures remain visible while stale completions are ignored.
- [x] 1.4 Add account-change/unmount tests and retain list/flashcard regression coverage; verify no bridge change.

## 2. Integration validation

- [x] 2.1 Run cd frontend && bun run typecheck && bun run test Record results and any environment limitation; do not substitute live account requests for deterministic tests.
- [x] 2.2 Run `openspec validate scope-list-progress-to-selected-list --strict --no-interactive` and review the final diff against the stated non-goals; verify no unrelated application changes or secret-bearing fixtures were introduced.

## Workflow follow-up

- After implementation and review, synchronize any spec deltas and archive this change using the OpenSpec workflow. Tooling/refactor changes with skipped specs have no deltas to synchronize.
