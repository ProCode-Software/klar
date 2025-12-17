package klon

import (
	"reflect"
	"testing"
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

func Test_makeStructFields(t *testing.T) {
	type testCase struct {
		name   string
		flags  Flags
		expect int
	}
	var (
		rt    = reflect.TypeFor[testStruct]()
		cases = []testCase{
			{"default", 0, 4},
			{"with KeyedEmbeddedFields", KeyedEmbeddedFields, 5},
		}
	)
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fields, err := makeStructFields(rt, test.flags)
			if err != nil {
				t.Error(err)
			}
			if len(fields.Flat) != test.expect {
				t.Errorf("expected %d fields, got %d", test.expect, len(fields.Flat))
			}
		})
	}
}
