package auth

import (
	"net/http"

	"commerce/api/internal/constants"

	auth "commerce/api/internal/auth"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("whoami", h.WhoAmI)
}

// WhoAmI returns the authenticated caller's identity.
// @Summary      Identity of authenticated caller
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} authdto.WhoAmI
// @Failure      401 {object} errdto.ErrorResponse
// @Router       /api/auth/whoami [get]
func (h *AuthHandler) WhoAmI(c *gin.Context) {
	v, exists := c.Get(constants.ContextKeys.Identity)
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	id := v.(*auth.Identity)

	c.JSON(http.StatusOK, gin.H{
		"subject":    id.Subject,
		"scope":      id.Scopes,
		"expires_at": id.ExpiresAt,
	})
}
