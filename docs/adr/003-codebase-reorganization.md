# ADR-003: Codebase Reorganization (src/tests)

## Status

Accepted

## Date

2025-11-10

## Context

The repository had a flat structure with 20+ package directories at the root level, plus test files scattered throughout. This made the root directory cluttered and hard to navigate.

**Before:**
```
screen-ocr-llm/
├── clipboard/
├── config/
├── main/
├── ... (20 package directories)
├── api_debug_test.go
├── test_*.go
└── ...
```

**Problems:**
- Cluttered root directory
- Hard to distinguish config files from code
- No clear separation between source and tests
- Non-standard Go project layout

## Decision

Reorganize into a cleaner structure following Go community best practices:

**After:**
```
screen-ocr-llm/
├── src/               # All source code packages
│   ├── clipboard/
│   ├── config/
│   ├── main/
│   ├── cmd/
│   └── ... (all 20 packages)
├── tests/             # Integration and debug tests
│   ├── api_debug_test.go
│   ├── debug_test.go
│   └── test_*.go
├── docs/              # All documentation
├── .env
└── README.md
```

**Implementation:**
1. Create `src/` and `tests/` directories
2. Move all 20 package directories to `src/`
3. Move integration/debug tests to `tests/`
4. Keep unit tests with their packages in `src/`
5. Update all imports: `screen-ocr-llm/package` → `screen-ocr-llm/src/package`
6. Update build scripts (`build.cmd`, `Makefile`)
7. Update documentation (README.md, AGENTS.md, BUILD_INSTRUCTIONS.md)

**Files Moved:**
- **To src/**: 20 package directories (clipboard, config, eventloop, gui, hotkey, llm, logutil, main, messages, notification, ocr, overlay, popup, process, router, screenshot, singleinstance, tray, worker, cmd)
- **To tests/**: 9 integration test files
- **Unit tests**: Remain with their packages in `src/`

## Consequences

### Positive

- **Cleaner root**: Config files and documentation more visible
- **Clear separation**: Source code in `src/`, tests in `tests/`
- **Better organization**: All packages grouped together
- **Standard structure**: Follows common Go project layout patterns
- **Easier navigation**: Clear hierarchy for new contributors
- **Scalability**: Easy to add more packages without cluttering root

### Negative

- **Breaking change**: All imports changed
- **Build command changes**: Must use `./src/main` instead of `./main`
- **Migration needed**: Developers with local changes must update
- **More typing**: Slightly longer import paths

### Neutral

- File history preserved (used `git mv`)
- Test structure remains the same (Go finds tests anywhere)
- Binary functionality unchanged

## References

- Import path pattern: `screen-ocr-llm/src/package`
- Build command: `go build -ldflags "-H=windowsgui" -o screen-ocr-llm.exe ./src/main`
- All documentation updated: README.md, AGENTS.md, BUILD_INSTRUCTIONS.md
- Verified working: Build successful, OCR test passed (2,288 chars)
- See: REORGANIZATION_SUMMARY.md for complete details
