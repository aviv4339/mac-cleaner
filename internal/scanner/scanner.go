package scanner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Options configures a scan operation.
type Options struct {
	// Categories filters the scan to specific category names.
	// If empty, all categories are scanned.
	Categories []string
	// MinSize filters out results below this size in bytes.
	MinSize int64
	// HomeDir overrides the home directory (useful for testing).
	HomeDir string
}

// Scanner orchestrates disk usage scanning across categories.
type Scanner struct {
	homeDir string
}

// New creates a new Scanner. If homeDir is empty, os.UserHomeDir() is used.
func New(homeDir string) (*Scanner, error) {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("determining home directory: %w", err)
		}
	}
	return &Scanner{homeDir: homeDir}, nil
}

// Scan runs a scan across the specified categories and returns results.
// It uses context for cancellation support.
func (s *Scanner) Scan(ctx context.Context, opts Options) ([]ScanResult, error) {
	categories := DefaultCategories()

	if len(opts.Categories) > 0 {
		filtered := make([]Category, 0, len(opts.Categories))
		for _, name := range opts.Categories {
			cat, err := FindCategory(name)
			if err != nil {
				return nil, err
			}
			filtered = append(filtered, cat)
		}
		categories = filtered
	}

	homeDir := s.homeDir
	if opts.HomeDir != "" {
		homeDir = opts.HomeDir
	}

	var (
		results []ScanResult
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	for _, cat := range categories {
		wg.Add(1)
		go func(c Category) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			result := s.scanCategory(ctx, c, homeDir)

			if opts.MinSize > 0 && result.TotalSize < opts.MinSize {
				return
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(cat)
	}

	wg.Wait()

	if ctx.Err() != nil {
		return results, ctx.Err()
	}

	// Sort by size, largest first.
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalSize > results[j].TotalSize
	})

	return results, nil
}

// scanCategory scans a single category and returns its result.
func (s *Scanner) scanCategory(ctx context.Context, cat Category, homeDir string) ScanResult {
	result := ScanResult{
		CategoryName: cat.Name,
		DisplayName:  cat.DisplayName,
		Risk:         cat.Risk,
	}

	// Use custom scanner if available.
	if cat.CustomScan != nil {
		entries, err := cat.CustomScan(homeDir)
		if err != nil {
			result.Error = err.Error()
			return result
		}
		result.Entries = entries
		for _, e := range entries {
			result.TotalSize += e.Size
		}
		return result
	}

	// Default: scan each path.
	paths := cat.Paths(homeDir)
	for _, p := range paths {
		select {
		case <-ctx.Done():
			return result
		default:
		}

		entries, err := scanPath(p)
		if err != nil {
			// Permission errors and missing dirs are not fatal.
			if !os.IsNotExist(err) && !os.IsPermission(err) {
				if result.Error == "" {
					result.Error = err.Error()
				}
			}
			continue
		}
		result.Entries = append(result.Entries, entries...)
	}

	for _, e := range result.Entries {
		result.TotalSize += e.Size
	}

	return result
}

// scanPath scans a directory path and returns entries for its top-level children.
func scanPath(path string) ([]ScanEntry, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return []ScanEntry{{
			Path:         path,
			Size:         info.Size(),
			Description:  filepath.Base(path),
			LastModified: info.ModTime(),
			IsDir:        false,
		}}, nil
	}

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", path, err)
	}

	var entries []ScanEntry
	for _, de := range dirEntries {
		fullPath := filepath.Join(path, de.Name())
		size, modTime := dirSizeAndModTime(fullPath)
		entries = append(entries, ScanEntry{
			Path:         fullPath,
			Size:         size,
			Description:  de.Name(),
			LastModified: modTime,
			IsDir:        de.IsDir(),
		})
	}

	return entries, nil
}

// dirSizeAndModTime calculates the total size and most recent modification
// time of a file or directory tree.
func dirSizeAndModTime(path string) (int64, time.Time) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, time.Time{}
	}

	if !info.IsDir() {
		return info.Size(), info.ModTime()
	}

	var totalSize int64
	latestMod := info.ModTime()

	_ = filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors (permission denied, etc.)
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			totalSize += fi.Size()
		}
		if fi.ModTime().After(latestMod) {
			latestMod = fi.ModTime()
		}
		return nil
	})

	return totalSize, latestMod
}

