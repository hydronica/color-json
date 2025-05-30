package colorjson

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type TerminalColor string

const (
	reset        TerminalColor = "\033[0m"
	cyanColor    TerminalColor = "\033[36m"   // cyan
	greenColor   TerminalColor = "\033[32m"   // green
	yellowColor  TerminalColor = "\033[33m"   // yellow
	magentaColor TerminalColor = "\033[35m"   // magenta
	whiteColor   TerminalColor = "\033[37m"   // white
	bWhiteColor  TerminalColor = "\033[37;1m" // bright white
	bBlueColor   TerminalColor = "\033[34;1m" // bright blue
	bCyanColor   TerminalColor = "\033[36;1m" // bright cyan
	bYellowColor TerminalColor = "\033[33;1m" // bright yellow
	bRedColor    TerminalColor = "\033[31;1m" // bright red
	redColor     TerminalColor = "\033[31m"   // red
	blueColor    TerminalColor = "\033[34m"   // blue
	grayColor    TerminalColor = "\033[90m"   // gray
	// Additional colors
	boldColor      TerminalColor = "\033[1m"  // bold
	italicColor    TerminalColor = "\033[3m"  // italic
	underlineColor TerminalColor = "\033[4m"  // underline
	blackColor     TerminalColor = "\033[30m" // black
	bgRedColor     TerminalColor = "\033[41m" // background red
	bgGreenColor   TerminalColor = "\033[42m" // background green
	bgYellowColor  TerminalColor = "\033[43m" // background yellow
	bgBlueColor    TerminalColor = "\033[44m" // background blue
	bgMagentaColor TerminalColor = "\033[45m" // background magenta
	bgCyanColor    TerminalColor = "\033[46m" // background cyan
	bgWhiteColor   TerminalColor = "\033[47m" // background white
	// 256-color mode
	orangeColor TerminalColor = "\033[38;5;208m" // orange (256-color mode)
	purpleColor TerminalColor = "\033[38;5;129m" // purple (256-color mode)
	pinkColor   TerminalColor = "\033[38;5;213m" // pink (256-color mode)
	tealColor   TerminalColor = "\033[38;5;23m"  // teal (256-color mode)
	noColor     TerminalColor = ""               // don't modify
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
	HandlerOptions

	out    io.Writer
	attrs  []slog.Attr // persistent attributes from WithAttrs
	groups []string    // group hierarchy from WithGroup
}

// HandlerOptions is a custom options struct that extends slog.HandlerOptions
type HandlerOptions struct {
	// AddSource causes the handler to compute the source code position
	// of the log statement and add a SourceKey attribute to the output.
	Source SrcFormat

	// Minimum level to log (Default: slog.LevelInfo)
	Level slog.Leveler

	// ReplaceAttr is called to rewrite each non-group attribute before it is logged.
	// See https://pkg.go.dev/log/slog#HandlerOptions for details.
	ReplaceAttr func(groups []string, attr slog.Attr) slog.Attr

	// TimeFormat allows customizing how time is formatted
	// If empty, time.TimeOnly will be used
	TimeFormat string

	// ColorScheme defines preset color schemes
	// Valid values are: "default", "tint", "monochrome"
	ColorScheme Colors
}

// NewHandler creates a new handler for colorized JSON output
func NewHandler(w io.Writer, opts *HandlerOptions) *ColorJSONHandler {
	if opts == nil {
		opts = &HandlerOptions{
			ColorScheme: ColorDefault,
		}
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = time.TimeOnly
	}

	return &ColorJSONHandler{
		out:            w,
		HandlerOptions: *opts,
	}
}

// Enabled implements slog.Handler.
func (h *ColorJSONHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.Level == nil {
		return level >= slog.LevelInfo
	}
	return level >= h.Level.Level()
}

// Handle implements slog.Handler.
func (h *ColorJSONHandler) Handle(ctx context.Context, r slog.Record) error {
	colorized := h.coloredJSON(r, h.ColorScheme)

	// Write the colorized JSON to the output
	_, err := fmt.Fprint(h.out, colorized)
	return err
}

