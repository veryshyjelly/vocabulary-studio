# Project structure

The repository contains a reusable SDK and applications built on it. There are
two Go modules: `sdk/go.mod` for the library, and the root `go.mod` for desktop,
CLI, and development tools. The root module resolves the SDK through a local
replacement, so a checkout builds without publishing anything.

```text
sdk/                         Independent Go library and module
  internal/htmlparse/        Private HTML parsers
  testdata/cassettes/         Sanitized SDK replay fixtures
  *_test.go                  Tests grouped by domain
cmd/
  vocab-desktop/             Wails application entry point
  vocab-cli/                 CLI entry point
  record-fixtures/           Live fixture recording tool
internal/desktop/            Desktop adapter, services, and safe UI DTOs
frontend/
  embed.go                   Go adapter embedding the Vite output
  src/
    app/                     Application shell and integration tests
    features/                Auth, home, practice, dictionary, lists,
                             discovery, progress, and settings
    components/ui/           Reusable controls, layout, feedback, and dialogs
    components/shared/       Rich text, audio, clipboard, and error components
    lib/                     Wails bridge adapter and keyboard utilities
    styles/                  Fonts, tokens, theme, typography, base, motion
    bindings/                Generated Wails bindings; regenerate, do not edit
    main.tsx                 Production entry point
    index.css                Stable stylesheet entry point
  tests/
    fixtures/                Isolated visual-preview entry and sample data
    support/                 Vitest setup
    preview.html             Stable development-only preview URL
  public/                    Bundled fonts, licenses, and application logo
build/macos/                 macOS bundle metadata template
scripts/package-macos.sh     Reproducible local macOS packaging
bin/, dist/, frontend/dist/  Generated executables, bundles, and web assets
```

## Dependency boundaries

- The SDK never imports desktop code, Wails, or frontend assets. It can be built
  and tested entirely from its own directory with CGO disabled.
- Desktop services orchestrate the SDK and return safe DTOs. Each domain has its
  own service and type file; trainer, bee, and quiz adapters are separate files
  in the same Go package so Wails service names remain stable.
- Feature screens use `frontend/src/lib/bridge.ts` for backend access. Generated
  SDK model paths are confined to that adapter.
- UI primitives import their sibling utilities directly; feature-specific
  authentication stays under `features/auth/`.
- SDK consumers import `github.com/veryshyjelly/vocabulary-studio/sdk`, as
  documented in [sdk/README.md](../sdk/README.md). The root module contains the
  applications and development tools.

## Common workflows

```sh
make test          # SDK, desktop/CLI, and frontend tests and type checks
make build         # Build frontend assets, then the native desktop binary
make bindings      # Regenerate Wails bindings and rebuild the frontend
make package-macos # Build, sign, and verify the local .app bundle
```

Existing entry-point commands remain available, including
`go run ./cmd/vocab-cli` and `./scripts/package-macos.sh`.

The SDK is a nested module: root-level `go test ./...` does not include its tests.
Use `make test` for the whole project or run `go test ./...` inside `sdk/`.
Build the frontend before commands that load the desktop entry point, because
`frontend/embed.go` embeds the generated `frontend/dist/` directory.

See the [development guide](DEVELOPMENT.md) for setup, packaging, individual test
commands, CLI usage, and visual previews.

## Implementation notes

### SDK authentication and gameplay

The SDK owns Vocabulary.com authentication, dictionary lookups, lists, learning
progress, and gameplay. Authentication combines web session cookies with bearer
JWTs from the upstream token endpoint; concurrent token requests are deduplicated
with `singleflight`.

VocabTrainer, list practice, Spelling Bee, and quizzes share the SDK's timing
abstractions and pacing controls. `PacingGuard` spaces answer submissions and
limits bursts of requests. Keep this behavior in the SDK rather than duplicating
it in desktop or CLI callers.

`TrainerSession` tracks idle, active-question, question-completed, and trial-over
states. First attempts can be evaluated and persisted by the server; subsequent
attempts can be evaluated locally. The `EvaluatedLocally` result distinguishes
these paths.

See the [SDK guide](../sdk/README.md) for usage and independent library tests.

### Desktop services and session storage

Seven Wails services expose authentication, practice, dictionary, lists,
discovery, progress, and settings through generated TypeScript bindings.
Desktop adapters return UI DTOs rather than raw SDK session data. Passwords,
cookies, and bearer tokens stay out of bridge responses and browser storage.

Session persistence supports the OS keychain, a restricted local file with
`0600` permissions and atomic writes, or ephemeral memory. Storage backends do
not silently downgrade on failure.

Upstream HTML is parsed into structured `RichSpan` data before reaching React.
The frontend renders these spans without `dangerouslySetInnerHTML`. Desktop
services cancel obsolete autocomplete, lookup, and search requests.

### Frontend

The React and TypeScript frontend uses Vite, Tailwind CSS, Lucide icons, and Bun.
The application shell provides Home, Practice, Dictionary, My Lists, Discover,
Progress, and Settings views. Shared controls, styles, and the bridge adapter
remain separate from feature screens.

Go embeds the production frontend assets. The preview entry under
`frontend/tests/` uses sample data and is excluded from the production bundle;
use the native Wails application to verify real account and SDK behavior.
