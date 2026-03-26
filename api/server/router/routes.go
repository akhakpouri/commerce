package router

import (
	tax_handler "commerce/api/internal/handlers/tax"
	tax_service "commerce/api/internal/services/tax"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")
	taxSvc := tax_service.NewTaxService()
	taxHandler := tax_handler.NewTaxHandler(taxSvc)
	taxHandler.RegisterRoutes(v1.Group("/taxes"))
}
