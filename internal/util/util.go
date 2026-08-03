package util

import (
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

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
