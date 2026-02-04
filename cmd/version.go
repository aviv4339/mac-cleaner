package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// These variables are set at build time via ldflags.
var (
	// Version is the semantic version.
	Version = "dev"
	// Commit is the git commit SHA.
	Commit = "unknown"
	// Date is the build date.
	Date = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("mac-cleaner %s\n", Version)
		fmt.Printf("  Commit:  %s\n", Commit)
		fmt.Printf("  Built:   %s\n", Date)
		fmt.Printf("  Go:      %s\n", runtime.Version())
		fmt.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
