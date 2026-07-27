package main

import (
	"commerce/internal/shared/database"
	"commerce/utils/internal/managers"
	"embed"
	"fmt"
	"log/slog"
)

//go:embed configs/config.json
var content embed.FS

func main() {
	fmt.Println("Welcome to the Commerce Utility Application!")
	migrateDatabase("configs/config.json")
}

// migrateDatabase is the main entry point for the utility application.
// It reads the database configuration from the specified file and performs database migrations.
func migrateDatabase(filePath string) {

	dbconfig, err := content.ReadFile(filePath)
	if err != nil {
		slog.Error("Error reading config file, falling back to environment variables:", "error", err)
	}

	cfg, err := managers.NewDbConfig(dbconfig)
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	database.Migrate(cfg)
}
