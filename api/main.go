package main

import (
	"commerce/api/configs"
	"commerce/api/container"
	"commerce/api/server"
	"log/slog"

	routes "commerce/api/server/router"

	_ "commerce/api/docs"

	"github.com/gin-gonic/gin"
)

//go:generate swag init -g main.go --output docs --parseInternal
func main() {
	config := configs.NewConfig()
	db, err := config.Database.Connect()
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		panic("Failed to connect to the database")
	}
	container := container.NewContainer(db)
	router := gin.Default()
	router.Use(config.CorsNew())

	routes.RegisterRoutes(router, container)
	server := server.NewServer(*slog.Default(), router, config)
	server.Run()
}
