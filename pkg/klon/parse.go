package klon

import (
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

const MaxDepth = 10000

func (rd *reader) parseDocument() (*ast.Document, []error) {
	var (
		tok = rd.currTok()
		res = rd.parseValue(tok)
		doc = &ast.Document{Variables: rd.vars, Body: res}
	)
	if tok.Kind == EOF {
		return doc, rd.errs
	}
	// Check for EOF
	rd.skipLines()
	if tok = rd.currTok(); tok.Kind != EOF {
		rd.tokenError(ErrExpectedEOF, tok, "Expected end of file")
	}
	doc.SetPos(res.Pos().Start, tok.Pos)
	return doc, rd.errs
}

func (rd *reader) parseValue(tok Token) ast.Value {
	var items []ast.Value
	hasNewline := tok.Kind == Newline
	if hasNewline {
		rd.skipLines()
		tok = rd.currTok()
	}
	if (rd.depth == 0 && rd.peekTok().Kind == Colon) ||
		(hasNewline && rd.currTok().Kind == Dash) {
		return rd.parseObject()
	}
	if tok.Kind == EOF {
		return &ast.None{BaseNode: ast.BaseNode{Range: tokenRange(tok)}}
	}
loop:
	for rd.hasTokens() {
		var res ast.Value
		switch tok.Kind {
		case EOF, Newline:
			break loop
		case Comma:
			if len(items) == 0 {
				rd.tokenError(ErrExpectedValue, tok, "Expected a value before comma")
				rd.advanceTok()
			}
			break loop
		case RightBracket, RightCurly:
			if len(items) == 0 {
				rd.tokenError(ErrUnmatchedBracket, tok, "Unmatched '%s'", tok.Src)
				rd.advanceTok()
			}
			break loop
		case Number:
			res = rd.parseNumber(tok)
		case String:
			res = rd.parseString(tok)
		case None:
			res = rd.parseNone(tok)
		case Boolean:
			res = rd.parseBoolean(tok)
		case At:
			res = rd.parseClass(tok)
		case Arrow:
			res = rd.parseRest(tok)
			// Only allowed at top level
			if rd.depth > 0 {
				rd.tokenError(ErrInvalidArrow, tok, "'<-' is only allowed at the top level")
				res = &ast.Bad{BaseNode: ast.BaseNode{res.Pos()}, Value: res}
			}
		case LeftBracket:
			res = rd.parseList(tok)
		case LeftCurly:
			res = rd.parseBracedObject(tok)
		case Dash:
			// Invalid dash
			if rd.depth == 0 {
				rd.tokenError(ErrDashAtTopLevel, tok,
					"Top-level objects and lists can't start with a dash",
				)
				// Still parse the object
				rd.depthUp()
				res = rd.parseObject()
				rd.depthDown()
				res = &ast.Bad{BaseNode: ast.BaseNode{res.Pos()}, Value: res}
			} else {
				rd.tokenError(ErrDashWithoutNewline, tok,
					"An object or list must begin on a new line",
				)
			}
		default:
			rd.tokenError(ErrUnexpectedToken, tok, "Unexpected token")
			// TODO: see if we need to continue or break
			rd.advanceTok()
			continue
		}
		items = append(items, res)
		tok = rd.currTok()
	}
	if len(items) == 1 {
		return items[0]
	}
	return &ast.StringGroup{
		BaseNode: ast.BaseNode{Range: ranges.Between(
			items[0].Pos(),
			items[len(items)-1].Pos(),
		)},
		Values: items,
	}
}

func (rd *reader) parseNumber(num Token) *ast.Number {
	float, err := strconv.ParseFloat(num.Src, 64)
	if err != nil {
		panic(err) // Shouldn't happen
	}
	rd.advanceTok()
	return &ast.Number{
		BaseNode: ast.BaseNode{Range: tokenRange(num)},
		Source:   num.Src,
		Value:    float,
	}
}

func (rd *reader) parseRest(arrow Token) *ast.ArrowRef {
	rd.advanceTok() // <-
	if k := rd.currTok().Kind; k != Variable {
		rd.tokenError(ErrExpectedVarInArrow, arrow, "Expected a variable after '<-'")
		return &ast.ArrowRef{}
	}
	varRef := rd.parseVariable(rd.currTok())
	return &ast.ArrowRef{
		BaseNode: ast.BaseNode{Range: ranges.Range{arrow.Pos, varRef.Pos().End}},
		Var:      varRef,
	}
}

func (rd *reader) parseVariable(vr Token) *ast.VarRef {
	rd.advanceTok()
	var (
		name   string
		err    = vr.Attrs["err"].(Code)
		braces = vr.Attrs["brace"].(bool)
	)
	if braces {
		if err == ErrUnterminatedVar {
			name = vr.Src[2:]
			rd.tokenError(err, vr, "Expected '{' to close variable")
		} else {
			name = vr.Src[1 : len(vr.Src)-1]
		}
	} else {
		name = vr.Src[1:]
	}
	if err == ErrInvalidIdentifier {
		rd.tokenError(err, vr, "Variable name can't begin with a digit")
	}
	return &ast.VarRef{
		BaseNode: ast.BaseNode{Range: tokenRange(vr)},
		Name:     name,
		Braces:   braces,
	}
}

func (rd *reader) parseList(lb Token) *ast.List {
	rd.depthUp()
	defer rd.depthDown()
	oldComma := rd.comma
	rd.comma = true
	rd.advanceTok()

	var items []ast.Value
	for rd.hasTokens() && rd.currTok().Kind != RightBracket {
		items = append(items, rd.parseValue(rd.currTok()))
		if rd.currTok().Kind != RightBracket {
			rd.skipLines()
			rd.expectError(Comma, ErrExpectedToken,
				"Expected ',' between list items or ']' to end list",
			)
		}
	}
	rd.comma = oldComma
	rd.skipLines()
	rb := rd.expectError(RightBracket, ErrUnterminatedList, "Expected ']' to end list")
	return &ast.List{
		BaseNode: ast.BaseNode{ranges.Range{lb.Pos, rb.Pos}},
		Inline:   true,
		Items:    items,
	}
}

func (rd *reader) parseString(str Token) *ast.String {
	rd.advanceTok()
	var (
		wrap  bool
		src   string
		quote rune
	)
	if str.Attrs == nil || str.Attrs["quote"].(rune) == 0 {
		src = str.Src
	} else {
		wrap = str.Attrs["wrap"].(bool)
		quote = str.Attrs["quote"].(rune)
		if wrap {
			src = str.Src[2:]
		} else {
			src = str.Src[1:]
		}
		if str.Attrs["unterm"].(bool) {
			rd.tokenError(ErrUnterminatedString, str, "This string was left open")
		} else {
			src = src[:len(src)-1]
		}
	}
	// Read segments
	var b strings.Builder
	b.Grow(len(src))
	var segments []string
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch c {
		case '\\':
			if i+1 >= len(src) {
				continue // End of file; error already reported for unterminated
			}
			c := src[i+1]
			char, ok := getEscape(c)
			if ok {
				b.WriteByte(char)
				i++ // Skip escape letter
				continue
			}
			// Unknown escape; keep in string
			rd.tokenError(ErrUnknownEscape, str, `Unknown escape sequence '\%c'`, c)
			b.WriteByte('\\')
			b.WriteByte(c)
		case '$':
			if i+1 >= len(src) {
				// '$' is last character; leave
				b.WriteByte(c)
				continue
			}
			// Read a variable name
			start := i + 1
			if src[start] == '{' {
				start++
			}
			var end int
			for i, r := range src[start:] {
				if r == '}' || !isValidIdentChar(r) {
					end = i
					break
				}
			}
			// Write normalized variable name as segment (no braces)
			varName := src[start : start+end]
			if len(varName) == 0 {
				b.WriteByte('$')
				continue
			}
			// End current segment
			segments = append(segments, b.String())
			b.Reset()
			// Add variable as own segment
			b.WriteByte('$')
			b.WriteString(varName)
			segments = append(segments, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}

	return &ast.String{
		BaseNode: ast.BaseNode{Range: tokenRange(str)},
		Raw:      src,
		Value:    segments,
		Wrap:     wrap,
		Quote:    quote,
	}
}

func getEscape(c byte) (byte, bool) {
	escapes := map[byte]byte{
		'f': '\f', 'n': '\n', 'r': '\r', 't': '\t', 'v': '\v', 'e': '\x1b',
		'\\': '\\', '"': '"', '\'': '\'', '$': '$',
	}
	esc, ok := escapes[c]
	return esc, ok
}

func (rd *reader) parseObject() ast.Value {
	panic("not implemented")
	return nil
}

func (rd *reader) parseBracedObject(lc Token) *ast.Object {
	return nil
}

func (rd *reader) parseBoolean(b Token) *ast.Boolean {
	rd.advanceTok()
	return &ast.Boolean{
		BaseNode: ast.BaseNode{Range: tokenRange(b)},
		Value:    b.Attrs["value"].(bool),
	}
}

func (rd *reader) parseNone(none Token) *ast.None {
	rd.advanceTok()
	return &ast.None{BaseNode: ast.BaseNode{Range: tokenRange(none)}}
}

func (rd *reader) parseClass(at Token) *ast.Class {
	return nil
}
