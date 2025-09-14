package cli

import (
	"io"
	"text/tabwriter"
)

type TabWriter struct {
	Output io.Writer
	Spacing int // Spacing between columns
 	MinWidth int // Min width of each column
	TabSize int // Number of spaces for tab, one '\t' character if 0
}
