package tax

import (
	"commerce/api/internal/services/tax"

	dto "commerce/api/internal/dto/tax"

	"github.com/gin-gonic/gin"
)

type TaxHandler struct {
	svc tax.TaxServiceI
}

func NewTaxHandler(svc tax.TaxServiceI) *TaxHandler {
	return &TaxHandler{svc: svc}
}

func (h *TaxHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/states", h.GetAll)
	rg.GET("/", h.GetStatesAndTaxes)
}

// GetStates godoc
//
//	@Summary        Prints the list of states
//	@Tags           tax
//	@Produce        json
//	@Success        200  {array}  string
//	@Router         /api/tax/states [get]
func (h *TaxHandler) GetAll(c *gin.Context) {
	states := h.svc.GetStates()
	c.JSON(200, states)
}

// GetStates godoc
//
//	@Summary        Prints the list of states and Taxes
//	@Tags           tax
//	@Produce        json
//	@Success        200  {array}  dto.Tax
//	@Router         /api/tax [get]
func (h *TaxHandler) GetStatesAndTaxes(c *gin.Context) {
	states := []dto.Tax{}
	states = h.svc.GetAll()
	c.JSON(200, states)
}
