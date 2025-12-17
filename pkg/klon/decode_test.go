package klon

import (
	"reflect"
	"testing"
)

type (
	optionsObjValue struct {
		Options objValue
	}
	objValue struct {
		Obj value
	}
	value struct{ Value int }
)

func TestMain(t *testing.T) {
	testCase(t, "BasicString", "'Hello, World!'\n", "Hello, World!", false)
	testCase(t, "BasicInt", `1`, 1, false)
	testCase(t, "BasicBool", ` true `, true, false)
	testCase(t, "InlineList", `[4, 1, 6, 7]`, [4]uint{4, 1, 6, 7}, false)
	testCase(t, "Invalid", `3`, make(chan int), true)
	testCase(t, "anonymous struct", `options:
		- value: 2
		- obj2:
			-- value: 5`, optionsObjValue{objValue{value{5}}}, false)
	testCase(t, "anonymous struct but in braces", `
		options: {
			value: 5,
			obj2:
				- value: 42
		}`, optionsObjValue{objValue{value{42}}}, false)
}

func testCase[T any](t *testing.T,
	name, document string, expected T, wantErr bool,
) {
	t.Run(name, func(t *testing.T) {
		var v T
		err := Unmarshall([]byte(document), &v)
		switch {
		case err == nil && wantErr:
			t.Error("expected error, but got nil")
		case err != nil && !wantErr:
			t.Errorf("expected %#v, but got unmarshall error:\n\t%v", expected, err)
		case !reflect.DeepEqual(expected, v) && !wantErr:
			t.Errorf("expected %v, but got %v", expected, v)
		}
	})
}
