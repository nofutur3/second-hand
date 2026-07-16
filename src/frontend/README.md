# Snoopy - Frontend

A modern Nuxt.js frontend application for browsing search results from Czech second-hand marketplaces.

## 🚀 Features

- 📋 **Browse Searches** - View all performed searches
- 🔍 **Search Details** - Click on a search to see all found products
- 🎨 **Modern UI** - Clean, responsive design with gradient styling
- 🔗 **Direct Links** - Click products to open them in the original marketplace
- 📊 **Product Details** - Price, condition, location, shop source
- 🐳 **Docker Ready** - Runs in Docker container

## 🏗️ Technology Stack

- **Framework**: Nuxt.js 3
- **Language**: Vue.js 3 (Composition API)
- **Styling**: Pure CSS with gradients
- **Data Fetching**: useFetch composable
- **Port**: 8092

## 📦 Quick Start

### Option 1: Docker (Recommended)

```bash
# From project root
docker-compose up -d --build

# Frontend will be available at:
# http://localhost:8092
```

### Option 2: Local Development

```bash
# Install dependencies
cd frontend
npm install

# Start development server
npm run dev

# Open http://localhost:3000
```

## 🌐 API Configuration

The frontend connects to the API at:
- **Default**: `http://localhost:8091/api/v1`
- **Configurable**: Set `NUXT_PUBLIC_API_BASE` environment variable

## 📱 Pages

### Home Page (`/`)
- Lists all searches
- Displays search keyword and creation date
- Click on a search card to view products

### Search Detail Page (`/search/:id`)
- Shows search information
- Lists all products found for the search
- Each product shows:
  - Title with link to original listing
  - Price and currency
  - Shop source (bazos.cz, sbazar.cz, etc.)
  - Condition (new, used, refurbished)
  - Type (auction or sale)
  - Location
  - Description (if available)

## 🎨 UI Features

- **Responsive Design** - Works on mobile, tablet, and desktop
- **Gradient Headers** - Beautiful purple gradient design
- **Hover Effects** - Cards lift and change shadow on hover
- **Loading States** - Shows loading indicator while fetching data
- **Error Handling** - Displays friendly error messages
- **Empty States** - Helpful messages when no data is available

## 🛠️ Development

### Project Structure

```
frontend/
├── assets/
│   └── css/
│       └── main.css          # Global styles
├── pages/
│   ├── index.vue             # Home page (search list)
│   └── search/
│       └── [id].vue          # Search detail page
├── app.vue                   # Root component
├── nuxt.config.ts            # Nuxt configuration
├── package.json              # Dependencies
├── Dockerfile                # Docker build
└── README.md                 # This file
```

### Available Scripts

```bash
# Development
npm run dev              # Start dev server (port 3000)

# Production
npm run build            # Build for production
npm run preview          # Preview production build
npm start                # Start production server

# Generate static site
npm run generate         # Generate static site
```

## 🐳 Docker

### Build Image

```bash
docker build -t secondhand-frontend .
```

### Run Container

```bash
docker run -p 8092:8092 \
  -e NUXT_PUBLIC_API_BASE=http://localhost:8091/api/v1 \
  secondhand-frontend
```

### Docker Compose

The frontend is included in the main `docker-compose.yml`:

```yaml
frontend:
  build: ./frontend
  ports:
    - "8092:8092"
  environment:
    NUXT_PUBLIC_API_BASE: http://localhost:8091/api/v1
```

## 🔧 Configuration

### Environment Variables

- `NUXT_PUBLIC_API_BASE` - API base URL (default: `http://localhost:8091/api/v1`)
- `NUXT_HOST` - Host to bind (default: `0.0.0.0`)
- `NUXT_PORT` - Port to run on (default: `8092`)

### nuxt.config.ts

```typescript
export default defineNuxtConfig({
  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8091/api/v1'
    }
  }
})
```

## 📊 API Endpoints Used

| Endpoint | Usage |
|----------|-------|
| `GET /searches` | Fetch all searches for home page |
| `GET /searches/{id}/products` | Fetch products for search detail page |

## 🎯 User Flow

1. **Visit Homepage** → See list of all searches
2. **Click on Search** → Navigate to search detail page
3. **View Products** → Browse all found products
4. **Click Product** → Open in original marketplace (new tab)
5. **Go Back** → Return to search list

## 🎨 Design System

### Colors

- **Primary**: Purple gradient (`#667eea` → `#764ba2`)
- **Background**: Light gray (`#f5f5f5`)
- **Cards**: White with shadows
- **Text**: Dark gray (`#333`)
- **Links**: Purple (`#667eea`)

### Badges

- **Shop**: Blue (`#e3f2fd`)
- **Condition**: Purple/Orange/Green (depends on condition)
- **Type**: Pink (`#fce4ec`)
- **Location**: Teal (`#e0f2f1`)

## 🔍 Example Screenshots

### Home Page
```
┌─────────────────────────────────────┐
│  🔍 Second-Hand Shop Scraper       │
│  Browse products from Czech shops  │
└─────────────────────────────────────┘

Found 3 searches

┌──────────────┐ ┌──────────────┐
│ hemingway    │ │ rypadlo      │
│              │ │              │
│ Feb 3, 2026  │ │ Feb 3, 2026  │
│ View →       │ │ View →       │
└──────────────┘ └──────────────┘
```

### Search Detail Page
```
← Back to Searches

┌─────────────────────────────────────┐
│ hemingway                           │
│ Created: Feb 3, 2026                │
│ Total Products: 29                  │
└─────────────────────────────────────┘

Products (29 found)

┌─────────────────────────────────────┐
│ Hemingwayové - John Hemingway       │
│ 30 CZK                              │
│                                     │
│ Kniha v dobrém stavu               │
│                                     │
│ 🏪 bazos.cz  ♻️ Used  💰 Sale      │
│ 📍 Praha                            │
└─────────────────────────────────────┘
```

## 🚦 Status Indicators

- **Loading**: Animated "Loading..." text
- **Error**: Red error box with message
- **Empty**: Friendly message with instructions
- **Success**: List of items with full details

## 🔗 Links

- **Frontend**: http://localhost:8092
- **API**: http://localhost:8091/api/v1
- **Database Admin**: http://localhost:8099

## 📄 License

MIT License

## 🤝 Contributing

1. Make changes in `frontend/` directory
2. Test locally with `npm run dev`
3. Build with `npm run build`
4. Test in Docker with `docker-compose up --build frontend`

## 💡 Tips

- Run searches first to have data to display
- Use `docker-compose exec api ./search -keyword="test"` to add data
- Check browser console for API errors
- Make sure API is running before starting frontend

---

**Port**: 8092  
**Framework**: Nuxt.js 3  
**Status**: ✅ Production Ready
