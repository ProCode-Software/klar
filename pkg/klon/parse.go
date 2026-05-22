package klon

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

const MaxDepth = 10000

// parseDocument parses a full KLON document.
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

// parseValue parses a single value, which could be a primitive, list, object,
// or a StringGroup if multiple values are on the same line.
func (rd *reader) parseValue(tok Token) ast.Value {
	// Skip leading newlines
	hasNewline := tok.Kind == Newline
	if hasNewline {
		rd.skipLines()
		tok = rd.currTok()
	}

	// A. Top-level object
	if rd.depth == 0 {
		if rd.peekTok().Kind == Colon || tok.Kind == Dash /* Invalid but still parse it */ {
			return rd.parseObject()
		}
	}
	if tok.Kind == Dash && hasNewline {
		return rd.parseObject()
	}

	// B. EOF
	if tok.Kind == EOF {
		return &ast.None{BaseNode: ast.BaseNode{Range: tok.Range()}}
	}

	// C. Read 1+ items in a group
	var items []ast.Value
	startLine := tok.Pos.Line
loop:
	for rd.hasTokens() && rd.currTok().Pos.Line == startLine {
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
		case AtRef:
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
			res = rd.parseInlineList(tok)
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
	}

	switch len(items) {
	case 0:
		return &ast.None{BaseNode: ast.BaseNode{Range: tok.Range()}}
	case 1:
		return items[0]
	default:
		// More than 1 item should be in a StringGroup
		return &ast.StringGroup{
			BaseNode: ast.BaseNode{Range: sliceRange(items)},
			Values:   items,
		}
	}
}

// parseNumber parses a numeric literal.
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

// parseRest parses a spread operator reference (<- $var).
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

// parseVariable parses a variable reference ($name or ${name}).
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
			rd.tokenError(err, vr, "Expected closing '}' in variable reference")
		} else {
			name = vr.Src[2 : len(vr.Src)-1]
		}
	} else {
		name = vr.Src[1:]
	}
	if err == ErrInvalidIdentifier {
		rd.tokenError(err, vr, "A variable name can't start with a digit")
	}
	return &ast.VarRef{
		BaseNode: ast.BaseNode{Range: vr.Range()},
		Name:     name,
		Braces:   braces,
	}
}

