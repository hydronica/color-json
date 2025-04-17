package colorjson

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type TerminalColor string

const (
	Reset        TerminalColor = "\033[0m"
	CyanColor    TerminalColor = "\033[36m"   // cyan
	GreenColor   TerminalColor = "\033[32m"   // green
	YellowColor  TerminalColor = "\033[33m"   // yellow
	MagentaColor TerminalColor = "\033[35m"   // magenta
	WhiteColor   TerminalColor = "\033[37;1m" // bright white
	BBlueColor   TerminalColor = "\033[34;1m" // bright blue
	BCyanColor   TerminalColor = "\033[36;1m" // bright cyan
	RedColor     TerminalColor = "\033[31m"   // red
	BlueColor    TerminalColor = "\033[34m"   // blue
	GrayColor    TerminalColor = "\033[90m"   // gray
	// Additional colors
	BoldColor      TerminalColor = "\033[1m"  // bold
	ItalicColor    TerminalColor = "\033[3m"  // italic
	UnderlineColor TerminalColor = "\033[4m"  // underline
	BlackColor     TerminalColor = "\033[30m" // black
	BgRedColor     TerminalColor = "\033[41m" // background red
	BgGreenColor   TerminalColor = "\033[42m" // background green
	BgYellowColor  TerminalColor = "\033[43m" // background yellow
	BgBlueColor    TerminalColor = "\033[44m" // background blue
	BgMagentaColor TerminalColor = "\033[45m" // background magenta
	BgCyanColor    TerminalColor = "\033[46m" // background cyan
	BgWhiteColor   TerminalColor = "\033[47m" // background white
	// 256-color mode
	OrangeColor TerminalColor = "\033[38;5;208m" // orange (256-color mode)
	PurpleColor TerminalColor = "\033[38;5;129m" // purple (256-color mode)
	PinkColor   TerminalColor = "\033[38;5;213m" // pink (256-color mode)
	TealColor   TerminalColor = "\033[38;5;23m"  // teal (256-color mode)
)

// Colors is a struct that contains the ANSI color codes for JSON syntax highlighting
type Colors struct {
	String     TerminalColor // string color
	Number     TerminalColor // number color
	Boolean    TerminalColor // boolean color
	Null       TerminalColor // null color
	Key        TerminalColor // key color
	Brace      TerminalColor // brace color
	LevelInfo  TerminalColor // level info color
	LevelDebug TerminalColor // level debug color
	LevelWarn  TerminalColor // level warn color
	LevelError TerminalColor // level error color
}

// ColorJSONHandler is a custom handler that produces colorized JSON output
type ColorJSONHandler struct {
	Colors      Colors // allows for customizing colors
	out         io.Writer
	opts        *slog.HandlerOptions
	baseHandler slog.Handler
}

// NewHandler creates a new handler for colorized JSON output
func NewHandler(w io.Writer, opts *slog.HandlerOptions) *ColorJSONHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	// Create a buffer to store JSON output temporarily
	buf := new(bytes.Buffer)

	// Create the base JSON handler that writes to our buffer
	baseHandler := slog.NewJSONHandler(buf, opts)

	return &ColorJSONHandler{
		out:         w,
		opts:        opts,
		baseHandler: baseHandler,
		// Default colors
		Colors: Colors{
			String:     GreenColor,
			Number:     YellowColor,
			Boolean:    MagentaColor,
			Null:       WhiteColor,
			Key:        CyanColor,
			Brace:      BBlueColor,
			LevelInfo:  BCyanColor,
			LevelDebug: BCyanColor,
			LevelWarn:  YellowColor,
			LevelError: RedColor,
		},
	}
}

