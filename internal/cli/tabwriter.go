package cli

import (
	"io"
	"os"
	"slices"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/char"
)

type TabWriterFlags uint8

const (
	AlignCenter TabWriterFlags = 1 << iota
	AlignRight
	DiscardEmptyColumns
)

const (
	TabEscape   = '\xFF'
	TabUnescape = '\xFE'
)

var newline = []byte{'\n'}

// A TabWriter writes bytes to a writer with vertical column alignment. A zero TabWriter
// is ready for use. Unlike Go's [text/tabwriter.Writer],
type TabWriter struct {
	Output     io.Writer
	Spacing    int  // Spacing between columns
	MinWidth   int  // Minimum width of each column
	Margin     int  // Left margin of each line
	PadChar    byte // Pad character between printed cells, ' ' by default
	Separator  byte // Separator character, '\t' by default
	MarginChar byte // Margin character, ' ' by default
	Flags      TabWriterFlags

	isInit     bool
	cells      [][]cell
	cellWidths []int
	colCap     int
	currLine   int

	rem        []byte // Bytes after last terminated cell
	remExclude int

	padBytes    []byte
	marginBytes []byte
}

type cell struct {
	content []byte
	length  int
}

// NewTabWriter returns a [*TabWriter] with the recommended settings
func NewTabWriter() *TabWriter {
	return &TabWriter{
		Output:     os.Stdout,
		Spacing:    2,
		Separator:  '\t',
		MarginChar: ' ',
		PadChar:    ' ',
		isInit:     true,
	}
}

// NewTabWriter returns a [*TabWriter] with the recommended settings and
// Output set to w.
func NewTabWriterOutput(w io.Writer) *TabWriter {
	return &TabWriter{
		Output:     w,
		Spacing:    2,
		Separator:  '\t',
		PadChar:    ' ',
		MarginChar: ' ',
		isInit:     true,
	}
}

func (tw *TabWriter) init() {
	if tw.isInit {
		return
	}
	if tw.Output == nil {
		tw.Output = os.Stdout
	}
	if tw.Separator == 0 {
		tw.Separator = '\t'
	}
	if tw.PadChar == 0 {
		tw.PadChar = ' '
	}
	if tw.MarginChar == 0 {
		tw.MarginChar = ' '
	}
	tw.isInit = true
}

// ReserveCapacity expands the capacity of rows and columns that can be stored.
// ReserveCapacity panics if lines < 0 or cols < 0.
func (tw *TabWriter) ReserveCapacity(lines, cols int) {
	if lines < 0 || cols < 0 {
		panic("tw.ReserveCapacity(lines, cols): lines or cols cannot be negative")
	}
	tw.colCap = cols
	tw.cells = slices.Grow(tw.cells, lines)
	tw.cellWidths = slices.Grow(tw.cellWidths, cols)
}

func (tw *TabWriter) fill(n int) []byte {
	if n > len(tw.padBytes) || (len(tw.padBytes) > 0 && tw.padBytes[0] != tw.PadChar) {
		tw.padBytes = char.Repeat(tw.PadChar, n)
	}
	return tw.padBytes[:n]
}

func (tw *TabWriter) margin() []byte {
	if tw.MarginChar == tw.PadChar {
		return tw.fill(tw.Margin)
	}
	if tw.Margin > len(tw.marginBytes) ||
		(len(tw.marginBytes) > 0 && tw.marginBytes[0] != tw.MarginChar) {
		tw.marginBytes = char.Repeat(tw.MarginChar, tw.Margin)
	}
	return tw.marginBytes[:tw.Margin]
}

// Flush writes the calculated cells to tw.Output, returning the number of bytes written,
// and any error that occured while writing.
func (tw *TabWriter) Flush() (n int, err error) {
	var writeArray [][]byte
	tw.init()
	write := func(b []byte) error {
		wn, err := tw.Output.Write(b)
		n += wn
		return err
	}
	// Remove last empty row
	if l := len(tw.cells); l > 0 && len(tw.cells[l-1]) == 0 {
		tw.cells = tw.cells[:len(tw.cells)-1]
	}
	for _, line := range tw.cells {
		if err := write(tw.margin()); err != nil {
			return n, err
		}
		for colI, cell := range line {
			if tw.Flags&DiscardEmptyColumns != 0 && cell.length == 0 {
				continue
			}
			offset := tw.fill(tw.cellWidths[colI] - cell.length)
			space := tw.fill(tw.Spacing)
			switch {
			default:
				writeArray = [][]byte{cell.content, offset, space}
			case tw.Flags&AlignCenter != 0:
				half := len(offset) / 2
				writeArray = [][]byte{offset[:half], cell.content, offset[half:], space}
			case tw.Flags&AlignRight != 0:
				writeArray = [][]byte{cell.content, offset, space}
			}
			if colI == len(line)-1 { // Trim whitespace at end of line
				writeArray = writeArray[:len(writeArray)-2]
			}
			for _, seg := range writeArray {
				if err = write(seg); err != nil {
					return n, err
				}
			}
		}
		if err := write(newline); err != nil {
			return n, err
		}
	}
	// Reset cells after flushing
	tw.cells = tw.cells[:0]
	tw.cellWidths = tw.cellWidths[:0]
	tw.currLine = 0
	return
}

