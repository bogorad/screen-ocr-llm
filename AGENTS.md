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
- Primary app build (Windows GUI): `go build -ldflags "-H=windowsgui" -o screen-ocr-llm.exe ./src/main` or `build.cmd` or `make build-windows`.
- Cross-platform builds: use `make build-{windows,macos,macos-arm,linux}`; always target `./src/main`.
- Linux CLI tool: from `src/cmd/cli`, `go build -o ocr-tool .` (or `make build-cli-linux`).
- Lint: `go vet ./...`; if configured, `golangci-lint run ./...`.
- Tests (all): `go test ./...`.
- Single test: `go test ./path -run TestName` (use full package path when scripting).
- Preferred dev loop: `go fmt ./...`, `go vet ./...`, `go test ./...`, then platform-specific build.

## Code Style & Architecture
- Formatting/imports: run `gofmt`; keep `goimports`-style groups (stdlib, external, then internal `screen-ocr-llm/src/...`).
- Types: favor concrete types; use small interfaces only at boundaries; avoid stuttered names.
- Naming: idiomatic Go `MixedCaps`; export only when needed across packages or tests.
- Errors: use `error` returns (no panics for control flow); wrap with context; compare via `errors.Is/As`.
- Logging: use existing logging utilities; add structured, concise logs (especially around OCR, LLM, providers, and timeouts) consistent with current patterns.
- Concurrency: use `context.Context` and channels; avoid data races; respect existing eventloop/worker designs.
- Platform-specific: keep OS-specific code in `*_windows.go` / `*_stub.go`; do not introduce Windows-only deps into shared packages.
- Configuration: use the `config` package and documented `.env`/`SCREEN_OCR_LLM`/secret file conventions; never ad-hoc env parsing scattered in other packages.
- Layout: Sources in `src/` directory; keep related logic in existing packages (gui, overlay, screenshot, llm, worker, etc.); avoid bloated `main` packages; integration tests in `tests/` directory.
- Public contracts: treat `src/messages/` and `src/router/` (and CLI flags in `src/cmd/cli`) as stable API surfaces; change them carefully and update docs/tests.

## Product & Behavior Expectations
- Preserve the two execution modes: resident tray app and `--run-once` delegating mode; maintain single-instance behavior via TCP and delegation, not duplication.
- For CLI (`src/cmd/cli`), keep it GUI-independent, file/stdin driven, and aligned with documented config precedence.
- Respect logging and diagnostics conventions used to debug OCR/LLM/PROVIDERS/timeouts; do not silently weaken them.
- When STATUS.md or related design docs describe an architecture or recent fixes (callbacks, countdown popup, timeouts), treat them as ground truth unless intentionally updating them.

## Tooling & Meta Rules
- If Cursor/Copilot rules appear (`.cursor/rules`, `.cursorrules`, `.github/copilot-instructions.md`), treat them as authoritative and update this file to match.
- Before commits/PRs: run build + tests for relevant targets; ensure no stray `fmt.Printf`/debug code or unused imports.
- Do not add heavy dependencies without strong justification; prefer stdlib and existing libraries.
- Use beads (`bd`) for all issue/task tracking in this repository; do not use OpenCode's default task mechanism.
- Testing protocol: run incremental tests for changed packages during implementation; do not repeatedly rerun unchanged suites per issue. Run full `go test ./...` once near handoff, or when changes are cross-cutting, or when explicitly requested.

## Deep Analysis Requirement
- Before substantial or cross-cutting changes, ingest `AUGSTER.xml` and current `STATUS.md` fully and follow their rules and task guidance precisely.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

<!-- bv-agent-instructions-v1 -->

---
## Beads Issue Tracking

This project uses **bd** (beads) for issue tracking. Issues are stored in `.beads/` and tracked in git.

### Essential Commands

```bash
bd ready              # Show issues ready to work (no blockers)
bd list --status=open # All open issues
bd show <id>          # Full issue details with dependencies
bd create --title="..." --type=task --priority=2
bd update <id> --status=in_progress
bd close <id> --reason="Completed"
bd close <id1> <id2>  # Close multiple issues at once
bd sync               # Commit and push beads changes
```

### Workflow Pattern

1. **Start**: Run `bd ready` to find actionable work
2. **Claim**: `bd update <id> --status=in_progress` immediately followed `bd sync --flush-only`
3. **Work**: Implement the task
4. **Complete**: `bd close <id>` followed by `bd sync --flush-only`
5. **Sync**: Always run `bd sync` at session end

### Key Concepts

- **Dependencies**: Issues can block other issues. `bd ready` shows only unblocked work.
- **Priority**: P0=critical, P1=high, P2=medium, P3=low, P4=backlog (use numbers, not words)
- **Types**: task, bug, feature, epic, question, docs
- **Blocking**: `bd dep add <issue> <depends-on>` to add dependencies

### Best Practices

- Check `bd ready` at session start to find available work
- Update status as you work (`in_progress` -> closed)
- Create new issues with `bd create` when you discover tasks during work
- Use descriptive titles and set appropriate priority/type
- Always `bd sync` before ending a session

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Use `bd create` for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - `bd close <id>` finished work, `bd update <id> --status=in_progress` for in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

---

## Beads Workflow Integration

This project uses [beads_viewer](https://github.com/Dicklesworthstone/beads_viewer) for issue tracking. Issues are stored in `.beads/` and tracked in git.

### Essential Commands

```bash
# View issues (launches TUI - avoid in automated sessions)
bv

# CLI commands for agents (use these instead)
bd ready              # Show issues ready to work (no blockers)
bd list --status=open # All open issues
bd show <id>          # Full issue details with dependencies
bd create --title="..." --type=task --priority=2
bd update <id> --status=in_progress
bd close <id> --reason="Completed"
bd close <id1> <id2>  # Close multiple issues at once
bd sync               # Commit and push changes
```

### Workflow Pattern

1. **Start**: Run `bd ready` to find actionable work
2. **Claim**: Use `bd update <id> --status=in_progress`
3. **Work**: Implement the task
4. **Complete**: Use `bd close <id>`
5. **Sync**: Always run `bd sync` at session end

### Key Concepts

- **Dependencies**: Issues can block other issues. `bd ready` shows only unblocked work.
- **Priority**: P0=critical, P1=high, P2=medium, P3=low, P4=backlog (use numbers, not words)
- **Types**: task, bug, feature, epic, question, docs
- **Blocking**: `bd dep add <issue> <depends-on>` to add dependencies

### Session Protocol

**Before ending any session, run this checklist:**

```bash
git status              # Check what changed
git add <files>         # Stage code changes
bd sync                 # Commit beads changes
git commit -m "..."     # Commit code
bd sync                 # Commit any new beads changes
git push                # Push to remote
```

### Best Practices

- Check `bd ready` at session start to find available work
- Update status as you work (in_progress → closed)
- Create new issues with `bd create` when you discover tasks
- Use descriptive titles and set appropriate priority/type
- Always `bd sync` before ending session

<!-- end-bv-agent-instructions -->