// parseInlineList parses an inline list literal [...].
func (rd *reader) parseInlineList(lb Token) *ast.List {
	oldFlags := rd.addParseFlags(noComma | allowDot)
	defer rd.resetParseFlags(oldFlags)

	rd.depthUp()
	defer rd.depthDown()
	rd.advanceTok() // [

	var items []ast.Value
	for rd.hasTokens() && rd.currTok().Kind != RightBracket {
		items = append(items, rd.parseValue(rd.advanceTok()))
		rd.skipLines()
		if rd.currTok().Kind != RightBracket {
			rd.expectError(Comma, ErrExpectedToken,
				"Expected ',' between list items or ']' to end the list",
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

// parseString parses a quoted string literal.
func (rd *reader) parseString(str Token) *ast.String {
	rd.advanceTok()
	fmt.Println(str)
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
		src = str.Src[1:]
		if wrap {
			src = str.Src[2:]
		}
		if str.Attrs["unterm"].(bool) {
			rd.tokenError(ErrUnterminatedString, str, "This string was left open")
		} else {
			src = src[:len(src)-1]
		}
	}
	// Read segments
	// TODO: move segment parsing to lexer
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

// getEscape returns the unescaped byte for a given escape character.
func getEscape(c byte) (byte, bool) {
	escapes := map[byte]byte{
		'f': '\f', 'n': '\n', 'r': '\r', 't': '\t', 'v': '\v', 'e': '\x1b',
		'\\': '\\', '"': '"', '\'': '\'', '$': '$',
	}
	esc, ok := escapes[c]
	return esc, ok
}

// parseBoolean parses a boolean literal.
func (rd *reader) parseBoolean(b Token) *ast.Boolean {
	rd.advanceTok()
	return &ast.Boolean{
		BaseNode: ast.BaseNode{Range: b.Range()},
		Value:    b.Attrs["value"].(bool),
	}
}

// parseNone parses a 'none' literal.
func (rd *reader) parseNone(none Token) *ast.None {
	rd.advanceTok()
	return &ast.None{BaseNode: ast.BaseNode{Range: none.Range()}}
}

// parseClass parses a class name (@identifier).
func (rd *reader) parseClass(cls Token) *ast.Class {
	rd.advanceTok() // @
	if cls.Attrs != nil && cls.Attrs["invalid"] == true {
		rd.tokenError(ErrInvalidIdentifier, cls, "A class name can't start with a digit")
	}
	return &ast.Class{
		BaseNode: ast.BaseNode{cls.Range()},
		Name:     cls.Src[1:],
	}
}

// parseList parses a line-separated list of values. The first value was
// already parsed by [reader.parseEntry].
func (rd *reader) parseList(first ast.Value) *ast.List {
	items := []ast.Value{first}

	old := rd.addParseFlags(objectValue | allowDot)
	defer rd.resetParseFlags(old)

	for rd.hasTokens() {
		if !rd.checkDashes(rd.parseDashes()) {
			break
		}
		items = append(items, rd.parseValue(rd.readToken()))
	}
	return &ast.List{
		BaseNode: ast.BaseNode{Range: sliceRange(items)},
		Items:    items,
	}
}

// parseObject parses an unbraced object or dashed list.
// It determines the type of the collection (List or Object) based on the first entry.
func (rd *reader) parseObject() ast.Value {
	var (
		fields   []*ast.Field
		start    = rd.currTok().Pos
		isObject bool
	)
	for rd.hasTokens() {
		item, dashes := rd.parseEntry(isObject)
		if dashes < rd.depth {
			break
			// TODO: return
		}
		var field *ast.Field
		switch item := item.(type) {
		case *ast.Field:
			field = item
		case *ast.ArrowRef:
			field = &ast.Field{Arrow: item}
		default:
			return rd.parseList(item) // It's a list
		}
		isObject = true
		fields = append(fields, field)
	}

	if len(fields) == 0 {
		// It didn't successfully read a field, due to EOF
		return &ast.None{}
	}

	end := fields[len(fields)-1].Range.End
	return &ast.Object{
		BaseNode: ast.BaseNode{ranges.Range{start, end}},
		Fields:   fields,
		Inline:   false,
	}
}

// parseBracedObject parses a braced object { ... }.
func (rd *reader) parseBracedObject(lc Token) ast.Value {
	old := rd.addParseFlags(noComma)
	defer rd.resetParseFlags(old)

	// Dash depth resets in braced objects
	oldDepth := rd.depth
	rd.depth = 0
	defer func() { rd.depth = oldDepth }()

	rd.advanceTok() // {

	var fields []*ast.Field
	for rd.hasTokens() && rd.currTok().Kind != RightCurly {
		field, dashes := rd.parseEntry(true)
		if dashes < rd.depth {
			break
		}
		fields = append(fields, field.(*ast.Field))

		if rd.currTok().Kind != RightCurly {
			// TODO: also allow newline as a separator
			rd.expectError(Comma, ErrExpectedToken,
				"Expected ',' to separate inline object fields",
			)
		}
	}
	rc := rd.expectError(RightCurly, ErrUnterminatedObject,
		"Expected '}' to close inline object",
	)
	return &ast.Object{
		BaseNode: ast.BaseNode{ranges.Range{lc.Pos, rc.End()}},
		Fields:   fields,
		Inline:   true,
	}
}

// parseEntry parses a single entry within an object or list block.
// The entry can be keyed or unkeyed, or a rest. Variable declarations
// are handled. If the entry is a rest, an [*ast.ArrowRef] is returned.
// If isObject == true, an error is reported if the entry doesn't have a key.
func (rd *reader) parseEntry(forceObject bool) (entry ast.Value, dashes int) {
	var (
		varName   *ast.VarRef
		singleKey ast.Value
		path      *[]ast.Value
		keyStart  lexer.Position

		value ast.Value
	)
	// Check dashes
	dashes = rd.parseDashes()
	if !rd.checkDashes(dashes) {
		return nil, dashes
	}

	switch tok := rd.currTok(); tok.Kind {
	case Arrow:
		// Rest
		return rd.parseRest(tok), dashes
	case Variable:
		// Variable declaration
		varName = rd.parseVariable(tok)
		singleKey = varName
	default:
		// Normal key or list item
		singleKey, path, keyStart = rd.parseKey()
	}

	// Key-value field
	if rd.currTok().Kind == Colon {
		rd.advanceTok() // :
		old := rd.addParseFlags(allowDot | objectValue)
		value = rd.parseValue(rd.currTok())
		rd.resetParseFlags(old)

		// If the variable is a field, then it's a declaration
		if varName != nil {
			rd.declareVariable(varName, value)
			return rd.parseEntry(forceObject) // Read another entry
		}

		return &ast.Field{
			BaseNode: ast.BaseNode{ranges.Range{keyStart, value.Pos().End}},
			Key:      singleKey,
			KeyPath:  path,
			Value:    value,
		}, dashes
	}

	// Unkeyed list item
	// =====

	// If we read a key-path, it has to be converted to a single value
	if path != nil {
		singleKey = rd.convertKeyPath(path)
	}
	// If forceObject == true, report an error because this should be a key-value pair
	if forceObject {
		rd.rangeError(ErrExpectedKeyValue, singleKey.Pos(),
			"Expected a key-value pair in this object",
		)
		singleKey = &ast.Field{Key: &ast.Bad{Value: singleKey}}
	}
	return singleKey, dashes
}

func (rd *reader) declareVariable(name *ast.VarRef, value ast.Value) {
	if rd.depth != 0 {
		rd.rangeError(ErrVarNotTopLevel, name.Range, "Variables must be declared at the top level")
	}
	if name.Braces {
		rd.rangeError(ErrInvalidVarDecl, name.Range, "Variable declarations can't use braces")
	}
	if rd.vars == nil {
		rd.vars = make(map[string]ast.Value)
	}
	rd.vars[name.Name] = value
}

func (rd *reader) convertKeyPath(path *[]ast.Value) ast.Value {
	return nil
}

// parseKey parses a key for a field. The key can be either a single value,
// or a dot-path.
func (rd *reader) parseKey() (singleKey ast.Value, dotPath *[]ast.Value, start lexer.Position) {
	validate := func(v ast.Value) bool {
		switch v.(type) {
		case *ast.String, *ast.Number, *ast.Bad, *ast.Boolean:
			return true
		default:
			rd.rangeError(ErrInvalidKey, v.Pos(), "A field key must be a string, number, or boolean")
			return false
		}
	}

	// Single key
	singleKey = rd.parseValue(rd.advanceTok())
	start = singleKey.Pos().Start
	if !validate(singleKey) {
		singleKey = &ast.Bad{Value: singleKey}
	}
	if rd.currTok().Kind != Dot {
		return singleKey, nil, start
	}

	// Dot-separated key path
	dotPath = &[]ast.Value{singleKey}
	for rd.currTok().Kind == Dot {
		rd.advanceTok() // .
		singleKey = rd.parseValue(rd.advanceTok())
		if !validate(singleKey) {
			singleKey = &ast.Bad{Value: singleKey}
		}
		*dotPath = append(*dotPath, singleKey)
	}
	return nil, dotPath, start
}

func (rd *reader) parseDashes() (n int) {
	rd.skipLines()
	for rd.hasTokens() && rd.currTok().Kind == Dash {
		n++
		rd.advanceTok()
	}
	return n
}

func (rd *reader) checkDashes(n int) bool {
	if n < rd.depth {
		return false
	}
	if n > rd.depth {
		// Error
		return false
	}
	return true
}
