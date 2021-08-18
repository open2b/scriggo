// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scripts

import (
	"reflect"
	"testing"

	"github.com/open2b/scriggo/internal/compiler"
	"github.com/open2b/scriggo/native"
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

func TestCombinedPackage(t *testing.T) {
	pkg1 := native.MapPackage{"main", map[string]interface{}{"a": 1, "b": 2}}
	pkg2 := native.MapPackage{"main2", map[string]interface{}{"b": 3, "c": 4, "d": 5}}
	pkg := native.CombinedPackage{pkg1, pkg2}
	expected := []string{"a", "b", "c", "d"}
	// Test Name.
	if pkg.Name() != pkg1.Name() {
		t.Fatalf("unexpected name %s, expecting %s", pkg.Name(), pkg1.Name())
	}
	// Test Lookup.
	for _, name := range expected {
		if decl := pkg.Lookup(name); decl == nil {
			t.Fatalf("unexpected nil, expecting declaration of %s", name)
		}
	}
	if decl := pkg.Lookup("notExistent"); decl != nil {
		t.Fatalf("unexpected %#v for non-existent declaration, expecting nil", decl)
	}
	// Test DeclarationNames.
	names := pkg.DeclarationNames()
	if len(names) != len(expected) {
		t.Fatalf("unexpected %d declarations, expecting %d", len(names), len(expected))
	}
	has := map[string]bool{}
	for _, name := range names {
		if has[name] {
			t.Fatalf("unexpected duplicated name %s", name)
		}
		has[name] = true
	}

	for _, name := range expected {
		if !has[name] {
			t.Fatalf("missing name %s from declarations", name)
		}
	}
}
