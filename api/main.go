package main

import (
	"commerce/api/configs"
	"commerce/api/server"
	"log/slog"

	routes "commerce/api/server/router"

	"github.com/gin-gonic/gin"
)

func main() {
	config := configs.NewConfig()

	router := gin.Default()
	router.Use(config.CorsNew())

	routes.RegisterRoutes(router)
	server := server.NewServer(*slog.Default(), router, config)
	server.Run()
}
