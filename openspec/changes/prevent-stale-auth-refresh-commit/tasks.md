# Tasks

## 1. Bounded implementation with characterization

Prerequisites: None. First recommended change.

- [x] 1.1 Add or retain passing characterization coverage for: Existing shared refresh; cancellation of one waiter does not abort others; failed login preserves the previous account; 401 refresh retry. Verify preservation tests pass against the pre-change behavior before refactoring; do not preserve the identified defect as an expected success.
- [x] 1.2 Add barrier-controlled refresh/logout and refresh/account-replacement regressions; assert memory, saved snapshots, and returned credentials cannot contain obsolete results.
- [x] 1.3 Implement generation-scoped acquisition and coordinated lifecycle persistence; verify new accounts cannot join old refreshes.
- [x] 1.4 Replace timing guesses in affected concurrency tests with explicit channels; verify repeated race-enabled runs.
- [x] 1.5 Document invalidation/error semantics in sdk/README.md and verify source compatibility.

## 2. Integration validation

- [x] 2.1 Run cd sdk && CGO_ENABLED=1 go test -race ./... Record results and any environment limitation; do not substitute live account requests for deterministic tests.
- [x] 2.2 Run `openspec validate prevent-stale-auth-refresh-commit --strict --no-interactive` and review the final diff against the stated non-goals; verify no unrelated application changes or secret-bearing fixtures were introduced.

## Workflow follow-up

- After implementation and review, synchronize any spec deltas and archive this change using the OpenSpec workflow. Tooling/refactor changes with skipped specs have no deltas to synchronize.
