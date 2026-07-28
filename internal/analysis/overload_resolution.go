package analysis

import (
	"maps"
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// A paramSet represents the parameters passed to a function call, used
// for overload resolution. Rests in variadic parameters are represented
// as their element type as a single parameter.
type paramSet struct {
	params   []Type
	nodeMap  []ast.Expression // For positional params
	labelled map[string]Type
	// Labelled params with multiple values. This is separate from labelled
	// because I don't expect many labelled params to be variadic in practice.
	variadicLabelled map[string][]Type
}

func (ps paramSet) String() string {
	var b strings.Builder
	b.WriteByte('(')
	for i, typ := range ps.params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(typ.String())
	}
	for name, typ := range ps.labelled {
		if b.Len() > 1 {
			b.WriteString(", ")
		}
		b.WriteString(name)
		b.WriteString(": ")
		if len(ps.variadicLabelled[name]) == 0 {
			// Single value
			b.WriteString(typ.String())
		} else {
			// Label with multiple variadic values
			for i, typ := range ps.variadicLabelled[name] {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(typ.String())
			}
		}
	}
	b.WriteByte(')')
	return b.String()
}

// Does not guarantee correct arity
func (c *Checker) resolveOverload(overloads []*Overload, ps paramSet) (*Overload, *klarerrs.Error) {
	if len(overloads) == 1 {
		return overloads[0], nil
	}
	scores := make(map[*Overload]int)
	// checkWinner returns nil if there is no clear winner. A clear winner is one
	// that has the highest score, with no other overload having an equal score.
	checkWinner := func() (highest *Overload) {
		var bestScore int
		isFirst := true
		for ov, score := range scores {
			switch {
			case isFirst:
				highest, bestScore = ov, score
				isFirst = false
			case score > bestScore:
				highest, bestScore = ov, score
			case score == bestScore:
				highest = nil
			}
		}
		return highest
	}

	// Pass 1: Score based on whether the param is a concrete type.
	scoreOverloadByConcrete(overloads, scores, ps)
	if winner := checkWinner(); winner != nil {
		return winner, nil
	}

	// Retain only the top overloads
	var (
		top       = []*Overload{}
		bestScore int
		isFirst   = true
	)
	for _, ov := range overloads {
		score := scores[ov]
		switch {
		case isFirst:
			top = append(top, ov)
			bestScore = score
			isFirst = false
		case score > bestScore:
			top = top[:0]
			top = append(top, ov)
			bestScore = score
		case score == bestScore: // Negative score
			top = append(top, ov)
		}
	}

	// Compare the broadness of the interfaces in each overload. The overload with the
	// overall least broad interfaces wins.
	sortOverloadsByBroadness(top, scores, ps)
	if winner := checkWinner(); winner != nil {
		return winner, nil
	}
	// If there is no clear winner at this point, there is a bug. We may have
	// not validated the declarations enough. We will tell the user to report an
	// issue. In this case `top[0]` is tied with other overloads, but it will be
	// selected. This choice may not be what the user is expecting.
	warn := klarerrs.Range(klarerrs.WarnOverloadResolve, ranges.Range{}).MarkWarning()
	warn.Label = quote(ps.String()) + " was provided"
	warn.Desc = "The type checker picks the best overload by scoring each declared overload, multiple overloads are tied for the highest score. As a last resort, we picked " +
		quote(top[0].StringWithName(top[0].Name)) + ".\n" +
		"This isn't your fault; please file an issue on GitHub."
	return top[0], warn
}

// scoreOverloadByConcrete scores based on whether the param is a concrete type.
// This is faster, but has limitations. Suppose we have:
//
//	// Assuming the same interface structure as Klar's AST, and parameters
//	// implement the one above
//	func fn(_: Node)
//	func fn(_: Statement)
//	func fn(_: TypeDeclaration)
//
//	fn(StructDeclaration)
//
// There is no clear winner because all overloads take non-concrete types.
// If one overload was a StructDeclaration, there would be a winner. Pass 2
// solves this by sorting each interface by smallest to largest (specificity -
// so TypeDeclaration wins because Node and Statement don't implement it)
func scoreOverloadByConcrete(overloads []*Overload, scores map[*Overload]int, ps paramSet) {
	scoreParam := func(exp, got Type) int {
		switch {
		case !Compatible(got, exp):
			return 0
		case !IsConcreteType(exp) && !TypesEqual(got, exp):
			// fn(Intf) <- fn(Impl) scores less than fn(Impl) <- fn(Impl)
			// But fn(Impl) <- fn(Impl) is an exact match
			return 1
		default:
			return 2 // Concrete type
		}
	}
	for _, ov := range overloads {
		// Positional params
		if !ov.Arity.InRange(len(ps.params)) {
			// Scores much less points, but don't skip it
			// TODO: Should we take more points off based on number of params?
			scores[ov] -= 5
		}
		for i, got := range ps.params {
			if len(ov.Params) == 0 {
				break
			}
			exp := ov.Params[min(i, len(ov.Params)-1)]
			expType := exp.Type
			if exp.Object.Flags.Has(VariadicParam) {
				expType = expType.(*List).Elem
			} else if i >= len(ov.Params) {
				break
			}
			scores[ov] += scoreParam(expType, got)
		}

		// Labelled params
		for name, got := range ps.labelled {
			exp, ok := ov.labelMap[name]
			if !ok {
				scores[ov] -= 3
				continue
			}
			expType := exp.Type
			if exp.Object.Flags.Has(VariadicParam) {
				expType = expType.(*List).Elem
			}
			scores[ov] += scoreParam(expType, got)
		}
		// Check if any required labelled params are missing
		for name := range ov.labelMap {
			if _, ok := ps.labelled[name]; !ok {
				scores[ov] -= 2
			}
		}
	}
}

