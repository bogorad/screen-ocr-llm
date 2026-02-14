# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.6.0] - 2026-02-14

### Added
- Lasso selection mode in the Windows overlay, toggled with `Space`, with loop-completion on mouse-up near the start point.
- Polygon-aware capture path: selected lasso polygons are preserved in region metadata and applied as a mask before OCR.
- Embedded custom lasso cursor asset (`src/gui/lasso.cur`) used in lasso mode, with safe fallback to system hand cursor.
- Configurable initial selection mode via `DEFAULT_MODE` and CLI `--default-mode` (`rect`, `rectangle`, `lasso`).

### Changed
- Selection now supports both rectangle and lasso flows while preserving the rectangular PNG payload contract to OCR/LLM.
- In lasso mode, pixels outside the selected polygon are filled solid white prior to OCR.
- Resident and run-once overlays now share the same default-mode initialization behavior.

### Fixed
- Resident hotkey flow now handles `Space` and `Escape` reliably even when overlay `WM_KEYDOWN` is not delivered.
- Debounced and edge-detected key handling to avoid missed quick taps and accidental rapid double-toggle.

## [2.5.1] - 2026-02-14

### Added
- Cobra-based flag parsing for resident app, Linux CLI, and stress tool.
- Configurable API key file path resolution via `--api-key-path` and `OPENROUTER_API_KEY_FILE` with documented precedence.
- Hotkey keycode mapping for `F13` through `F24`.
- Shared runtime bootstrap and shared OCR session execution pipeline to reduce duplicated resident/run-once logic.
- Run-once refactor guardrails document in `docs/RUN_ONCE_REFACTOR_GUARDRAILS.md`.
- `.justfile` for chooser default, Windows app build, Linux CLI build, and test recipes.

### Changed
- Eventloop hotkey and delegated request paths now share the same internal request/result handling flow.
- Standalone `--run-once` fallback now uses shared session execution while preserving delegation behavior.
- Linux build target guidance aligned to CLI-only binary expectations.

### Fixed
- Full test suite reliability issues in `tests/`, `src/gui`, and `src/singleinstance`.
- Platform compile coverage by moving Windows-only behavior behind `*_windows.go` and `*_stub.go` boundaries.
- TCP address composition updated to use `net.JoinHostPort`, eliminating IPv6 formatting vet warnings.

## [2.5.0] - 2025-12-14

### Added
- Comprehensive diagnostic logging for multi-monitor troubleshooting
- Virtual screen coordinate offset support for vertical monitor arrangements

### Fixed
- Multi-monitor overlay cursor focus issues (SetForegroundWindow failures)
- Incorrect region coordinates on monitors positioned above/below primary display
- Cursor display during region selection (ensures crosshairs instead of circle)
- Coordinate offset calculations for non-zero virtual screen origins

### Changed
- Enhanced window focus management with BringWindowToTop fallback
- Improved cursor loading and caching for consistent display

## [2.4.0] - 2025-11-10

### Added
- Configurable OCR timeout via `OCR_DEADLINE_SEC` environment variable (default: 20s)
- Architecture Decision Records (ADRs) in `docs/adr/`
- Comprehensive test coverage analysis documentation
- Documentation organization with `docs/` directory
- 8 ADRs documenting key architectural decisions

### Changed
- **[BREAKING]** Reorganized codebase: all packages moved to `src/` directory
- **[BREAKING]** All imports changed from `screen-ocr-llm/package` to `screen-ocr-llm/src/package`
- **[BREAKING]** Build command changed from `./main` to `./src/main`
- Moved all documentation to `docs/` directory (except README.md)
- Integration tests moved to `tests/` directory
- OCR timeout default increased from 15s to 20s
- Updated all documentation to reflect new structure

### Fixed
- Context leaks in eventloop timeout handling (added proper cleanup)
- Restored accidentally deleted `handleResult()` and `handleHotkey()` functions
- Fixed `handleConn()` to properly pass connection to callbacks

### Documentation
- Created comprehensive ADRs for major architectural decisions
- Updated all build instructions and developer guides
- Added `docs/README.md` as documentation index
- Reviewed and updated all documentation files (100% current)

## [2.3.0] - 2025-10-01

### Added
- Startup LLM connectivity check with blocking error dialog
- DPI awareness for high-DPI displays and multi-monitor setups
- Config load order: `.env` in executable directory, then `SCREEN_OCR_LLM` path
- Countdown popup during OCR processing
- Provider routing support via `PROVIDERS` configuration
- Comprehensive logging for OCR pipeline and provider usage

### Changed
- Windows builds now use GUI subsystem (no console window)
- Thread isolation fix: main goroutine locked to OS thread
- Config respects `ENABLE_FILE_LOGGING` properly (no forced early logging)

### Fixed
- Callback overwriting issue in region selection (refactored to direct return)
- Popup thread isolation (WM_EXIT_LOOP instead of PostQuitMessage)
- Second hotkey activation failure (message queue cleanup)
- Consecutive hotkey presses stability
- High-DPI selection coverage (full virtual screen support)
- Delegated --run-once countdown reliability

### Security
- LLM ping validates connectivity before running resident
- Delegated --run-once clients skip redundant ping

## [2.2.0] - Earlier

### Added
- TCP-based single instance detection and delegation
- --run-once mode for command-line invocations
- Resident mode with system tray integration
- Global hotkey support (configurable)

### Features
- Region selection with mouse drag
- OCR via OpenRouter vision models
- Clipboard integration
- Multi-monitor support
- Configurable providers and models

## [2.1.0] - Earlier

### Added
- Linux CLI tool (`src/cmd/cli`)
- JSON output support for CLI
- Stdin input support for pipelines

## [2.0.0] - Earlier

### Added
- Initial Go rewrite from Python
- Windows GUI implementation
- OpenRouter API integration

---

## Version History

- **2.6.0** (2026-02-14) - Lasso mode + masked capture + default mode config + custom lasso cursor
- **2.5.1** (2026-02-14) - Cobra migration + key file path support + delegation-preserving refactor
- **2.5.0** (2025-12-14) - Multi-monitor fixes + diagnostic logging
- **2.4.0** (2025-11-10) - Configurable timeout + codebase reorganization + ADRs
- **2.3.0** (2025-10-01) - DPI awareness + thread isolation + provider routing
- **2.2.0** - Single instance + delegation
- **2.1.0** - Linux CLI tool
- **2.0.0** - Initial Go implementation

[2.6.0]: https://github.com/user/screen-ocr-llm/compare/2.5.1...2.6.0
[2.5.1]: https://github.com/user/screen-ocr-llm/compare/2.5.0...2.5.1
[2.5.0]: https://github.com/user/screen-ocr-llm/compare/2.4...2.5.0
[2.4.0]: https://github.com/user/screen-ocr-llm/compare/2.3...2.4.0
[2.3.0]: https://github.com/user/screen-ocr-llm/compare/2.2...2.3.0
[2.2.0]: https://github.com/user/screen-ocr-llm/compare/2.1...2.2.0
[2.1.0]: https://github.com/user/screen-ocr-llm/compare/2.0...2.1.0
[2.0.0]: https://github.com/user/screen-ocr-llm/releases/tag/2.0
