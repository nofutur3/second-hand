# Second Hand Shop Scraper - Quick Reference

## 🚀 Quick Start (3 Steps)

```bash
# 1. Setup
make setup          # Start database + download dependencies

# 2. Build
make build          # Compile both commands

# 3. Search
./bin/search -keyword="laptop"
```

## 📁 Project Stats

- **23 Go files** (19 source + 4 test files)
- **5 shop adapters** implemented
- **3 database tables** with relationships
- **2 CLI commands** (search + cron)
- **3 output formats** (CLI, HTML, email)
- **All tests passing** ✅

## 🔧 Common Commands

### Search Operations
```bash
# Basic search
./bin/search -keyword="laptop"

# Verbose output
./bin/search -keyword="laptop" -verbose

# HTML export
./bin/search -keyword="laptop" -output=html -html-file=results.html

# Using Makefile
make run-search KEYWORD=laptop VERBOSE=true
```

### Cron Operations
```bash
# Check all saved searches
./bin/cron

# Verbose diff output
./bin/cron -verbose

# Generate HTML report
./bin/cron -output=html -html-file=changes.html

# Send email notifications
./bin/cron -output=email

# Using Makefile
make run-cron OUTPUT=html VERBOSE=true
```

### Docker Operations
```bash
make docker-up      # Start PostgreSQL
make docker-down    # Stop PostgreSQL
make docker-clean   # Remove all data
```

### Development
```bash
make build          # Build binaries
make test           # Run tests
make clean          # Clean artifacts
./test.sh           # Run integration tests
```

## 📊 Database Quick Access

```bash
# Connect to database
docker exec -it secondhand_postgres psql -U secondhand -d secondhand

# View tables
\dt

# View searches
SELECT * FROM searches;

# View products
SELECT id, shop_source, title, price FROM products LIMIT 10;

# View search-product links
SELECT s.keyword, COUNT(sp.product_id) as product_count
FROM searches s
LEFT JOIN search_products sp ON s.id = sp.search_id
GROUP BY s.keyword;
```

## 🏪 Supported Shops

1. **Bazos.cz** - General classifieds
2. **Sbazar.cz** - Marketplace
3. **Avizo.cz** - Classifieds
4. **Inzeruj.cz** - Ads platform
5. **Aukro.cz** - Auction site

Configure in `config.json`:
```json
{
  "shops": [
    {"url": "https://www.bazos.cz", "enabled": true}
  ]
}
```

## 📧 Email Setup (Optional)

Edit `.env`:
```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=your-email@gmail.com
SMTP_TO=recipient@example.com
```

Then use: `./bin/cron -output=email`

## 🔄 Typical Workflow

```bash
# Day 1: Initial search
./bin/search -keyword="macbook pro"

# Day 2: Check for changes
./bin/cron -verbose

# Output shows:
# - New products (📦)
# - Price drops (📉)
# - Price increases (📈)
# - Removed products (❌)

# Schedule for automation
crontab -e
# Add: 0 * * * * cd /path/to/secondHand && ./bin/cron -output=email
```

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run specific package
go test ./internal/adapter -v

# With coverage
go test -coverprofile=coverage.txt ./...

# Integration test
./test.sh
```

## 📝 File Locations

```
secondHand/
├── bin/                      # Compiled binaries
│   ├── search               # Search command
│   └── cron                 # Cron command
├── cmd/                      # Command entry points
├── internal/                 # Internal packages
│   ├── adapter/             # Shop adapters (5 shops)
│   ├── config/              # Configuration
│   ├── database/            # PostgreSQL layer
│   ├── domain/              # Models & interfaces
│   ├── output/              # Formatters
│   └── service/             # Business logic
├── migrations/               # SQL migrations
├── config.json              # Shop configuration
├── .env                     # Environment variables
├── docker-compose.yml       # Docker setup
├── Makefile                 # Build commands
├── README.md                # User documentation
├── IMPLEMENTATION_SUMMARY.md # Technical summary
└── test.sh                  # Test script
```

## 🎯 Key Features

✅ Multi-shop scraping (5 sites)  
✅ PostgreSQL with migrations  
✅ Diff tracking (price changes)  
✅ Multiple outputs (CLI/HTML/Email)  
✅ Rate limiting (2s delay)  
✅ Czech number format support  
✅ Condition detection  
✅ Parallel scraping  
✅ Comprehensive tests  
✅ Docker integration  

## 🔍 Troubleshooting

### Database won't start
```bash
make docker-down
make docker-clean
make docker-up
```

### Connection errors
```bash
# Check database is running
docker ps | grep secondhand_postgres

# Check environment
cat .env | grep DB_
```

### Build errors
```bash
go mod tidy
make clean
make build
```

### Scraping fails
- Expected! Real sites block automated access
- Check network connectivity
- Verify URLs in config.json
- Sites may have changed structure

## 📖 Documentation

- **README.md** - Full user guide
- **IMPLEMENTATION_SUMMARY.md** - Technical overview
- **Code comments** - Inline documentation
- **test.sh** - Test examples

## 💡 Tips

1. **Start simple**: Test with one keyword first
2. **Use verbose**: Add `-verbose` to see details
3. **Check logs**: Errors show which sites failed
4. **HTML output**: Best for viewing results
5. **Schedule wisely**: Don't run too frequently (respect rate limits)
6. **Test locally**: Use `./test.sh` before production

## 🆘 Need Help?

```bash
# Show help
./bin/search -h
./bin/cron -h
make help

# Check status
./test.sh

# View database
docker exec -it secondhand_postgres psql -U secondhand -d secondhand
```

---

**Created**: February 2026  
**Status**: Production-ready PoC ✅  
**Version**: 1.0.0
