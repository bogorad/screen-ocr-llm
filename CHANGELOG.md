# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

- **2.4.0** (2025-11-10) - Configurable timeout + codebase reorganization + ADRs
- **2.3.0** (2025-10-01) - DPI awareness + thread isolation + provider routing
- **2.2.0** - Single instance + delegation
- **2.1.0** - Linux CLI tool
- **2.0.0** - Initial Go implementation

[2.4.0]: https://github.com/user/screen-ocr-llm/compare/2.3...2.4.0
[2.3.0]: https://github.com/user/screen-ocr-llm/compare/2.2...2.3.0
[2.2.0]: https://github.com/user/screen-ocr-llm/compare/2.1...2.2.0
[2.1.0]: https://github.com/user/screen-ocr-llm/compare/2.0...2.1.0
[2.0.0]: https://github.com/user/screen-ocr-llm/releases/tag/2.0
