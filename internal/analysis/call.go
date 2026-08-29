package analysis

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (c *Checker) checkCallExpr(expr *ast.CallExpression, t *Expr) {
	lhs := t.NewChild()
	if lhsExpr, ok := expr.Callee.(*ast.Symbol); ok {
		// Allow types for initializers
		c.checkSymbolExpr(lhsExpr, true, lhs)
		if t.Type == nil {
			t.Type = InvalidType
		}
		c.Info.Expressions[expr] = t
	} else {
		c.checkExpr(expr.Callee, lhs)
	}
	if lhs.Kind() == InvalidType {
		t.Type = InvalidType
		return
	}
	canCall := true
	switch fn := UnderlyingTypeName(lhs.Type, true).(type) {
	case *Overload, *Lambda, *TypeName, *EnumFunction:
	case *Function:
		if isTODO(fn) {
			t.Root.mode |= todoExpr
		}
		// TODO: This is temporary and will be removed when generic inference
		// is implemented
		if isCloneBuiltin(fn) && len(expr.Args) >= 1 {
			p1 := c.checkExprFrom(expr.Args[0].Value, t)
			t.Type = p1.Type
			return
		}
	case *UntypedInit:
		if fn.kind == KindEnum && fn.Params == nil {
			enum := expr.Callee.(*ast.EnumLiteral)
			calledInit := &UntypedInit{kind: KindEnum, Node: enum, Params: expr.Args}
			t.Type = calledInit
			c.queue(func() { c.checkEnumParams(expr, t) }, true)
			return // Won't check params now
		}
		canCall = false
	default:
		canCall = false
		if obj, ok := lhs.Type.(*Object); ok && obj.IsTypeName() {
			if _, ok := obj.TypeName().Type.(*TypeAlias); ok {
				// Allow all type aliases to be called for type casts. There isn't
				// always an expression syntax for every type (such as unions), so
				// if the user aliases them, they can cast.
				// 	type MyUnion = String | Int
				// 	union := MyUnion("hello")
				canCall = true
			}
		}
	}
	if !canCall {
		// Not a function (or initializer)
		err := klarerrs.Node(klarerrs.ErrNotAFunction, expr.Callee)
		// If the user tries to call an enum item that doesn't take parameters,
		// show a different error
		if ei, ok := Underlying(lhs.Type).(*EnumRef); ok {
			err.Code = klarerrs.ErrEnumItemNoParams
			err.Label = fmt.Sprintf("Can't pass parameters to %s.%s", ei.Enum.Name, ei.Name)
			err.Name = ei.Name
		} else {
			typ := quoteAka(lhs.Type)
			err.Label = "This callee has type " + typ + " and can't be called"
			err.Name = typ
		}
		c.fileError(err, t.Context.File)
		t.Type = InvalidType
	}
	c.checkCallArgs(lhs.Type, expr.Args, expr.Parens, t)
}

