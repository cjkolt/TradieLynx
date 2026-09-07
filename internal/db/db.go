package db

import (
	"database/sql"
	"fmt"

	// Is not called directly in our code, just imported to enable PostgreSQL driver support
	_ "github.com/lib/pq" // '_' keeps import

	"tradielynx/internal/config"
)

// Global DB variable
var DB *sql.DB

// Initializes and returns PostgreSQL database connection
func InitializeDB() (*sql.DB, error) {
	dbConfig := config.AppConfig.DBConfig
	dbConnectionString := fmt.Sprintf( // Builds the string from the JSON file
		"user=%s password=%s dbname=%s host=%s port=%d sslmode=disable",
		dbConfig.DBUser,
		dbConfig.DBPassword,
		dbConfig.DBName,
		dbConfig.DBHost,
		dbConfig.DBPort,
	)

	// Opens the database
	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		return nil, fmt.Errorf("Error opening database: %w", err)
	}

	// Tests database connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("Error: could not ping DB: %w", err)
	}

	return db, nil
}
