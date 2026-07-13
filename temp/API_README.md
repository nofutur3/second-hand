# Second-Hand Shop Scraper API

REST API for accessing search history and product results from Czech second-hand marketplaces.

## Quick Start with Docker

### Start the API and Database

```bash
# Build and start all services
docker-compose up -d --build

# Check if services are running
docker-compose ps

# View logs
docker-compose logs -f api
```

The API will be available at: **http://localhost:8091**

### Stop the Services

```bash
docker-compose down

# To also remove volumes (database data)
docker-compose down -v
```

## API Endpoints

### Base URL
```
http://localhost:8091/api/v1
```

### 1. Health Check
Check if the API is running.

**Endpoint:** `GET /health`

**Example:**
```bash
curl http://localhost:8091/api/v1/health
```

**Response:**
```json
{
  "status": "ok",
  "time": "2026-02-03T10:30:00Z"
}
```

---

### 2. Get All Searches
Retrieve all search queries.

**Endpoint:** `GET /searches`

**Example:**
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
  },
  {
    "id": 2,
    "keyword": "rypadlo",
    "created_at": "2026-02-03T11:00:00Z",
    "updated_at": "2026-02-03T11:00:00Z"
  }
]
```

---

### 3. Get Products for a Search
Retrieve all products found for a specific search.

**Endpoint:** `GET /searches/{searchId}/products`

**Parameters:**
- `searchId` (path, required) - ID of the search

**Example:**
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
      "description": "Kniha v dobrém stavu",
      "price": 30.0,
      "currency": "CZK",
      "url": "https://www.bazos.cz/inzerat/214456837/hemingway.php",
      "image_url": "",
      "location": "Praha",
      "shop_source": "bazos.cz",
      "auction_type": "sale",
      "condition": "used",
      "ending_time": null,
      "created_at": "2026-02-03T10:00:00Z",
      "updated_at": "2026-02-03T10:00:00Z"
    }
  ],
  "total": 1
}
```

---

## Running Searches

Before you can use the API, you need to perform some searches:

```bash
# Run a search for "hemingway" (using Docker)
docker-compose exec api ./search -keyword="hemingway"

# Or run locally (requires Go and database connection)
./search -keyword="hemingway"
```

## API Documentation

Full OpenAPI 3.0 specification is available in `openapi.yaml`.

You can view it using:
- [Swagger Editor](https://editor.swagger.io/) - paste the content of `openapi.yaml`
- [Redoc](https://github.com/Redocly/redoc) - for beautiful API documentation

## Environment Variables

Configure the API using environment variables or `.env` file:

```env
# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_USER=secondhand
DB_PASSWORD=secondhand_dev
DB_NAME=secondhand
DB_SSLMODE=disable

# API Configuration
API_PORT=8091
```

## Docker Services

The `docker-compose.yml` includes:

1. **postgres** - PostgreSQL 16 database
   - Port: 5432
   - User: secondhand
   - Password: secondhand_dev
   - Database: secondhand

2. **api** - Second-Hand Scraper API
   - Port: 8091
   - Depends on: postgres
   - Auto-runs migrations on startup

3. **adminer** - Database administration tool
   - Port: 8099
   - Access: http://localhost:8099

## Development

### Build Locally

```bash
# Install dependencies
go mod download

# Build the API
go build ./cmd/api

# Run the API
./api
```

### Testing the API

```bash
# Health check
curl http://localhost:8091/api/v1/health

# Get all searches
curl http://localhost:8091/api/v1/searches | jq

# Get products for search ID 1
curl http://localhost:8091/api/v1/searches/1/products | jq
```

### Pretty Print JSON with jq

```bash
# Install jq (if not already installed)
brew install jq  # macOS
apt-get install jq  # Ubuntu/Debian

# Use jq to format JSON responses
curl http://localhost:8091/api/v1/searches | jq .
```

## Error Responses

The API returns JSON error responses:

```json
{
  "error": "Search not found",
  "message": "No search found with ID: 999"
}
```

HTTP Status Codes:
- `200` - Success
- `400` - Bad Request (invalid parameters)
- `404` - Not Found
- `500` - Internal Server Error

## Database Schema

### Tables

**searches**
- `id` - Primary key
- `keyword` - Search keyword
- `created_at` - When search was created
- `last_checked_at` - When search was last executed

**products**
- `id` - Primary key
- `shop_source` - Source marketplace (bazos.cz, sbazar.cz, etc.)
- `title` - Product title
- `description` - Product description
- `price` - Price
- `currency` - Currency (CZK)
- `url` - Product URL
- `image_url` - Image URL
- `location` - Location
- `auction_type` - Type (auction/sale)
- `condition` - Condition (new/used/refurbished/unknown)
- `ending_time` - Auction end time (nullable)
- `created_at` - When product was first found
- `updated_at` - When product was last updated

**search_products** (junction table)
- `search_id` - Foreign key to searches
- `product_id` - Foreign key to products
- `found_at` - When product was found for this search
- `is_new` - Whether this is a new product

## Supported Marketplaces

- **Bazos.cz** - General marketplace
- **Sbazar.cz** - Second-hand marketplace
- **Avizo.cz** - Classified ads
- **Inzeruj.cz** - Advertisements
- **Aukro.cz** - Auction site

## License

MIT License

## Support

For issues and questions, please open an issue on GitHub.
