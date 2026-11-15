# Design

## Context

See [proposal.md](proposal.md) for source evidence and motivation. Existing baseline specs remain the record of implemented behavior; the planned acceptance scenarios belong only to this change until implementation.

## Goals / Non-Goals

**Goals:** preserve session ownership during storage changes with focused, deterministic coverage.

**Non-Goals:** Do not end practice as a shortcut or solve every migration failure case here.

## Decisions

Keep the existing SDK client as the owner and switch its persistence backend through a minimal synchronized SDK operation. Preserve active practice, cookies, transport and pacing. Ensure previous in-flight persistence cannot write to the retired backend after success returns. Keep desktop mode names and preference files out of the SDK.

### Alternatives considered

A broad subsystem rewrite would expand compatibility risk without being required by the evidence. Use the smallest lifecycle/status/request-ownership change described above. Cancellation alone, ignored persistence errors, or blanket failure of otherwise usable data do not satisfy the acceptance scenarios.

## Characterization Before Changes

- Practice state, attempts and feedback across a successful switch
- same-mode no-op
- failed destination Save leaves old mode usable.

Add regression tests for the incorrect cases separately from characterization of behavior that must be preserved. A defect reproduction may fail before the fix; preservation tests must pass before refactoring. Use barriers, deferred promises, fake clocks, and failing stores rather than sleeps or live accounts.

## Dependencies and Sequencing

Coordinate with prevent-stale-auth-refresh-commit and reuse its lifecycle synchronization. Establishes ownership for migration failure handling.

Dependencies here are an implementation ordering contract; OpenSpec artifact readiness does not automatically enforce cross-change sequencing. Re-read prerequisites and current source before applying.

## Risks / Trade-offs

- Concurrent operations or partial failures may expose untested ordering → reproduce the specific scenario deterministically and keep IO/lock ordering explicit.
- Contract drift → preserve the compatibility boundaries in proposal.md and regenerate bindings instead of editing generated output.
- Over-scoping → stop at the acceptance scenarios; capture additional defects as separate changes.

## Migration Plan

SDK store ownership and desktop switching; retain active trainer state and snapshot format. Prefer an additive SDK operation without Wails signature changes.

Land preservation/regression tests with the implementation that satisfies them. No persistent data-format migration is planned. Revert the bounded implementation if regressions appear; do not delete user session data as a rollback technique. Reverting a security fix restores its known risk and should not be treated as remediation.

## Validation

### Implementation evidence

- `Client.SetSessionStore` saves to the actual destination instance and changes the selected backend under the authentication lifecycle mutex. A nil destination is rejected and failed saves retain the previous backend. It preserves the authentication generation, client, trainer service, and pacing guard; callers own cleanup of retired stores.
- The desktop switches the existing client rather than constructing a replacement. A dedicated switch mutex orders backend changes and existing old-store cleanup without holding the general Hub mutex across backend IO.
- Focused characterization passed before implementation for active-question/attempt/feedback preservation, same-mode switching, and failed destination saves.
- The new active-trainer regression failed before the fix because forcing refresh through the original owner recreated `session.json`; it passes after the fix. Additional SDK coverage proves a refresh pending during a switch persists its eventual token to the actual destination instance.
- Focused SDK and desktop tests passed. The parent integration run verified the full SDK and desktop/CLI race suites, all 18 strict OpenSpec validations, and diff whitespace checks against the combined worktree.
- Existing desktop integration setup now selects temporary memory storage before initialization, keeping tests independent of the user's OS keychain.

The separate migration-failure-semantics proposal still owns transactional preference publication and reporting/retry of retired-store cleanup failures. This change retains the existing preference-save and best-effort cleanup outcomes.

SDK race tests and CGO_ENABLED=1 go test -race ./internal/...

Also run `openspec validate preserve-session-owner-on-store-switch --strict --no-interactive` and confirm all tasks are checked only after their verification succeeds.
