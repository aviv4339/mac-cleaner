package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aviv4339/mac-cleaner/internal/scanner"
	"github.com/aviv4339/mac-cleaner/internal/ui"
	"github.com/spf13/cobra"
)

var (
	scanAll        bool
	scanCategories []string
	scanMinSize    string
	scanJSON       bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for reclaimable disk space",
	Long: `Scan your system for files, caches, and build artifacts that can be 
safely removed to free up disk space.

Results are sorted by size (largest first) and color-coded by risk level:
  [safe]     - Can be deleted without side effects
  [moderate] - May require re-downloading packages
  [caution]  - Review before deleting`,
	RunE: runScan,
}

func init() {
	scanCmd.Flags().BoolVar(&scanAll, "all", false, "Scan all categories (default behavior)")
	scanCmd.Flags().StringSliceVarP(&scanCategories, "category", "c", nil, "Scan specific categories (comma-separated)")
	scanCmd.Flags().StringVar(&scanMinSize, "min-size", "", "Minimum size to display (e.g., 100MB, 1GB)")
	scanCmd.Flags().BoolVar(&scanJSON, "json", false, "Output results as JSON")

	rootCmd.AddCommand(scanCmd)

	// Make scan the default command when no subcommand is given.
	rootCmd.RunE = runScan
	// Copy flags to root so they work with default command.
	rootCmd.Flags().BoolVar(&scanAll, "all", false, "Scan all categories (default behavior)")
	rootCmd.Flags().StringSliceVarP(&scanCategories, "category", "c", nil, "Scan specific categories (comma-separated)")
	rootCmd.Flags().StringVar(&scanMinSize, "min-size", "", "Minimum size to display (e.g., 100MB, 1GB)")
	rootCmd.Flags().BoolVar(&scanJSON, "json", false, "Output results as JSON")
}

func runScan(_ *cobra.Command, _ []string) error {
	out := ui.New(noColor)

	// Parse min-size.
	var minSize int64
	if scanMinSize != "" {
		var err error
		minSize, err = parseSize(scanMinSize)
		if err != nil {
			return fmt.Errorf("invalid --min-size value %q: %w", scanMinSize, err)
		}
	}

	// Validate categories.
	if len(scanCategories) > 0 {
		for _, c := range scanCategories {
			if _, err := scanner.FindCategory(c); err != nil {
				availableNames := scanner.CategoryNames()
				return fmt.Errorf("unknown category %q\nAvailable categories: %s", c, strings.Join(availableNames, ", "))
			}
		}
	}

	// Set up context with signal handling.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if !scanJSON {
		out.PrintBanner()
		out.PrintInfo("Scanning for reclaimable disk space...")
	}

	s, err := scanner.New("")
	if err != nil {
		return fmt.Errorf("initializing scanner: %w", err)
	}

	results, err := s.Scan(ctx, scanner.Options{
		Categories: scanCategories,
		MinSize:    minSize,
	})
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	if scanJSON {
		return out.PrintJSON(results)
	}

	out.PrintScanResults(results)
	return nil
}

// parseSize parses a human-readable size string like "100MB", "1.5GB".
func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))

	multipliers := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			numStr = strings.TrimSpace(numStr)

			var val float64
			if _, err := fmt.Sscanf(numStr, "%f", &val); err != nil {
				return 0, fmt.Errorf("parsing number %q: %w", numStr, err)
			}
			return int64(val * float64(mult)), nil
		}
	}

	// Try as plain number (bytes).
	var val int64
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return 0, fmt.Errorf("cannot parse size %q (use format like 100MB, 1GB)", s)
	}
	return val, nil
}