func (c *Checker) checkCallArgs(
	lhs Type, args []*ast.CallParam, parens ranges.Range, t *Expr,
) (overload Type) {
	if c.debug() {
		c.logger.push(fmt.Sprintf(
			"call to %s at %s:%s", lhs, c.Module.ResolveFile(t.FileID()), parens,
		))
		defer c.logger.pop()
	}

	var name string
	if obj, ok := lhs.(*Object); ok {
		name = obj.Name
	}
	switch fn := UnderlyingTypeName(lhs, true).(type) {
	case *TypeName, *Map, *List: // List/map cast
		t.Type = lhs
		// For type casts `T(v)`, `v` must be compatible with `T`.
		//
		// Or, if `v` is an implementation of `T` - `V(t)`, `V?` is returned.
		// `t` must be a union, tag, or interface. If `t` is an optional or Result,
		// show a hint that there's a more idiomatic way to check.
		//
		// If we have `T(t)`, show an error that the cast is redundant.
		und := fn
		if tn, ok := fn.(*TypeName); ok {
			und = tn.Type
		}

		// Builtin initializer
		var isBuiltin bool
		switch und2 := und.(type) {
		case Kind:
			isBuiltin = true
			und = builtinModule.Context.Lookup(und2.String()).TypeName().Type
			if u, ok := und.(*bootstrapType); ok {
				und = u.asDeclared
			}
		case *bootstrapType:
			isBuiltin = true
			und = und2.asDeclared
		}

		var castParam Type
		// Call might be an initializer. Structs, enums, and builtins can be
		// initialized in other ways, not just casts.
		switch und := Underlying(und).(type) {
		case *Struct:
			ps := c.inferCallParams(args, t)
			// Custom initializers have priority over default initializers
			if ok := c.tryCheckInitializer(und.Initializers, ps, args, parens, t); ok {
				// TODO: t.Type may be set to an optional or result
				// t.Type = lhs // Preserve type name given in LHS
				return
			}
			// Now try using a default initializer
			// Users can't access default initializers for builtins, except Error
			if !isBuiltin || name == "Error" {
				if ok := c.checkDefaultStructInit(und, args, ps, t.FileID()); ok {
					t.Type = lhs // Default initializer is never fallable
					return
				}
			}
			castParam = ps.params[0]
		case *Enum:
			ps := c.inferCallParams(args, t)
			if ok := c.tryCheckInitializer(und.Initializers, ps, args, parens, t); ok {
				// TODO: Preserve type name given in LHS
				return
			}
			// TODO: If we support raw-value initializers or flag enums, attempt
			// to check for that.
			if ok := c.checkDefaultEnumInit(und, args, ps, t); ok {
				return
			}
			// TODO: Check arity count like below
			castParam = ps.params[0]
		default:
			if !c.checkArity(args, Arity{1, 1}, len(args), parens, t.FileID()) {
				return
			}
			castParam = c.checkExprFrom(args[0].Value, t).Type
		}
		// Otherwise, it's a type cast
		if !c.checkArity(args, Arity{1, 1}, len(args), parens, t.FileID()) {
			return // There must be exactly 1 parameter
		}
		c.checkTypeCast(lhs, castParam, args[0].Value, t)
	case *EnumFunction:
		t.Type = lhs
	case *Lambda:
		c.checkLambdaParams(fn, args, parens, t)
	case *Function:
		t.Type = fn.Return
		pset := c.inferCallParams(args, t)
		ov, _, err := c.resolveOverload(fn.Overloads, pset, false)
		if err != nil {
			err.Range = parens
			c.fileError(err, t.FileID())
			// ov still isn't nil
		}
		c.checkOverloadParams(ov, pset, args, parens, t)
	case *Overload:
		t.Type = fn.Return
		// c.checkOverloadParams(fn, )
	case *UntypedInit: // Not resolved yet
		t.Type = InvalidType
	case Kind:
	default:
		panic(fmt.Sprintf(
			"checkCallArgs: unhandled LHS type after being handled by checkCallExpr: %T (underlying: %T)",
			lhs, fn,
		))
	}
	return lhs // TODO
}

func (c *Checker) checkArity(
	args []*ast.CallParam, arity Arity, got int, parens ranges.Range, fid FileID,
) bool {
	if arity.InRange(got) {
		return true
	}
	err := arityError(args, arity, got, parens)
	c.fileError(err, fid)
	return false
}

func arityError(args []*ast.CallParam, arity Arity, got int, parens ranges.Range) *klarerrs.Error {
	err := klarerrs.Range(klarerrs.ErrWrongParamCount, parens)
	err.SetParam("got", got)
	expParam := strconv.Itoa(arity.MinParams)
	switch {
	case arity.MaxParams == -1:
		expParam += "+" // Expected 2+ params
	case arity.MinParams != arity.MaxParams:
		expParam += "-" + strconv.Itoa(arity.MaxParams) // Expected 2-4 params
	}
	err.SetParam("expected", expParam)
	if got < arity.MinParams {
		err.SetParam("notEnough", true)
	}
	switch got {
	case 0:
		err.Label = "No parameters were provided"
	case 1:
		err.Label = "1 parameter was provided"
	default:
		err.Label = strconv.Itoa(got) + " parameters were provided"
	}
	return err
}

