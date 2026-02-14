# Documentation

This directory contains all project documentation for Screen OCR LLM.

## Quick Links

- **Project Overview**: [../README.md](../README.md) (at repository root)
- **Build Instructions**: [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md)
- **Release Notes**: [releases/](releases/) (detailed per-version docs)
- **Contributing**: [AGENTS.md](AGENTS.md) (for AI agents and developers)
- **Architecture Decisions**: [adr/](adr/) (Architecture Decision Records)

## Documentation Structure

```
docs/
├── README.md                      # This file
├── AGENTS.md                      # Guidelines for AI coding agents
├── BUILD_INSTRUCTIONS.md          # How to build the project
├── RUN_ONCE_REFACTOR_GUARDRAILS.md # Run-once delegation guardrails
├── releases/                      # Detailed release notes
│   ├── README.md                  # Release notes index
│   └── 2.6.0.md                   # Release 2.6.0 details
└── adr/                          # Architecture Decision Records
    ├── README.md                  # ADR index
    ├── adr-template.md           # Template for new ADRs
    ├── 001-callback-to-direct-return.md
    ├── 002-configurable-timeout.md
    ├── 003-codebase-reorganization.md
    ├── 004-popup-thread-isolation.md
    ├── 005-windows-gui-subsystem.md
    ├── 006-dpi-awareness.md
    ├── 007-tcp-single-instance.md
    ├── 008-provider-routing.md
    ├── 009-multi-monitor-support.md
    └── 010-lasso-selection-and-masked-capture.md
```

## Architecture Overview

### Directory Structure

- `src/` - All source code packages
- `tests/` - Integration and debug test files
- Unit tests (`*_test.go`) remain with their respective packages

At a high level, the Windows app is structured as:

- `src/main`:
  - Parses Cobra flags (`--run-once`, `--api-key-path`, `--default-mode`) and keeps compatibility with legacy single-dash forms.
  - Ensures single resident instance via a TCP preflight on the configured port.
  - Uses shared runtime bootstrap (`src/runtimeinit`) to load config, set logging, initialize OCR/LLM/clipboard dependencies, and perform startup ping checks.
  - Enables DPI awareness and monitor diagnostics via Windows-specific helpers.
  - In resident mode:
    - Starts the central event loop (`src/eventloop`), the system tray (`src/tray`), and global hotkey listener (`src/hotkey`).
  - In `--run-once` mode:
    - First tries to delegate to a running resident via `src/singleinstance.Client`.
    - If no resident is available, runs a standalone capture+OCR flow.

- `src/eventloop`:
  - Owns the single-instance TCP server (`src/singleinstance.Server`).
  - Listens for:
    - Global hotkey triggers.
    - Delegated `--run-once` requests.
  - For each request:
    - Uses `src/overlay.Selector`/`src/gui` to run the interactive region selector.
    - Submits OCR work to a bounded worker pool (`src/worker` + `src/ocr` + `src/llm`).
    - Routes completion through shared result-target behavior (clipboard/stdout/delegated response), updates UI via `src/popup`, and preserves busy semantics.
  - Enforces that only one OCR job runs at a time ("busy" behavior).

- `src/session`:
  - Provides a shared OCR session executor for region selection, countdown popup lifecycle, OCR with deadline, and pluggable output targets.
  - Used by standalone run-once fallback and shared with resident-related result target logic.

- `src/overlay` + `src/gui` + `src/screenshot`:
  - Implement the Windows overlay window with rectangle/lasso region selection.
  - Capture the selected region (multi-monitor aware) as PNG bytes for OCR.
  - In lasso mode, capture the bounding rectangle and fill outside-polygon pixels with white before OCR.

- `src/llm` + `src/ocr`:
  - `src/ocr` captures the region and forwards it to `src/llm`.
  - `src/llm` calls the OpenRouter Chat Completions API with a strict OCR-style prompt and optional `PROVIDERS` routing.

- `src/singleinstance`:
  - TCP-based discovery and delegation so `--run-once` clients hand work to the resident when available.

- `src/tray` + `src/popup` + `src/notification`:
  - System tray icon, About/Exit menu, and small non-intrusive popups.
  - Countdown popup appears during OCR and is updated/closed when results arrive.

## Prerequisites

- Go toolchain installed (`go build`) if you want to build locally; otherwise use `.exe` from releases.
- Windows (current overlay/hotkey path targets Windows)
- OpenRouter API key and a vision-capable model (`:free` models are also supported, but not recommended)

## Linux CLI Tool

The Linux CLI (`src/cmd/cli`) is a separate, GUI-free binary that reuses `src/config` and `src/llm` to run OCR on PNG input (file or stdin) and print plain-text or JSON output.

