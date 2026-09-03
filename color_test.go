package colorjson

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

func TestUseColor(t *testing.T) {
	cases := map[string]struct {
		env  map[string]string
		want bool
	}{
		"no color set":         {env: map[string]string{"NO_COLOR": "1", "TERM": "xterm-256color"}, want: false},
		"force color set":      {env: map[string]string{"FORCE_COLOR": "1", "TERM": "dumb"}, want: true},
		"no color beats force": {env: map[string]string{"NO_COLOR": "1", "FORCE_COLOR": "1", "TERM": "xterm-256color"}, want: false},
		"empty term":           {env: map[string]string{"TERM": ""}, want: false},
		"dumb term":            {env: map[string]string{"TERM": "dumb"}, want: false},
		"color terminal":       {env: map[string]string{"TERM": "xterm-256color"}, want: true},
		"default when unset":   {env: map[string]string{"TERM": "screen"}, want: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for _, key := range []string{"NO_COLOR", "FORCE_COLOR", "TERM"} {
				if err := os.Unsetenv(key); err != nil {
					t.Fatal(err)
				}
			}
			for key, val := range tc.env {
				t.Setenv(key, val)
			}
			if got := useColor(); got != tc.want {
				t.Fatalf("useColor() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestApplyColorDisabled(t *testing.T) {
	old := colorEnabled
	colorEnabled = false
	t.Cleanup(func() { colorEnabled = old })

	buf := new(bytes.Buffer)
	h := NewHandler(buf, &HandlerOptions{
		TimeFormat: time.RFC3339,
		Colors:     ColorDefault,
	})
	rec := slog.NewRecord(time.Date(2024, 5, 28, 12, 34, 56, 0, time.UTC), slog.LevelInfo, "hello", 0)
	rec.AddAttrs(slog.String("foo", "bar"))

	if err := h.Handle(nil, rec); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if strings.Contains(out, "\033") {
		t.Fatalf("expected no ANSI codes when color is disabled, got %q", out)
	}
	expected := `{"time":"2024-05-28T12:34:56Z","level":"INFO","msg":"hello","foo":"bar"}` + "\n"
	if out != expected {
		t.Fatalf("got %q, want %q", out, expected)
	}
}

func TestNewHandlerResolvesColors(t *testing.T) {
	old := colorEnabled
	t.Cleanup(func() { colorEnabled = old })

	colorEnabled = false
	h := NewHandler(nil, &HandlerOptions{Colors: ColorDefault})
	if h.Colors != NoColor {
		t.Fatalf("Colors = %#v, want NoColor", h.Colors)
	}

	colorEnabled = true
	h = NewHandler(nil, &HandlerOptions{Colors: ColorDefault})
	if h.Colors != ColorDefault {
		t.Fatalf("Colors = %#v, want ColorDefault", h.Colors)
	}

	h = NewHandler(nil, &HandlerOptions{Colors: NoColor})
	if h.Colors != NoColor {
		t.Fatalf("explicit NoColor should be preserved")
	}
}

func TestPaint(t *testing.T) {
	buf := new(strings.Builder)

	paint(buf, noColor, "plain")
	if got := buf.String(); got != "plain" {
		t.Fatalf("paint with no color = %q, want plain text", got)
	}

	buf.Reset()
	paint(buf, redColor, "err")
	if got := buf.String(); got != string(redColor)+"err"+string(reset) {
		t.Fatalf("paint with color = %q", got)
	}
}
