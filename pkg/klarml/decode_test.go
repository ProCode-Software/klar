package klarml

import (
	"testing"
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
	testCase(t, "BasicString", `"Hello, World!"`+"\n", "Hello, World!")
	testCase(t, "BasicInt", `1`, 1)
	testCase(t, "BasicBool", ` true `, true)
	testCase(t, "BasicArray", `[4, 1, 6, 7]`, [...]uint{4, 1, 6, 7})
	// testCase(t, "Invalid", `3`, make(chan int))
	/* testCase(t, "anomynous struct", `options:
		- value: 2
		- obj2:
			-- value: 5`, struct0{struct0_inner{2, valStruct{5}}})
	testCase(t, "anomynous struct but in braces", `options: {
			value: 5, obj2: 
				- value: 42 
		}`, struct0{struct0_inner{5, valStruct{42}}}) */
}

func testCase[T comparable](t *testing.T, name, document string, expected T) {
	t.Run(name, func(t *testing.T) {
		var v T
		err := Unmarshall([]byte(document), &v)
		if err != nil && any(expected) != "error" {
			t.Errorf("expected %#v, but got unmarshall error:\n\t%v", expected, err)
		} else if v != expected {
			t.Errorf("expected %v, but got %v", expected, v)
		}
	})
}
