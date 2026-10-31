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
sdk_compat.go                Deprecated original Go import facade
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
- The root Go facade contains compatibility declarations, never SDK logic. New
  consumers should use the module documented in [sdk/README.md](../sdk/README.md).

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
