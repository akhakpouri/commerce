package user

type Authenticate struct {
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
}
