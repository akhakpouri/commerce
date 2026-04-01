package product

import (
	errdto "commerce/api/internal/dto/err"
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
		errorResponse := errdto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
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
		errorResponse := errdto.ErrorResponse{Code: 400, Message: "invalid id"}
		c.JSON(400, errorResponse)
		return
	}

	var product *dto.Product
	product, err = h.svc.GetById(uint(id))
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 404, Message: err.Error()}
		c.JSON(404, errorResponse)
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
//	@Failure	400 {object} errdto.ErrorResponse
//	@Failure	500 {object} errdto.ErrorResponse
func (h *ProductHandler) Save(c *gin.Context) {
	var product *dto.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		errorResponse := errdto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(400, errorResponse)
		return
	}
	err := h.svc.Save(product)
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
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
//	@Failure	400 {object} errdto.ErrorResponse
//	@Failure	500 {object} errdto.ErrorResponse
func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 400, Message: "invalid id"}
		c.JSON(400, errorResponse)
		return
	}
	err = h.svc.Delete(uint(id), false)
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.JSON(204, nil)
}
