package klarbuild

import "github.com/ProCode-Software/klar/pkg/klon"

var Context = klon.NewContext()

func Parse(path string) (config *File, err error) {
	return &File{Configuration: Configuration{}}, nil
}

func Default() *File {
	return &File{Configuration: Configuration{}}
}
