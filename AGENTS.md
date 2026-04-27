# AGENTS.md

## 1) Status of this file

All directives in this file are mandatory for this repository.

This file adds project-specific rules on top of the global agent instructions. If this file conflicts with the global instructions, follow this file inside this repository.

---

## 2) Build, lint, and test

Use beads (`bd` command) for issue tracking and history, read /home/chuck/.dotfiles/opencode/BEADS.md

Use the current `src/` layout.

- Primary app build on Windows: `go build -ldflags "-H=windowsgui" -o screen-ocr-llm.exe ./src/main`, `build.cmd`, or `make build-windows`.
- Do not attempt Windows GUI builds in this Linux environment. Leave Windows build verification to a Windows environment unless the user explicitly overrides this.
- Cross-platform builds: use `make build-{windows,macos,macos-arm,linux}` and always target `./src/main`.
- Linux CLI build: from `src/cmd/cli`, run `go build -o ocr-tool .`, or use `make build-cli-linux`.
- Lint: run `go vet ./...`.
- If `golangci-lint` is configured, run `golangci-lint run ./...`.
- All tests: run `go test ./...`.
- Single test: run `go test ./path -run TestName` with the full package path when scripting.

Preferred development loop:

```bash
go fmt ./...
go vet ./...
go test ./...
```

Run platform-specific builds only when they are relevant and supported by the current environment.

---

## 3) Code style and architecture

- Format Go code with `gofmt`.
- Keep import groups in this order: standard library, external packages, then internal `screen-ocr-llm/src/...` packages.
- Prefer concrete types.
- Use small interfaces only at package boundaries.
- Use idiomatic Go `MixedCaps` names.
- Export names only when they are needed across packages or tests.
- Return errors instead of panicking for control flow.
- Wrap errors with useful context.
- Compare errors with `errors.Is` and `errors.As`.
- Use existing logging utilities.
- Keep logs structured and concise, especially around OCR, LLM providers, and timeouts.
- Use `context.Context` and channels for concurrency.
- Avoid data races.
- Respect the existing event loop and worker designs.
- Keep OS-specific code in `*_windows.go` or matching stub files.
- Do not add Windows-only dependencies to shared packages.
- Use the `config` package for configuration.
- Follow the documented `.env`, `SCREEN_OCR_LLM`, and secret file conventions.
- Do not scatter ad hoc environment parsing across packages.
- Keep related logic in existing packages such as `gui`, `overlay`, `screenshot`, `llm`, and `worker`.
- Do not bloat `main` packages.
- Keep integration tests in `tests/`.

---

## 4) Stable contracts

Treat these surfaces as stable APIs:

- `src/messages/`
- `src/router/`
- CLI flags in `src/cmd/cli`

Change them carefully. Update docs and tests when their behavior changes.

---

## 5) Product behavior

- Preserve both execution modes: resident tray app and `--run-once` delegating mode.
- Preserve single-instance behavior through TCP and delegation.
- Do not create duplicate resident instances.
- Keep `src/cmd/cli` GUI-independent.
- Keep CLI behavior file/stdin driven.
- Keep CLI configuration precedence aligned with the documented behavior.
- Respect logging and diagnostics used to debug OCR, LLM providers, and timeouts.
- Do not weaken diagnostics silently.
- Treat `STATUS.md` and related design docs as ground truth for documented architecture and recent fixes unless intentionally updating them.

---

## 6) Repository tooling

- If `.cursor/rules`, `.cursorrules`, or `.github/copilot-instructions.md` appears, treat those rules as authoritative project instructions and update this file to match.
- Before commits or pull requests, run the relevant tests, linters, and builds for the changed area.
- Do not leave stray `fmt.Printf` debug code.
- Do not leave unused imports.
- Do not add heavy dependencies without strong justification.
- Prefer the standard library and existing dependencies.
- During implementation, run incremental tests for changed packages.
- Do not repeatedly rerun unchanged suites for the same issue.
- Run full `go test ./...` once near handoff when changes are cross-cutting or when the user explicitly requests it.

---

## 7) Deep analysis

Before substantial or cross-cutting changes:

1. Read `AUGSTER.xml` fully.
2. Read current `STATUS.md` fully.
3. Follow their rules and task guidance precisely.

---

## 8) Session completion

When ending a work session, complete these steps:

1. Record any remaining follow-up work in the repository's normal tracking system.
2. If code changed, run the relevant quality gates.
3. Update the status of any tracked work.
4. Pull with rebase.
5. Push to remote.
6. Verify `git status` shows the branch is up to date with `origin`.
7. Hand off concise context for the next session.

Work is not complete until `git push` succeeds.

Do not stop with unpushed local commits unless the user explicitly instructs you not to push.
