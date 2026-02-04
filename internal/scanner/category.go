// Package scanner provides the core scanning engine for mac-cleaner.
// It discovers and measures disk usage across various development tool
// caches, build artifacts, and system files.
package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RiskLevel indicates how safe it is to delete a category's files.
type RiskLevel int

const (
	// RiskSafe means files can be safely deleted with no side effects.
	// Examples: caches that will be regenerated, old logs.
	RiskSafe RiskLevel = iota
	// RiskModerate means files can be deleted but may cause minor inconvenience.
	// Examples: downloaded packages that will need re-downloading.
	RiskModerate
	// RiskCaution means files should be reviewed before deletion.
	// Examples: old downloads, application data.
	RiskCaution
)

// String returns a human-readable representation of the risk level.
func (r RiskLevel) String() string {
	switch r {
	case RiskSafe:
		return "safe"
	case RiskModerate:
		return "moderate"
	case RiskCaution:
		return "caution"
	default:
		return "unknown"
	}
}

// Category represents a scannable category of files on disk.
type Category struct {
	// Name is the unique identifier for this category.
	Name string
	// DisplayName is the human-readable name shown in output.
	DisplayName string
	// Description explains what this category contains.
	Description string
	// Risk indicates how safe it is to delete these files.
	Risk RiskLevel
	// Paths returns the list of filesystem paths to scan for this category.
	// The function receives the user's home directory as an argument.
	Paths func(homeDir string) []string
	// CustomScan optionally provides a custom scanning function.
	// If non-nil, this is used instead of the default directory walker.
	// It receives the home directory and returns a list of scan entries.
	CustomScan func(homeDir string) ([]ScanEntry, error)
}

// ScanEntry represents a single item found during scanning.
type ScanEntry struct {
	// Path is the filesystem path of the item.
	Path string `json:"path"`
	// Size is the total size in bytes.
	Size int64 `json:"size"`
	// Description provides additional context about this entry.
	Description string `json:"description"`
	// LastModified is when the item was last modified.
	LastModified time.Time `json:"last_modified"`
	// IsDir indicates whether this entry is a directory.
	IsDir bool `json:"is_dir"`
}

// ScanResult holds the complete results for a single category scan.
type ScanResult struct {
	// CategoryName is the name of the scanned category.
	CategoryName string `json:"category_name"`
	// DisplayName is the human-readable category name.
	DisplayName string `json:"display_name"`
	// Risk is the risk level for this category.
	Risk RiskLevel `json:"risk"`
	// TotalSize is the aggregate size of all entries in bytes.
	TotalSize int64 `json:"total_size"`
	// Entries contains the individual items found.
	Entries []ScanEntry `json:"entries"`
	// Error contains any error encountered during scanning.
	Error string `json:"error,omitempty"`
}

// DefaultCategories returns all built-in scan categories.
func DefaultCategories() []Category {
	return []Category{
		homebrewCategory(),
		dockerCategory(),
		xcodeCategory(),
		nodeCategory(),
		pythonCategory(),
		rustCategory(),
		goCategory(),
		rubyCategory(),
		javaCategory(),
		systemCachesCategory(),
		systemLogsCategory(),
		trashCategory(),
		downloadsCategory(),
		appLeftoversCategory(),
		mailCategory(),
		composerCategory(),
		cocoapodsCategory(),
	}
}

// CategoryNames returns all available category names.
func CategoryNames() []string {
	cats := DefaultCategories()
	names := make([]string, len(cats))
	for i, c := range cats {
		names[i] = c.Name
	}
	return names
}

// FindCategory returns the category with the given name, or an error if not found.
func FindCategory(name string) (Category, error) {
	for _, c := range DefaultCategories() {
		if c.Name == name {
			return c, nil
		}
	}
	return Category{}, fmt.Errorf("unknown category: %q", name)
}

func homebrewCategory() Category {
	return Category{
		Name:        "homebrew",
		DisplayName: "Homebrew",
		Description: "Homebrew package manager caches and old downloads",
		Risk:        RiskSafe,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, "Library", "Caches", "Homebrew"),
			}
		},
	}
}

func dockerCategory() Category {
	return Category{
		Name:        "docker",
		DisplayName: "Docker",
		Description: "Docker images, containers, volumes, and build cache",
		Risk:        RiskModerate,
		Paths: func(_ string) []string {
			return []string{
				filepath.Join(string(os.PathSeparator), "Users", "Shared", "Docker"),
			}
		},
		CustomScan: scanDocker,
	}
}

func xcodeCategory() Category {
	return Category{
		Name:        "xcode",
		DisplayName: "Xcode",
		Description: "Xcode DerivedData, archives, device support, and simulators",
		Risk:        RiskSafe,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, "Library", "Developer", "Xcode", "DerivedData"),
				filepath.Join(homeDir, "Library", "Developer", "Xcode", "Archives"),
				filepath.Join(homeDir, "Library", "Developer", "Xcode", "iOS DeviceSupport"),
				filepath.Join(homeDir, "Library", "Developer", "CoreSimulator", "Devices"),
			}
		},
	}
}

