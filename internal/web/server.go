// Package web provides an HTTP server that serves a dashboard for mac-cleaner scan results.
package web

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/aviv4339/mac-cleaner/internal/scanner"
	"github.com/aviv4339/mac-cleaner/internal/ui"
)

//go:embed templates/*.html templates/partials/*.html
var templateFS embed.FS

//go:embed static/logo.webp
var logoData []byte

// EntriesData is the data passed to the category_entries partial template.
type EntriesData struct {
	CategoryName string
	Entries      []scanner.ScanEntry
	TotalEntries int
	ShowCount    int
	Expanded     bool
}

// DashboardData is the data passed to the dashboard template.
type DashboardData struct {
	Results       []scanner.ScanResult
	TotalSize     int64
	MaxSize       int64
	CategoryCount int
	EntryCount    int
}

// Server is the web dashboard HTTP server.
type Server struct {
	addr    string
	scanner *scanner.Scanner
	tmpl    *template.Template
	results []scanner.ScanResult
	mu      sync.RWMutex
	httpSrv *http.Server
}

// NewServer creates a new web dashboard server.
func NewServer(addr string, s *scanner.Scanner, initialResults []scanner.ScanResult) (*Server, error) {
	funcMap := template.FuncMap{
		"formatSize":   ui.FormatSize,
		"riskClass":    riskClass,
		"riskBgClass":  riskBgClass,
		"riskBarClass": riskBarClass,
		"riskLabel":    riskLabel,
		"barWidth":     barWidth,
		"entriesData":  entriesData,
		"sub":          func(a, b int) int { return a - b },
		"mul":          func(a, b int) int { return a * b },
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS,
		"templates/layout.html",
		"templates/dashboard.html",
		"templates/partials/category_entries.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}

	srv := &Server{
		addr:    addr,
		scanner: s,
		tmpl:    tmpl,
		results: initialResults,
	}
	srv.httpSrv = &http.Server{Addr: addr, Handler: srv.Handler()}
	return srv, nil
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/static/logo.webp", handleLogo)
	mux.HandleFunc("/api/scan", s.handleScan)
	mux.HandleFunc("/api/results", s.handleResults)
	mux.HandleFunc("/api/category/", s.handleCategory)
	return mux
}

// Serve starts the HTTP server on the given listener.
func (s *Server) Serve(ln net.Listener) error {
	return s.httpSrv.Serve(ln)
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) dashboardData() DashboardData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return buildDashboardData(s.results)
}

func buildDashboardData(results []scanner.ScanResult) DashboardData {
	var totalSize, maxSize int64
	var entryCount int
	catCount := 0

	for _, r := range results {
		if r.TotalSize > 0 {
			catCount++
			totalSize += r.TotalSize
			entryCount += len(r.Entries)
			if r.TotalSize > maxSize {
				maxSize = r.TotalSize
			}
		}
	}

	return DashboardData{
		Results:       results,
		TotalSize:     totalSize,
		MaxSize:       maxSize,
		CategoryCount: catCount,
		EntryCount:    entryCount,
	}
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := s.dashboardData()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	results, err := s.scanner.Scan(r.Context(), scanner.Options{})
	if err != nil {
		http.Error(w, fmt.Sprintf("scan error: %v", err), http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.results = results
	s.mu.Unlock()

	data := buildDashboardData(results)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "results_section", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	sortBy := r.URL.Query().Get("sort")

	s.mu.RLock()
	results := make([]scanner.ScanResult, len(s.results))
	copy(results, s.results)
	s.mu.RUnlock()

	switch sortBy {
	case "name":
		sort.Slice(results, func(i, j int) bool {
			return strings.ToLower(results[i].DisplayName) < strings.ToLower(results[j].DisplayName)
		})
	case "risk":
		sort.Slice(results, func(i, j int) bool {
			return results[i].Risk > results[j].Risk
		})
	default: // "size" or empty
		sort.Slice(results, func(i, j int) bool {
			return results[i].TotalSize > results[j].TotalSize
		})
	}

	data := buildDashboardData(results)
	// Preserve the sorted order.
	data.Results = results

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "results_section", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleCategory(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/category/")
	if name == "" {
		http.NotFound(w, r)
		return
	}

	collapsed := r.URL.Query().Get("collapsed") == "true"
	expanded := r.URL.Query().Get("expanded") == "true"

	s.mu.RLock()
	var found *scanner.ScanResult
	for i := range s.results {
		if s.results[i].CategoryName == name {
			found = &s.results[i]
			break
		}
	}
	s.mu.RUnlock()

	if found == nil {
		http.NotFound(w, r)
		return
	}

	showCount := 5
	isExpanded := false
	if expanded && !collapsed {
		showCount = len(found.Entries)
		isExpanded = true
	}

	data := entriesData(*found, showCount, isExpanded)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "category_entries", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleLogo(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(logoData)
}

// Template helper functions.

func riskClass(risk scanner.RiskLevel) string {
	switch risk {
	case scanner.RiskSafe:
		return "text-green-400"
	case scanner.RiskModerate:
		return "text-yellow-400"
	case scanner.RiskCaution:
		return "text-red-400"
	default:
		return "text-slate-400"
	}
}

func riskBgClass(risk scanner.RiskLevel) string {
	switch risk {
	case scanner.RiskSafe:
		return "bg-green-900/50"
	case scanner.RiskModerate:
		return "bg-yellow-900/50"
	case scanner.RiskCaution:
		return "bg-red-900/50"
	default:
		return "bg-slate-700"
	}
}

func riskBarClass(risk scanner.RiskLevel) string {
	switch risk {
	case scanner.RiskSafe:
		return "bg-green-500"
	case scanner.RiskModerate:
		return "bg-yellow-500"
	case scanner.RiskCaution:
		return "bg-red-500"
	default:
		return "bg-slate-500"
	}
}

func riskLabel(risk scanner.RiskLevel) string {
	switch risk {
	case scanner.RiskSafe:
		return "safe"
	case scanner.RiskModerate:
		return "moderate"
	case scanner.RiskCaution:
		return "caution"
	default:
		return "unknown"
	}
}

func barWidth(size, maxSize int64) int {
	if maxSize == 0 {
		return 0
	}
	pct := int(float64(size) / float64(maxSize) * 100)
	if pct < 1 && size > 0 {
		return 1
	}
	return pct
}

func entriesData(result scanner.ScanResult, showCount int, expanded bool) EntriesData {
	entries := result.Entries
	if showCount < len(entries) {
		entries = entries[:showCount]
	}
	return EntriesData{
		CategoryName: result.CategoryName,
		Entries:      entries,
		TotalEntries: len(result.Entries),
		ShowCount:    showCount,
		Expanded:     expanded,
	}
}
