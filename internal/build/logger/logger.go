// Package logger implements a [slog.Handler] that writes log records to an [io.Writer].
// The implementation is based on:
// https://github.com/golang/example/blob/master/slog-handler-guide/README.md
package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
)

type Flags uint8

const (
	NoColor Flags = 1 << iota
	ShowSource
)

func (fl Flags) Has(flag Flags) bool { return (fl & flag) != 0 }

const LevelSuccess slog.Level = 1

var levelStyles = map[slog.Level]string{
	slog.LevelDebug: ansi.BrightMagenta("Debug"),
	slog.LevelInfo:  ansi.BrightBlue("Info"),
	LevelSuccess:    ansi.BrightGreen("Success"),
	slog.LevelWarn:  ansi.BrightYellow("Warn"),
	slog.LevelError: ansi.BrightRed("Error"),
}

var levelStylesNoColor = map[slog.Level]string{
	slog.LevelDebug: "Debug",
	slog.LevelInfo:  "Info",
	LevelSuccess:    "Success",
	slog.LevelWarn:  "Warn",
	slog.LevelError: "Error",
}

type groupOrAttrs struct {
	group string
	attrs []slog.Attr
}

// LogHandler implements [slog.Handler].
//
//	[2025-12-29 15:12:16] (main.go:1:1) Info: Hello, world | file: 1, length: 2
type LogHandler struct {
	output io.Writer
	mu     *sync.Mutex
	groups []groupOrAttrs
	flags  Flags
	skip   int
}

func NewLogHandler(w io.Writer, flags Flags) *LogHandler {
	return &LogHandler{
		output: w,
		mu:     &sync.Mutex{},
		flags:  flags,
	}
}

// Close closes the h's [io.Writer], if needed.
func (h *LogHandler) Close() error {
	switch h.output {
	case nil, os.Stderr, os.Stdout:
	default:
		if w, ok := h.output.(io.Closer); ok {
			return w.Close()
		}
	}
	return nil
}

func (h *LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h != nil
}

func (h *LogHandler) SetSkip(skip int) {
	h.skip = skip
}

func (h *LogHandler) Handle(ctx context.Context, r slog.Record) error {
	// Color
	if h.flags.Has(NoColor) {
		oldDisableColor := ansi.DisableColor
		ansi.DisableColor = true
		defer func() { ansi.DisableColor = oldDisableColor }()
	}
	b := new(bytes.Buffer)
	// Time
	if !r.Time.IsZero() {
		b.WriteByte('[')
		if h.flags.Has(NoColor) {
			b.WriteString(r.Time.Format(time.DateTime))
		} else {
			b.WriteString(ansi.Magenta(r.Time.Format(time.DateOnly)))
			b.WriteByte(' ')
			b.WriteString(ansi.Cyan(r.Time.Format(time.TimeOnly)))
		}
		b.WriteString("] ")
	}
	// Level
	styles := levelStyles
	if h.flags.Has(NoColor) {
		styles = levelStylesNoColor
	}
	if levelStyle, ok := styles[r.Level]; ok {
		b.WriteString(levelStyle)
	} else {
		b.WriteString(r.Level.String())
	}
	b.WriteString(": ")
	// Message
	b.WriteString(r.Message)

	// Attributes and WithGroup/WithAttrs
	numAttrs := r.NumAttrs()
	if numAttrs > 0 || len(h.groups) > 0 {
		b.WriteString(" { ")
	}
	depth := h.writeState(b, numAttrs) // WithGroup and WithAttrs
	// Normal attributes
	if numAttrs > 0 {
		depth++
		var i int
		for a := range r.Attrs {
			h.writeAttr(b, a, i)
			i++
		}
	}
	// Add closing braces
	if depth > 0 {
		b.Grow(len(" }") * depth)
		for range depth {
			b.WriteString(" }")
		}
	}
	// Caller
	if r.PC != 0 && h.flags.Has(ShowSource) {
		if h.skip > 0 {
			var pcs [1]uintptr
			runtime.Callers(h.skip+1, pcs[:])
			r.PC = pcs[0]
			h.skip = 0
		}
		frames := runtime.CallersFrames([]uintptr{r.PC})
		fr, _ := frames.Next()
		fmt.Fprintf(b, ansi.Gray(" (%s:%d)"), filepath.Base(fr.File), fr.Line)
	}
	// Final newline
	b.WriteByte('\n')
	// Write
	if h.output == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := b.WriteTo(h.output)
	return err
}

var typeColors = map[slog.Kind]string{
	slog.KindString: ansi.CodeGreen,
	slog.KindInt64: ansi.CodeYellow,
	slog.KindUint64: ansi.CodeYellow,
	slog.KindBool: ansi.CodeYellow,
	slog.KindFloat64: ansi.CodeYellow,
	slog.KindTime: ansi.CodeCyan,
}

func (h *LogHandler) writeAttr(b *bytes.Buffer, a slog.Attr, i int) {
	if i > 0 {
		b.WriteString(", ")
	}
	a.Value = a.Value.Resolve()
	if a.Key != "" {
		b.WriteString(a.Key)
		b.WriteString(": ")
	}
	color, reset := ansi.Partial(typeColors[a.Value.Kind()]), ansi.Partial(ansi.CodeReset)
	if color != "" {
		b.WriteString(color)
	}
	switch v := a.Value; v.Kind() {
	case slog.KindGroup:
		b.WriteString("{ ")
		for i, a := range v.Group() {
			h.writeAttr(b, a, i)
		}
		b.WriteString(" }")
	case slog.KindString:
		q := strconv.AppendQuote(nil, v.String())
		b.Write(q)
	case slog.KindUint64:
		b.WriteString(strconv.FormatUint(v.Uint64(), 10))
	case slog.KindTime:
		b.WriteString(v.Time().Format(time.DateTime))
	default:
		b.WriteString(v.String())
	}
	b.WriteString(reset)
}

func (h *LogHandler) withGroupOrAttrs(ga groupOrAttrs) *LogHandler {
	new := *h
	new.groups = make([]groupOrAttrs, len(h.groups)+1)
	copy(new.groups, h.groups)
	new.groups[len(h.groups)] = ga
	return &new
}

func (h *LogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{name, nil})
}

func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

func (h *LogHandler) writeState(b *bytes.Buffer, numAttrs int) (depth int) {
	gs := h.groups
	if len(gs) == 0 {
		return
	}
	if numAttrs == 0 {
		for len(gs) > 0 && gs[len(gs)-1].group != "" {
			gs = gs[:len(gs)-1]
		}
	}
	for _, ga := range gs {
		if ga.group != "" {
			fmt.Fprintf(b, "%s: { ", ga.group)
			depth++
		} else {
			for i, a := range ga.attrs {
				h.writeAttr(b, a, i)
			}
			if numAttrs > 0 {
				b.WriteString(", ")
			}
		}
	}
	return
}
