# Implementation backlog

Nine bounded changes capture the repository review. Each change's task checklist records implementation and verification progress; planning readiness alone does not mean implementation is complete. Completed work remains available for review before synchronization and archive.

## Recommended sequence

| Order | Change | Risk / impact | Dependencies |
| --- | --- | --- | --- |
| 1 | [prevent-stale-auth-refresh-commit](changes/prevent-stale-auth-refresh-commit/proposal.md) | High | None. First recommended change. |
| 2 | [preserve-session-owner-on-store-switch](changes/preserve-session-owner-on-store-switch/proposal.md) | High | Coordinate with prevent-stale-auth-refresh-commit and reuse its lifecycle synchronization. Establishes ownership for migration failure handling. |
| 3 | [surface-session-persistence-failures](changes/surface-session-persistence-failures/proposal.md) | High | Independent; coordinate status/error vocabulary with define-storage-migration-failure-semantics. |
| 4 | [invalidate-obsolete-gameplay-operations](changes/invalidate-obsolete-gameplay-operations/proposal.md) | High | Use the ownership decision from preserve-session-owner-on-store-switch; coordinate account invalidation with prevent-stale-auth-refresh-commit. |
| 5 | [define-storage-migration-failure-semantics](changes/define-storage-migration-failure-semantics/proposal.md) | Medium | After preserve-session-owner-on-store-switch; reuse persistence diagnostics from surface-session-persistence-failures. |
| 6 | [scope-list-progress-to-selected-list](changes/scope-list-progress-to-selected-list/proposal.md) | Medium | None; independently implementable. |
| 7 | [report-partial-dashboard-failures](changes/report-partial-dashboard-failures/proposal.md) | Medium | None. |
| 8 | [validate-pull-requests](changes/validate-pull-requests/proposal.md) | Medium | None; can proceed alongside higher-risk corrections. External branch protection was not inspected. |
| 9 | [simplify-trainer-answer-bookkeeping](changes/simplify-trainer-answer-bookkeeping/proposal.md) | Low | Characterization passes before extraction. Schedule after lifecycle fixes settle; otherwise logically independent. |

The order prioritizes authentication/session correctness, then user-visible data correctness, validation infrastructure, and finally refactoring. Independent work such as list progress and PR validation can proceed earlier. Cross-change dependencies are documented here and in each design; the CLI does not enforce them automatically.

## Evidence and review limits

The source review found credential resurrection after logout, obsolete client ownership after store switching, discarded persistence errors, stale gameplay commits, partial migration failures, and stale list enrichment. These are source-level control-flow findings, not newly added failing reproductions. Existing Go tests, including race-enabled runs, and all 13 frontend tests passed during review. Each corrective change adds the missing deterministic regression coverage before its fix.

Dashboard availability and PR validation are evidence-backed production-readiness gaps. Shared trainer bookkeeping is a maintenance/simplification opportunity, not an independently demonstrated correctness defect. The PR-validation and trainer-simplification changes use `skip_specs: true`; no product requirement was invented for them.

## Coverage and characterization

| Area | Preserve before changing | Missing regression coverage |
| --- | --- | --- |
| Authentication | Shared refresh, independent waiter cancellation, failed-login rollback, 401 retry | Refresh against logout/account replacement; obsolete save ordering |
| Storage ownership | Active practice state, shared pacing, same-mode no-op | Continue existing trainer after switching file to memory and forcing refresh |
| Persistence feedback | Valid web authentication distinct from API-token availability; safe DTO/JSON output | Save failures during login/refresh and successful retry |
| Gameplay lifecycle | First-save/local-retry behavior, Bee progression, quiz completion | Old results after replacement/logout; logout with failed store Clear |
| Migration | Valid preference filtering, events, successful switching, unrelated preferences | Fail each persistence boundary and overlap switches |
| List details | Navigation, flashcards, partial detail usefulness | Reverse-order progress, stale errors, navigation/account changes |
| Dashboard | Partial success, anonymous daily word, result limits | Empty success versus unavailable sections; account verification outage |
| CI | Existing release triggers and deterministic local commands | PR-triggered execution with no release secrets |
| Trainer refactor | Request counts, reward clearing, normalization, hints, secret rotation, state/error transitions | Characterize both answer modes and failure/cancellation paths before extraction |

## Implementation progress

Implemented and locally validated:

- [prevent-stale-auth-refresh-commit](changes/prevent-stale-auth-refresh-commit/proposal.md)
- [preserve-session-owner-on-store-switch](changes/preserve-session-owner-on-store-switch/proposal.md)

7 changes remain planned. Completed changes await review, synchronization, and archive.
