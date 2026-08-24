package lsp

import (
	"bytes"
	"io/fs"
	"os"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
)

type FileSystem struct {
	Files map[string]*File
}

type File struct {
	Content    []byte
	Modified   time.Time
	Module     *Module
	ModulePath string // TODO: Not needed if Module != nil
}

func (fs *FileSystem) WriteFile(path string, b []byte) {
	file := fs.Files[path]
	if file == nil {
		file = &File{}
		fs.Files[path] = file
	}
	file.Content = b
	file.Modified = time.Now() // May not be exact
}

func (fs *FileSystem) DeleteFromMemory(path string) {
	if f, ok := fs.Files[path]; ok {
		f.Content = nil
	}
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
