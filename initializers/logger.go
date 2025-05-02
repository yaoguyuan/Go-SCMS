package initializers

import (
	"log/slog"
	"os"
	"path/filepath"
)

var LOGGER *slog.Logger

func InitLogger() {
	// Open the log file or create it if it doesn't exist
	EnsureLogFileDefault()
	logFilePath := filepath.Join(LogFileDir, LogFileDefault)
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic("Failed to open log file: " + err.Error())
	}

	// Create a slog handler
	handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Create a logger with the handler
	LOGGER = slog.New(handler)

	// Set the logger as the default logger
	slog.SetDefault(LOGGER)
}
