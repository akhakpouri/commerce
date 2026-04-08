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
	rg.GET("/:id", h.GetById)
	rg.GET("/", h.GetAll)
	rg.POST("/authenticate", h.Authenticate)
	rg.GET("/email/:email", h.GetByEmail)
	rg.DELETE("/:id", h.Delete)
	rg.POST("/", h.Save)
}

// GetUser godoc
//
//	@Summary	Get the user
//	@Tags		user
//	@Produce	json
//	@Router		/api/user/{id} [get]
//	@Param		id	path	int	true	"User Id"
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
//	@Summary	Get all of the user
//	@Tags		user
//	@Produce	json
//	@Router		/api/user [get]
//	@Success	200 {array} dto.User
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *UserHandler) GetAll(c *gin.Context) {
	var users []*dto.User

	users, err := h.svc.GetAll()
	if err != nil {
		response := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(response.Code, response)
		return
	}
	c.JSON(200, users)
}

// GetUser godoc
//
//	@Summary	Get the user
//	@Tags		user
//	@Produce	json
//	@Router		/api/user/authenticate [post]
//	@Param	authenticate  body      dto.Authenticate  true  "Provide authenticate object"
//	@Success	204 {object} nil
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *UserHandler) Authenticate(c *gin.Context) {
	var auth *dto.Authenticate
	if err := c.ShouldBindJSON(&auth); err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(400, errorResponse)
		return
	}
	_, err := h.svc.Authenticate(auth.Email, auth.Password)
	if err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	c.JSON(204, nil)
}

// GetUser godoc
//
//	@Summary	Get the user by email address
//	@Tags		user
//	@Produce	json
//	@Router		/api/user/email/{email} [get]
//	@Param		email	path	string	true	"Email Address"
//	@Success	204 {object} nil
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *UserHandler) GetByEmail(c *gin.Context) {
	email := c.Param("email")
	_, err := h.svc.GetByEmail(email)
	if err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 404, Message: err.Error()}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	c.JSON(204, nil)
}

// Deleteuser godoc
//
//	@Summary	Delete the user
//	@Tags		user
//	@Produce	json
//	@Router		/api/user/{id} [delete]
//	@Param		id	path	int	true	"User Id"
//	@Success	204
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := helpers.ParseParamToUint(c.Param("id"))
	if err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 400, Message: "invalid id"}
		c.JSON(errorResponse.Code, errorResponse)
		return
	}
	err = h.svc.Delete(*id)
	if err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.JSON(204, nil)
}

// Saveuser godoc
//
//	@Summary	Save the user
//	@Tags		user
//	@Produce	json
//	@Router		/api/user [post]
//	@Param   user  body      dto.User  true  "Provide user object"
//	@Success	201 {object} dto.User
//	@Failure	400 {object} err_dto.ErrorResponse
//	@Failure	500 {object} err_dto.ErrorResponse
func (h *UserHandler) Save(c *gin.Context) {
	var user *dto.User
	if err := c.ShouldBindJSON(&user); err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 400, Message: err.Error()}
		c.JSON(400, errorResponse)
		return
	}
	err := h.svc.Save(user)
	if err != nil {
		errorResponse := err_dto.ErrorResponse{Code: 500, Message: err.Error()}
		c.JSON(500, errorResponse)
		return
	}
	c.JSON(201, user)
}
