package analysis

import (
	"fmt"
	"iter"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// followDestructure follows a destructure expression, yielding each
// [ast.Assignable] on the left (from dest) and its corresponding [Type]
// from the right (from e). Errors are reported for type mismatches,
// and for [ast.BadExpression] nodes if nameOnly == false.
//
// All [ast.BadExpression] nodes are skipped (if nameOnly == true,
// errors would have already been reported by [Checker.declareVars]). If
// nameOnly == true, all yielded nodes are [*ast.Symbol].
func (c *Checker) followDestructure(
	lhs ast.Assignable, rhs Type, fid FileID, r ranges.Range, nameOnly bool,
) iter.Seq2[ast.Assignable, Type] {
	dc := &destructureContext{fid: fid, rhsRange: r, nameOnly: nameOnly}
	return func(yield func(ast.Assignable, Type) bool) {
		dc.walk = func(dest ast.Expression, typ Type) bool {
			// Ensure the expression can actually be in a destructure pattern.
			// For a top-level destructure `[a, b] = ...`, this will always be the
			// case, but in lists and tuples, it isn't checked at parse-time.
			assg, ok := dest.(ast.Assignable)
			if !ok {
				if !nameOnly {
					err := klarerrs.Node(klarerrs.ErrInvalidAssignment, dest)
					err.Label = "Can't assign to this expression"
					c.fileError(err, dc.fid)
				}
				return true
			}
			// We don't want walk{List,Tuple}Destructure to report an error if the
			// type isn't a list/tuple.
			if typ.Kind() == InvalidType {
				return yield(assg, typ)
			}
			switch dest := assg.(type) {
			case *ast.Symbol:
				return yield(dest, typ)
			case *ast.ListLiteral:
				return c.walkListDestructure(dest, typ, dc)
			case *ast.TupleLiteral:
				return c.walkTupleDestructure(dest, typ, dc)
			case *ast.Discard:
				return true
			case *ast.BadExpression:
				return true // Syntax error already reported
			case ast.Destructurable:
				panic(fmt.Sprintf(
					"followDestructure: unhandled ast.Destructurable pattern type: %T",
					dest,
				))
			default:
				if nameOnly {
					return true // Error already reported by [Checker.declareVars]
				}
				return yield(dest, typ)
			}
		}
		dc.walk(lhs, rhs)
	}
}

type destructureContext struct {
	fid      FileID
	walk     func(ast.Expression, Type) bool
	rhsRange ranges.Range
	nameOnly bool // Reports syntax errors if false
}

// TODO: In List and Tuple destructure, ensure there is only 1 rest. Also properly
// check for the next destructure patterns after the rest.

func (c *Checker) walkListDestructure(dest *ast.ListLiteral,
	t Type, dc *destructureContext,
) bool {
	if t.Kind() != KindList {
		err := klarerrs.TypeError(
			klarerrs.ErrTypeMismatch, dc.rhsRange, KindList.String(), t.String(),
		)
		err.AddHighlight("This pattern destructures a list", dest.Range)
		c.fileError(err, dc.fid)
		return true
	}
	list := Underlying(t).(*List)
	elem := list.Elem
	for _, item := range dest.Items {
		// Rest destructure
		// 	[a, b, rest...] := [1, 2, 3, 4, 5]
		// 	[a, b, obj.items...] = [1, 2, 3, 4, 5]
		if rest, ok := item.(*ast.RestExpression); ok {
			// The rest item has type [T]
			if !dc.walk(rest.Expression, &List{Elem: elem}) {
				return false
			}
		} else
		// Normal destructure
		// 	[a, b, c] := [1, 2, 3]
		// 	[first, (key, value)] := [('a', 1), ('b', 2)]
		if !dc.walk(item, elem) {
			return false
		}
	}
	return true
}

func (c *Checker) walkTupleDestructure(dest *ast.TupleLiteral,
	t Type, dc *destructureContext,
) bool {
	if t.Kind() != KindTuple {
		err := klarerrs.TypeError(
			klarerrs.ErrTypeMismatch, dc.rhsRange, KindTuple.String(), t.String(),
		)
		err.AddHighlight("This pattern destructures a tuple", dest.Range)
		c.fileError(err, dc.fid)
		return true
	}
	rhs := Underlying(t).(Tuple)
	for i, item := range dest.Values {
		if i >= len(rhs) {
			// More variables on the left than items on the right.
			// 	(a, b, c) := (1, 2)
			err := makeMismatchTupleDestructError(dest.Values[i:], len(rhs), dc.rhsRange)
			c.fileError(err, dc.fid)
			return true
		}
		if rest, ok := item.(*ast.RestExpression); ok {
			// Rest item in destructure
			// 	(a, b...) := (1, 2, 3, 4)
			// 'b' gets (2, 3, 4)
			restTuple := rhs[i:]
			if len(restTuple) < 2 {
				// The rest item must have at least 2 items, so the rest in the
				// example above is invalid if the RHS is (1, 2) or (1, 2, 3).
				err := makeTupleRestDestructError(rest, len(restTuple), len(rhs), dc.rhsRange)
				c.fileError(err, dc.fid)
				continue
			}
			if !dc.walk(rest.Expression, restTuple) {
				return false
			}
		} else if !dc.walk(item, rhs[i]) { // Non-rest item on LHS
			return false
		}
	}
	// Currently, there can be LESS items on the left than the right
	// 	(a, b) := (1, 2, 3)
	return true
}

// The tuple destructure on the LHS has more variables than the RHS tuple's values.
func makeMismatchTupleDestructError(remaining []ast.Expression,
	rhsLen int, rhsRange ranges.Range,
) *klarerrs.Error {
	err := klarerrs.Slice(klarerrs.ErrMismatchTupleDestruct, remaining)
	err.Label = "Not enough values on the right to assign to " +
		klarerrs.FormatThis(len(remaining))
	if len(remaining) == 0 {
		err.AddHighlight("The tuple on the right has no values to destructure", rhsRange)
	} else {
		err.AddHighlight(
			"The tuple on the right has only "+klarerrs.FormatCount(rhsLen, "value"),
			rhsRange,
		)
	}
	err.SetParam("remaining", len(remaining))
	return err
}

// The rest in the tuple destructure will get less than 2 items.
func makeTupleRestDestructError(rest *ast.RestExpression,
	willGet, tupleLen int, rhsRange ranges.Range,
) *klarerrs.Error {
	// TODO: Convert rest.Expression to a string and store in err.Name
	err := klarerrs.Node(klarerrs.ErrTupleRestDestruct, rest)
	targetRange := rest.Expression.GetRange()
	switch willGet {
	case 0:
		err.AddHighlight("This variable will have no items", targetRange)
	case 1:
		err.AddHighlight("This variable will have only 1 item", targetRange)
	default:
		panic(fmt.Sprintf("unreachable: %d", willGet))
	}
	err.AddHighlight("This tuple has "+klarerrs.FormatCount(tupleLen, "item"), rhsRange)
	return err
}
