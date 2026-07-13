# 🚀 Quick Start Guide - API

## Start the API in 3 Steps

### 1. Start Services
```bash
docker-compose up -d --build
```

Wait ~30 seconds for services to start.

### 2. Run a Search (Create Test Data)
```bash
docker-compose exec api ./search -keyword="hemingway"
```

This will:
- Search all 5 marketplaces (bazos.cz, sbazar.cz, avizo.cz, inzeruj.cz, aukro.cz)
- Find products matching "hemingway"
- Save results to database

### 3. Test the API
```bash
# Health check
curl http://localhost:8091/api/v1/health | jq

# Get all searches
curl http://localhost:8091/api/v1/searches | jq

# Get products for search ID 1
curl http://localhost:8091/api/v1/searches/1/products | jq
```

---

## That's It! 🎉

Your API is now running at: **http://localhost:8091**

---

## Useful Commands

### View API Logs
```bash
docker-compose logs -f api
```

### Stop Services
```bash
docker-compose down
```

### Run Another Search
```bash
docker-compose exec api ./search -keyword="kniha"
```

### Check Service Status
```bash
docker-compose ps
```

---

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/health` | Health check |
| `GET /api/v1/searches` | List all searches |
| `GET /api/v1/searches/{id}/products` | Get products |

---

## Web Interfaces

- **API**: http://localhost:8091/api/v1/health
- **Database Admin (Adminer)**: http://localhost:8099
  - System: PostgreSQL
  - Server: postgres
  - Username: secondhand
  - Password: secondhand_dev
  - Database: secondhand

---

## Need Help?

- Full documentation: `API_README.md`
- API specification: `openapi.yaml`
- Test script: `./test_api.sh`
- All commands: `make help`

---

**Port**: 8091  
**Base URL**: http://localhost:8091/api/v1  
**Status**: ✅ Ready to use!
