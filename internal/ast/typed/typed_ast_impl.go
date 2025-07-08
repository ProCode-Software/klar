package typed

import "github.com/ProCode-Software/klar/internal/ranges"

func (d BaseDecl) GetName() string {
	return d.Name
}
func (n BaseNode) At() ranges.Range {
	return n.Position
}