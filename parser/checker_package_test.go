package parser

import (
	"fmt"
	"strings"
	"testing"

	"scrigo/ast"
)

var cases = []struct {
	source   []string
	expected Package
}{
	{
		[]string{
			"package main",
		},
		Package{
			Name: "main",
		},
	},
	{
		[]string{
			"package main",
			"var A = 10",
			"var B = 20",
		},
		Package{
			Name: "main",
			Declarations: map[string]*ast.TypeInfo{
				"A": &ast.TypeInfo{Type: intType, Properties: ast.PropertyAddressable},
				"B": &ast.TypeInfo{Type: intType, Properties: ast.PropertyAddressable},
			},
			VariablesEvaluationOrder: []string{"A", "B"},
		},
	},
}

func TestCheckPackage(t *testing.T) {
	for _, c := range cases {
		src := strings.Join(c.source, "\n")
		tree, err := ParseSource([]byte(src), ast.ContextNone)
		if err != nil {
			t.Errorf("source: %q, parser error: %s", src, err)
			continue
		}
		gotPkg, err := CheckPackage(tree.Nodes[0].(*ast.Package))
		if err != nil {
			t.Errorf("source: %q, check package error: %s", src, err)
			continue
		}
		err = equalsPkg(c.expected, gotPkg)
		if err != nil {
			t.Errorf("source: %q, packages are different: %s", src, err)
			continue
		}
	}
}

func equalsPkg(expected, got Package) error {
	if expected.Name != got.Name {
		return fmt.Errorf("expecting name %s, got %s", expected.Name, got.Name)
	}
	if len(expected.Declarations) != len(got.Declarations) {
		return fmt.Errorf("expecting %d declarations, got %d", len(expected.Declarations), len(got.Declarations))
	}
	for ident := range expected.Declarations {
		got, ok := got.Declarations[ident]
		if !ok {
			return fmt.Errorf("expecting a declaration with name %s, but not found", ident)
		}
		err := equalTypeInfo(expected.Declarations[ident], got)
		if err != nil {
			return fmt.Errorf("TypeInfo of %q does not match: %s", ident, err.Error())
		}
	}
	if len(expected.VariablesEvaluationOrder) != len(got.VariablesEvaluationOrder) {
		return fmt.Errorf("expecting %d evaluations orders variables, got %d", len(expected.VariablesEvaluationOrder), len(got.VariablesEvaluationOrder))
	}
	for i := range expected.VariablesEvaluationOrder {
		if expected.VariablesEvaluationOrder[i] != got.VariablesEvaluationOrder[i] {
			return fmt.Errorf("expecting variable %s (in evaluation order), got %s", expected.VariablesEvaluationOrder[i], got.VariablesEvaluationOrder)
		}
	}
	return nil
}
