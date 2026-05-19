package user

import (
	dto "commerce/api/internal/dto/user"
	repo "commerce/internal/shared/repositories/user"
	"errors"
	"log/slog"
)

type UserServiceI interface {
	Authenticate(email, password string) (*dto.User, error)
	GetAll() ([]*dto.User, error)
	GetById(id uint) (*dto.User, error)
	GetByEmail(email string) (*dto.User, error)
	ResolveByAuth(sub, email, firstName, lastName string) (*dto.User, error)
	Delete(id uint) error
	Save(user *dto.User) error
}

func NewUserService(repo repo.UserRepositoryI) UserServiceI {
	return &UserService{repo: repo}
}

type UserService struct {
	repo repo.UserRepositoryI
}

// ResolveByAuth implements [UserServiceI].
func (u *UserService) ResolveByAuth(sub string, email string, firstName string, lastName string) (*dto.User, error) {
	user, err := u.getByAuthSub(sub)
	if err == nil && user != nil {
		return user, nil
	}
	newUser := dto.User{
		AuthSub:   sub,
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}
	err = u.Save(&newUser)
	if err != nil {
		slog.Error("exception occured when saving the new user", "error", err)
		return nil, err
	}
	return &newUser, nil
}

func (u *UserService) getByAuthSub(sub string) (*dto.User, error) {
	user, err := u.repo.GetByAuthSub(sub)
	if err != nil {
		slog.Error("exception occured when retrieving user by auth sub", "error", err)
		return nil, err
	}
	return dto.FromModel(user), nil
}

// Authenticate implements [UserServiceI].
func (u *UserService) Authenticate(email string, password string) (*dto.User, error) {
	model, err := u.repo.GetByEmail(email)
	if err != nil {
		slog.Error("Exception occured retrieving user by email", "email", email, "error", err)
		return nil, err
	}
	if !model.CheckPassword(password) {
		return nil, errors.New("invalid credentials")
	}
	return dto.FromModel(model), nil
}

// GetAll implements [UserServiceI].
func (u *UserService) GetAll() ([]*dto.User, error) {
	users, err := u.repo.GetAll()
	if err != nil {
		slog.Error("Exception occured retrieving all of the users")
		return nil, err
	}
	dtos := []*dto.User{}

	for _, user := range users {
		dtos = append(dtos, dto.FromModel(user))
	}
	return dtos, nil
}

// Delete implements [UserServiceI].
func (u *UserService) Delete(id uint) error {
	return u.repo.Delete(id, false)
}

// GetByEmail implements [UserServiceI].
func (u *UserService) GetByEmail(email string) (*dto.User, error) {
	model, err := u.repo.GetByEmail(email)
	if err != nil {
		slog.Error("Exception occured retrieving user by email", "email", email, "error", err)
		return nil, errors.New("invalid credentials")
	}
	return dto.FromModel(model), nil
}

// GetById implements [UserServiceI].
func (u *UserService) GetById(id uint) (*dto.User, error) {
	model, err := u.repo.GetById(id)
	if err != nil {
		slog.Error("Exception occured retrieving user by id", "id", id, "error", err)
		return nil, err
	}
	return dto.FromModel(model), nil
}

// Save implements [UserServiceI].
func (u *UserService) Save(user *dto.User) error {
	model := dto.ToModel(user)
	return u.repo.Save(model)
}
