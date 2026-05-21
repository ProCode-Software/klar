package klon

import (
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

const MaxDepth = 10000

func (rd *reader) parseDocument() (*ast.Document, []error) {
	var (
		tok = rd.currTok()
		res = rd.parseValue(tok)
		doc = &ast.Document{Variables: rd.vars, Body: res, Comments: rd.comments}
	)
	if tok.Kind == EOF {
		return doc, rd.errs
	}
	// Check for EOF
	rd.skipLines()
	if tok = rd.currTok(); tok.Kind != EOF {
		rd.tokenError(ErrExpectedEOF, tok, "Expected end of file")
	}
	doc.Comments = rd.comments
	doc.SetPos(res.Pos().Start, tok.Pos)
	return doc, rd.errs
}

func (rd *reader) parseValue(tok Token) ast.Value {
	var (
		items     []ast.Value
		separated []bool
	)
	hasNewline := tok.Kind == Newline
	if hasNewline {
		rd.skipLines()
		tok = rd.currTok()
	}
	if (rd.depth == 0 && rd.peekTok().Kind == Colon) ||
		((hasNewline || tok.Pos == (lexer.Position{Line: 1, Col: 1})) && tok.Kind == Dash) {
		return rd.parseObject()
	}
	if tok.Kind == EOF {
		return &ast.None{BaseNode: ast.BaseNode{Range: tok.Range()}}
	}
	startLine := tok.Pos.Line
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
		case Variable:
			res = rd.parseVariable(tok)
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
				rd.advanceTok()
				continue
			}
		default:
			rd.tokenError(ErrUnexpectedToken, tok, "Unexpected token")
			rd.advanceTok()
			continue
		}
		items = append(items, res)
		prevEnd := res.Pos().End
		tok = rd.currTok()
		if tok.Pos.Line != startLine {
			break
		}
		separated = append(separated, tok.Pos != prevEnd)
	}
	if len(items) == 0 {
		return &ast.None{BaseNode: ast.BaseNode{Range: tok.Range()}}
	}
	if len(items) == 1 {
		return items[0]
	}
	return &ast.StringGroup{
		BaseNode: ast.BaseNode{Range: ranges.Between(
			items[0].Pos(),
			items[len(items)-1].Pos(),
		)},
		Values:    items,
		Separated: separated,
	}
}

func (rd *reader) parseNumber(num Token) *ast.Number {
	float, err := strconv.ParseFloat(num.Src, 64)
	if err != nil {
		panic(err) // Shouldn't happen
	}
	rd.advanceTok()
	return &ast.Number{
		BaseNode: ast.BaseNode{Range: num.Range()},
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
		BaseNode: ast.BaseNode{ranges.Range{arrow.Pos, varRef.Pos().End}},
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
		BaseNode: ast.BaseNode{Range: vr.Range()},
		Name:     name,
		Braces:   braces,
	}
}

