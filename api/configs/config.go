package configs

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"commerce/api/internal/constants"

	db "github.com/akhakpouri/gorm-kit/database"
	pg "github.com/akhakpouri/gorm-kit/pg"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lpernett/godotenv"
	"gorm.io/gorm"
)

type serverConfig struct {
	Address string
}

type databaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DbName   string
	SSLMode  string
	Schema   string
}

type authConfig struct {
	Domain   string
	Audience string
}

func (d *databaseConfig) Connect() (*gorm.DB, error) {
	return pg.Connect(db.DbConfig{
		Host:     d.Host,
		Port:     d.Port,
		User:     d.User,
		Password: d.Password,
		DbName:   d.DbName,
		SSLMode:  d.SSLMode,
		Schema:   d.Schema,
	})
}

type Config struct {
	Server   serverConfig
	Database databaseConfig
	Auth     authConfig
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
		Server: serverConfig{
			Address: GetEnvOrPanic(constants.EnvKeys.ServerAddress),
		},
		Database: databaseConfig{
			Host:     GetEnvOrPanic(constants.EnvKeys.DBHost),
			Port:     port,
			User:     GetEnvOrPanic(constants.EnvKeys.DBUser),
			Password: GetEnvOrPanic(constants.EnvKeys.DBPassword),
			DbName:   GetEnvOrPanic(constants.EnvKeys.DBName),
			SSLMode:  GetEnvOrPanic(constants.EnvKeys.DBSSLMode),
			Schema:   GetEnvOrPanic(constants.EnvKeys.DBSchema),
		},
		Auth: authConfig{
			Domain:   GetEnvOrPanic(constants.EnvKeys.AuthDomain),
			Audience: GetEnvOrPanic(constants.EnvKeys.AuthAudience),
		},
	}

	return c
}

func GetEnvOrPanic(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("environment variable %s not set", key))
	}

	return value
}

func (conf *Config) CorsNew() gin.HandlerFunc {
	allowedOrigin := GetEnvOrPanic(constants.EnvKeys.CorsAllowedOrigin)

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
