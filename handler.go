package colorjson

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
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
	Default    TerminalColor // default color
	Message    TerminalColor
	DateTime   TerminalColor
	Key        TerminalColor // key color
	Persistent TerminalColor // color for persistent WithAttrs attributes
	LevelInfo  TerminalColor // level info color
	LevelDebug TerminalColor // level debug color
	LevelWarn  TerminalColor // level warn color
	LevelError TerminalColor // level error color
}

// ColorJSONHandler is a custom handler that produces colorized JSON output
type ColorJSONHandler struct {
	HandlerOptions

	mu                *sync.Mutex
	out               io.Writer
	attrs             []slog.Attr // persistent attributes from WithAttrs
	groups            []string    // group hierarchy from WithGroup
	preformattedAttrs string      // colored JSON fragment from WithAttrs
	nOpenGroups       int         // groups opened in preformattedAttrs
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

	// Colors defines preset color schemes. The zero value disables ANSI colors.
	Colors Colors
}

// NewHandler creates a new handler for colorized JSON output
func NewHandler(w io.Writer, opts *HandlerOptions) *ColorJSONHandler {
	if opts == nil {
		opts = &HandlerOptions{}
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = time.TimeOnly
	}

	h := &ColorJSONHandler{
		mu:             &sync.Mutex{},
		out:            w,
		HandlerOptions: *opts,
	}
	if !colorEnabled {
		h.Colors = Colors{}
	}
	return h
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
	colorized := h.coloredJSON(r)

	h.mu.Lock()
	_, err := fmt.Fprint(h.out, colorized)
	h.mu.Unlock()
	return err
}

// WithAttrs implements slog.Handler.
func (h *ColorJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	newFragment, groupsOpened := buildPreformatted(h, h.groups[h.nOpenGroups:], attrs)
	nOpenGroups := h.nOpenGroups
	if newFragment != "" && groupsOpened > 0 {
		nOpenGroups = h.nOpenGroups + groupsOpened
	}

	return &ColorJSONHandler{
		mu:                h.mu,
		out:               h.out,
		HandlerOptions:    h.HandlerOptions,
		attrs:             append(append([]slog.Attr(nil), h.attrs...), attrs...),
		groups:            append([]string(nil), h.groups...),
		preformattedAttrs: h.preformattedAttrs + newFragment,
		nOpenGroups:       nOpenGroups,
	}
}

// WithGroup implements slog.Handler.
func (h *ColorJSONHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &ColorJSONHandler{
		mu:                h.mu,
		out:               h.out,
		HandlerOptions:    h.HandlerOptions,
		attrs:             append([]slog.Attr(nil), h.attrs...),
		groups:            append(append([]string(nil), h.groups...), name),
		preformattedAttrs: h.preformattedAttrs,
		nOpenGroups:       h.nOpenGroups,
	}
}

func (h *ColorJSONHandler) coloredJSON(r slog.Record) string {
	buf := &strings.Builder{}
	buf.WriteByte('{')

	h.appendAttr(buf, nil, slog.TimeKey, slog.TimeValue(r.Time), h.Colors.Key, h.Colors.DateTime)
	h.writeLevel(buf, r.Level)
	h.appendAttr(buf, nil, slog.MessageKey, slog.StringValue(r.Message), h.Colors.Key, h.Colors.Message)

	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		h.writeSource(buf, f)
	}

	if h.preformattedAttrs != "" {
		buf.WriteString(h.preformattedAttrs)
	}

	nOpenGroups := h.nOpenGroups
	if r.NumAttrs() > 0 {
		for _, group := range h.groups[nOpenGroups:] {
			h.openGroup(buf, group)
		}
		pos := buf.Len()
		wrote := false
		r.Attrs(func(a slog.Attr) bool {
			if h.writeAttr(buf, h.groups, a, h.Colors.Default) {
				wrote = true
			}
			return true
		})
		if !wrote {
			content := buf.String()
			buf.Reset()
			buf.WriteString(content[:pos])
		} else {
			nOpenGroups = len(h.groups)
		}
	}

	for range h.groups[:nOpenGroups] {
		trimTrailingComma(buf)
		buf.WriteByte('}')
	}

	trimTrailingComma(buf)
	buf.WriteByte('}')
	buf.WriteByte('\n')
	return buf.String()
}

