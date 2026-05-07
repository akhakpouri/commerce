package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
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

func Gin(mw middleware.JWTMiddleware) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var passed bool

		adapter := mw.CheckJWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//pull claims out of context
			vc, _ := middleware.GetClaims[*validator.ValidatedClaims](r.Context())

			id := &Identity{
				Subject:   vc.RegisteredClaims.Subject,
				ExpiresAt: time.Unix(vc.RegisteredClaims.Expiry, 0),
			}
			if cc, ok := vc.CustomClaims.(*Claim); ok && cc != nil {
				id.Scope = strings.Fields(cc.Scope) //split on whitespace.
			}

			ctx.Set(constants.ContextKeys.Identity, id)

			//context carries the validated claims
			ctx.Request = r
		}))
		//run the middleware against gin's requst
		adapter.ServeHTTP(ctx.Writer, ctx.Request)

		if !passed {
			ctx.Abort()
		}
	}
}
