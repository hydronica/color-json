package colorjson

import (
	"os"
	"strings"
)

// colorEnabled is set once at init from the terminal environment.
var colorEnabled = useColor()

// useColor reports whether ANSI colors should be applied to terminal output.
// It respects the NO_COLOR and FORCE_COLOR conventions and checks TERM.
func useColor() bool {
	if _, noColor := os.LookupEnv("NO_COLOR"); noColor {
		return false
	}
	if _, forceColor := os.LookupEnv("FORCE_COLOR"); forceColor {
		return true
	}
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}

// paint writes text to buf, wrapping it in ANSI color codes when color is set.
func paint(buf *strings.Builder, color TerminalColor, text string) {
	if color == "" {
		buf.WriteString(text)
		return
	}
	buf.WriteString(string(color) + text + string(reset))
}
