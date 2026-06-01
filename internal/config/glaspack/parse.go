package glaspack

import (
	"github.com/ProCode-Software/klar/internal/config"
	"github.com/ProCode-Software/klar/pkg/klon"
)

func Parse(path string) (conf *Manifest, warn []*klon.Error, err error) {
	conf = &Manifest{}
	if warn, err = config.ReadFromFile(path, &conf, Context); err != nil {
		return conf, warn, err
	}
	return conf, warn, nil
}

var Context = &klon.Context{
	// TODO: classes
}
