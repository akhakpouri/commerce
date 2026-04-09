package order

import (
	"commerce/api/internal/helpers"
	"commerce/api/internal/services/order"

	err_dto "commerce/api/internal/dto/err"
	dto "commerce/api/internal/dto/order"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	svc order.OrderServiceI
}

func NewOrderHandler(svc order.OrderServiceI) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/:id", h.GetById)
	rg.GET("/statuses", h.GetStatuses)
	rg.POST("/", h.Save)
	rg.PATCH("/:id/status", h.UpdateStatus)
	rg.DELETE("/:id", h.Delete)
}

// GetOrder godoc
//
//	@Summary	Get the order
//	@Tags		order
//	@Produce	json
//	@Router		/api/order/{id} [get]
//	@Param		id	path	int	true	"Order Id"
//	@Success	200 {object} dto.Order
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *OrderHandler) GetById(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		response := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}

	var order *dto.Order
	order, err = h.svc.GetById(*id)
	if err != nil {
		response := err_dto.ErrorResponse{Code: 404, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	c.JSON(200, order)
}

// GetStatuses godoc
//
//	@Summary	Get list of order statuses
//	@Tags		order
//	@Produce	json
//	@Router		/api/order/statuses [get]
//	@Success	200 {array} dto.OrderStatus
//	@Failure	404	{object}	err_dto.ErrorResponse
func (h *OrderHandler) GetStatuses(c *gin.Context) {
	var statuses []dto.OrderStatus
	statuses = h.svc.GetStatuses()
	c.JSON(200, statuses)
}

// UpdateStatus godoc
//
//	@Summary	update order status
//	@Tags		order
//	@Produce	json
//	@Router		/api/order/{id}/status [patch]
//	@Param		id		path		int		true	"Order ID"
//	@Param   order_status  body      dto.OrderStatus  true  "Provide order status object"
//	@Success	204
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		response := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	var status *dto.OrderStatus
	if err := c.ShouldBindJSON(&status); err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	err = h.svc.UpdateStatus(*id, status.Status)
	if err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.JSON(204, nil)
}

// DeleteOrder godoc
//
//	@Summary	Delete the order
//	@Tags		order
//	@Produce	json
//	@Param		id		path		int		true	"Order ID"
//	@Param		hard	query		bool	false	"Hard delete"
//	@Router		/api/order/{id} [delete]
//	@Success	204
//	@Failure	400	{object}	err_dto.ErrorResponse
//	@Failure	500	{object}	err_dto.ErrorResponse
func (h *OrderHandler) Delete(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	hard := c.DefaultQuery("hard", "false") == "true"
	err = h.svc.Delete(*id, hard)
	if err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	c.JSON(204, nil)
}

// Saveorder godoc
//
//	@Summary	Save the order
//	@Tags		order
//	@Produce	json
//	@Router		/api/order [post]
//	@Param   order  body      dto.Order  true  "Provide order object"
//	@Success	201 {object} dto.Order
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *OrderHandler) Save(c *gin.Context) {
	var order *dto.Order
	if err := c.ShouldBindJSON(&order); err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	err := h.svc.Save(*order)
	if err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.JSON(201, order)
}

// GetOrders godoc
//
//	@Summary	Get orders by user
//	@Tags		order
//	@Produce	json
//	@Router		/api/user/{user_id}/order [get]
//	@Param		user_id	path	int	true	"User Id"
//	@Success	200 {array} dto.Order
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *OrderHandler) GetByUser(c *gin.Context) {
	userId, err := helpers.ParseParamToUint(c.Param("user_id"))
	if err != nil {
		response := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	var orders []*dto.Order
	orders, err = h.svc.GetByUserId(*userId)
	if err != nil {
		response := err_dto.ErrorResponse{Code: 404, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	c.JSON(200, orders)
}