func (c *Checker) checkLambdaParams(fn *Lambda, args []*ast.CallParam, parens ranges.Range, t *Expr) {
	t.Type = fn.Return
	arity := fn.Arity()
	if !c.checkArity(args, arity, len(args), parens, t.FileID()) {
		return
	}
	for i := 0; i < len(args); i++ {
		var (
			arg        = args[i]
			isVariadic = fn.Variadic && i >= len(fn.Params)
			expType    = fn.Params[min(i, len(fn.Params)-1)]
		)
		if isVariadic {
			expType = expType.(*List).Elem
		}
		// Allowed if tuple or in variadic param
		if rest, ok := arg.Value.(*ast.RestExpression); ok {
			var (
				rhs      = c.checkExprFrom(rest.Expression, t.NewChild())
				rhsRang  = rest.Expression.GetRange()
				itemType Type
			)
			switch rhs.Kind() {
			case KindTuple:
				tup := As[*Tuple](rhs.Type)
				switch {
				case tup.Len() < 2:
					c.fileError(makeTupleSpreadCountError(rest, tup.Len(), false), t.FileID())
				case isVariadic:
					commonType, err := canSpreadTupleIntoList(tup)
					if err != nil {
						err.Range = rhsRang
						c.fileError(err, t.FileID())
						break
					}
					if !Compatible(commonType, expType) {
						c.fileError(typeMismatch(expType, commonType, rhsRang), t.FileID())
					}
				case arity.MaxParams != -1 && i+tup.Len() > arity.MaxParams:
				// Tuple has more parameters than the function accepts
				// TODO: Error
				default:
					for j, item := range tup.Items {
						expType := fn.Params[min(i+j, len(fn.Params)-1)]
						if !Compatible(item, expType) {
							c.fileError(typeMismatch(expType, item, rhsRang), t.FileID())
						}
					}
					i += tup.Len()
				}
			case KindList:
				itemType = As[*List](rhs.Type).Elem
				fallthrough
			case StringType:
				if !isVariadic {
					c.fileError(misplacedListRestError(rhs.Kind(), rest), t.FileID())
				}
				itemType = cmp.Or[Type](itemType, StringType)
				if !Compatible(itemType, expType) {
					c.fileError(typeMismatch(expType, itemType, rhsRang), t.FileID())
				}
			default:
				c.fileError(invalidRestTypeError(rhs.Type, rest.Expression), t.FileID())
			}
			continue
		}
		e := c.checkExpr(arg.Value, t.NewChild().withHint(expType))
		if !Compatible(e.Type, expType) {
			err := typeMismatch(expType, e.Type, arg.Range)
			// If [T] is passed to a parameter ...T, show a hint
			// TODO: Probably also show a hint for tuples and strings
			if list, ok := Underlying(e.Type).(*List); ok &&
				isVariadic && Compatible(list.Elem, expType) {
				hintWithDiff(
					err, "Did you mean to spread this list?",
					klarerrs.AddedString{Pos: arg.Range.End, String: "..."},
				)
			}
			c.fileError(err, t.FileID())
		}
	}
}

func (c *Checker) checkEnumParams(expr *ast.CallExpression, t *Expr) {
}

func (c *Checker) checkTypeCast(
	target, param Type, arg ast.Expression, t *Expr) {
}

// Returned parameters are untyped
func (c *Checker) inferCallParams(params []*ast.CallParam, callExpr *Expr) (ps paramSet) {
	for _, arg := range params {
		var typ Type
		if rest, ok := arg.Value.(*ast.RestExpression); ok {
			// TODO: If the param is labelled, a rest is only allowed if the
			// param is variadic
			inner := c.checkExprFrom(rest.Expression, callExpr)
			switch inner.Kind() {
			case KindTuple:
				tup := As[*Tuple](inner.Type)
				ps.params = append(ps.params, tup.Items...)
				for range tup.Items {
					ps.nodeMap = append(ps.nodeMap, rest.Expression)
				}
				continue
			case KindList:
				typ = As[*List](inner.Type).Elem
			case StringType:
				typ = StringType
			default:
				c.fileError(invalidRestTypeError(inner.Type, rest.Expression), callExpr.FileID())
				typ = inner.Type
			}
		} else {
			typ = c.checkExprFrom(arg.Value, callExpr).Type
		}
		// TODO: Labelled param followed by unlabelled params is considered variadic
		if arg.Label != nil {
			if ps.labelled == nil {
				ps.labelled = make(map[string]Type)
			}
			ps.labelled[arg.Label.Name] = typ
		} else {
			ps.params = append(ps.params, typ)
			ps.nodeMap = append(ps.nodeMap, arg.Value)
		}
	}
	return ps
}

