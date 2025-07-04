package types

import (
	"fmt"
	"slices"
)

func Walk(t Type, fn func(t, parent *Type)) Type {
	walk(&t, nil, fn)
	return t
}

func walk(node *Type, parent *Type, fn func(t, parent *Type)) {
	if node == nil || fn == nil {
		return
	}
	walkAll := func(s *[]Type) {
		for i, item := range *s {
			walk(&item, parent, fn)
			(*s)[i] = item
		}
	}
	// Call visitor on self
	fn(node, parent)
	switch t := (*node).(type) {
	case List:
		walk(&t.Of, node, fn)
		*node = t
	case Tuple:
		walkAll(&t.Items)
		*node = t
	case Union:
		walkAll(&t.Options)
		*node = t
	case Optional:
		walk(&t.Underlying, node, fn)
		*node = t
	case Lambda:
		for i, p := range t.Params {
			paramType := p.Type
			walk(&paramType, node, fn)
			t.Params[i].Type = paramType
		}
		walk(&t.Return, node, fn)
		*node = t
	case Result:
		walk(&t.SuccessType, node, fn)
		walk(&t.FailureType, node, fn)
		*node = t
	case Map:
		walk(&t.KeyType, node, fn)
		walk(&t.ValueType, node, fn)
		*node = t
	case Overloads:
		for _, o := range t {
			for i, p := range o.Params {
				paramType := p.Type
				walk(&paramType, node, fn)
				o.Params[i].Type = paramType
			}
			walk(&o.Return, node, fn)
		}
		*node = t
	case Ref:
		fn(t.Value, node)
		// *node = t
	// Do nothing else
	case Interface:
	case Enum:
	case Struct:
	case CoreType:
	case Untyped:
	}
}

func WalkUnionOptional(node *Type, fn func(*Type)) {
	fn(node)
	switch t := (*node).(type) {
	case Ref:
		WalkUnionOptional(t.Value, fn)
	case Union:
		for i, opt := range t.Options {
			WalkUnionOptional(&opt, fn)
			if opt == nil {
				t.Options = slices.Delete(t.Options, i, i+1)
				continue
			}
			t.Options[i] = opt
		}
		*node = t
	case Optional:
		WalkUnionOptional(&t.Underlying, fn)
		*node = t
	}
}

func FlattenUnion(union Union) Union {
	options := make([]Type, len(union.Options))
	for _, option := range union.Options {
		if opt, ok := option.(Union); ok {
			options = append(options, FlattenUnion(opt).Options...)
		} else {
			options = append(options, option)
		}
	}
	final := make([]Type, 0, len(options))
	for _, opt := range options {
		fmt.Printf("%#v\n", opt)
		if opt != nil && !slices.Contains(final, opt) {
			final = append(final, opt)
		}
	}
	return Union{slices.Clip(final)}
}
