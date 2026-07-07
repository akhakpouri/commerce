package models

import "time"

type Base struct {
	Id          uint      `gorm:"primaryKey;autoIncrement"`
	CreatedDate time.Time `gorm:"autoCreateTime"`
	UpdatedDate time.Time `gorm:"autoUpdateTime"`
	DeletedDate time.Time `gorm:"index"`
}