func (h *ColorJSONHandler) appendAttr(buf *strings.Builder, groups []string, key string, value slog.Value, keyColor, valueColor TerminalColor) {
	attr := slog.Attr{Key: key, Value: value}
	attr.Value = attr.Value.Resolve()
	if h.ReplaceAttr != nil && attr.Value.Kind() != slog.KindGroup {
		attr = h.ReplaceAttr(groups, attr)
		attr.Value = attr.Value.Resolve()
	}
	if attr.Equal(slog.Attr{}) {
		return
	}
	// Format built-in time after ReplaceAttr so callbacks see KindTime.
	if attr.Value.Kind() == slog.KindTime {
		attr.Value = slog.StringValue(attr.Value.Time().Format(h.TimeFormat))
	}
	attr = convertSpecial(attr)
	if attr.Equal(slog.Attr{}) {
		return
	}
	if attr.Value.Kind() == slog.KindGroup {
		h.writeAttr(buf, groups, attr, valueColor)
		return
	}
	h.cJSON(buf, attr.Key, attr.Value, keyColor, valueColor)
}

func (h *ColorJSONHandler) writeLevel(buf *strings.Builder, level slog.Level) {
	var valueColor TerminalColor
	switch level {
	case slog.LevelInfo:
		valueColor = h.Colors.LevelInfo
	case slog.LevelDebug:
		valueColor = h.Colors.LevelDebug
	case slog.LevelWarn:
		valueColor = h.Colors.LevelWarn
	case slog.LevelError:
		valueColor = h.Colors.LevelError
	default:
		valueColor = h.Colors.Default
	}
	h.appendAttr(buf, nil, slog.LevelKey, slog.AnyValue(level), h.Colors.Key, valueColor)
}

func (h *ColorJSONHandler) writeSource(buf *strings.Builder, f runtime.Frame) {
	switch h.Source {
	case SrcFull:
		h.appendAttr(buf, nil, slog.SourceKey, slog.AnyValue(&slog.Source{
			Function: f.Function,
			File:     f.File,
			Line:     f.Line,
		}), h.Colors.Key, h.Colors.Default)
	case SrcShortFile:
		h.appendAttr(buf, nil, "file", slog.StringValue(filepath.Base(f.File)+":"+strconv.Itoa(f.Line)), h.Colors.Key, h.Colors.Default)
	case SrcLongFile:
		h.appendAttr(buf, nil, "file", slog.StringValue(f.File+":"+strconv.Itoa(f.Line)), h.Colors.Key, h.Colors.Default)
	}
}

func buildPreformatted(h *ColorJSONHandler, groups []string, attrs []slog.Attr) (string, int) {
	if len(attrs) == 0 {
		return "", 0
	}

	buf := &strings.Builder{}
	for _, group := range groups {
		h.openGroup(buf, group)
	}

	wrote := false
	for _, attr := range attrs {
		if h.writeAttr(buf, h.groups[:h.nOpenGroups+len(groups)], attr, h.Colors.Persistent) {
			wrote = true
		}
	}
	if !wrote {
		return "", 0
	}

	return buf.String(), len(groups)
}

func (h *ColorJSONHandler) openGroup(buf *strings.Builder, name string) {
	paint(buf, h.Colors.Key, jsonString(name))
	buf.WriteString(`:{`)
}

func (h *ColorJSONHandler) writeAttr(buf *strings.Builder, groups []string, attr slog.Attr, valueColor TerminalColor) bool {
	attr.Value = attr.Value.Resolve()
	if attr.Value.Kind() == slog.KindGroup {
		members := attr.Value.Group()
		if attr.Key == "" {
			wrote := false
			for _, member := range members {
				if h.writeAttr(buf, groups, member, valueColor) {
					wrote = true
				}
			}
			return wrote
		}
		if len(members) == 0 {
			return false
		}
		pos := buf.Len()
		h.openGroup(buf, attr.Key)
		childGroups := append(groups, attr.Key)
		wrote := false
		for _, member := range members {
			if h.writeAttr(buf, childGroups, member, valueColor) {
				wrote = true
			}
		}
		if !wrote {
			content := buf.String()
			buf.Reset()
			buf.WriteString(content[:pos])
			return false
		}
		trimTrailingComma(buf)
		buf.WriteByte('}')
		buf.WriteByte(',')
		return true
	}

	if h.ReplaceAttr != nil {
		attr = h.ReplaceAttr(groups, attr)
		attr.Value = attr.Value.Resolve()
	}
	if attr.Equal(slog.Attr{}) {
		return false
	}
	attr = convertSpecial(attr)
	if attr.Equal(slog.Attr{}) {
		return false
	}
	if attr.Value.Kind() == slog.KindGroup {
		return h.writeAttr(buf, groups, attr, valueColor)
	}

	h.cJSON(buf, attr.Key, attr.Value, h.Colors.Key, valueColor)
	return true
}

