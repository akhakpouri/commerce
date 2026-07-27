package main

import (
	cfg "commerce/internal/shared/configs"
	"commerce/internal/shared/database"
	"embed"
	"fmt"
)

//go:embed configs/config.json
var content embed.FS

func main() {
	fmt.Println("Welcome to the Commerce Utility Application!")
	migrateDatabase()
}

// migrateDatabase is the main entry point for the utility application.
// It reads the database configuration from the specified file and performs database migrations.
func migrateDatabase() {

	config := cfg.NewConfig()
	database.Migrate(config.Database)
}