// WriteCells writes multiple strings to tw and breaks the row. Cells are escaped to avoid
// breaking between strings.
func (tw *TabWriter) WriteCells(cells ...string) {
	tw.init()
	for _, col := range cells {
		tw.readCell([]byte(col), true, true)
	}
	tw.breakLine()
}

// WriteString is equivalent to tw.WriteBytes but accepts a string.
// WriteString always returns len(b) and a nil error.
func (tw *TabWriter) WriteString(s string) (int, error) {
	return tw.Write([]byte(s))
}

// Write writes b, calculating the cells in it. Write always returns len(b) and a nil error.
func (tw *TabWriter) Write(b []byte) (int, error) {
	tw.init()
	tw.readCell(b, false, false)
	return len(b), nil
}

const ansiEndMax byte = 0x7E

// escapeLen does not include the '\x1b' byte, but b does.
func (tw *TabWriter) readANSIEscape(b []byte) (escapeLen int) {
	if len(b) < 2 {
		return 0
	}
	var ansiEndMin byte = 0x30
	start := 1
	if b[1] == '[' {
		ansiEndMin, start = 0x40, 2
		escapeLen++
	}
	for _, c := range b[start:] {
		escapeLen++
		if c >= ansiEndMin && c <= ansiEndMax {
			break
		}
	}
	return
}

func (tw *TabWriter) readCell(b []byte, isEscape, breakAfter bool) {
	var cellStart, exclude int
	if len(tw.cells) <= tw.currLine {
		tw.cells = append(tw.cells, make([]cell, 0, tw.colCap))
	}
	line := tw.cells[tw.currLine]
	for i, c := range b {
		switch c {
		case TabEscape, TabUnescape:
			isEscape = c == TabEscape
			if cellStart == i {
				cellStart++
			}
			exclude++
		case '\x1b':
			// ANSI escape
			escapeLen := tw.readANSIEscape(b[i:])
			exclude += escapeLen + 1
			i += escapeLen
		case tw.Separator:
			if isEscape {
				continue
			}
			content := append(tw.rem, b[cellStart:i]...) // Add previous bytes
			line = append(line, cell{content: content})
			tw.evalLastCellWidth(line, exclude+tw.remExclude)
			tw.rem, tw.remExclude = nil, 0
			cellStart, exclude = i+1, 0
		case '\n':
			if isEscape {
				continue
			}
			content := append(tw.rem, b[cellStart:i]...) // Add previous bytes
			line = append(line, cell{content: content})  // Exclude \n
			tw.evalLastCellWidth(line, exclude+tw.remExclude)
			tw.rem, tw.remExclude = nil, 0
			cellStart, exclude = i+1, 0

			tw.cells[tw.currLine] = line // Apply current line
			tw.breakLine()
			line = tw.cells[tw.currLine] // Get next line
		}
	}
	// Last cell
	switch {
	case cellStart >= len(b):
	case breakAfter:
		line = append(line, cell{content: b[cellStart:]})
		tw.evalLastCellWidth(line, exclude+tw.remExclude)
	default:
		tw.rem = append(tw.rem, b[cellStart:]...)
		tw.remExclude = exclude
	}
	tw.cells[tw.currLine] = line
}

func (tw *TabWriter) breakLine() {
	tw.currLine++
	tw.cells = append(tw.cells, make([]cell, 0, tw.colCap))
	tw.rem, tw.remExclude = nil, 0
}

func (tw *TabWriter) evalLastCellWidth(line []cell, exclude int) {
	col := len(line) - 1
	width := utf8.RuneCount(line[col].content)
	width -= exclude
	line[col].length = width
	if len(tw.cellWidths) <= col {
		tw.cellWidths = append(tw.cellWidths, max(width, tw.MinWidth))
		return
	}
	tw.cellWidths[col] = max(width, tw.cellWidths[col], tw.MinWidth)
}
