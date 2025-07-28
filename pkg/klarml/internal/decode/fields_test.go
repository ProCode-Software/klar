package decode

import (
	"reflect"
	"testing"

	"github.com/sanity-io/litter"
)

func Test_makeStructFields(t *testing.T) {
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
	rt := reflect.TypeFor[testStruct]()
	fields, err := makeStructFields(rt, 0)
	if err != nil {
		t.Fatal(fields)
	}
	litter.Dump(fields)
}
