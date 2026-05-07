package auth

import (
	"commerce/api/internal/auth"
	"encoding/json"
	"fmt"
	"net/http"

	middleware "github.com/auth0/go-jwt-middleware/v3"
	"github.com/auth0/go-jwt-middleware/v3/validator"
)

type AuthServiceI interface {
	SayHi(w http.ResponseWriter, r *http.Request)
	HasScope(r *http.Request, expected string) (bool, error)
}

type AuthService struct {
}

// HasScope implements [AuthServiceI].
func (a *AuthService) HasScope(r *http.Request, expected string) (bool, error) {
	claims, err := middleware.GetClaims[*validator.ValidatedClaims](r.Context())

	if err != nil {
		return false, err
	}

	customClaims, ok := claims.CustomClaims.(*auth.Claim)
	if !ok || !customClaims.HasScope(expected) {
		return false, fmt.Errorf("insufficient scope")
	}
	return true, nil
}

// SayHi implements [AuthServiceI].
func (a *AuthService) SayHi(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"message": "Hello from a commerce api! You need to be authenticated to see this.",
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func NewAuthService() AuthServiceI {
	return &AuthService{}
}
