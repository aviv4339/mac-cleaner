// Package ui provides terminal output formatting for mac-cleaner.
// It handles colored output, human-readable sizes, progress display,
// and structured output formats like JSON.
package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aviv4339/mac-cleaner/internal/scanner"
)

// ANSI color codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// Output handles terminal formatting and display.
type Output struct {
	w       io.Writer
	noColor bool
}

// New creates a new Output writer.
func New(noColor bool) *Output {
	return &Output{
		w:       os.Stdout,
		noColor: noColor,
	}
}

// NewWithWriter creates a new Output with a custom writer (useful for testing).
func NewWithWriter(w io.Writer, noColor bool) *Output {
	return &Output{
		w:       w,
		noColor: noColor,
	}
}

// FormatSize formats a byte count as a human-readable string.
func FormatSize(bytes int64) string {
	if bytes < 0 {
		return "0 B"
	}

	const (
		_          = iota
		kb float64 = 1 << (10 * iota)
		mb
		gb
		tb
	)

	switch {
	case float64(bytes) >= tb:
		return fmt.Sprintf("%.1f TB", float64(bytes)/tb)
	case float64(bytes) >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/gb)
	case float64(bytes) >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/mb)
	case float64(bytes) >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/kb)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// PrintScanResults displays scan results in a formatted table.
func (o *Output) PrintScanResults(results []scanner.ScanResult) {
	if len(results) == 0 {
		o.println(o.color(colorGreen, "✓ No cleanable items found."))
		return
	}

	var totalSize int64
	for _, r := range results {
		totalSize += r.TotalSize
	}

	o.println("")
	o.println(o.color(colorBold, "📊 Disk Usage Scan Results"))
	o.println(o.color(colorDim, strings.Repeat("─", 60)))
	o.println("")

	for _, r := range results {
		if r.TotalSize == 0 && r.Error == "" {
			continue
		}

		riskColor := o.riskColor(r.Risk)
		riskBadge := o.riskBadge(r.Risk)

		o.printf("  %s %s %s\n",
			o.color(colorBold, fmt.Sprintf("%-25s", r.DisplayName)),
			o.color(riskColor, fmt.Sprintf("%10s", FormatSize(r.TotalSize))),
			riskBadge,
		)

		if r.Error != "" {
			o.printf("    %s\n", o.color(colorRed, fmt.Sprintf("⚠ %s", r.Error)))
		}

		// Show top entries (up to 5).
		maxEntries := 5
		if len(r.Entries) < maxEntries {
			maxEntries = len(r.Entries)
		}
		for i := 0; i < maxEntries; i++ {
			e := r.Entries[i]
			o.printf("    %s %s\n",
				o.color(colorDim, "├─"),
				fmt.Sprintf("%-40s %s", truncate(e.Description, 40), o.color(colorDim, FormatSize(e.Size))),
			)
		}
		if len(r.Entries) > 5 {
			o.printf("    %s\n", o.color(colorDim, fmt.Sprintf("└─ ... and %d more items", len(r.Entries)-5)))
		}
		o.println("")
	}

	o.println(o.color(colorDim, strings.Repeat("─", 60)))
	o.printf("  %s %s\n\n",
		o.color(colorBold, fmt.Sprintf("%-25s", "Total reclaimable:")),
		o.color(colorBold+colorCyan, FormatSize(totalSize)),
	)
}

// PrintJSON outputs scan results as JSON.
func (o *Output) PrintJSON(results []scanner.ScanResult) error {
	enc := json.NewEncoder(o.w)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

// PrintCleanSummary displays the summary after a clean operation.
func (o *Output) PrintCleanSummary(freed int64, deleted, skipped, failed int, dryRun bool) {
	o.println("")
	if dryRun {
		o.println(o.color(colorYellow, "🔍 Dry-run complete (no files were deleted)"))
	} else {
		o.println(o.color(colorGreen, "✓ Cleanup complete!"))
	}
	o.println(o.color(colorDim, strings.Repeat("─", 40)))

	if dryRun {
		o.printf("  Would free:    %s\n", o.color(colorCyan, FormatSize(freed)))
		o.printf("  Items found:   %d\n", deleted+skipped+failed)
	} else {
		o.printf("  Space freed:   %s\n", o.color(colorGreen, FormatSize(freed)))
		o.printf("  Items deleted: %s\n", o.color(colorGreen, fmt.Sprintf("%d", deleted)))
		if skipped > 0 {
			o.printf("  Items skipped: %s\n", o.color(colorYellow, fmt.Sprintf("%d", skipped)))
		}
		if failed > 0 {
			o.printf("  Items failed:  %s\n", o.color(colorRed, fmt.Sprintf("%d", failed)))
		}
	}
	o.println("")
}

// PrintError displays an error message.
func (o *Output) PrintError(msg string) {
	o.printf("%s %s\n", o.color(colorRed, "✗"), msg)
}

// PrintWarning displays a warning message.
func (o *Output) PrintWarning(msg string) {
	o.printf("%s %s\n", o.color(colorYellow, "⚠"), msg)
}

// PrintInfo displays an informational message.
func (o *Output) PrintInfo(msg string) {
	o.printf("%s %s\n", o.color(colorBlue, "ℹ"), msg)
}

// ConfirmAction asks the user for confirmation and returns their response.
func (o *Output) ConfirmAction(prompt string) bool {
	o.printf("%s %s [y/N]: ", o.color(colorYellow, "?"), prompt)

	var response string
	_, _ = fmt.Fscanln(os.Stdin, &response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// riskColor returns the ANSI color code for a risk level.
func (o *Output) riskColor(risk scanner.RiskLevel) string {
	switch risk {
	case scanner.RiskSafe:
		return colorGreen
	case scanner.RiskModerate:
		return colorYellow
	case scanner.RiskCaution:
		return colorRed
	default:
		return colorReset
	}
}

// riskBadge returns a colored risk badge string.
func (o *Output) riskBadge(risk scanner.RiskLevel) string {
	switch risk {
	case scanner.RiskSafe:
		return o.color(colorGreen, "[safe]")
	case scanner.RiskModerate:
		return o.color(colorYellow, "[moderate]")
	case scanner.RiskCaution:
		return o.color(colorRed, "[caution]")
	default:
		return ""
	}
}

// color wraps text with ANSI color codes if colors are enabled.
func (o *Output) color(code, text string) string {
	if o.noColor {
		return text
	}
	return fmt.Sprintf("%s%s%s", code, text, colorReset)
}

// println writes a line to the output.
func (o *Output) println(s string) {
	fmt.Fprintln(o.w, s)
}

// printf writes formatted text to the output.
func (o *Output) printf(format string, args ...interface{}) {
	fmt.Fprintf(o.w, format, args...)
}

// truncate shortens a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
