#!/bin/bash
cd /Users/jakub.vyvazil/Projects/personal/secondHand
echo "=== Testing Aukro Adapter ==="
go run ./cmd/search -adapter="aukro.cz" -keyword="hemingway" 2>&1 | tee temp/output/aukro_run.log
echo ""
echo "=== Test Complete ==="
echo "Output saved to: temp/output/aukro_run.log"
ls -lh temp/output/aukro_run.log 2>/dev/null
