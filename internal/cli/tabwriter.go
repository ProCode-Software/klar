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
	Output    io.Writer
	Spacing   int  // Spacing between columns
	MinWidth  int  // Minimum width of each column
	PadChar   byte // Pad character, ' ' by default
	Separator byte // Separator character, '\t' by default
	Flags     TabWriterFlags

	isInit     bool
	cells      [][]cell
	colCap     int
	currLine   int
	padBytes   []byte
	cellWidths []int
}

type cell struct {
	content []byte
	length  int
}

// NewTabWriter returns a [*TabWriter] with the recommended settings,
func NewTabWriter() *TabWriter {
	return &TabWriter{
		Output:    os.Stdout,
		Spacing:   2,
		Separator: '\t',
		PadChar:   ' ',
		isInit:    true,
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
	if n > len(tw.padBytes) || (len(tw.padBytes) > 0 && tw.padBytes[0] != tw.Separator) {
		tw.padBytes = char.Repeat(tw.Separator, n)
	}
	return tw.padBytes[:n]
}

// Flush writes the calculated cells to tw.Output, returning the number of bytes written,
// and any error that occured while writing.
func (tw *TabWriter) Flush() (n int, err error) {
	var writeArray [][]byte
	tw.init()
	/* if len(tw.cells) > 0 {
		tw.cells = tw.cells[:len(tw.cells)-1] // Remove last empty row
	} */
	for _, line := range tw.cells {
		for colI, cell := range line {
			if tw.Flags&DiscardEmptyColumns != 0 && cell.length == 0 {
				continue
			}
			offset := tw.fill(tw.cellWidths[colI] - cell.length)
			switch {
			default:
				writeArray = [][]byte{cell.content, offset}
			case tw.Flags&AlignCenter != 0:
				half := len(offset) / 2
				writeArray = [][]byte{offset[:half], cell.content, offset[half:]}
			case tw.Flags&AlignRight != 0:
				writeArray = [][]byte{cell.content, offset}
			}
			for _, seg := range writeArray {
				wn, err := tw.Output.Write(seg)
				n += wn
				if err != nil {
					return n, err
				}
			}
		}
		wn, err := tw.Output.Write(newline)
		n += wn
		if err != nil {
			return n, err
		}
	}
	return
}

// Write writes multiple strings to tw and breaks the row. Cells are escaped to avoid
// breaking between strings.
func (tw *TabWriter) Write(cells ...string) {
	tw.init()
	for _, col := range cells {
		tw.readCell([]byte(col), true)
	}
	tw.breakLine()
}

// WriteString is equivalent to tw.WriteBytes but accepts a string.
func (tw *TabWriter) WriteString(s string) {
	tw.WriteBytes([]byte(s))
}

// WriteBytes writes b, calculating the cells in it.
func (tw *TabWriter) WriteBytes(b []byte) {
	tw.init()
	tw.readCell(b, false)
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
		ansiEndMin = 0x40
		start = 2
	}
	for _, c := range b[start:] {
		escapeLen++
		if c >= ansiEndMin && c <= ansiEndMax {
			break
		}
	}
	return
}

func (tw *TabWriter) readCell(b []byte, isEscape bool) {
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
			if cellStart == i {
				cellStart += escapeLen + 1
			}
			exclude += escapeLen + 1
			i += escapeLen
		case tw.Separator:
			if isEscape {
				continue
			}
			line = append(line, cell{content: b[cellStart:i]})
			tw.evalCellWidth(line, exclude)
			cellStart = i + 1 // Check if next character is counted or not
		case '\n':
			if isEscape {
				continue
			}
			line = append(line, cell{content: b[cellStart:i]}) // Exclude \n
			tw.evalCellWidth(line, exclude)
			tw.breakLine()
			line = tw.cells[tw.currLine]
		}
	}
	// Last cell
	if cellStart < len(b)-1 {
		line = append(line, cell{content: b[cellStart:]})
		tw.evalCellWidth(line, exclude)
	}
}

func (tw *TabWriter) breakLine() {
	tw.currLine++
	tw.cells = append(tw.cells, make([]cell, 0, tw.colCap))
}

func (tw *TabWriter) evalCellWidth(line []cell, exclude int) {
	col := len(line) - 1
	width := utf8.RuneCount(line[col].content)
	line[col].length = width - exclude
	if len(tw.cellWidths) <= col {
		tw.cellWidths = append(tw.cellWidths, max(width, tw.MinWidth))
		return
	}
	tw.cellWidths[col] = max(width, tw.cellWidths[col], tw.MinWidth)
}
