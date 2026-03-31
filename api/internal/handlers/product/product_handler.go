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
	rg.POST("/", h.Save)
	rg.DELETE("/:id", h.Delete)
}

// GetProducts godoc
//
//	@Summary	Get the list of products
//	@Tags		product
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
//	@Router		/api/products/:id [get]
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

// SaveProduct godoc
//
//	@Summary	Save the product
//	@Tags		product
//	@Produce	json
//	@Router		/api/products [post]
//	@Success	201 {object} dto.Product
// @Failure	400 {object} gin.H
// @Failure	500 {object} gin.H

func (h *ProductHandler) Save(c *gin.Context) {
	var product *dto.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	err := h.svc.Save(product)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, product)
}

// DeleteProduct godoc
//
//	@Summary	Delete the product
//	@Tags		product
//	@Produce	json
//	@Router		/api/products/:id [delete]
//	@Success	204
func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	err = h.svc.Delete(uint(id), false)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(204, nil)
}
