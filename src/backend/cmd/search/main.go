package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"secondHand/src/backend/internal/adapter"
	"secondHand/src/backend/internal/config"
	database2 "secondHand/src/backend/internal/database"
	output2 "secondHand/src/backend/internal/output"
	"secondHand/src/backend/internal/service"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Command line flags
	keyword := flag.String("keyword", "", "Search keyword (required)")
	verbose := flag.Bool("verbose", false, "Verbose output")
	outputFormat := flag.String("output", "cli", "Output format: cli, html (cli, html)")
	htmlFile := flag.String("html-file", "results.html", "HTML output file path")
	configFile := flag.String("config", "config.json", "Configuration file path")
	adapterName := flag.String("adapter", "", "Search only specific adapter (e.g., bazos.cz)")
	maxPrice := flag.Float64("max-price", 0, "Good-offer price ceiling for this search (optional; 0 = unset)")
	avgDiscountPct := flag.Float64("avg-discount-pct", 0, "Good-offer trailing-average discount %% for this search (optional; 0 = unset)")
	flag.Parse()

	if *keyword == "" {
		fmt.Println("Error: -keyword flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Run migrations
	if err := database2.Migrate(pool, "migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repository
	repo, err := database2.NewPostgresRepository(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Initialize adapter registry
	adapterRegistry := adapter.NewRegistry(cfg)

	// Initialize services
	searchService := service.NewSearchService(repo, adapterRegistry)

	// Filter adapter if specified
	if *adapterName != "" {
		fmt.Printf("Using only adapter: %s\n", *adapterName)
		// This will be handled by the service layer
	}

	// Perform search
	numAdapters := len(adapterRegistry.GetAllAdapters())
	if *adapterName != "" {
		numAdapters = 1
	}
	fmt.Printf("Searching for '%s' across %d shop(s)...\n", *keyword, numAdapters)
	products, err := searchService.SearchWithFilter(ctx, *keyword, *adapterName)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	// Set good-offer thresholds, if provided
	if *maxPrice != 0 || *avgDiscountPct != 0 {
		search, err := repo.GetSearchByKeyword(ctx, *keyword)
		if err != nil {
			log.Fatalf("Failed to look up search for good-offer config: %v", err)
		}
		var maxPricePtr, avgDiscountPtr *float64
		if *maxPrice != 0 {
			maxPricePtr = maxPrice
		}
		if *avgDiscountPct != 0 {
			avgDiscountPtr = avgDiscountPct
		}
		if err := repo.SetGoodOfferConfig(ctx, search.ID, maxPricePtr, avgDiscountPtr); err != nil {
			log.Fatalf("Failed to set good-offer config: %v", err)
		}
		fmt.Printf("Good-offer config set for '%s' (max_price=%v, avg_discount_pct=%v)\n", *keyword, maxPricePtr, avgDiscountPtr)
	}

	// Format and output results
	switch *outputFormat {
	case "cli":
		formatter := output2.NewCLIFormatter()
		result, err := formatter.FormatProducts(products, *verbose)
		if err != nil {
			log.Fatalf("Failed to format output: %v", err)
		}
		fmt.Print(result)

	case "html":
		formatter := output2.NewHTMLFormatter()
		result, err := formatter.FormatProducts(products, *verbose)
		if err != nil {
			log.Fatalf("Failed to format output: %v", err)
		}

		// Create temp/output directory if it doesn't exist
		os.MkdirAll("temp/output", 0755)

		// Use provided filename or generate with timestamp
		outputFile := *htmlFile
		if outputFile == "results.html" {
			// Default - save to temp/output with timestamp
			timestamp := time.Now().Format("20060102_150405")
			outputFile = fmt.Sprintf("temp/output/search_%s_%s.html", *keyword, timestamp)
		}

		if err := os.WriteFile(outputFile, []byte(result), 0644); err != nil {
			log.Fatalf("Failed to write HTML file: %v", err)
		}
		fmt.Printf("HTML output saved to %s\n", outputFile)

	default:
		log.Fatalf("Unknown output format: %s", *outputFormat)
	}
}
