package build

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"time"
)

type CompilerFS interface {
	fs.FS
	fs.StatFS
	fs.ReadDirFS
}

var SystemFS = os.DirFS("/").(CompilerFS)

var _ CompilerFS = (*VirtualFS)(nil)

type VirtualFS struct {
	Files map[string]*VirtualFile
}

type VirtualFile struct {
	FileName string
	Dir      bool
	Entries  []*VirtualFS
	Bytes    []byte
	Reader   io.Reader
}

func (vfs *VirtualFS) Open(name string) (fs.File, error) {
	if file, ok := vfs.Files[name]; ok {
		return file, nil
	}
	return nil, os.ErrNotExist
}

func (vfs *VirtualFS) Stat(name string) (fs.FileInfo, error) {
	if file, ok := vfs.Files[name]; ok {
		return file, nil
	}
	return nil, os.ErrNotExist
}

func (vfs *VirtualFS) ReadDir(name string) ([]fs.DirEntry, error) {
}

func (vf *VirtualFile) Close() error               { return nil }
func (vf *VirtualFile) Stat() (fs.FileInfo, error) { return vf, nil }
func (vf *VirtualFile) Name() string               { return vf.FileName }
func (vf *VirtualFile) IsDir() bool                { return vf.Dir }
func (vf *VirtualFile) ModTime() (t time.Time)     { return }
func (vf *VirtualFile) Size() int64                { return int64(len(vf.Bytes)) }
func (vf *VirtualFile) Sys() any                   { return nil }
func (vf *VirtualFile) Mode() fs.FileMode {
	if vf.Dir {
		return fs.ModeDir
	}
	return 0
}

func (vf *VirtualFile) Read(p []byte) (int, error) {
	if vf.Dir {
		return 0, nil
	}
	if vf.Bytes != nil && vf.Reader == nil {
		vf.Reader = bytes.NewReader(vf.Bytes)
	}
	return vf.Reader.Read(p)
}

func (vf *VirtualFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if vf.Dir {
		return nil, nil
	}
	return nil, nil
}
