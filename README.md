# BullMQ TUI

A beautiful terminal-based user interface for monitoring and managing [BullMQ](https://docs.bullmq.io/) job queues.

## Features

- 📊 **Real-time Queue Monitoring** - View queue counts and stats updated live
- 🎯 **Job Management** - Browse, view, retry, and delete jobs across all states
- ➕ **Job Creation** - Add new jobs directly from the TUI with JSON input
- 🔄 **Bulk Operations** - Retry all failed jobs, drain queue states, or clean entire queues
- 📋 **Clipboard Integration** - Copy job JSON data with a single keystroke
- 📈 **Visual Analytics** - Sparkline charts showing throughput trends
- ⚡ **Fast Navigation** - Vim-style keybindings for efficient workflow
- 🔄 **Event Streaming** - Real-time updates via Redis pub/sub
- 💾 **Multiple Connections** - Easily manage and switch between Redis instances
- 🎨 **Beautiful Interface** - Clean, colorful TUI powered by Bubbletea

## Installation

### Prerequisites

For clipboard functionality to work, you need:
- **Linux**: Install `xsel`, `xclip`, or `wl-clipboard` (Wayland)
  ```bash
  # Debian/Ubuntu
  sudo apt-get install xsel
  # or
  sudo apt-get install xclip
  # or for Wayland
  sudo apt-get install wl-clipboard

  # Fedora/RHEL
  sudo dnf install xsel
  # or
  sudo dnf install xclip
  # or for Wayland
  sudo dnf install wl-clipboard

  # Arch Linux
  sudo pacman -S xsel
  # or
  sudo pacman -S xclip
  # or for Wayland
  sudo pacman -S wl-clipboard
  ```
- **macOS**: Works out of the box (uses `pbcopy`)
- **Windows**: Works out of the box

### From Source

```bash
go install github.com/AurelienConte/bullmq-tui@latest
```

### Build Locally

```bash
git clone https://github.com/AurelienConte/bullmq-tui
cd bullmq-tui
make build
```

## Quick Start

1. Initialize configuration:

```bash
bullmq-tui config init
```

2. Add your Redis connection:

```bash
bullmq-tui config add production \
  --host redis.example.com \
  --port 6379 \
  --password "${REDIS_PASSWORD}" \
  --tls
```

3. Launch the TUI:

```bash
bullmq-tui
```

## Usage

### Commands

```bash
# Launch TUI with default connection
bullmq-tui

# Launch with specific connection
bullmq-tui connect production
bullmq-tui -c staging

# Configuration management
bullmq-tui config init              # Create default config
bullmq-tui config add <name>        # Add new connection
bullmq-tui config remove <name>     # Remove connection
bullmq-tui config list              # List all connections
bullmq-tui config set-default <name> # Set default connection
bullmq-tui config edit              # Open config in $EDITOR
bullmq-tui config path              # Show config file location

# Other
bullmq-tui version                  # Show version info
```

### Keyboard Shortcuts

#### Navigation
- `↑↓` or `j/k` - Navigate up/down
- `←→` or `h/l` - Navigate left/right or switch tabs
- `tab` / `shift+tab` - Switch between panels
- `1-5` - Jump to job state tabs (Waiting/Active/Delayed/Completed/Failed)

#### Actions
- `a` - Add new job to selected queue
- `enter` - View job details (press `c` to copy JSON)
- `r` - Retry selected job
- `R` - Retry all failed jobs in queue
- `d` - Delete selected job
- `D` - Drain current state (from jobs panel) / Clean all jobs (from queue panel)
- `p` - Pause/resume queue
- `ctrl+r` - Force refresh

#### Other
- `?` - Show help
- `q` / `ctrl+c` - Quit

## Configuration

Config file location (follows XDG Base Directory Specification):
- Linux/macOS: `~/.config/bullmq-tui/config.yaml`
- Windows: `%APPDATA%\bullmq-tui\config.yaml`

### Example Configuration

```yaml
version: 1
default_connection: local

connections:
  local:
    name: Local Development
    host: localhost
    port: 6379
    password: ""
    db: 0
    tls: false
    prefix: "bull"

  production:
    name: Production Redis
    host: redis.example.com
    port: 6380
    password: "${PROD_REDIS_PASSWORD}"  # Environment variable expansion
    db: 0
    tls: true
    tls_skip_verify: false
    prefix: "bull"

settings:
  refresh_interval_ms: 1000      # Refresh rate in milliseconds
  stats_window_minutes: 30       # Stats collection window
  max_jobs_display: 100          # Max jobs to show per state
  theme: default
  date_format: "2006-01-02 15:04:05"
```

### Environment Variables

You can use environment variable expansion in connection passwords:

```yaml
connections:
  prod:
    password: "${REDIS_PASSWORD}"
```

## Screenshots

(Screenshots would go here when available)

## Architecture

BullMQ TUI is built with:
- **Go 1.24+** - Modern, fast, compiled language
- **[Cobra](https://github.com/spf13/cobra)** - CLI framework
- **[Bubbletea](https://github.com/charmbracelet/bubbletea)** - Terminal UI framework
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - Style definitions and layout
- **[Bubbles](https://github.com/charmbracelet/bubbles)** - TUI components
- **[go-redis](https://github.com/redis/go-redis)** - Redis client

### Project Structure

```
bullmq-tui/
├── cmd/                    # CLI commands (Cobra)
├── internal/
│   ├── config/             # Configuration management
│   ├── redis/              # Redis client & BullMQ operations
│   ├── stats/              # Statistics collection
│   └── ui/                 # Bubbletea application
│       └── components/     # UI components
└── main.go
```

## Development

### Prerequisites

- Go 1.24 or higher
- Redis server (for testing)
- Access to a BullMQ instance

### Building

```bash
make build        # Build binary
make install      # Install to $GOPATH/bin
make test         # Run tests
make clean        # Clean build artifacts
```

### Running Locally

```bash
go run . -c local
```

### Branching Strategy

This project uses a two-branch workflow:

- **`main`** - Stable release branch. All releases are tagged from this branch.
- **`develop`** - Development branch. All feature work is merged here first.

#### Development Workflow

1. Create feature branches from `develop`
2. Submit pull requests to `develop`
3. When ready for release, merge `develop` into `main`
4. Tag the release on `main` with `v*` (e.g., `v1.0.0`)

### Release Process

Releases are automated via GitHub Actions:

1. Merge all changes to `develop` branch
2. Merge `develop` into `main`
3. Create and push a version tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
4. GitHub Actions will automatically:
   - Run tests
   - Build binaries for Linux (amd64, arm64), macOS (amd64, arm64), and Windows (amd64)
   - Create a GitHub release with binaries and checksums
   - Generate release notes

### CI/CD

The project uses GitHub Actions for continuous integration:

- **Test Workflow** - Runs on all pushes and PRs to `main` and `develop`
  - Runs tests, formatting checks, and linting
  - Builds the binary to ensure compilation succeeds

- **Release Workflow** - Triggers on version tags (e.g., `v1.0.0`)
  - Builds multi-platform binaries
  - Creates GitHub release with assets

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Credits

- Inspired by [BullMQ](https://docs.bullmq.io/)
- Built with [Charm](https://charm.sh/) tools
- Created by [@aurelg](https://github.com/AurelienConte)

## Support

- 📝 [Report Issues](https://github.com/AurelienConte/bullmq-tui/issues)
- 💬 [Discussions](https://github.com/AurelienConte/bullmq-tui/discussions)
- 📖 [Documentation](https://github.com/AurelienConte/bullmq-tui/wiki)
