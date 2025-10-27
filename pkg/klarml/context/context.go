package context

type ClassDirection int

const (
	LeftClass      ClassDirection = iota // Arguments on left
	RightClass                           // Arguments on right
	LeftRightClass                       // Arguments on both sides
	VoidClass                            // No arguments
)

type Class struct {
	Name      string
	Direction ClassDirection
	// TODO: transform func
}

type Context struct {
	Classes map[string]Class
	Enums   map[string]map[string]any
}
