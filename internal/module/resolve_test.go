package module_test

import (
	"testing"

	"github.com/ProCode-Software/klar/internal/module"
)

func TestProjectDir(t *testing.T) {
	tests := []struct{ firstPath, expect string }{
		{"/a/b/c/d", "/a/b/c/d"},
		{"/a/b/c/glas.pack", "/a/b/c"},
		{"/dj/sjsk/sjwj/src/file", "/dj/sjsk/sjwj"},
		{"/dj/sjsk/sjwj/src/file.klar", "/dj/sjsk/sjwj"},
	}
	for _, tt := range tests {
		t.Run(tt.firstPath, func(t *testing.T) {
			got, err := module.ProjectDir(tt.firstPath)
			if err != nil {
				t.Errorf("ProjectDir(%#v) failed: %v", tt.firstPath, err)
				return
			}
			if got != tt.expect {
				t.Errorf("ProjectDir(%#v) = %#v, want %#v", tt.firstPath, got, tt.expect)
			}
		})
	}
}
