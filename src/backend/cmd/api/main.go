package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"secondHand/src/backend/internal/config"
	database2 "secondHand/src/backend/internal/database"
	"secondHand/src/backend/internal/domain"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

type API struct {
	repo database2.Repository
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SearchResponse struct {
	ID        int64     `json:"id"`
	Keyword   string    `json:"keyword"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProductResponse struct {
	ID          int64              `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Price       float64            `json:"price"`
	Currency    string             `json:"currency"`
	URL         string             `json:"url"`
	ImageURL    string             `json:"image_url,omitempty"`
	Location    string             `json:"location,omitempty"`
	ShopSource  string             `json:"shop_source"`
	AuctionType domain.AuctionType `json:"auction_type"`
	Condition   domain.Condition   `json:"condition"`
	EndingTime  *time.Time         `json:"ending_time,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type SearchWithProductsResponse struct {
	Search   SearchResponse    `json:"search"`
	Products []ProductResponse `json:"products"`
	Total    int               `json:"total"`
}

func (a *API) handleGetSearches(w http.ResponseWriter, r *http.Request) {
	searches, err := a.repo.GetAllSearches(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch searches", err)
		return
	}

	response := make([]SearchResponse, len(searches))
	for i, s := range searches {
		updatedAt := s.CreatedAt
		if s.LastCheckedAt != nil {
			updatedAt = *s.LastCheckedAt
		}
		response[i] = SearchResponse{
			ID:        s.ID,
			Keyword:   s.Keyword,
			CreatedAt: s.CreatedAt,
			UpdatedAt: updatedAt,
		}
	}

	respondJSON(w, http.StatusOK, response)
}

func (a *API) handleGetSearchProducts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	searchIDStr := vars["searchId"]

	searchID, err := strconv.ParseInt(searchIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid search ID", err)
		return
	}

	// Get the search
	search, err := a.repo.GetSearchByID(r.Context(), searchID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Search not found", err)
		return
	}

	// Get products for this search
	products, err := a.repo.GetProductsBySearchID(r.Context(), searchID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch products", err)
		return
	}

	productResponses := make([]ProductResponse, len(products))
	for i, p := range products {
		productResponses[i] = ProductResponse{
			ID:          p.ID,
			Title:       p.Title,
			Description: p.Description,
			Price:       p.Price,
			Currency:    p.Currency,
			URL:         p.URL,
			ImageURL:    p.ImageURL,
			Location:    p.Location,
			ShopSource:  p.ShopSource,
			AuctionType: p.AuctionType,
			Condition:   p.Condition,
			EndingTime:  p.EndingTime,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		}
	}

	updatedAt := search.CreatedAt
	if search.LastCheckedAt != nil {
		updatedAt = *search.LastCheckedAt
	}

	response := SearchWithProductsResponse{
		Search: SearchResponse{
			ID:        search.ID,
			Keyword:   search.Keyword,
			CreatedAt: search.CreatedAt,
			UpdatedAt: updatedAt,
		},
		Products: productResponses,
		Total:    len(productResponses),
	}

	respondJSON(w, http.StatusOK, response)
}

func (a *API) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
		log.Printf("Error: %s - %v", message, err)
	}

	response := ErrorResponse{
		Error:   message,
		Message: errMsg,
	}

	respondJSON(w, status, response)
}

func main() {
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

	// Initialize API
	api := &API{repo: repo}

	// Setup router
	r := mux.NewRouter()

	// API routes
	apiRouter := r.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/health", api.handleHealthCheck).Methods("GET")
	apiRouter.HandleFunc("/searches", api.handleGetSearches).Methods("GET")
	apiRouter.HandleFunc("/searches/{searchId}/products", api.handleGetSearchProducts).Methods("GET")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	// Get port from environment or use default
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8091"
	}

	// Start server
	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("Starting API server on %s", addr)
	log.Printf("Health check: http://localhost:%s/api/v1/health", port)
	log.Printf("Searches: http://localhost:%s/api/v1/searches", port)
	log.Printf("Products: http://localhost:%s/api/v1/searches/{id}/products", port)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
