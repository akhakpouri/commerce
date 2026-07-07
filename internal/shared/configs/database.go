package configs

import (
	db "github.com/akhakpouri/gorm-kit/database"
	pg "github.com/akhakpouri/gorm-kit/pg"
	"gorm.io/gorm"
)

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DbName   string
	SSLMode  string
	Schema   string
}

func (d *DatabaseConfig) Connect() (*gorm.DB, error) {
	return pg.Connect(db.DbConfig{
		Host:     d.Host,
		Port:     d.Port,
		User:     d.User,
		Password: d.Password,
		DbName:   d.DbName,
		SSLMode:  d.SSLMode,
		Schema:   d.Schema,
	})
}
