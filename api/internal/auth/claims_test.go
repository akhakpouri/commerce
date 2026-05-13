package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClaim_Validate(t *testing.T) {
	cases := []struct {
		name    string
		scope   string
		wantErr bool
	}{
		{"empty scope is valid", "", false},
		{"single scope is valid", "orders:read", false},
		{"multiple scopes are valid", "orders:read orders:write", false},
		{"leading whitespace is rejected", " orders:read", true},
		{"trailing whitespace is rejected", "orders:read ", true},
		{"double space is rejected", "orders:read  orders:write", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Claim{Scope: tc.scope}
			err := c.Validate(context.Background())
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClaim_HasScope(t *testing.T) {
	cases := []struct {
		name     string
		scope    string
		expected string
		want     bool
	}{
		{"empty scope never has anything", "", "orders:read", false},
		{"matches single scope", "orders:read", "orders:read", true},
		{"matches one of many scopes", "orders:read orders:write", "orders:write", true},
		{"non-match returns false", "orders:read", "orders:write", false},
		{"no partial prefix match", "orders:read", "orders", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Claim{Scope: tc.scope}
			assert.Equal(t, tc.want, c.HasScope(tc.expected))
		})
	}
}
