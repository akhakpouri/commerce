package user

import "commerce/internal/shared/models"

type User struct {
	Id        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password,omitempty"`
	AuthSub   string `json:"-"`
}

func FromModel(user *models.User) *User {
	return &User{
		Id:        user.Id,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		AuthSub:   user.AuthSub,
	}
}

func ToModel(user *User) *models.User {
	return &models.User{
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Password:  user.Password,
		AuthSub:   user.AuthSub,
	}
}
