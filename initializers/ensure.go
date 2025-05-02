package initializers

import (
	"os"
	"path/filepath"
)

func EnsureAvatarDefault() {
	// Check if the avatar directory exists
	if _, err := os.Stat(AvatarDir); os.IsNotExist(err) {
		panic("Avatar directory does not exist")
	}
	// Check if the default avatar exists
	defaultAvatarPath := filepath.Join(AvatarDir, AvatarDefault)
	if _, err := os.Stat(defaultAvatarPath); os.IsNotExist(err) {
		panic("Default avatar does not exist")
	}
}

func EnsureLogFileDefault() {
	// Check if the log file directory exists
	if _, err := os.Stat(LogFileDir); os.IsNotExist(err) {
		panic("Log file directory does not exist")
	}
	// Check if the default log file exists
	defaultLogFilePath := filepath.Join(LogFileDir, LogFileDefault)
	if _, err := os.Stat(defaultLogFilePath); os.IsNotExist(err) {
		panic("Default log file does not exist")
	}
}
