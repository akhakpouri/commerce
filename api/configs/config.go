package configs

import (
	"fmt"
	"net/http"
	"os"

	constants "commerce/api/internal/constants"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lpernett/godotenv"
)

type serverConfig struct {
	Address string
}

type databaseConfig struct {
	Driver     string
	DataSource string
}

type Config struct {
	Server   serverConfig
	Database databaseConfig
}

func NewConfig() *Config {
	err := godotenv.Load("configs/dev.env")
	if err != nil {
		panic("Error loading the .env file!")
	}

	c := &Config{
		Server: serverConfig{
			Address: GetEnvOrPanic(constants.EnvKeys.ServerAddress),
		},
		Database: databaseConfig{
			Driver:     GetEnvOrPanic(constants.EnvKeys.DBDriver),
			DataSource: GetEnvOrPanic(constants.EnvKeys.DBSource),
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
