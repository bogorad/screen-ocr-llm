package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestResidentPortConflictReturnsDiagnostic(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	t.Setenv("SINGLEINSTANCE_PORT_START", port)
	t.Setenv("SINGLEINSTANCE_PORT_END", port)
	t.Setenv("ENABLE_FILE_LOGGING", "false")
	oldWriter, oldFlags := log.Writer(), log.Flags()
	t.Cleanup(func() { log.SetOutput(oldWriter); log.SetFlags(oldFlags) })
	err = runApplication(mainOptions{})
	if err == nil || !strings.Contains(err.Error(), "127.0.0.1:"+port) {
		t.Fatalf("expected port conflict diagnostic, got %v", err)
	}
	var networkError *net.OpError
	if !errors.As(err, &networkError) {
		t.Fatalf("diagnostic lost the original bind error: %v", err)
	}
}

func TestApplicationErrorVisibleWithLoggingDisabled(t *testing.T) {
	t.Setenv("ENABLE_FILE_LOGGING", "false")
	oldWriter := log.Writer()
	t.Cleanup(func() { log.SetOutput(oldWriter) })
	log.SetOutput(io.Discard)
	called := false
	reportApplicationError(errors.New("missing API key"), func(title, message string) {
		called = true
		for _, want := range []string{"missing API key", "Executable:", "Working directory:", "File logging is disabled", "Ctrl+C"} {
			if !strings.Contains(message, want) {
				t.Errorf("dialog missing %q: %s", want, message)
			}
		}
		if title != "Screen OCR LLM - Error" {
			t.Errorf("unexpected title: %q", title)
		}
	})
	if !called {
		t.Fatal("fatal error did not display a dialog")
	}
}

func TestApplicationErrorRecordsContext(t *testing.T) {
	t.Setenv("ENABLE_FILE_LOGGING", "true")
	var output bytes.Buffer
	oldWriter := log.Writer()
	t.Cleanup(func() { log.SetOutput(oldWriter) })
	log.SetOutput(&output)
	reportApplicationError(errors.New("port access denied"), func(_, message string) {
		if !strings.Contains(message, "screen_ocr_debug.log") {
			t.Errorf("dialog missing log location: %s", message)
		}
	})
	for _, want := range []string{"event=application_failed", "pid=", "executable=", "working_directory=", `error="port access denied"`} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("log missing %q: %s", want, output.String())
		}
	}
}