// Enabled implements slog.Handler.
func (h *ColorJSONHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.baseHandler.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (h *ColorJSONHandler) Handle(ctx context.Context, r slog.Record) error {
	// Create a buffer to store the JSON output
	buf := new(bytes.Buffer)

	// Use the baseHandler to format as JSON, writing to our buffer
	tempHandler := slog.NewJSONHandler(buf, h.opts)
	if err := tempHandler.Handle(ctx, r); err != nil {
		return err
	}

	// Get the JSON string and colorize it
	jsonStr := buf.String()
	colorized := colorizeJSON(jsonStr, h.Colors)

	// Write the colorized JSON to the output
	_, err := fmt.Fprint(h.out, colorized)
	return err
}

// WithAttrs implements slog.Handler.
func (h *ColorJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ColorJSONHandler{
		out:         h.out,
		opts:        h.opts,
		baseHandler: h.baseHandler.WithAttrs(attrs),
	}
}

// WithGroup implements slog.Handler.
func (h *ColorJSONHandler) WithGroup(name string) slog.Handler {
	return &ColorJSONHandler{
		out:         h.out,
		opts:        h.opts,
		baseHandler: h.baseHandler.WithGroup(name),
	}
}

// colorizeJSON adds ANSI color codes to format a JSON string
func colorizeJSON(jsonStr string, colors Colors) string {
	type tokenType int
	const (
		tokenString tokenType = iota
		tokenNumber
		tokenBoolean
		tokenNull
		tokenBrace
		tokenKey
		tokenColon
		tokenComma
		tokenOther
		tokenLevel // New token type for log levels
	)

	var result strings.Builder
	var tokens []struct {
		content string
		typ     tokenType
	}

	// First pass: tokenize the JSON
	i := 0
	// Track whether we're about to see a level value
	possibleLevelKey := false

	for i < len(jsonStr) {
		c := jsonStr[i]
		switch c {
		case ' ', '\t', '\n', '\r':
			// Whitespace
			start := i
			for i < len(jsonStr) && (jsonStr[i] == ' ' || jsonStr[i] == '\t' || jsonStr[i] == '\n' || jsonStr[i] == '\r') {
				i++
			}
			tokens = append(tokens, struct {
				content string
				typ     tokenType
			}{content: jsonStr[start:i], typ: tokenOther})
		case '{', '}', '[', ']':
			// Braces/brackets
			tokens = append(tokens, struct {
				content string
				typ     tokenType
			}{content: string(c), typ: tokenBrace})
			i++
		case ':':
			// Colon
			tokens = append(tokens, struct {
				content string
				typ     tokenType
			}{content: ":", typ: tokenColon})
			i++
		case ',':
			// Comma
			tokens = append(tokens, struct {
				content string
				typ     tokenType
			}{content: ",", typ: tokenComma})
			i++
		case '"':
			// String or key
			start := i
			i++ // Skip opening quote
			strContent := ""
			for i < len(jsonStr) {
				if jsonStr[i] == '\\' && i+1 < len(jsonStr) {
					strContent += string(jsonStr[i]) + string(jsonStr[i+1])
					i += 2 // Skip escape sequence
					continue
				}
				if jsonStr[i] == '"' {
					strContent += string(jsonStr[i])
					i++ // Include closing quote
					break
				}
				strContent += string(jsonStr[i])
				i++
			}
			content := jsonStr[start:i]
			strValue := strings.Trim(strContent, "\"")

			// Look ahead to see if this is a key (followed by colon)
			isKey := false
			for j := i; j < len(jsonStr); j++ {
				if jsonStr[j] == ' ' || jsonStr[j] == '\t' || jsonStr[j] == '\n' || jsonStr[j] == '\r' {
					continue
				}
				if jsonStr[j] == ':' {
					isKey = true
				}
				break
			}

			if isKey {
				// Set flag if this is the level key
				if strValue == "level" {
					possibleLevelKey = true
				} else {
					possibleLevelKey = false
				}

				tokens = append(tokens, struct {
					content string
					typ     tokenType
				}{content: content, typ: tokenKey})
			} else if possibleLevelKey && isLogLevel(strValue) {
				// This is a log level value, mark it as such
				tokens = append(tokens, struct {
					content string
					typ     tokenType
				}{content: content, typ: tokenLevel})
				possibleLevelKey = false
			} else {
				tokens = append(tokens, struct {
					content string
					typ     tokenType
				}{content: content, typ: tokenString})
				possibleLevelKey = false
			}
		case 't':
			// true
			if i+3 < len(jsonStr) && jsonStr[i:i+4] == "true" {
				tokens = append(tokens, struct {
					content string
					typ     tokenType
				}{content: "true", typ: tokenBoolean})
				i += 4
			} else {
				tokens = append(tokens, struct {
					content string
					typ     tokenType
				}{content: string(c), typ: tokenOther})
				i++
			}
		case 'f':
			// false
			if i+4 < len(jsonStr) && jsonStr[i:i+5] == "false" {
				tokens = append(tokens, struct {
					content string
					typ     tokenType
				}{content: "false", typ: tokenBoolean})
				i += 5
			} else {
				tokens = append(tokens, struct {
					content string
					typ     tokenType
				}{content: string(c), typ: tokenOther})
				i++
			}
		case 'n':
			// null
			if i+3 < len(jsonStr) && jsonStr[i:i+4] == "null" {
				tokens = append(tokens, struct {
					content string
					typ     tokenType
				}{content: "null", typ: tokenNull})
				i += 4
			} else {
				tokens = append(tokens, struct {
					content string
					typ     tokenType
				}{content: string(c), typ: tokenOther})
				i++
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
			// Number
			start := i
			for i < len(jsonStr) && ((jsonStr[i] >= '0' && jsonStr[i] <= '9') ||
				jsonStr[i] == '.' || jsonStr[i] == 'e' || jsonStr[i] == 'E' ||
				jsonStr[i] == '+' || jsonStr[i] == '-') {
				i++
			}
			tokens = append(tokens, struct {
				content string
				typ     tokenType
			}{content: jsonStr[start:i], typ: tokenNumber})
		default:
			tokens = append(tokens, struct {
				content string
				typ     tokenType
			}{content: string(c), typ: tokenOther})
			i++
		}
	}

	// Second pass: colorize tokens
	for _, token := range tokens {
		switch token.typ {
		case tokenBrace:
			result.WriteString(string(colors.Brace) + token.content + string(Reset))
		case tokenKey:
			result.WriteString(string(colors.Key) + token.content + string(Reset))
		case tokenString:
			result.WriteString(string(colors.String) + token.content + string(Reset))
		case tokenNumber:
			result.WriteString(string(colors.Number) + token.content + string(Reset))
		case tokenBoolean:
			result.WriteString(string(colors.Boolean) + token.content + string(Reset))
		case tokenNull:
			result.WriteString(string(colors.Null) + token.content + string(Reset))
		case tokenLevel:
			// Apply the appropriate color based on the log level
			levelContent := strings.Trim(token.content, "\"")
			switch levelContent {
			case "INFO":
				result.WriteString(string(colors.LevelInfo) + token.content + string(Reset))
			case "DEBUG":
				result.WriteString(string(colors.LevelDebug) + token.content + string(Reset))
			case "WARN":
				result.WriteString(string(colors.LevelWarn) + token.content + string(Reset))
			case "ERROR":
				result.WriteString(string(colors.LevelError) + token.content + string(Reset))
			default:
				result.WriteString(token.content)
			}
		default:
			result.WriteString(token.content)
		}
	}

	return result.String()
}

// isLogLevel checks if a string is a valid log level
func isLogLevel(s string) bool {
	return s == "INFO" || s == "DEBUG" || s == "WARN" || s == "ERROR"
}
