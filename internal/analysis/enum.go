package analysis

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// Not a value.
type Enum struct {
	ItemType     Type
	Items        []*EnumItem
	itemMap      map[string]*EnumItem
	Union        *EnumItem // Params in common with all items. Can be nil
	Generics     []*Generic
	Methods      []*Object // Type [*Function]
	Initializers []*Object // Type [*Overload]
}

// Can be used as a value.
type EnumItem struct {
	Name       string
	Params     []Type
	paramMap   map[string]int
	Value      ConstValue
	Enum       *Enum // For access to value type and methods
	Attributes *Attributes
}

func (*Enum) Kind() Kind       { return KindEnum }
func (e *Enum) String() string { return fmt.Sprintf("<enum (%d items)>", len(e.Items)) }

func (item *EnumItem) ParamByName(label string) Type {
	i, ok := item.paramMap[label]
	if !ok {
		return nil
	}
	return item.Params[i]
}

func (c *Checker) checkEnumDecl(o *Object, node *ast.EnumDeclaration) {
	fctx := o.FileContext()
	e := &Enum{
		Items:    make([]*EnumItem, 0, len(node.Values)),
		itemMap:  make(map[string]*EnumItem, len(node.Values)),
		Generics: c.parseGenerics(node.Generics, o.file, fctx),
	}
	o.typ.(*TypeName).Type = e

	// Keep track of unique values
	valueMap := make(map[ConstValue]*EnumItem)
	// Value of [ConstValue.ConstValue](), or casing mode if [StringConst]
	var firstValue any
	for _, entry := range node.Values {
		ei := &EnumItem{
			Name:       entry.Identifier.Name,
			Attributes: c.parseAttributes(entry.Attributes, enumVariantAttribute, o.file),
		}
		e.Items = append(e.Items, ei)
		e.itemMap[ei.Name] = ei

		// Value - must be unique for each item
		c.checkEnumValue(
			o, e, ei, entry.Range, entry.Value,
			valueMap, &firstValue, &ranges.Range{}, fctx,
		)

		// Params
		if ei.Params == nil {
			continue
		}
		ei.Params = make([]Type, 0, len(entry.Parameters.Values))
		ei.paramMap = make(map[string]int, len(entry.Parameters.Values))
		for _, p := range entry.Parameters.Values {
			typ := c.parseType(p.Value, fctx) // TODO: Context should include the generic
			for _, name := range p.Keys {
				if name.Name != "" && !name.IsDiscard() {
					if other, ok := ei.paramMap[name.Name]; ok {
						// Parameter has the same name
						_ = other
						c.fileError(nil, o.file)
					} else {
						ei.paramMap[name.Name] = len(ei.Params)
					}
				}
			}
			ei.Params = append(ei.Params, typ)
		}
	}
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
			c.fileError(err, o.file)
		}

		// Check uniqueness of value
		if otherItem, ok := valueMap[cons]; ok {
			err := klarerrs.Node(klarerrs.ErrEnumSameValue, expr)
			err.Label = "Enum values must be unique"
			err.SetParam("key", ei.Name)
			err.SetParam("otherKey", otherItem.Name)
			err.AddDetail(
				"Item "+klarerrs.Quote(otherItem.Name)+" was declared here",
				c.module.ResolveFile(o.file), expr.GetRange(),
			)
			c.fileError(err, o.file)
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
				c.fileError(klarerrs.Range(klarerrs.ErrCantInferStringEnum, r), o.file)
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
