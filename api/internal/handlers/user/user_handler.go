package user

import (
	"commerce/api/internal/helpers"
	"commerce/api/internal/services/user"

	err_dto "commerce/api/internal/dto/err"
	dto "commerce/api/internal/dto/user"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc user.UserServiceI
}

func NewUserHandler(svc user.UserServiceI) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/{id}", h.GetById)
}

// GetUser godoc
//
//	@Summary	Get the user
//	@Tags		user
//	@Produce	json
//	@Router		/api/user/{id} [get]
//	@Success	200 {object} dto.User
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *UserHandler) GetById(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		response := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	var usr *dto.User
	usr, err = h.svc.GetById(*id)
	if err != nil {
		response := err_dto.ErrorResponse{Code: 404, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	c.JSON(200, usr)
}

// GetUser godoc
//
//	@Summary	Get the user
//	@Tags		user
//	@Produce	json
//	@Router		/api/user/authenticate [post]
//	@Param	authenticate  body      dto.Authenticate  true  "Provide authenticate object"
//	@Success	200 {object} dto.Authenticat
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *UserHandler) Authenticate(c *gin.Context) {

}
