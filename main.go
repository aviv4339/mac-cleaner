// mac-cleaner is a macOS disk cleanup utility that scans for and removes
// unnecessary files, caches, and build artifacts from development tools.
package main

import (
	"os"

	"github.com/aviv4339/mac-cleaner/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
