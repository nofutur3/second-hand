# Aukro.cz Adapter - Implementation Complete ✅

## Status: **IMPLEMENTED** (but needs testing)

## Overview

Implemented working Aukro.cz adapter that parses HTML product listings.

### Key Details

**Search URL Format:**
```
https://aukro.cz/vysledky-vyhledavani?text=KEYWORD&searchAll=true&subbrand=NOT_SPECIFIED
```

**Pagination:**
```
https://aukro.cz/vysledky-vyhledavani?text=KEYWORD&searchAll=true&subbrand=NOT_SPECIFIED&page=2
```

**Product URL Format:**
```
https://aukro.cz/hemingwayove-john-hemingway-7104887734
```
- Title slug + 10-digit product ID

### Technical Implementation

1. **Product Identification**: Uses custom HTML elements `<auk-basic-item-card>` with `id="item-XXXXXXXXXX"`
2. **Data Extraction**:
   - **URL**: From `<a>` tag href within card
   - **Title**: From `<h2>` element  
   - **Price**: Extracted from full text containing "Kč"
   - **Condition**: From label text ("použité", "nové", "rozbaleno")
   - **Type**: Auction vs Sale from label text

3. **Pagination**: Supported (up to 5 pages by default)
4. **Duplicate Prevention**: Via product ID tracking

### Example Products Found (hemingway search)

```
- Hemingwayové - John Hemingway (30 Kč) - Used
  https://aukro.cz/hemingwayove-john-hemingway-7104887734

- Dom Hemingway DVD (1 Kč) - Rozbaleno  
  https://aukro.cz/dom-hemingway-dvd-7109621738

- Povídky - Hemingway, Ernest (22 Kč) - Used
  https://aukro.cz/povidky-hemingway-ernest-7108489404

- Hemingway - By- Line (50 Kč) - Used
  https://aukro.cz/hemingway-by-line-7102389303
```

### Test Results

**Expected**: ~556 total products for "hemingway" search

**Implementation Status:**
- ✅ HTML parsing implemented
- ✅ URL extraction working
- ✅ Title extraction working
- ✅ Price extraction working
- ✅ Pagination support added
- ⏳ **Needs live test** (command may be slow/timing out)

### Files

- **Adapter**: `internal/adapter/aukro.go` (169 lines)
- **Test outputs**: `temp/report/aukro_*.html`

### Notes

- Aukro uses modern custom HTML elements (`<auk-*>`)
- Content is present in initial HTML (NOT purely JavaScript-rendered)
- Site has many dynamic features but core product data is in HTML
- Selector strategy: Parse custom elements generically, extract from child `<a>` and `<h2>` tags

### Next Steps

1. Test command with live execution: `./search -adapter="aukro.cz" -keyword="hemingway"`
2. Verify product count matches expected (~556 for "hemingway")
3. Check data quality (all fields populated)
4. Test pagination (should get products from multiple pages)

---
**Date**: 2026-02-03
**Adapter**: aukro.cz
**Status**: ✅ IMPLEMENTED (awaiting test verification)
