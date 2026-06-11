package user

import (
	dto "commerce/api/internal/dto/user"
	"commerce/internal/shared/models"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Hit — existing user found, no insert.
func TestResolveByAuth_ExistingUser_NoSave(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockRepo := NewMockUserRepositoryI(ctl)

	mockRepo.EXPECT().GetByAuthSub("auth0|abc123").Return(&models.User{
		Base: models.Base{
			Id:          7,
			CreatedDate: time.Now(),
			UpdatedDate: time.Now(),
		},
		AuthSub:   "auth0|abc123",
		Email:     "ali@example.com",
		FirstName: "Ali",
		LastName:  "Khakpouri",
	}, nil)
	// Save must NOT be called — gomock will fail the test if it is.

	svc := NewUserService(mockRepo)
	user, err := svc.ResolveByAuth("auth0|abc123", "ali@example.com", "Ali", "Khakpouri")

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, uint(7), user.Id)
	assert.Equal(t, "ali@example.com", user.Email)
	assert.Equal(t, "Ali", user.FirstName)
	assert.Equal(t, "Khakpouri", user.LastName)
}

// Miss — repo returns not-found, service builds a new user and saves it.
// DoAndReturn captures the model and simulates GORM populating Id on insert.
func TestResolveByAuth_NewUser_SavesWithClaimFields(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockRepo := NewMockUserRepositoryI(ctl)

	mockRepo.EXPECT().
		GetByAuthSub("auth0|new123").
		Return(nil, errors.New("record not found"))

	mockRepo.EXPECT().
		Save(gomock.Any()).
		DoAndReturn(func(m *models.User) error {
			// Assert the model was built from claim fields.
			assert.Equal(t, "auth0|new123", m.AuthSub)
			assert.Equal(t, "ali@example.com", m.Email)
			assert.Equal(t, "Ali", m.FirstName)
			assert.Equal(t, "Khakpouri", m.LastName)
			assert.Equal(t, uint(0), m.Id, "Id must be zero before insert")
			// Simulate GORM populating the primary key on insert.
			m.Id = 99
			return nil
		})

	svc := NewUserService(mockRepo)
	user, err := svc.ResolveByAuth("auth0|new123", "ali@example.com", "Ali", "Khakpouri")

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, uint(99), user.Id, "service must surface the GORM-populated Id")
	assert.Equal(t, "ali@example.com", user.Email)
	assert.Equal(t, "Ali", user.FirstName)
	assert.Equal(t, "Khakpouri", user.LastName)
	assert.Equal(t, "auth0|new123", user.AuthSub)
}

// Save error + concurrent winner — repo.Save hits the unique constraint
// (a concurrent request inserted the same brand-new sub first). The service
// re-SELECTs, finds the winner's row, and returns it without error.
func TestResolveByAuth_SaveError_ReSelectFindsExisting_ReturnsUser(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockRepo := NewMockUserRepositoryI(ctl)

	// Call #1: initial lookup misses.
	mockRepo.EXPECT().
		GetByAuthSub("auth0|new123").
		Return(nil, errors.New("record not found"))
	// Save loses the race against the concurrent inserter.
	mockRepo.EXPECT().
		Save(gomock.Any()).
		Return(errors.New("duplicate key violates unique constraint"))
	// Call #2: re-SELECT now finds the winner's row.
	mockRepo.EXPECT().
		GetByAuthSub("auth0|new123").
		Return(&models.User{
			Base:      models.Base{Id: 42},
			AuthSub:   "auth0|new123",
			Email:     "ali@example.com",
			FirstName: "Ali",
			LastName:  "Khakpouri",
		}, nil)

	svc := NewUserService(mockRepo)
	user, err := svc.ResolveByAuth("auth0|new123", "ali@example.com", "Ali", "Khakpouri")

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, uint(42), user.Id, "service must surface the concurrent winner's row")
	assert.Equal(t, "auth0|new123", user.AuthSub)
}

// Save error + re-SELECT also fails — the Save failure is not a recoverable
// race (or the row is still absent). The service propagates the original
// Save error and returns no user.
func TestResolveByAuth_SaveError_ReSelectAlsoFails_PropagatesAndReturnsNil(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockRepo := NewMockUserRepositoryI(ctl)

	// Call #1: initial lookup misses.
	mockRepo.EXPECT().
		GetByAuthSub("auth0|new123").
		Return(nil, errors.New("record not found"))
	mockRepo.EXPECT().
		Save(gomock.Any()).
		Return(errors.New("duplicate key violates unique constraint"))
	// Call #2: re-SELECT also fails — nothing to recover, original error wins.
	mockRepo.EXPECT().
		GetByAuthSub("auth0|new123").
		Return(nil, errors.New("record not found"))

	svc := NewUserService(mockRepo)
	user, err := svc.ResolveByAuth("auth0|new123", "ali@example.com", "Ali", "Khakpouri")

	require.Error(t, err)
	assert.Nil(t, user)
}
