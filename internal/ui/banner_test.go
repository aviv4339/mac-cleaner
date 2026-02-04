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
	if !strings.Contains(output, "bunghole") {
		t.Error("expected 'bunghole' in banner output")
	}
	if !strings.Contains(output, "disk space") {
		t.Error("expected 'disk space' tagline in banner output")
	}
	if !strings.Contains(output, "██") {
		t.Error("expected block-letter art in banner output")
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
	if !strings.Contains(output, "bunghole") {
		t.Error("expected 'bunghole' in colored banner")
	}
}

func TestBannerConstant(t *testing.T) {
	lines := splitLines(Banner)
	if len(lines) < 10 {
		t.Errorf("banner seems too short: %d lines", len(lines))
	}

	// Check that it has the box-drawing frame.
	hasBox := false
	for _, line := range lines {
		if strings.Contains(line, "╔") || strings.Contains(line, "╚") {
			hasBox = true
			break
		}
	}
	if !hasBox {
		t.Error("banner doesn't contain expected box-drawing characters")
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
