package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

type Claim struct {
	Scope string `json:"scope"`
}

func (c *Claim) Validate(ctx context.Context) error {
	if c.Scope == "" {
		return nil
	}
	if strings.TrimSpace(c.Scope) != c.Scope {
		slog.Error("scope claims has invalid whitespaces")
		return fmt.Errorf("scope claims has invalid whitespaces")
	}

	if strings.Contains(c.Scope, "  ") {
		slog.Error("scope claim contains double spaces")
		return fmt.Errorf("scope claim contains double spaces")
	}

	return nil
}

func (c *Claim) HasScope(expected string) bool {
	if c.Scope == "" {
		return false
	}
	scopes := strings.SplitSeq(c.Scope, " ")
	for scope := range scopes {
		if scope == expected {
			return true
		}
	}
	return false
}
