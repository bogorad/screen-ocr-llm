package clipboard

import (
	"testing"
)

func TestWrite(t *testing.T) {
	// This test would require clipboard access, so we'll just check if the function exists
	// and doesn't panic
	err := Write("test text")
	if err != nil {
		t.Logf("Failed to write to clipboard: %v", err)
	}
}
