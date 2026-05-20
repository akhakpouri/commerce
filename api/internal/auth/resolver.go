package auth

import (
	"commerce/api/internal/constants"
	errdto "commerce/api/internal/dto/err"
	userService "commerce/api/internal/services/user"
	"log/slog"
	"net/http"
	"strings"

	middleware "github.com/auth0/go-jwt-middleware/v3"
	"github.com/auth0/go-jwt-middleware/v3/validator"
	"github.com/gin-gonic/gin"
)

func ResolveIdentity(svc userService.UserServiceI) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		v, exists := ctx.Get(constants.ContextKeys.Identity)
		if !exists {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		id := v.(*Identity)

		//if M2M then skip
		if strings.HasSuffix(id.Subject, "@clients") {
			return
		}

		//pull claims out of context
		vc, err := middleware.GetClaims[*validator.ValidatedClaims](ctx.Request.Context())

		if err != nil || vc == nil {
			slog.Error("resolver: no validated claims in context", "error", err)
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		cc, _ := vc.CustomClaims.(*Claim)

		if cc == nil || cc.Email == "" {
			slog.Error("email in the claim is empty.")
			ctx.AbortWithStatusJSON(
				http.StatusUnauthorized,
				errdto.ErrorResponse{
					Code:    http.StatusUnauthorized,
					Message: "non-M2M token missing required token"})
			return
		}

		u, err := svc.ResolveByAuth(id.Subject, cc.Email, cc.FirstName, cc.LastName)
		if err != nil {
			slog.Error("resolver: failed to resolve user", "sub", id.Subject, "error", err)
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		id.UserId = &u.Id
	}
}
