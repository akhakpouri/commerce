package managers

import (
	"commerce/internal/shared/database"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

func NewDbConfig(dbconfig []byte) (database.DbConfig, error) {
	cfg, err := dbConfigFromFile(dbconfig)
	if err != nil {
		slog.Error("Error parsing config file, falling back to environment variables:", "error", err)
		return getConfigFromEnv(), nil
	}

	if cfg == (database.DbConfig{}) {
		slog.Info("Error parsing config file, falling back...")
		return getConfigFromEnv(), nil
	}

	return cfg, nil
}

func getConfigFromEnv() database.DbConfig {
	cfg := database.DbConfig{}
	if host, ok := os.LookupEnv("DB_HOST"); ok {
		cfg.Host = host
	}

	if user, ok := os.LookupEnv("DB_USER"); ok {
		cfg.User = user
	}

	if dbName, ok := os.LookupEnv("DB_NAME"); ok {
		cfg.DbName = dbName
	}

	if password, ok := os.LookupEnv("DB_PASSWORD"); ok {
		cfg.Password = password
	}

	if sslMode, ok := os.LookupEnv("DB_SSLMODE"); ok {
		cfg.SSLMode = sslMode
	}

	if schema, ok := os.LookupEnv("DB_SCHEMA"); ok {
		cfg.Schema = schema
	}

	cfg.Port = getPortFromEnv()
	return cfg
}

func getPortFromEnv() int {
	const defaultPort = 5432
	if port, ok := os.LookupEnv("DB_PORT"); ok {
		p, err := strconv.Atoi(port)
		if err != nil {
			slog.Error("Invalid port value in environment variable:", "error", err)
			return defaultPort
		}
		return p
	}
	return defaultPort
}

func dbConfigFromFile(config []byte) (database.DbConfig, error) {
	var cfg database.DbConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		slog.Error("failed to decode config file:", "error", err)
		return database.DbConfig{}, fmt.Errorf("failed to decode config file: %w", err)
	}

	return cfg, nil
}

