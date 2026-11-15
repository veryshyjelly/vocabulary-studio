# Repository Guidelines

## Project Structure & Module Organization

The independent `sdk/` Go module implements the Vocabulary.com SDK: authentication, dictionary, lists, gameplay, and progress. `sdk/internal/htmlparse/` parses upstream HTML; root-module `internal/desktop/` exposes domain services to Wails. Entry points live in `cmd/vocab-cli/`, `cmd/vocab-desktop/`, and `cmd/record-fixtures/`.

`frontend/src/` contains React views, shared components, and the bridge adapter in `lib/bridge.ts`. Wails generates `frontend/src/bindings/`. Static assets live in `frontend/public/`; production assets in `frontend/dist/` are embedded by `frontend/embed.go`. Go tests sit beside source files; recorded HTTP fixtures live in `sdk/testdata/cassettes/`.

## Build, Test, and Development Commands

Use the Go version declared in `go.mod` (currently `1.27.3`), Bun, and Wails v3. Native builds require platform WebView dependencies and CGO.

- `cd frontend && bun install`: install frontend dependencies.
- `cd frontend && bun run dev`: start Vite for frontend development.
- `cd frontend && bun run build`: type-check and build embedded assets; run before building the desktop binary.
- `go build -o bin/vocab-desktop ./cmd/vocab-desktop`: build the desktop app; launch with `./bin/vocab-desktop`.
- `go run ./cmd/vocab-cli lookup callow`: exercise a CLI dictionary lookup.
- `make test`: run SDK, desktop/CLI, and frontend tests and type checks.
- `cd sdk && CGO_ENABLED=0 go test ./...`: run offline SDK tests separately from the root module.
- `CGO_ENABLED=0 go test ./internal/... ./cmd/vocab-cli ./cmd/record-fixtures`: run offline desktop and CLI tests.
- `cd sdk && CGO_ENABLED=1 go test -race ./...`: check SDK concurrency with a C compiler available; run the root test packages with the same flags for desktop and CLI coverage.
- `go vet ./...`: run root Go static analysis; also run it from `sdk/` for the library.
- `cd frontend && bun run test`: run Vitest; `bun run typecheck` checks TypeScript separately.

## Coding Style & Naming Conventions

Format Go with `gofmt`; use tab indentation, exported PascalCase names, and package-private camelCase names. Keep the SDK independent of desktop dependencies. Follow existing TypeScript formatting: two spaces, double quotes, semicolons, PascalCase components, and camelCase functions. Regenerate Wails bindings after service changes rather than editing generated files.

## Testing Guidelines

Use Go's `testing` package with `*_test.go` files and `TestName` functions. Prefer `httptest`, replayed cassettes, and fake clocks for deterministic offline tests. Frontend tests use `*.test.tsx`, Vitest, React Testing Library, and jsdom. Cover changed behavior and failure paths; no numeric coverage threshold is configured.

## Commit & Pull Request Guidelines

Use concise imperative subjects. PRs should explain behavior changes, list validation commands and results, link relevant issues, and include screenshots for UI changes.

## Security & Configuration

Copy `.env.example` to `.env.local` for credentials. Never commit credentials, session files, or unsanitized cassettes. Keep passwords, cookies, and tokens out of bridge responses and browser storage.

## OpenSpec Workflow

Read `openspec/README.md` for the capability index and workflow. Current behavior lives in `openspec/specs/`; proposed behavior and tasks live in `openspec/changes/<change-name>/`. Use `.agents/skills/openspec-*` workflows to propose, update, apply, synchronize, and archive changes. Keep baseline specs aligned with implemented behavior and use requirement deltas for planned changes. Validate artifacts with `openspec validate --all --strict --no-interactive`; implementation changes also require the relevant project tests.
