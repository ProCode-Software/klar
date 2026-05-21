package klon

import (
	"testing"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

func TestParser_DashedBlock(t *testing.T) {
	input := `
object:
  - a: 1
  - b: 2
`
	doc, errs := Parse([]byte(input))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	obj, ok := doc.Body.(*ast.Object)
	if !ok {
		t.Fatalf("expected *ast.Object, got %T", doc.Body)
	}

	if len(obj.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(obj.Fields))
	}

	innerObj, ok := obj.Fields[0].Value.(*ast.Object)
	if !ok {
		t.Fatalf("expected *ast.Object for 'object' value, got %T", obj.Fields[0].Value)
	}

	if len(innerObj.Fields) != 2 {
		t.Fatalf("expected 2 fields in inner object, got %d", len(innerObj.Fields))
	}
}

func TestParser_InlineList(t *testing.T) {
	input := `[4, 1, 6, 7]`
	doc, errs := Parse([]byte(input))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	list, ok := doc.Body.(*ast.List)
	if !ok {
		t.Fatalf("expected *ast.List, got %T", doc.Body)
	}

	if len(list.Items) != 4 {
		t.Fatalf("expected 4 items in list, got %d", len(list.Items))
	}
}
