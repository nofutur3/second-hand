#!/bin/bash

echo "=========================================="
echo "Second-Hand Shop Scraper API - Test Script"
echo "=========================================="
echo ""

API_URL="http://localhost:8091/api/v1"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test function
test_endpoint() {
    local name=$1
    local endpoint=$2
    local expected_status=$3

    echo -e "${YELLOW}Testing:${NC} $name"
    echo "Endpoint: $endpoint"

    response=$(curl -s -w "\n%{http_code}" "$endpoint")
    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)

    if [ "$status_code" == "$expected_status" ]; then
        echo -e "${GREEN}✓ Status: $status_code (Expected: $expected_status)${NC}"
        if [ -n "$body" ]; then
            echo "Response:"
            echo "$body" | jq . 2>/dev/null || echo "$body"
        fi
        echo ""
        return 0
    else
        echo -e "${RED}✗ Status: $status_code (Expected: $expected_status)${NC}"
        echo "Response: $body"
        echo ""
        return 1
    fi
}

# Check if API is running
echo "Checking if API is accessible..."
if ! curl -s --connect-timeout 5 "$API_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}✗ API is not accessible at $API_URL${NC}"
    echo ""
    echo "Please start the API first:"
    echo "  Docker: docker-compose up -d api"
    echo "  Local:  ./api"
    exit 1
fi

echo -e "${GREEN}✓ API is accessible${NC}"
echo ""

# Run tests
test_endpoint "Health Check" "$API_URL/health" "200"
test_endpoint "Get All Searches" "$API_URL/searches" "200"

# Try to get products for search ID 1
echo -e "${YELLOW}Note:${NC} The following test might fail if no searches exist yet."
echo "       Run './search -keyword=\"hemingway\"' first to create test data."
echo ""
test_endpoint "Get Products for Search ID 1" "$API_URL/searches/1/products" "200"

# Test invalid search ID
test_endpoint "Get Products for Invalid Search" "$API_URL/searches/99999/products" "404"

echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo ""
echo "API Base URL: $API_URL"
echo ""
echo "Available Endpoints:"
echo "  GET /health                        - Health check"
echo "  GET /searches                      - List all searches"
echo "  GET /searches/{id}/products        - Get products for search"
echo ""
echo "For full API documentation, see openapi.yaml"
