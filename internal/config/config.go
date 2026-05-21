package config

import (
	"os"

	"github.com/ProCode-Software/klar/pkg/klon"
)

func ReadFromFile[T any](path string, config *T, ctx *klon.Context) (err error) {
	var fr *os.File
	fr, err = os.Open(path)
	if err != nil {
		return err
	}
	defer fr.Close()

	if err = klon.UnmarshallReadContext(fr, config, ctx); err != nil {
		return err
	}
	return nil
}
