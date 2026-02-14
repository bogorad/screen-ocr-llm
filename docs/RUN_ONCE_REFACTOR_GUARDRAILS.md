# Run-Once Refactor Guardrails

This document defines behavior that MUST remain unchanged while refactoring
resident and `--run-once` execution paths.

## Behavior Matrix

| Flow | Trigger | Region Selection Owner | OCR Executor | Result Target | Popup Behavior | Exit/Response Behavior |
| --- | --- | --- | --- | --- | --- | --- |
| Resident hotkey | Global hotkey callback into event loop | Resident event loop (`overlay.Select`) | Worker pool (`worker.Pool`) | Clipboard | Countdown starts before OCR; success updates popup text; OCR error closes popup; clipboard error closes + shows clipboard error popup | Resident process stays alive |
| Delegated `--run-once` (resident active) | Client `TryRunOnce(..., stdout=false)` | Resident event loop | Worker pool | Clipboard (for `--run-once`) | Resident starts countdown before OCR, updates popup text on success, closes on errors | Resident returns `SUCCESS`/`ERROR`; delegator exits if success; on delegation error, caller falls back to standalone |
| Standalone `--run-once` fallback (no resident) | `TryRunOnce` returns `delegated=false` OR delegation error fallback branch | Local process (`gui.StartRegionSelection`) | Local process (`ocr.Recognize`) | Clipboard for `--run-once` | Local countdown starts before OCR; success updates popup text and keeps visible briefly; errors close popup | Process exits non-zero on errors, zero on success |
| Busy handling | Concurrent trigger/request while resident busy | None | None | None | Hotkey path shows "Busy, please retry" popup; delegated path sends busy error response | Delegated caller receives `ERROR` and enters existing fallback behavior |

## Non-Negotiable Invariants

1. Delegation stays enabled for `--run-once` and continues to use TCP loopback.
2. Single resident ownership remains enforced via configured start port binding.
3. Busy gating remains serialized in event loop and blocks concurrent OCR starts.
4. Countdown popup starts before OCR execution for hotkey, delegated, and standalone run-once flows.
5. Existing destination semantics remain unchanged:
   - resident hotkey -> clipboard
   - delegated `--run-once` -> clipboard
   - standalone `--run-once` -> clipboard
6. Existing API key/config precedence behavior remains unchanged.

## Refactor Guard Checklist

Before closing each child issue under `screen-ocr-llm-tyn`:

- [ ] No behavior change against the matrix above.
- [ ] No new duplicated OCR orchestration blocks introduced.
- [ ] Error messages on user-facing failures remain equivalent in meaning.
- [ ] Incremental tests for changed packages pass.

Final verification before closing epic:

- [ ] `go test ./... -count=1` passes.
- [ ] `go vet` passes for changed packages.
- [ ] Windows build target and Linux CLI build target succeed.
- [ ] No executable artifacts are tracked or left in working tree.
