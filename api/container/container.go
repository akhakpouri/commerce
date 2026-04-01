package container

import (
	address_repo "commerce/internal/shared/repositories/address"
	category_repo "commerce/internal/shared/repositories/category"
	order_repo "commerce/internal/shared/repositories/order"
	order_item_repo "commerce/internal/shared/repositories/order-item"
	payment_repo "commerce/internal/shared/repositories/payment"
	product_repo "commerce/internal/shared/repositories/product"
	review_repo "commerce/internal/shared/repositories/review"
	user_repo "commerce/internal/shared/repositories/user"

	address_service "commerce/api/internal/services/address"
	category_service "commerce/api/internal/services/category"
	order_service "commerce/api/internal/services/order"
	order_item_service "commerce/api/internal/services/order-item"
	payment_service "commerce/api/internal/services/payment"
	product_service "commerce/api/internal/services/product"
	review_service "commerce/api/internal/services/review"
	tax_service "commerce/api/internal/services/tax"
	user_service "commerce/api/internal/services/user"

	"gorm.io/gorm"
)

type Container struct {
	AddressService   address_service.AddressServiceI
	CategoryService  category_service.CategoryServiceI
	OrderService     order_service.OrderServiceI
	OrderItemService order_item_service.OrderItemServiceI
	PaymentService   payment_service.PaymentServiceI
	ProductService   product_service.ProductServiceI
	ReviewService    review_service.ReviewServiceI
	TaxService       tax_service.TaxServiceI
	UserService      user_service.UserServiceI
}

func NewContainer(db *gorm.DB) *Container {
	addressRepo := address_repo.NewAddressRepository(db)
	categoryRepo := category_repo.NewCategoryRepository(db)
	orderItemRepo := order_item_repo.NewOrderItemRepository(db)
	orderRepo := order_repo.NewOrderRepository(db)
	paymentRepo := payment_repo.NewPaymentRepository(db)
	productRepo := product_repo.NewProductRepository(db)
	reviewRepo := review_repo.NewReviewRepository(db)
	userRepo := user_repo.NewUserRepository(db)

	taxService := tax_service.NewTaxService()
	orderService := order_service.NewOrderService(orderRepo, taxService)

	return &Container{
		AddressService:   address_service.NewAddressService(addressRepo),
		CategoryService:  category_service.NewCategoryService(categoryRepo),
		OrderItemService: order_item_service.NewOrderItemService(orderItemRepo),
		OrderService:     orderService,
		TaxService:       taxService,
		PaymentService:   payment_service.NewPaymentService(paymentRepo),
		ProductService:   product_service.NewProductService(productRepo),
		ReviewService:    review_service.NewReviewService(reviewRepo),
		UserService:      user_service.NewUserService(userRepo),
	}
}
