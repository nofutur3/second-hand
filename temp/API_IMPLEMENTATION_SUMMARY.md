# Second-Hand Shop Scraper - API Implementation Summary

## 🎉 Implementation Complete!

A REST API has been successfully implemented for the Second-Hand Shop Scraper project.

---

## 📋 What Was Created

### 1. API Server (`cmd/api/main.go`)
- REST API with 3 endpoints
- Built with Gorilla Mux router
- CORS enabled for cross-origin requests
- Proper error handling and JSON responses
- Auto-runs database migrations on startup

### 2. OpenAPI Specification (`openapi.yaml`)
- Full OpenAPI 3.0 documentation
- Complete schema definitions
- Request/response examples
- Error response formats

### 3. Docker Configuration
- **Dockerfile.api** - Multi-stage Docker build for API
- **docker-compose.yml** - Updated with API service
  - PostgreSQL 16 database
  - API server (port 8091)
  - Adminer database UI (port 8099)

### 4. Documentation
- **API_README.md** - Complete API usage guide
- **test_api.sh** - Automated API testing script
- **Makefile** - Updated with API commands

---

## 🔌 API Endpoints

### Base URL: `http://localhost:8091/api/v1`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/searches` | Get all searches |
| GET | `/searches/{id}/products` | Get products for a search |

---

## 🚀 Quick Start

### Option 1: Docker (Recommended)

```bash
# Start all services (database + API)
docker-compose up -d --build

# Check if services are running
docker-compose ps

# View API logs
docker-compose logs -f api

# Test the API
./test_api.sh
```

The API will be available at: **http://localhost:8091**

### Option 2: Local Development

```bash
# Build the API
make build-api

# Run the API (requires database running)
make run-api

# Or use Go directly
go run ./cmd/api
```

---

## 📡 Example API Calls

### Health Check
```bash
curl http://localhost:8091/api/v1/health
```

**Response:**
```json
{
  "status": "ok",
  "time": "2026-02-03T12:00:00Z"
}
```

### Get All Searches
```bash
curl http://localhost:8091/api/v1/searches
```

**Response:**
```json
[
  {
    "id": 1,
    "keyword": "hemingway",
    "created_at": "2026-02-03T10:00:00Z",
    "updated_at": "2026-02-03T10:00:00Z"
  }
]
```

### Get Products for Search
```bash
curl http://localhost:8091/api/v1/searches/1/products
```

**Response:**
```json
{
  "search": {
    "id": 1,
    "keyword": "hemingway",
    "created_at": "2026-02-03T10:00:00Z",
    "updated_at": "2026-02-03T10:00:00Z"
  },
  "products": [
    {
      "id": 1,
      "title": "Hemingwayové - John Hemingway",
      "description": "Kniha",
      "price": 30.0,
      "currency": "CZK",
      "url": "https://www.bazos.cz/inzerat/214456837/hemingway.php",
      "shop_source": "bazos.cz",
      "auction_type": "sale",
      "condition": "used",
      "created_at": "2026-02-03T10:00:00Z",
      "updated_at": "2026-02-03T10:00:00Z"
    }
  ],
  "total": 1
}
```

---

## 🧪 Testing

### Automated Testing
```bash
# Test all API endpoints
./test_api.sh

# Or use Make
make test-api
```

### Manual Testing with curl
```bash
# Health check
curl http://localhost:8091/api/v1/health | jq

# Get searches
curl http://localhost:8091/api/v1/searches | jq

# Get products for search ID 1
curl http://localhost:8091/api/v1/searches/1/products | jq
```

---

## 📦 Docker Services

### Services Overview

| Service | Port | Description |
|---------|------|-------------|
| **postgres** | 5432 | PostgreSQL 16 database |
| **api** | 8091 | REST API server |
| **adminer** | 8099 | Database admin UI |

### Docker Commands

```bash
# Start all services
docker-compose up -d

# Stop all services
docker-compose down

# View logs
docker-compose logs -f api

# Rebuild and restart
docker-compose up -d --build

# Check service status
docker-compose ps

# Access API container shell
docker-compose exec api sh

# Run search in Docker
docker-compose exec api ./search -keyword="hemingway"
```

---

## 🗄️ Database Integration

The API automatically:
- ✅ Connects to PostgreSQL database
- ✅ Runs migrations on startup
- ✅ Uses connection pooling
- ✅ Handles transactions properly

**Database Schema:**
- `searches` - Search history
- `products` - Product details
- `search_products` - Many-to-many relationship

---

## 🔧 Configuration

### Environment Variables

