package product

import (
	svc "commerce/api/internal/services/product"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	svc svc.ProductServiceI
}

func NewPaymentHandler(svc svc.ProductServiceI) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

func (h *PaymentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// GetProducts godoc
	// @Summary Get the list of products
	// @Tags products
	// @Produce json
	// @Router /api/products [get]
	// @Success 200
	rg.GET("/", func(c *gin.Context) {
		products, err := h.svc.GetAll()
		if err != nil {
			c.JSON(500, err)
		}
		c.JSON(200, products)
	})
}
