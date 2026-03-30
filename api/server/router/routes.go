package router

import (
	tax_handler "commerce/api/internal/handlers/tax"
	tax_service "commerce/api/internal/services/tax"

	"github.com/gin-gonic/gin"
	swagger "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api")
	taxSvc := tax_service.NewTaxService()
	taxHandler := tax_handler.NewTaxHandler(taxSvc)
	// productRepo := product_rep.NewProductRepository()
	// productService := product_service.NewProductService(productRepo)
	// productHandler := product_handlder.NewPaymentHandler(productService)

	taxHandler.GetStates(api.Group("/taxes"))
	// productHandler.RegisterRoutes(api.Group("/products"))
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swagger.Handler))
}
