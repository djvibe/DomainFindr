# Repository Guidelines

## Project Structure & Module Organization

This repository contains a Go CLI for domain availability checks and domain-consultant session workflows.

- `cmd/domainfindr/`: executable entrypoint.
- `internal/app/`: CLI argument parsing and top-level orchestration.
- `internal/consult/`: brief parsing and consultant session artifact generation.
- `internal/input/`: Markdown and CSV parsing.
- `internal/lookup/`: RDAP lookup client and lookup-specific tests.
- `internal/runner/`: worker pool, retry logic, and rate limiting.
- `internal/output/`: table, CSV, and JSON formatting.
- `internal/model/`: shared result and input types.
- `docs/features/`: tracked feature planning and completed launch docs.
- `docs/domain_research_lessons.md`: persistent lessons learned from domain strategy sessions.
- `workspace/`: local-only batch input/output area. This path is Git-ignored.

Keep new production code under `internal/` unless it is the binary entrypoint.

Before starting new domain-strategy or naming workflow changes, review `docs/domain_research_lessons.md` for session-to-session context and keep consultant artifacts aligned with the `workspace/results/` conventions.

Default branch policy:

- use `main` as the default working and integration branch
- only create or use feature branches when explicitly requested
- if work starts on a side branch by mistake, move the commits back to `main` before considering the task complete

## Build, Test, and Development Commands

Use the Go toolchain installed at `/usr/local/go/bin`, or ensure it is on `PATH`.

- `go test ./...`: run the full test suite.
- `gofmt -w cmd internal`: format all Go source files.
- `go build -o domainFindr ./cmd/domainfindr`: build the local binary.
- `go run ./cmd/domainfindr openai.com`: run the CLI without building first.
- `go run ./cmd/domainfindr consult --brief brief.md --input candidates.md`: create a consultant session with optional availability sweep.

If sandboxed environments block the default build cache, use:
`GOCACHE=/tmp/domainfindr-go-build-cache go test ./...`

For live RDAP or registrar verification commands, prefer running outside the sandbox when needed because network access and default state/history paths are commonly blocked in sandboxed sessions. This repo guidance does not override platform approval requirements.

## Coding Style & Naming Conventions

Follow standard Go conventions.

- Format all code with `gofmt`.
- Use tabs as emitted by `gofmt`; do not hand-align indentation.
- Keep packages small and focused by behavior (`input`, `lookup`, `runner`).
- Exported names use `CamelCase`; unexported helpers use `camelCase`.
- Prefer standard library dependencies unless a new dependency is clearly justified.

## Testing Guidelines

Tests use Go’s built-in `testing` package and live next to the code as `*_test.go`.

- Name tests as `TestXxx`.
- Prefer deterministic unit tests over live network calls.
- Mock HTTP behavior in lookup tests rather than relying on external RDAP services.
- When explicitly validating live registrar integrations, use escalated execution as needed instead of treating sandbox network failures as product failures.
- Run `go test ./...` before opening a PR.
- When changing consultant mode, verify the generated artifact set and folder naming remain consistent with `workspace/results/<session>-<date>/`.

## Commit & Pull Request Guidelines

Use simple, imperative commit messages such as:
`Add positional domain arguments to CLI`

For pull requests:

- describe the user-visible change,
- list test coverage added or updated,
- note any environment requirements or follow-up work,
- include example CLI usage when changing command behavior.

## Security & Configuration Tips

Do not hardcode API keys or credentials. Registrar verification already uses environment-driven credentials where needed. Keep future provider or purchase credentials in environment variables and document them in `docs/` instead of committing secrets.
