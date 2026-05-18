package category

import (
	auth "commerce/api/internal/auth"
	dto "commerce/api/internal/dto/category"
	errdto "commerce/api/internal/dto/err"
	product_dto "commerce/api/internal/dto/product"
	"commerce/api/internal/helpers"
	category_svc "commerce/api/internal/services/category"
	product_svc "commerce/api/internal/services/product"

	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	productSvc product_svc.ProductServiceI
	svc        category_svc.CategoryServiceI
}

func NewCategoryHandler(productSvc product_svc.ProductServiceI,
	svc category_svc.CategoryServiceI) *CategoryHandler {
	return &CategoryHandler{
		productSvc: productSvc,
		svc:        svc,
	}
}

func (h *CategoryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/:id/products", auth.RequireScope(auth.Scopes.Products.Read), h.GetAllProductsByCategory)
	rg.GET("/:id", auth.RequireScope(auth.Scopes.Category.Read), h.GetById)
	rg.GET("/:id/children", auth.RequireScope(auth.Scopes.Category.Read), h.GetAllByParentId)
	rg.GET("/", auth.RequireScope(auth.Scopes.Category.Read), h.GetAll)
	rg.POST("/", auth.RequireScope(auth.Scopes.Category.Write), h.Save)
	rg.DELETE("/:id", auth.RequireScope(auth.Scopes.Category.Write), h.Delete)
}

// DeleteCategory godoc
//
//	@Summary	Delete category
//	@Tags		category
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"Category ID"
//	@Router		/api/category/{id} [delete]
//	@Success	204
//	@Failure	400	{object}	errdto.ErrorResponse
//	@Failure	500	{object}	errdto.ErrorResponse
//	@Failure	401 {object}	errdto.ErrorResponse
//	@Failure	403 {object}	errdto.ErrorResponse
func (h *CategoryHandler) Delete(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 400, Message: "invalid id"}
		c.JSON(400, errorResponse)
		return
	}
	err = h.svc.Delete(*id, false)
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.Status(204)
}

// GetAllProductsByCategory godoc
//
//	@Summary	Get products by category
//	@Tags		category
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"Category ID"
//	@Router		/api/category/{id}/products [get]
//	@Success	200	{array}		product_dto.Product
//	@Failure	400	{object}	errdto.ErrorResponse
//	@Failure	500	{object}	errdto.ErrorResponse
//	@Failure	401 {object}	errdto.ErrorResponse
//	@Failure	403 {object}	errdto.ErrorResponse
func (h *CategoryHandler) GetAllProductsByCategory(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 400, Message: "invalid id"}
		c.JSON(400, errorResponse)
		return
	}
	var products []*product_dto.Product
	products, err = h.productSvc.GetAllByCategory(*id)
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 404, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	c.JSON(200, products)
}

// GetAllCategories godoc
//
//	@Summary	Get all categories
//	@Tags		category
//	@Produce	json
//	@Security	BearerAuth
//	@Router		/api/category [get]
//	@Success	200	{array}		dto.Category
//	@Failure	500	{object}	errdto.ErrorResponse
//	@Failure	401 {object}	errdto.ErrorResponse
//	@Failure	403 {object}	errdto.ErrorResponse
func (h *CategoryHandler) GetAll(c *gin.Context) {
	var categories []*dto.Category
	categories, err := h.svc.GetAll()
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.JSON(200, categories)
}

// GetAllByParentId godoc
//
//	@Summary	Get subcategories by parent category
//	@Tags		category
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"Category ID"
//	@Router		/api/category/{id}/children [get]
//	@Success	200	{array}		dto.Category
//	@Failure	400	{object}	errdto.ErrorResponse
//	@Failure	500	{object}	errdto.ErrorResponse
//	@Failure	401 {object}	errdto.ErrorResponse
//	@Failure	403 {object}	errdto.ErrorResponse
func (h *CategoryHandler) GetAllByParentId(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 400, Message: "invalid id"}
		c.JSON(400, errorResponse)
		return
	}
	var categories []*dto.Category
	categories, err = h.svc.GetAllByParentId(*id)
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.JSON(200, categories)
}

// GetCategory godoc
//
//	@Summary	Get the category
//	@Tags		category
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"Category ID"
//	@Router		/api/category/{id} [get]
//	@Success	200	{object}	dto.Category
//	@Failure	400	{object}	errdto.ErrorResponse
//	@Failure	500	{object}	errdto.ErrorResponse
//	@Failure	401 {object}	errdto.ErrorResponse
//	@Failure	403 {object}	errdto.ErrorResponse
func (h *CategoryHandler) GetById(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 400, Message: "invalid id"}
		c.JSON(400, errorResponse)
		return
	}

	var category *dto.Category
	category, err = h.svc.GetById(*id)
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	c.JSON(200, category)
}

// SaveCategory godoc
//
//	@Summary	Save the category
//	@Tags		category
//	@Produce	json
//	@Security	BearerAuth
//	@Router		/api/category 	[post]
//	@Param   category  body     dto.Category  true  "Provide category object"
//	@Success	201 {object} 	dto.Category
//	@Failure	400 {object} 	errdto.ErrorResponse
//	@Failure	500 {object} 	errdto.ErrorResponse
//	@Failure	401 {object}	errdto.ErrorResponse
//	@Failure	403 {object}	errdto.ErrorResponse
func (h *CategoryHandler) Save(c *gin.Context) {
	var category *dto.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		errorResponse := errdto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(400, errorResponse)
		return
	}
	err := h.svc.Save(category)
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.JSON(201, category)
}
