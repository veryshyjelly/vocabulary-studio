# Design

## Context

See [proposal.md](proposal.md) for source evidence and motivation. Existing baseline specs remain the record of implemented behavior; the planned acceptance scenarios belong only to this change until implementation.

## Goals / Non-Goals

**Goals:** validate pull requests with existing checks with focused, deterministic coverage.

**Non-Goals:** No product requirement, coverage quota, new release-platform promise or signing changes.

## Decisions

Add a pull_request workflow with read-only contents permission, no release secrets and no pull_request_target. Reuse existing toolchain/runner setup. Test sdk/ separately from root packages; typecheck/build/test frontend with lockfile; pin OpenSpec and validate all artifacts strictly. Optional branch-protection configuration is a separate follow-up.

### Alternatives considered

Release-only checks discover failures late. A new CI framework or platform matrix adds unsupported scope; reuse existing commands and runner setup instead.

## Characterization Before Changes

- Existing release triggers and test commands
- lockfile-based installation
- tests require no Vocabulary.com credentials.

Add regression tests for the incorrect cases separately from characterization of behavior that must be preserved. A defect reproduction may fail before the fix; preservation tests must pass before refactoring. Use barriers, deferred promises, fake clocks, and failing stores rather than sleeps or live accounts.

## Dependencies and Sequencing

None; can proceed alongside higher-risk corrections. External branch protection was not inspected.

Dependencies here are an implementation ordering contract; OpenSpec artifact readiness does not automatically enforce cross-change sequencing. Re-read prerequisites and current source before applying.

## Risks / Trade-offs

- Concurrent operations or partial failures may expose untested ordering → reproduce the specific scenario deterministically and keep IO/lock ordering explicit.
- Contract drift → preserve the compatibility boundaries in proposal.md and regenerate bindings instead of editing generated output.
- Over-scoping → stop at the acceptance scenarios; capture additional defects as separate changes.

## Migration Plan

CI configuration and development docs only; no application behavior or release publishing changes.

Land preservation/regression tests with the implementation that satisfies them. No persistent data-format migration is planned. Revert the bounded implementation if regressions appear; do not delete user session data as a rollback technique. Reverting a security fix restores its known risk and should not be treated as remediation.

## Validation

Run focused tests appropriate to the affected layers and preserve the characterized behavior.

Run `openspec validate validate-pull-requests --strict --no-interactive` before implementation is considered complete.
