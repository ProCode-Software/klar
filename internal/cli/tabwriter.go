package cli

import (
	"io"
	"os"
)

const TabEscape = '\xFF'

type TabWriter struct {
	Output      io.Writer
	Spacing     int  // Spacing between columns
	MinWidth    int  // Min width of each column
	TabSize     int  // Number of spaces for tab, one '\t' character if 0
	Separator   byte // Separator character, '\t' by default
	AlignCenter bool

	cells [][]cell
	colCap int
}

type cell struct {
	len     int
	content []byte
}

func NewTabWriter() *TabWriter {
	return &TabWriter{
		Output:  os.Stdout,
		Spacing: 2,
	}
}

func (tw *TabWriter) ReserveCapacity(lines, cols int) {
	tw.colCap = cols
	tw.cells = make([][]cell, 0, lines)
}

func (tw *TabWriter) Flush() (n int, err error) {
	
}

func (tw *TabWriter) Write(cols ...string) {
	newCol := make([]cell, 0, tw.colCap)
	for _, col := range cols {
		cell := tw.readCell([]byte(col))
		newCol = append(newCol, cell...)
	}
	tw.cells = append(tw.cells, newCol)
}

func (tw *TabWriter) WriteString(s string) {
}

func (tw *TabWriter) WriteBytes(b []byte) {
}

func (tw *TabWriter) readANSIEscape() (escapeLen int) {

}

func (tw *TabWriter) readCell(b []byte) (cells []cell) {

}