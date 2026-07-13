package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the application configuration
type Config struct {
	Shops    []ShopConfig   `json:"shops"`
	Database DatabaseConfig `json:"-"`
	SMTP     SMTPConfig     `json:"-"`
	Scraping ScrapingConfig `json:"-"`
}

// ShopConfig represents configuration for a shop
type ShopConfig struct {
	URL             string `json:"url"`
	Name            string `json:"name,omitempty"`
	SearchURLFormat string `json:"search_url_format,omitempty"`
	Enabled         bool   `json:"enabled,omitempty"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// SMTPConfig represents SMTP configuration for email notifications
type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
	To       string
}

// ScrapingConfig represents scraping configuration
type ScrapingConfig struct {
	DelayMS        int
	RequestTimeout int
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	// Load JSON config
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Load environment variables
	config.Database = DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "secondhand"),
		Password: getEnv("DB_PASSWORD", "secondhand_dev"),
		DBName:   getEnv("DB_NAME", "secondhand"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	config.SMTP = SMTPConfig{
		Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		Port:     getEnv("SMTP_PORT", "587"),
		User:     getEnv("SMTP_USER", ""),
		Password: getEnv("SMTP_PASSWORD", ""),
		From:     getEnv("SMTP_FROM", ""),
		To:       getEnv("SMTP_TO", ""),
	}

	config.Scraping = ScrapingConfig{
		DelayMS:        getEnvInt("SCRAPE_DELAY_MS", 2000),
		RequestTimeout: getEnvInt("REQUEST_TIMEOUT_SEC", 30),
	}

	// Set default enabled state for shops
	for i := range config.Shops {
		if config.Shops[i].Enabled == false {
			config.Shops[i].Enabled = true
		}
	}

	return &config, nil
}

// ConnectionString returns the PostgreSQL connection string
func (d *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}
