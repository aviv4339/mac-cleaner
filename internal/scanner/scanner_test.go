package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("with explicit home dir", func(t *testing.T) {
		s, err := New("/tmp/test-home")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.homeDir != "/tmp/test-home" {
			t.Errorf("expected homeDir /tmp/test-home, got %s", s.homeDir)
		}
	})

	t.Run("with default home dir", func(t *testing.T) {
		s, err := New("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.homeDir == "" {
			t.Error("expected homeDir to be set")
		}
	})
}

func TestScan(t *testing.T) {
	// Create a temporary directory structure to scan.
	tmpDir := t.TempDir()

	// Create a fake "Library/Caches/Homebrew" structure.
	brewCache := filepath.Join(tmpDir, "Library", "Caches", "Homebrew")
	if err := os.MkdirAll(brewCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brewCache, "test-package.tar.gz"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := New(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	results, err := s.Scan(context.Background(), Options{
		Categories: []string{"homebrew"},
	})
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.CategoryName != "homebrew" {
		t.Errorf("expected category 'homebrew', got %q", r.CategoryName)
	}
	if r.TotalSize != 1024 {
		t.Errorf("expected total size 1024, got %d", r.TotalSize)
	}
	if len(r.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(r.Entries))
	}
}

func TestScanMinSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create small cache that should be filtered out.
	brewCache := filepath.Join(tmpDir, "Library", "Caches", "Homebrew")
	if err := os.MkdirAll(brewCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brewCache, "small.tar.gz"), make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := New(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	results, err := s.Scan(context.Background(), Options{
		Categories: []string{"homebrew"},
		MinSize:    1024, // Larger than our test file.
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results (filtered by min-size), got %d", len(results))
	}
}

func TestScanInvalidCategory(t *testing.T) {
	s, err := New("/tmp")
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Scan(context.Background(), Options{
		Categories: []string{"nonexistent"},
	})
	if err == nil {
		t.Error("expected error for invalid category")
	}
}

func TestScanContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	s, err := New("/tmp")
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Scan(ctx, Options{})
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestScanMissingDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	s, err := New(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Homebrew cache doesn't exist — should not error, just return empty.
	results, err := s.Scan(context.Background(), Options{
		Categories: []string{"homebrew"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].TotalSize != 0 {
		t.Errorf("expected 0 total size for missing dir, got %d", results[0].TotalSize)
	}
}

func TestScanMultipleCategories(t *testing.T) {
	tmpDir := t.TempDir()

	// Set up Homebrew cache.
	brewCache := filepath.Join(tmpDir, "Library", "Caches", "Homebrew")
	if err := os.MkdirAll(brewCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brewCache, "brew.tar.gz"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set up Go module cache.
	goCache := filepath.Join(tmpDir, "go", "pkg", "mod", "cache")
	if err := os.MkdirAll(goCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goCache, "mod.zip"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := New(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	results, err := s.Scan(context.Background(), Options{
		Categories: []string{"homebrew", "go"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Should be sorted by size, homebrew first (2048 > 1024).
	if results[0].CategoryName != "homebrew" {
		t.Errorf("expected homebrew first (larger), got %s", results[0].CategoryName)
	}
}

func TestDirSizeAndModTime(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure.
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file2.txt"), make([]byte, 200), 0o644); err != nil {
		t.Fatal(err)
	}

	size, modTime := dirSizeAndModTime(tmpDir)

	if size != 300 {
		t.Errorf("expected size 300, got %d", size)
	}
	if modTime.IsZero() {
		t.Error("expected non-zero mod time")
	}
}

func TestDirSizeAndModTimeSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(f, make([]byte, 500), 0o644); err != nil {
		t.Fatal(err)
	}

	size, modTime := dirSizeAndModTime(f)
	if size != 500 {
		t.Errorf("expected size 500, got %d", size)
	}
	if modTime.IsZero() {
		t.Error("expected non-zero mod time")
	}
}

func TestDirSizeAndModTimeNonexistent(t *testing.T) {
	size, modTime := dirSizeAndModTime("/nonexistent/path")
	if size != 0 {
		t.Errorf("expected size 0, got %d", size)
	}
	if !modTime.IsZero() {
		t.Error("expected zero mod time")
	}
}

func TestScanPath(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "b.txt"), make([]byte, 200), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Check that we have both a file and a directory.
	var hasFile, hasDir bool
	for _, e := range entries {
		if e.IsDir {
			hasDir = true
			if e.Size != 200 {
				t.Errorf("expected dir size 200, got %d", e.Size)
			}
		} else {
			hasFile = true
			if e.Size != 100 {
				t.Errorf("expected file size 100, got %d", e.Size)
			}
		}
	}
	if !hasFile || !hasDir {
		t.Error("expected both file and directory entries")
	}
}

func TestScanPathNonexistent(t *testing.T) {
	_, err := scanPath("/nonexistent/path/12345")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestScanOldDownloads(t *testing.T) {
	tmpDir := t.TempDir()
	downloadsDir := filepath.Join(tmpDir, "Downloads")
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create an "old" file.
	oldFile := filepath.Join(downloadsDir, "old-file.zip")
	if err := os.WriteFile(oldFile, make([]byte, 500), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().AddDate(0, 0, -60)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create a recent file.
	newFile := filepath.Join(downloadsDir, "new-file.zip")
	if err := os.WriteFile(newFile, make([]byte, 300), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanOldDownloads(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 old entry, got %d", len(entries))
	}
	if entries[0].Size != 500 {
		t.Errorf("expected size 500, got %d", entries[0].Size)
	}
}

func TestScanOldDownloadsMissing(t *testing.T) {
	entries, err := scanOldDownloads("/nonexistent/home")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for missing downloads, got %d", len(entries))
	}
}

func TestParseDockerSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"0B", 0},
		{"", 0},
		{"100MB", 100 * 1024 * 1024},
		{"2.5GB", int64(2.5 * 1024 * 1024 * 1024)},
		{"1TB", 1024 * 1024 * 1024 * 1024},
		{"512KB", 512 * 1024},
		{"1.2GB (500MB reclaimable)", 1288490188},
		{"garbage", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDockerSize(tt.input)
			if result != tt.expected {
				t.Errorf("parseDockerSize(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}
