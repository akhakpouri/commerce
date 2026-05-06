package auth

import (
	"encoding/json"
	"net/http"

	"commerce/api/internal/auth"
	errdto "commerce/api/internal/dto/err"

	middleware "github.com/auth0/go-jwt-middleware/v3"
	"github.com/auth0/go-jwt-middleware/v3/validator"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"message": "Hello from a commerce api! You need to be authenticated to see this.",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func ScopeHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := middleware.GetClaims[*validator.ValidatedClaims](r.Context())
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		respnse := errdto.ErrorResponse{Code: 401, Message: "UnAuthorized"}
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(respnse)
		return
	}

	customClaims, ok := claims.CustomClaims.(*auth.Claim)
	if !ok || !customClaims.HasScope("read:messages") {
		respnse := errdto.ErrorResponse{Code: 403, Message: "Insufficient scope"}

		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(respnse)
		return
	}

	respnse := errdto.ErrorResponse{
		Code:    200,
		Message: "Hello from a commerce api! You need to be authenticated and have a scope of read:messages to see this",
	}

	_ = json.NewEncoder(w).Encode(respnse)
}
