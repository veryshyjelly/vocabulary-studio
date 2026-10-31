.PHONY: build frontend bindings test test-sdk test-desktop test-frontend package-macos

build: frontend
	go build -o bin/vocab-desktop ./cmd/vocab-desktop

frontend:
	cd frontend && bun install --frozen-lockfile && bun run build

bindings: frontend
	wails3 generate bindings -ts -i -names -d frontend/src/bindings ./cmd/vocab-desktop
	cd frontend && bun run build

test: test-sdk test-desktop test-frontend

test-sdk:
	cd sdk && CGO_ENABLED=0 go test ./...

test-desktop:
	CGO_ENABLED=0 go test ./internal/... ./cmd/vocab-cli ./cmd/record-fixtures

test-frontend:
	cd frontend && bun install --frozen-lockfile && bun run typecheck --noUnusedLocals && bun run test

package-macos:
	./scripts/package-macos.sh
