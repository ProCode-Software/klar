package help

type Topic struct {
	Title       string
	Description string
	SeeAlso     []string
}

var Topics = map[string]Topic{
	"js": {
		Title:       "JavaScript Compilation",
		Description: "Hello, JavaScript Compiler!",
		SeeAlso:     nil,
	},
}
