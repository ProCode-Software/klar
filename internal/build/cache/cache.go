package cache

import (
	"encoding/gob"
	"hash/fnv"
	"strconv"
	"time"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
)

type Module struct {
	Path       string                  // Directory path, or file if single-file
	Programs   map[string]*ast.Program // Keys are file basenames (with extensions)
	ModTimes   map[string]time.Time    // Same basenames as Programs
	Checked    *analysis.Module        // Typechecked module
	SingleFile bool
	Stdin      bool
}

func LoadFor(path string) (*Module, error) {
	return nil, nil
}

func HashPath(path string) string {
	hash := fnv.New64a()
	hash.Write([]byte(path))
	return strconv.FormatUint(hash.Sum64(), 36)
}

func Decode(m *Module) {
	_ = gob.NewDecoder(nil)
}
