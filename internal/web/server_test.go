package web

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aviv4339/mac-cleaner/internal/scanner"
)

func testResults() []scanner.ScanResult {
	return []scanner.ScanResult{
		{
			CategoryName: "homebrew",
			DisplayName:  "Homebrew",
			Risk:         scanner.RiskSafe,
			TotalSize:    1024 * 1024 * 500, // 500 MB
			Entries: []scanner.ScanEntry{
				{Path: "/usr/local/Cellar/foo", Size: 1024 * 1024 * 300, Description: "foo"},
				{Path: "/usr/local/Cellar/bar", Size: 1024 * 1024 * 200, Description: "bar"},
			},
		},
		{
			CategoryName: "docker",
			DisplayName:  "Docker",
			Risk:         scanner.RiskModerate,
			TotalSize:    1024 * 1024 * 1024 * 2, // 2 GB
			Entries: []scanner.ScanEntry{
				{Path: "docker:Images", Size: 1024 * 1024 * 1024, Description: "Docker Images (reclaimable)"},
				{Path: "docker:Containers", Size: 1024 * 1024 * 512, Description: "Docker Containers (reclaimable)"},
				{Path: "docker:Volumes", Size: 1024 * 1024 * 256, Description: "Docker Volumes (reclaimable)"},
				{Path: "docker:BuildCache", Size: 1024 * 1024 * 256, Description: "Docker BuildCache (reclaimable)"},
			},
		},
		{
			CategoryName: "downloads",
			DisplayName:  "Downloads",
			Risk:         scanner.RiskCaution,
			TotalSize:    1024 * 1024 * 100, // 100 MB
			Entries: []scanner.ScanEntry{
				{Path: "/Users/test/Downloads/old.zip", Size: 1024 * 1024 * 100, Description: "old.zip"},
			},
		},
	}
}

// testResultsWithManyEntries returns a result with >5 entries for testing expand/collapse.
func testResultsWithManyEntries() []scanner.ScanResult {
	entries := make([]scanner.ScanEntry, 8)
	for i := range entries {
		entries[i] = scanner.ScanEntry{
			Path:        "/cache/item" + string(rune('a'+i)),
			Size:        int64(1024 * (8 - i)),
			Description: "item-" + string(rune('a'+i)),
		}
	}
	return []scanner.ScanResult{
		{
			CategoryName: "big-category",
			DisplayName:  "Big Category",
			Risk:         scanner.RiskSafe,
			TotalSize:    36 * 1024,
			Entries:      entries,
		},
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	s, err := scanner.New("/tmp/nonexistent-test-home")
	if err != nil {
		t.Fatalf("creating scanner: %v", err)
	}
	srv, err := NewServer("127.0.0.1:0", s, testResults())
	if err != nil {
		t.Fatalf("creating server: %v", err)
	}
	return srv
}

func newTestServerWithResults(t *testing.T, results []scanner.ScanResult) *Server {
	t.Helper()
	s, err := scanner.New("/tmp/nonexistent-test-home")
	if err != nil {
		t.Fatalf("creating scanner: %v", err)
	}
	srv, err := NewServer("127.0.0.1:0", s, results)
	if err != nil {
		t.Fatalf("creating server: %v", err)
	}
	return srv
}

// --- Dashboard handler tests ---

func TestHandleDashboard(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	checks := []string{
		"mac-cleaner",
		"bunghole",
		"/static/logo.webp",
		"Total Reclaimable Space",
		"Homebrew",
		"Docker",
		"Downloads",
		"Re-scan",
		"Sort by:",
		"Are you threatening my disk space?",
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Errorf("dashboard missing expected text: %q", check)
		}
	}

	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}
}

func TestHandleDashboardHTMXAttributes(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	// Verify HTMX attributes are present.
	htmxChecks := []string{
		`hx-get="/api/scan"`,
		`hx-target="#results"`,
		`hx-get="/api/results?sort=size"`,
		`hx-get="/api/results?sort=name"`,
		`hx-get="/api/results?sort=risk"`,
	}
	for _, check := range htmxChecks {
		if !strings.Contains(body, check) {
			t.Errorf("dashboard missing HTMX attribute: %q", check)
		}
	}
}

