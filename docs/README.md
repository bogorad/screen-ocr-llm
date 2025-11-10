# Documentation

This directory contains all project documentation for Screen OCR LLM.

## Quick Links

- **Project Overview**: [../README.md](../README.md) (at repository root)
- **Build Instructions**: [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md)
- **Contributing**: [AGENTS.md](AGENTS.md) (for AI agents and developers)
- **Architecture Decisions**: [adr/](adr/) (Architecture Decision Records)

## Documentation Structure

```
docs/
├── README.md                      # This file
├── AGENTS.md                      # Guidelines for AI coding agents
├── BUILD_INSTRUCTIONS.md          # How to build the project
├── DOCUMENTATION_REVIEW.md        # Documentation status review
├── REORGANIZATION_SUMMARY.md      # Codebase reorganization details
├── TEST_COVERAGE_ANALYSIS.md      # Test coverage report
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
    └── 008-provider-routing.md
```

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
5. [ADR-002: Configurable Timeout](adr/002-configurable-timeout.md) - User configuration

## Project Status

### Latest Updates (2025-11-10)

- ✅ Configurable timeout implemented (ADR-002)
- ✅ Codebase reorganized to src/tests structure (ADR-003)
- ✅ All documentation updated and moved to docs/
- ✅ ADRs created for major architectural decisions

### Test Coverage

**55% of packages have tests** (11/20)
- Well tested: config, hotkey, worker, singleinstance, cmd/cli
- Partially tested: clipboard, gui, llm, ocr, screenshot
- **Critical gaps**: eventloop, notification, overlay, popup, main

See [TEST_COVERAGE_ANALYSIS.md](TEST_COVERAGE_ANALYSIS.md) for details.

### Documentation Health

**100% current** (as of 2025-11-10)
- All documentation reflects src/tests structure
- All build commands updated
- ADRs capture major decisions

See [DOCUMENTATION_REVIEW.md](DOCUMENTATION_REVIEW.md) for review details.

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
3. Check test coverage analysis for tested areas
4. Create an issue with relevant details

---

**Last Updated**: 2025-11-10
