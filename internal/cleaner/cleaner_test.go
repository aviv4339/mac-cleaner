package cleaner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aviv4339/mac-cleaner/internal/scanner"
)

func TestNewCleaner(t *testing.T) {
	c := New(true)
	if c == nil {
		t.Fatal("expected non-nil cleaner")
	}
	if !c.dryRun {
		t.Error("expected dryRun to be true")
	}
}

func TestCleanDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(true) // Dry run mode.

	results := []scanner.ScanResult{
		{
			CategoryName: "test",
			Entries: []scanner.ScanEntry{
				{
					Path:  testFile,
					Size:  5,
					IsDir: false,
				},
			},
		},
	}

	summary := c.Clean(results)

	// File should still exist.
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("file was deleted in dry-run mode")
	}

	if summary.ItemsDeleted != 0 {
		t.Errorf("expected 0 items deleted in dry-run, got %d", summary.ItemsDeleted)
	}
	if summary.ItemsSkipped != 1 {
		t.Errorf("expected 1 item skipped in dry-run, got %d", summary.ItemsSkipped)
	}
	if summary.TotalFreed != 0 {
		t.Errorf("expected 0 bytes freed in dry-run, got %d", summary.TotalFreed)
	}
	if summary.TotalSkipped != 5 {
		t.Errorf("expected 5 bytes skipped in dry-run, got %d", summary.TotalSkipped)
	}
}

func TestCleanActualDeletion(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file to delete.
	testFile := filepath.Join(tmpDir, "deleteme.txt")
	if err := os.WriteFile(testFile, make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a directory to delete.
	testDir := filepath.Join(tmpDir, "deleteme-dir")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "inner.txt"), make([]byte, 512), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(false)

	results := []scanner.ScanResult{
		{
			CategoryName: "test",
			Entries: []scanner.ScanEntry{
				{Path: testFile, Size: 1024, IsDir: false},
				{Path: testDir, Size: 512, IsDir: true},
			},
		},
	}

	summary := c.Clean(results)

	// Files should be gone.
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("file was not deleted")
	}
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Error("directory was not deleted")
	}

	if summary.ItemsDeleted != 2 {
		t.Errorf("expected 2 items deleted, got %d", summary.ItemsDeleted)
	}
	if summary.TotalFreed != 1536 {
		t.Errorf("expected 1536 bytes freed, got %d", summary.TotalFreed)
	}
}

func TestCleanEntries(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "entry.txt")
	if err := os.WriteFile(testFile, make([]byte, 256), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(false)
	entries := []scanner.ScanEntry{
		{Path: testFile, Size: 256, IsDir: false},
	}

	summary := c.CleanEntries(entries, "test")

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("file was not deleted")
	}
	if summary.ItemsDeleted != 1 {
		t.Errorf("expected 1 item deleted, got %d", summary.ItemsDeleted)
	}
}

func TestCleanNonexistentPath(t *testing.T) {
	c := New(false)

	// This path doesn't exist — RemoveAll doesn't error for nonexistent paths.
	entries := []scanner.ScanEntry{
		{Path: "/tmp/mac-cleaner-nonexistent-12345", Size: 100, IsDir: false},
	}

	summary := c.CleanEntries(entries, "test")

	// RemoveAll doesn't error for nonexistent paths, so it counts as deleted.
	if summary.ItemsFailed != 0 {
		t.Errorf("expected 0 failures, got %d", summary.ItemsFailed)
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"root", "/", true},
		{"system", "/System", true},
		{"applications", "/Applications", true},
		{"users", "/Users", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath(%q): error = %v, wantErr = %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePathAllowsCaches(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	// Path under home should be allowed.
	path := filepath.Join(homeDir, "Library", "Caches", "test-app")
	if err := validatePath(path); err != nil {
		t.Errorf("expected path under home to be allowed: %v", err)
	}
}

func TestSummary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two files.
	f1 := filepath.Join(tmpDir, "f1.txt")
	f2 := filepath.Join(tmpDir, "f2.txt")
	if err := os.WriteFile(f1, make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, make([]byte, 200), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(false)

	results := []scanner.ScanResult{
		{
			CategoryName: "cat1",
			Entries: []scanner.ScanEntry{
				{Path: f1, Size: 100, IsDir: false, LastModified: time.Now()},
			},
		},
		{
			CategoryName: "cat2",
			Entries: []scanner.ScanEntry{
				{Path: f2, Size: 200, IsDir: false, LastModified: time.Now()},
			},
		},
	}

	summary := c.Clean(results)

	if summary.ItemsDeleted != 2 {
		t.Errorf("expected 2 deleted, got %d", summary.ItemsDeleted)
	}
	if summary.TotalFreed != 300 {
		t.Errorf("expected 300 bytes freed, got %d", summary.TotalFreed)
	}
	if len(summary.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(summary.Results))
	}
}
