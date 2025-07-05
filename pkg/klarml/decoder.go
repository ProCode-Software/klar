package klarml

import (
	"fmt"
	"reflect"
)

type decoder struct {
	caller string
}
func (d *decoder) TypeError(got string, rt reflect.Type) error {
	return fmt.Errorf("%s: cannot unmarshall input type %s into expected type %T", d.caller, got, rt)
}
func (d *decoder) String(node Node, rv reflect.Value) error {
	return nil
}
func (d *decoder) Object(node Object, rv reflect.Value) error {
	rt := rv.Type()
	switch rt.Kind() {
	case reflect.Struct:
	case reflect.Map:
	case reflect.Interface:
		
	default:
		return d.TypeError("object", rt)
	}
	// Map fields in struct
	fields := make(map[string]reflect.Value)
	for i := 0; i < rt.NumField(); i++ {
		currField := rt.Field(i)
		val, ok := currField.Tag.Lookup("klarml")
		if ok {
			fields[val] = rv.Field(i)
		}
	}
	// Assign from document
	for _, prop := range node.Properties {
		name, value := prop.Key, prop.Value
		propVal, ok := fields[name]
		if !ok {
			continue
		}
		propType := propVal.Type()
		
	}
	return nil
}