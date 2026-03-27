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
	rg.GET("/", func(c *gin.Context) {
		states := h.svc.GetStates()
		c.JSON(200, states)
	})
}
