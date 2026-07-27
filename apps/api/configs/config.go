package configs

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	cfg "commerce/internal/shared/configs"
	"commerce/internal/shared/constants"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lpernett/godotenv"
)

type serverConfig struct {
	Address string
}

type authConfig struct {
	Domain   string
	Audience string
}

type Config struct {
	Server   serverConfig
	Database cfg.DatabaseConfig
	Auth     authConfig
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
		Server: serverConfig{
			Address: cfg.GetEnvOrPanic(constants.EnvKeys.ServerAddress),
		},
		Database: cfg.DatabaseConfig{
			Host:     cfg.GetEnvOrPanic(constants.EnvKeys.DBHost),
			Port:     port,
			User:     cfg.GetEnvOrPanic(constants.EnvKeys.DBUser),
			Password: cfg.GetEnvOrPanic(constants.EnvKeys.DBPassword),
			DbName:   cfg.GetEnvOrPanic(constants.EnvKeys.DBName),
			SSLMode:  cfg.GetEnvOrPanic(constants.EnvKeys.DBSSLMode),
			Schema:   cfg.GetEnvOrPanic(constants.EnvKeys.DBSchema),
		},
		Auth: authConfig{
			Domain:   cfg.GetEnvOrPanic(constants.EnvKeys.AuthDomain),
			Audience: cfg.GetEnvOrPanic(constants.EnvKeys.AuthAudience),
		},
	}

	return c
}

func (conf *Config) CorsNew() gin.HandlerFunc {
	allowedOrigin := cfg.GetEnvOrPanic(constants.EnvKeys.CorsAllowedOrigin)

	return cors.New(cors.Config{
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders:     []string{constants.Headers.Origin},
		ExposeHeaders:    []string{constants.Headers.ContentLength},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return origin == allowedOrigin
		},
		MaxAge: constants.MaxAge,
	})
}
