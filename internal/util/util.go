package util

import (
	"io"
	"math/rand/v2"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
)

func FormatDuration(dur time.Duration) string {
	switch {
	case dur >= time.Hour:
		hours := float64(dur) / float64(time.Hour)
		return formatFloat(hours) + "hr"
	case dur >= time.Minute:
		minutes := float64(dur) / float64(time.Minute)
		return formatFloat(minutes) + "m"
	case dur >= time.Second:
		seconds := float64(dur) / float64(time.Second)
		return formatFloat(seconds) + "s"
	case dur >= time.Millisecond:
		ms := float64(dur) / float64(time.Millisecond)
		return formatFloat(ms) + "ms"
	case dur >= time.Microsecond:
		us := float64(dur) / float64(time.Microsecond)
		return formatFloat(us) + "µs"
	default:
		return strconv.FormatInt(int64(dur), 10) + "ns"
	}
}

func formatFloat(f float64) string {
	prec := 2
	if f >= 100 {
		prec--
	}
	s := strconv.FormatFloat(f, 'f', prec, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

type AllWriter interface {
	io.Writer
	io.StringWriter
	io.ByteWriter
}

func RelPath(basePath, targPath string) string {
	rel, err := filepath.Rel(basePath, targPath)
	if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return targPath
	}
	return rel
}

// RandomSlice returns a random element from a slice.
func RandomSlice[T any](s []T) T { return s[rand.IntN(len(s))] }

func KlarGradient(text string) string {
	// This is just for the VSCode color dialog
	rgba := func(r, g, b, _ uint8) [3]int { return [3]int{int(r), int(g), int(b)} }
	return ansi.Gradient(text, rgba(189, 247, 90, 1), rgba(91, 220, 230, 1))
}
