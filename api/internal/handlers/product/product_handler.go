package product

import (
	dto "commerce/api/internal/dto/product"
	svc "commerce/api/internal/services/product"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	svc svc.ProductServiceI
}

func NewProductHandler(svc svc.ProductServiceI) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func (h *ProductHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/", h.GetAll)
	rg.GET("/:id", h.GetById)
}

// GetProducts godoc
//
//	@Summary	Get the list of products
//	@Tags		products
//	@Produce	json
//	@Router		/api/products [get]
//	@Success	200 {array} dto.Product
func (h *ProductHandler) GetAll(c *gin.Context) {
	var products []*dto.Product
	products, err := h.svc.GetAll()
	if err != nil {
		c.JSON(500, err)
	}
	c.JSON(200, products)
}

// GetProduct godoc
//
//	@Summary	Get the product
//	@Tags		product
//	@Produce	json
//	@Router		/api/products:id [get]
//	@Success	200 {object} dto.Product
func (h *ProductHandler) GetById(c *gin.Context) {

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	var product *dto.Product
	product, err = h.svc.GetById(uint(id))
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, product)
}
