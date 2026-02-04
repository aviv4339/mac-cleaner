// Package cleaner provides safe deletion logic for mac-cleaner.
// It handles file removal with confirmation, dry-run support, and
// proper error handling for permission issues.
package cleaner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aviv4339/mac-cleaner/internal/scanner"
)

// Options configures a clean operation.
type Options struct {
	// DryRun previews what would be deleted without actually deleting.
	DryRun bool
	// Force skips confirmation prompts.
	Force bool
	// Categories filters which categories to clean.
	Categories []string
}

// CleanResult holds the outcome of a clean operation for a single entry.
type CleanResult struct {
	// Path is the filesystem path that was (or would be) deleted.
	Path string
	// Size is the size in bytes that was (or would be) freed.
	Size int64
	// Deleted indicates whether the item was actually deleted.
	Deleted bool
	// Skipped indicates the item was skipped (dry-run or user declined).
	Skipped bool
	// Error contains any error that occurred during deletion.
	Error error
}

// Summary holds aggregate results of a clean operation.
type Summary struct {
	// TotalFreed is the total bytes actually freed.
	TotalFreed int64
	// TotalSkipped is the total bytes that were skipped.
	TotalSkipped int64
	// ItemsDeleted is the number of items deleted.
	ItemsDeleted int
	// ItemsSkipped is the number of items skipped.
	ItemsSkipped int
	// ItemsFailed is the number of items that failed to delete.
	ItemsFailed int
	// Results contains individual results per entry.
	Results []CleanResult
}

// Cleaner handles the deletion of scanned items.
type Cleaner struct {
	dryRun bool
}

// New creates a new Cleaner with the given options.
func New(dryRun bool) *Cleaner {
	return &Cleaner{dryRun: dryRun}
}

// Clean deletes (or previews deletion of) the given scan results.
// For Docker entries (paths starting with "docker:"), it runs docker system prune.
// For regular filesystem entries, it removes the files/directories.
func (c *Cleaner) Clean(results []scanner.ScanResult) Summary {
	var summary Summary

	for _, result := range results {
		for _, entry := range result.Entries {
			cr := c.cleanEntry(entry, result.CategoryName)
			summary.Results = append(summary.Results, cr)

			if cr.Deleted {
				summary.TotalFreed += cr.Size
				summary.ItemsDeleted++
			} else if cr.Skipped {
				summary.TotalSkipped += cr.Size
				summary.ItemsSkipped++
			} else if cr.Error != nil {
				summary.ItemsFailed++
			}
		}
	}

	return summary
}

// CleanEntries deletes (or previews deletion of) specific entries.
func (c *Cleaner) CleanEntries(entries []scanner.ScanEntry, categoryName string) Summary {
	var summary Summary

	for _, entry := range entries {
		cr := c.cleanEntry(entry, categoryName)
		summary.Results = append(summary.Results, cr)

		if cr.Deleted {
			summary.TotalFreed += cr.Size
			summary.ItemsDeleted++
		} else if cr.Skipped {
			summary.TotalSkipped += cr.Size
			summary.ItemsSkipped++
		} else if cr.Error != nil {
			summary.ItemsFailed++
		}
	}

	return summary
}

// cleanEntry handles deletion of a single entry.
func (c *Cleaner) cleanEntry(entry scanner.ScanEntry, categoryName string) CleanResult {
	cr := CleanResult{
		Path: entry.Path,
		Size: entry.Size,
	}

	if c.dryRun {
		cr.Skipped = true
		return cr
	}

	// Handle Docker entries specially.
	if strings.HasPrefix(entry.Path, "docker:") {
		err := cleanDocker()
		if err != nil {
			cr.Error = fmt.Errorf("cleaning docker: %w", err)
		} else {
			cr.Deleted = true
		}
		return cr
	}

	// Validate the path before deletion.
	if err := validatePath(entry.Path); err != nil {
		cr.Error = fmt.Errorf("unsafe path %q: %w", entry.Path, err)
		return cr
	}

	// Remove the file or directory.
	err := os.RemoveAll(entry.Path)
	if err != nil {
		cr.Error = fmt.Errorf("removing %s: %w", entry.Path, err)
		return cr
	}

	cr.Deleted = true
	return cr
}

// validatePath ensures a path is safe to delete.
// It prevents deletion of critical system directories.
func validatePath(path string) error {
	// Resolve to absolute path.
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Never allow deleting root or home directory itself.
	homeDir, _ := os.UserHomeDir()
	dangerous := []string{
		"/",
		"/System",
		"/Applications",
		"/Users",
		"/Library",
		"/bin",
		"/sbin",
		"/usr",
		"/var",
		"/etc",
		"/private",
		homeDir,
	}

	for _, d := range dangerous {
		if absPath == d {
			return fmt.Errorf("refusing to delete critical path: %s", absPath)
		}
	}

	// Must be under home directory, /Library/Caches, or /tmp (for testing).
	if !strings.HasPrefix(absPath, homeDir) &&
		!strings.HasPrefix(absPath, "/Library/Caches") &&
		!strings.HasPrefix(absPath, "/tmp") {
		return fmt.Errorf("path %s is outside allowed directories", absPath)
	}

	return nil
}

// cleanDocker runs docker system prune to clean up Docker resources.
func cleanDocker() error {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("docker not found: %w", err)
	}

	cmd := exec.Command(dockerPath, "system", "prune", "-af")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
