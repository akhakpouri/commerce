package main

import (
	cfg "commerce/internal/shared/configs"
	"commerce/internal/shared/database"
	"fmt"
)

func main() {
	fmt.Println("Welcome to the Commerce Utility Application!")
	migrateDatabase()
}

func migrateDatabase() {

	config := cfg.NewConfig()
	database.Migrate(config.Database)
}
