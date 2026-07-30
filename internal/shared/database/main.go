package database

import (
	"commerce/internal/shared/models"
	"log"

	database "github.com/akhakpouri/gorm-kit/database"
	pg "github.com/akhakpouri/gorm-kit/pg"
)

func Migrate(cfg database.DbConfig) {
	db, err := pg.Connect(cfg)
	if err != nil {
		log.Fatal(err)
		panic("Failed to connect to the database")
	}

	log.Println("Connected to the database successfully.")
	log.Println("Running migration.")

	if err := database.Migrate(
		db,
		&models.Address{},
		&models.User{},
		&models.Product{},
		&models.Category{},
		&models.ProductCategory{},
		&models.Review{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
		&models.Outbox{},
		&models.Event{},
	); err != nil {
		log.Fatal("Migration failed: ", err)
	}
	log.Println("Migration completed successfully.")
}
