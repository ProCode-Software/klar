package reader

import "github.com/ProCode-Software/klar/pkg/klarml/ast"

const MaxDepth = 10000

func (rd *Reader) ParseDocument() (*ast.Document, []error) {
	var res ast.Value
	tok := rd.readToken()
	switch tok.Kind {
	case EOF:
		none := &ast.None{}
		none.SetPos(tok.Pos, tok.Pos)
		doc := &ast.Document{Body: none}
		doc.SetPos(tok.Pos, tok.Pos)
		return doc, rd.errs
	default:
		res = rd.parseValue(tok)
	}
	doc := &ast.Document{Variables: rd.vars, Body: res}
	tok = rd.readToken()
	doc.SetPos(res.Pos().Start, tok.Pos)
	if tok.Kind != EOF {
		rd.errs = append(rd.errs, &ParseError{
			Code:  ErrExpectedEOF,
			Range: TokenRange(tok),
		})
	}
	return doc, nil
}


func (rd *Reader) parseValue(tok Token) ast.Value {
	
	return nil
}