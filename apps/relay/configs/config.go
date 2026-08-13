package configs

import (
	"log/slog"
	"os"

	"commerce/internal/shared/constants"

	cfg "commerce/internal/shared/configs"

	db "github.com/akhakpouri/gorm-kit/database"

	"github.com/lpernett/godotenv"
)

type Config struct {
	Database    db.DbConfig
	Aws         cfg.AWSConfig
	Consumer    cfg.ConsumerConfig
	MaxAttempts int
}

func NewConfig() *Config {
	if _, err := os.Stat("configs/dev.env"); err == nil {
		if err = godotenv.Load("configs/dev.env"); err != nil {
			slog.Error("Error loading configs/dev.env", "error", err)
		}
	}

	c := &Config{
		Database: *cfg.NewDbConfig(),
		Aws: cfg.AWSConfig{
			AccessKeyID:     cfg.GetEnvOrDefault(constants.EnvKeys.AWSAccessKeyID, ""),
			SecretAccessKey: cfg.GetEnvOrDefault(constants.EnvKeys.AWSSecretAccessKey, ""),
			Region:          cfg.GetEnvOrDefault(constants.EnvKeys.AWSRegion, "us-east-1"),
			Endpoint:        cfg.GetEnvOrDefault(constants.EnvKeys.AWSEndpoint, ""),
		},
		Consumer: cfg.ConsumerConfig{
			Max:      cfg.GetEnvOrDefaultToInt(constants.EnvKeys.MaxMessages, 10),
			Timeout:  cfg.GetEnvOrDefaultToInt(constants.EnvKeys.VisibilityTimeout, 30),
			WaitTime: cfg.GetEnvOrDefaultToInt(constants.EnvKeys.WaitTime, 30),
			Count:    cfg.GetEnvOrDefaultToInt(constants.EnvKeys.WorkerCount, 5),
		},
		MaxAttempts: cfg.GetEnvOrDefaultToInt(constants.EnvKeys.OutboxMaxAttempts, 10),
	}

	return c
}