func (c *Checker) checkOverloadParams(
	o *Overload, ps paramSet, params []*ast.CallParam, parens ranges.Range, t *Expr,
) {
	t.Type = o.Return
	// Arity check. We're not directly calling [Checker.checkArity] so we can
	// offer hints if positional params were used instead of labelled ones.
	if !o.Arity.InRange(len(ps.params)) {
		c.overloadArityError(o, ps, params, parens, t)
		return
	}

	checkParam := func(expVar *Variable, got Type, getNode func() ast.Expression) {
		exp := expVar.Type
		isVariadic := expVar.Object.Flags.Has(VariadicParam)
		if isVariadic {
			exp = exp.(*List).Elem
		}
		if !Compatible(got, exp) {
			err := typeMismatch(exp, got, getNode().GetRange())
			c.fileError(err, t.FileID())
		}
		// Set the inferred types for lambdas and shorthands
		// TODO: This isn't a perfect solution. If these types are a list element,
		// they won't be set.
		switch Underlying(got).(type) {
		case *UntypedLambda, *UntypedInit:
			if e := c.Info.Expressions[getNode()]; e != nil {
				e.Type = exp
			}
		}
	}

	// Positional params
	for i, got := range ps.params {
		expVar := o.Params[min(i, len(o.Params)-1)]
		checkParam(expVar, got, func() ast.Expression { return ps.nodeMap[i] })
	}

	// Labelled params
	orderedLabels := orderedLabelledParams(params)
	var labelledNodes map[string]*ast.CallParam // Lazily initialized
	getLabelledNodes := func() map[string]*ast.CallParam {
		if labelledNodes == nil {
			labelledNodes = makeLabelledParamMap(params)
		}
		return labelledNodes
	}
	for _, name := range orderedLabels {
		got := ps.labelled[name]
		expVar, ok := o.labelMap[name]
		if !ok {
			// Labelled param doesn't exist
			labelNode := getLabelledNodes()[name].Label
			err := klarerrs.Node(klarerrs.ErrParamLabelUndefined, labelNode)
			err.Desc = "These params are defined: " + strings.Join(
				slices.Sorted(maps.Keys(o.labelMap)), ", ",
			)
			c.fileError(err, t.FileID())
			continue
		}
		checkParam(expVar, got, func() ast.Expression {
			return getLabelledNodes()[name].Value
		})
	}
}

func orderedLabelledParams(nodes []*ast.CallParam) (labels []string) {
	for _, node := range nodes {
		if node.Label != nil {
			labels = append(labels, node.Label.Name)
		}
	}
	return
}

func makeLabelledParamMap(params []*ast.CallParam) map[string]*ast.CallParam {
	m := make(map[string]*ast.CallParam)
	for _, node := range params {
		if node.Label != nil {
			// Respecified labels already checked at parse-time
			m[node.Label.Name] = node
		}
	}
	return m
}

// Try to show a helpful message if the user provided a labelled
// parameter as positional
func (c *Checker) overloadArityError(
	o *Overload, ps paramSet, params []*ast.CallParam, parens ranges.Range, t *Expr,
) {
	defaultError := func() {
		// If there's no special hint we can give, use the generic message
		c.checkArity(params, o.Arity, len(ps.params), parens, t.FileID())
	}
	// To show a different message, the user has to provide all required
	// positional params (may be 0)
	if len(ps.params) <= len(o.Params) {
		defaultError()
		return
	}
	var hadAltError bool
	for i, posParam := range ps.params[len(o.Params):] {
		// Labelled params must be in order to show the special message
		if i >= len(o.LabelledParams) {
			break
		}
		labelled := o.LabelledParams[i]
		if !Compatible(posParam, labelled.Variable.Type) {
			continue
		}
		hadAltError = true
		node := ps.nodeMap[len(o.Params)+i]
		err := klarerrs.Node(klarerrs.ErrMissingParamLabel, node)
		err.Name = labelled.Label
		err.Label = fmt.Sprintf("Missing parameter label '%s:'", labelled.Label)
		hintWithDiff(err, "Add the missing label", klarerrs.AddedString{
			Pos: node.GetRange().Start, String: labelled.Label + ": ",
		})
		c.fileError(err, t.FileID())
	}
	if !hadAltError {
		defaultError()
	}
}
