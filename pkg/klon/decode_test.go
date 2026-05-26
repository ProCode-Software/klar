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
		- obj:
			-- value: 5`, optionsObjValue{objValue{value{5}}}, false)
	testCase(t, "anonymous struct but in braces", `
		options: {
			value: 5,
			obj:
				- value: 42
		}`, optionsObjValue{objValue{value{42}}}, false)
}

func TestInterfaceMerge(t *testing.T) {
	t.Run("MergeIntoInitializedPointer", func(t *testing.T) {
		var i int = 10
		var anyVal any = &i
		err := Unmarshall([]byte(`20`), &anyVal)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if i != 20 {
			t.Errorf("expected i to be 20, but got %d", i)
		}
		if p, ok := anyVal.(*int); !ok || p != &i {
			t.Errorf("expected anyVal to still hold the same pointer &i")
		}
	})

	t.Run("MergeIntoNilPointer", func(t *testing.T) {
		var anyVal any = (*int)(nil)
		err := Unmarshall([]byte(`42`), &anyVal)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p, ok := anyVal.(*int)
		if !ok {
			t.Fatalf("expected anyVal to hold an *int, but got %T", anyVal)
		}
		if *p != 42 {
			t.Errorf("expected *p to be 42, but got %d", *p)
		}
	})

	t.Run("FlexibleFallbackOnTypeMismatch", func(t *testing.T) {
		var i int = 10
		var anyVal any = &i
		// Try to decode a string into an interface holding an *int.
		// It should fail to merge into *int and fallback to any (overwriting with string).
		err := Unmarshall([]byte(`'hello'`), &anyVal)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s, ok := anyVal.(string)
		if !ok {
			t.Fatalf("expected anyVal to be a string due to fallback, but got %T", anyVal)
		}
		if s != "hello" {
			t.Errorf("expected s to be 'hello', but got %q", s)
		}
		if i != 10 {
			t.Errorf("expected i to remain 10, but got %d", i)
		}
	})

	t.Run("NullClearsInterface", func(t *testing.T) {
		var i int = 10
		var anyVal any = &i
		err := Unmarshall([]byte(`null`), &anyVal)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyVal != nil {
			t.Errorf("expected anyVal to be nil after decoding null, but got %v (%T)", anyVal, anyVal)
		}
	})
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
