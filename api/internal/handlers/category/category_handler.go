package category

import (
	errdto "commerce/api/internal/dto/err"
	product_dto "commerce/api/internal/dto/product"
	category_svc "commerce/api/internal/services/category"
	product_svc "commerce/api/internal/services/product"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	productSvc product_svc.ProductServiceI
	svc        category_svc.CategoryServiceI
}

func NewCategoryHanlder(productSvc product_svc.ProductServiceI,
	svc category_svc.CategoryServiceI) *CategoryHandler {
	return &CategoryHandler{
		productSvc: productSvc,
		svc:        svc,
	}
}

func (h *CategoryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/:id/products", h.GetAllProductsByCategory)
}

// GetAllProductsByCategory godoc
//
//	@Summary	Get products by category
//	@Tags		category
//	@Produce	json
//	@Param		id	path		int	true	"Category ID"
//	@Router		/api/category/:id/products [get]
//	@Success	200	{array}		product_dto.Product
//	@Failure	400	{object}	errdto.ErrorResponse
//	@Failure	500	{object}	errdto.ErrorResponse
func (h *CategoryHandler) GetAllProductsByCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 400, Message: "invalid id"}
		c.JSON(400, errorResponse)
		return
	}
	var products = []*product_dto.Product{}
	products, err = h.productSvc.GetAllByCategory(uint(id))
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 404, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	c.JSON(200, products)
}
