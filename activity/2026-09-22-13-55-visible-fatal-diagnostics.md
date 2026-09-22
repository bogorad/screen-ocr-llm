# Visible fatal diagnostics

Startup failures previously could terminate the Windows GUI application with
no visible error. File logging now starts before resident port checks. Fatal
startup, event-loop, and standalone capture errors use one blocking dialog
with the error, executable, working directory, and logging guidance.

The existing optional file-logging policy remains in effect. Selection
cancellation remains silent, and startup LLM errors use the common dialog to
avoid duplicate notifications. Diagnostic records include a stable event name
and process context. The operator corrected the API key separately.
