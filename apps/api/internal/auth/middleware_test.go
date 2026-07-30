package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"commerce/internal/shared/constants"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v3"
	"github.com/auth0/go-jwt-middleware/v3/validator"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer   = "https://test-issuer/"
	testAudience = "urn:test-api"
	testSubject  = "auth0|abc123"
	testSecret   = "test-secret"
)

func signHS256(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	body := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write([]byte(body))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return body + "." + sig
}

func newTestMiddleware(t *testing.T) *jwtmiddleware.JWTMiddleware {
	t.Helper()
	v, err := validator.New(
		validator.WithKeyFunc(func(context.Context) (any, error) {
			return []byte(testSecret), nil
		}),
		validator.WithAlgorithm(validator.HS256),
		validator.WithIssuer(testIssuer),
		validator.WithAudience(testAudience),
		validator.WithCustomClaims(func() validator.CustomClaims {
			return &Claim{}
		}),
	)
	require.NoError(t, err)

	mw, err := NewMiddleware(v)
	require.NoError(t, err)
	return mw
}

func newAuthedRouter(t *testing.T, after ...gin.HandlerFunc) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlers := append([]gin.HandlerFunc{Gin(newTestMiddleware(t))}, after...)
	handlers = append(handlers, func(c *gin.Context) {
		v, _ := c.Get(constants.ContextKeys.Identity)
		id, _ := v.(*Identity)
		c.JSON(http.StatusOK, gin.H{"sub": id.Subject, "scopes": id.Scopes})
	})
	r.GET("/protected", handlers...)
	return r
}

func TestGin_MissingBearer_Returns401(t *testing.T) {
	r := newAuthedRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGin_InvalidSignature_Returns401(t *testing.T) {
	r := newAuthedRouter(t)
	token := signHS256(t, map[string]any{
		"iss": testIssuer,
		"aud": testAudience,
		"sub": testSubject,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	// tamper with signature
	tampered := token[:len(token)-4] + "AAAA"

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tampered)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGin_WrongIssuer_Returns401(t *testing.T) {
	r := newAuthedRouter(t)
	token := signHS256(t, map[string]any{
		"iss": "https://attacker/",
		"aud": testAudience,
		"sub": testSubject,
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGin_ExpiredToken_Returns401(t *testing.T) {
	r := newAuthedRouter(t)
	token := signHS256(t, map[string]any{
		"iss": testIssuer,
		"aud": testAudience,
		"sub": testSubject,
		"exp": time.Now().Add(-time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGin_ValidToken_PopulatesIdentity(t *testing.T) {
	r := newAuthedRouter(t)
	exp := time.Now().Add(time.Hour).Unix()
	token := signHS256(t, map[string]any{
		"iss":   testIssuer,
		"aud":   testAudience,
		"sub":   testSubject,
		"exp":   exp,
		"scope": "orders:read orders:write",
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Sub    string   `json:"sub"`
		Scopes []string `json:"scopes"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, testSubject, body.Sub)
	assert.Equal(t, []string{"orders:read", "orders:write"}, body.Scopes)
}

func TestGin_ValidToken_NoScopeClaim(t *testing.T) {
	r := newAuthedRouter(t)
	token := signHS256(t, map[string]any{
		"iss": testIssuer,
		"aud": testAudience,
		"sub": testSubject,
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Sub    string   `json:"sub"`
		Scopes []string `json:"scopes"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, testSubject, body.Sub)
	assert.Empty(t, body.Scopes)
}

func TestRequireScope_NoIdentity_Returns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", RequireScope("orders:read"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireScope_MissingScope_Returns403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x",
		func(c *gin.Context) {
			c.Set(constants.ContextKeys.Identity, &Identity{
				Subject: testSubject,
				Scopes:  []string{"orders:read"},
			})
		},
		RequireScope("orders:write"),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireScope_HasScope_PassesThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	called := false
	r.GET("/x",
		func(c *gin.Context) {
			c.Set(constants.ContextKeys.Identity, &Identity{
				Subject: testSubject,
				Scopes:  []string{"orders:read", "orders:write"},
			})
		},
		RequireScope("orders:write"),
		func(c *gin.Context) {
			called = true
			c.Status(http.StatusNoContent)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, called)
}
