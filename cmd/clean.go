package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aviv4339/mac-cleaner/internal/cleaner"
	"github.com/aviv4339/mac-cleaner/internal/scanner"
	"github.com/aviv4339/mac-cleaner/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cleanDryRun     bool
	cleanForce      bool
	cleanCategories []string
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove reclaimable files to free disk space",
	Long: `Clean removes files, caches, and build artifacts to free up disk space.

By default, it runs interactively — showing what will be deleted and asking 
for confirmation before proceeding.

Use --dry-run to preview what would be deleted without removing anything.
Use --force to skip confirmation (use with caution!).`,
	RunE: runClean,
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Preview what would be deleted without deleting")
	cleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "Skip confirmation prompts")
	cleanCmd.Flags().StringSliceVarP(&cleanCategories, "category", "c", nil, "Clean specific categories (comma-separated)")

	rootCmd.AddCommand(cleanCmd)
}

func runClean(_ *cobra.Command, _ []string) error {
	out := ui.New(noColor)

	// Validate categories.
	if len(cleanCategories) > 0 {
		for _, c := range cleanCategories {
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
		out.PrintWarning("Interrupted! Stopping...")
		cancel()
	}()

	out.PrintInfo("Scanning for reclaimable disk space...")

	s, err := scanner.New("")
	if err != nil {
		return fmt.Errorf("initializing scanner: %w", err)
	}

	results, err := s.Scan(ctx, scanner.Options{
		Categories: cleanCategories,
	})
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	// Filter to only results with entries.
	var nonEmpty []scanner.ScanResult
	for _, r := range results {
		if len(r.Entries) > 0 {
			nonEmpty = append(nonEmpty, r)
		}
	}

	if len(nonEmpty) == 0 {
		out.PrintInfo("Nothing to clean! Your system is already tidy.")
		return nil
	}

	// Show what we found.
	out.PrintScanResults(nonEmpty)

	// Calculate total.
	var totalSize int64
	var totalItems int
	for _, r := range nonEmpty {
		totalSize += r.TotalSize
		totalItems += len(r.Entries)
	}

	if cleanDryRun {
		out.PrintCleanSummary(totalSize, totalItems, 0, 0, true)
		return nil
	}

	// Confirm unless --force.
	if !cleanForce {
		prompt := fmt.Sprintf("Delete %d items (%s)?", totalItems, ui.FormatSize(totalSize))
		if !out.ConfirmAction(prompt) {
			out.PrintInfo("Cleanup cancelled.")
			return nil
		}
	}

	// Perform cleanup.
	c := cleaner.New(false)
	summary := c.Clean(nonEmpty)

	out.PrintCleanSummary(summary.TotalFreed, summary.ItemsDeleted, summary.ItemsSkipped, summary.ItemsFailed, false)

	// Report any errors.
	for _, r := range summary.Results {
		if r.Error != nil {
			out.PrintWarning(fmt.Sprintf("Failed to delete %s: %s", r.Path, r.Error))
		}
	}

	return nil
}
