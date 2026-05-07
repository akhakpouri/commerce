package router

import (
	"commerce/api/configs"
	"commerce/api/container"
	address_handler "commerce/api/internal/handlers/address"
	category_handler "commerce/api/internal/handlers/category"
	order_handler "commerce/api/internal/handlers/order"
	payment_handler "commerce/api/internal/handlers/payment"
	product_handler "commerce/api/internal/handlers/product"
	review_handler "commerce/api/internal/handlers/review"
	tax_handler "commerce/api/internal/handlers/tax"
	user_handler "commerce/api/internal/handlers/user"

	health_handler "commerce/api/internal/handlers/health"

	"github.com/gin-gonic/gin"
	swagger "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterRoutes(router *gin.Engine, c *container.Container, config *configs.Config) {
	api := router.Group("/api")
	health := router.Group("/health")
	addressHandler := address_handler.NewAddressHandler(c.AddressService)
	categoryHandler := category_handler.NewCategoryHandler(c.ProductService, c.CategoryService)
	taxHandler := tax_handler.NewTaxHandler(c.TaxService)
	orderHandler := order_handler.NewOrderHandler(c.OrderService)
	paymentHandler := payment_handler.NewPaymentHandler(c.PaymentService)
	productHandler := product_handler.NewProductHandler(c.ProductService)
	userHandler := user_handler.NewUserHandler(c.UserService)
	reviewHandler := review_handler.NewReviewHandler(c.ReviewService)

	healthHandler := health_handler.NewHealthHandler()
	addressHandler.RegisterRoutes(api.Group("/address"))
	categoryHandler.RegisterRoutes(api.Group("/category"))
	taxHandler.RegisterRoutes(api.Group("/tax"))
	orderHandler.RegisterRoutes(api.Group("/orders"))
	paymentHandler.RegisterRoutes(api.Group("/payment"))
	productHandler.RegisterRoutes(api.Group("/products"))
	userHandler.RegisterRoutes(api.Group("/user"))
	reviewHandler.RegisterRoutes(api.Group("/review"))

	healthHandler.RegisterRoutes(health.Group("/status"))

	api.Group("/users/:user_id").GET("/addresses", addressHandler.GetByUserId)
	api.Group("/users/:user_id").GET("/orders", orderHandler.GetByUser)
	api.Group("/orders/:id").GET("/payments", paymentHandler.GetByOrder)
	api.Group("/products/:id").GET("/reviews", reviewHandler.GetAllByProduct)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swagger.Handler))
}
