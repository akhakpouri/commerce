package configs

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

func GetEnvOrPanic(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("environment variable %s not set", key))
	}

	return value
}

func GetEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func GetEnvOrDefaultToInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	num, err := strconv.Atoi(key)
	if err != nil {
		slog.Error("exception occured when converting string to int", "error", err)
		return defaultValue
	}

	return num
}
