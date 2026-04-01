package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg DbConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s dbname=%s port=%d password=%s sslmode=%s search_path=%s",
		cfg.Host, cfg.User, cfg.DbName, cfg.Port, cfg.Password, cfg.SSLMode, cfg.Schema)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func Migrate(cfg DbConfig) {
	db, err := Connect(cfg)
	if err != nil {
		log.Fatal(err)
		panic("Failed to connect to the database")
	}

	log.Println("Connected to the database successfully.")
	log.Println("Running migration.")

	err = setup(db)
	if err != nil {
		log.Fatal("Migration failed: ", err)
		panic(fmt.Sprintf("Failed to migrate database, %v", err))
	}
	log.Println("Migration completed successfully.")
}
