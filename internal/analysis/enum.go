package analysis

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type Enum struct {
	ItemType     Type
	Name         string
	Inherited    map[Type]struct{} // Enums and tags
	Items        []*EnumItem
	itemMap      map[string]*EnumItem
	Union        *EnumItem // Params in common with all items. Can be nil
	Generics     []*Generic
	Methods      []*Object // Type [*Function]
	Initializers []*Object // Type [*Overload]
	MethodSet
}

// Not a value.
type EnumItem struct {
	*Object  // Type is [*EnumItem]
	Params   []Type
	paramMap map[string]int
	Value    ConstValue
	Enum     *Enum // For access to value type and methods
}

func (ei *EnumItem) Underlying() Type { return ei }
func (*EnumItem) Kind() Kind          { return KindEnum }
func (*EnumItem) objKind()            {}
func (ei *EnumItem) String() string   { return ei.Enum.Name }

func (*Enum) Kind() Kind       { return KindEnum }
func (e *Enum) String() string { return e.Name }

// Can be used as a value. An EnumRef can be indexed by the enum's methods,
// the item's param names, and the builtin `name` and `value` fields.
type EnumRef struct {
	*EnumItem
	// Uncomment for constant analysis
	// Params []Type // Nil if Called is false or there are no params
	Called bool // If the params have been passed. True if there are no params.
}

func NewEnumRef(ei *EnumItem) *EnumRef {
	return &EnumRef{EnumItem: ei, Called: len(ei.Params) == 0}
}

type EnumFunction struct {
	*Lambda
	*EnumItem
}

func (ef *EnumFunction) String() string   { return ef.Enum.Name }
func (ef *EnumFunction) Kind() Kind       { return KindFunction }
func (ef *EnumFunction) Underlying() Type { return ef }

func newEnumFunction(ei *EnumItem) *EnumFunction {
	return &EnumFunction{
		Lambda: &Lambda{
			Params:   ei.Params,
			Return:   &EnumRef{EnumItem: ei, Called: true},
			Complete: true,
		},
		EnumItem: ei,
	}
}

func (er *EnumRef) Kind() Kind {
	if er.Called {
		return KindEnum
	}
	return KindFunction
}
func (er *EnumRef) String() string { return er.Enum.Name }
func (er *EnumRef) Underlying() Type {
	if er.Called {
		return er
	}
	return newEnumFunction(er.EnumItem)
}

func (c *Checker) checkEnumDecl(o *Object, node *ast.EnumDeclaration) {
	fctx := o.LookupContext()
	e := &Enum{
		Name:      o.Name,
		Items:     make([]*EnumItem, 0, len(node.Values)),
		itemMap:   make(map[string]*EnumItem, len(node.Values)),
		Generics:  c.parseGenerics(node.Generics, o.File, fctx),
		Inherited: c.checkInheritedTypes(node.Inherited, KindEnum, fctx),
	}
	o.TypeName().Type = e

	// Keep track of unique values
	valueMap := make(map[ConstValue]*EnumItem)
	// Value of [ConstValue.ConstValue](), or casing mode if [StringConst]
	var firstValue any
	for _, entry := range node.Values {
		ei := &EnumItem{Enum: e, Object: NewObject(
			entry.Identifier.Name, o.File, entry.Range, o.Module, nil,
		)}
		e.Items = append(e.Items, ei)
		e.itemMap[ei.Name] = ei // Duplicates are checked during parsing
		ei.Object.Type = ei
		ei.Object.attrs = c.parseAttributes(
			entry.Attributes, attrTargetKindOf(entry, true), entry.Range, o.File,
		)

		// Value - must be unique for each item
		c.checkEnumValue(
			o, e, ei, entry.Range, entry.Value,
			valueMap, &firstValue, &ranges.Range{}, fctx,
		)

		// Params
		if entry.Parameters == nil || len(entry.Parameters.Values) == 0 {
			continue
		}
		ei.Params = make([]Type, 0, len(entry.Parameters.Values))
		ei.paramMap = make(map[string]int, len(entry.Parameters.Values))
		for _, pair := range entry.Parameters.Values {
			typ := c.parseType(pair.Value, fctx) // TODO: Context should include the generic
			for _, key := range pair.Keys {
				ei.Params = append(ei.Params, typ)
				if key.IsDiscard() {
					continue
				}
				if _, ok := ei.paramMap[key.Name]; ok {
					err := klarerrs.Node(klarerrs.ErrRedeclaredParamLabel, key)
					err.Label = "A parameter named " + quote(key.Name) + " already exists"
					err.AddHighlight(
						"It was first defined here",
						firstParamDecl(entry.Parameters.Values, key.Name).Range(),
					)
					c.fileError(err, o.File)
				} else {
					ei.paramMap[key.Name] = len(ei.Params) - 1
				}
			}
			if len(pair.Keys) == 0 {
				ei.Params = append(ei.Params, typ)
			}
		}
	}
}

