package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/target"
)

func (c *Checker) checkFunctionImpls(ov *Overload, stmt *ast.FunctionDeclaration) {
	if !c.Options.EnforceTargetSupport {
		return
	}
	isExternal := ov.attrs != nil && len(ov.attrs.External) > 0
	hasBody := stmt.Body != nil || stmt.Expression != nil
	switch {
	case !isExternal && !hasBody:
		err := klarerrs.Node(klarerrs.ErrFuncNoBody, stmt)
		err.Name = ov.Name
		err.Label = "This function has no body/implementation"
		c.fileError(err, ov.File)
	case !c.Options.EnforceTargetSupport:
		return
	case hasBody:
		// If isExternal, the function has a fallback implementation.
		// We won't report a warning if the fallback is never reachable (the
		// function always has an external impl) because people may want
		// to futureproof their code for future targets.
		// Otherwise, this is a normal function
	default:
		// External with no body. Check that the function has an implementation
		// on all possible targets
		definedImpls := make(map[target.Target]struct{})
		for _, ext := range ov.attrs.External {
			_ = ext
		}
		_ = definedImpls
	}
}

// See https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Lexical_grammar
var jsKeywords = map[string]struct{}{
	"arguments": {}, "async": {}, "await": {}, "break": {}, "case": {}, "catch": {},
	"class": {}, "const": {}, "continue": {}, "debugger": {}, "default": {}, "delete": {},
	"do": {}, "else": {}, "enum": {}, "eval": {}, "export": {}, "extends": {},
	"false": {}, "finally": {}, "for": {}, "function": {}, "if": {}, "implements": {},
	"import": {}, "in": {}, "instanceof": {}, "interface": {}, "let": {}, "new": {},
	"null": {}, "package": {}, "private": {}, "protected": {}, "public": {}, "return": {},
	"static": {}, "super": {}, "switch": {}, "this": {}, "throw": {}, "true": {},
	"try": {}, "typeof": {}, "undefined": {}, "var": {}, "void": {}, "while": {},
	"with": {}, "yield": {},
	// Added myself
	"using": {},
}

// Public declarations can't be named after JavaScript keywords when compiling
// to the JS target. The `@name(js:)` attribute can be used to change the name
// of the object for JS.
func (c *Checker) validateJSNames(ctx *Context) {
	for _, obj := range ctx.SortedDecls() {
		// @name attribute already checked
		if !obj.Public || (obj.attrs != nil && obj.attrs.Name[target.JavaScript] != "") {
			continue
		}
		if _, ok := jsKeywords[obj.Name]; ok {
			err := klarerrs.Range(klarerrs.ErrReservedJSKeyword, obj.Range)
			err.Name = obj.Name
			err.Label = quote(obj.Name) + " is reserved in JavaScript"
			hintWithDiff(
				err,
				"Choose a different name, or apply the '@target(js:)' attribute to override the name used in the compiled JavaScript.",
			)
			c.fileError(err, obj.File)
		}
	}
}
