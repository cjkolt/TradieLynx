package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"tradielynx/internal/config"
)

func InitializeDB() (*sql.DB, error) {
	dbConfig := config.AppConfig.DBConfig
	dbConnectionString := fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%d sslmode=disable",
		dbConfig.DBUser,
		dbConfig.DBPassword,
		dbConfig.DBName,
		dbConfig.DBHost,
		dbConfig.DBPort,
	)

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping database: %w", err)
	}

	return db, nil
}
