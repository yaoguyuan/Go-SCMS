package models

type User struct {
	// gorm.Model
	ID       uint   `gorm:"primaryKey" redis:"id"`
	Email    string `gorm:"unique" redis:"email"`
	Password string `redis:"password"`
	Address  string `redis:"address"`
	Avatar   string `gorm:"default:'default_avatar.png'" redis:"avatar"`
	Role     string `gorm:"default:'user'" redis:"role"`
}

func (User) TableName() string {
	return "users"
}
