# Tasks

## 1. Bounded implementation with characterization

Prerequisites: Use the ownership decision from preserve-session-owner-on-store-switch; coordinate account invalidation with prevent-stale-auth-refresh-commit.

- [ ] 1.1 Add or retain passing characterization coverage for: Trainer first-save/local-retry semantics; Bee progression/surrender visibility; quiz completion; successful logout status/events. Verify preservation tests pass against the pre-change behavior before refactoring; do not preserve the identified defect as an expected success.
- [ ] 1.2 Add deterministic quiz-submit/new-quiz and trainer-start/logout tests; verify obsolete results never commit.
- [ ] 1.3 Add logout tests for active trainer, Bee and quiz with failing store Clear; verify all local modes become inactive.
- [ ] 1.4 Implement ownership checks and immutable input snapshots; verify race tests and no mutable Bee request inputs are read outside their lock.
- [ ] 1.5 Test safe logout cleanup-failure UI reporting and document lifecycle semantics; regenerate bindings if contracts change.

## 2. Integration validation

- [ ] 2.1 Run SDK and desktop race tests plus frontend practice/auth tests Record results and any environment limitation; do not substitute live account requests for deterministic tests.
- [ ] 2.2 Run `openspec validate invalidate-obsolete-gameplay-operations --strict --no-interactive` and review the final diff against the stated non-goals; verify no unrelated application changes or secret-bearing fixtures were introduced.

## Workflow follow-up

- After implementation and review, synchronize any spec deltas and archive this change using the OpenSpec workflow. Tooling/refactor changes with skipped specs have no deltas to synchronize.
