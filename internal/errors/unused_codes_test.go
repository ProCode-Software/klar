package errors

// This test checks if all error codes have been used in the project, regadless
// of whether they have messages defined. TODO: implement for non-ParseError

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestUnusedErrorCodes(t *testing.T) {
	root, err := findModuleRoot()
	if err != nil {
		t.Fatalf("findModuleRoot: %v", err)
	}

	errorsDir := filepath.Join(root, "internal", "errors")

	var allCodes []string
	defFileRanges := make(map[string][]excludeRange)

	entries, err := os.ReadDir(errorsDir)
	if err != nil {
		t.Fatalf("reading errors dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(errorsDir, entry.Name())
		codes, err := collectErrorCodes(filePath)
		if err != nil {
			t.Fatalf("collectErrorCodes from %s: %v", entry.Name(), err)
		}
		allCodes = append(allCodes, codes...)

		if len(codes) > 0 {
			ranges, err := getExcludeRanges(filePath)
			if err != nil {
				t.Fatalf("getExcludeRanges from %s: %v", entry.Name(), err)
			}
			defFileRanges[filepath.Clean(filePath)] = ranges
		}
	}

	if len(allCodes) == 0 {
		t.Skip("no error codes found")
	}

	used := make(map[string]bool)
	ignoreDirs := map[string]struct{}{
		".git": {}, "vendor": {}, "node_modules": {}, ".klar": {},
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if _, skip := ignoreDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil || isGenerated(path, content) {
			return nil
		}

		cleanPath := filepath.Clean(path)
		excludeRanges := defFileRanges[cleanPath]

		for _, code := range allCodes {
			if used[code] {
				continue
			}
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(code) + `\b`)
			matches := re.FindAllIndex(content, -1)

			for _, loc := range matches {
				inExcluded := false
				if len(excludeRanges) > 0 {
					for _, r := range excludeRanges {
						if loc[0] >= r.start && loc[1] <= r.end {
							inExcluded = true
							break
						}
					}
				}

				if !inExcluded {
					used[code] = true
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking directory: %v", err)
	}

	var unused []string
	for _, code := range allCodes {
		if !used[code] {
			unused = append(unused, code)
		}
	}

	if len(unused) > 0 {
		var b strings.Builder
		for _, code := range unused {
			b.WriteString("  " + code)
		}
		t.Errorf("%d unused error codes\n%s", len(unused), b.String())
	}
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return os.Getwd()
		}
		dir = parent
	}
}

func collectErrorCodes(filename string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}
	var codes []string
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			for _, id := range vs.Names {
				if strings.HasPrefix(id.Name, "Err") {
					codes = append(codes, id.Name)
				}
			}
		}
	}
	return codes, nil
}

type excludeRange struct {
	start, end int
}

func getExcludeRanges(filename string) ([]excludeRange, error) {
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}

	var ranges []excludeRange

	// Exclude const blocks with error codes
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		// Check if this const block has Err* constants
		hasErrCode := false
		for _, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			for _, id := range vs.Names {
				if strings.HasPrefix(id.Name, "Err") {
					hasErrCode = true
					break
				}
			}
			if hasErrCode {
				break
			}
		}
		if hasErrCode {
			ranges = append(ranges, excludeRange{
				start: fset.Position(gen.Pos()).Offset,
				end:   fset.Position(gen.End()).Offset,
			})
		}
	}

	// Exclude error() method body
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != "error" || fn.Body == nil {
			continue
		}
		if len(fn.Recv.List) == 0 {
			continue
		}
		var recvName string
		switch rt := fn.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			if id, ok := rt.X.(*ast.Ident); ok {
				recvName = id.Name
			}
		case *ast.Ident:
			recvName = rt.Name
		}
		if recvName == "ParseError" {
			ranges = append(ranges, excludeRange{
				start: fset.Position(fn.Body.Pos()).Offset,
				end:   fset.Position(fn.Body.End()).Offset,
			})
		}
	}

	return ranges, nil
}

func isGenerated(path string, content []byte) bool {
	return strings.HasSuffix(path, "_string.go") ||
		strings.Contains(string(content), "Code generated by")
}
