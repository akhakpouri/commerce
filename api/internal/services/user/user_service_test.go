package user

import (
	dto "commerce/api/internal/dto/user"
	"commerce/internal/shared/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthenticate(t *testing.T) {
	password := "hashed_password"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	mockRepo := NewMockUserRepositoryI(ctl)
	mockRepo.EXPECT().GetByEmail("jon.doe@example.com").Return(&models.User{
		Base: models.Base{
			Id:          1,
			CreatedDate: time.Now(),
			UpdatedDate: time.Now(),
		},
		Email:     "jon.doe@example.com",
		Password:  string(hashedPassword),
		FirstName: "Jon",
		LastName:  "Doe",
	}, nil)
	svc := NewUserService(mockRepo)
	user, err := svc.Authenticate("jon.doe@example.com", password)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "jon.doe@example.com", user.Email)
	assert.Equal(t, "Jon", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
}

func TestInvalidAuthentication(t *testing.T) {
	password, wrong_password := "hashed_password", "wrong_password"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockRepo := NewMockUserRepositoryI(ctl)
	mockRepo.EXPECT().GetByEmail("jon.doe@example.com").Return(&models.User{
		Base: models.Base{
			Id:          1,
			CreatedDate: time.Now(),
			UpdatedDate: time.Now(),
		},
		Email:     "jon.doe@example.com",
		Password:  string(hashedPassword),
		FirstName: "Jon",
		LastName:  "Doe",
	}, nil)

	svc := NewUserService(mockRepo)
	user, err := svc.Authenticate("jon.doe@example.com", wrong_password)
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestGetById(t *testing.T) {
	id := uint(1)
	ctl := gomock.NewController(t)
	mockRepo := NewMockUserRepositoryI(ctl)
	defer ctl.Finish()
	mockRepo.EXPECT().GetById(id).Return(&models.User{
		Base: models.Base{
			Id:          id,
			CreatedDate: time.Now(),
			UpdatedDate: time.Now(),
		},
		Email:     "jon.doe@example.com",
		FirstName: "Jon",
		LastName:  "Doe",
	}, nil)
	svc := NewUserService(mockRepo)
	user, err := svc.GetById(1)
	assert.NotNil(t, user)
	assert.NoError(t, err)
}

func TestGetAll(t *testing.T) {
	ctl := gomock.NewController(t)
	mockRepo := NewMockUserRepositoryI(ctl)
	defer ctl.Finish()
	mockRepo.EXPECT().GetAll().Return([]*models.User{
		{
			Base:      models.Base{Id: 1, CreatedDate: time.Now(), UpdatedDate: time.Now()},
			FirstName: "Jon",
			LastName:  "Doe",
		},
		{
			Base:      models.Base{Id: 2, CreatedDate: time.Now(), UpdatedDate: time.Now()},
			FirstName: "Jane",
			LastName:  "Doe",
		},
		{
			Base:      models.Base{Id: 3, CreatedDate: time.Now(), UpdatedDate: time.Now()},
			FirstName: "Joe",
			LastName:  "Smith",
		},
	}, nil)
	svc := NewUserService(mockRepo)
	users, err := svc.GetAll()
	assert.NoError(t, err)
	assert.NotEmpty(t, users)
}

func TestSave(t *testing.T) {
	ctl := gomock.NewController(t)
	mockRepo := NewMockUserRepositoryI(ctl)
	defer ctl.Finish()
	mockRepo.EXPECT().Save(gomock.Any()).Return(nil)
	svc := NewUserService(mockRepo)
	err := svc.Save(&dto.User{
		FirstName: "Jon",
		LastName:  "Doe",
		Email:     "jon.doe@example.com",
		Password:  "secret",
	})
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	id := uint(1)
	ctl := gomock.NewController(t)
	mockRepo := NewMockUserRepositoryI(ctl)
	defer ctl.Finish()
	mockRepo.EXPECT().Delete(id, false).Return(nil)
	svc := NewUserService(mockRepo)
	err := svc.Delete(id)
	assert.NoError(t, err)
}

