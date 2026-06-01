package target

import (
	"fmt"
)

type Target int

const (
	Unknown    Target = iota
	JavaScript        // Any JavaScript environment
	KlarVM
	// Specifc JavaScript runtimes
	Browser
	Node
	Deno
	Bun
)

var Names = map[string]any{
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

func (t *Target) UnmarshalText(text []byte) error {
	s := string(text)
	if name, ok := Names[s]; ok {
		*t = name.(Target)
		return nil
	}
	return fmt.Errorf("Unknown target '%s'", s)
}

// IsJavaScript returns true if the target is a JavaScript environment.
func (t Target) IsJavaScript() bool {
	switch t {
	case JavaScript, Browser, Node, Deno, Bun:
		return true
	default:
		return false
	}
}