func (rd *reader) parseList(lb Token) *ast.List {
	oldFlags := rd.addParseFlags(noComma)
	defer rd.resetParseFlags(oldFlags)

	rd.depthUp()
	defer rd.depthDown()
	rd.advanceTok() // [

	var items []ast.Value
	for rd.hasTokens() && rd.currTok().Kind != RightBracket {
		old := rd.addParseFlags(allowDot)
		res := rd.parseValue(rd.currTok())
		items = append(items, res)
		rd.resetParseFlags(old)
		if rd.currTok().Kind != RightBracket {
			rd.skipLines()
			rd.expectError(Comma, ErrExpectedToken,
				"Expected ',' between list items or ']' to end list",
			)
		}
	}
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
		BaseNode: ast.BaseNode{Range: str.Range()},
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

func (rd *reader) parseKey() ast.Value {
	tok := rd.currTok()
	switch tok.Kind {
	case Number:
		return rd.parseNumber(tok)
	case String:
		return rd.parseString(tok)
	case Boolean:
		return rd.parseBoolean(tok)
	default:
		rd.tokenError(ErrInvalidKey, tok,
			"Expected a key (string, number, or boolean)",
		)
		rd.advanceTok()
		return &ast.Bad{BaseNode: ast.BaseNode{Range: tok.Range()}}
	}
}

func (rd *reader) readKey() (ast.Value, *[]ast.Value, int) {
	dashes := rd.lastDashes
	rd.lastDashes = 0
	for rd.currTok().Kind == Dash {
		dashes++
		rd.advanceTok()
	}
	if dashes < rd.depth {
		rd.lastDashes = dashes
		return nil, nil, dashes
	}
	if dashes > rd.depth+1 {
		rd.tokenError(ErrDashSkip, rd.currTok(),
			"Too many dashes: expected up to %d, but found %d", rd.depth, dashes,
		)
	}
	key := rd.parseKey()
	if rd.currTok().Kind != String || rd.currTok().Src != "." {
		return key, nil, dashes
	}
	path := []ast.Value{key}
	for rd.currTok().Kind == String && rd.currTok().Src == "." {
		rd.advanceTok() // .
		path = append(path, rd.parseKey())
	}
	return path[len(path)-1], &path, dashes
}

func (rd *reader) parseObject() ast.Value {
	var (
		fields   []*ast.Field
		items    []ast.Value
		start    = rd.currTok().Pos
		isDashed bool
		isList   bool
	)
	for rd.hasTokens() {
		tok := rd.currTok()
		if tok.Kind == Variable {
			varRef := rd.parseVariable(tok)
			if rd.depth != 0 {
				rd.tokenError(ErrVarNotTopLevel, tok, "Variables must be declared at the top level")
			}
			if varRef.Braces {
				rd.tokenError(ErrInvalidVariableDecl, tok, "Variable declarations can't use braces")
			}
			rd.expectError(Colon, ErrExpectedToken, "Expected ':' after variable name")
			old := rd.addParseFlags(allowDot)
			val := rd.parseValue(rd.currTok())
			rd.resetParseFlags(old)
			if rd.vars == nil {
				rd.vars = make(map[string]ast.Value)
			}
			rd.vars[varRef.Name] = val
			rd.skipLines()
			continue
		}
		if tok.Kind == Arrow {
			arrowRef := rd.parseRest(tok)
			if isList {
				items = append(items, arrowRef)
			} else {
				fields = append(fields, &ast.Field{
					BaseNode: ast.BaseNode{Range: arrowRef.Pos()},
					Value:    arrowRef,
				})
			}
			rd.skipLines()
			continue
		}
		key, path, dashes := rd.readKey()
		if key == nil {
			break
		}

		// Handle depth transition on first field/item
		if len(fields) == 0 && len(items) == 0 && dashes == rd.depth+1 {
			rd.depthUp()
			isDashed = true
		} else if dashes != rd.depth {
			if dashes < rd.depth {
				break
			}
			rd.tokenError(ErrDashSkip, rd.currTok(),
				"Unexpected dash depth %d (expected %d)", dashes, rd.depth,
			)
		}

		if rd.currTok().Kind == Colon {
			if isList {
				rd.tokenError(ErrUnexpectedToken, rd.currTok(), "Unexpected ':' in list")
			}
			rd.advanceTok() // :
			old := rd.addParseFlags(allowDot)
			val := rd.parseValue(rd.currTok())
			rd.resetParseFlags(old)
			fields = append(fields, &ast.Field{
				BaseNode: ast.BaseNode{Range: ranges.Between(
					key.Pos(), val.Pos(),
				)},
				Key:     key,
				KeyPath: path,
				Value:   val,
			})
		} else {
			if len(fields) > 0 {
				rd.tokenError(ErrExpectedToken, rd.currTok(), "Expected ':' after key")
			}
			isList = true
			// If it's a list item, the "key" was actually the first part of a value
			// We need to parse the rest of the line as a StringGroup if necessary
			if rd.currTok().Pos.Line == key.Pos().End.Line && rd.currTok().Kind != Newline && rd.currTok().Kind != EOF {
				// Wait, parseValue will call parseObject again if it sees a dash.
				// This is not quite right.
				// We should probably just call parseValue with the key already parsed.
				items = append(items, rd.parseValueWithFirst(key))
			} else {
				items = append(items, key)
			}
		}
		rd.skipLines()
	}
	if isDashed {
		rd.depthDown()
	}
	end := rd.currTok().Pos
	if isList {
		if len(items) > 0 {
			end = items[len(items)-1].Pos().End
		}
		return &ast.List{
			BaseNode: ast.BaseNode{ranges.Range{
				Start: start, End: end,
			}},
			Items: items,
		}
	}
	if len(fields) > 0 {
		end = fields[len(fields)-1].Pos().End
	}
	return &ast.Object{
		BaseNode: ast.BaseNode{ranges.Range{start, end}},
		Fields:   fields,
	}
}

func (rd *reader) parseValueWithFirst(first ast.Value) ast.Value {
	var (
		items     = []ast.Value{first}
		separated []bool
	)
	startLine := first.Pos().Start.Line
	prevEnd := first.Pos().End
	for rd.hasTokens() {
		tok := rd.currTok()
		if tok.Pos.Line != startLine || tok.Kind == Newline || tok.Kind == EOF || tok.Kind == Comma || tok.Kind == RightBracket || tok.Kind == RightCurly {
			break
		}
		separated = append(separated, tok.Pos != prevEnd)
		res := rd.parseValue(rd.currTok())
		items = append(items, res)
		prevEnd = res.Pos().End
	}
	if len(items) == 1 {
		return items[0]
	}
	return &ast.StringGroup{
		BaseNode: ast.BaseNode{Range: ranges.Between(
			items[0].Pos(),
			items[len(items)-1].Pos(),
		)},
		Values:    items,
		Separated: separated,
	}
}

func (rd *reader) parseBracedObject(lc Token) *ast.Object {
	oldFlags := rd.addParseFlags(noComma)
	defer rd.resetParseFlags(oldFlags)
	oldDepth := rd.depth
	rd.depth = 0
	defer func() { rd.depth = oldDepth }()
	rd.advanceTok() // {
	var fields []*ast.Field
	for rd.hasTokens() && rd.currTok().Kind != RightCurly {
		rd.skipLines()
		if rd.currTok().Kind == RightCurly {
			break
		}
		if rd.currTok().Kind == Arrow {
			arrowRef := rd.parseRest(rd.currTok())
			fields = append(fields, &ast.Field{
				BaseNode: ast.BaseNode{Range: arrowRef.Pos()},
				Value:    arrowRef,
			})
		} else {
			key := rd.parseKey()
			rd.expectError(Colon, ErrExpectedToken,
				"Expected ':' after key",
			)
			old := rd.addParseFlags(allowDot)
			val := rd.parseValue(rd.currTok())
			rd.resetParseFlags(old)
			fields = append(fields, &ast.Field{
				BaseNode: ast.BaseNode{Range: ranges.Between(
					key.Pos(), val.Pos(),
				)},
				Key:   key,
				Value: val,
			})
		}
		rd.skipLines()
		if rd.currTok().Kind == Comma {
			rd.advanceTok()
		}
	}
	rb := rd.expectError(RightCurly, ErrUnterminatedObject,
		"Expected '}' to end object",
	)
	return &ast.Object{
		BaseNode: ast.BaseNode{ranges.Range{
			Start: lc.Pos, End: rb.Pos,
		}},
		Fields: fields,
		Inline: true,
	}
}

func (rd *reader) parseBoolean(b Token) *ast.Boolean {
	rd.advanceTok()
	return &ast.Boolean{
		BaseNode: ast.BaseNode{Range: b.Range()},
		Value:    b.Attrs["value"].(bool),
	}
}

func (rd *reader) parseNone(none Token) *ast.None {
	rd.advanceTok()
	return &ast.None{BaseNode: ast.BaseNode{Range: none.Range()}}
}

func (rd *reader) parseClass(at Token) *ast.Class {
	rd.advanceTok() // @
	key := rd.parseKey()
	str, ok := key.(*ast.String)
	if !ok || str.Quote != 0 {
		rd.tokenError(ErrExpectedClassName, at,
			"Class name must be an unquoted identifier",
		)
	}
	name := ""
	if str != nil {
		name = str.Raw
	}
	return &ast.Class{
		BaseNode: ast.BaseNode{ranges.Range{
			Start: at.Pos, End: key.Pos().End,
		}},
		Name: name,
	}
}
