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
}
