package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"tradielynx/internal/models"
)

// Global Config variable
var AppConfig *models.Config

func LoadConfig(configPath string) (*models.Config, error) {
	cfg := models.Config{}

	// Load from JSON file
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("Error reading config file %s: %w", configPath, err)
		}
		fmt.Printf("Config file '%s' not found. Using environment variables and defaults.\n",
			configPath)
	} else {
		if err := json.Unmarshal(configFile, &cfg); err != nil {
			return nil, fmt.Errorf("Error parsing config file %s: %w", configPath, err)
		}
	}

	// Apply environment variable overrides
	if logDir := os.Getenv("APP_LOG_DIR"); logDir != "" {
		cfg.LogDirectory = logDir
	}
	if dataDir := os.Getenv("APP_DATA_DIR"); dataDir != "" {
		cfg.DataDirectory = dataDir
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		cfg.DBConfig.DBUser = dbUser
	}
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		cfg.DBConfig.DBPassword = dbPassword
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		cfg.DBConfig.DBName = dbName
	}
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		cfg.DBConfig.DBHost = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		port, err := strconv.Atoi(dbPort)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_PORT %q: %w", dbPort, err)
		}
		cfg.DBConfig.DBPort = port
	}

	// Set defaults if still empty (fallback if no file and no env var)
	if cfg.LogDirectory == "" {
		cfg.LogDirectory = filepath.Join(os.TempDir(), "tradielynx-logs")
	}
	if cfg.DataDirectory == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not find user home directory for data default: %w",
				err)
		}
		cfg.DataDirectory = filepath.Join(homeDir, "tradielynx-data")
	}

	return &cfg, nil
}
