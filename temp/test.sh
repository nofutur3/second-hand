#!/bin/bash

# Test script for secondHand application
# This demonstrates the functionality without requiring live sites

set -e

echo "=== Second Hand Shop Scraper - Test Suite ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test 1: Check if binaries are built
echo "Test 1: Checking binaries..."
if [ -f "bin/search" ] && [ -f "bin/cron" ]; then
    echo -e "${GREEN}✓ Binaries exist${NC}"
else
    echo -e "${RED}✗ Binaries not found. Run: make build${NC}"
    exit 1
fi

# Test 2: Check if PostgreSQL is running
echo ""
echo "Test 2: Checking PostgreSQL..."
if docker ps | grep -q secondhand_postgres; then
    echo -e "${GREEN}✓ PostgreSQL is running${NC}"
else
    echo -e "${RED}✗ PostgreSQL not running. Run: make docker-up${NC}"
    exit 1
fi

# Test 3: Run unit tests
echo ""
echo "Test 3: Running unit tests..."
if go test ./internal/domain ./internal/adapter ./internal/config ./internal/output -v > /tmp/test-output.txt 2>&1; then
    echo -e "${GREEN}✓ Unit tests passed${NC}"
    grep "PASS" /tmp/test-output.txt | tail -5
else
    echo -e "${RED}✗ Unit tests failed${NC}"
    cat /tmp/test-output.txt
    exit 1
fi

# Test 4: Check configuration
echo ""
echo "Test 4: Checking configuration..."
if [ -f "config.json" ] && [ -f ".env" ]; then
    echo -e "${GREEN}✓ Configuration files exist${NC}"
    echo "Shops configured:"
    cat config.json | grep '"url"' | wc -l | xargs echo "  -"
else
    echo -e "${RED}✗ Configuration files missing${NC}"
    exit 1
fi

# Test 5: Database connectivity
echo ""
echo "Test 5: Testing database connectivity..."
if docker exec secondhand_postgres psql -U secondhand -d secondhand -c "SELECT 1;" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Database connection successful${NC}"
else
    echo -e "${RED}✗ Database connection failed${NC}"
    exit 1
fi

# Test 6: Check migrations
echo ""
echo "Test 6: Checking database schema..."
TABLE_COUNT=$(docker exec secondhand_postgres psql -U secondhand -d secondhand -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | tr -d ' ')
if [ "$TABLE_COUNT" -ge "3" ]; then
    echo -e "${GREEN}✓ Database tables created (found $TABLE_COUNT tables)${NC}"
    docker exec secondhand_postgres psql -U secondhand -d secondhand -c "\dt" 2>/dev/null | grep -E "searches|products|search_products" || true
else
    echo -e "${RED}✗ Database schema not initialized${NC}"
fi

echo ""
echo "=== All tests completed ==="
echo ""
echo "To test the search command:"
echo "  ./bin/search -keyword=\"laptop\" -verbose"
echo ""
echo "To test the cron command:"
echo "  ./bin/cron -output=cli -verbose"
echo ""
echo "Note: Actual scraping may fail due to:"
echo "  - Network connectivity"
echo "  - Rate limiting from websites"
echo "  - Changes in website structure"
echo "  - Anti-bot protection"
echo ""
echo "This is expected behavior for a PoC."
