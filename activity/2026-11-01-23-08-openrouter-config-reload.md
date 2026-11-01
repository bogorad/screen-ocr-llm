# OpenRouter startup and resident config reload

Decision: Accept a successful reasoning-only one-token ping as a valid startup response. Keep API and choice errors fatal.

Change: Add structured request and response diagnostics without payloads or keys. Read fresh config snapshots, apply resident changes from Windows directory notifications, and retain the last valid config on reload failure. Preserve logging on log-file open failure and remove raw keyboard-event logs.

Reason: The selected model used its one ping token for reasoning, so startup failed despite HTTP 200. File settings also needed to apply while the resident runs without holding the config file open.
