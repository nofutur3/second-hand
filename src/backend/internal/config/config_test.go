package config

import (
	"os"
	"testing"
)

func TestConnectionString(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	result := cfg.ConnectionString()

	if result != expected {
		t.Errorf("ConnectionString() = %q, want %q", result, expected)
	}
}

func TestGetEnv(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	result := getEnv("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("getEnv() = %q, want %q", result, "test_value")
	}

	// Test with non-existing env var
	result = getEnv("NON_EXISTING_VAR", "default")
	if result != "default" {
		t.Errorf("getEnv() = %q, want %q", result, "default")
	}
}

func TestGetEnvInt(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	result := getEnvInt("TEST_INT", 10)
	if result != 42 {
		t.Errorf("getEnvInt() = %d, want %d", result, 42)
	}

	// Test with non-existing env var
	result = getEnvInt("NON_EXISTING_INT", 10)
	if result != 10 {
		t.Errorf("getEnvInt() = %d, want %d", result, 10)
	}

	// Test with invalid int
	os.Setenv("INVALID_INT", "not_a_number")
	defer os.Unsetenv("INVALID_INT")

	result = getEnvInt("INVALID_INT", 10)
	if result != 10 {
		t.Errorf("getEnvInt() with invalid int = %d, want %d", result, 10)
	}
}
