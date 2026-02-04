// Package cmd implements the CLI commands for mac-cleaner.
package cmd

import (
	"github.com/spf13/cobra"
)

var (
	noColor bool
)

// rootCmd is the base command for mac-cleaner.
var rootCmd = &cobra.Command{
	Use:   "mac-cleaner",
	Short: "A macOS disk cleanup utility for developers",
	Long: `mac-cleaner scans your macOS system for unnecessary files, caches, 
build artifacts, and other space-consuming items from development tools.

It supports cleaning up after Homebrew, Docker, Xcode, Node.js, Python, 
Rust, Go, Ruby, Java, and many more tools.

Run 'mac-cleaner scan' to see what's taking up space, or 
'mac-cleaner clean' to free up disk space interactively.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	// Set scan as the default command.
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "__help",
		Hidden: true,
	})
}