func nodeCategory() Category {
	return Category{
		Name:        "node",
		DisplayName: "Node.js",
		Description: "npm/yarn/pnpm caches and abandoned node_modules directories",
		Risk:        RiskSafe,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, ".npm", "_cacache"),
				filepath.Join(homeDir, ".yarn", "cache"),
				filepath.Join(homeDir, "Library", "pnpm", "store"),
				filepath.Join(homeDir, "Library", "Caches", "Yarn"),
			}
		},
	}
}

func pythonCategory() Category {
	return Category{
		Name:        "python",
		DisplayName: "Python",
		Description: "pip cache, __pycache__ directories, conda packages",
		Risk:        RiskSafe,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, "Library", "Caches", "pip"),
				filepath.Join(homeDir, ".conda", "pkgs"),
				filepath.Join(homeDir, "anaconda3", "pkgs"),
				filepath.Join(homeDir, "miniconda3", "pkgs"),
			}
		},
	}
}

func rustCategory() Category {
	return Category{
		Name:        "rust",
		DisplayName: "Rust",
		Description: "Cargo registry cache and target build directories",
		Risk:        RiskSafe,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, ".cargo", "registry"),
			}
		},
	}
}

func goCategory() Category {
	return Category{
		Name:        "go",
		DisplayName: "Go",
		Description: "Go module cache",
		Risk:        RiskSafe,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, "go", "pkg", "mod", "cache"),
			}
		},
	}
}

func rubyCategory() Category {
	return Category{
		Name:        "ruby",
		DisplayName: "Ruby",
		Description: "Gem cache and Bundler cache",
		Risk:        RiskSafe,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, ".gem"),
				filepath.Join(homeDir, ".bundle", "cache"),
				filepath.Join(homeDir, "Library", "Caches", "CocoaPods"),
			}
		},
	}
}

func javaCategory() Category {
	return Category{
		Name:        "java",
		DisplayName: "Java/Kotlin",
		Description: "Gradle caches and Maven local repository",
		Risk:        RiskModerate,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, ".gradle", "caches"),
				filepath.Join(homeDir, ".m2", "repository"),
			}
		},
	}
}

func systemCachesCategory() Category {
	return Category{
		Name:        "system-caches",
		DisplayName: "System Caches",
		Description: "Per-application caches in ~/Library/Caches and /Library/Caches",
		Risk:        RiskSafe,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, "Library", "Caches"),
			}
		},
		CustomScan: scanSystemCaches,
	}
}

func systemLogsCategory() Category {
	return Category{
		Name:        "system-logs",
		DisplayName: "System Logs",
		Description: "User and system log files",
		Risk:        RiskSafe,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, "Library", "Logs"),
			}
		},
	}
}

func trashCategory() Category {
	return Category{
		Name:        "trash",
		DisplayName: "Trash",
		Description: "Files in the user's Trash",
		Risk:        RiskSafe,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, ".Trash"),
			}
		},
	}
}

func downloadsCategory() Category {
	return Category{
		Name:        "downloads",
		DisplayName: "Downloads",
		Description: "Files in ~/Downloads older than 30 days",
		Risk:        RiskCaution,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, "Downloads"),
			}
		},
		CustomScan: scanOldDownloads,
	}
}

func appLeftoversCategory() Category {
	return Category{
		Name:        "app-leftovers",
		DisplayName: "Application Leftovers",
		Description: "Application Support data for apps that may no longer be installed",
		Risk:        RiskCaution,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, "Library", "Application Support"),
			}
		},
		CustomScan: scanAppLeftovers,
	}
}

func mailCategory() Category {
	return Category{
		Name:        "mail",
		DisplayName: "Mail Attachments",
		Description: "Cached mail attachments and downloads",
		Risk:        RiskModerate,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, "Library", "Mail", "V10"),
				filepath.Join(homeDir, "Library", "Mail", "V9"),
				filepath.Join(homeDir, "Library", "Mail", "V8"),
				filepath.Join(homeDir, "Library", "Containers", "com.apple.mail", "Data", "Library", "Mail Downloads"),
			}
		},
	}
}

func composerCategory() Category {
	return Category{
		Name:        "composer",
		DisplayName: "Composer (PHP)",
		Description: "PHP Composer package cache",
		Risk:        RiskSafe,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, ".composer", "cache"),
			}
		},
	}
}

func cocoapodsCategory() Category {
	return Category{
		Name:        "cocoapods",
		DisplayName: "CocoaPods",
		Description: "CocoaPods spec and download caches",
		Risk:        RiskSafe,
		Paths: func(homeDir string) []string {
			return []string{
				filepath.Join(homeDir, "Library", "Caches", "CocoaPods"),
			}
		},
	}
}
