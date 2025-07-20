package main

import (
	"fmt"
	"os"

	"github.com/ProCode-Software/klar/pkg/klarml"
)

func main() {
	fp := os.Args[1]
	file, err := os.Open(fp)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	var val string
	if err = klarml.UnmarshallRead(file, &val); err != nil {
		panic(err)
	}
	fmt.Printf("%q\n", val)
}