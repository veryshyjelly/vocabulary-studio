# Vocabulary.com Go SDK

A standalone, pure-Go library for authentication, dictionary lookups, vocabulary
lists, learning progress, VocabTrainer, Spelling Bee, and quizzes.

This directory is its own Go module:

```go
import vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
```

It has no dependency on Wails, React, Bun, or desktop assets. The desktop app and
CLI consume this library through the root module's local `replace` directive.

## Use the SDK

```go
package main

import (
    "context"
    "fmt"
    "log"

    vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
)

func main() {
    client, err := vocab.NewClient()
    if err != nil {
        log.Fatal(err)
    }
    entry, err := client.Dictionary.Lookup(context.Background(), "serendipity")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(entry.Word)
}
```

For another local Go project, require the module and point it at this checkout:

```sh
go mod edit -require=github.com/veryshyjelly/vocabulary-studio/sdk@v0.0.0
go mod edit -replace=github.com/veryshyjelly/vocabulary-studio/sdk=/absolute/path/to/vocabulary-studio/sdk
go mod tidy
```

Authentication, session persistence, rate limiting, and gameplay pacing remain
inside the SDK. Callers should use the SDK's client options rather than duplicate
these mechanisms.

### Authentication lifecycle

Token acquisition is shared only within the current account generation. Logout,
successful login, and importing a non-nil session invalidate earlier token work.
An invalidated `EnsureToken` call returns an error matching `ErrSessionExpired`
and no token; its late response cannot restore credentials or cookies or persist
the previous account. New-account callers do not join the previous refresh.
Canceling one waiting caller still leaves other callers' shared refresh running.

Credential commits and session saves are ordered with logout and session import.
Logout waits for an already committing save before clearing local credentials
and the selected store. A store-clear failure is returned to the caller; local
authentication is still invalidated. These guarantees do not assert remote token
revocation. Custom session stores must not call client lifecycle methods from
their `Load`, `Save`, or `Clear` callbacks, which execute under lifecycle ordering.

Use `client.SetSessionStore(ctx, destination)` to change persistence backends
without replacing the client or its active trainer sessions. It saves the current
snapshot into the actual destination instance before selecting it; a failed save
leaves the previous backend selected. Pending token work and later session saves
use the selected backend after the switch returns. This operation preserves the
authentication generation and pacing state, rejects a nil destination, and does
not clear the previous store. Callers retiring a durable backend own its cleanup.

### Persistence health and retry

`client.PersistenceStatus()` returns a safe copied value with a `state` of
`unknown`, `saved`, or `failed`. It never includes credentials, filesystem paths,
or raw backend error details. Health describes the current account's most recent
save to the selected backend; it is `unknown` before a save and after logout or
session import. A rejected login that restores the previous account preserves
that account's health. Successful saves to memory also report `saved`; this means
the chosen backend accepted the snapshot, not that it survives process exit.

Web login and token refresh retain their successful authentication results when
automatic session saving fails. Inspect persistence health to detect that partial
success, then call `client.FlushSession(ctx)` to retry the selected backend. A
successful retry clears `failed` to `saved`. A failed destination save during
`SetSessionStore` leaves the current backend and its health unchanged; a successful
switch reports `saved`. Persistence health uses a separate state lock from
lifecycle ordering, so store callbacks may inspect `PersistenceStatus()` safely.

## Work on the library

Run these commands from `sdk/`:

```sh
CGO_ENABLED=0 go test ./...
CGO_ENABLED=1 go test -race ./...
go vet ./...
```

`client.go` owns HTTP transport; `auth.go` and `session_store.go` own authentication
and persistence. `trainer.go` contains protocol operations; `trainer_session.go`
contains the state machine. `clock.go` and `pacing.go` own timing abstractions.
Other service files follow their domain names. Tests sit beside their subjects;
`internal/htmlparse/` contains implementation-only parsers, and
`testdata/cassettes/` contains sanitized replay fixtures.

To refresh live fixtures from the repository root, use
`go run ./cmd/record-fixtures`; its default output is `sdk/testdata/cassettes/`.

## Releases

The root module currently uses `v0.0.0` with a local replacement for development.
Before publishing a root-module release, publish the SDK with a submodule tag such
as `sdk/v0.1.0`, then update the root requirement to that published version.
Downstream consumers do not inherit the root module's local replacement.