Create a `.env` file or set environment variables:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=secondhand
DB_PASSWORD=secondhand_dev
DB_NAME=secondhand
DB_SSLMODE=disable

# API
API_PORT=8091
```

### Docker Compose Variables

The `docker-compose.yml` uses these defaults:
- Database: `postgres:5432`
- API: `0.0.0.0:8091`
- Auto-restart: Yes
- Health checks: Enabled

---

## 📝 Data Flow

1. **Search** → Run `./search -keyword="hemingway"`
   - Searches all configured shops
   - Saves results to database
   - Links products to search

2. **API** → Query via REST API
   - GET `/searches` → List all searches
   - GET `/searches/{id}/products` → Get specific results

3. **CRON** → Periodic updates (to be implemented)
   - Re-run searches
   - Compare results
   - Send notifications

---

## 🛠️ Make Commands

```bash
# Build
make build          # Build everything
make build-api      # Build API only

# Run
make run-api        # Run API locally
make run-search     # Run search (KEYWORD=xxx)

# Test
make test           # Run Go tests
make test-api       # Test API endpoints

# Docker
make docker-up      # Start services
make docker-down    # Stop services

# Help
make help           # Show all commands
```

---

## 📊 API Response Formats

### Success Response
```json
{
  "search": {...},
  "products": [...],
  "total": 42
}
```

### Error Response
```json
{
  "error": "Search not found",
  "message": "No search found with ID: 999"
}
```

### HTTP Status Codes
- `200` - Success
- `400` - Bad Request
- `404` - Not Found
- `500` - Internal Server Error

---

## 🔐 Security Features

- ✅ CORS enabled (configurable)
- ✅ Input validation (search ID must be integer)
- ✅ SQL injection protection (using parameterized queries)
- ✅ Error messages don't leak sensitive info
- ✅ Database connection pooling with limits

---

## 📈 Performance

### API Response Times (Expected)
- Health check: < 5ms
- List searches: < 50ms (up to 1000 searches)
- Get products: < 100ms (up to 1000 products)

### Optimizations
- ✅ Connection pooling
- ✅ Indexed database queries
- ✅ Efficient JSON serialization
- ✅ No N+1 queries

---

## 📚 Additional Resources

### Files Created
- `cmd/api/main.go` - API server implementation (230 lines)
- `openapi.yaml` - OpenAPI 3.0 specification (300+ lines)
- `Dockerfile.api` - Multi-stage Docker build
- `docker-compose.yml` - Service orchestration (updated)
- `API_README.md` - Detailed API documentation
- `test_api.sh` - Automated test script
- `Makefile` - Build and run commands (updated)

### Dependencies Added
- `github.com/gorilla/mux` - HTTP router
- `github.com/rs/cors` - CORS middleware

---

## ✅ Implementation Checklist

- [x] API server with 3 endpoints
- [x] Health check endpoint
- [x] Get all searches endpoint
- [x] Get products by search ID endpoint
- [x] OpenAPI 3.0 specification
- [x] Docker support (Dockerfile)
- [x] Docker Compose integration
- [x] CORS enabled
- [x] Error handling
- [x] JSON responses
- [x] Documentation (API_README.md)
- [x] Test script (test_api.sh)
- [x] Make commands
- [x] Environment variable support
- [x] Database migrations on startup
- [x] Port 8091 configuration

---

## 🎯 Next Steps

The API is ready for use! You can now:

1. ✅ **Run searches** via CLI → Data stored in database
2. ✅ **Query results** via API → JSON responses
3. ⏳ **Implement CRON** → Periodic updates & diff detection
4. ⏳ **Add email notifications** → Send updates via email
5. ⏳ **Create web UI** → Frontend to consume API

---

## 🐛 Troubleshooting

### API not starting?
```bash
# Check if database is running
docker-compose ps

# Check API logs
docker-compose logs api

# Check if port 8091 is available
lsof -i :8091
```

### Database connection failed?
```bash
# Restart database
docker-compose restart postgres

# Wait for database to be ready
docker-compose exec postgres pg_isready -U secondhand
```

### No search results?
```bash
# Run a search first
./search -keyword="hemingway"

# Or in Docker
docker-compose exec api ./search -keyword="hemingway"
```

---

## 📞 Support

For issues or questions:
1. Check `API_README.md` for detailed documentation
2. Review `openapi.yaml` for API specification
3. Run `./test_api.sh` to verify setup
4. Check logs: `docker-compose logs -f api`

---

**Status**: ✅ **API Implementation Complete**

**Date**: February 3, 2026

**Port**: 8091

**Base URL**: http://localhost:8091/api/v1

**Ready for**: Production use, frontend integration, CRON implementation
