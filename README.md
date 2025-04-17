# colorjson

A Go package that provides a colorized JSON handler for the Go standard library's `slog` package.

## Features

- Pretty-prints JSON logs with syntax highlighting
- Color-coded log levels (INFO=green, DEBUG=cyan, WARN=yellow, ERROR=red)
- Properly formats and colorizes strings, numbers, booleans, and null values
- Implements the `slog.Handler` interface for seamless integration

## Installation

```bash
go get github.com/zjeremiah/colorjson
```

## Usage

```go
package main

import (
	"log/slog"
	"os"

	"github.com/zjeremiah/colorjson"
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
```

## Configuration

The `NewHandler` function accepts the same parameters as the standard `slog.NewJSONHandler`:

- `w io.Writer` - The output destination (typically `os.Stderr`)
- `opts *slog.HandlerOptions` - Handler options including:
  - `Level` - The minimum log level to output
  - `AddSource` - Whether to add source code information
  - `ReplaceAttr` - A function to customize log attribute handling

## Output

The output will be colorized JSON with:

- JSON keys in cyan
- Strings in green
- Numbers in yellow
- Booleans in magenta
- Null values in bright white
- Braces/brackets in bright blue
- Log levels colored according to severity:
  - INFO: green
  - DEBUG: bright cyan
  - WARN: yellow
  - ERROR: red

## Terminal Support

The colorization uses ANSI escape codes, which are supported by most modern terminals. If you're redirecting output to a file or using a terminal that doesn't support colors, you might see the raw ANSI codes.

## License

GNU General Public License v3.0