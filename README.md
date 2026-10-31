# vocabulary_go & Vocabulary Studio Desktop

A complete Vocabulary.com Go SDK (`vocabulary_go`) and cross-platform native desktop application (**Vocabulary Studio**) built with **Wails v3**, **Go**, **React 19**, **TypeScript**, **Vite**, **Tailwind CSS**, and **Lucide**.

---

## 1. Architecture Overview

The repository is structured into three cleanly separated layers:

1. **Core Go SDK (`github.com/veryshyjelly/vocabulary_go` root package + `internal/htmlparse`)**:
   - **100% Pure Go (`CGO_ENABLED=0`)**: Zero CGo dependencies in the SDK layer.
   - **Dual-Stage Authentication**: Manages web session cookies (`SESSION`, `guid`, `tz`, `__cf_bm`) and `singleflight`-deduplicated Bearer JWTs (`POST https://api.vocabulary.com/1.0/auth/token`).
   - **Protocol-Accurate Gameplay & Anti-Cheat Pacing**: Implements VocabTrainer (`v=4`), List Practice (`v=3`), Spelling Bee, and Quizzes. `PacingGuard` enforces minimum response times (`>= 950ms` 1st try, `>= 475ms` retry, `>= 1550ms` quiz submit) and reserves slots in the 30.5-second rolling window (`< 10` answers per `30s`) to prevent triggering Vocabulary.com's 5-minute cooldown lockout.
   - **Explicit `TrainerSession` State Machine**: Tracks `SessionStateIdle`, `SessionStateActiveQuestion`, `SessionStateQuestionCompleted`, and `SessionStateTrialOver`, distinguishing server-persisted 1st attempts (`EvaluatedLocally: false`) from client-evaluated subsequent attempts (`EvaluatedLocally: true`).
2. **Desktop Application Layer (`internal/desktop` & `cmd/vocab-desktop`)**:
   - Exposes 7 domain-oriented Wails v3 services (`AuthService`, `TrainerService`, `DictionaryService`, `ListsService`, `DiscoveryService`, `ProgressService`, `SettingsService`) with auto-generated TypeScript bindings (`frontend/src/bindings`).
   - **Strict Secret Isolation**: Passwords, session cookies, GUIDs, and Bearer JWTs never cross the Go/JS bridge and are never stored in `localStorage`, `sessionStorage`, or `IndexedDB`.
   - **Pluggable Session Storage**: Supports `OS Native Keyring` (`KeyringSessionStore`), `Restricted Local File` (`FileSessionStore` with `0600` atomic writes), and `Ephemeral Memory` (`MemorySessionStore`) without silent security downgrades.
   - **HTML Sanitization at the Boundary**: Parses inline HTML blurbs and question prompts into structured `RichSpan` arrays (`text`, `bold`, `italic`) in Go so React never uses `dangerouslySetInnerHTML`.
   - **Per-Operation Context Cancellation**: Automatically cancels obsolete in-flight autocomplete, dictionary lookup, and list search requests.
3. **Frontend Layer (`frontend/`)**:
   - Built with React 19, TypeScript, Vite, Tailwind CSS v4, and Lucide icons using **Bun**.
   - Seven cohesive navigation destinations:
     - **Home**: Continue Learning hero, Word of the Day, 7-day study effort, in-progress activities, words needing attention, recent lists, and recent lookups.
     - **Practice**: Adaptive VocabTrainer (`v=4`), List Practice (`v=3`), Spelling Bee, and Quiz modes supporting all 16 question types (`A, D, P, F, I, G, L, H, T, S, Y, R, C, M, E, U`), lifelines (`50/50`, `Definition`, `Word in the Wild`), and keyboard-first shortcuts (`1–4`, `Enter`, `H`).
     - **Dictionary**: Debounced autocomplete, `/` shortcut autofocus, US/UK IPA and audio playback, conversational blurbs, senses, clickable synonyms/antonyms, Word Family tree, and Corpus usage examples ("Word in the Wild") with domain filters.
     - **My Lists**: Personal lists, public list search (70,000+ lists), list detail inspector, interactive **Flashcards** (`Space` to flip, `←`/`→` to navigate), list creator, and **InstaList** (`VocabGrabber` text vocabulary extractor).
     - **Discover**: Curated Collections across all 9 root categories (`Literature`, `Non-Fiction`, `Test Prep`, `Textbooks`, `Current Events`, `Morphology & Roots`, `Historical Documents`, `Speeches`, `Just for Fun`), Word of the Day + monthly archive, and pattern-based **WordFinder**.
     - **Progress**: Words I'm Learning, Trouble Words, and Mastered Words with mastery bars and priority controls, activity logs with pagination, achievements, and global leaderboards.
     - **Settings**: Light/Dark/System editorial theme, US/UK pronunciation accent preference, startup view, session storage security backend selector, keyboard shortcut reference, and account management.

---

## 2. Development & Build Instructions

### Prerequisites
- **Go 1.26+**
- **Bun** (`bun`)
- **Wails v3 CLI** (`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`)
- Platform WebView dependencies (macOS WKWebView, Windows WebView2, or Linux GTK3/WebKit2GTK). Note: while the `vocabulary_go` SDK is 100% pure Go (`CGO_ENABLED=0`), building the native Wails desktop binary (`cmd/vocab-desktop`) uses CGo to link native OS window/WebView APIs.

### Install Frontend Dependencies & Generate Wails Bindings
```bash
# Generate TypeScript bindings from Go services
wails3 generate bindings -ts -i -names -d frontend/src/bindings ./cmd/vocab-desktop

# Install frontend packages and build production assets with Bun
cd frontend
bun install
bun run build
cd ..
```

### Build & Run the Native Desktop Application
```bash
# Compile the native desktop binary (embeds frontend/dist)
go build -o bin/vocab-desktop ./cmd/vocab-desktop

# Launch the desktop application
./bin/vocab-desktop
```

### Run All Tests & Checks (100% Offline)
```bash
# Run pure-Go SDK, HTML parser, CLI, and Desktop service tests
CGO_ENABLED=0 go test -v . ./internal/... ./cmd/vocab-cli ./cmd/record-fixtures

# Run Go race detector and static analysis
CGO_ENABLED=0 go test -race . ./internal/... ./cmd/vocab-cli ./cmd/record-fixtures
go vet ./...

# Run Frontend Vitest + React Testing Library suite
cd frontend && bun run test
```

---

## 3. CLI Usage (`cmd/vocab-cli`)

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
