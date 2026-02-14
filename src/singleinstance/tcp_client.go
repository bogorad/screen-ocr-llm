package singleinstance

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"time"
)

type tcpClient struct{}

func newTcpClient() Client { return &tcpClient{} }

func (c *tcpClient) TryRunOnce(ctx context.Context, outputToStdout bool) (bool, string, error) {
	deadline := 2 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		if d := time.Until(dl); d > 0 {
			deadline = d
		}
	}
	// scan configured range for resident using PING then request
	start, end := getPortRange()
	for port := start; port <= end; port++ {
		addr := net.JoinHostPort(residentHost, strconv.Itoa(port))
		if !ping(addr, deadline) {
			continue
		}
		// connect for request
		conn, err := net.DialTimeout("tcp", addr, deadline)
		if err != nil {
			continue
		}
		w := bufio.NewWriter(conn)
		if outputToStdout {
			_, err = w.WriteString("STDOUT\n")
		} else {
			_, err = w.WriteString("CLIPBOARD\n")
		}
		if err != nil {
			conn.Close()
			return true, "", err
		}
		if err := w.Flush(); err != nil {
			conn.Close()
			return true, "", err
		}
		br := bufio.NewReader(conn)
		status, err := br.ReadString('\n')
		if err != nil {
			conn.Close()
			return true, "", err
		}
		if status == "SUCCESS\n" {
			b, _ := io.ReadAll(br)
			conn.Close()
			return true, string(b), nil
		}
		if status == "ERROR\n" {
			msg, _ := io.ReadAll(br)
			conn.Close()
			return true, "", errors.New(string(msg))
		}
		conn.Close()
	}
	return false, "", nil
}
