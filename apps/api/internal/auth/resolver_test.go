package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	userdto "commerce/api/internal/dto/user"
	"commerce/internal/shared/constants"

	"github.com/auth0/go-jwt-middleware/v3/core"
	"github.com/auth0/go-jwt-middleware/v3/validator"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// runResolverTest builds a Gin router that:
//  1. simulates the upstream Gin() middleware by setting Identity in context (if non-nil)
//  2. chains ResolveIdentity(svc)
//  3. terminates in a sink handler that 200s
//
// Tests assert on the response recorder + observe mutations to the Identity pointer.
func runResolverTest(t *testing.T, id *Identity, claims *Claim, svc *MockUserServiceI) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x",
		func(c *gin.Context) {
			if id != nil {
				c.Set(constants.ContextKeys.Identity, id)
			}
		},
		ResolveIdentity(svc),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if claims != nil {
		vc := &validator.ValidatedClaims{
			RegisteredClaims: validator.RegisteredClaims{Subject: id.Subject},
			CustomClaims:     claims,
		}
		req = req.WithContext(core.SetClaims(req.Context(), vc))
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// 1. No Identity in context — programming error; Gin() didn't run upstream.
func TestResolveIdentity_NoIdentity_Returns401(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := NewMockUserServiceI(ctrl)
	// svc must not be called — gomock will fail the test if it is

	w := runResolverTest(t, nil, nil, svc)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 2. M2M token (sub ends with "@clients") — short-circuit, no DB work, UserId stays zero.
func TestResolveIdentity_M2MSub_SkipsLookup(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := NewMockUserServiceI(ctrl)
	// no EXPECT calls — service must not be invoked

	id := &Identity{Subject: "abc123@clients"}
	w := runResolverTest(t, id, nil, svc)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, id.UserId)
}

// 3. Non-M2M token with empty email claim — refuse rather than half-populate a row.
func TestResolveIdentity_MissingEmail_Returns401(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := NewMockUserServiceI(ctrl)
	// service must not be called

	id := &Identity{Subject: "auth0|abc123"}
	claims := &Claim{Scope: "products:read"} // Email is empty
	w := runResolverTest(t, id, claims, svc)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "non-M2M token missing required")
	assert.Nil(t, id.UserId)
}

// 4. Happy path — claims complete, service resolves a row, UserId stamped on Identity.
func TestResolveIdentity_HappyPath_SetsUserId(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := NewMockUserServiceI(ctrl)
	svc.EXPECT().
		ResolveByAuth("auth0|abc123", "ali@example.com", "Ali", "Khakpouri").
		Return(&userdto.User{
			Id:        42,
			Email:     "ali@example.com",
			FirstName: "Ali",
			LastName:  "Khakpouri",
			AuthSub:   "auth0|abc123",
		}, nil)

	id := &Identity{Subject: "auth0|abc123"}
	claims := &Claim{
		Scope:     "products:read",
		Email:     "ali@example.com",
		FirstName: "Ali",
		LastName:  "Khakpouri",
	}
	w := runResolverTest(t, id, claims, svc)

	require.NotNil(t, id.UserId)
	assert.Equal(t, uint(42), *id.UserId)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, id.UserId)
	assert.Equal(t, uint(42), *id.UserId)
}

// 5. Service-layer error (DB down, etc.) propagates as 500.
func TestResolveIdentity_ServiceError_Returns500(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := NewMockUserServiceI(ctrl)
	svc.EXPECT().
		ResolveByAuth("auth0|abc123", "ali@example.com", "Ali", "Khakpouri").
		Return(nil, errors.New("db down"))

	id := &Identity{Subject: "auth0|abc123"}
	claims := &Claim{
		Scope:     "products:read",
		Email:     "ali@example.com",
		FirstName: "Ali",
		LastName:  "Khakpouri",
	}
	w := runResolverTest(t, id, claims, svc)
	assert.Nil(t, id.UserId)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Nil(t, id.UserId)
}

// 6. Identity set but no claims in request context — Gin() middleware was skipped or
// failed silently upstream. Resolver should refuse, not panic. (This test will FAIL
// until the nil-check fix described above is in resolver.go.)
func TestResolveIdentity_NoClaimsContext_Returns401(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := NewMockUserServiceI(ctrl)
	// service must not be called

	id := &Identity{Subject: "auth0|abc123"}
	w := runResolverTest(t, id, nil, svc) // claims=nil → no SetClaims call

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Nil(t, id.UserId) // for m2m skip + reject cases``
}
