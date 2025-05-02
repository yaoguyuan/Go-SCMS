package initializers

import (
	"os"

	"github.com/joho/godotenv"
)

var (
	SecretKey      string
	AvatarDir      string
	AvatarDefault  string
	LogFileDir     string
	LogFileDefault string
)

func LoadEnvVar() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
	SecretKey = os.Getenv("SECRET_KEY")
	AvatarDir = os.Getenv("AVATAR_DIR")
	AvatarDefault = os.Getenv("AVATAR_DEFAULT")
	LogFileDir = os.Getenv("LOG_FILE_DIR")
	LogFileDefault = os.Getenv("LOG_FILE_DEFAULT")
}
