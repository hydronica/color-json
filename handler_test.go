package colorjson

import (
	"bytes"
	"log/slog"
	"os"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/hydronica/trial"
)

func TestOutput(t *testing.T) {
	levels := []struct {
		level    slog.Level
		levelStr string
	}{
		{slog.LevelInfo, "INFO"},
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelWarn, "WARN"},
		{slog.LevelError, "ERROR"},
	}

	for _, l := range levels {
		buf := new(bytes.Buffer)
		h := NewHandler(buf, &HandlerOptions{Source: SrcShortFile, ColorScheme: ColorDefault})
		pc, _, _, _ := runtime.Caller(0)
		rec := slog.NewRecord(time.Now(), l.level, "Test message for "+l.levelStr, pc)
		rec.AddAttrs(
			// Add a variety of types
			slog.Int("int", 42),
			slog.Float64("float", 3.14),
			slog.Bool("bool", true),
			slog.String("string", "hello world"),
		)
		if err := h.Handle(nil, rec); err != nil {
			t.Fatal(err)
		}

		// Print to stdout for visual inspection
		os.Stdout.Write(buf.Bytes())

	}
}

var regRmColors = regexp.MustCompile(`\033\[[0-9;]+m`)

func TestColoredJSON(t *testing.T) {
	testTime := time.Date(2024, 5, 28, 12, 34, 56, 0, time.UTC)
	pc, _, _, _ := runtime.Caller(0)
	h := HandlerOptions{ColorScheme: ColorDefault, TimeFormat: time.RFC3339}
	testFn := func(rec slog.Record) (string, error) {
		out := h.coloredJSON(rec, ColorDefault)
		// strip colors out for testing
		out = regRmColors.ReplaceAllString(out, "")
		return out, nil
	}
	cases := trial.Cases[slog.Record, string]{
		"basic info": {
			Input: func() slog.Record {
				rec := slog.NewRecord(testTime, slog.LevelInfo, "hello", pc)
				rec.AddAttrs(slog.String("foo", "bar"))
				return rec
			}(),
			Expected: func() string {
				// We ignore color codes for comparison
				return `{"time":"2024-05-28T12:34:56Z","level":"INFO","msg":"hello","foo":"bar"}` + "\n"
			}(),
		},
		"with numbers and bool": {
			Input: func() slog.Record {
				rec := slog.NewRecord(testTime, slog.LevelWarn, "warn msg", pc)
				rec.AddAttrs(
					slog.Int("int", 42),
					slog.Float64("float", 3.14),
					slog.Bool("bool", true),
				)
				return rec
			}(),
			Expected: func() string {
				return `{"time":"2024-05-28T12:34:56Z","level":"WARN","msg":"warn msg","int":42,"float":3.14,"bool":true}` + "\n"
			}(),
		},
	}
	trial.New(testFn, cases).Test(t)
}
