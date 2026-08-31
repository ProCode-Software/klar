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
