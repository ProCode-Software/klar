package cache

import (
	"encoding/gob"
	"errors"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

type Module struct {
	Warnings   []*klarerrs.Error       // Cached warnings for the module
	Version    string                  // Klar version + commit hash that created the cache
	Path       string                  // Directory path, or file if single-file
	Programs   map[string]*ast.Program // Keys are file basenames (with extensions)
	ModTimes   map[string]time.Time    // Same basenames as Programs
	Checked    any                     /* *analysis.Module */ // Typechecked module
	SingleFile bool
}

func (m *Module) FilePath(base string) string {
	if m.SingleFile {
		return m.Path
	}
	return filepath.Join(m.Path, base)
}

func nameOf(p string) string { return strings.TrimSuffix(filepath.Base(p), ".klar") }

func HashPath(path string) string {
	hash := fnv.New64a()
	hash.Write([]byte(path))
	return strconv.FormatUint(hash.Sum64(), 36)
}

func dirFor(hashPath string) string {
	// ~/.cache/klar/a1/a1b2c3.../
	return filepath.Join(hashPath[:2], hashPath)
}

// If the cache file doesn't exist, (nil, nil) is returned.
func Load(cacheDir, modulePath string) (*Module, error) {
	hashPath := HashPath(modulePath)
	file, err := os.Open(filepath.Join(cacheDir, dirFor(hashPath), nameOf(modulePath)))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	mod := &Module{}
	if err := gob.NewDecoder(file).Decode(mod); err != nil {
		return nil, err
	}
	// Ensure the version that generated the cache is compatible with the current
	// version of Klar
	if mod.Version != cli.KlarVersionAndCommit {
		return nil, nil
	}
	return mod, nil
}

func Save(cacheDir string, m *Module) error {
	hashPath := HashPath(m.Path)
	path := filepath.Join(cacheDir, dirFor(hashPath), nameOf(m.Path))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	m.Version = cli.KlarVersionAndCommit
	return gob.NewEncoder(file).Encode(m)
}
