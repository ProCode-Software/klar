package attribute

import (
	"github.com/ProCode-Software/klar/internal/version"
)

type Platform string

type Attribute interface {
	Name() string
}

type Deprecated struct {
	Version     version.Version
	Message     string
	Replacement string
}
type Added struct {
	Version version.Version
}
type Target struct {
	Platform Platform
}
type ExternImpl struct {
	Path   string
	Export string
}
type External struct {
	Impl map[Platform]ExternImpl
}
