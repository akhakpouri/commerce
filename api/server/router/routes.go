package router

import (
	"commerce/api/container"
	address_handler "commerce/api/internal/handlers/address"
	category_handler "commerce/api/internal/handlers/category"
	payment_handler "commerce/api/internal/handlers/payment"
	product_handler "commerce/api/internal/handlers/product"
	tax_handler "commerce/api/internal/handlers/tax"
	user_handler "commerce/api/internal/handlers/user"

	"github.com/gin-gonic/gin"
	swagger "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterRoutes(router *gin.Engine, c *container.Container) {
	api := router.Group("/api")
	addressHandler := address_handler.NewAddressHandler(c.AddressService)
	categoryHandler := category_handler.NewCategoryHandler(c.ProductService, c.CategoryService)
	taxHandler := tax_handler.NewTaxHandler(c.TaxService)
	paymentHandler := payment_handler.NewPaymentHandler(c.PaymentService)
	productHandler := product_handler.NewProductHandler(c.ProductService)
	userHandler := user_handler.NewUserHandler(c.UserService)

	addressHandler.RegisterRoutes(api.Group("/address"))
	categoryHandler.RegisterRoutes(api.Group("/category"))
	taxHandler.RegisterRoutes(api.Group("/tax"))
	paymentHandler.RegisterRoutes(api.Group("/payment"))
	productHandler.RegisterRoutes(api.Group("/products"))
	userHandler.RegisterRoutes(api.Group("/user"))

	api.Group("/users/:user_id").GET("/addresses", addressHandler.GetByUserId)
	api.Group("/orders/:order_id").GET("/payments", paymentHandler.GetByOrder)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swagger.Handler))
}
