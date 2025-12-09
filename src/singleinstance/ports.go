package singleinstance

import (
	"os"
	"strconv"
)

const (
	defaultPortStart = 49500
	defaultPortEnd   = 49550
)

// getPortRange returns the configured TCP port range. Environment variables:
// SINGLEINSTANCE_PORT_START and SINGLEINSTANCE_PORT_END (integers, inclusive).
// Falls back to defaults when unset/invalid, and clamps to [1024, 65535].
func getPortRange() (int, int) {
	start := defaultPortStart
	end := defaultPortEnd
	if v := os.Getenv("SINGLEINSTANCE_PORT_START"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			start = n
		}
	}
	if v := os.Getenv("SINGLEINSTANCE_PORT_END"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			end = n
		}
	}
	if start < 1024 {
		start = 1024
	}
	if end > 65535 {
		end = 65535
	}
	if end < start {
		start, end = end, start
	}
	return start, end
}

// GetPortRangeForDebug exposes the current effective port range for logging/debugging.
func GetPortRangeForDebug() (int, int) { return getPortRange() }
