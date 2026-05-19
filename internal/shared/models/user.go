package models

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	Base
	FirstName string    `gorm:"not null;size:100"`
	LastName  string    `gorm:"not null;size:100"`
	Email     string    `gorm:"unique;size:250"`
	Password  string    `gorm:"not null"`
	Addresses []Address `gorm:"foreignKey:UserId;constraint:OnDelete:CASCADE"`
	Orders    []Order   `gorm:"foreignKey:UserId;constraint:OnDelete:CASCADE"`
	Reviews   []Review  `gorm:"foreignKey:UserId;constraint:OnDelete:CASCADE"`
	AuthSub   string    `gorm:"unique;size:250"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Password == "" {
		return errors.New("password is required")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(u.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) BeforeUpdate(tx *gorm.DB) error {
	if tx.Statement.Changed("Password") {
		hashedPassword, err := bcrypt.GenerateFromPassword(
			[]byte(u.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

func (User) TableName() string {
	return "users"
}
