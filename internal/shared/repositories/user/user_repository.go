package user

import (
	"commerce/internal/shared/models"
	"time"

	"gorm.io/gorm"
)

type UserRepositoryI interface {
	GetById(id uint) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByAuthSub(sub string) (*models.User, error)
	GetAll() ([]*models.User, error)
	Save(user *models.User) error
	Delete(id uint, hard bool) error
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepositoryI {
	return &UserRepository{db: db}
}

// GetByAuthSub implements [UserRepositoryI].
func (u *UserRepository) GetByAuthSub(sub string) (*models.User, error) {
	var user models.User
	if err := u.db.Where("auth_sub = ?", sub).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Delete implements [UserRepositoryI].
func (u *UserRepository) Delete(id uint, hard bool) error {
	if hard {
		return u.db.Delete(&models.User{}, id).Error
	}
	var user models.User
	if err := u.db.First(&user, id).Error; err != nil {
		return err
	}
	user.DeletedDate = time.Now()
	return u.db.Save(&user).Error
}

// GetAll implements [UserRepositoryI].
func (u *UserRepository) GetAll() ([]*models.User, error) {
	var users []*models.User
	if err := u.db.Order("created_date desc").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetByEmail implements [UserRepositoryI].
func (u *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	if err := u.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetById implements [UserRepositoryI].
func (u *UserRepository) GetById(id uint) (*models.User, error) {
	var user models.User
	if err := u.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Save implements [UserRepositoryI].
func (u *UserRepository) Save(user *models.User) error {
	if user.Id == 0 {
		return u.db.Create(user).Error
	} else {
		if _, err := u.GetById(user.Id); err != nil {
			return err
		}
		return u.db.Save(user).Error
	}
}
