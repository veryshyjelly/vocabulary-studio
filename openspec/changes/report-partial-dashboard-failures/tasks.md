# Tasks

## 1. Bounded implementation with characterization

Prerequisites: None.

- [ ] 1.1 Add or retain passing characterization coverage for: Success and anonymous dashboard; current result limits/history; partial success retains available sections. Verify preservation tests pass against the pre-change behavior before refactoring; do not preserve the identified defect as an expected success.
- [ ] 1.2 Add aggregation tests for empty success, one/all section failures and account verification outage.
- [ ] 1.3 Extend safe availability metadata and mapping; verify no raw session/upstream-sensitive data appears.
- [ ] 1.4 Regenerate bindings and render partial failure/retry feedback; test recovery and retained successful sections.
- [ ] 1.5 Document availability semantics and verify existing consumers compile.

## 2. Integration validation

- [ ] 2.1 Run Desktop tests, binding regeneration and frontend typecheck/build/tests Record results and any environment limitation; do not substitute live account requests for deterministic tests.
- [ ] 2.2 Run `openspec validate report-partial-dashboard-failures --strict --no-interactive` and review the final diff against the stated non-goals; verify no unrelated application changes or secret-bearing fixtures were introduced.

## Workflow follow-up

- After implementation and review, synchronize any spec deltas and archive this change using the OpenSpec workflow. Tooling/refactor changes with skipped specs have no deltas to synchronize.
