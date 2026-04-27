package clipboard

import (
	"strings"
	"testing"
)

func TestWriteSanitizesUnprintableCharacters(t *testing.T) {
	originalWriteText := writeText
	defer func() { writeText = originalWriteText }()

	var got string
	writeText = func(text string) <-chan struct{} {
		got = text
		return make(chan struct{})
	}

	err := Write("keep:/?*\"<>| text\nnext\tcol\rline\x00\x01\x7f\u0085done")
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	want := "keep:/?*\"<>| text\nnext\tcol\rlinedone"
	if got != want {
		t.Fatalf("Expected sanitized text %q, got %q", want, got)
	}
}

func TestWriteReturnsErrorWhenClipboardWriteFails(t *testing.T) {
	originalWriteText := writeText
	defer func() { writeText = originalWriteText }()

	writeText = func(text string) <-chan struct{} {
		return nil
	}

	err := Write("test text")
	if err == nil {
		t.Fatal("Expected error when clipboard write fails")
	}
	if !strings.Contains(err.Error(), "clipboard write failed") {
		t.Fatalf("Expected clipboard failure error, got %v", err)
	}
}

func TestSanitizeTextPreservesPrintableUnicode(t *testing.T) {
	input := "Invoice №42: café/東京?*<>|"
	if got := sanitizeText(input); got != input {
		t.Fatalf("Expected printable text to be preserved, got %q", got)
	}
}
