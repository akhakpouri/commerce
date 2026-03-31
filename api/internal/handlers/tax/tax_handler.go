package tax

import (
	"commerce/api/internal/services/tax"

	"github.com/gin-gonic/gin"
)

type TaxHandler struct {
	svc tax.TaxServiceI
}

func NewTaxHandler(svc tax.TaxServiceI) *TaxHandler {
	return &TaxHandler{svc: svc}
}

func (h *TaxHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/", h.GetAll)
}

// GetStates godoc
//
//	@Summary        Prints the list of states
//	@Tags           States
//	@Produce        json
//	@Success        200  {array}  string
//	@Router         /api/taxes [get]
func (h *TaxHandler) GetAll(c *gin.Context) {
	states := h.svc.GetStates()
	c.JSON(200, states)
}