func sortOverloadsByBroadness(overloads []*Overload, scores map[*Overload]int, ps paramSet) {
	scoreTypes := func(expA, expB, _ Type, sizeA, sizeB *int) {
		compatAB, compatBA := Compatible(expA, expB), Compatible(expB, expA)
		switch {
		case compatAB && !compatBA:
			*sizeB++
		case !compatAB && compatBA:
			*sizeA++
		}
	}
	// Smallest type first
	slices.SortFunc(overloads, func(a, b *Overload) int {
		var sizeA, sizeB int
		for i, got := range ps.params {
			if len(a.Params) == 0 || len(b.Params) == 0 {
				break
			}
			var expA Type = a.Params[min(i, len(a.Params)-1)]
			if elem := isVariadicParam(expA); elem != nil {
				expA = elem
			}
			var expB Type = b.Params[min(i, len(b.Params)-1)]
			if elem := isVariadicParam(expB); elem != nil {
				expB = elem
			}
			scoreTypes(expA, expB, got, &sizeA, &sizeB)
		}
		for name, got := range ps.labelled {
			expA, okA := a.labelMap[name]
			if !okA {
				sizeA += 3 // Least points wins, so increase the size for a penalty
			}
			expB, okB := b.labelMap[name]
			if !okB {
				sizeB += 3
			}
			if !okA || !okB {
				continue
			}
			scoreTypes(expA, expB, got, &sizeA, &sizeB)
		}
		// For the scores map, highest wins
		if sizeB > sizeA {
			scores[a] += 1
			return -1
		} else if sizeA > sizeB {
			scores[b] += 1
			return 1
		}
		return 0 // Same size
	})
}

// checkOverloadAmbiguity checks the given overloads for duplicates and ambiguous
// options (as described in the [Function Overloads] section in the Klar
// Type System). If there are errors, the first of each conflicting overload
// pair is retained in the result.
//
// [Function Overloads]:
func (c *Checker) checkOverloadAmbiguity(overloads []*Overload) []*Overload {
	origCount := len(overloads)
	if origCount == 1 {
		return overloads // Nothing to check
	}
	// Sort by arity so we can make pairs using i+1 and i-1
	byArity := slices.Clone(overloads)
	slices.SortStableFunc(byArity, sortByArity)

	var overloadsWithErrors map[*Overload]struct{}
	addError := func(ov *Overload) {
		if overloadsWithErrors == nil {
			overloadsWithErrors = make(map[*Overload]struct{})
		}
		overloadsWithErrors[ov] = struct{}{}
	}
	checked := make(map[[2]*Overload]struct{}, origCount*origCount-origCount)
	// First, check for redeclared overloads
	for _, a := range byArity {
		for _, b := range byArity {
			// Already checked
			_, ok := checked[[2]*Overload{a, b}]
			_, ok2 := checked[[2]*Overload{b, a}]
			if ok || ok2 || a == b {
				continue
			}

			// TODO: This has to be looped in a quadratic fashion
			// Otherwise, a setup in this order wouldn't report an error:
			//
			// 	func isNumber(char: String) = char in '0'...'9'
			//  func isNumber(char: Int) = char in '0'...'9'
			//  func isNumber(char: String) = char in '0'...'9'
			if ok := c.checkRedeclaredOverload(a, b); !ok {
				// The 2nd redeclaration is the one with the error
				err := klarerrs.Range(klarerrs.ErrRedeclaredOverload, b.Range)
				err.Name = a.StringWithName(a.Name)
				err.Label = "An overload with these same parameters already exists"
				err.AddDetail("It was already declared here", a.FilePath(), a.Range)
				c.fileError(err, b.File)
				addError(b)
			}
		}
	}

	if len(overloadsWithErrors) == 0 {
		return overloads // No errors
	}
	// Retain only the overloads without errors
	deduped := make([]*Overload, 0, origCount-len(overloadsWithErrors))
	for _, ov := range overloads {
		if _, ok := overloadsWithErrors[ov]; !ok {
			deduped = append(deduped, ov)
		}
	}
	return deduped
}

// Overload Validation
// ======

func sortByArity(a, b *Overload) int {
	if byMin := a.Arity.MinParams - b.Arity.MinParams; byMin != 0 {
		return byMin
	}
	// -1 is considered greater
	if a.Arity.MaxParams == -1 || b.Arity.MaxParams == -1 {
		return b.Arity.MaxParams - a.Arity.MaxParams
	}
	return a.Arity.MaxParams - b.Arity.MaxParams
}

func (c *Checker) checkRedeclaredOverload(a, b *Overload) (ok bool) {
	if a.Arity != b.Arity {
		return true
	}
	typesEqual := func(a, b *Variable) bool { return TypesEqual(a.Type, b.Type) }
	equal := slices.EqualFunc(a.Params, b.Params, typesEqual) &&
		maps.EqualFunc(a.labelMap, b.labelMap, typesEqual)
	return !equal
}