```sh
# Build
cd src/cmd/cli
go build -o ocr-tool .

# Usage
./ocr-tool --file screenshot.png
```

See [../src/cmd/cli/README.md](../src/cmd/cli/README.md) for details.

## Runtime Configuration Sources and Precedence

This section is the canonical reference for config source loading and precedence behavior.

### Dotenv source resolution

At startup, the app loads one dotenv source in this order:

1. `.env` in the executable directory
2. `SCREEN_OCR_LLM` path (only if executable-local `.env` is missing)

### API key file path precedence (`OPENROUTER_API_KEY_FILE` / `--api-key-path`)

From lowest to highest precedence:

1. Built-in default: `/run/secrets/api_keys/openrouter`
2. Process environment variable `OPENROUTER_API_KEY_FILE`
3. Loaded dotenv value `OPENROUTER_API_KEY_FILE`
4. CLI argument `--api-key-path`

### API key value precedence

From highest to fallback:

1. Content of the resolved API key file path (if the file exists and is non-empty)
2. `OPENROUTER_API_KEY` environment variable

### Default selection mode precedence (`DEFAULT_MODE` / `--default-mode`)

From highest to fallback:

1. CLI argument `--default-mode`
2. `DEFAULT_MODE` in environment/dotenv
3. Default: `rectangle`

Accepted values for env/CLI mode selection: `rect`, `rectangle`, `lasso`.

### Delegation precedence in `--run-once`

If `--run-once` delegates to an already-running resident instance, resident configuration remains authoritative.
Client-side `--api-key-path` and `--default-mode` do not override the resident process.

## For Developers

### Getting Started

1. Read [../README.md](../README.md) for project overview
2. Read [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md) to set up your environment
3. Read [AGENTS.md](AGENTS.md) for coding guidelines and standards
4. Browse [adr/](adr/) to understand key architectural decisions

### For AI Coding Agents

**Start here**: [AGENTS.md](AGENTS.md)

This document defines:
- Build, lint, and test commands
- Code style and architecture guidelines
- Product behavior expectations
- Meta rules for tooling

### Understanding the Architecture

Read these ADRs in order:
1. [ADR-007: TCP-based Single Instance](adr/007-tcp-single-instance.md) - Core delegation mechanism
2. [ADR-001: Callback to Direct Return](adr/001-callback-to-direct-return.md) - Region selection flow
3. [ADR-004: Popup Thread Isolation](adr/004-popup-thread-isolation.md) - UI reliability
4. [ADR-006: DPI Awareness](adr/006-dpi-awareness.md) - Multi-monitor support
5. [ADR-009: Multi-Monitor Support](adr/009-multi-monitor-support.md) - Virtual-screen coordinate correctness
6. [ADR-010: Lasso Selection and Masked Capture](adr/010-lasso-selection-and-masked-capture.md) - Free-form selection with rectangular payload contract
7. [ADR-002: Configurable Timeout](adr/002-configurable-timeout.md) - User configuration

## Project Status

### Latest Updates (2026-02-14)

- ✅ Lasso selection mode added with polygon-masked capture for OCR
- ✅ Configurable initial selection mode (`DEFAULT_MODE`, `--default-mode`)
- ✅ Embedded custom lasso cursor integrated into overlay mode switching
- ✅ ADR-010 added for lasso mode and masked capture contract

### Documentation Health

**100% current** (as of 2025-11-10)
- Core docs reflect current src/tests structure and active release behavior
- All build commands updated
- ADRs capture major decisions

## Contributing

### Before Making Changes

1. Read [AGENTS.md](AGENTS.md) for guidelines
2. Check [adr/](adr/) to understand existing decisions
3. Run tests: `go test ./...`
4. Run lint: `go vet ./...`
5. Format code: `go fmt ./...`

### Proposing Architecture Changes

1. Create a new ADR using [adr/adr-template.md](adr/adr-template.md)
2. Document context, decision, and consequences
3. Update [adr/README.md](adr/README.md) index
4. Submit for review

### Documentation Updates

When making changes:
- Update relevant markdown files
- Keep README.md as the entry point
- Document breaking changes
- Update ADRs if decisions change

## External Documentation

### API References

- **OpenRouter**: https://openrouter.ai/docs
- **Go Documentation**: https://golang.org/doc/
- **Windows API**: https://learn.microsoft.com/en-us/windows/win32/

### Related Projects

- **Original Python version**: https://github.com/cherjr/screen-ocr-llm
- **Go Screenshots**: https://github.com/kbinani/screenshot

## Support

For issues, questions, or contributions:
1. Check existing documentation first
2. Review ADRs for architectural context
3. Check release notes for tested and changed areas
4. Create an issue with relevant details

---

**Last Updated**: 2026-02-14
