// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/ast/astutil"
	"github.com/open2b/scriggo/internal/compiler"
	"github.com/open2b/scriggo/internal/fstest"
)

func TestInitGlobals(t *testing.T) {

	// Test no globals.
	globals := initGlobalVariables([]compiler.Global{}, nil)
	if globals != nil {
		t.Fatalf("expected nil, got %v", globals)
	}

	// Test zero value.
	global := compiler.Global{
		Pkg:  "p",
		Name: "a",
		Type: reflect.TypeOf(0),
	}
	globals = initGlobalVariables([]compiler.Global{global}, nil)
	g := globals[0]
	if g.Kind() != reflect.Int {
		t.Fatalf("unexpected kind %v", g.Kind())
	}
	if g.Interface() != 0 {
		t.Fatalf("unexpected %v, expecting 0", g.Interface())
	}

	// Test pointer value in globals.
	n := 1
	global = compiler.Global{
		Pkg:   "p",
		Name:  "a",
		Type:  reflect.TypeOf(n),
		Value: reflect.ValueOf(&n).Elem(),
	}
	globals = initGlobalVariables([]compiler.Global{global}, nil)
	n = 2
	g = globals[0]
	if g.Kind() != reflect.Int {
		t.Fatalf("unexpected kind %v", g.Kind())
	}
	iface := g.Interface()
	if iface != n {
		t.Fatalf("unexpected %v (type %T), expecting %d (type %T)", iface, iface, n, n)
	}

	// Test pointer value in init.
	global = compiler.Global{
		Pkg:  "main",
		Name: "a",
		Type: reflect.TypeOf(n),
	}
	init := map[string]interface{}{"a": &n}
	globals = initGlobalVariables([]compiler.Global{global}, init)
	if globals == nil {
		t.Fatalf("unexpected %v, expecting nil", globals)
	}
	n = 3
	g = globals[0]
	if g.Kind() != reflect.Int {
		t.Fatalf("unexpected kind %v", g.Kind())
	}
	iface = g.Interface()
	if iface != n {
		t.Fatalf("unexpected %v (type %T), expecting %d (type %T)", iface, iface, n, n)
	}

	// Test non pointer value in init.
	global = compiler.Global{
		Pkg:  "main",
		Name: "a",
		Type: reflect.TypeOf(n),
	}
	init = map[string]interface{}{"a": n}
	globals = initGlobalVariables([]compiler.Global{global}, init)
	if globals == nil {
		t.Fatalf("unexpected %v, expecting nil", globals)
	}
	g = globals[0]
	if g.Kind() != reflect.Int {
		t.Fatalf("unexpected kind %v", g.Kind())
	}
	iface = g.Interface()
	if iface != n {
		t.Fatalf("unexpected %v (type %T), expecting %d (type %T)", iface, iface, n, n)
	}

}

func recoverInitGlobalsPanic(t *testing.T, expected string) {
	got := recover()
	if got == nil {
		t.Fatalf("expecting panic")
	}
	if _, ok := got.(string); !ok {
		panic(got)
	}
	if got.(string) != expected {
		t.Fatalf("unexpected panic %q, expecting panic %q", got, expected)
	}
}

func TestInitGlobalsAlreadyInitializedError(t *testing.T) {
	defer recoverInitGlobalsPanic(t, "variable \"a\" already initialized")
	n := 2
	global := compiler.Global{
		Pkg:   "main",
		Name:  "a",
		Type:  reflect.TypeOf(n),
		Value: reflect.ValueOf(&n).Elem(),
	}
	init := map[string]interface{}{"a": 5}
	_ = initGlobalVariables([]compiler.Global{global}, init)
}

func TestInitGlobalsNilError(t *testing.T) {
	defer recoverInitGlobalsPanic(t, "variable initializer \"a\" cannot be nil")
	global := compiler.Global{
		Pkg:  "main",
		Name: "a",
		Type: reflect.TypeOf(0),
	}
	init := map[string]interface{}{"a": nil}
	_ = initGlobalVariables([]compiler.Global{global}, init)
}

func TestInitGlobalsInvalidTypeError(t *testing.T) {
	defer recoverInitGlobalsPanic(t, "variable initializer \"a\" must have type int or *int, but have bool")
	global := compiler.Global{
		Pkg:  "main",
		Name: "a",
		Type: reflect.TypeOf(0),
	}
	init := map[string]interface{}{"a": true}
	_ = initGlobalVariables([]compiler.Global{global}, init)
}

func TestInitGlobalsNilPointerError(t *testing.T) {
	defer recoverInitGlobalsPanic(t, "variable initializer \"a\" cannot be a nil pointer")
	global := compiler.Global{
		Pkg:  "main",
		Name: "a",
		Type: reflect.TypeOf(0),
	}
	init := map[string]interface{}{"a": (*int)(nil)}
	_ = initGlobalVariables([]compiler.Global{global}, init)
}

type testFormatFS struct {
	fstest.Files
	format Format
}

func (fsys testFormatFS) Format(_ string) (Format, error) {
	return fsys.format, nil
}

// TestFormatFS tests that BuildTemplate gets the file format by calling the
// Format method of FormatFS.
func TestFormatFS(t *testing.T) {
	fsys := testFormatFS{Files: fstest.Files{"index": ""}}
	for _, format := range []Format{FormatText, FormatHTML, FormatCSS, FormatJS, FormatJSON, FormatMarkdown} {
		fsys.format = format
		options := BuildOptions{
			ExpandedTransformer: func(tree *ast.Tree) error {
				if tree.Format != ast.Format(format) {
					return fmt.Errorf("expected format %s, got %s", format, tree.Format)
				}
				return nil
			},
		}
		_, err := BuildTemplate(fsys, "index", &options)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestUnexpandedTransformer ensures the transformer runs before expansion.
func TestUnexpandedTransformer(t *testing.T) {
	fsys := fstest.Files{
		"index.html": `before {{ render "a.html" }} after`,
		"a.html":     "A",
		"b.html":     "B",
	}
	options := BuildOptions{
		UnexpandedTransformer: func(tree *ast.Tree) error {
			if tree.Path != "index.html" {
				return nil
			}
			var render *ast.Render
			astutil.Inspect(tree, func(node ast.Node) bool {
				if render == nil {
					render, _ = node.(*ast.Render)
				}
				return render == nil
			})
			if render == nil {
				return fmt.Errorf("expected render node, got none")
			}
			render.Path = "b.html"
			return nil
		},
	}
	template, err := BuildTemplate(fsys, "index.html", &options)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	var out strings.Builder
	err = template.Run(&out, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	const expected = "before B after"
	if out.String() != expected {
		t.Fatalf("expected %q, got %q", expected, out.String())
	}
}
