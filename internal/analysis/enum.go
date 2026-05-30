package analysis

import (
	"strings"
	"unicode"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

// Not a value.
type Enum struct {
	ItemType     Type
	Items        []*EnumItem
	Union        *EnumItem // Params in common with all items. Can be nil
	itemMap      map[string]*EnumItem
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

func (*Enum) Kind() Kind                        { return KindEnum }
func (*Enum) String() string                    { return "" }
func (*Enum) StringWithName(name string) string { return name }

func (item *EnumItem) ParamByName(label string) Type {
	i, ok := item.paramMap[label]
	if !ok {
		return nil
	}
	return item.Params[i]
}

func (c *Checker) checkEnumDecl(o *Object, node *ast.EnumDeclaration, fctx *Context) {
	e := o.typ.(*Enum)
	// TODO: Generics
	e.Items = make([]*EnumItem, 0, len(node.Values))
	valueMap := make(map[ConstValue]*EnumItem)
	for _, entry := range node.Values {
		ei := &EnumItem{
			Name:       entry.Identifier.Name,
			Attributes: c.parseAttributes(entry.Attributes, enumVariantAttribute),
		}
		e.Items = append(e.Items, ei)
		e.itemMap[ei.Name] = ei

		// Value of [ConstValue.ConstValue](), or casing mode if [StringConst]
		var firstValue any
		// Value - must be unique for each item
		if entry.Value != nil {
			// Check type of value
			cons := c.checkEnumValue(entry.Value, fctx)
			valType := cons.Type()
			if e.ItemType == nil {
				// First value. Determine type
				e.ItemType = valType
				firstValue = cons.ConstValue()
				// For strings, determine casing mode
				if valType == StringType {
					str := cons.ConstValue().(string)
					firstValue = getCasingMode(ei.Name, str)
				}
			} else if e.ItemType != valType {
				// TODO: Untyped Int then Float is allowed

				err := klarerrs.Node(klarerrs.ErrTypeMismatch, entry.Value)
				err.Label = "Enum values must have the same type"
				err.SetParam("expected", e.ItemType)
				err.SetParam("actual", valType)
				c.fileError(err, o.file)
			}
			// Check uniqueness of value
			if otherItem, ok := valueMap[cons]; ok {
				err := klarerrs.Node(klarerrs.ErrEnumSameValue, entry.Value)
				err.Label = "Enum values must be unique"
				err.SetParam("key", ei.Name)
				err.SetParam("otherKey", otherItem.Name)
				err.AddDetail(
					"Item "+klarerrs.Quote(otherItem.Name)+" was declared here",
					c.module.ResolveFile(o.file), entry.Value.GetRange(),
				)
				c.fileError(err, o.file)
			} else {
				valueMap[cons] = ei
			}
		} else { // No explicit value
			var value ConstValue
			switch e.ItemType {
			case nil:
				// First value
				// Enum values are Int by default
				e.ItemType = IntType
				firstValue = int64(0)

			// Infer item value. None of these will be the first value.
			case IntType:
				value = IntConst{firstValue.(int64) + int64(len(e.Items)) - 1}
			case FloatType:
				value = FloatConst{firstValue.(float64) + float64(len(e.Items)) - 1}
			case StringType:
				switch firstValue.(casingMode) {
				case noCasePattern:
					// Can't infer this value
				}
			default:
				panic("invalid enum item type: " + TypeToString(e.ItemType))
			}
			ei.Value = value
			valueMap[value] = ei
		}

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

func toPascalCase(s string) string {
	return s
}

func (c *Checker) checkEnumValue(expr ast.Expression, ctx *Context) ConstValue {
	return nil
}
