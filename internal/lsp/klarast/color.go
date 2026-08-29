package klarast

import (
	"regexp"
	"strconv"

	"github.com/ProCode-Software/klar/internal/ast"
)

var hexRegex = regexp.MustCompile(`#[0-9a-fA-F]{3,8}`)

// Each float is in the range [0, 1]
func GetColors(prog *ast.Program) (colors [][4]float64) {
	prog.Walk(func(c *ast.Cursor) ast.StopCode {
		node, ok := c.Node().(*ast.StringLiteral)
		if !ok || len(node.Fragments) != 1 {
			return 0
		}
		text, ok := node.Fragments[0].(ast.TextFragment)
		if !ok || !hexRegex.MatchString(text.Source) {
			return 0
		}
		var rs, gs, bs, as string
		hex := text.Source[1:]
		switch len(hex) {
		case 3:
			rs, gs, bs = hex[0:1], hex[1:2], hex[2:3]
		case 4:
			rs, gs, bs, as = hex[0:1], hex[1:2], hex[2:3], hex[3:4]
		case 6:
			rs, gs, bs = hex[0:2], hex[2:4], hex[4:6]
		case 8:
			rs, gs, bs, as = hex[0:2], hex[2:4], hex[4:6], hex[6:8]
		default:
			return 0
		}
		var color [4]float64
		for i, s := range []string{rs, gs, bs, as} {
			if s == "" {
				continue
			}
			n, err := strconv.ParseUint(s, 16, 8)
			if err != nil {
				return 0
			}
			color[i] = float64(n) / 255
		}
		colors = append(colors, color)
		return 0
	}, nil)
	return colors
}
