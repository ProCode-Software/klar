package main

import (
	"testing"

	"github.com/ProCode-Software/klar/pkg/klarml"
)

type (
	struct0       struct{ Options struct0_inner }
	struct0_inner struct {
		Value int
		Obj2  struct{ Value int }
	}
	valStruct struct{ Value int }
)

func TestMain(t *testing.T) {
	testCase(t, "basic string", `"Hello, World!"`+"\n", "Hello, World!")
	testCase(t, "basic int", `1`, 1)
	testCase(t, "basic boolean", ` true `, true)
	testCase(t, "anomynous struct", `options:
		- value: 2
		- obj2:
			-- value: 5`, struct0{struct0_inner{2, valStruct{5}}})
	testCase(t, "anomynous struct but in braces", `options: {
			value: 5, obj2: 
				- value: 42 
		}`, struct0{struct0_inner{5, valStruct{42}}})
}

func testCase[T comparable](t *testing.T, name, document string, expected T) {
	t.Run(name, func(t *testing.T) {
		var v T
		err := klarml.Unmarshall([]byte(document), &v)
		if err != nil && any(expected) != "error" {
			t.Errorf("expected %#v, but got unmarshall error:\n\t%v", expected, err)
		} else if v != expected {
			t.Errorf("expected %v, but got %v", expected, v)
		}
	})
}
