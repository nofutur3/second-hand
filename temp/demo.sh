#!/bin/bash

# Demo script using mock adapters to show functionality

echo "=== Second Hand Shop Scraper - Functional Demo ===="
echo ""
echo "Using mock adapters to demonstrate functionality..."
echo "(Real sites may block automated access)"
echo ""

# Build if needed
if [ ! -f "bin/search" ]; then
    echo "Building..."
    make build
fi

# Use test config with mock adapters
export CONFIG_FILE="config.test.json"

echo "1. First search for 'hemingway'..."
./bin/search -keyword="hemingway" 2>&1 | grep -A 20 "Found"

echo ""
echo "2. Search for 'laptop'..."
./bin/search -keyword="laptop" 2>&1 | grep -A 20 "Found"

echo ""
echo "3. Checking for changes (should show new products)..."
./bin/cron -verbose 2>&1 | grep -A 30 "Changes for"

echo ""
echo "=== Demo Complete ===="
echo ""
echo "This demonstrates:"
echo "  ✓ Searching across multiple shops"
echo "  ✓ Saving to database"
echo "  ✓ Detecting new products"
echo "  ✓ Tracking searches"
echo ""
echo "Note: Real sites may block scrapers. Mock adapters ensure"
echo "the application logic works correctly."
