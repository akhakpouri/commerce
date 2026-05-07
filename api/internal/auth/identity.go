package auth

import "time"

type Identity struct {
	Subject   string
	Scope     []string //parsed from `scope` claims
	ExpiresAt time.Time
}
