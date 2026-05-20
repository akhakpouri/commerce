package router

import (
	"commerce/api/configs"
	"commerce/api/container"
	"commerce/api/internal/auth"
	address_handler "commerce/api/internal/handlers/address"
	auth_handler "commerce/api/internal/handlers/auth"
	category_handler "commerce/api/internal/handlers/category"
	order_handler "commerce/api/internal/handlers/order"
	payment_handler "commerce/api/internal/handlers/payment"
	product_handler "commerce/api/internal/handlers/product"
	review_handler "commerce/api/internal/handlers/review"
	tax_handler "commerce/api/internal/handlers/tax"
	user_handler "commerce/api/internal/handlers/user"
	"fmt"

	health_handler "commerce/api/internal/handlers/health"

	"github.com/gin-gonic/gin"
	swagger "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterRoutes(router *gin.Engine, c *container.Container, config *configs.Config) {
	api := router.Group("/api")
	health := router.Group("/health")

	valid, err := auth.NewValidator(config.Auth.Domain, config.Auth.Audience)
	if err != nil {
		panic(fmt.Errorf("auth validator: %w", err))
	}
	mw, err := auth.NewMiddleware(valid)
	if err != nil {
		panic(fmt.Errorf("auth middleware: %w", err))
	}
	ginAuth := auth.Gin(mw)

	authGroup := api.Group("/auth", ginAuth)
	authHandler := auth_handler.NewAuthHandler()
	authHandler.RegisterRoutes(authGroup)

	authedApi := api.Group("", ginAuth, auth.ResolveIdentity(c.UserService))

	addressHandler := address_handler.NewAddressHandler(c.AddressService)
	categoryHandler := category_handler.NewCategoryHandler(c.ProductService, c.CategoryService)
	taxHandler := tax_handler.NewTaxHandler(c.TaxService)
	orderHandler := order_handler.NewOrderHandler(c.OrderService)
	paymentHandler := payment_handler.NewPaymentHandler(c.PaymentService)
	productHandler := product_handler.NewProductHandler(c.ProductService)
	userHandler := user_handler.NewUserHandler(c.UserService)
	reviewHandler := review_handler.NewReviewHandler(c.ReviewService)

	healthHandler := health_handler.NewHealthHandler()
	taxHandler.RegisterRoutes(api.Group("/tax"))

	addressHandler.RegisterRoutes(authedApi.Group("/address"))
	categoryHandler.RegisterRoutes(authedApi.Group("/category"))
	orderHandler.RegisterRoutes(authedApi.Group("/orders"))
	paymentHandler.RegisterRoutes(authedApi.Group("/payment"))
	productHandler.RegisterRoutes(authedApi.Group("/products"))
	userHandler.RegisterRoutes(authedApi.Group("/user"))
	reviewHandler.RegisterRoutes(authedApi.Group("/review"))

	healthHandler.RegisterRoutes(health.Group("/status"))

	authedApi.Group("/users/:user_id").GET("/addresses", auth.RequireScope(auth.Scopes.Users.Read), addressHandler.GetByUserId)
	authedApi.Group("/users/:user_id").GET("/orders", auth.RequireScope(auth.Scopes.Orders.Read), orderHandler.GetByUser)

	authedApi.Group("/orders/:id").GET("/payments", auth.RequireScope(auth.Scopes.Payment.Read), paymentHandler.GetByOrder)

	authedApi.Group("/products/:id").GET("/reviews", auth.RequireScope(auth.Scopes.Reviews.Read), reviewHandler.GetAllByProduct)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swagger.Handler))
}
