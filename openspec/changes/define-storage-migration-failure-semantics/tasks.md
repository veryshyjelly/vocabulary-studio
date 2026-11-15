# Tasks

## 1. Bounded implementation with characterization

Prerequisites: After preserve-session-owner-on-store-switch; reuse persistence diagnostics from surface-session-persistence-failures.

- [ ] 1.1 Add or retain passing characterization coverage for: Same-mode no-op; successful file/memory switches; invalid enum behavior; preference events; destination-save failure preserves old mode; unrelated preferences survive. Verify preservation tests pass against the pre-change behavior before refactoring; do not preserve the identified defect as an expected success.
- [ ] 1.2 Inject destination-save, preference-write, old-clear and staged-cleanup failures; assert live/durable selection and diagnostic outcomes at each boundary.
- [ ] 1.3 Implement serialized staged migration with stable ownership; test concurrent switches and retirement of old stores.
- [ ] 1.4 Persist candidate ordinary preferences before publishing; test failed writes leave old values and emit no success event.
- [ ] 1.5 Expose retryable former-store cleanup through shared persistence diagnostics; test recovery, document partial success and regenerate bindings when needed.

## 2. Integration validation

- [ ] 2.1 Run Desktop race tests, SDK persistence tests and settings UI tests Record results and any environment limitation; do not substitute live account requests for deterministic tests.
- [ ] 2.2 Run `openspec validate define-storage-migration-failure-semantics --strict --no-interactive` and review the final diff against the stated non-goals; verify no unrelated application changes or secret-bearing fixtures were introduced.

## Workflow follow-up

- After implementation and review, synchronize any spec deltas and archive this change using the OpenSpec workflow. Tooling/refactor changes with skipped specs have no deltas to synchronize.
