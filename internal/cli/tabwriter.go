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
	WrapTerminalColumns
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
	WrapIndent int  // Indent for wrapped lines
	PadChar    byte // Pad character between printed cells, ' ' by default
	Separator  byte // Separator character, '\t' by default
	MarginChar byte // Margin character, ' ' by default
	TermWidth  int  // File descriptor of terminal
	Flags      TabWriterFlags

	isInit     bool
	cells      [][]cell
	cellWidths []int
	colCap     int
	currLine   int

	rem        []byte // Bytes after last terminated cell
	remExclude int
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

func (tw *TabWriter) fill(n int) []byte { return char.Repeat(tw.PadChar, n) }
func (tw *TabWriter) margin() []byte    { return char.Repeat(tw.MarginChar, tw.Margin) }

// Flush writes the calculated cells to tw.Output, returning the number of bytes written,
// and any error that occurred while writing.
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
	// Calculate width of terminal and width of preceding columns
	var precColWidth int
	if tw.Flags&WrapTerminalColumns != 0 {
		precColWidth = tw.Margin
		for _, colSize := range tw.cellWidths[:len(tw.cellWidths)-1] {
			precColWidth += colSize + tw.Spacing
		}
	}
	for _, line := range tw.cells {
		if err := write(tw.margin()); err != nil {
			return n, err
		}
		for colI, cell := range line {
			if tw.Flags&DiscardEmptyColumns != 0 && cell.length == 0 {
				continue
			}
			var (
				offset = tw.fill(tw.cellWidths[colI] - cell.length)
				space  = tw.fill(tw.Spacing)
				isLast = colI == len(line)-1
			)
			switch {
			case tw.TermWidth > 0 && isLast && tw.Flags&WrapTerminalColumns != 0 &&
				cell.length+precColWidth > tw.TermWidth:
				writeArray = tw.wrapCell(writeArray, cell, precColWidth)
				isLast = false // Avoid clearing 2 items in writeArray
			default:
				writeArray = [][]byte{cell.content, offset, space}
			case tw.Flags&AlignCenter != 0:
				half := len(offset) / 2
				writeArray = [][]byte{offset[:half], cell.content, offset[half:], space}
			case tw.Flags&AlignRight != 0:
				writeArray = [][]byte{offset, cell.content, space}
			}
			if isLast { // Trim whitespace at end of line
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

// Left align only. TODO: fix
func (tw *TabWriter) wrapCell(writeArray [][]byte, cell cell, prevColWidth int) [][]byte {
	target := tw.TermWidth - prevColWidth
	writeArray = writeArray[:0]
	if estSize := (cell.length/target + 1) * 3; cap(writeArray) < estSize {
		writeArray = make([][]byte, 0, estSize)
	}
	cont, next := tw.noBreakWords(cell.content, 0, target)
	writeArray = append(writeArray, cont)
	target -= tw.WrapIndent
	for next < len(cell.content) {
		cont, next = tw.noBreakWords(cell.content, next, next+target)
		writeArray = append(
			writeArray,
			newline, tw.margin(), tw.fill(prevColWidth+tw.WrapIndent-tw.Margin), cont,
		)
	}
	return writeArray
}

func (tw *TabWriter) noBreakWords(b []byte, start, targetLen int) (wrapped []byte, next int) {
	// Calculate visible length accounting for ANSI escapes
	visibleLen := 0
	bytePos := start
	lastSpace := -1

	for bytePos < len(b) && visibleLen < targetLen {
		if b[bytePos] == '\x1b' {
			// Skip ANSI escape sequence
			escapeLen := tw.readANSIEscape(b[bytePos:])
			bytePos += escapeLen + 1
			continue
		}
		if b[bytePos] == ' ' {
			lastSpace = bytePos
		}
		visibleLen++
		bytePos++
	}

	// If we hit the target and there's more content, try to break at last space
	if bytePos < len(b) && lastSpace > start {
		return b[start : lastSpace+1], lastSpace + 1
	}

	return b[start:bytePos], bytePos
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
		tw.remExclude += exclude
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
