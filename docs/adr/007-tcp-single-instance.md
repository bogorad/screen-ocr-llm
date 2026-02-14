# ADR-007: TCP-based Single Instance

## Status

Accepted

## Date

2025-10-01

## Context

The application needs to:
1. Ensure only one resident instance runs at a time
2. Allow `--run-once` clients to delegate to the resident
3. Prevent conflicts when multiple users invoke the tool
4. Support communication between client and resident

**Requirements:**
- Single resident instance enforcement
- Client-to-resident delegation
- Request/response protocol
- Port range for avoiding conflicts
- Clean shutdown handling

**Alternatives Considered:**
- **Named pipes**: Windows-specific, complex permissions
- **File locking**: Race conditions, cleanup issues
- **Windows mutexes**: No communication channel
- **TCP loopback**: Chosen for simplicity and testability

## Decision

Implement TCP-based single instance detection and delegation:

**Architecture:**
```
Resident:
  1. Bind to start port in configured range (default 49500-49550)
  2. Listen for client connections
  3. Accept delegation requests
  4. Process OCR and respond

Client (--run-once):
  1. Scan port range for resident
  2. If found: Delegate request and exit
  3. If not found: Run standalone OCR
```

**Protocol:**
```text
# discovery
Client -> Resident: PING\n
Resident -> Client: PONG\n

# request
Client -> Resident: CLIPBOARD\n | STDOUT\n

# response success
Resident -> Client: SUCCESS\n<optional payload>

# response error
Resident -> Client: ERROR\n<error message>
```

**Configuration:**
```bash
# .env
SINGLEINSTANCE_PORT_START=49500
SINGLEINSTANCE_PORT_END=49550
```

**Implementation:**
```go
// Resident startup
server := singleinstance.NewServer()
if err := server.Start(ctx); err != nil {
    // Port in use → another resident exists
    log.Fatal("Resident already running")
}

// Client delegation
client := singleinstance.NewClient()
delegated, response, err := client.TryRunOnce(ctx, stdout)
if delegated {
    // Request handled by resident
    return
}
// No resident, run standalone
```

**Pre-flight Check:**
```go
// main.go - before starting resident
startPort, _ := singleinstance.GetPortRangeForDebug()
listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", startPort))
if err != nil {
    log.Fatal("Resident already running on port", startPort)
}
listener.Close()
```

## Consequences

### Positive

- **Simple protocol**: line-based text framing over loopback TCP
- **Loopback only**: No network exposure (127.0.0.1)
- **Port range**: Avoids conflicts (51 ports available)
- **Clean delegation**: Client exits after delegating
- **Testable**: Easy to write integration tests
- **Cross-platform compatible**: Works on Linux/Mac if needed

### Negative

- **Port conflicts possible**: If all 50 ports in use
- **Firewall warnings**: Some security software may alert on localhost binding
- **Port scanning overhead**: Client scans range to find resident
- **State management**: Server must track busy/idle state

### Neutral

- TCP overhead negligible for loopback
- Port range configurable via environment variables
- Alternative named pipes not needed (TCP simpler)

## References

- Package: `src/singleinstance`
- Default range: 49500-49550 (51 ports)
- Environment: `SINGLEINSTANCE_PORT_START`, `SINGLEINSTANCE_PORT_END`
- Protocol: `PING/PONG`, then `CLIPBOARD|STDOUT`, then `SUCCESS|ERROR` response framing
- Test: `singleinstance_test.go` - server/client roundtrip
- Related: Pre-flight check in main.go

## Implementation Notes (2026-02-14)

- Resident hotkey requests and delegated `--run-once` requests now share a unified eventloop request pipeline.
- Standalone `--run-once` fallback reuses shared OCR session execution helpers.
- The architectural decision in this ADR is unchanged: TCP loopback delegation remains authoritative for resident coordination.
