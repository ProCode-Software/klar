package cli

import (
	"io"
	"os"
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
}

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

func (tw *TabWriter) ReserveCapacity(lines, cols int) {
	tw.colCap = cols
	tw.cells = make([][]cell, 0, lines)
	tw.cellWidths = make([]int, 0, cols)
}

func (tw *TabWriter) fill(n int) []byte {
	if n > len(tw.padBytes) || (len(tw.padBytes) > 0 && tw.padBytes[0] != tw.Separator) {
		tw.padBytes = char.Repeat(tw.Separator, n)
	}
	return tw.padBytes[:n]
}

func (tw *TabWriter) evalLen() (cols []int) {
	if len(tw.cells) == 0 {
		return
	}
	cols = make([]int, len(tw.cells[0]))
	// Evaluate the length of each column
	for _, line := range tw.cells {
		for colI, cell := range line {
			cols[colI] = max(cols[colI], len(cell.content))
		}
	}
	return
}

func (tw *TabWriter) Flush() (n int, err error) {
	var writeArray [][]byte
	tw.init()
	for _, line := range tw.cells {
		for colI, cell := range line {
			cl := utf8.RuneCount(cell.content)
			if tw.Flags&DiscardEmptyColumns != 0 && cl == 0 {
				continue
			}
			offset := tw.fill(tw.cellWidths[colI] - cl)
			switch {
			default:
				writeArray = [][]byte{cell.content, offset}
			case tw.Flags&AlignCenter != 0:
				half := len(offset) / 2
				// If cell ends in newline, add after the spaces
				if cl > 0 && cell.content[cl-1] == '\n' {
					writeArray = [][]byte{
						offset[:half], cell.content[:cl-1], offset[half:], {'\n'},
					}
				} else {
					writeArray = [][]byte{offset[:half], cell.content, offset[half:]}
				}
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
	}
	return
}

// The row is broken after cols
func (tw *TabWriter) Write(cols ...string) {
	tw.init()
	for _, col := range cols {
		tw.readCell(append([]byte{TabEscape}, col...))
	}
	tw.breakLine()
}

func (tw *TabWriter) WriteString(s string) {
	tw.WriteBytes([]byte(s))
}

func (tw *TabWriter) WriteBytes(b []byte) {
	tw.init()
	tw.readCell(b)
}

// escapeLen does not include the '\x1b' byte, but b does.
func (tw *TabWriter) readANSIEscape(b []byte) (escapeLen int) {
	if len(b) < 2 {
		return 0
	}
	const ansiEndMax byte = 0x7E
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

func (tw *TabWriter) readCell(b []byte) {
	var isEscape bool
	var cellStart int
	if len(tw.cells) <= tw.currLine {
		tw.cells = append(tw.cells, make([]cell, 0, tw.colCap))
	}
	line := tw.cells[tw.currLine]
	for i, c := range b {
		switch c {
		case TabEscape:
			isEscape = true
			if cellStart == i {
				cellStart++
			}
			continue
		case TabUnescape:
			isEscape = false
			if cellStart == i {
				cellStart++
			}
			continue
		case '\x1b':
			// ANSI escape
			escapeLen := tw.readANSIEscape(b[i:])
			if cellStart == i {
				cellStart += escapeLen + 1
			}
			i += escapeLen
			continue
		case tw.Separator:
			if !isEscape && (tw.Flags&DiscardEmptyColumns == 0 || cellStart-i == 0) {
				line = append(line, cell{content: b[cellStart:i]})
				tw.evalCellWidth(line)
				cellStart = i + 1 // Check if next character is counted or not
				continue
			}
		case '\n':
			if isEscape {
				continue
			}
			line = append(line, cell{content: b[cellStart : i+1]})
			tw.evalCellWidth(line)
			tw.breakLine()
			line = tw.cells[tw.currLine]
		}
	}
	// Last cell
	if cellStart < len(b)-1 {
		line = append(line, cell{content: b[cellStart:]})
		tw.evalCellWidth(line)
	}
}

func (tw *TabWriter) breakLine() {
	tw.currLine++
	tw.cells = append(tw.cells, make([]cell, 0, tw.colCap))
}

func (tw *TabWriter) evalCellWidth(line []cell) {
	col := len(line) - 1
	width := utf8.RuneCount(line[col].content)
	if len(tw.cellWidths) <= col {
		tw.cellWidths = append(tw.cellWidths, max(width, tw.MinWidth))
		return
	}
	tw.cellWidths[col] = max(width, tw.cellWidths[col], tw.MinWidth)
}
