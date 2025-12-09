package singleinstance

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

const (
	residentHost = "127.0.0.1"
	pingRequest  = "PING\n"
	pongResponse = "PONG\n"
)

// tcpServer implements Server over TCP loopback.
type tcpServer struct {
	lis      net.Listener
	incoming chan *tcpConn
	port     int
}

func newTcpServer() Server { return &tcpServer{incoming: make(chan *tcpConn, 8)} }

// Start binds ONLY the start port of the configured range. If occupied, fail.
func (s *tcpServer) Start(ctx context.Context) error {
	if s.lis != nil {
		return nil
	}
	start, _ := getPortRange()
	addr := fmt.Sprintf("%s:%d", residentHost, start)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("singleinstance: failed to bind %s: %v", addr, err)
		return err
	}
	s.lis = lis
	s.port = start
	log.Printf("singleinstance: listening on %s", addr)
	go s.acceptLoop(ctx)
	return nil
}

// Port returns the bound port (0 if not started).
func (s *tcpServer) Port() int { return s.port }

func (s *tcpServer) acceptLoop(ctx context.Context) {
	for {
		c, err := s.lis.Accept()
		if err != nil {
			return
		}
		remote := c.RemoteAddr().String()
		_ = c.SetDeadline(time.Now().Add(3 * time.Second))
		br := bufio.NewReader(c)
		line, _ := br.ReadString('\n')
		bw := bufio.NewWriter(c)
		if line == pingRequest {
			log.Printf("singleinstance: PING from %s -> PONG", remote)
			_, _ = bw.WriteString(pongResponse)
			_ = bw.Flush()
			_ = c.Close()
			continue
		}
		// Non-PING: treat first line as request (STDOUT/CLIPBOARD)
		_ = c.SetDeadline(time.Time{})
		stdout := line == "STDOUT\n"
		log.Printf("singleinstance: request from %s mode=%s", remote, map[bool]string{true: "STDOUT", false: "CLIPBOARD"}[stdout])
		req := Request{OutputToStdout: stdout}
		select {
		case s.incoming <- &tcpConn{c: c, r: req, w: bw, br: br}:
		case <-ctx.Done():
			_ = c.Close()
			return
		}
	}
}

func (s *tcpServer) Next(ctx context.Context) (Conn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case tc := <-s.incoming:
		return tc, nil
	}
}

func (s *tcpServer) Close() error {
	if s.lis != nil {
		_ = s.lis.Close()
		s.lis = nil
	}
	close(s.incoming)
	return nil
}

type tcpConn struct {
	c  net.Conn
	r  Request
	w  *bufio.Writer
	br *bufio.Reader
}

func (tc *tcpConn) Request() Request { return tc.r }

func (tc *tcpConn) RespondSuccess(text string) error {
	if _, err := tc.w.WriteString("SUCCESS\n"); err != nil {
		return err
	}
	if len(text) > 0 {
		if _, err := tc.w.WriteString(text); err != nil {
			return err
		}
	}
	return tc.w.Flush()
}

func (tc *tcpConn) RespondError(msg string) error {
	if _, err := tc.w.WriteString("ERROR\n" + msg); err != nil {
		return err
	}
	return tc.w.Flush()
}

func (tc *tcpConn) Close() error { return tc.c.Close() }
