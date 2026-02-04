# 🧹 mac-cleaner

A fast, safe macOS disk cleanup utility for developers. Scans and removes caches, build artifacts, and unnecessary files from development tools.

[![CI](https://github.com/aviv4339/mac-cleaner/actions/workflows/ci.yml/badge.svg)](https://github.com/aviv4339/mac-cleaner/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/aviv4339/mac-cleaner)](https://goreportcard.com/report/github.com/aviv4339/mac-cleaner)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **17 scan categories** covering all major development tools
- **Safe by default** — never deletes without explicit confirmation
- **Risk-level indicators** — know what's safe to delete
- **Fast parallel scanning** — concurrent category scanning
- **JSON output** — pipe results to other tools
- **Color-coded results** — instantly spot the biggest offenders
- **Docker-aware** — detects Docker disk usage via `docker system df`

## Installation

### Homebrew (recommended)

```bash
brew tap aviv4339/tap
brew install mac-cleaner
```

### Download Binary

Download the latest release from the [Releases page](https://github.com/aviv4339/mac-cleaner/releases).

```bash
# Apple Silicon (M1/M2/M3)
curl -LO https://github.com/aviv4339/mac-cleaner/releases/latest/download/mac-cleaner_darwin_arm64.tar.gz
tar xzf mac-cleaner_darwin_arm64.tar.gz
sudo mv mac-cleaner /usr/local/bin/

# Intel Mac
curl -LO https://github.com/aviv4339/mac-cleaner/releases/latest/download/mac-cleaner_darwin_amd64.tar.gz
tar xzf mac-cleaner_darwin_amd64.tar.gz
sudo mv mac-cleaner /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/aviv4339/mac-cleaner.git
cd mac-cleaner
make build
# Binary is in bin/mac-cleaner
```

## Usage

### Scan for reclaimable space

```bash
# Scan everything (default command)
mac-cleaner

# Explicit scan command
mac-cleaner scan

# Scan specific categories
mac-cleaner scan --category homebrew,docker,xcode

# Only show categories using more than 1GB
mac-cleaner scan --min-size 1GB

# Output as JSON
mac-cleaner scan --json

# Disable colors (for piping)
mac-cleaner scan --no-color
```

### Clean up disk space

```bash
# Interactive cleanup (shows what will be deleted, asks for confirmation)
mac-cleaner clean

# Preview what would be deleted
mac-cleaner clean --dry-run

# Clean specific categories
mac-cleaner clean --category homebrew,trash

# Skip confirmation (use with caution!)
mac-cleaner clean --force
```

### Version info

```bash
mac-cleaner version
```

## Scan Categories

| Category | Name | Risk | What it scans |
|----------|------|------|---------------|
| 🍺 Homebrew | `homebrew` | Safe | `~/Library/Caches/Homebrew` — old formula downloads |
| 🐳 Docker | `docker` | Moderate | Images, containers, volumes, build cache via `docker system df` |
| 🔨 Xcode | `xcode` | Safe | DerivedData, Archives, iOS DeviceSupport, Simulators |
| 📦 Node.js | `node` | Safe | npm/yarn/pnpm caches, abandoned `node_modules` |
| 🐍 Python | `python` | Safe | pip cache, conda packages |
| 🦀 Rust | `rust` | Safe | Cargo registry cache |
| 🐹 Go | `go` | Safe | Module cache (`~/go/pkg/mod/cache`) |
| 💎 Ruby | `ruby` | Safe | Gem cache, Bundler cache |
| ☕ Java/Kotlin | `java` | Moderate | Gradle caches, Maven repository |
| 🗄️ System Caches | `system-caches` | Safe | `~/Library/Caches/*` per-app breakdown |
| 📜 System Logs | `system-logs` | Safe | `~/Library/Logs` |
| 🗑️ Trash | `trash` | Safe | `~/.Trash` |
| 📥 Downloads | `downloads` | Caution | Files in `~/Downloads` older than 30 days |
| 📱 App Leftovers | `app-leftovers` | Caution | Application Support data for uninstalled apps |
| ✉️ Mail | `mail` | Moderate | Cached mail attachments |
| 🎼 Composer | `composer` | Safe | PHP Composer cache |
| 🥥 CocoaPods | `cocoapods` | Safe | CocoaPods spec and download caches |

### Risk Levels

- **🟢 Safe** — Files that are pure caches and will be regenerated automatically. No data loss.
- **🟡 Moderate** — Packages and images that will need to be re-downloaded if needed again.
- **🔴 Caution** — Files that should be reviewed before deletion. May contain data you want to keep.

## Flags Reference

### Global Flags

| Flag | Description |
|------|-------------|
| `--no-color` | Disable colored terminal output |

### `scan` Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--all` | | Scan all categories (default) |
| `--category` | `-c` | Scan specific categories (comma-separated) |
| `--min-size` | | Minimum size to display (e.g., `100MB`, `1GB`) |
| `--json` | | Output results as JSON |

### `clean` Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--dry-run` | | Preview without deleting |
| `--force` | `-f` | Skip confirmation prompts |
| `--category` | `-c` | Clean specific categories (comma-separated) |

## Examples

```bash
# Quick overview of disk usage
$ mac-cleaner

📊 Disk Usage Scan Results
────────────────────────────────────────────────────────────

  Xcode                          4.2 GB [safe]
    ├─ DerivedData                        3.1 GB
    ├─ iOS DeviceSupport                  800 MB
    ├─ Archives                           300 MB

  Docker                         2.8 GB [moderate]
    ├─ Docker Images (reclaimable)        2.1 GB
    ├─ Docker Build Cache (reclaimable)   700 MB

  System Caches                  1.5 GB [safe]
    ├─ com.spotify.client                 400 MB
    ├─ com.apple.Safari                   350 MB
    ...

────────────────────────────────────────────────────────────
  Total reclaimable:              12.3 GB

# Clean up just the safe stuff
$ mac-cleaner clean -c homebrew,trash,system-caches

# Pipe to jq for scripting
$ mac-cleaner scan --json | jq '.[] | select(.total_size > 1073741824)'
```

## Development

```bash
# Run tests
make test

# Run tests with coverage
make test-cover

# Lint
make lint

# Build
make build

# Format code
make fmt
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

## License

MIT — see [LICENSE](LICENSE) for details.
