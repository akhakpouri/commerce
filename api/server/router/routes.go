package router

import (
	"commerce/api/container"
	product_handler "commerce/api/internal/handlers/product"
	tax_handler "commerce/api/internal/handlers/tax"

	"github.com/gin-gonic/gin"
	swagger "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterRoutes(router *gin.Engine, c *container.Container) {
	api := router.Group("/api")
	taxHandler := tax_handler.NewTaxHandler(c.TaxService)
	productHandler := product_handler.NewProductHandler(c.ProductService)

	taxHandler.RegisterRoutes(api.Group("/tax"))
	productHandler.RegisterRoutes(api.Group("/products"))
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swagger.Handler))
}
