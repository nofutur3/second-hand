# Snoopy

A Go application that scrapes multiple Czech second-hand marketplaces, stores results in PostgreSQL, and tracks changes over time.

## ⚠️ Important Note About Web Scraping

Real website scraping is challenging due to anti-bot protection, changing HTML structures, and legal considerations. This application includes **mock adapters** that demonstrate full functionality without accessing real sites.

**For Testing/Demo:** Use `config.test.json` with mock adapters  
**For Production:** Real sites require legal permission and site-specific configuration

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

- Docker and Docker Compose — no local Go or Node toolchain needed, everything
  (build, test, run) goes through the Makefile into Docker containers

## Quick Start

1. **Clone and setup:**
   ```bash
   # Start the database, install dependencies
   make setup
   ```

2. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your settings (SMTP for email notifications)
   ```

3. **Test with mock adapters (recommended):**
   ```bash
   docker compose run --rm backend-dev sh -c 'cd src/backend && go run ./cmd/search -config=config/config.test.json -keyword="hemingway"'
   ```

4. **Or search real sites (may fail due to anti-bot protection):**
   ```bash
   make run-search KEYWORD=laptop
   ```

5. **Check for changes:**
   ```bash
   make run-cron VERBOSE=true
   ```

## Usage

### Mock Adapters (Recommended for Testing)

Use mock adapters to demonstrate functionality without web scraping:

```bash
# Search with mock data
docker compose run --rm backend-dev sh -c 'cd src/backend && go run ./cmd/search -config=config/config.test.json -keyword="hemingway"'

# Check for changes
make run-cron VERBOSE=true

# Generate HTML report
docker compose run --rm backend-dev sh -c 'cd src/backend && go run ./cmd/cron -config=config/config.test.json -output=html'
```

### Search Command

Search for products across all configured shops:

```bash
# Basic search
make run-search KEYWORD=laptop

# Verbose output
make run-search KEYWORD=laptop VERBOSE=true
```

### Cron Command

Check saved searches for changes (meant for scheduled execution):

```bash
# CLI output
make run-cron

# Verbose CLI output
make run-cron VERBOSE=true

# HTML output
make run-cron OUTPUT=html

# Email notifications
make run-cron OUTPUT=email
```

In production this runs as a Kubernetes `CronJob` rather than a plain
crontab entry — see "eBay Nintendo-Parts Watcher" below.

## Configuration

### src/backend/config/config.json

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
every 30 minutes); see `k8s/ebay-secret.yaml.example` for the secret it
expects.

## Project Structure

```
second-hand/
├── src/
│   ├── backend/
│   │   ├── cmd/{search,cron,api}/   # CLI commands + HTTP API
│   │   ├── internal/
│   │   │   ├── adapter/             # Shop-specific adapters
│   │   │   ├── config/              # Configuration management
│   │   │   ├── database/            # Database layer
│   │   │   ├── domain/              # Domain models and interfaces
│   │   │   ├── output/              # Output formatters (CLI, HTML, Email, Telegram)
│   │   │   └── service/             # Business logic layer
│   │   ├── migrations/              # Database migrations
│   │   └── config/                  # config.json / config.test.json
│   └── frontend/                    # Nuxt 3 app
├── docker/{api,backend,cron,frontend}/Dockerfile
├── k8s/                             # Kubernetes manifests
├── compose.yaml
└── Makefile                         # Build and run commands (Docker-wrapped)
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
# Start PostgreSQL, API, and frontend
make up

# Stop containers
make down

# Clean (remove volumes)
make docker-clean
```

## Database Schema

### Tables

- **searches**: Saved search queries
- **products**: Product listings from shops
- **search_products**: Many-to-many relationship between searches and products

### Migrations

Migrations live in `src/backend/migrations/` and run automatically on
startup of `search`, `cron`, and `api` — nothing to run by hand.

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
docker compose run --rm backend-dev sh -c 'cd src/backend && go run ./cmd/search -config=config/config.json -keyword=laptop -output=html -html-file=results.html'
```

### Email

HTML email sent via SMTP:

```bash
make run-cron OUTPUT=email
```

## Testing

Run tests with coverage:

```bash
make test
```

## Adding New Shops

1. Create new adapter in `src/backend/internal/adapter/newshop.go`
2. Implement `ShopAdapter` interface
3. Register in `src/backend/internal/adapter/registry.go`
4. Add to `src/backend/config/config.json`

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
