package lsp

import (
	"bytes"
	"io/fs"
	"os"
	"strings"
	"time"

	klarast "github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/pkg/klon"
	klonast "github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/lsp"
)

type FileSystem struct {
	// Files don't contain URI schemes
	Files map[string]*File
}

type File struct {
	Content  []byte
	Modified time.Time
	// Nil if the file is all-ASCII or the LSP is using UTF-32
	PosMapper *PositionMapper
	// Based on language ID. Exactly 1 is set
	Klar *KlarFile
	Klon *KlonFile
}

type KlarFile struct {
	AST         *klarast.Program
	Module      *Module
	ModulePath  string            // TODO: Not needed if Module != nil
	Diagnostics []*klarerrs.Error // Errors and warnings
}

type KlonFile struct {
	AST         *klonast.Document
	Diagnostics []*klon.Error // Errors and warnings
}

func (fs *FileSystem) WriteFile(path string, b []byte) {
	file := fs.Files[path]
	if file == nil {
		file = &File{}
		fs.Files[path] = file
	}
	file.Content = b
	file.Modified = time.Now() // May not be exact
	// TODO: I'm not currently seeing encoding issues without this
	// If this has to be reenabled, calculate concurrently
	// file.makePositionMap()
}

func (fs *FileSystem) DeleteFromMemory(path string) {
	if f, ok := fs.Files[path]; ok {
		f.Content = nil
	}
}

func (f *File) SetLanguage(langID lsp.LanguageKind) {
	// Note that the language may be manually changed by the user in the editor
	switch langID {
	case lsp.LanguageKlar:
		if f.Klar == nil {
			f.Klar = &KlarFile{}
		}
		f.Klon = nil
	case lsp.LanguageKlon:
		if f.Klon == nil {
			f.Klon = &KlonFile{}
		}
		f.Klar = nil
	default: // Language isn't supposed to be handled by KlarLS. Includes glaslock
	}
}

func (f *File) IsKlar() bool      { return f.Klar != nil }

func StripScheme(uri lsp.DocumentURI) string {
	path := string(uri)
	path, ok := strings.CutPrefix(path, "file://")
	if ok {
		return path
	}
	return strings.TrimPrefix(path, "untitled:")
}

// [build.CompilerFS] implementation
// ========

var _ build.CompilerFS = &FileSystem{}

type stattableFile struct {
	*File
	path string
	*bytes.Reader
}

func (fsys *FileSystem) Open(path string) (fs.File, error) {
	file, ok := fsys.Files[path]
	if !ok {
		return os.Open(path)
	}
	return &stattableFile{
		File:   file,
		path:   path,
		Reader: bytes.NewReader(file.Content),
	}, nil
}

func (fsys *FileSystem) ReadDir(dirname string) ([]fs.DirEntry, error) {
	// TODO: The returned DirEntrys may not have the latest Info()
	return os.ReadDir(dirname)
}

func (fsys *FileSystem) Stat(path string) (fs.FileInfo, error) {
	if cachedFile, ok := fsys.Files[path]; ok {
		// Reader not needed
		sf := &stattableFile{File: cachedFile, path: path}
		return sf.Stat()
	}
	return os.Stat(path)
}
func (file *stattableFile) Close() error { return nil }
func (file *stattableFile) Stat() (fs.FileInfo, error) {
	return fileInfo{
		name:    file.path,
		size:    int64(len(file.Content)),
		modTime: file.Modified,
		mode: func() fs.FileMode {
			if localStat, err := os.Stat(file.path); err == nil {
				return localStat.Mode()
			}
			return 0
		},
	}, nil
}

type fileInfo struct {
	name    string
	size    int64
	modTime time.Time
	mode    func() fs.FileMode
}

func (info fileInfo) Name() string       { return info.name }
func (info fileInfo) Size() int64        { return info.size }
func (info fileInfo) Mode() fs.FileMode  { return info.mode() }
func (info fileInfo) ModTime() time.Time { return info.modTime }
func (info fileInfo) IsDir() bool        { return false }
func (info fileInfo) Sys() any           { return nil }
