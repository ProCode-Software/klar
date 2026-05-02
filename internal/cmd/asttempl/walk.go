package main

import (
	"bytes"
	"fmt"
	"go/types"
	"strings"
)

func GenerateWalk(b *bytes.Buffer, nodes NodeList, pkg Package) error {
	fmt.Fprint(b, Header)
	fmt.Fprintln(b)
	for _, node := range nodes {
		if node.Name() == "Identifier" {
			continue
		}
		fmt.Fprintf(b,
			`func (n *%s) Walk(v Visitor, c *Cursor) StopCode {
	return walkFields(v, n, c,`, node.Name(),
		)
		s := node.Type().Underlying().(*types.Struct)
		for i := range s.NumFields() {
			var (
				f         = s.Field(i)
				typ       string
				l, isList = f.Type().(*types.Slice)
			)
			switch {
			case f.Name() == "BaseNode":
				continue
			case isList && IsNode(l.Elem()):
				// Type parameter
				elm := strings.Replace(l.Elem().String(), pkg.Path()+".", "", 1)
				typ = "walkSlice" + "[" + elm + "]"
			case IsNode(f.Type()):
				typ = "walkNode"
			default:
				continue
			}
			fmt.Fprintf(b, "%s{%d, n.%s}, ", typ, i, f.Name())
		}
		fmt.Fprint(b, ")\n}\n\n")
	}
	return nil
}
