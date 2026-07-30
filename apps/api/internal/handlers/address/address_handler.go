package address

import (
	auth "commerce/api/internal/auth"
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
	rg.GET("/:id", auth.RequireScope(auth.Scopes.Users.Read), h.GetById)
	rg.POST("/", auth.RequireScope(auth.Scopes.Users.Write), h.Save)
	rg.DELETE("/:id", auth.RequireScope(auth.Scopes.Users.Write), h.Delete)
}

// GetAddress godoc
//
//	@Summary	Get the Address
//	@Tags		address
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"Address ID"
//	@Router		/api/address/{id} [get]
//	@Success	200	{object}	dto.Address
//	@Failure	400	{object}	err_dto.ErrorResponse
//	@Failure	500	{object}	err_dto.ErrorResponse
//	@Failure	401 {object}	err_dto.ErrorResponse
//	@Failure	403 {object}	err_dto.ErrorResponse
func (h *AddressHandler) GetById(c *gin.Context) {
	var address *dto.Address
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
//	@Security	BearerAuth
//	@Param		id		path		int		true	"Address ID"
//	@Param		hard	query		bool	false	"Hard delete"
//	@Router		/api/address/{id} [delete]
//	@Success	204
//	@Failure	400	{object}	err_dto.ErrorResponse
//	@Failure	500	{object}	err_dto.ErrorResponse
//	@Failure	401 {object}	err_dto.ErrorResponse
//	@Failure	403 {object}	err_dto.ErrorResponse
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

// SaveAddress godoc
//
//	@Summary	Save the address
//	@Tags		address
//	@Produce	json
//	@Security	BearerAuth
//	@Param		address	body		dto.Address	true	"Provide address object"
//	@Router		/api/address [post]
//	@Success	201	{object}	dto.Address
//	@Failure	400	{object}	err_dto.ErrorResponse
//	@Failure	500	{object}	err_dto.ErrorResponse
//	@Failure	401 {object}	err_dto.ErrorResponse
//	@Failure	403 {object}	err_dto.ErrorResponse
func (h *AddressHandler) Save(c *gin.Context) {
	var address *dto.Address
	if err := c.ShouldBindJSON(&address); err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	err := h.svc.Save(address)
	if err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	c.JSON(201, address)
}

// GetAddress godoc
//
//	@Summary	Get the list of addresses by user
//	@Tags		address
//	@Produce	json
//	@Security	BearerAuth
//	@Param		user_id		path		int		true	"user id"
//	@Router		/api/users/{user_id}/addresses [get]
//	@Success	200 {array} dto.Address
//	@Failure	400	{object}	err_dto.ErrorResponse
//	@Failure	500	{object}	err_dto.ErrorResponse
//	@Failure	400	{object}	err_dto.ErrorResponse
//	@Failure	404	{object}	err_dto.ErrorResponse
func (h *AddressHandler) GetByUserId(c *gin.Context) {
	userId, err := helpers.ParseParamToUint(c.Param("user_id"))
	if err != nil {
		response := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	var addresses []*dto.Address
	addresses, err = h.svc.GetAllByUserId(*userId)
	if err != nil {
		response := err_dto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	if len(addresses) == 0 {
		response := err_dto.ErrorResponse{Code: 404, Message: "No addresses found"}
		c.JSON(response.Code, response)
		return
	}
	c.JSON(200, addresses)

}
