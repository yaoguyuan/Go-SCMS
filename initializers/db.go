package initializers

import (
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"auth/models"
)

var DB *gorm.DB

func ConnectToDB() {
	dsn := os.Getenv("DSN")
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database: " + err.Error())
	}
}

func SyncDB() {
	err := DB.AutoMigrate(&models.User{}, &models.Article{}, &models.Comment{})
	if err != nil {
		panic("Failed to synchronize database: " + err.Error())
	}
}
