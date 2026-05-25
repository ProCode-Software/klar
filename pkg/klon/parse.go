package klon

import (
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

const MaxDepth = 10000

// parseDocument parses a full Klon document.
func (rd *reader) parseDocument() (*ast.Document, []error) {
	res := rd.parseValue()

	// Check for EOF
	rd.skipLines()
	if tok := rd.currTok(); tok.Kind != EOF {
		rd.tokenError(klonerrs.ErrExpectedEOF, tok, "Expected end of file")
	}

	return &ast.Document{
		BaseNode:  ast.BaseNode{ranges.Range{res.Pos().Start, rd.offset}},
		Body:      res,
		Variables: rd.vars,
		Comments:  rd.comments,
	}, rd.errs
}

// parseValue parses one or more values on a single line. parseValue may also
// parse an object or list if a dash is encountered on the next line.
func (rd *reader) parseValue() ast.Value {
	tok := rd.currTok()

	// Skip leading newlines
	if tok.Kind == Newline {
		old := rd.removeParseFlags(objectValue)
		rd.skipLines()
		tok = rd.currTok()
		// `value:\n - x` and `\n x` technically start with the same depth
		isTopLevel := rd.depth == 0 && old&objectValue == 0
		rd.resetParseFlags(old)

		// Object as a value
		if tok.Kind == Dash {
			if !isTopLevel {
				rd.depthUp()
				defer rd.depthDown()
			}
			return rd.parseObject()
		}
	}

	// A. Top-level object
	if rd.depth == 0 {
		if (rd.parseFlags&key == 0 && rd.peekTok().Kind == Colon) ||
			tok.Kind == Dash /* Only valid as a list */ {
			return rd.parseObject()
		}
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
		switch tok = rd.currTok(); tok.Kind {
		case EOF, Newline:
			break loop
		case Comma:
			if len(items) == 0 {
				rd.tokenError(klonerrs.ErrExpectedValue, tok, "Expected a value before comma")
				rd.advanceTok()
			}
			break loop
		case RightBracket, RightCurly:
			if len(items) == 0 {
				rd.tokenError(klonerrs.ErrUnmatchedBracket, tok, "Unmatched '%s'", tok.Src)
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
		case LeftBracket:
			res = rd.parseInlineList(tok)
		case LeftCurly:
			res = rd.parseBracedObject(tok)

		case Dash:
			// Invalid dash
			if rd.depth == 0 {
				rd.tokenError(klonerrs.ErrDashAtTopLevel, tok,
					"Top-level objects and lists can't start with a dash",
				)
				// Still parse the object
				rd.depthUp()
				res = rd.parseObject()
				rd.depthDown()
				res = &ast.Bad{BaseNode: ast.BaseNode{res.Pos()}, Value: res}
			} else {
				rd.tokenError(klonerrs.ErrDashWithoutNewline, tok,
					"An object or list must begin on a new line",
				)
				rd.advanceTok()
				continue
			}

		case Colon:
			if rd.parseFlags&key != 0 {
				break loop
			}
			fallthrough
		default:
			rd.tokenError(klonerrs.ErrUnexpectedToken, tok, "Unexpected token")
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
		rd.tokenError(klonerrs.ErrExpectedVarInArrow, arrow, "Expected a variable after '<-'")
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
		err    = vr.Attrs["err"].(klonerrs.Code)
		braces = vr.Attrs["brace"].(bool)
	)
	if braces {
		if err == klonerrs.ErrUnterminatedVar {
			name = vr.Src[2:]
			rd.tokenError(err, vr, "Expected closing '}' in variable reference")
		} else {
			name = vr.Src[2 : len(vr.Src)-1]
		}
	} else {
		name = vr.Src[1:]
	}
	if err == klonerrs.ErrInvalidIdentifier {
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
		items = append(items, rd.parseValue())
		rd.skipLines()
		if rd.currTok().Kind != RightBracket {
			rd.expect(Comma, klonerrs.ErrExpectedToken,
				"Expected ',' between list items or ']' to end the list",
			)
		}
	}

	rd.skipLines()
	rb := rd.expect(RightBracket, klonerrs.ErrUnterminatedList, "Expected ']' to end list")
	return &ast.List{
		BaseNode: ast.BaseNode{ranges.Range{lb.Pos, rb.Pos}},
		Inline:   true,
		Items:    items,
	}
}

// parseString parses a quoted string literal.
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
		src = str.Src[1:]
		if wrap {
			src = str.Src[2:]
		}
		if str.Attrs["unterm"].(bool) {
			rd.tokenError(klonerrs.ErrUnterminatedString, str, "This string was left open")
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
			rd.tokenError(klonerrs.ErrUnknownEscape, str, `Unknown escape sequence '\%c'`, c)
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
	return &ast.None{BaseNode: ast.BaseNode{Range: none.Range()}, Literal: true}
}

// parseClass parses a class name (@identifier).
func (rd *reader) parseClass(cls Token) *ast.Class {
	rd.advanceTok() // @
	if cls.Attrs != nil && cls.Attrs["invalid"] == true {
		rd.tokenError(klonerrs.ErrInvalidIdentifier, cls, "A class name can't start with a digit")
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

	old := rd.addParseFlags(allowDot)
	defer rd.resetParseFlags(old)
	// objectValue was already removed by parseObject. It will be restored by its defer call.

	for rd.hasTokens() {
		if !rd.checkDashes(rd.parseDashes()) {
			break
		}
		// TODO: will an item that starts with a newline and indents be parsed?
		old := rd.removeParseFlags(objectValue)
		items = append(items, rd.parseValue())
		rd.resetParseFlags(old)
	}
	return &ast.List{
		BaseNode: ast.BaseNode{Range: sliceRange(items)},
		Items:    items,
	}
}

// parseObject parses an unbraced object or dashed list.
// It determines the type of the collection (List or Object) based on the first entry.
func (rd *reader) parseObject() ast.Value {
	old := rd.removeParseFlags(objectValue)
	defer rd.addParseFlags(old)

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
		var needsNl bool
		switch item := item.(type) {
		case nil:
			goto next // Variable declaration
		case *ast.Field:
			field = item
			// If the value was an object, the newline was consumed before dedenting,
			// so we don't need to check for a newline.
			needsNl = needsNewlineAfter(item.Value)
		case *ast.ArrowRef:
			field = &ast.Field{Arrow: item}
		default:
			return rd.parseList(item) // It's a list
		}
		isObject = true
		fields = append(fields, field)
	next:
		if rd.currTok().Kind != EOF && needsNl {
			rd.expect(Newline, klonerrs.ErrExpectedToken, "Expected a newline between fields")
		}
	}

	if len(fields) == 0 {
		// It didn't successfully read a field, due to variable declaration
		return &ast.None{}
	}

	end := fields[len(fields)-1].Range.End
	return &ast.Object{
		BaseNode: ast.BaseNode{ranges.Range{start, end}},
		Fields:   fields,
		Inline:   false,
	}
}

func needsNewlineAfter(item ast.Value) bool {
	switch item := item.(type) {
	case *ast.Object:
		return item.Inline
	case *ast.List:
		return item.Inline
	}
	return true
}

// parseBracedObject parses a braced object { ... }.
func (rd *reader) parseBracedObject(lc Token) ast.Value {
	rd.depthUp()
	defer rd.depthDown()

	old := rd.addParseFlags(noComma)
	defer rd.resetParseFlags(old)

	old2 := rd.removeParseFlags(objectValue)
	defer rd.addParseFlags(old2)

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

		if curr := rd.currTok(); curr.Kind != RightCurly {
			if curr.Kind != Comma && curr.Kind != Newline {
				rd.tokenError(klonerrs.ErrExpectedToken, curr,
					"Expected ',' or newline to separate inline object fields",
				)
				continue
			}
			rd.advanceTok()
		}
	}
	rc := rd.expect(RightCurly, klonerrs.ErrUnterminatedObject,
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
// are handled and return a nil entry when encountered. If the entry is a
// rest, an [*ast.ArrowRef] is returned. If isObject == true, an error is
// reported if the entry doesn't have a key.
func (rd *reader) parseEntry(forceObject bool) (entry ast.Value, dashes int) {
	var (
		varName   *ast.VarRef
		singleKey ast.Value
		path      *[]ast.Value
		keyStart  lexer.Position
		value     ast.Value
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
		value = rd.parseValue()
		rd.resetParseFlags(old)

		// If the variable is a field, then it's a declaration
		if varName != nil {
			rd.declareVariable(varName, value)
			return nil, dashes
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
		rd.rangeError(klonerrs.ErrExpectedKeyValue, singleKey.Pos(),
			"Expected a key-value pair in this object",
		)
		singleKey = &ast.Field{Key: &ast.Bad{Value: singleKey}}
	}
	return singleKey, dashes
}

func (rd *reader) declareVariable(name *ast.VarRef, value ast.Value) {
	switch {
	case rd.flags.Has(klonflags.NoVariables):
		rd.rangeError(klonerrs.ErrVarsDisabled, name.Range,
			"Variables aren't allowed to be declared in this file",
		)
	case rd.depth != 0:
		rd.rangeError(klonerrs.ErrVarNotTopLevel, name.Range,
			"Variables must be declared at the top level",
		)
	case name.Braces:
		rd.rangeError(klonerrs.ErrInvalidVarDecl, name.Range,
			"Variable declarations can't use braces",
		)
	case rd.vars != nil && rd.vars[name.Name] != nil:
		existing := rd.vars[name.Name]
		rd.rangeError(klonerrs.ErrVarAlreadyDeclared, name.Range,
			"Variable '%s' was already declared at %s", name.Name, existing.Pos(),
		)
	default:
		if rd.vars == nil {
			rd.vars = make(map[string]ast.Value)
		}
		rd.vars[name.Name] = value
	}
}

// convertKeyPath converts a dot-path to a StringGroup value.
func (rd *reader) convertKeyPath(path *[]ast.Value) ast.Value {
	return &ast.StringGroup{
		BaseNode: ast.BaseNode{sliceRange(*path)},
		Values:   *path,
	}
}

// parseKey parses a key for a field. The key can be either a single value,
// or a dot-path.
func (rd *reader) parseKey() (singleKey ast.Value, dotPath *[]ast.Value,
	start lexer.Position,
) {
	validate := func(v ast.Value) bool {
		switch v.(type) {
		case *ast.String, *ast.Number, *ast.Bad, *ast.Boolean:
			return true
		default:
			rd.rangeError(klonerrs.ErrInvalidKey, v.Pos(),
				"A field key must be a string, number, or boolean",
			)
			return false
		}
	}
	old := rd.addParseFlags(key)
	defer rd.resetParseFlags(old)

	// Single key
	singleKey = rd.parseValue()
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
		singleKey = rd.parseValue()
		if !validate(singleKey) {
			singleKey = &ast.Bad{Value: singleKey}
		}
		*dotPath = append(*dotPath, singleKey)
	}
	return nil, dotPath, start
}

// parseDashes parses consecutive dashes on a line, returning the count.
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
		return false // Dedent
	}
	if n > rd.depth+1 {
		// Too many dashes
		if rd.depth == 0 {
			rd.tokenError(klonerrs.ErrDashSkip, rd.currTok(),
				"The top level object shouldn't include a dash",
			)
		} else {
			rd.tokenError(klonerrs.ErrDashSkip, rd.currTok(),
				"Too many dashes: expected up to %d, there are %d", rd.depth+1, n,
			)
		}
		return true // For recovery
	}
	return true
}
