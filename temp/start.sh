#!/bin/bash

set -e  # Exit on error

echo "╔════════════════════════════════════════════════════════════════════╗"
echo "║                                                                    ║"
echo "║         🎉 SECOND-HAND SHOP SCRAPER - QUICK START 🎉             ║"
echo "║                                                                    ║"
echo "╚════════════════════════════════════════════════════════════════════╝"
echo ""
echo "Starting all services..."
echo ""

# Start services
if ! docker compose up -d --build; then
    echo "❌ Error: Failed to start services"
    echo "Please check Docker is running and try again"
    exit 1
fi

echo ""
echo "⏳ Waiting for services to be ready..."
echo ""

# Wait for postgres to be healthy
echo "  • Waiting for PostgreSQL..."
for i in {1..30}; do
    if docker compose exec -T postgres pg_isready -U secondhand > /dev/null 2>&1; then
        echo "    ✓ PostgreSQL is ready"
        break
    fi
    sleep 1
done

# Wait for API to be ready
echo "  • Waiting for API..."
sleep 5
for i in {1..30}; do
    if curl -s http://localhost:8091/api/v1/health > /dev/null 2>&1; then
        echo "    ✓ API is ready"
        break
    fi
    sleep 1
done

# Wait for frontend to be ready
echo "  • Waiting for Frontend..."
sleep 3
for i in {1..30}; do
    if curl -s http://localhost:8092 > /dev/null 2>&1; then
        echo "    ✓ Frontend is ready"
        break
    fi
    sleep 1
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Service Status"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
docker compose ps
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 Running test search for 'hemingway'..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Run search - don't fail if it doesn't work
if docker compose exec -T api ./search -keyword="hemingway" 2>&1; then
    echo ""
    echo "✅ Search completed successfully!"
else
    echo ""
    echo "⚠️  Search command failed. The API might still be starting up."
    echo "   You can run it manually with:"
    echo "   docker compose exec api ./search -keyword=\"hemingway\""
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🌐 Access Your Application"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "  🎨 Frontend:       http://localhost:8092"
echo "  🔌 API:            http://localhost:8091/api/v1"
echo "  🗄️  Database Admin: http://localhost:8099"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "💡 Next Steps"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "  1. Open your browser: http://localhost:8092"
echo "  2. View the search results for 'hemingway'"
echo "  3. Click on a search to see products"
echo "  4. Click on a product to open it on the marketplace"
echo ""
echo "  Run more searches:"
echo "    docker compose exec api ./search -keyword=\"iphone\""
echo "    docker compose exec api ./search -keyword=\"rypadlo\""
echo ""
echo "  View logs:"
echo "    docker compose logs -f frontend"
echo "    docker compose logs -f api"
echo ""
echo "  Stop services:"
echo "    docker compose down"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "✨ Your application is ready! ✨"
echo ""