func firstParamDecl(params []*ast.TypePair, name string) ast.Identifier {
	for _, param := range params {
		for _, key := range param.Keys {
			if key.Name == name {
				return key
			}
		}
	}
	panic("param " + name + " not found")
}

func (c *Checker) checkEnumValue(o *Object, e *Enum, ei *EnumItem,
	r ranges.Range, expr ast.Expression,
	valueMap map[ConstValue]*EnumItem, firstValue *any, firstRange *ranges.Range,
	fctx *Context,
) {
	if expr != nil {
		// Parse the expression as a constant and validate uniqueness
		cons := c.checkEnumValueExpr(expr, fctx)
		valType := cons.Type()
		if *firstValue == nil {
			// First value. Determine type for the entire enum
			e.ItemType = valType
			*firstValue = cons.ConstValue()
			// For strings, determine casing mode and store that in firstValue
			if valType == StringType {
				str := cons.ConstValue().(string)
				*firstValue = getCasingMode(ei.Name, str)
			}
		} else if e.ItemType != valType { // Type mismatch
			// TODO: Untyped Int then Float is allowed
			err := typeMismatch(e.ItemType, valType, expr.GetRange())
			if !firstRange.IsZero() {
				err.AddHighlight(
					"First value of the enum has type "+klarerrs.Quote(e.ItemType.String()),
					*firstRange,
				)
			}
			c.fileError(err, o.File)
		}

		// Check uniqueness of value
		if otherItem, ok := valueMap[cons]; ok {
			err := klarerrs.Node(klarerrs.ErrEnumSameValue, expr)
			err.Label = "Enum values must be unique"
			err.SetParam("key", ei.Name)
			err.SetParam("otherKey", otherItem.Name)
			err.AddDetail(
				"Item "+klarerrs.Quote(otherItem.Name)+" was declared here",
				c.module.ResolveFile(o.File), expr.GetRange(),
			)
			c.fileError(err, o.File)
		} else {
			valueMap[cons] = ei
		}
	} else {
		// No explicit value
		var value ConstValue
		i := len(e.Items) - 1
		switch e.ItemType {
		case nil:
			// First value
			// Enum values are Int by default
			e.ItemType = IntType
			*firstValue = int64(0)

		// Infer item value. None of these will be the first value.
		case IntType:
			// First value (or 0) + index of current item
			value = IntConst{(*firstValue).(int64) + int64(i)}
		case FloatType:
			value = FloatConst{(*firstValue).(float64) + float64(i)}
		case StringType:
			// Set the value to the name in a modified case (based on first value)
			var str string
			switch (*firstValue).(casingMode) {
			case noCasePattern:
				// Can't infer this value
				c.fileError(klarerrs.Range(klarerrs.ErrCantInferStringEnum, r), o.File)
				str = ei.Name
			case nameCase:
				str = ei.Name
			case lowerCasing:
				str = strings.ToLower(ei.Name)
			case upperCasing:
				str = strings.ToUpper(ei.Name)
			case pascalCasing:
				str = toPascalCase(ei.Name)
			default:
				panic(fmt.Sprintf(
					"invalid string casing mode: %d", (*firstValue).(casingMode),
				))
			}
			value = NewStringConst(str)
		default:
			panic("invalid enum item type: " + e.ItemType.String())
		}
		ei.Value = value
		valueMap[value] = ei
	}
}

type casingMode int

const (
	noCasePattern casingMode = iota
	nameCase                 // Value is the same as name
	upperCasing
	lowerCasing
	pascalCasing
)

func getCasingMode(name, value string) casingMode {
	if len(name) != len(value) {
		return noCasePattern
	}
	switch value {
	case name:
		return nameCase
	case strings.ToUpper(name):
		return upperCasing
	case strings.ToLower(name):
		return lowerCasing
	default:
		// Check for PascalCase
		var firstLower int
		for i, r := range value {
			if unicode.IsLower(r) {
				firstLower = i
				break
			}
		}
		if firstLower > 0 &&
			value == strings.ToUpper(name[:firstLower])+name[firstLower:] {
			return pascalCasing
		}
		return noCasePattern
	}
}

func toPascalCase(s string) string { return strings.ToUpper(s[:1]) + s[1:] }

func (c *Checker) checkEnumValueExpr(expr ast.Expression, ctx *Context) ConstValue {
	return &IntConst{0} // TODO
}

func (item *EnumItem) ParamByName(label string) Type {
	if item.paramMap == nil {
		return nil
	}
	i, ok := item.paramMap[label]
	if !ok {
		return nil
	}
	return item.Params[i]
}

func (e *Enum) Index(name string, t *Expr) *klarerrs.Error {
	if item, ok := e.itemMap[name]; ok {
		t.Type = NewEnumRef(item)
		return nil
	}
	// Show a more concise error message if the user tries to access a method
	if e.methodMap != nil {
		if _, ok := e.methodMap[name]; ok {
			err := &klarerrs.Error{
				Code:  klarerrs.ErrIndexEnumMethod,
				Label: "Choose an enum item to access this method",
				Name:  name,
			}
			return err
		}
	}
	return fieldNotFound(name)
}
