package build

import (
	"io"
	"log/slog"
	"os"
)

// https://github.com/golang/example/blob/master/slog-handler-guide/README.md
// TODO: this whole file

// LogHandler implements [slog.Handler].
type LogHandler struct {
	*slog.TextHandler
	output io.Writer
}

func NewLogHandler(w io.Writer) *LogHandler {
	if w == nil {
		w = io.Discard
	}
	return &LogHandler{
		slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}),
		w,
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
