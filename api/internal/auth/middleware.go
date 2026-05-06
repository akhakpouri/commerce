package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	errdto "commerce/api/internal/dto/err"

	middleware "github.com/auth0/go-jwt-middleware/v3"
	"github.com/auth0/go-jwt-middleware/v3/validator"
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
