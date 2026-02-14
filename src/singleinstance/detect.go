package singleinstance

import (
	"bufio"
	"context"
	"net"
	"strconv"
	"time"
)

// DetectResidentPort scans the port range and returns (port, true) if a resident responds to PING.
func DetectResidentPort(ctx context.Context) (int, bool) {
	deadline := 300 * time.Millisecond
	if dl, ok := ctx.Deadline(); ok {
		if d := time.Until(dl); d > 0 {
			deadline = d
		}
	}
	start, end := getPortRange()
	for port := start; port <= end; port++ {
		addr := net.JoinHostPort(residentHost, strconv.Itoa(port))
		if ping(addr, deadline) {
			return port, true
		}
	}
	return 0, false
}

func ping(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	w := bufio.NewWriter(conn)
	if _, err := w.WriteString(pingRequest); err != nil {
		return false
	}
	if err := w.Flush(); err != nil {
		return false
	}
	br := bufio.NewReader(conn)
	resp, err := br.ReadString('\n')
	return err == nil && resp == pongResponse
}
