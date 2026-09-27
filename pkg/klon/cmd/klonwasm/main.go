//go:build js && wasm

package main

import (
	"github.com/ProCode-Software/klar/pkg/klon"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

func main() {}

//export parse
func Parse(s string, flags uint32) (any, string) {
	v, err := klon.UnmarshallAny([]byte(s), klonflags.Flags(flags))
	if err != nil {
		return nil, err.Error()
	}
	return v, ""
}
