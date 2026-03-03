# Algorithmic RSS

A personal algorithmic RSS feed system that combines Miniflux with local LLMs for content filtering and rating. Built in Go with a TUI interface for interactive reading and a background service for automated feed management.

## Overview

Algorithmic RSS is designed to create a personalized news feed by:
- Fetching RSS feeds via Miniflux
- Storing and tracking article ratings in PostgreSQL
- Providing a TUI for interactive reading and rating
- Running a background service to manage unread entries with algorithmic filtering

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Algorithmic RSS                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐ │
│  │     CLI     │    │     TUI     │    │       Service       │ │
│  │  (summary)  │    │ (interactive│    │  (background job)   │ │
│  │             │    │   reader)   │    │                     │ │
│  └──────┬──────┘    └──────┬──────┘    └──────────┬──────────┘ │
│         │                  │                      │            │
│         └──────────────────┼──────────────────────┘            │
│                            │                                   │
│                    ┌───────▼────────┐                          │
│                    │    PostgreSQL   │                          │
│                    │   (storage)     │                          │
│                    └───────┬────────┘                          │
│                            │                                   │
│                    ┌───────▼────────┐                          │
│                    │   Miniflux API  │                          │
│                    │   (RSS source)  │                          │
│                    └─────────────────┘                          │
└─────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
.
├── cli/              # CLI tool for generating summary reports
│   ├── main.go       # Entry point, loads config and generates summary
│   └── summary.go    # Summary generation and matrix printing
├── domain/           # Domain models and constants
│   ├── types.go      # Category, Feed, Entry structs
│   └── categories.go # Category ID constants
├── service/          # Background service for feed management
│   └── service.go    # Main service with algorithmic filtering
├── storage/          # Data access layer
│   ├── client.go     # PostgreSQL client connection
│   ├── migrations.go # Database schema migrations
│   ├── tui_repo.go   # Repository for TUI operations
│   └── cli_repo.go   # Repository for CLI operations
├── tui/              # Terminal User Interface
│   ├── main.go       # Entry point and Miniflux sync
│   ├── tui.go        # Bubbletea model and views
│   └── miniflux.go   # Miniflux API client wrapper
├── Makefile          # Build and deployment targets
├── go.mod            # Go module definition
└── go.sum            # Dependency checksums
```

## Domain Model

### Category
```go
type Category struct {
    ID    int64
    Title string
}
```

### Feed
```go
type Feed struct {
    ID         int64
    CategoryID int64
    FeedURL    string
    SiteURL    string
    Title      string
}
```

### Entry
```go
type Entry struct {
    ID      int64
    FeedID  int64
    Title   string
    URL     string
    Content string  // HTML converted to Markdown
}
```

### Categories
| ID | Constant | Description |
|----|----------|-------------|
| 2 | `CatVideo` | Video content (YouTube, ccc.de) |
| 3 | `CatSmallWeb` | Small web/personal blogs |
| 4 | `CatCompany` | Company news |
| 5 | `CatProject` | Project updates |
| 6 | `CatNewsAggregator` | News aggregators |
| 8 | `CatMusic` | Music content |

### Entry Ratings
| Rating | Description |
|--------|-------------|
| `not_opened` | Entry was not opened |
| `only_comments` | Only comments were read |
| `not_finished` | Article was not finished |
| `finished` | Article was fully read |

## Components

### 1. CLI Tool (`cli/`)

Generates summary reports from the database showing entry distribution across categories and ratings.

**Usage:**
```bash
algorithmic-rss-cli -config /path/to/config.toml
```

**Output:** Matrix showing entry counts by category and rating status.

### 2. TUI (`tui/`)

Interactive terminal interface for reading and rating RSS entries.

**Features:**
- Browse unread entries from Miniflux
- Switch between categories (Personal / News Aggregator)
- Rate entries with keyboard shortcuts (1-4)
- Markdown rendering of article content
- Refresh entries with 'r'

**Controls:**
| Key | Action |
|-----|--------|
| `1` | Rate: Not opened |
| `2` | Rate: Only comments |
| `3` | Rate: Not finished |
| `4` | Rate: Finished |
| `←/→` | Switch category |
| `↑/↓` | Navigate entries |
| `r` | Refresh |
| `q` | Quit |

### 3. Background Service (`service/`)

Automated feed management that runs every 10 minutes.

**Algorithm:**
1. Fetches unread entries from specific categories (Video, News Aggregator, Small Web)
2. Applies filtering rules:
   - Skips YouTube Shorts
   - Skips German ccc.de content
   - Applies category-specific timeouts:
     - Videos: 7 days
     - Small Web: 48 hours
     - News Aggregator: 24 hours
3. Keeps 10 random entries unread per category
4. Marks remaining entries as read

## Storage Layer

### Database Schema

```sql
-- Categories table
CREATE TABLE category (
    id INTEGER PRIMARY KEY,
    title TEXT
);

-- Feeds table
CREATE TABLE feed (
    id INTEGER PRIMARY KEY,
    category_id INTEGER REFERENCES category(id),
    site_url TEXT,
    feed_url TEXT,
    title TEXT
);

