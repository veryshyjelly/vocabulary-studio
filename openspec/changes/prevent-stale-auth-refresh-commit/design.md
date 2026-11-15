# Design

## Context

See [proposal.md](proposal.md) for source evidence and motivation. Existing baseline specs remain the record of implemented behavior; the planned acceptance scenarios belong only to this change until implementation.

## Goals / Non-Goals

**Goals:** prevent stale authentication refresh commits with focused, deterministic coverage.

**Non-Goals:** No storage migration or gameplay redesign.

## Decisions

Generation-scope token acquisition and singleflight. Serialize commit and persistence with logout/account replacement; a generation check followed by an uncoordinated Save is insufficient. Keep network IO outside the general client mutex. Preserve independent cancellation of waiters in the same generation.

### Alternatives considered

A broad subsystem rewrite would expand compatibility risk without being required by the evidence. Use the smallest lifecycle/status/request-ownership change described above. Cancellation alone, ignored persistence errors, or blanket failure of otherwise usable data do not satisfy the acceptance scenarios.

## Characterization Before Changes

- Existing shared refresh
- cancellation of one waiter does not abort others
- failed login preserves the previous account
- 401 refresh retry.

Add regression tests for the incorrect cases separately from characterization of behavior that must be preserved. A defect reproduction may fail before the fix; preservation tests must pass before refactoring. Use barriers, deferred promises, fake clocks, and failing stores rather than sleeps or live accounts.

## Dependencies and Sequencing

None. First recommended change.

Dependencies here are an implementation ordering contract; OpenSpec artifact readiness does not automatically enforce cross-change sequencing. Re-read prerequisites and current source before applying.

## Risks / Trade-offs

- Concurrent operations or partial failures may expose untested ordering → reproduce the specific scenario deterministically and keep IO/lock ordering explicit.
- Contract drift → preserve the compatibility boundaries in proposal.md and regenerate bindings instead of editing generated output.
- Over-scoping → stop at the acceptance scenarios; capture additional defects as separate changes.

## Migration Plan

SDK auth internals and tests; preserve exported signatures, failed-login rollback and session format. No bridge change expected.

Land preservation/regression tests with the implementation that satisfies them. No persistent data-format migration is planned. Revert the bounded implementation if regressions appear; do not delete user session data as a rollback technique. Reverting a security fix restores its known risk and should not be treated as remediation.

## Validation

### Implementation evidence

- Token acquisition and singleflight keys carry an authentication generation; obsolete results return `ErrSessionExpired` without a token.
- Lifecycle ordering serializes token commits and session saves with logout, restoration, and imports. Token HTTP work runs outside that lock.
- Token HTTP clients wrap their cookie jar with the generation check, preventing late response cookies from restoring credentials before body decoding.
- Independent review strengthened the regressions to inspect exported and saved credential cookies for logout, import, and login transitions. All three fail when the cookie guard is removed in an isolated temporary SDK copy; the guarded regressions pass 20 race-enabled repetitions.
- Barrier-controlled regressions cover logout, import, successful login during an old refresh, and logout during an in-flight save. The logout regression was verified to fail against the original source in a temporary checkout copy.
- The authentication regressions and existing rollback/singleflight/401 characterization passed 20 repeated race-enabled runs. Complete SDK and desktop/CLI race suites and SDK `go vet ./...` passed.

cd sdk && CGO_ENABLED=1 go test -race ./...

Also run `openspec validate prevent-stale-auth-refresh-commit --strict --no-interactive` and confirm all tasks are checked only after their verification succeeds.
