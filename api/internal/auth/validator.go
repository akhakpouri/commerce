package auth

import (
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/auth0/go-jwt-middleware/v3/jwks"
	"github.com/auth0/go-jwt-middleware/v3/validator"
)

// Signature verification - Using Auth0’s public keys from JWKS
// Issuer validation - iss claim matches your Auth0 domain
// Audience validation - aud claim matches your API identifier
// Expiration check - Token hasn’t expired (exp claim)
// Time validity - Token is currently valid (nbf and iat claims)
func NewValidator(domain, audience string) (*validator.Validator, error) {
	issuer, err := url.Parse("https://" + domain + "/")
	if err != nil {
		slog.Error("failed to parse the url", "error", err, "domain", domain)
		return nil, fmt.Errorf("failed to parse the url %q %w", domain, err)
	}
	provider, err := jwks.NewCachingProvider(
		jwks.WithIssuerURL(issuer),
		jwks.WithCacheTTL(5*time.Minute),
	)
	if err != nil {
		slog.Error("failed to create a JWKS provider", "error", err)
		return nil, fmt.Errorf("failed to create a JWKS provider %q: %w", domain, err)
	}

	jwtValidator, err := validator.New(
		validator.WithKeyFunc(provider.KeyFunc),
		validator.WithAlgorithm(validator.RS256),
		validator.WithIssuer(issuer.String()),
		validator.WithAudience(audience),
		validator.WithCustomClaims(func() validator.CustomClaims {
			return &Claim{}
		}),
	)
	if err != nil {
		slog.Error("failed to create validator", "error", err)
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	return jwtValidator, nil
}
