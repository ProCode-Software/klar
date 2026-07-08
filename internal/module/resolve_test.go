package module

import (
	"path/filepath"
	"testing"
)

func TestPackageRoot(t *testing.T) {
	tests := []struct{ input, wantPkg, wantProj string }{
		{"/foo/bar/pkg", "", "/foo/bar"},
		{"/path/to/dir/src/x", "/path/to/dir", "/path/to/dir"},
		{"/alfa/bravo/", "/alfa/bravo", "/alfa/bravo"},
		{"ax/bx/c/src", "ax/bx/c", "ax/bx/c"},
		{
			input:    "one/two/three/pkg/four/src/five.klar",
			wantPkg:  "one/two/three/pkg/four",
			wantProj: "one/two/three",
		},
		{"uno/dos/pkg/tres/", "uno/dos/pkg/tres", "uno/dos"},
		{"/", "/", "/"},
		// Invalid projects. Not verified
		{"/a/b/c/d/e/pkg/f/pkg/g", "/a/b/c/d/e/pkg/f/pkg/g", "/a/b/c/d/e/pkg/f"},
		{"q/w/e/r/pkg/t/y/src/u", "q/w/e/r/pkg/t/y", "q/w/e/r/pkg/t/y"},
	}
	for _, tc := range tests {
		tc.wantPkg, tc.wantProj = filepath.Clean(tc.wantPkg), filepath.Clean(tc.wantProj)
		t.Run("", func(t *testing.T) {
			pkg, proj := PackageRoot(tc.input)
			if filepath.Clean(pkg) != tc.wantPkg || filepath.Clean(proj) != tc.wantProj {
				t.Errorf(
					"PackageRoot(%#v) = (pkg %#v, proj %#v)\n\twant (%#v, %#v)",
					tc.input, pkg, proj, tc.wantPkg, tc.wantProj,
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
			if got := IsPackage(tt.input); got != tt.want {
				t.Errorf("IsPackage(%#v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
