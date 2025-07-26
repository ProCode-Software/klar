package main

import (
	"testing"

	"github.com/ProCode-Software/klar/pkg/klarml"
)

func TestMain(t *testing.T) {
	testCase(t, "basic string", `"Hello, World!"`+"\n", "Hello, World!")
	testCase(t, "basic int", `1`, 1)
	testCase(t, "basic boolean", ` true `, true)
}

func testCase[T comparable](t *testing.T, name, document string, expected T) {
	fail := "\033[31m[FAIL]\033[m "
	t.Run(name, func(t *testing.T) {
		var v T
		err := klarml.Unmarshall([]byte(document), &v)
		if err != nil {
			if any(expected) != "error" {
				t.Errorf(fail+"expected %#v, but got unmarshall error:\n\t%v",
					expected, err)
			}
		} else if v != expected {
			t.Errorf(fail+"expected %v, but got %v", expected, v)
		} else {
			t.Logf("\033[32m[PASS]\033[m %#q -> %#v", document, expected)
		}
	})
}