func TestHandleDashboard404(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestHandleDashboardEmptyResults(t *testing.T) {
	srv := newTestServerWithResults(t, nil)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Total Reclaimable Space") {
		t.Error("expected dashboard to render even with empty results")
	}
}

// --- Scan handler tests ---

func TestHandleScan(t *testing.T) {
	// Create a temp dir with a known file to scan.
	tmpHome := t.TempDir()
	cacheDir := filepath.Join(tmpHome, "Library", "Caches", "Homebrew")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(cacheDir, "test.tar.gz")
	if err := os.WriteFile(testFile, make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := scanner.New(tmpHome)
	if err != nil {
		t.Fatalf("creating scanner: %v", err)
	}
	srv, err := NewServer("127.0.0.1:0", s, nil)
	if err != nil {
		t.Fatalf("creating server: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/scan", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}

	// After scan, cached results should be updated.
	srv.mu.RLock()
	hasResults := len(srv.results) > 0
	srv.mu.RUnlock()
	if !hasResults {
		t.Error("expected results to be cached after scan")
	}
}

// --- Results handler tests ---

func TestHandleResults(t *testing.T) {
	srv := newTestServer(t)

	tests := []struct {
		sort     string
		firstCat string
		lastCat  string
	}{
		{"size", "Docker", "Downloads"},
		{"name", "Docker", "Homebrew"},
		{"risk", "Downloads", "Homebrew"},
		{"", "Docker", "Downloads"}, // default is size
	}

	for _, tt := range tests {
		t.Run("sort="+tt.sort, func(t *testing.T) {
			url := "/api/results"
			if tt.sort != "" {
				url += "?sort=" + tt.sort
			}
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", w.Code)
			}

			body := w.Body.String()
			firstIdx := strings.Index(body, tt.firstCat)
			lastIdx := strings.Index(body, tt.lastCat)
			if firstIdx < 0 {
				t.Errorf("expected %q to appear in results", tt.firstCat)
			}
			if lastIdx < 0 {
				t.Errorf("expected %q to appear in results", tt.lastCat)
			}
			if firstIdx >= 0 && lastIdx >= 0 && firstIdx > lastIdx {
				t.Errorf("expected %q (idx %d) to appear before %q (idx %d)",
					tt.firstCat, firstIdx, tt.lastCat, lastIdx)
			}
		})
	}
}

// --- Category handler tests ---

func TestHandleCategory(t *testing.T) {
	srv := newTestServer(t)

	// Default (collapsed).
	req := httptest.NewRequest("GET", "/api/category/homebrew", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "foo") {
		t.Error("expected entry 'foo' in category response")
	}
	if !strings.Contains(body, "bar") {
		t.Error("expected entry 'bar' in category response")
	}
}

func TestHandleCategoryExpanded(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/category/homebrew?expanded=true", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Show less") {
		t.Error("expected 'Show less' button in expanded response")
	}
}

func TestHandleCategoryCollapsed(t *testing.T) {
	srv := newTestServerWithResults(t, testResultsWithManyEntries())

	// Collapsed explicitly.
	req := httptest.NewRequest("GET", "/api/category/big-category?collapsed=true", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	// Should show the "Show all" button since we only show 5 of 8.
	if !strings.Contains(body, "Show all") {
		t.Error("expected 'Show all' button when collapsed with >5 entries")
	}
	if strings.Contains(body, "Show less") {
		t.Error("should not contain 'Show less' when collapsed")
	}
}

func TestHandleCategoryExpandedThenCollapsed(t *testing.T) {
	srv := newTestServerWithResults(t, testResultsWithManyEntries())

	// Expanded=true but collapsed=true: collapsed wins.
	req := httptest.NewRequest("GET", "/api/category/big-category?expanded=true&collapsed=true", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "Show less") {
		t.Error("collapsed=true should take priority over expanded=true")
	}
}

func TestHandleCategoryNotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/category/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestHandleCategoryEmptyName(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/category/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for empty category name, got %d", w.Code)
	}
}

