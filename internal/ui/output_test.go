package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aviv4339/mac-cleaner/internal/scanner"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{-1, "0 B"},
		{1, "1 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
		{2684354560, "2.5 GB"},
		{524288, "512.0 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestOutputNoColor(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriter(&buf, true) // No color.

	text := o.color(colorRed, "test")
	if text != "test" {
		t.Errorf("expected plain text with noColor, got %q", text)
	}

	// Verify color is stripped.
	if strings.Contains(text, "\033") {
		t.Error("expected no ANSI codes with noColor")
	}
}

func TestOutputWithColor(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriter(&buf, false) // With color.

	text := o.color(colorRed, "test")
	if !strings.Contains(text, "\033[31m") {
		t.Errorf("expected ANSI red code, got %q", text)
	}
	if !strings.Contains(text, "test") {
		t.Error("expected text content")
	}
}

func TestPrintScanResultsEmpty(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriter(&buf, true)

	o.PrintScanResults(nil)

	output := buf.String()
	if !strings.Contains(output, "No cleanable items found") {
		t.Errorf("expected 'no items' message, got %q", output)
	}
}

func TestPrintScanResults(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriter(&buf, true)

	results := []scanner.ScanResult{
		{
			CategoryName: "homebrew",
			DisplayName:  "Homebrew",
			Risk:         scanner.RiskSafe,
			TotalSize:    1073741824, // 1 GB
			Entries: []scanner.ScanEntry{
				{Path: "/test/brew-cache", Size: 1073741824, Description: "brew-cache"},
			},
		},
		{
			CategoryName: "docker",
			DisplayName:  "Docker",
			Risk:         scanner.RiskModerate,
			TotalSize:    524288000, // ~500 MB
			Entries: []scanner.ScanEntry{
				{Path: "docker:Images", Size: 524288000, Description: "Docker Images"},
			},
		},
	}

	o.PrintScanResults(results)

	output := buf.String()
	if !strings.Contains(output, "Homebrew") {
		t.Error("expected Homebrew in output")
	}
	if !strings.Contains(output, "Docker") {
		t.Error("expected Docker in output")
	}
	if !strings.Contains(output, "1.0 GB") {
		t.Error("expected '1.0 GB' in output")
	}
	if !strings.Contains(output, "Total reclaimable") {
		t.Error("expected 'Total reclaimable' in output")
	}
}

func TestPrintScanResultsWithError(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriter(&buf, true)

	results := []scanner.ScanResult{
		{
			CategoryName: "docker",
			DisplayName:  "Docker",
			Risk:         scanner.RiskModerate,
			Error:        "docker not available",
		},
	}

	o.PrintScanResults(results)

	output := buf.String()
	if !strings.Contains(output, "docker not available") {
		t.Error("expected error message in output")
	}
}

func TestPrintScanResultsManyEntries(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriter(&buf, true)

	entries := make([]scanner.ScanEntry, 10)
	for i := range entries {
		entries[i] = scanner.ScanEntry{
			Path:        "/test/entry",
			Size:        1024,
			Description: "entry",
		}
	}

	results := []scanner.ScanResult{
		{
			CategoryName: "test",
			DisplayName:  "Test",
			Risk:         scanner.RiskSafe,
			TotalSize:    10240,
			Entries:      entries,
		},
	}

	o.PrintScanResults(results)

	output := buf.String()
	if !strings.Contains(output, "and 5 more items") {
		t.Error("expected 'and 5 more items' in output")
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriter(&buf, true)

	results := []scanner.ScanResult{
		{
			CategoryName: "homebrew",
			DisplayName:  "Homebrew",
			TotalSize:    1024,
			Entries: []scanner.ScanEntry{
				{Path: "/test", Size: 1024, Description: "test"},
			},
		},
	}

	if err := o.PrintJSON(results); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, `"category_name": "homebrew"`) {
		t.Error("expected JSON category_name")
	}
	if !strings.Contains(output, `"total_size": 1024`) {
		t.Error("expected JSON total_size")
	}
}

func TestPrintCleanSummary(t *testing.T) {
	t.Run("actual clean", func(t *testing.T) {
		var buf bytes.Buffer
		o := NewWithWriter(&buf, true)

		o.PrintCleanSummary(1073741824, 10, 2, 1, false)

		output := buf.String()
		if !strings.Contains(output, "Cleanup complete") {
			t.Error("expected 'Cleanup complete'")
		}
		if !strings.Contains(output, "1.0 GB") {
			t.Error("expected freed size")
		}
		if !strings.Contains(output, "10") {
			t.Error("expected items deleted count")
		}
	})

	t.Run("dry run", func(t *testing.T) {
		var buf bytes.Buffer
		o := NewWithWriter(&buf, true)

		o.PrintCleanSummary(1073741824, 10, 0, 0, true)

		output := buf.String()
		if !strings.Contains(output, "Dry-run") {
			t.Error("expected 'Dry-run' in output")
		}
		if !strings.Contains(output, "Would free") {
			t.Error("expected 'Would free' in output")
		}
	})
}

func TestPrintMessages(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		var buf bytes.Buffer
		o := NewWithWriter(&buf, true)
		o.PrintError("something failed")
		if !strings.Contains(buf.String(), "something failed") {
			t.Error("expected error message")
		}
	})

	t.Run("warning", func(t *testing.T) {
		var buf bytes.Buffer
		o := NewWithWriter(&buf, true)
		o.PrintWarning("be careful")
		if !strings.Contains(buf.String(), "be careful") {
			t.Error("expected warning message")
		}
	})

	t.Run("info", func(t *testing.T) {
		var buf bytes.Buffer
		o := NewWithWriter(&buf, true)
		o.PrintInfo("just so you know")
		if !strings.Contains(buf.String(), "just so you know") {
			t.Error("expected info message")
		}
	})
}

func TestRiskBadge(t *testing.T) {
	o := NewWithWriter(&bytes.Buffer{}, true)

	tests := []struct {
		risk     scanner.RiskLevel
		expected string
	}{
		{scanner.RiskSafe, "[safe]"},
		{scanner.RiskModerate, "[moderate]"},
		{scanner.RiskCaution, "[caution]"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			badge := o.riskBadge(tt.risk)
			if badge != tt.expected {
				t.Errorf("riskBadge(%d) = %q, want %q", tt.risk, badge, tt.expected)
			}
		})
	}
}

func TestRiskColor(t *testing.T) {
	o := NewWithWriter(&bytes.Buffer{}, false) // Colors enabled.

	if c := o.riskColor(scanner.RiskSafe); c != colorGreen {
		t.Errorf("expected green for safe, got %q", c)
	}
	if c := o.riskColor(scanner.RiskModerate); c != colorYellow {
		t.Errorf("expected yellow for moderate, got %q", c)
	}
	if c := o.riskColor(scanner.RiskCaution); c != colorRed {
		t.Errorf("expected red for caution, got %q", c)
	}
}
