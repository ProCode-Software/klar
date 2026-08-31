package klarast

import (
	"regexp"
	"strconv"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/pkg/lsp"
)

var HexColorRegex = regexp.MustCompile(`#[0-9a-fA-F]{3,8}`)

// Each float is in the range [0, 1]
func GetColors(prog *ast.Program) (colors []lsp.Color) {
	prog.Walk(func(c *ast.Cursor) ast.StopCode {
		node, ok := c.Node().(*ast.StringLiteral)
		if !ok || len(node.Fragments) != 1 {
			return ast.ContinueWalk
		}
		text, ok := node.Fragments[0].(ast.TextFragment)
		if !ok || !HexColorRegex.MatchString(text.Source) {
			return ast.ContinueWalk
		}
		if color := ParseHex(text.Source); color != nil {
			colors = append(colors, *color)
		}
		return ast.ContinueWalk
	}, nil)
	return colors
}

// Hex includes the '#' prefix.
func ParseHex(hex string) *lsp.Color {
	var rs, gs, bs, as string
	hex = hex[1:]
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
		return nil
	}
	color := [4]float64{0, 0, 0, 1} // Default alpha
	for i, hex := range [...]string{rs, gs, bs, as} {
		if hex == "" {
			continue
		}
		if len(hex) == 1 {
			hex += hex // #rgb -> #rrggbb
		}
		dec, err := strconv.ParseUint(hex, 16, 8)
		if err != nil {
			return nil
		}
		color[i] = float64(dec) / 255
	}
	return &lsp.Color{color[0], color[1], color[2], color[3]}
}
