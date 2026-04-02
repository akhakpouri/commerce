package address

import (
	"commerce/api/internal/helpers"
	"commerce/api/internal/services/address"

	dto "commerce/api/internal/dto/address"
	err_dto "commerce/api/internal/dto/err"

	"github.com/gin-gonic/gin"
)

type AddressHandler struct {
	svc address.AddressServiceI
}

func NewAddressHandler(svc address.AddressServiceI) *AddressHandler {
	return &AddressHandler{svc: svc}
}

func (h *AddressHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/:id", h.GetById)
	rg.DELETE("/:id", h.Delete)
}

// GetAddress godoc
//
//	@Summary	Get the Address
//	@Tags		address
//	@Produce	json
//	@Param		id	path		int	true	"Address ID"
//	@Router		/api/address/{id} [get]
//	@Success	200	{object}	dto.Address
//	@Failure	400	{object}	err_dto.ErrorResponse
//	@Failure	500	{object}	err_dto.ErrorResponse
func (h *AddressHandler) GetById(c *gin.Context) {
	var address = &dto.Address{}
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		response := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	address, err = h.svc.GetById(*id)
	if err != nil {
		response := err_dto.ErrorResponse{Code: 404, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	c.JSON(200, address)
}

// DeleteAddress godoc
//
//	@Summary	Delete the address
//	@Tags		address
//	@Produce	json
//	@Param		id		path		int		true	"Address ID"
//	@Param		hard	query		bool	false	"Hard delete"
//	@Router		/api/address/{id} [delete]
//	@Success	204
//	@Failure	400	{object}	err_dto.ErrorResponse
//	@Failure	500	{object}	err_dto.ErrorResponse
func (h *AddressHandler) Delete(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		response := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	hard := c.DefaultQuery("hard", "false") == "true"
	err = h.svc.Delete(*id, hard)
	if err != nil {
		response := err_dto.ErrorResponse{Code: 404, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	c.JSON(204, nil)
}
