//go:build js && wasm

// TODO: Compiling this module doesn't work. We'll try to find a way to port Klon to Wasm.

package main

import (
	"encoding/json/v2"

	"github.com/ProCode-Software/klar/pkg/klon"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

func main() {}

//export parse
func Parse(s string, flags uint32) (res struct {
	value string
	error string
},
) {
	v, err := klon.UnmarshallAny([]byte(s), klonflags.Flags(flags))
	if err != nil {
		res.error = err.Error()
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		res.error = err.Error()
		return
	}
	res.value = string(b)
	return
}
