# Tasks

## 1. Bounded implementation with characterization

Prerequisites: None; can proceed alongside higher-risk corrections. External branch protection was not inspected.

- [ ] 1.1 Add or retain passing characterization coverage for: Existing release triggers and test commands; lockfile-based installation; tests require no Vocabulary.com credentials. Verify preservation tests pass against the pre-change behavior before refactoring; do not preserve the identified defect as an expected success.
- [ ] 1.2 Create least-privilege PR workflow and verify trigger/permission configuration.
- [ ] 1.3 Wire separate SDK/root tests, frontend checks and strict OpenSpec validation; execute corresponding commands locally.
- [ ] 1.4 Pin validation tooling and document job names; validate YAML and inspect first PR run when available before declaring operational success.

## 2. Integration validation

- [ ] 2.1 Run Workflow syntax and wired commands; first PR run when GitHub execution is available Record results and any environment limitation; do not substitute live account requests for deterministic tests.
- [ ] 2.2 Run `openspec validate validate-pull-requests --strict --no-interactive` and review the final diff against the stated non-goals; verify no unrelated application changes or secret-bearing fixtures were introduced.

## Workflow follow-up

- Inspect `Validate pull requests / Project validation` on the first PR containing this workflow; hosted execution is not yet verified. Configure branch protection separately only if requested.
- After implementation and review, synchronize any spec deltas and archive this change using the OpenSpec workflow. Tooling/refactor changes with skipped specs have no deltas to synchronize.
