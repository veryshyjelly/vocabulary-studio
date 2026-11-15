# Development

Build and work on Vocabulary Studio from source. For the prebuilt macOS app,
see [installation in the README](../README.md#install-on-macos).

Run the commands below from the repository root unless noted otherwise.
See [Project structure](ARCHITECTURE.md) for module boundaries and the
[SDK guide](../sdk/README.md) to work on the standalone library.


## Prerequisites
- **Go 1.27.3 or newer**, matching `go.mod`.
- **Bun** (`bun`)
- **Wails v3 CLI**, matching the repository's dependency: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`.
- Platform WebView dependencies (macOS WKWebView, Windows WebView2, or Linux GTK3/WebKit2GTK). Note: while the SDK is 100% pure Go (`CGO_ENABLED=0`), building the native Wails desktop binary (`cmd/vocab-desktop`) uses CGo to link native OS window/WebView APIs.

## Build from source on macOS

Clone the repository and run the commands below from its root:

```bash
git clone https://github.com/veryshyjelly/vocabulary-studio.git
cd vocabulary-studio
```

Install Go and Bun first. Install Apple's Command Line Tools if needed:

```bash
xcode-select --install
```

The packaging helper also uses `python3`, `qlmanage`, `codesign`, and `plutil`.
Ensure `python3` is available; the other tools are supplied by macOS. Then install
the matching Wails CLI and build:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
export PATH="$(go env GOPATH)/bin:$PATH"
./scripts/package-macos.sh
```

The helper installs the locked frontend dependencies, builds the frontend and
production Go executable, adds the app icon, and signs and verifies a local
`dist/Vocabulary Studio.app` bundle. It builds for the Mac running the command:
Apple Silicon on Apple Silicon, or Intel on Intel.

Install through Finder:

```bash
open dist
```

Drag **Vocabulary Studio.app** into **Applications**, then open it. Alternatively,
install for your user account from the terminal:

```bash
mkdir -p "$HOME/Applications"
ditto "dist/Vocabulary Studio.app" "$HOME/Applications/Vocabulary Studio.app"
open "$HOME/Applications/Vocabulary Studio.app"
```

Frontend assets and Monaspace fonts are included in the app. You do not need to
run a development server, install fonts, or create a `.env` file to use the
desktop app; sign in through its interface. Run the helper again after pulling
changes, then replace the installed app with the new bundle.

The helper also writes a versioned ZIP and SHA-256 checksum in `dist/`, ready
for release. Set `GOARCH=arm64` or `GOARCH=amd64` to select the target architecture;
by default it builds for the current Mac. It uses ad-hoc signing, and requires
macOS 13 or later. The release workflow builds each architecture on its matching
Mac runner. See [Releasing](RELEASING.md) for the release and tap update process.

## Development Build and Optional Binding Regeneration

Build the frontend first: Go embeds `frontend/dist`, which is generated and is not
included in a fresh checkout.

```bash
cd frontend
bun install --frozen-lockfile
bun run build
cd ..
```

TypeScript bindings are checked in. Regenerate them when Go service interfaces
change, then rebuild the frontend:

```bash
wails3 generate bindings -ts -i -names -d frontend/src/bindings ./cmd/vocab-desktop
cd frontend
bun run build
cd ..
```

## Build & Run the Native Desktop Application
```bash
# Compile the native desktop binary (embeds frontend/dist)
go build -o bin/vocab-desktop ./cmd/vocab-desktop

# Launch the desktop application
./bin/vocab-desktop
```

## Tests and static analysis

Pull requests run the **Validate pull requests / Project validation** GitHub
Actions check from `.github/workflows/validate.yml`. It uses the existing
`macos-15` environment, the Go version in `go.mod`, Bun `1.4.2`, Node.js `22`,
and OpenSpec CLI `1.16.1`. It installs locked frontend dependencies, checks types,
builds and tests the frontend, tests the SDK and desktop/CLI packages separately,
and validates all OpenSpec artifacts strictly. No Vocabulary.com credentials or
release secrets are required. The checkout does not persist its token.

The release workflow remains separate. Adding the PR workflow does not configure
branch protection or make its check required. After pushing this workflow in a
PR, inspect its first run in GitHub Actions before treating CI as operational.
Equivalent local commands are:

```bash
cd frontend
bun install --frozen-lockfile
bun run typecheck --noUnusedLocals
bun run build
bun run test
cd ..
make test-sdk test-desktop
openspec validate --all --strict --no-interactive
```

Install the matching OpenSpec CLI when needed with
`npm install --global @fission-ai/openspec@1.16.1`.

```bash
# Run every module and the frontend (installs locked frontend dependencies)
make test

# Or run the standalone SDK and its parsers independently
cd sdk
CGO_ENABLED=0 go test -v ./...
cd ..

# Run CLI and desktop-service tests
CGO_ENABLED=0 go test -v ./internal/... ./cmd/vocab-cli ./cmd/record-fixtures

# Run the SDK race detector and static analysis
cd sdk
CGO_ENABLED=1 go test -race ./...
go vet ./...
cd ..
go vet ./...

# Run Frontend Vitest + React Testing Library suite
cd frontend && bun run test
```

---

## CLI usage

Explicit `login` reports authentication and persistence separately. If sign-in
succeeds but the session file cannot be saved, it prints a generic warning to
stderr and exits with status `1`; it does not claim the session was saved. Check
the selected session path and sign in again to retry. With `--json`, stdout
remains a valid account JSON document even on this persistence failure. The
`login` and `me` JSON account output omits the credential-bearing `guid` field;
other profile fields are preserved.

Copy `.env.example` to `.env.local` (gitignored) and set `VOCAB_USERNAME` and `VOCAB_PASSWORD`, then run:

```bash
go run ./cmd/vocab-cli login
go run ./cmd/vocab-cli me
go run ./cmd/vocab-cli lookup callow
go run ./cmd/vocab-cli examples callow
go run ./cmd/vocab-cli wod
go run ./cmd/vocab-cli search-lists gre
go run ./cmd/vocab-cli play
```

## UI development and visual checks

The stylesheet entry is `frontend/src/index.css`; fonts, tokens, typography, and
component styles are separated under `frontend/src/styles/`. Shared controls live
in `frontend/src/components/ui/`; screens live in `frontend/src/features/`.
See [the project structure](ARCHITECTURE.md) for ownership and build workflows.

For isolated visual checks, run `cd frontend && bun run dev`, then open
`http://localhost:5173/tests/preview.html?screen=practice&scenario=spelling&theme=dark`.
This development-only entry uses sample data and never enters the production bundle.
Available scenarios include `choices`, `spelling`, `images`, `long`, `empty`,
`error`, `loading`, and `auth`; the `screen` parameter selects an application section.
Use the Wails application to validate real account and SDK behavior.