// scanDocker uses the docker CLI to estimate Docker disk usage.
func scanDocker(homeDir string) ([]ScanEntry, error) {
	// Check if Docker is available.
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return nil, nil // Docker not installed, not an error.
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, dockerPath, "system", "df", "--format", "{{.Type}}\t{{.Size}}\t{{.Reclaimable}}")
	out, err := cmd.Output()
	if err != nil {
		// Docker daemon might not be running.
		return nil, nil
	}

	var entries []ScanEntry
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		typeName := strings.TrimSpace(parts[0])
		reclaimable := strings.TrimSpace(parts[2])

		size := parseDockerSize(reclaimable)
		if size > 0 {
			entries = append(entries, ScanEntry{
				Path:         fmt.Sprintf("docker:%s", typeName),
				Size:         size,
				Description:  fmt.Sprintf("Docker %s (reclaimable)", typeName),
				LastModified: time.Now(),
				IsDir:        false,
			})
		}
	}

	return entries, nil
}

// parseDockerSize parses Docker's human-readable size strings like "2.5GB", "100MB".
// It extracts the numeric portion before any parenthesis (e.g., "1.2GB (500MB)" → parse "1.2GB").
func parseDockerSize(s string) int64 {
	// Docker often outputs "1.2GB (500MB reclaimable)" — take the first part.
	if idx := strings.Index(s, "("); idx > 0 {
		s = strings.TrimSpace(s[:idx])
	}

	s = strings.TrimSpace(s)
	if s == "" || s == "0B" {
		return 0
	}

	re := regexp.MustCompile(`^([\d.]+)\s*([KMGT]?B)$`)
	matches := re.FindStringSubmatch(strings.ToUpper(s))
	if matches == nil {
		return 0
	}

	val, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	switch matches[2] {
	case "KB":
		return int64(val * 1024)
	case "MB":
		return int64(val * 1024 * 1024)
	case "GB":
		return int64(val * 1024 * 1024 * 1024)
	case "TB":
		return int64(val * 1024 * 1024 * 1024 * 1024)
	default:
		return int64(val)
	}
}

// scanSystemCaches scans ~/Library/Caches with per-app breakdown.
func scanSystemCaches(homeDir string) ([]ScanEntry, error) {
	cachesDir := filepath.Join(homeDir, "Library", "Caches")
	return scanPath(cachesDir)
}

// scanOldDownloads finds files in ~/Downloads older than 30 days.
func scanOldDownloads(homeDir string) ([]ScanEntry, error) {
	downloadsDir := filepath.Join(homeDir, "Downloads")
	cutoff := time.Now().AddDate(0, 0, -30)

	dirEntries, err := os.ReadDir(downloadsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading downloads: %w", err)
	}

	var entries []ScanEntry
	for _, de := range dirEntries {
		fullPath := filepath.Join(downloadsDir, de.Name())
		size, modTime := dirSizeAndModTime(fullPath)

		if modTime.Before(cutoff) {
			entries = append(entries, ScanEntry{
				Path:         fullPath,
				Size:         size,
				Description:  fmt.Sprintf("%s (last modified: %s)", de.Name(), modTime.Format("2006-01-02")),
				LastModified: modTime,
				IsDir:        de.IsDir(),
			})
		}
	}

	return entries, nil
}

// scanAppLeftovers scans ~/Library/Application Support for directories
// whose corresponding app is not in /Applications.
func scanAppLeftovers(homeDir string) ([]ScanEntry, error) {
	appSupportDir := filepath.Join(homeDir, "Library", "Application Support")
	applicationsDir := filepath.Join("/", "Applications")

	dirEntries, err := os.ReadDir(appSupportDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading application support: %w", err)
	}

	// Build a set of installed app names (lowercase, without .app).
	installedApps := make(map[string]bool)
	appEntries, _ := os.ReadDir(applicationsDir)
	for _, ae := range appEntries {
		name := strings.TrimSuffix(ae.Name(), ".app")
		installedApps[strings.ToLower(name)] = true
	}

	var entries []ScanEntry
	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		// Skip Apple/system directories.
		name := de.Name()
		if strings.HasPrefix(name, "com.apple.") || name == "Apple" || name == ".DS_Store" {
			continue
		}

		// Check if a corresponding app exists.
		if installedApps[strings.ToLower(name)] {
			continue
		}

		fullPath := filepath.Join(appSupportDir, name)
		size, modTime := dirSizeAndModTime(fullPath)

		if size > 0 {
			entries = append(entries, ScanEntry{
				Path:         fullPath,
				Size:         size,
				Description:  fmt.Sprintf("%s (app not found in /Applications)", name),
				LastModified: modTime,
				IsDir:        true,
			})
		}
	}

	return entries, nil
}
