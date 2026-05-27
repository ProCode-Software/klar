package klon

import (
	"reflect"
	"testing"

	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

type Embedded struct {
	Used  bool
	Items []struct {
		Id     int
		Object any
	}
}
type testStruct struct {
	Name string
	Id   int
	Embedded
}

type WithEmbedded struct {
	Embedded
	EmbedsAnother
}

type (
	EmbedsAnother struct{ Count }
	Count         struct{ N int }
)

const (
	unkeyed klonflags.Flags = 0
	keyed                   = klonflags.KeyedEmbeddedFields
)

func TestStructFieldCount(t *testing.T) {
	type testCase struct {
		name   string
		obj    any
		flags  klonflags.Flags
		expect int
	}

	cases := []testCase{
		{"Level1Embed_Unkeyed", testStruct{}, unkeyed, 4},
		{"Level1Embed_Keyed", testStruct{}, keyed, 5},
		{"Level2Embed_Unkeyed", WithEmbedded{}, unkeyed, 2},
		{"Level2Embed_Keyed", WithEmbedded{}, keyed, 6},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fields, err := makeStructFields(reflect.TypeOf(test.obj), test.flags)
			if err != nil {
				t.Error(err)
			}
			if len(fields.Flat) != test.expect {
				names := make([]string, len(fields.Flat))
				for i, field := range fields.Flat {
					names[i] = field.Name
				}
				t.Errorf("expected %d fields, got %d: %#v", test.expect, len(fields.Flat), names)
			}
		})
	}
}