// convertSpecial maps slog.Level and *slog.Source to their JSON output forms
// after ReplaceAttr, matching slog.JSONHandler.
func convertSpecial(attr slog.Attr) slog.Attr {
	if attr.Value.Kind() != slog.KindAny {
		return attr
	}
	switch v := attr.Value.Any().(type) {
	case slog.Level:
		attr.Value = slog.StringValue(v.String())
	case *slog.Source:
		if v == nil || *v == (slog.Source{}) {
			return slog.Attr{}
		}
		var as []slog.Attr
		if v.Function != "" {
			as = append(as, slog.String("function", v.Function))
		}
		if v.File != "" {
			as = append(as, slog.String("file", v.File))
		}
		if v.Line != 0 {
			as = append(as, slog.Int("line", v.Line))
		}
		attr.Value = slog.GroupValue(as...)
	}
	return attr
}

// cJSON writes a key/value pair using the handler's resolved color scheme.
func (h *ColorJSONHandler) cJSON(buf *strings.Builder, key string, value slog.Value, keyColor, valueColor TerminalColor) {
	paint(buf, keyColor, jsonString(key))
	buf.WriteByte(':')
	h.appendValue(buf, value, valueColor)
	buf.WriteByte(',')
}

func (h *ColorJSONHandler) appendValue(buf *strings.Builder, value slog.Value, color TerminalColor) {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		paint(buf, color, jsonString(value.String()))
	case slog.KindInt64:
		paint(buf, color, strconv.FormatInt(value.Int64(), 10))
	case slog.KindUint64:
		paint(buf, color, strconv.FormatUint(value.Uint64(), 10))
	case slog.KindFloat64:
		b, err := json.Marshal(value.Float64())
		if err != nil {
			paint(buf, color, fmt.Sprintf(`"!ERROR:%v"`, err))
		} else {
			paint(buf, color, string(b))
		}
	case slog.KindBool:
		paint(buf, color, strconv.FormatBool(value.Bool()))
	case slog.KindDuration:
		paint(buf, color, jsonString(value.Duration().String()))
	case slog.KindTime:
		paint(buf, color, jsonString(value.Time().Format(time.RFC3339Nano)))
	case slog.KindAny:
		appendJSONAny(buf, value.Any(), color)
	default:
		appendJSONAny(buf, value.Any(), color)
	}
}

func jsonString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

func appendJSONAny(buf *strings.Builder, v any, color TerminalColor) {
	if v == nil {
		paint(buf, color, "null")
		return
	}
	if err, ok := v.(error); ok {
		paint(buf, color, jsonString(err.Error()))
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		paint(buf, color, "null")
		return
	}
	paint(buf, color, string(b))
}

func trimTrailingComma(buf *strings.Builder) {
	if buf.Len() == 0 {
		return
	}
	s := buf.String()
	if s[len(s)-1] == ',' {
		buf.Reset()
		buf.WriteString(s[:len(s)-1])
	}
}

var (
	ColorStandard = Colors{
		Key:        grayColor,
		Default:    grayColor,
		Persistent: bWhiteColor,
		DateTime:   orangeColor,
		Message:    orangeColor,
		LevelInfo:  whiteColor,
		LevelDebug: bWhiteColor,
		LevelWarn:  bYellowColor,
		LevelError: bRedColor,
	}
	Colorful = Colors{
		Key:        tealColor,
		Persistent: bWhiteColor,
		DateTime:   purpleColor,
		Message:    redColor,
		LevelInfo:  bWhiteColor,
		LevelDebug: bCyanColor,
		LevelWarn:  bYellowColor,
		LevelError: bRedColor,
	}
)

type SrcFormat int

const (
	SrcFull      SrcFormat = 1 + iota // {"source":{"function":"repo/package.function","file":"a/c/d/file.go","line":26}}
	SrcShortFile                      // {"file":"file.go:26"} see log.LshortFile
	SrcLongFile                       // {"file":"a/c/d/file.go:26"} see log.LlongFile
)
