package configs

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"commerce/internal/shared/constants"

	cfg "commerce/internal/shared/configs"

	"github.com/lpernett/godotenv"
)

type Config struct {
	Database cfg.DatabaseConfig
}

func NewConfig() *Config {
	if _, err := os.Stat("configs/dev.env"); err == nil {
		if err = godotenv.Load("configs/dev.env"); err != nil {
			slog.Error("Error loading configs/dev.env", "error", err)
		}
	}

	portStr := cfg.GetEnvOrPanic(constants.EnvKeys.DBPort)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		panic(fmt.Sprintf("invalid DB_PORT value: %s", portStr))
	}

	c := &Config{
		Database: cfg.DatabaseConfig{
			Host:     cfg.GetEnvOrPanic(constants.EnvKeys.DBHost),
			Port:     port,
			User:     cfg.GetEnvOrPanic(constants.EnvKeys.DBUser),
			Password: cfg.GetEnvOrPanic(constants.EnvKeys.DBPassword),
			DbName:   cfg.GetEnvOrPanic(constants.EnvKeys.DBName),
			SSLMode:  cfg.GetEnvOrPanic(constants.EnvKeys.DBSSLMode),
			Schema:   cfg.GetEnvOrPanic(constants.EnvKeys.DBSchema),
		},
	}

	return c
}
