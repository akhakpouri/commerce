package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"commerce/api/internal/constants"
	errdto "commerce/api/internal/dto/err"

	middleware "github.com/auth0/go-jwt-middleware/v3"
	"github.com/auth0/go-jwt-middleware/v3/validator"
	"github.com/gin-gonic/gin"
)

func NewMiddleware(jwtValidator *validator.Validator) (*middleware.JWTMiddleware, error) {

	return middleware.New(
		middleware.WithValidator(jwtValidator),
		middleware.WithValidateOnOptions(false),
		middleware.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
			errDto := errdto.ErrorResponse{Code: http.StatusUnauthorized, Message: "Failed to validate JWT."}
			slog.Error("JWT validation failed", "error", err, "path", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(errDto)
		}),
	)
}

func Gin(mw *middleware.JWTMiddleware) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var passed bool

		adapter := mw.CheckJWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//pull claims out of context
			vc, err := middleware.GetClaims[*validator.ValidatedClaims](r.Context())

			if err != nil {
				slog.Error("failed to get claims from request context", "error", err)
				return
			}

			id := &Identity{
				Subject:   vc.RegisteredClaims.Subject,
				ExpiresAt: time.Unix(vc.RegisteredClaims.Expiry, 0),
			}
			if cc, ok := vc.CustomClaims.(*Claim); ok && cc != nil {
				id.Scopes = strings.Fields(cc.Scope) //split on whitespace.
			}

			ctx.Set(constants.ContextKeys.Identity, id)

			//context carries the validated claims
			ctx.Request = r
			passed = true
		}))
		//run the middleware against gin's requst
		adapter.ServeHTTP(ctx.Writer, ctx.Request)

		if !passed {
			ctx.Abort()
		}
	}
}

func RequireScope(expected string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		v, exists := ctx.Get(constants.ContextKeys.Identity)
		if !exists {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		id := v.(*Identity)
		if !slices.Contains(id.Scopes, expected) {
			ctx.AbortWithStatusJSON(http.StatusForbidden, errdto.ErrorResponse{
				Code:    http.StatusForbidden,
				Message: "insufficient scope",
			})
			return
		}
		ctx.Next()
	}
}
