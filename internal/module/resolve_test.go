package module

import (
	"os"
	"strings"
	"testing"
)

func tryGetwd(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	cwd += sep
	return cwd
}

func TestPackageRoot(t *testing.T) {
	cwd := tryGetwd(t)
	tests := []struct{ input, wantPkg, wantProj string }{
		{"/foo/bar/pkg", "", "/foo/bar"},
		{"/path/to/dir/src/x", "/path/to/dir", "/path/to/dir"},
		{"/alfa/bravo/", "/alfa/bravo", "/alfa/bravo"},
		{"ax/bx/c/src", cwd + "ax/bx/c", cwd + "ax/bx/c"},
		{
			input:    "one/two/three/pkg/four/src/five.klar",
			wantPkg:  cwd + "one/two/three/pkg/four",
			wantProj: cwd + "one/two/three",
		},
		{"uno/dos/pkg/tres/", cwd + "uno/dos/pkg/tres", cwd + "uno/dos"},
		{"/", "/", "/"},
		// Invalid projects. Not verified
		{"/a/b/c/d/e/pkg/f/pkg/g", "/a/b/c/d/e/pkg/f/pkg/g", "/a/b/c/d/e/pkg/f"},
		{"q/w/e/r/pkg/t/y/src/u", cwd + "q/w/e/r/pkg/t/y", cwd + "q/w/e/r/pkg/t/y"},
	}
	for _, tc := range tests {
		short := func(p string) string {
			if strings.HasPrefix(p, cwd) {
				return "$CWD/" + p[len(cwd):]
			}
			return p
		}
		t.Run("", func(t *testing.T) {
			pkg, proj, err := PackageRoot(tc.input)
			if err != nil {
				t.Errorf("PackageRoot(%#v) failed: %v", tc.input, err)
				return
			}
			if pkg != tc.wantPkg || proj != tc.wantProj {
				t.Errorf("PackageRoot(%#v) = (pkg %#v, proj %#v)\n\twant (%#v, %#v)",
					short(tc.input),
					short(pkg), short(proj),
					short(tc.wantPkg), short(tc.wantProj),
				)
			}
		})
	}
}

func TestIsPackage(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"/foo/bar/pkg", false},
		{"/path/to/dir/src/x", false},
		{"/alpha/bravo", true},
		{"/one/two/three/src/", false},
		{"ax/bx/c/pkg/r/", true},
		{"/", true},
		{"uno/dos/tres/pkg/cuatro", true},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := IsPackage(tt.input)
			if err != nil {
				t.Errorf("IsPackage(%#v) failed: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("IsPackage(%#v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
