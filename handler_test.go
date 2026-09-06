package colorjson

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/hydronica/trial"
)

var (
	pc    uintptr
	lFile string
	sFile string
	line  string
)

func TestMain(t *testing.M) {
	ptr, file, lineNum, _ := runtime.Caller(0)
	pc = ptr
	lFile = file
	_, sFile = filepath.Split(file)
	line = strconv.Itoa(lineNum)

	t.Run()
}

func TestOutput(t *testing.T) {
	// Visual sample: iterate presets, one short line per level, attrs split by level.
	testTime := time.Date(2024, 5, 28, 12, 34, 56, 0, time.UTC)
	base := HandlerOptions{TimeFormat: time.TimeOnly}
	names := []string{"Default", "Colorful", "No Color"}
	profiles := []Colors{ColorDefault, Colorful, NoColor}

	write := func(colors Colors, opts HandlerOptions, persistent []slog.Attr, rec slog.Record) {
		buf := new(bytes.Buffer)
		opts.Colors = colors
		var h slog.Handler = NewHandler(buf, &opts)
		if len(persistent) > 0 {
			h = h.WithAttrs(persistent)
		}
		if err := h.Handle(nil, rec); err != nil {
			t.Fatal(err)
		}
		os.Stdout.Write(buf.Bytes())
	}

	for i, colors := range profiles {
		fmt.Println(names[i])

		rec := slog.NewRecord(testTime, slog.LevelDebug, "debug", 0)
		rec.AddAttrs(slog.Bool("ok", true))
		write(colors, base, nil, rec)

		rec = slog.NewRecord(testTime, slog.LevelInfo, "info", pc)
		rec.AddAttrs(slog.String("s", "hi"))
		write(colors, HandlerOptions{TimeFormat: time.TimeOnly, Source: SrcShortFile}, nil, rec)

		rec = slog.NewRecord(testTime, slog.LevelWarn, "warn", 0)
		rec.AddAttrs(slog.Int("n", 42), slog.Float64("f", 3.14))
		write(colors, base, nil, rec)

		rec = slog.NewRecord(testTime, slog.LevelError, "error", 0)
		rec.AddAttrs(
			slog.Any("null", nil),
			slog.Group("http", slog.String("method", "GET")),
		)
		write(colors, base, []slog.Attr{slog.String("trace_id", "abc")}, rec)
	}
}

var regRmColors = regexp.MustCompile(`\033\[[0-9;]+m`)

func TestColoredJSON(t *testing.T) {
	type input struct {
		Opts HandlerOptions
		Rec  slog.Record
	}

	testTime := time.Date(2024, 5, 28, 12, 34, 56, 0, time.UTC)
	//pc, _, _, _ := runtime.Caller(0)
	regRmColors := regexp.MustCompile(`\033\[[0-9;]+m`)

	testFn := func(in input) (string, error) {
		h := NewHandler(io.Discard, &in.Opts)
		out := h.coloredJSON(in.Rec)
		return regRmColors.ReplaceAllString(out, ""), nil
	}

	cases := trial.Cases[input, string]{
		"basic": {
			Input: input{
				Opts: HandlerOptions{TimeFormat: time.RFC3339},
				Rec: func() slog.Record {
					rec := slog.NewRecord(testTime, slog.LevelInfo, "hello", pc)
					rec.AddAttrs(slog.String("foo", "bar"))
					return rec
				}(),
			},
			Expected: `{"time":"2024-05-28T12:34:56Z","level":"INFO","msg":"hello","foo":"bar"}` + "\n",
		},
		"with numbers and bool": {
			Input: input{
				Opts: HandlerOptions{TimeFormat: time.RFC3339},
				Rec: func() slog.Record {
					rec := slog.NewRecord(testTime, slog.LevelWarn, "warn msg", pc)
					rec.AddAttrs(
						slog.Int("int", 42),
						slog.Float64("float", 3.14),
						slog.Bool("bool", true),
					)
					return rec
				}(),
			},
			Expected: `{"time":"2024-05-28T12:34:56Z","level":"WARN","msg":"warn msg","int":42,"float":3.14,"bool":true}` + "\n",
		},
		"date only": {
			Input: input{
				Opts: HandlerOptions{TimeFormat: time.DateOnly},
				Rec: func() slog.Record {
					rec := slog.NewRecord(testTime, slog.LevelInfo, "date only", pc)
					rec.AddAttrs(slog.String("foo", "bar"))
					return rec
				}(),
			},
			Expected: `{"time":"2024-05-28","level":"INFO","msg":"date only","foo":"bar"}` + "\n",
		},
		"time only": {
			Input: input{
				Opts: HandlerOptions{TimeFormat: time.TimeOnly},
				Rec: func() slog.Record {
					rec := slog.NewRecord(testTime, slog.LevelInfo, "time only", pc)
					rec.AddAttrs(slog.String("foo", "bar"))
					return rec
				}(),
			},
			Expected: `{"time":"12:34:56","level":"INFO","msg":"time only","foo":"bar"}` + "\n",
		},
		"source full": {
			Input: input{
				Opts: HandlerOptions{TimeFormat: time.DateOnly, Source: SrcFull},
				Rec:  slog.NewRecord(testTime, slog.LevelInfo, "src full", pc),
			},
			Expected: `{"time":"2024-05-28","level":"INFO","msg":"src full","source":{"function":"github.com/hydronica/color-json.TestMain","file":"` + lFile + `","line":` + line + `}}` + "\n",
		},
		"source short": {
			Input: input{
				Opts: HandlerOptions{TimeFormat: time.DateOnly, Source: SrcShortFile},
				Rec:  slog.NewRecord(testTime, slog.LevelInfo, "src short file", pc),
			},
			Expected: `{"time":"2024-05-28","level":"INFO","msg":"src short file","file":"` + sFile + ":" + line + `"}` + "\n",
		},
		"source long ": {
			Input: input{
				Opts: HandlerOptions{TimeFormat: time.DateOnly, Source: SrcLongFile},
				Rec:  slog.NewRecord(testTime, slog.LevelInfo, "src long file", pc),
			},
			Expected: `{"time":"2024-05-28","level":"INFO","msg":"src long file","file":"` + lFile + ":" + line + `"}` + "\n",
		},
	}

	trial.New(testFn, cases).Test(t)
}

