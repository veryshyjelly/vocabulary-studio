# Tasks

## 1. Bounded implementation with characterization

Prerequisites: Characterization passes before extraction. Schedule after lifecycle fixes settle; otherwise logically independent.

- [ ] 1.1 Add or retain passing characterization coverage for: First success saves once; retries stay local with zero points/bonus/achievements; canceled/failed/missing-answer responses preserve attempts; secret rotation/hints/completion/trial-over/pacing; choice versus spelling semantics and clone isolation. Verify preservation tests pass against the pre-change behavior before refactoring; do not preserve the identified defect as an expected success.
- [ ] 1.2 Add table-driven characterization for both answer modes and failures; verify the existing implementation passes before extraction.
- [ ] 1.3 Extract shared commit/reward bookkeeping only; verify unchanged outcomes and request counts.
- [ ] 1.4 Review API/locking diff and run SDK race suite; verify no wire, public API or behavior change.

## 2. Integration validation

- [ ] 2.1 Run cd sdk && CGO_ENABLED=1 go test -race ./... Record results and any environment limitation; do not substitute live account requests for deterministic tests.
- [ ] 2.2 Run `openspec validate simplify-trainer-answer-bookkeeping --strict --no-interactive` and review the final diff against the stated non-goals; verify no unrelated application changes or secret-bearing fixtures were introduced.

## Workflow follow-up

- After implementation and review, synchronize any spec deltas and archive this change using the OpenSpec workflow. Tooling/refactor changes with skipped specs have no deltas to synchronize.
