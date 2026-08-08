package klon

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
)

func TestDecodePrimitive(t *testing.T) {
	t.Run("BasicString", func(t *testing.T) {
		var v string
		input := "'Hello, World!'\n"
		if err := Unmarshall([]byte(input), &v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "Hello, World!" {
			t.Errorf("expected 'Hello, World!', got %q", v)
		}
	})
	t.Run("BasicInt", func(t *testing.T) {
		var v int
		if err := Unmarshall([]byte("1"), &v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 1 {
			t.Errorf("expected 1, got %d", v)
		}
	})
	t.Run("BasicBool", func(t *testing.T) {
		var v bool
		if err := Unmarshall([]byte(` true `), &v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != true {
			t.Errorf("expected true, got %v", v)
		}
	})
}

func TestDecodeCollection(t *testing.T) {
	t.Run("InlineList", func(t *testing.T) {
		var v [4]uint
		unmarshal(t, `[4, 1, 6, 7]`, &v)
		expected := [4]uint{4, 1, 6, 7}
		if v != expected {
			t.Errorf("expected %v, got %v", expected, v)
		}
	})
}

func unmarshal[V any](t *testing.T, input string, v *V) {
	t.Helper()
	if err := Unmarshall([]byte(input), v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertErrorCode(t *testing.T, err error, code klonerrs.Code) {
	t.Helper()
	switch err := err.(type) {
	case nil:
		t.Fatal("expected an error")
	case *Error:
		if err.Code != code {
			t.Errorf("expected code %d, got %d", code, err.Code)
		}
	default:
		t.Errorf("expected *Error, got %T", err)
	}
}

func assertNonKlonError(t *testing.T, err error) {
	t.Helper()
	switch err.(type) {
	case nil:
		t.Fatal("expected an error")
	case *Error:
		t.Errorf("expected non-klon error, got *Error")
	}
}

func TestDecodeStruct(t *testing.T) {
	t.Run("NestedStructs", func(t *testing.T) {
		type Port struct {
			Number int
			Host   string
		}
		type ServerOptions struct{ Port Port }
		type BuildConfig struct {
			Server    ServerOptions
			HotReload bool
		}

		var v BuildConfig
		input := `
		server:
			- port:
				-- host: 'localhost'
				-- number: 3000
		hotReload: true`
		unmarshal(t, input, &v)
		exp := BuildConfig{ServerOptions{Port{3000, "localhost"}}, true}
		if !reflect.DeepEqual(v, exp) {
			t.Errorf("expected %+v, got %+v", exp, v)
		}
	})
	t.Run("NestedStructsInBraces", func(t *testing.T) {
		type Port struct {
			Number int
			Host   string
		}
		type ServerOptions struct{ Port Port }
		type BuildConfig struct {
			Server    ServerOptions
			HotReload bool
		}
		var v BuildConfig
		input := `{
			server: { port: { host: 'localhost', number: 3000 } }
			hotReload: true
		}`
		unmarshal(t, input, &v)
		exp := BuildConfig{ServerOptions{Port{3000, "localhost"}}, true}
		if !reflect.DeepEqual(v, exp) {
			t.Errorf("expected %+v, got %+v", exp, v)
		}
	})
}

func TestDecodeInterface(t *testing.T) {
	t.Run("PointerInterface", func(t *testing.T) {
		var i int = 10
		var anyVal any = &i
		if err := Unmarshall([]byte(`20`), &anyVal); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if i != 20 {
			t.Errorf("expected i == 20, but got %d", i)
		}
		if p, ok := anyVal.(*int); !ok || p != &i {
			t.Errorf("expected anyVal to still hold the same pointer &i")
		}
	})
	t.Run("NilPointerInterface", func(t *testing.T) {
		var anyVal any = (*int)(nil)
		if err := Unmarshall([]byte(`42`), &anyVal); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p, ok := anyVal.(*int)
		if !ok {
			t.Fatalf("expected anyVal.(*int), but got %T", anyVal)
		}
		if *p != 42 {
			t.Errorf("expected *p to be 42, but got %d", *p)
		}
	})
	t.Run("InterfaceFallback", func(t *testing.T) {
		// When unmarshalling into a pointer interface, try to mutate the
		// pointer value if compatible, otherwise change the interface type from
		// the pointer.
		var i int = 10
		var anyVal any = &i
		unmarshal(t, `'hello'`, &anyVal)
		s, ok := anyVal.(string)
		if !ok {
			t.Fatalf("expected anyVal.(string) due to fallback, but got %T", anyVal)
		}
		if s != "hello" {
			t.Errorf(`expected s == "hello", but got %q`, s)
		}
		if i != 10 {
			t.Errorf("expected i == 10, but got %d", i)
		}
	})
}

func TestDecodeInvalid(t *testing.T) {
	t.Run("Chan", func(t *testing.T) {
		var v chan int
		input := `3`
		assertNonKlonError(t, Unmarshall([]byte(input), &v))
	})
	t.Run("InterfaceWithMethods", func(t *testing.T) {
		var v interface{ Len() int }
		input := `3`
		assertNonKlonError(t, Unmarshall([]byte(input), &v))
	})
}

type (
	klonUnmarshallerVersion struct{ Major, Minor int }
	textUnmarshalerVersion  klonUnmarshallerVersion
	bothUnmarshallerVersion struct {
		Major, Minor int
		UsedKlon     bool
	}
)

func (v *klonUnmarshallerVersion) UnmarshallKlon(node ast.Node) error {
	s, ok := node.(*ast.String)
	if !ok {
		return fmt.Errorf("expected string")
	}
	_, err := fmt.Sscanf(s.Raw, "%d.%d", &v.Major, &v.Minor)
	return err
}

func (v *textUnmarshalerVersion) UnmarshalText(text []byte) error {
	_, err := fmt.Sscanf(string(text), "%d.%d", &v.Major, &v.Minor)
	return err
}

func (v *bothUnmarshallerVersion) UnmarshallKlon(node ast.Node) error {
	v.UsedKlon = true
	return nil
}

func (v *bothUnmarshallerVersion) UnmarshalText(text []byte) error {
	v.UsedKlon = false
	return nil
}

func TestDecodeCustom(t *testing.T) {
	t.Run("KlonUnmarshaller", func(t *testing.T) {
		var v klonUnmarshallerVersion
		if err := Unmarshall([]byte("'1.2'"), &v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != (klonUnmarshallerVersion{1, 2}) {
			t.Errorf("expected 1.2, got %d.%d", v.Major, v.Minor)
		}
	})
	t.Run("TextUnmarshaler", func(t *testing.T) {
		var v textUnmarshalerVersion
		if err := Unmarshall([]byte("'3.4'"), &v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != (textUnmarshalerVersion{3, 4}) {
			t.Errorf("expected 3.4, got %d.%d", v.Major, v.Minor)
		}
	})
	t.Run("KlonUnmarshallerPriority", func(t *testing.T) {
		var v bothUnmarshallerVersion
		if err := Unmarshall([]byte("'any'"), &v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !v.UsedKlon {
			t.Error("expected UnmarshallKlon to be used over UnmarshalText")
		}
	})
	t.Run("ErrorWrapping", func(t *testing.T) {
		var v klonUnmarshallerVersion
		err := Unmarshall([]byte("123"), &v) // Not a string
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if ke, ok := err.(*Error); !ok || ke.Code != klonerrs.ErrUnmarshallerError {
			t.Errorf("expected ErrUnmarshallerError, got %v", err)
		}
	})
}
