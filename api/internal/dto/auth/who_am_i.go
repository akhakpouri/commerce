package auth

import "time"

type WhoAmI struct {
	Subject   string    `json:"subject"`
	Scope     []string  `json:"scope"`
	ExpiresAt time.Time `json:"expires_at"`
}
