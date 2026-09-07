package models

// Holds all application configuration
type Config struct {
	LogDirectory  string   `json:"log_directory"`
	DataDirectory string   `json:"data_directory"`
	DBConfig      DBConfig `json:"db_config"`
	//TODO: add other configuration (database connection strings, API Keys)
}

// Holds all database configuration, used by Config struct as var
type DBConfig struct {
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	DBHost     string `json:"db_host"`
	DBPort     int    `json:"db_port"`
}
