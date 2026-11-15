# Tasks

## 1. Bounded implementation with characterization

Prerequisites: Independent; coordinate status/error vocabulary with define-storage-migration-failure-semantics.

- [x] 1.1 Add or retain passing characterization coverage for: Successful web login with token endpoint failure; successful durable and memory-only login; credential-free status and CLI JSON. Verify preservation tests pass against the pre-change behavior before refactoring; do not preserve the identified defect as an expected success.
- [x] 1.2 Inject store Save failures during login and refresh; verify inspectable persistence health while authenticated memory state remains valid.
- [x] 1.3 Implement additive SDK persistence health and retry contract; test warning recovery without exposing raw sensitive errors.
- [x] 1.4 Map status into desktop feedback and CLI stderr/exit handling; add partial-success and retry tests.
- [x] 1.5 Regenerate bindings for changed DTOs/services and update SDK/CLI docs; verify frontend typecheck/build and JSON output tests.

## 2. Integration validation

- [x] 2.1 Run SDK, desktop/CLI tests and frontend typecheck/build/tests Record results and any environment limitation; do not substitute live account requests for deterministic tests.
- [x] 2.2 Run `openspec validate surface-session-persistence-failures --strict --no-interactive` and review the final diff against the stated non-goals; verify no unrelated application changes or secret-bearing fixtures were introduced.

## Workflow follow-up

- After implementation and review, synchronize any spec deltas and archive this change using the OpenSpec workflow. Tooling/refactor changes with skipped specs have no deltas to synchronize.
