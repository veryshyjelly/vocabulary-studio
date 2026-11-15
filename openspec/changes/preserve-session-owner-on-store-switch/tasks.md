# Tasks

## 1. Bounded implementation with characterization

Prerequisites: Coordinate with prevent-stale-auth-refresh-commit and reuse its lifecycle synchronization. Establishes ownership for migration failure handling.

- [x] 1.1 Add or retain passing characterization coverage for: Practice state, attempts and feedback across a successful switch; same-mode no-op; failed destination Save leaves old mode usable. Verify preservation tests pass against the pre-change behavior before refactoring; do not preserve the identified defect as an expected success.
- [x] 1.2 Add an integration test that starts trainer practice, switches file to memory, forces refresh and continues; verify the old session file is not recreated.
- [x] 1.3 Implement stable client ownership and coordinated backend replacement; verify pre-existing session references use the new backend.
- [x] 1.4 Document ownership and add same-mode/destination-failure coverage; verify snapshot and Wails compatibility.

## 2. Integration validation

- [x] 2.1 Run SDK race tests and CGO_ENABLED=1 go test -race ./internal/... Record results and any environment limitation; do not substitute live account requests for deterministic tests.
- [x] 2.2 Run `openspec validate preserve-session-owner-on-store-switch --strict --no-interactive` and review the final diff against the stated non-goals; verify no unrelated application changes or secret-bearing fixtures were introduced.

## Workflow follow-up

- After implementation and review, synchronize any spec deltas and archive this change using the OpenSpec workflow. Tooling/refactor changes with skipped specs have no deltas to synchronize.