// WithAttrs implements slog.Handler.
func (h *ColorJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Create a copy of existing attributes and append new ones
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)

	return &ColorJSONHandler{
		out:            h.out,
		HandlerOptions: h.HandlerOptions,
		attrs:          newAttrs,
		groups:         append([]string(nil), h.groups...), // copy group hierarchy
	}
}

// WithGroup implements slog.Handler.
func (h *ColorJSONHandler) WithGroup(name string) slog.Handler {
	// Create a copy of existing groups and append new group
	newGroups := make([]string, len(h.groups), len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups = append(newGroups, name)

	return &ColorJSONHandler{
		out:            h.out,
		HandlerOptions: h.HandlerOptions,
		attrs:          append([]slog.Attr(nil), h.attrs...), // copy persistent attributes
		groups:         newGroups,
	}
}

func (h *ColorJSONHandler) coloredJSON(r slog.Record, colors Colors) string {
	buf := &strings.Builder{}
	buf.WriteString("{")

	// Write time
	cJSON(buf, "time", r.Time.Format(h.TimeFormat), h.ColorScheme.Key, bWhiteColor)

	// Write level
	switch r.Level {
	case slog.LevelInfo:
		cJSON(buf, "level", r.Level.String(), h.ColorScheme.Key, h.ColorScheme.LevelInfo)
	case slog.LevelDebug:
		cJSON(buf, "level", r.Level.String(), h.ColorScheme.Key, h.ColorScheme.LevelDebug)
	case slog.LevelWarn:
		cJSON(buf, "level", r.Level.String(), h.ColorScheme.Key, h.ColorScheme.LevelWarn)
	case slog.LevelError:
		cJSON(buf, "level", r.Level.String(), h.ColorScheme.Key, h.ColorScheme.LevelError)
	}

	// Write message
	cJSON(buf, "msg", r.Message, h.ColorScheme.Key, bWhiteColor)

	// Write source if available
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		switch h.Source {
		case SrcFull:
			buf.WriteString(`"source":{"function":"` + f.Function + `","file":"` + f.File + `","line":` + strconv.Itoa(f.Line) + `}`)
		case SrcShortFile:
			cJSON(buf, "file", filepath.Base(f.File)+":"+strconv.Itoa(f.Line), h.ColorScheme.Key, bWhiteColor)
		case SrcLongFile:
			cJSON(buf, "file", f.File+":"+strconv.Itoa(f.Line), h.ColorScheme.Key, bWhiteColor)
		}
	}

	// Helper function to write attributes, handling grouping
	writeAttrs := func(attrs []slog.Attr, groups []string) {
		if len(groups) == 0 {
			// No groups - write attributes directly
			for _, attr := range attrs {
				if h.ReplaceAttr != nil {
					attr = h.ReplaceAttr(nil, attr)
				}
				if attr.Value.Kind() == slog.KindGroup {
					// Handle group attribute
					buf.WriteString(string(h.ColorScheme.Key) + `"` + attr.Key + `"` + string(reset) + `:{`)
					for _, groupAttr := range attr.Value.Group() {
						if h.ReplaceAttr != nil {
							groupAttr = h.ReplaceAttr(nil, groupAttr)
						}
						cJSON(buf, groupAttr.Key, groupAttr.Value.Any(), h.ColorScheme.Key, bWhiteColor)
					}
					// Remove the last character (trailing comma)
					content := buf.String()
					buf.Reset()
					buf.WriteString(content[:len(content)-1])
					buf.WriteString("},")
					continue
				}
				cJSON(buf, attr.Key, attr.Value.Any(), h.ColorScheme.Key, bWhiteColor)
			}
		} else {
			// Build nested group structure
			h.writeGroupedAttrs(buf, attrs, groups, 0)
		}
	}

	// Write persistent attributes (from WithAttrs) - always at top level
	if len(h.attrs) > 0 {
		for _, attr := range h.attrs {
			if h.ReplaceAttr != nil {
				attr = h.ReplaceAttr(nil, attr)
			}
			cJSON(buf, attr.Key, attr.Value.Any(), h.ColorScheme.Key, bWhiteColor)
		}
	}

	// Write record attributes - grouped if there are groups
	if r.NumAttrs() > 0 {
		var recordAttrs []slog.Attr
		r.Attrs(func(a slog.Attr) bool {
			recordAttrs = append(recordAttrs, a)
			return true
		})
		writeAttrs(recordAttrs, h.groups)
	}

	return strings.TrimRight(buf.String(), ",") + "}\n"
}

