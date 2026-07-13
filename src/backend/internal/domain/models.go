package domain

import (
	"time"
)

// Condition represents the condition of a product
type Condition string

const (
	ConditionNew     Condition = "new"
	ConditionUsed    Condition = "used"
	ConditionLikeNew Condition = "like_new"
	ConditionGood    Condition = "good"
	ConditionFair    Condition = "fair"
	ConditionPoor    Condition = "poor"
	ConditionDamaged Condition = "damaged"
	ConditionUnknown Condition = "unknown"
)

// AuctionType represents whether the product is for sale or auction
type AuctionType string

const (
	AuctionTypeSale    AuctionType = "sale"
	AuctionTypeAuction AuctionType = "auction"
)

// Product represents a product listing from a second-hand shop
type Product struct {
	ID          int64
	ShopSource  string
	Title       string
	Description string
	Price       float64
	Currency    string
	AuctionType AuctionType
	EndingTime  *time.Time
	Condition   Condition
	URL         string
	ImageURL    string
	Location    string
	SellerName  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Search represents a saved search query
type Search struct {
	ID            int64
	Keyword       string
	CreatedAt     time.Time
	LastCheckedAt *time.Time
}

// SearchProduct represents the many-to-many relationship between searches and products
type SearchProduct struct {
	SearchID  int64
	ProductID int64
	FoundAt   time.Time
	IsNew     bool
}

// ProductDiff represents changes in a product
type ProductDiff struct {
	Product      Product
	DiffType     DiffType
	OldPrice     *float64
	NewPrice     *float64
	PriceChanged bool
}

// DiffType represents the type of change
type DiffType string

const (
	DiffTypeNew       DiffType = "new"
	DiffTypeRemoved   DiffType = "removed"
	DiffTypePriceUp   DiffType = "price_up"
	DiffTypePriceDown DiffType = "price_down"
	DiffTypeUnchanged DiffType = "unchanged"
)
