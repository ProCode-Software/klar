package decode

import "github.com/ProCode-Software/klar/pkg/klarml/ast"

func (d *Decoder) parseLiteral() ast.Value {
	var lit ast.Value
	switch d.Curr() {
	case '\'', '"':
		// String literal
		lit = d.parseStringLit()
	}
}

func (d *Decoder) parseStringLit() *ast.String {
	quote := d.Advance()
	s := &ast.String{
		Quote: map[byte]int{
			'\'': ast.SingleQuote,
			'"':  ast.DoubleQuote,
		}[quote],
	}
}
