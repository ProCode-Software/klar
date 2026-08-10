package config

import (
	"os"

	"github.com/ProCode-Software/klar/pkg/klon"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

const DefaultKlonFlags = klonflags.NoUnknownFields

func ReadFromFile[T any](path string, config *T, ctx *klon.Context) (warn []*klon.Error, err error) {
	var fr *os.File
	fr, err = os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fr.Close()

	if ctx == nil {
		ctx = klon.NewContext()
	} else {
		ctx.Warnings = ctx.Warnings[:0] // Clear previous warnings
	}
	ctx.SetWarningKinds(klonerrs.ErrFieldNotFound)

	if err = ctx.UnmarshallRead(fr, config, DefaultKlonFlags); err != nil {
		return ctx.Warnings, err
	}
	return ctx.Warnings, nil
}

const KlonIndentSize = 4

// It is recommended to write an AST to a file, rather than a struct directly.
func WriteToFile[T any](path string, config T, ctx *klon.Context) (err error) {
	_ = ctx
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	return klon.MarshallWrite(config, KlonIndentSize, f)
}