func TestEnabled(t *testing.T) {
	type input struct {
		handlerLevel slog.Leveler
		logLevel     slog.Level
	}
	testFn := func(in input) (bool, error) {
		h := NewHandler(nil, &HandlerOptions{Level: in.handlerLevel})
		return h.Enabled(nil, in.logLevel), nil
	}
	cases := trial.Cases[input, bool]{
		"nil handler level, info": {
			Input:    input{handlerLevel: nil, logLevel: slog.LevelInfo},
			Expected: true,
		},
		"nil handler level, debug": {
			Input:    input{handlerLevel: nil, logLevel: slog.LevelDebug},
			Expected: false,
		},
		"handler info, info": {
			Input:    input{handlerLevel: slog.LevelInfo, logLevel: slog.LevelInfo},
			Expected: true,
		},
		"handler info, debug": {
			Input:    input{handlerLevel: slog.LevelInfo, logLevel: slog.LevelDebug},
			Expected: false,
		},
		"handler debug, debug": {
			Input:    input{handlerLevel: slog.LevelDebug, logLevel: slog.LevelDebug},
			Expected: true,
		},
		"handler warn, info": {
			Input:    input{handlerLevel: slog.LevelWarn, logLevel: slog.LevelInfo},
			Expected: false,
		},
		"handler warn, error": {
			Input:    input{handlerLevel: slog.LevelWarn, logLevel: slog.LevelError},
			Expected: true,
		},
	}
	trial.New(testFn, cases).Test(t)
}

func TestWithAttrsAndWithGroup(t *testing.T) {
	baseHandler := NewHandler(nil, &HandlerOptions{TimeFormat: time.DateOnly})

	testFn := func(in slog.Handler) (string, error) {
		buf := new(bytes.Buffer)
		handler := in.(*ColorJSONHandler)
		handler.out = buf

		// Create a record with attributes that should go into groups
		testTime := time.Date(2024, 5, 28, 12, 34, 56, 0, time.UTC)
		pc, _, _, _ := runtime.Caller(0)
		rec := slog.NewRecord(testTime, slog.LevelInfo, "hello world", pc)

		// Add attributes that will be placed in the current group context
		rec.AddAttrs(
			slog.String("method", "POST"),
			slog.Int("status", 200),
		)

		// Handle the record
		if err := handler.Handle(nil, rec); err != nil {
			return "", err
		}

		// Remove colors for easier testing
		return regRmColors.ReplaceAllString(buf.String(), ""), nil
	}

	cases := trial.Cases[slog.Handler, string]{
		"with attrs": {
			Input: baseHandler.WithAttrs([]slog.Attr{
				slog.String("user_id", "123"),
				slog.String("session", "abc-def"),
			}),
			Expected: `{"time":"2024-05-28","level":"INFO","msg":"hello world","user_id":"123","session":"abc-def","method":"POST","status":200}` + "\n",
		},
		"with group": {
			Input:    baseHandler.WithGroup("http"),
			Expected: `{"time":"2024-05-28","level":"INFO","msg":"hello world","http":{"method":"POST","status":200}}` + "\n",
		},
		"with attrs and group": {
			Input: baseHandler.WithAttrs([]slog.Attr{
				slog.String("trace_id", "xyz789"),
			}).WithGroup("request"),
			Expected: `{"time":"2024-05-28","level":"INFO","msg":"hello world","trace_id":"xyz789","request":{"method":"POST","status":200}}` + "\n",
		},
		"nested groups": {
			Input:    baseHandler.WithGroup("service").WithGroup("database").(*ColorJSONHandler),
			Expected: `{"time":"2024-05-28","level":"INFO","msg":"hello world","service":{"database":{"method":"POST","status":200}}}` + "\n",
		},
	}

	trial.New(testFn, cases).Test(t)
}
