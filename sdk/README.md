# Vocabulary.com Go SDK

A standalone, pure-Go library for authentication, dictionary lookups, vocabulary
lists, learning progress, VocabTrainer, Spelling Bee, and quizzes.

This directory is its own Go module:

```go
import vocab "github.com/veryshyjelly/vocabulary_go/sdk"
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

    vocab "github.com/veryshyjelly/vocabulary_go/sdk"
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
go mod edit -require=github.com/veryshyjelly/vocabulary_go/sdk@v0.0.0
go mod edit -replace=github.com/veryshyjelly/vocabulary_go/sdk=/absolute/path/to/vocabulary_go/sdk
go mod tidy
```

Authentication, session persistence, rate limiting, and gameplay pacing remain
inside the SDK. Callers should use the SDK's client options rather than duplicate
these mechanisms.

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

## Compatibility and releases

The original `github.com/veryshyjelly/vocabulary_go` import remains a deprecated
compatibility facade. It aliases the canonical SDK types and delegates functions;
new code should import this module directly.

The root module currently uses `v0.0.0` with a local replacement for development.
Before publishing a root-module release, publish the SDK with a submodule tag such
as `sdk/v0.1.0`, then update the root requirement to that published version.
Downstream consumers do not inherit the root module's local replacement.
