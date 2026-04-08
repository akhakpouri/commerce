package review

import (
	errdto "commerce/api/internal/dto/err"
	dto "commerce/api/internal/dto/review"
	"commerce/api/internal/helpers"
	"commerce/api/internal/services/review"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	svc review.ReviewServiceI
}

func NewReviewHandler(svc review.ReviewServiceI) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

func (h *ReviewHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/:id", h.GetById)
	rg.POST("/", h.Save)
	rg.DELETE("/:id", h.Delete)
}

// Deletereview godoc
//
//	@Summary	Delete review
//	@Tags		review
//	@Produce	json
//	@Param		id	path		int	true	"review ID"
//	@Param 		hard query 		bool false "hard delete"
//	@Router		/api/review/{id} [delete]
//	@Success	204
//	@Failure	400	{object}	errdto.ErrorResponse
//	@Failure	500	{object}	errdto.ErrorResponse
func (h *ReviewHandler) Delete(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 400, Message: "invalid id"}
		c.JSON(400, errorResponse)
		return
	}
	hard := c.DefaultQuery("hard", "false") == "true"
	err = h.svc.Delete(*id, hard)

	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.Status(204)
}

// GetAllByParentId godoc
//
//	@Summary	Get reviews for productg
//	@Tags		product
//	@Produce	json
//	@Param		id	path		int	true	"product id"
//	@Router		/api/products/{id}/reviews [get]
//	@Success	200	{array}		dto.Review
//	@Failure	400	{object}	errdto.ErrorResponse
//	@Failure	500	{object}	errdto.ErrorResponse
func (h *ReviewHandler) GetAllByProduct(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 400, Message: "invalid id"}
		c.JSON(400, errorResponse)
		return
	}
	var reviews []*dto.Review
	reviews, err = h.svc.GetAllByProduct(*id)
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.JSON(200, reviews)
}

// Getreview godoc
//
//	@Summary	Get the review
//	@Tags		review
//	@Produce	json
//	@Param		id	path		int	true	"review ID"
//	@Router		/api/review/{id} [get]
//	@Success	200	{object}	dto.Review
//	@Failure	400	{object}	errdto.ErrorResponse
//	@Failure	500	{object}	errdto.ErrorResponse
func (h *ReviewHandler) GetById(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 400, Message: "invalid id"}
		c.JSON(400, errorResponse)
		return
	}

	var review *dto.Review
	review, err = h.svc.GetById(*id)
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	c.JSON(200, review)
}

// Savereview godoc
//
//	@Summary	Save the review
//	@Tags		review
//	@Produce	json
//	@Router		/api/review [post]
//	@Param   review  body      dto.Review  true  "Provide review object"
//	@Success	201 {object} dto.Review
//	@Failure	400 {object} errdto.ErrorResponse
//	@Failure	500 {object} errdto.ErrorResponse
func (h *ReviewHandler) Save(c *gin.Context) {
	var review *dto.Review
	if err := c.ShouldBindJSON(&review); err != nil {
		errorResponse := errdto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(400, errorResponse)
		return
	}
	err := h.svc.Save(review)
	if err != nil {
		errorResponse := errdto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.JSON(201, review)
}
