package imports

import "strings"

type ImportPath []string

func (p ImportPath) String() string {
	return strings.Join(p, ".")
}