// writeGroupedAttrs writes attributes with proper group nesting
func (h *ColorJSONHandler) writeGroupedAttrs(buf *strings.Builder, attrs []slog.Attr, groups []string, depth int) {
	if depth >= len(groups) {
		// No more groups - write attributes directly
		for _, attr := range attrs {
			if h.ReplaceAttr != nil {
				attr = h.ReplaceAttr(groups, attr)
			}
			cJSON(buf, attr.Key, attr.Value.Any(), h.ColorScheme.Key, bWhiteColor)
		}
		return
	}

	// Create nested group object
	groupName := groups[depth]
	buf.WriteString(string(h.ColorScheme.Key) + `"` + groupName + `"` + string(reset) + `:{`)

	// Recursively write the rest
	h.writeGroupedAttrs(buf, attrs, groups, depth+1)

	// Close the group, removing trailing comma first
	content := buf.String()
	if strings.HasSuffix(content, ",") {
		buf.Reset()
		buf.WriteString(content[:len(content)-1])
	}
	buf.WriteString("},")
}

// cJSON will write the key/value to the buffer based on the defined Color pattern
func cJSON(buf *strings.Builder, key string, value any, keyColor, valueColor TerminalColor) {
	buf.WriteString(string(keyColor) + `"` + key + `"` + string(reset) + `:`) // key

	switch v := value.(type) {
	case string:
		buf.WriteString(string(valueColor) + `"` + v + `"` + string(reset) + `,`)
	case int64, int32, int16, int8, int,
		uint64, uint32, uint16, uint8, uint,
		float64, float32, bool:
		buf.WriteString(string(valueColor) + fmt.Sprintf("%v", v) + string(reset) + `,`)
	case nil:
		buf.WriteString(string(valueColor) + "null" + string(reset) + `,`)
	default:
		// Convert anything else to string with quotes
		buf.WriteString(string(valueColor) + `"` + fmt.Sprint(v) + `"` + string(reset) + `,`)
	}
}

var (
	ColorDefault = Colors{
		String:     whiteColor,  // All values white
		Number:     whiteColor,  // All values white
		Boolean:    whiteColor,  // All values white
		Null:       whiteColor,  // All values white
		Key:        grayColor,   // All keys gray
		Brace:      bBlueColor,  // Keep braces blue for readability
		LevelInfo:  greenColor,  // info - green
		LevelDebug: whiteColor,  // debug - white
		LevelWarn:  yellowColor, // warn - yellow
		LevelError: redColor,    // error - red
	}
	Colorful = Colors{
		String:     greenColor,
		Number:     yellowColor,
		Boolean:    magentaColor,
		Null:       whiteColor,
		Key:        cyanColor,
		Brace:      bBlueColor,
		LevelInfo:  bWhiteColor,
		LevelDebug: bCyanColor,
		LevelWarn:  bYellowColor,
		LevelError: bRedColor,
	}
	NoColor = Colors{}
)

type SrcFormat int

const (
	SrcFull      SrcFormat = 1 + iota // {"source":{"function":"repo/package.function","file":"a/c/d/file.go","line":26`}
	SrcShortFile                      // {"file":"file.go:26"} see log.LshortFile
	SrcLongFile                       // {"file":"a/c/d/file.go:26"} see log.LlongFile
)
