# AGENTS

This document defines how automated coding agents should operate in this repository. Follow it strictly.

## Build, Lint, Test
- Primary app build (Windows GUI): `go build -ldflags "-H=windowsgui" -o screen-ocr-llm.exe ./main` or `build.cmd` or `make build-windows`.
- Cross-platform builds: use `make build-{windows,macos,macos-arm,linux}`; always target `./main` (never plain `go build` in repo root).
- Linux CLI tool: from `cmd/cli`, `go build -o ocr-tool .` (or `make build-cli-linux`).
- Lint: `go vet ./...`; if configured, `golangci-lint run ./...`.
- Tests (all): `go test ./...`.
- Single test: `go test ./path -run TestName` (use full package path when scripting).
- Preferred dev loop: `go fmt ./...`, `go vet ./...`, `go test ./...`, then platform-specific build.

## Code Style & Architecture
- Formatting/imports: run `gofmt`; keep `goimports`-style groups (stdlib, external, then internal `screen-ocr-llm/...`).
- Types: favor concrete types; use small interfaces only at boundaries; avoid stuttered names.
- Naming: idiomatic Go `MixedCaps`; export only when needed across packages or tests.
- Errors: use `error` returns (no panics for control flow); wrap with context; compare via `errors.Is/As`.
- Logging: use existing logging utilities; add structured, concise logs (especially around OCR, LLM, providers, and timeouts) consistent with current patterns.
- Concurrency: use `context.Context` and channels; avoid data races; respect existing eventloop/worker designs.
- Platform-specific: keep OS-specific code in `*_windows.go` / `*_stub.go`; do not introduce Windows-only deps into shared packages.
- Configuration: use the `config` package and documented `.env`/`SCREEN_OCR_LLM`/secret file conventions; never ad-hoc env parsing scattered in other packages.
- Layout: keep related logic in existing packages (gui, overlay, screenshot, llm, worker, etc.); avoid bloated `main` packages.
- Public contracts: treat `messages/` and `router/` (and CLI flags in `cmd/cli`) as stable API surfaces; change them carefully and update docs/tests.

## Product & Behavior Expectations
- Preserve the two execution modes: resident tray app and `--run-once` delegating mode; maintain single-instance behavior via TCP and delegation, not duplication.
- For CLI (`cmd/cli`), keep it GUI-independent, file/stdin driven, and aligned with documented config precedence.
- Respect logging and diagnostics conventions used to debug OCR/LLM/PROVIDERS/timeouts; do not silently weaken them.
- When STATUS.md or related design docs describe an architecture or recent fixes (callbacks, countdown popup, timeouts), treat them as ground truth unless intentionally updating them.

## Tooling & Meta Rules
- If Cursor/Copilot rules appear (`.cursor/rules`, `.cursorrules`, `.github/copilot-instructions.md`), treat them as authoritative and update this file to match.
- Before commits/PRs: run build + tests for relevant targets; ensure no stray `fmt.Printf`/debug code or unused imports.
- Do not add heavy dependencies without strong justification; prefer stdlib and existing libraries.

## Deep Analysis Requirement
- Before substantial or cross-cutting changes, ingest `AUGSTER.xml` and current `STATUS.md` fully and follow their rules and task guidance precisely.