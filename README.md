# BullMQ TUI

A beautiful terminal-based user interface for monitoring and managing [BullMQ](https://docs.bullmq.io/) job queues.

## Features

- 📊 **Real-time Queue Monitoring** - View queue counts and stats updated live
- 🎯 **Job Management** - Browse, view, retry, and delete jobs across all states
- 📈 **Visual Analytics** - Sparkline charts showing throughput trends
- ⚡ **Fast Navigation** - Vim-style keybindings for efficient workflow
- 🔄 **Event Streaming** - Real-time updates via Redis pub/sub
- 💾 **Multiple Connections** - Easily manage and switch between Redis instances
- 🎨 **Beautiful Interface** - Clean, colorful TUI powered by Bubbletea

## Installation

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
- `enter` - View job details
- `r` - Retry selected job
- `R` - Retry all failed jobs in queue
- `d` - Delete selected job
- `D` - Drain all jobs in current state
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
- **Go 1.22+** - Modern, fast, compiled language
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

- Go 1.22 or higher
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

## Roadmap

### Implemented (MVP)
- ✅ Queue discovery and monitoring
- ✅ Job viewing and navigation
- ✅ Real-time stats with sparklines
- ✅ Job state filtering (waiting, active, delayed, completed, failed)
- ✅ Connection management
- ✅ Basic job actions (retry, delete)

### Planned Features
- ⏳ Job creation from TUI
- ⏳ Queue pause/resume actions
- ⏳ Bulk job operations (retry all, drain queue)
- ⏳ Job payload editing
- ⏳ Export stats to CSV
- ⏳ Custom themes
- ⏳ Flow (parent/child) visualization
- ⏳ Worker log viewing
- ⏳ Repeatable job management
- ⏳ Search and filtering

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
