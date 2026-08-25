package util

import (
	"cmp"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/build/logger"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"golang.org/x/term"
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

func FormatNumber(n int) string {
	orig := strconv.Itoa(n)
	// Numbers under 10,000 shouldn't have separators
	if len(orig) <= 4 || (len(orig) == 5 && orig[0] == '-') {
		return orig
	}
	isNegative := orig[0] == '-'
	if isNegative {
		orig = orig[1:]
	}
	var (
		numSeps = (len(orig) - 1) / 3
		buf     = make([]byte, len(orig)+numSeps)
		bufI    = len(buf)
	)
	for end := len(orig); end > 0; end -= 3 {
		start := max(end-3, 0)
		bufI -= end - start
		copy(buf[bufI:], orig[start:end])
		if start > 0 {
			bufI--
			buf[bufI] = ','
		}
	}
	if isNegative {
		return "-" + string(buf)
	}
	return string(buf)
}

type AllWriter interface {
	io.Writer
	io.StringWriter
	io.ByteWriter
}

type allWriter struct {
	w io.Writer
}

func WrapAllWriter(w io.Writer) AllWriter {
	return &allWriter{w}
}

func (aw *allWriter) Write(p []byte) (int, error) {
	return aw.w.Write(p)
}

func (aw *allWriter) WriteString(s string) (int, error) {
	return aw.w.Write([]byte(s))
}

func (aw *allWriter) WriteByte(c byte) error {
	_, err := aw.w.Write([]byte{c})
	return err
}

func RelPath(basePath, targPath string) string {
	rel, err := filepath.Rel(basePath, targPath)
	if err != nil || !filepath.IsLocal(rel) {
		return targPath
	}
	return rel
}

func ShortenHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if rel, err := filepath.Rel(home, path); err == nil && filepath.IsLocal(rel) {
		return filepath.Join("~", rel)
	}
	return path
}

// RandomSlice returns a random element from a slice.
func RandomSlice[T any](s []T) T { return s[rand.IntN(len(s))] }

func KlarGradient(text string) string {
	// This is just for the VSCode color dialog
	rgba := func(r, g, b, _ uint8) [3]int { return [3]int{int(r), int(g), int(b)} }
	return ansi.Gradient(text, rgba(189, 247, 90, 1), rgba(91, 220, 230, 1))
}

func FormatSize(size int64) string {
	const (
		// Go doesn't let you use [math.Pow] as a constant
		kb = 1000
		mb = 1000 * 1000
		gb = 1000 * 1000 * 1000
		tb = 1000 * 1000 * 1000 * 1000
	)
	f := float64(size)
	switch {
	case size == 1:
		return strconv.FormatInt(size, 10) + " byte"
	case size < kb:
		return strconv.FormatInt(size, 10) + " bytes"
	case size < mb:
		return fmt.Sprintf("%.2f KB", f/kb)
	case size < gb:
		return fmt.Sprintf("%.2f MB", f/mb)
	case size < tb:
		return fmt.Sprintf("%.2f GB", f/gb)
	default:
		return fmt.Sprintf("%.2f TB", f/tb)
	}
}

func DigitLen(x int) int {
	if x < 10 {
		return 1
	} else if x < 100 {
		return 2
	} else if x < 1000 {
		return 3
	}
	return len(strconv.FormatInt(int64(x), 10))
}

func ClearScreen() (ok bool) {
	var cmd *exec.Cmd
	switch {
	case !term.IsTerminal(int(os.Stdout.Fd())):
		return false
	case runtime.GOOS == "windows":
		cmd = exec.Command("cls")
	default:
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	return cmd.Run() == nil
}

// FastDelete deletes the item at index i from s. The slice will be
// out of order.
//
// FastDelete is faster than [slices.Delete] because it swaps the item to
// be deleted with the last item and then truncates the slice. This should
// be used when the order of the slice doesn't matter.
func FastDelete[T any](s []T, i int) []T {
	_ = s[i] // Bounds check
	if len(s) == 1 {
		s = s[:0]
		return s
	}
	last := len(s) - 1
	s[i], s[last] = s[last], s[i]
	s = s[:last]
	return s
}

// SetLogger sets b's Logger and verbosity. If verbose is true, b.Logger is set
// to [os.Stderr]. If the $KLAR_LOG_FILE environment variable is set, regardless
// of the value of verbose, b.Logger is set to write to that file. Otherwise,
// b.Logger is set to nil. SetLogger returns an error if it fails to
// open $KLAR_LOG_FILE.
func SetLogger(verbose, json bool) (l *slog.Logger, err error) {
	var (
		logFile = os.Getenv("KLAR_LOG_FILE")
		out     io.Writer
		flags   logger.Flags
		level   = slog.LevelInfo
	)
	switch {
	case logFile != "":
		//nolint:gosec // G703 - internal env var only
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create file at %s set by $KLAR_LOG_FILE: %w", logFile, err,
			)
		}
		out = file
		flags |= logger.NoColor
	case verbose || os.Getenv("KLAR_DEBUG") == "1":
		out = os.Stderr
		level = slog.LevelDebug
	default:
		return slog.New(slog.DiscardHandler), nil
	}
	if json {
		return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{
			Level: level,
		})), nil
	}
	return slog.New(logger.NewLogHandler(out, flags)), nil
}

func JoinFunc[T any](s []T, fn func(T) string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	var b strings.Builder
	for i, v := range s {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(fn(v))
	}
	return b.String()
}

func JoinColorFunc[T any](items []T, color string, f func(T) string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	if ansi.DisableColor {
		return JoinFunc(items, f, sep)
	}
	return ansi.Color(color, JoinFunc(items, f, ansi.ListSeparator(color, sep)))
}

type DecodeUnion[A, B any] struct {
	IsB bool
	v   any
}

func (u DecodeUnion[A, B]) A() A         { return u.v.(A) }
func (u DecodeUnion[A, B]) B() B         { return u.v.(B) }
func (u DecodeUnion[_, _]) IsZero() bool { return u.v == nil }

func (u *DecodeUnion[A, B]) UnmarshalJSON(data []byte) error {
	u.v = nil
	var a A
	if err := json.Unmarshal(data, &a); err == nil {
		u.IsB = false
		u.v = a
		return nil
	}
	var b B
	err := json.Unmarshal(data, &b)
	if err == nil {
		u.IsB = true
		u.v = b
		return nil
	}
	return fmt.Errorf("input couldn't be decoded into %T or %T: %w", a, b, err)
}

func HexToRGB(hex string) (r, g, b int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		rs, gs, bs := string(hex[0]), string(hex[1]), string(hex[2])
		hex = (rs + rs) + (gs + gs) + (bs + bs)
	}
	// Use bitSize 9 instead of 8 due to signedness
	r64, err1 := strconv.ParseInt(hex[:2], 16, 9)
	g64, err2 := strconv.ParseInt(hex[2:4], 16, 9)
	b64, err3 := strconv.ParseInt(hex[4:6], 16, 9)
	if err := cmp.Or(err1, err2, err3); err != nil {
		panic(fmt.Sprintf("invalid hex %q: %v", hex, err))
	}
	return int(r64), int(g64), int(b64)
}