-- Rating enum
CREATE TYPE rating AS ENUM (
    'not_opened', 'only_comments', 'not_finished', 'finished'
);

-- Entries table
CREATE TABLE entry (
    id INTEGER PRIMARY KEY,
    feed_id INTEGER REFERENCES feed(id),
    updated TIMESTAMP,
    rating rating,
    title TEXT,
    url TEXT,
    content TEXT
);

-- Migration tracking
CREATE TABLE migration (
    id SERIAL PRIMARY KEY,
    query TEXT
);
```

### Repository Pattern

Two repository types share the same PostgreSQL client:
- `TuiRepo`: For TUI operations (categories, feeds, store entries)
- `CliRepo`: For CLI operations (aggregation queries, statistics)

## Configuration

Configuration is loaded from TOML files:

**CLI Config (`~/.config/algorithmicrss/tui.toml`):**
```toml
postgres_hostname = "localhost"
postgres_port = "5432"
postgres_db_name = "algorithmicrss"
postgres_user = "your_user"
postgres_password = "your_password"
```

**TUI Config (`~/.config/algorithmicrss/tui.toml`):**
```toml
postgres_hostname = "localhost"
postgres_port = "5432"
postgres_db_name = "algorithmicrss"
postgres_user = "your_user"
postgres_password = "your_password"
miniflux_hostname = "https://miniflux.example.com"
miniflux_api_key = "your_api_key"
```

**Service Environment Variables:**
```bash
MINIFLUX_HOSTNAME=https://miniflux.example.com
MINIFLUX_API_KEY=your_api_key
```

## Build & Deployment

### Makefile Targets

| Target | Description |
|--------|-------------|
| `cli-build` | Build CLI binary |
| `tui-deploy` | Build TUI and deploy to server |
| `service-deploy` | Build service and deploy as systemd service |

### Building Manually

```bash
# Build CLI
go build -o algorithmic-rss-cli ./cli/...

# Build TUI
go build -o rss-tui ./tui/...

# Build Service
go build -o algorithmic-rss ./service/...
```

### Systemd Service

**Unit file (`/etc/systemd/system/algorithmic-rss.service`):**
```ini
[Unit]
Description=Algorithmic RSS service
After=network-online.target

[Service]
ExecStart=/usr/local/bin/algorithmic-rss
User=algorithmic-rss
Group=algorithmic-rss
Restart=always
RestartSec=3
Environment=MINIFLUX_HOSTNAME=https://miniflux.example.com
Environment=MINIFLUX_API_KEY=your_api_key

[Install]
WantedBy=default.target
```

**Setup commands:**
```bash
# Create user and group
sudo adduser --system --no-create-home --home /nonexistent --shell /sbin/nologin algorithmic-rss
sudo groupadd algorithmic-rss
sudo usermod -aG algorithmic-rss algorithmic-rss

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable algorithmic-rss
sudo systemctl start algorithmic-rss

# View logs
sudo journalctl -f -u algorithmic-rss
```

## Dependencies

| Package | Purpose |
|---------|---------|
| `charmbracelet/bubbletea` | TUI framework |
| `charmbracelet/glamour` | Markdown rendering |
| `miniflux.app/v2/client` | Miniflux API client |
| `lib/pq` | PostgreSQL driver |
| `JohannesKaufmann/html-to-markdown` | HTML to Markdown conversion |
| `BurntSushi/toml` | TOML configuration parsing |

## Development Guidelines

### Adding a New Category

1. Add constant to `domain/categories.go`
2. Update service filtering in `service/service.go` if needed
3. Add category to TUI category switching logic if needed

### Adding a New Rating

1. Update migration in `storage/migrations.go`
2. Update rating mapping in `tui/tui.go` (`rateEntry` function)
3. Update `PrintMatrix` in `cli/summary.go` if needed

### Modifying Filtering Rules

Edit `service/service.go` in the `checkUnread` function:
- Adjust timeout constants at the top of the file
- Modify the `switch category` block for category-specific rules
- Change `KeepEntriesPerCategory` to adjust how many entries stay unread

### Database Migrations

Migrations are defined in `storage/migrations.go` as an ordered slice of SQL statements. The migration system:
1. Tracks executed migrations in the `migration` table
2. Compares existing migrations with defined ones
3. Executes only missing migrations
4. Fails on incompatible migrations (different SQL for same migration index)

## Security Considerations

- API keys and passwords stored in config files - ensure proper file permissions
- Service runs as unprivileged user (`algorithmic-rss`)
- PostgreSQL SSL mode is disabled (adjust for production)

## Troubleshooting

**Service not starting:**
```bash
sudo journalctl -f -u algorithmic-rss
```

**Database connection issues:**
- Verify PostgreSQL is running
- Check config file credentials
- Ensure database exists and migrations have run

**TUI display issues:**
- Ensure terminal supports 256 colors
- Check `TERM` environment variable
- Verify Miniflux API is accessible

## Related Projects

- [Miniflux](https://miniflux.app/) - RSS reader backend
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown renderer