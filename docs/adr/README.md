# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records for the Screen OCR LLM project.

## What is an ADR?

An Architecture Decision Record (ADR) is a document that captures an important architectural decision made along with its context and consequences.

## ADR Format

Each ADR follows this structure:
- **Title**: Short noun phrase
- **Status**: Proposed, Accepted, Deprecated, Superseded
- **Context**: What is the issue we're seeing that motivates this decision?
- **Decision**: What decision are we proposing and/or doing?
- **Consequences**: What becomes easier or harder as a result of this change?

## Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [001](001-callback-to-direct-return.md) | Callback to Direct Return Refactoring | Accepted | 2025-10-01 |
| [002](002-configurable-timeout.md) | Configurable Timeout Implementation | Accepted | 2025-11-10 |
| [003](003-codebase-reorganization.md) | Codebase Reorganization (src/tests) | Accepted | 2025-11-10 |
| [004](004-popup-thread-isolation.md) | Popup Thread Isolation Fix | Accepted | 2025-10-01 |
| [005](005-windows-gui-subsystem.md) | Windows GUI Subsystem | Accepted | 2025-10-01 |
| [006](006-dpi-awareness.md) | DPI Awareness Implementation | Accepted | 2025-10-01 |
| [007](007-tcp-single-instance.md) | TCP-based Single Instance | Accepted | 2025-10-01 |
| [008](008-provider-routing.md) | Provider Routing Support | Accepted | 2025-10-01 |

## Creating a New ADR

1. Copy the template: `cp adr-template.md docs/adr/XXX-title.md`
2. Fill in the sections
3. Update this README index
4. Submit for review
