package target

type Target int

const (
	Unknown    Target = iota
	JavaScript        // Any JavaScript environment
	KlarVM
	Browser
	Node
	Deno
	Bun
)

var Names = map[string]Target{
	"unknown": Unknown,
	"js":      JavaScript,
	"klarvm":  KlarVM,
	"browser": Browser,
	"node":    Node,
	"deno":    Deno,
	"bun":     Bun,
}

func (t Target) String() string {
	return []string{
		Unknown:    "unknown",
		JavaScript: "js",
		KlarVM:     "klarvm",
		Browser:    "browser",
		Node:       "node",
		Deno:       "deno",
		Bun:        "bun",
	}[t]
}
