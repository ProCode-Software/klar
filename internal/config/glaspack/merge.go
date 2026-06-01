package glaspack

import "errors"

func Merge(pkg, proj *Manifest) (*Manifest, error) {
	switch {
	case pkg == nil && proj == nil:
		return nil, errors.New("Merge(pkg, proj): pkg and proj are both nil")
	case pkg == nil:
		return proj, nil
	case proj == nil:
		return pkg, nil
	}
	// TODO
	return proj, nil
}
