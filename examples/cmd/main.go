package main

import (
	"log/slog"
	"os"

	colorjson "github.com/hydronica/color-json"
)

func main() {
	// Create a new colorized JSON handler
	handler := colorjson.NewHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Set minimum level
	})
	// customize colors
	handler.Colors.Brace = colorjson.GrayColor
	// background red, white text
	handler.Colors.LevelError = colorjson.BgRedColor + colorjson.WhiteColor

	// Create a logger with the handler
	logger := slog.New(handler)

	// Set as the default logger
	slog.SetDefault(logger)

	// Example log messages
	slog.Info("Server started", "addr", ":8080")
	slog.Debug("Detailed debug message", "value", 123)
	slog.Debug(`Testing null & escaped quotes: "`, "value", nil)
	slog.Warn("Something might be wrong", "error", "connection timeout")
	slog.Error("Critical error occurred", "error", "file not found", "details", map[string]interface{}{
		"path":        "/var/log/app.log",
		"code":        404,
		"permissions": false,
	})
}
