# Second Hand Shop Scraper

A Go application that scrapes multiple Czech second-hand marketplaces, stores results in PostgreSQL, and tracks changes over time.

## ⚠️ Important Note About Web Scraping

Real website scraping is challenging due to anti-bot protection, changing HTML structures, and legal considerations. This application includes **mock adapters** that demonstrate full functionality without accessing real sites.

**For Testing/Demo:** Use `config.test.json` with mock adapters  
**For Production:** Real sites require legal permission and site-specific configuration (see [TROUBLESHOOTING.md](TROUBLESHOOTING.md))

## Features

- 🔍 Search across multiple second-hand shops (Bazos, Sbazar, Avizo, Inzeruj, Aukro, eBay)
- 💾 Store products in PostgreSQL database
- 📊 Track price changes and new/removed products
- 📧 Multiple output formats: CLI, HTML, and Email
- 📱 eBay Nintendo-parts watcher with Telegram "good offer" alerts (see below)
- ⚙️ Adapter pattern for easy shop integration
- 🐳 Docker support for PostgreSQL
- 🧪 Mock adapters for reliable testing

## Prerequisites

- Go 1.25+
- Docker and Docker Compose
- PostgreSQL (via Docker)

## Quick Start

1. **Clone and setup:**
   ```bash
   # Install dependencies and start database
   make setup
   ```

2. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your settings (SMTP for email notifications)
   ```

3. **Test with mock adapters (recommended):**
   ```bash
   go run ./cmd/search -config=config.test.json -keyword="hemingway"
   ```

4. **Or search real sites (may fail due to anti-bot protection):**
   ```bash
   make run-search KEYWORD=laptop
   ```

5. **Check for changes:**
   ```bash
   go run ./cmd/cron -config=config.test.json -verbose
   ```

## Usage

### Mock Adapters (Recommended for Testing)

Use mock adapters to demonstrate functionality without web scraping:

```bash
# Search with mock data
go run ./cmd/search -config=config.test.json -keyword="hemingway"

# Check for changes
go run ./cmd/cron -config=config.test.json -verbose

# Generate HTML report
go run ./cmd/cron -config=config.test.json -output=html
```

### Search Command

Search for products across all configured shops:

```bash
# Basic search
go run ./cmd/search -keyword="laptop"

# Verbose output
go run ./cmd/search -keyword="laptop" -verbose

# HTML output
go run ./cmd/search -keyword="laptop" -output=html -html-file=results.html

# Using Makefile
make run-search KEYWORD=laptop VERBOSE=true
```

### Cron Command

Check saved searches for changes (meant for scheduled execution):

```bash
# CLI output
go run ./cmd/cron

# Verbose CLI output
go run ./cmd/cron -verbose

# HTML output
go run ./cmd/cron -output=html -html-file=changes.html

# Email notifications
go run ./cmd/cron -output=email

# Using Makefile
make run-cron OUTPUT=html VERBOSE=true
```

### Schedule with Cron

Add to your crontab to check for changes every hour:

```bash
0 * * * * cd /path/to/secondHand && /usr/local/go/bin/go run ./cmd/cron -output=email
```

## Configuration

### config.json

Configure which shops to scrape:

```json
{
  "shops": [
    {
      "url": "https://www.bazos.cz",
      "enabled": true
    }
  ]
}
```

### Environment Variables (.env)

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=secondhand
DB_PASSWORD=secondhand_dev
DB_NAME=secondhand
DB_SSLMODE=disable

# SMTP (for email notifications)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=your-email@gmail.com
SMTP_TO=recipient@example.com

# Scraping
SCRAPE_DELAY_MS=2000
REQUEST_TIMEOUT_SEC=30

# eBay Browse API (OAuth2 client-credentials)
EBAY_CLIENT_ID=
EBAY_CLIENT_SECRET=
EBAY_API_BASE=https://api.ebay.com

# Telegram bot (good-offer notifications)
TELEGRAM_BOT_TOKEN=
TELEGRAM_CHAT_ID=
TELEGRAM_API_BASE=https://api.telegram.org
```

## eBay Nintendo-Parts Watcher

`cmd/cron` also runs an eBay-specific watcher alongside its normal
diffing: for any saved search against the `ebay.com` adapter that has a
`max_price` and/or `avg_discount_pct` threshold configured (via
`cmd/search -max-price=... -avg-discount-pct=...`), new or price-dropped
listings that meet either threshold trigger a Telegram notification via a
bot (`TELEGRAM_BOT_TOKEN`/`TELEGRAM_CHAT_ID` above). This is independent
of the `-output` flag — it always runs, on top of whatever CLI/HTML/email
output is also requested.

In production this runs as a Kubernetes `CronJob` (`k8s/ebay-cronjob.yaml`,
every 30 minutes) rather than the ad-hoc crontab example above; see
`k8s/ebay-secret.yaml.example` for the secret it expects.

## Project Structure

```
secondHand/
├── cmd/
│   ├── search/          # Search CLI command
│   └── cron/            # Cron CLI command
├── internal/
│   ├── adapter/         # Shop-specific adapters
│   ├── config/          # Configuration management
│   ├── database/        # Database layer
│   ├── domain/          # Domain models and interfaces
│   ├── output/          # Output formatters (CLI, HTML, Email)
│   └── service/         # Business logic layer
├── migrations/          # Database migrations
├── config.json          # Shop configuration
├── docker-compose.yml   # Docker configuration
└── Makefile            # Build and run commands
```

## Development

### Build

```bash
make build
```

### Run Tests

```bash
make test
```

### Docker Commands

```bash
# Start PostgreSQL
make docker-up

# Stop PostgreSQL
make docker-down

# Clean (remove volumes)
make docker-clean
```

## Database Schema

### Tables

- **searches**: Saved search queries
- **products**: Product listings from shops
- **search_products**: Many-to-many relationship between searches and products

### Migrations

Migrations are automatically applied when running commands. Manual migration:

```bash
# Migrations are in migrations/ directory
# They run automatically on startup
```

## Output Formats

### CLI

Default format with colored output showing product details:

```
=== Found 5 products ===

1. MacBook Pro 2020
   Shop: bazos.cz
   Price: 25000.00 CZK
   URL: https://www.bazos.cz/...
```

### HTML

Styled HTML report with tables and colors:

```bash
go run ./cmd/search -keyword=laptop -output=html -html-file=results.html
```

### Email

HTML email sent via SMTP:

```bash
go run ./cmd/cron -output=email
```

## Testing

Run tests with coverage:

```bash
go test -v -race -coverprofile=coverage.txt ./...
```

## Adding New Shops

1. Create new adapter in `internal/adapter/newshop.go`
2. Implement `ShopAdapter` interface
3. Register in `internal/adapter/registry.go`
4. Add to `config.json`

Example:

```go
type NewShopAdapter struct {
    *BaseAdapter
}

func (a *NewShopAdapter) Search(ctx context.Context, keyword string) ([]domain.Product, error) {
    // Implementation
}
```

## License

MIT

## Notes

- Respects robots.txt and implements rate limiting
- Default 2-second delay between requests
- Uses connection pooling for database efficiency
- Implements graceful error handling
- Supports concurrent scraping across multiple shops
