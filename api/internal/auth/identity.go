package auth

import "time"

type Identity struct {
	Subject   string
	Scopes    []string //parsed from `scope` claims
	ExpiresAt time.Time
	UserId    *uint
}
