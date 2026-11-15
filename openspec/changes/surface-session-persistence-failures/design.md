# Design

## Context

See [proposal.md](proposal.md) for source evidence and motivation. Existing baseline specs remain the record of implemented behavior; the planned acceptance scenarios belong only to this change until implementation.

## Goals / Non-Goals

**Goals:** report persistence failures separately from authentication with focused, deterministic coverage.

**Non-Goals:** No silent backend fallback; token acquisition failure is not invalid web credentials.

## Decisions

Preserve successful in-memory web authentication and expose additive safe persistence health separately from invalid credentials and API-token availability. Provide retry to the selected backend. Desktop shows an unsaved-session warning; CLI explicit persistent login reports failure on stderr and exits nonzero while preserving valid JSON stdout. Clear warning only on successful save or deliberate memory selection.

### Alternatives considered

A broad subsystem rewrite would expand compatibility risk without being required by the evidence. Use the smallest lifecycle/status/request-ownership change described above. Cancellation alone, ignored persistence errors, or blanket failure of otherwise usable data do not satisfy the acceptance scenarios.

## Characterization Before Changes

- Successful web login with token endpoint failure
- successful durable and memory-only login
- credential-free status and CLI JSON.

Add regression tests for the incorrect cases separately from characterization of behavior that must be preserved. A defect reproduction may fail before the fix; preservation tests must pass before refactoring. Use barriers, deferred promises, fake clocks, and failing stores rather than sleeps or live accounts.

## Dependencies and Sequencing

Independent; coordinate status/error vocabulary with define-storage-migration-failure-semantics.

Dependencies here are an implementation ordering contract; OpenSpec artifact readiness does not automatically enforce cross-change sequencing. Re-read prerequisites and current source before applying.

## Risks / Trade-offs

- Concurrent operations or partial failures may expose untested ordering → reproduce the specific scenario deterministically and keep IO/lock ordering explicit.
- Contract drift → preserve the compatibility boundaries in proposal.md and regenerate bindings instead of editing generated output.
- Over-scoping → stop at the acceptance scenarios; capture additional defects as separate changes.

## Migration Plan

SDK diagnostics, CLI exit/reporting, desktop safe DTOs and UI. Preserve SDK source compatibility and session format; regenerate bindings for additive bridge changes.

Land preservation/regression tests with the implementation that satisfies them. No persistent data-format migration is planned. Revert the bounded implementation if regressions appear; do not delete user session data as a rollback technique. Reverting a security fix restores its known risk and should not be treated as remediation.

## Validation

### Implemented contract

- SDK `Client.PersistenceStatus()` returns only a copied `SessionPersistenceStatus{State}`: `unknown`, `saved`, or `failed`. It uses the client state lock and is safe inside store callbacks. Existing `FlushSession` retries the selected backend and updates health; a failed destination switch leaves the current backend's health intact.
- Successful web authentication remains valid when token acquisition or session saving fails. Automatic saves record health independently, and account-generation invalidation prevents obsolete work from changing replacement health. Failed-login rollback preserves the restored account's prior warning.
- Desktop account status adds `persistence_state`. `GetPersistenceStatus` reads local state without network requests; `RetryPersistence` retries the same backend and returns a generic safe error rather than the store error. Wails bindings were regenerated with the repository flags.
- The global desktop warning survives modal closure. Local status polling every five seconds detects later save failures. Account/mode changes, unmount, and retry generations invalidate stale replies. Successful save or deliberate memory selection clears the warning; an unknown status cannot conceal a known failure.
- CLI login keeps valid success output but warns on stderr and exits 1 when persistence failed. `login --json` and `me --json` omit the refresh credential `guid`, while retaining other profile fields. This is an intentional removal of credential output, documented in the CLI guide.

### Verification evidence

Fault-injected SDK and desktop tests cover failed login/refresh saves, safe copied status, retry failure/recovery, backend selection, and obsolete work. Desktop tests assert retry makes no network requests and cannot expose raw backend secrets. CLI tests cover text/JSON success and partial success, token outages, and credential-free login/profile output. Frontend tests cover partial sign-in, retry, local polling, stale responses, and clearing stale retry feedback.

Final verification passed: complete SDK and desktop/CLI race suites; ten repeated SDK persistence race runs, including both login APIs with concurrent save failure and account replacement; SDK static analysis; frontend strict type checks, all 36 tests, and production build. Independent review identified the rollback-health ordering defect, and re-review confirmed its resolution. All 18 OpenSpec items pass strict validation. No live account credentials were needed.

SDK, desktop/CLI tests and frontend typecheck/build/tests

Also run `openspec validate surface-session-persistence-failures --strict --no-interactive` and confirm all tasks are checked only after their verification succeeds.
