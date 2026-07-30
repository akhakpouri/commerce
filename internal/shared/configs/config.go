package configs

import (
	"commerce/internal/shared/constants"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	db "github.com/akhakpouri/gorm-kit/database"
	"github.com/lpernett/godotenv"
)

type Config struct {
	Database db.DbConfig
}

func NewConfig() *Config {
	if _, err := os.Stat("configs/dev.env"); err == nil {
		if err = godotenv.Load("configs/dev.env"); err != nil {
			slog.Error("Error loading configs/dev.env", "error", err)
		}
	}

	portStr := GetEnvOrPanic(constants.EnvKeys.DBPort)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		panic(fmt.Sprintf("invalid DB_PORT value: %s", portStr))
	}

	c := &Config{
		Database: db.DbConfig{
			Host:     GetEnvOrPanic(constants.EnvKeys.DBHost),
			Port:     port,
			User:     GetEnvOrPanic(constants.EnvKeys.DBUser),
			Password: GetEnvOrPanic(constants.EnvKeys.DBPassword),
			DbName:   GetEnvOrPanic(constants.EnvKeys.DBName),
			SSLMode:  GetEnvOrPanic(constants.EnvKeys.DBSSLMode),
			Schema:   GetEnvOrPanic(constants.EnvKeys.DBSchema),
		},
	}

	return c
}

func NewDbConfig() *db.DbConfig {
	if _, err := os.Stat("configs/dev.env"); err == nil {
		if err = godotenv.Load("configs/dev.env"); err != nil {
			slog.Error("Error loading configs/dev.env", "error", err)
		}
	}

	portStr := GetEnvOrPanic(constants.EnvKeys.DBPort)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		panic(fmt.Sprintf("invalid DB_PORT value: %s", portStr))
	}

	c := &db.DbConfig{
		Host:     GetEnvOrPanic(constants.EnvKeys.DBHost),
		Port:     port,
		User:     GetEnvOrPanic(constants.EnvKeys.DBUser),
		Password: GetEnvOrPanic(constants.EnvKeys.DBPassword),
		DbName:   GetEnvOrPanic(constants.EnvKeys.DBName),
		SSLMode:  GetEnvOrPanic(constants.EnvKeys.DBSSLMode),
		Schema:   GetEnvOrPanic(constants.EnvKeys.DBSchema),
	}

	return c
}