// --- Server lifecycle tests ---

func TestServeAndShutdown(t *testing.T) {
	srv := newTestServer(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	// Give server a moment to start.
	time.Sleep(50 * time.Millisecond)

	// Make a request to verify it's running.
	resp, err := http.Get("http://" + ln.Addr().String() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	// Shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	// Serve should return http.ErrServerClosed.
	serveErr := <-errCh
	if serveErr != http.ErrServerClosed {
		t.Errorf("expected ErrServerClosed, got %v", serveErr)
	}
}

// --- Template function tests ---

func TestRiskClass(t *testing.T) {
	tests := []struct {
		risk scanner.RiskLevel
		want string
	}{
		{scanner.RiskSafe, "text-green-400"},
		{scanner.RiskModerate, "text-yellow-400"},
		{scanner.RiskCaution, "text-red-400"},
		{scanner.RiskLevel(99), "text-slate-400"},
	}
	for _, tt := range tests {
		if got := riskClass(tt.risk); got != tt.want {
			t.Errorf("riskClass(%v) = %q, want %q", tt.risk, got, tt.want)
		}
	}
}

func TestRiskBgClass(t *testing.T) {
	tests := []struct {
		risk scanner.RiskLevel
		want string
	}{
		{scanner.RiskSafe, "bg-green-900/50"},
		{scanner.RiskModerate, "bg-yellow-900/50"},
		{scanner.RiskCaution, "bg-red-900/50"},
		{scanner.RiskLevel(99), "bg-slate-700"},
	}
	for _, tt := range tests {
		if got := riskBgClass(tt.risk); got != tt.want {
			t.Errorf("riskBgClass(%v) = %q, want %q", tt.risk, got, tt.want)
		}
	}
}

func TestRiskBarClass(t *testing.T) {
	tests := []struct {
		risk scanner.RiskLevel
		want string
	}{
		{scanner.RiskSafe, "bg-green-500"},
		{scanner.RiskModerate, "bg-yellow-500"},
		{scanner.RiskCaution, "bg-red-500"},
		{scanner.RiskLevel(99), "bg-slate-500"},
	}
	for _, tt := range tests {
		if got := riskBarClass(tt.risk); got != tt.want {
			t.Errorf("riskBarClass(%v) = %q, want %q", tt.risk, got, tt.want)
		}
	}
}

func TestRiskLabel(t *testing.T) {
	tests := []struct {
		risk scanner.RiskLevel
		want string
	}{
		{scanner.RiskSafe, "safe"},
		{scanner.RiskModerate, "moderate"},
		{scanner.RiskCaution, "caution"},
		{scanner.RiskLevel(99), "unknown"},
	}
	for _, tt := range tests {
		if got := riskLabel(tt.risk); got != tt.want {
			t.Errorf("riskLabel(%v) = %q, want %q", tt.risk, got, tt.want)
		}
	}
}

func TestBarWidth(t *testing.T) {
	tests := []struct {
		size, max int64
		want      int
	}{
		{500, 1000, 50},
		{1000, 1000, 100},
		{0, 1000, 0},
		{1, 1000, 1},   // minimum 1% for nonzero
		{100, 0, 0},    // zero max
		{0, 0, 0},      // both zero
		{333, 1000, 33}, // rounds down
	}
	for _, tt := range tests {
		if got := barWidth(tt.size, tt.max); got != tt.want {
			t.Errorf("barWidth(%d, %d) = %d, want %d", tt.size, tt.max, got, tt.want)
		}
	}
}

func TestEntriesData(t *testing.T) {
	result := testResults()[1] // Docker with 4 entries

	t.Run("truncated", func(t *testing.T) {
		data := entriesData(result, 2, false)
		if len(data.Entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(data.Entries))
		}
		if data.TotalEntries != 4 {
			t.Errorf("expected TotalEntries=4, got %d", data.TotalEntries)
		}
		if data.Expanded {
			t.Error("expected Expanded=false")
		}
		if data.CategoryName != "docker" {
			t.Errorf("expected CategoryName=docker, got %q", data.CategoryName)
		}
	})

	t.Run("all entries", func(t *testing.T) {
		data := entriesData(result, 10, true)
		if len(data.Entries) != 4 {
			t.Errorf("expected 4 entries (all), got %d", len(data.Entries))
		}
		if !data.Expanded {
			t.Error("expected Expanded=true")
		}
	})

	t.Run("exact count", func(t *testing.T) {
		data := entriesData(result, 4, false)
		if len(data.Entries) != 4 {
			t.Errorf("expected 4 entries, got %d", len(data.Entries))
		}
	})
}

func TestBuildDashboardData(t *testing.T) {
	t.Run("with results", func(t *testing.T) {
		results := testResults()
		data := buildDashboardData(results)

		if data.CategoryCount != 3 {
			t.Errorf("expected 3 categories, got %d", data.CategoryCount)
		}
		if data.EntryCount != 7 {
			t.Errorf("expected 7 entries, got %d", data.EntryCount)
		}
		if data.TotalSize == 0 {
			t.Error("expected non-zero TotalSize")
		}
		expectedMax := int64(1024 * 1024 * 1024 * 2) // Docker = 2 GB
		if data.MaxSize != expectedMax {
			t.Errorf("expected MaxSize=%d, got %d", expectedMax, data.MaxSize)
		}
	})

	t.Run("empty results", func(t *testing.T) {
		data := buildDashboardData(nil)

		if data.CategoryCount != 0 {
			t.Errorf("expected 0 categories, got %d", data.CategoryCount)
		}
		if data.EntryCount != 0 {
			t.Errorf("expected 0 entries, got %d", data.EntryCount)
		}
		if data.TotalSize != 0 {
			t.Errorf("expected TotalSize=0, got %d", data.TotalSize)
		}
		if data.MaxSize != 0 {
			t.Errorf("expected MaxSize=0, got %d", data.MaxSize)
		}
	})

	t.Run("zero size categories excluded from count", func(t *testing.T) {
		results := []scanner.ScanResult{
			{CategoryName: "empty", DisplayName: "Empty", TotalSize: 0},
			{CategoryName: "notempty", DisplayName: "Not Empty", TotalSize: 100,
				Entries: []scanner.ScanEntry{{Size: 100}}},
		}
		data := buildDashboardData(results)
		if data.CategoryCount != 1 {
			t.Errorf("expected 1 category (zero-size excluded), got %d", data.CategoryCount)
		}
	})
}

// --- Dashboard content integration tests ---

func TestDashboardShowsRiskBadges(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	for _, badge := range []string{"safe", "moderate", "caution"} {
		if !strings.Contains(body, badge) {
			t.Errorf("dashboard missing risk badge: %q", badge)
		}
	}
}

func TestDashboardShowsSizeBars(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	// Should have progress bars with width styles.
	if !strings.Contains(body, "width:") {
		t.Error("expected size bars with width styles in dashboard")
	}
}

func TestDashboardShowsFormattedSizes(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	// Docker is 2 GB, Homebrew is 500 MB, Downloads is 100 MB.
	if !strings.Contains(body, "2.0 GB") {
		t.Error("expected '2.0 GB' for Docker size")
	}
	if !strings.Contains(body, "500.0 MB") {
		t.Error("expected '500.0 MB' for Homebrew size")
	}
}

func TestDashboardShowsExpandButton(t *testing.T) {
	srv := newTestServerWithResults(t, testResultsWithManyEntries())

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Show all 8 items") {
		t.Error("expected 'Show all 8 items' button for category with >5 entries")
	}
}

// --- Static asset tests ---

func TestHandleLogo(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/static/logo.webp", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "image/webp" {
		t.Errorf("expected Content-Type image/webp, got %q", ct)
	}

	if w.Body.Len() == 0 {
		t.Error("expected non-empty logo response body")
	}

	cc := w.Header().Get("Cache-Control")
	if !strings.Contains(cc, "max-age") {
		t.Errorf("expected Cache-Control with max-age, got %q", cc)
	}
}
