package main

import (
	"log/slog"
	"os"

	colorjson "github.com/hydronica/color-json"
)

func main() {
	colors := colorjson.ColorStandard
	colors.LevelError = colorjson.TerminalColor("\033[41m\033[37m") // white on red background

	handler := colorjson.NewHandler(os.Stderr, &colorjson.HandlerOptions{
		Level:  slog.LevelDebug,
		Colors: colors,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

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
