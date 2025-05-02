package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email    string `gorm:"unique"`
	Password string
	Address  string
	Avatar   string `gorm:"default:'default_avatar.png'"`
	Role     string `gorm:"default:'user'"`
}

func (User) TableName() string {
	return "users"
}
