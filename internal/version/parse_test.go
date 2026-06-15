package version

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		full    bool
		want    *Version
		wantErr bool
	}{
		{name: "valid version", input: "1.2.3", full: true, want: &Version{Parts: []int{1, 2, 3}}},
		{
			name:  "valid version with prefix",
			input: "v1.2.3", full: true,
			want: &Version{Parts: []int{1, 2, 3}},
		},
		{name: "one part", input: "v1", full: true, want: &Version{Parts: []int{1}}},
		{
			name:  "with build",
			input: "v1.0 beta",
			full:  true,
			want:  &Version{Parts: []int{1, 0}, Build: Beta},
		},
		{
			name:  "with build and number",
			input: "v1.0 beta 2",
			full:  true,
			want:  &Version{Parts: []int{1, 0}, Build: Beta, BuildNum: 2},
		},
		{
			name:  "backtrack trailing space if !full",
			input: "v1.0 ",
			full:  false,
			want:  &Version{Parts: []int{1, 0}},
		},
		{
			name:  "backtrack trailing space after build if !full",
			input: "v1.0 beta ",
			full:  false,
			want:  &Version{Parts: []int{1, 0}, Build: Beta},
		},
		{
			name:  "backtrack trailing space after build number if !full",
			input: "v1.0 beta 2 ",
			full:  false,
			want:  &Version{Parts: []int{1, 0}, Build: Beta, BuildNum: 2},
		},

		// Bad inputs
		{name: "trailing characters", input: "v1.2.3_=asdew", full: true, wantErr: true},
		{name: "trailing space", input: "v1.2.3 ", full: true, wantErr: true},
		{name: "trailing dot", input: "v1.2.3.", full: true, wantErr: true},
		{name: "more than 4 parts", input: "v1.2.3.4.5", full: true, wantErr: true},
		{name: "empty", input: "", full: true, wantErr: true},
		{name: "prefix only", input: "v", full: true, wantErr: true},
		{name: "decimal build number", input: "v1.0 beta 2.0", full: true, wantErr: true},
		{name: "trailing space after build", input: "v1.0 beta 2 ", full: true, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := parse(tc.input, tc.full)
			if err != nil {
				if !tc.wantErr {
					t.Errorf("parse() failed: %v", err)
				}
				return
			}
			if tc.wantErr {
				t.Fatalf("expected an error, but got version %q", got.String())
			}
			if tc.want.String() != got.String() {
				t.Errorf("parse() = %v, want %v", got, tc.want)
			}
		})
	}
}
