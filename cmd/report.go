package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/aviv4339/mac-cleaner/internal/scanner"
	"github.com/aviv4339/mac-cleaner/internal/ui"
	"github.com/aviv4339/mac-cleaner/internal/web"
	"github.com/spf13/cobra"
)

var (
	reportPort      int
	reportNoBrowser bool
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Launch a web dashboard with scan results",
	Long: `Run a scan and launch a local web dashboard in your browser.

The dashboard shows scan results with interactive sorting, expandable
categories, and a re-scan button — all powered by HTMX.

Press Ctrl+C to stop the server.`,
	RunE: runReport,
}

func init() {
	reportCmd.Flags().IntVarP(&reportPort, "port", "p", 8080, "Port for the web server")
	reportCmd.Flags().BoolVar(&reportNoBrowser, "no-browser", false, "Don't auto-open browser")

	rootCmd.AddCommand(reportCmd)
}

func runReport(_ *cobra.Command, _ []string) error {
	out := ui.New(noColor)
	out.PrintBanner()
	out.PrintInfo("Scanning for reclaimable disk space...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := scanner.New("")
	if err != nil {
		return fmt.Errorf("initializing scanner: %w", err)
	}

	results, err := s.Scan(ctx, scanner.Options{})
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", reportPort)
	srv, err := web.NewServer(addr, s, results)
	if err != nil {
		return fmt.Errorf("initializing web server: %w", err)
	}

	// Start the server in a goroutine.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}

	url := fmt.Sprintf("http://%s", addr)
	out.PrintInfo(fmt.Sprintf("Dashboard running at %s", url))
	out.PrintInfo("Press Ctrl+C to stop")

	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil {
			// Ignore closed errors from shutdown.
			select {
			case <-ctx.Done():
			default:
				fmt.Fprintf(os.Stderr, "server error: %v\n", serveErr)
			}
		}
	}()

	if !reportNoBrowser {
		// macOS: open browser.
		_ = exec.Command("open", url).Start()
	}

	// Wait for interrupt.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	out.PrintInfo("Shutting down...")
	cancel()
	return srv.Shutdown(context.Background())
}
