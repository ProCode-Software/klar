// Package jswriter writes JavaScript IR to JavaScript source code.
package jswriter

import (
	"bufio"
	"fmt"
	"io"

	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

type Writer struct {
	writer     *bufio.Writer
	IndentSize int // Number of spaces to use for indentation. 0 = minify
	level      int // Current indentation level
}

func NewWriter(w io.Writer, indent int) *Writer {
	return &Writer{writer: bufio.NewWriter(w), IndentSize: indent}
}

func WriteModule(mod *jsir.Module, wr io.Writer, indent int) error {
	return NewWriter(wr, indent).WriteNode(mod)
}

type writeError struct{ error }

func (w *Writer) WriteNode(node any) (err error) {
	defer func() {
		switch r := recover().(type) {
		case nil:
		case writeError:
			err = r.error
			w.writer.Flush()
		default:
			panic(r)
		}
	}()
	switch node := node.(type) {
	case *jsir.Module:
		w.writeModule(node)
	case jsir.Statement:
		w.writeStatement(node)
	case jsir.Expression:
		w.writeExpression(node)
	default:
		panic(fmt.Sprintf("invalid node: %T", node))
	}
	if err = w.writer.Flush(); err != nil {
		return err
	}
	return nil
}

func (w *Writer) writeModule(mod *jsir.Module) {
	// TODO: Comments
	w.writeStatementList(mod.Statements)
}

func (w *Writer) writeString(s string) {
	if _, err := w.writer.WriteString(s); err != nil {
		panic(writeError{err})
	}
}

func (w *Writer) writeByte(c byte) {
	if err := w.writer.WriteByte(c); err != nil {
		panic(writeError{err})
	}
}

func (w *Writer) write(b []byte) {
	if _, err := w.writer.Write(b); err != nil {
		panic(writeError{err})
	}
}

func (w *Writer) increaseLevel() { w.level++ }
func (w *Writer) decreaseLevel() { w.level-- }

func (w *Writer) writeIndent() {
	if w.IndentSize > 0 {
		w.write(char.Repeat(' ', w.IndentSize*w.level))
	}
}
