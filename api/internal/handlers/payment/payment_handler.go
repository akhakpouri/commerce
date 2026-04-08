package payment

import (
	"commerce/api/internal/helpers"
	"commerce/api/internal/services/payment"

	err_dto "commerce/api/internal/dto/err"
	dto "commerce/api/internal/dto/payment"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	svc payment.PaymentServiceI
}

func NewPaymentHandler(svc payment.PaymentServiceI) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

func (h *PaymentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/:id", h.GetById)
	rg.GET("/statuses", h.GetStatuses)
	rg.POST("/", h.Save)
	rg.PATCH("/:id/status", h.UpdateStatus)
	rg.DELETE("/:id", h.Delete)
}

// GetPayment godoc
//
//	@Summary	Get the payment
//	@Tags		payment
//	@Produce	json
//	@Router		/api/payment/{id} [get]
//	@Param		id	path	int	true	"Payment Id"
//	@Success	200 {object} dto.Payment
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *PaymentHandler) GetById(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		response := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	var payment *dto.Payment
	payment, err = h.svc.GetById(*id)
	if err != nil {
		response := err_dto.ErrorResponse{Code: 404, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	c.JSON(200, payment)
}

// GetPayment godoc
//
//	@Summary	Get payments by order
//	@Tags		payment
//	@Produce	json
//	@Router		/api/orders/{order_id}/payments [get]
//	@Param		order_id	path	int	true	"Payment Id"
//	@Success	200 {array} dto.Payment
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *PaymentHandler) GetByOrder(c *gin.Context) {
	orderId, err := helpers.ParseParamToUint(c.Param("order_id"))
	if err != nil {
		response := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	var payments []*dto.Payment
	payments, err = h.svc.GetByOrder(*orderId)
	if err != nil {
		response := err_dto.ErrorResponse{Code: 404, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	c.JSON(200, payments)
}

// Savepayment godoc
//
//	@Summary	Save the payment
//	@Tags		payment
//	@Produce	json
//	@Router		/api/payment [post]
//	@Param   payment  body      dto.Payment  true  "Provide payment object"
//	@Success	201 {object} dto.Payment
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *PaymentHandler) Save(c *gin.Context) {
	var payment *dto.Payment
	if err := c.ShouldBindJSON(&payment); err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	err := h.svc.Save(payment)
	if err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.JSON(201, payment)

}

// DeletePayment godoc
//
//	@Summary	Delete the payment
//	@Tags		payment
//	@Produce	json
//	@Param		id		path		int		true	"Payment ID"
//	@Param		hard	query		bool	false	"Hard delete"
//	@Router		/api/payment/{id} [delete]
//	@Success	204
//	@Failure	400	{object}	err_dto.ErrorResponse
//	@Failure	500	{object}	err_dto.ErrorResponse
func (h *PaymentHandler) Delete(c *gin.Context) {
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

// GetStatuses godoc
//
//	@Summary	Get list of payment statuses
//	@Tags		payment
//	@Produce	json
//	@Router		/api/payment/statuses [get]
//	@Success	200 {array} dto.PaymentStatus
//	@Failure	404	{object}	err_dto.ErrorResponse
func (h *PaymentHandler) GetStatuses(c *gin.Context) {
	var statuses []dto.PaymentStatus
	statuses = h.svc.GetStatuses()
	if len(statuses) == 0 {
		response := err_dto.ErrorResponse{Code: 404, Message: "No payment statuses were found"}
		c.JSON(response.Code, response)
		return
	}
	c.JSON(200, statuses)
}

// UpdateStatus godoc
//
//	@Summary	update payment status
//	@Tags		payment
//	@Produce	json
//	@Router		/api/payment/{id}/status [patch]
//	@Param		id		path		int		true	"Payment ID"
//	@Param   payment_status  body      dto.PaymentStatus  true  "Provide payment status object"
//	@Success	204
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *PaymentHandler) UpdateStatus(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		response := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	var status *dto.PaymentStatus
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
