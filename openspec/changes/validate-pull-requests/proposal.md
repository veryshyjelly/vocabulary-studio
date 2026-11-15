# Proposal

## Why

The only checked-in workflow runs on release tags or manual dispatch. Existing checks therefore have no repository-defined PR trigger before integration.

## What Changes

- Validate pull requests with existing checks through the bounded approach in design.md.
- Add focused regression and characterization coverage before changing implementation.
- Preserve product behavior; this change declares `skip_specs: true`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None; validation tooling only.

## Impact

CI configuration and development docs only; no application behavior or release publishing changes.

## Evidence and Coverage Gaps

- `.github/workflows/release.yml:3` (repository-relative source locations).
- `Makefile:13` (repository-relative source locations).
- `openspec/config.yaml` (repository-relative source locations).

Existing release triggers and test commands; lockfile-based installation; tests require no Vocabulary.com credentials.

The review established source-level failure paths or maintenance gaps. Existing Go tests including race runs and 13 frontend tests passed; targeted regression interleavings have not yet been added. Each task list requires those tests before the associated fix/refactor.

## Priority and Dependencies

Medium. None; can proceed alongside higher-risk corrections. External branch protection was not inspected.

## Out of Scope

No product requirement, coverage quota, new release-platform promise or signing changes.
