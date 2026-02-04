package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintBanner(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriter(&buf, true) // No color for easy assertions.

	o.PrintBanner()

	output := buf.String()

	// Verify key elements of the banner are present.
	if !strings.Contains(output, "CORNHOLIO") {
		t.Error("expected 'CORNHOLIO' in banner output")
	}
	if !strings.Contains(output, "BUNGHOLE") {
		t.Error("expected 'BUNGHOLE' in banner output")
	}
	if !strings.Contains(output, "m a c - c l e a n e r") {
		t.Error("expected 'mac-cleaner' title in banner output")
	}
	if !strings.Contains(output, "disk space") {
		t.Error("expected 'disk space' tagline in banner output")
	}
}

func TestPrintBannerWithColor(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriter(&buf, false) // With color.

	o.PrintBanner()

	output := buf.String()

	// Should contain ANSI codes.
	if !strings.Contains(output, "\033[") {
		t.Error("expected ANSI color codes in colored banner")
	}
	// Should still contain the text.
	if !strings.Contains(output, "CORNHOLIO") {
		t.Error("expected 'CORNHOLIO' in colored banner")
	}
}

func TestBannerConstant(t *testing.T) {
	// Verify the ASCII art has reasonable dimensions.
	lines := splitLines(banner)
	if len(lines) < 20 {
		t.Errorf("banner seems too short: %d lines", len(lines))
	}

	// Check that it has the character structure (arms raised).
	hasArms := false
	for _, line := range lines {
		if strings.Contains(line, "/") && strings.Contains(line, "\\") {
			hasArms = true
			break
		}
	}
	if !hasArms {
		t.Error("banner ASCII art doesn't appear to have the expected structure")
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty", "", 0},
		{"single line", "hello", 1},
		{"two lines", "hello\nworld", 2},
		{"trailing newline", "hello\n", 1},
		{"multiple lines", "a\nb\nc\nd", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			if len(result) != tt.expected {
				t.Errorf("splitLines(%q) returned %d lines, want %d", tt.input, len(result), tt.expected)
			}
		})
	}
}
