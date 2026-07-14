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
	domain2 "secondHand/src/backend/internal/domain"
	output2 "secondHand/src/backend/internal/output"
	service2 "secondHand/src/backend/internal/service"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Command line flags
	verbose := flag.Bool("verbose", false, "Verbose output")
	outputFormat := flag.String("output", "cli", "Output format: cli, html, email")
	htmlFile := flag.String("html-file", "diff-results.html", "HTML output file path")
	configFile := flag.String("config", "config.json", "Configuration file path")
	flag.Parse()

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
	searchService := service2.NewSearchService(repo, adapterRegistry)
	diffService := service2.NewDiffService(repo)

	// Get all searches
	searches, err := repo.GetAllSearches(ctx)
	if err != nil {
		log.Fatalf("Failed to get searches: %v", err)
	}

	if len(searches) == 0 {
		fmt.Println("No saved searches found. Use 'search' command first to create searches.")
		return
	}

	fmt.Printf("Checking %d saved searches for changes...\n\n", len(searches))

	// Check each search for changes
	allDiffs, err := diffService.GetDiffForAllSearches(ctx, searchService)
	if err != nil {
		log.Fatalf("Failed to get diffs: %v", err)
	}

	totalChanges := 0
	for _, diffs := range allDiffs {
		totalChanges += len(diffs)
	}

	if totalChanges == 0 {
		fmt.Println("\nNo changes found across all searches.")
		return
	}

	fmt.Printf("\nTotal changes: %d\n\n", totalChanges)

	// Format and output results
	switch *outputFormat {
	case "cli":
		cliFormatter := output2.NewCLIFormatter()
		for keyword, diffs := range allDiffs {
			fmt.Printf("\n=== Changes for '%s' ===\n", keyword)
			result, err := cliFormatter.FormatDiff(diffs, *verbose)
			if err != nil {
				log.Printf("Failed to format diff for '%s': %v\n", keyword, err)
				continue
			}
			fmt.Print(result)
		}

	case "html":
		htmlFormatter := output2.NewHTMLFormatter()

		// Combine all diffs into one HTML file
		var combinedHTML string
		combinedHTML += `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>All Changes</title></head><body>`

		for keyword, diffs := range allDiffs {
			combinedHTML += fmt.Sprintf("<h1>Changes for '%s'</h1>", keyword)
			result, err := htmlFormatter.FormatDiff(diffs, *verbose)
			if err != nil {
				log.Printf("Failed to format diff for '%s': %v\n", keyword, err)
				continue
			}
			combinedHTML += result
		}

		combinedHTML += `</body></html>`

		// Create temp/output directory if it doesn't exist
		os.MkdirAll("temp/output", 0755)

		// Use provided filename or generate with timestamp
		outputFile := *htmlFile
		if outputFile == "diff-results.html" {
			// Default - save to temp/output with timestamp
			timestamp := time.Now().Format("20060102_150405")
			outputFile = fmt.Sprintf("temp/output/diff_%s.html", timestamp)
		}

		if err := os.WriteFile(outputFile, []byte(combinedHTML), 0644); err != nil {
			log.Fatalf("Failed to write HTML file: %v", err)
		}
		fmt.Printf("HTML output saved to %s\n", outputFile)

	case "email":
		if cfg.SMTP.User == "" || cfg.SMTP.Password == "" {
			log.Fatal("SMTP credentials not configured. Please set SMTP_USER and SMTP_PASSWORD in .env file")
		}

		emailSender := output2.NewEmailSender(&cfg.SMTP)
		htmlFormatter := output2.NewHTMLFormatter()

		for keyword, diffs := range allDiffs {
			htmlContent, err := htmlFormatter.FormatDiff(diffs, *verbose)
			if err != nil {
				log.Printf("Failed to format diff for '%s': %v\n", keyword, err)
				continue
			}

			if err := emailSender.SendDiffEmail(keyword, htmlContent); err != nil {
				log.Printf("Failed to send email for '%s': %v\n", keyword, err)
				continue
			}

			fmt.Printf("Email sent for '%s'\n", keyword)
		}

		fmt.Println("Email notifications sent successfully")

	default:
		log.Fatalf("Unknown output format: %s", *outputFormat)
	}

	// Good-offer Telegram notifications: independent of -output above, this
	// runs for every eBay new/price-down diff whose search has a good-offer
	// threshold configured (see D1/D3 in .agents/plan.md).
	notifyGoodOffers(ctx, repo, output2.NewTelegramNotifier(&cfg.Telegram), searches, allDiffs)
}

// notifyGoodOffers evaluates each eBay new/price-down diff against its
// search's good-offer thresholds and sends a Telegram notification for any
// match.
func notifyGoodOffers(
	ctx context.Context,
	repo database2.Repository,
	notifier *output2.TelegramNotifier,
	searches []domain2.Search,
	allDiffs map[string][]domain2.ProductDiff,
) {
	searchByKeyword := make(map[string]domain2.Search, len(searches))
	for _, s := range searches {
		searchByKeyword[s.Keyword] = s
	}

	for keyword, diffs := range allDiffs {
		search, ok := searchByKeyword[keyword]
		if !ok || (search.MaxPrice == nil && search.AvgDiscountPct == nil) {
			continue
		}

		var storedProducts []domain2.Product
		for _, diff := range diffs {
			if diff.Product.ShopSource != "ebay.com" {
				continue
			}
			if diff.DiffType != domain2.DiffTypeNew && diff.DiffType != domain2.DiffTypePriceDown {
				continue
			}

			if storedProducts == nil {
				var err error
				storedProducts, err = repo.GetProductsBySearchID(ctx, search.ID)
				if err != nil {
					log.Printf("Good-offer check: failed to fetch prior products for '%s': %v\n", keyword, err)
					break
				}
			}

			var priorPrices []float64
			for _, p := range storedProducts {
				if p.URL != diff.Product.URL {
					priorPrices = append(priorPrices, p.Price)
				}
			}

			if !service2.EvaluateGoodOffer(search, diff.Product, priorPrices) {
				continue
			}

			if err := notifier.SendGoodOffer(diff.Product, search); err != nil {
				log.Printf("Good offer for '%s' (%s): Telegram send failed: %v\n", keyword, diff.Product.URL, err)
			} else {
				fmt.Printf("Good offer! Telegram notification sent for '%s': %s (%.2f %s)\n", keyword, diff.Product.Title, diff.Product.Price, diff.Product.Currency)
			}
		}
	}
}
