# OpenSpec in Vocabulary Studio

This directory records the repository's existing behavior and plans future changes.
It was initialized with OpenSpec CLI **1.16.1** using the `spec-driven` schema and
the core Codex profile. See the [OpenSpec documentation](https://github.com/Fission-AI/OpenSpec)
for CLI installation and workflow details.

## Layout

- `config.yaml` supplies project constraints and artifact rules.
- `specs/<capability>/spec.md` describes current behavior using requirements and scenarios.
- `changes/<change-name>/` holds a proposed change's proposal, design, spec deltas, and tasks.
- `changes/archive/` holds completed changes after their deltas have been synchronized.
- `../.agents/skills/openspec-*/` contains generated Codex workflow skills.

The baseline specs describe implemented capabilities. They do not introduce new
features or claim that every scenario already has a dedicated automated test.
Source and test references in each spec help locate the implementation and
existing coverage. There are no active change proposals at initialization.

## Capability index

| Capability | Contract |
| --- | --- |
| [SDK client](specs/sdk-client/spec.md) | Independent module, transport, token refresh, and pacing |
| [Authentication](specs/authentication/spec.md) | Sign-in, restoration, logout, and session storage |
| [Dictionary](specs/dictionary/spec.md) | Lookup, examples, pronunciation, and history |
| [Vocabulary lists](specs/vocabulary-lists/spec.md) | Browse, create, study, and manage lists |
| [Practice](specs/practice/spec.md) | Trainer state, retries, hints, Spelling Bee, and quizzes |
| [Discovery](specs/discovery/spec.md) | Word of the Day, curated collections, and WordFinder |
| [Learning progress](specs/learning-progress/spec.md) | Word learning, dashboard, activity, and achievements |
| [Desktop experience](specs/desktop-experience/spec.md) | Navigation, preferences, keyboard access, and safe rendering |
| [CLI](specs/cli/spec.md) | Commands, structured output, and interactive play |

## Change workflow

In Codex CLI or IDE, use `$openspec-propose "describe the change"` to generate a
proposal, design, spec deltas, and tasks. In the Codex desktop app, select
`openspec-propose` from Skills in the sidebar. Review those artifacts before
implementation, then use `openspec-apply-change`. Use `openspec-update-change`
to revise a proposal and `openspec-archive-change` when the work is complete.
The generated skills also include `openspec-explore` and `openspec-sync-specs`.

For a manual start from the repository root:

```sh
openspec new change <change-name>
openspec status --change <change-name>
openspec instructions proposal --change <change-name>
```

Use a descriptive kebab-case change name. Follow artifact instructions to write
the proposal, design, capability deltas, and checkable tasks in dependency order.
Keep planned behavior in the change directory until the change is implemented
and its deltas are synchronized into the baseline. Use `## ADDED Requirements`,
`## MODIFIED Requirements`, `## REMOVED Requirements`, or `## RENAMED Requirements`
in delta specs; baseline specs use `## Requirements`.

## Validation

Run from the repository root:

```sh
openspec list --specs
openspec validate --all --strict --no-interactive
```

For a specific proposal, use `openspec validate <change-name> --strict --no-interactive`.
OpenSpec validation checks artifact structure; implementation changes also need
the relevant checks in [the development guide](../docs/DEVELOPMENT.md).
`make test` includes the nested SDK module, root desktop/CLI tests, and frontend
type checks and tests. Build frontend assets before native desktop builds, and
regenerate bindings when changing Wails service contracts.
