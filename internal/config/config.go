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
		ctx = &klon.Context{}
	} else {
		ctx.Warnings = ctx.Warnings[:0] // Clear previous warnings
	}
	ctx.SetWarningKinds(klonerrs.ErrFieldNotFound)

	if err = klon.UnmarshallReadContext(fr, config, ctx, DefaultKlonFlags); err != nil {
		return ctx.Warnings, err
	}
	return ctx.Warnings, nil
}
