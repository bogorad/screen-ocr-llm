package hotkey

import (
	"testing"
)

func TestKeyNameToRawcodes(t *testing.T) {
	tests := []struct {
		keyName  string
		expected []uint16
	}{
		// Modifier keys
		{"ctrl", []uint16{162, 163}},
		{"alt", []uint16{164, 165}},
		{"shift", []uint16{160, 161}},
		{"win", []uint16{91, 92}},
		{"cmd", []uint16{91, 92}},
		{"super", []uint16{91, 92}},

		// Letter keys
		{"q", []uint16{81}},
		{"e", []uint16{69}},
		{"o", []uint16{79}},
		{"t", []uint16{84}},

		// Number keys
		{"0", []uint16{48}},
		{"1", []uint16{49}},
		{"9", []uint16{57}},

		// Function keys
		{"f1", []uint16{112}},
		{"f12", []uint16{123}},
		{"f13", []uint16{124}},
		{"f24", []uint16{135}},

		// Special keys
		{"space", []uint16{32}},
		{"enter", []uint16{13}},
		{"esc", []uint16{27}},

		// Unknown key
		{"unknown", nil},
	}

	for _, tt := range tests {
		t.Run(tt.keyName, func(t *testing.T) {
			result := keyNameToRawcodes(tt.keyName)
			if len(result) != len(tt.expected) {
				t.Errorf("keyNameToRawcodes(%q) returned %d rawcodes, expected %d",
					tt.keyName, len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("keyNameToRawcodes(%q)[%d] = %d, expected %d",
						tt.keyName, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestParseHotkey(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Ctrl+Alt+Q", []string{"ctrl", "alt", "q"}},
		{"Ctrl+Shift+O", []string{"ctrl", "shift", "o"}},
		{"Ctrl+alt+e", []string{"ctrl", "alt", "e"}},
		{"Alt+F4", []string{"alt", "f4"}},
		{"Ctrl+Shift+F13", []string{"ctrl", "shift", "f13"}},
		{"Alt+F24", []string{"alt", "f24"}},
		{"Ctrl+Shift+T", []string{"ctrl", "shift", "t"}},
		{"Ctrl+Win+E", []string{"ctrl", "cmd", "e"}},
		{"Win+Shift+S", []string{"cmd", "shift", "s"}},
		{"Super+Alt+T", []string{"cmd", "alt", "t"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseHotkey(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseHotkey(%q) returned %d keys, expected %d",
					tt.input, len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("parseHotkey(%q)[%d] = %q, expected %q",
						tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}
