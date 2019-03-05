// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"scrigo/ast"
	"strings"
	"testing"
)

func dumpTypeInfo(w io.Writer, ti *ast.TypeInfo) {
	_, _ = fmt.Fprint(w, "Type:")
	if ti.Type != nil {
		_, _ = fmt.Fprintf(w, " %s", ti.Type)
	}
	_, _ = fmt.Fprint(w, "\nProperties:")
	if ti.Nil() {
		_, _ = fmt.Fprint(w, " nil")
	}
	if ti.IsType() {
		_, _ = fmt.Fprint(w, " isType")
	}
	if ti.IsType() {
		_, _ = fmt.Fprint(w, " isType")
	}
	if ti.IsBuiltin() {
		_, _ = fmt.Fprint(w, " isBuiltin")
	}
	if ti.Addressable() {
		_, _ = fmt.Fprint(w, " addressable")
	}
	_, _ = fmt.Fprint(w, "\nConstant:")
	if ti.Constant != nil {
		switch dt := ti.Constant.DefaultType; dt {
		case ast.DefaultTypeInt, ast.DefaultTypeRune, ast.DefaultTypeFloat64:
			_, _ = fmt.Fprintf(w, " %s (%s)", ti.Constant.Number.ExactString(), dt)
		case ast.DefaultTypeString:
			_, _ = fmt.Fprintf(w, " %s (%s)", ti.Constant.String, dt)
		case ast.DefaultTypeBool:
			_, _ = fmt.Fprintf(w, " %t (%s)", ti.Constant.Bool, dt)
		}
	}
	_, _ = fmt.Fprint(w, "\nPackage:")
	if ti.Package != nil {
		_, _ = fmt.Fprintf(w, " %s", ti.Package.Name)
	}
	_, _ = fmt.Fprintln(w)

}

var checkerExprs = []struct {
	src   string
	scope typeCheckerScope
}{
	{`"a" == "b"`, nil},
	{`a`, typeCheckerScope{"a": &ast.TypeInfo{Type: reflect.TypeOf(0)}}},
	{`b + 10`, typeCheckerScope{"b": &ast.TypeInfo{Type: reflect.TypeOf(0)}}},
}

func TestCheckerExpressions(t *testing.T) {
	for _, expr := range checkerExprs {
		var lex = newLexer([]byte(expr.src), ast.ContextNone)
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*Error); ok {
						t.Errorf("source: %q, %s\n", expr, err)
					} else {
						panic(r)
					}
				}
			}()
			var p = &parsing{
				lex:       lex,
				ctx:       ast.ContextNone,
				ancestors: nil,
			}
			node, tok := p.parseExpr(token{}, false, false, false, false)
			if node == nil {
				t.Errorf("source: %q, unexpected %s, expecting expression\n", expr, tok)
				return
			}
			var scopes []typeCheckerScope
			if expr.scope != nil {
				scopes = []typeCheckerScope{expr.scope}
			}
			checker := &typechecker{scopes: scopes}
			ti := checker.checkExpression(node)
			dumpTypeInfo(os.Stderr, ti)
		}()
	}
}

// TODO (Gianluca): add blank identifier ("_") support.

const ok = ""

var checkerStmts = map[string]string{

	// Var declarations.
	`var a = 3`:       ok,
	`var a, b = 1, 2`: ok,
	// `var a, b = 1`:    "assignment mismatch: 2 variable but 1 values",

	// Expression errors.
	`v := 1 + "s"`: "mismatched types int and string",
	// `v := 5 + 8.9 + "2"`: `invalid operation: 5 + 8.9 + "2" (mismatched types float64 and string)`,

	// Assignments.
	`v := 1`:                          ok,
	`v = 1`:                           "undefined: v",
	`v := 1 + 2`:                      ok,
	`v := "s" + "s"`:                  ok,
	`v := 1; v = 2`:                   ok,
	`v := 1; v := 2`:                  "no new variables on left side of :=",
	`v := 1 + 2; v = 3 + 4`:           ok,
	`v1 := 0; v2 := 1; v3 := v2 + v1`: ok,
	// `v1 := 1; v2 := "a"; v1 = v2`:     "cannot use v2 (type int) as type string in assignment",

	// Blocks.
	`{ a := 1; a = 10 }`:         ok,
	`{ a := 1; { a = 10 } }`:     ok,
	`{ a := 1; a := 2 }`:         "no new variables on left side of :=",
	`{ { { a := 1; a := 2 } } }`: "no new variables on left side of :=",

	// If statements.
	`if 1 { }`:                     "non-bool 1 (type int) used as if condition",
	`if 1 == 1 { }`:                ok,
	`if 1 == 1 { a := 3 }; a = 1`:  "undefined: a",
	`if a := 1; a == 2 { }`:        ok,
	`if a := 1; a == 2 { b := a }`: ok,
	`if true { }`:                  "",

	// Slices.
	`v := []int{}`:      ok,
	`v := []int{1,2,3}`: ok,
	`v := []int{"a"}`:   `cannot convert "a" (type untyped string) to type int`,

	// Arrays.
	// `v := [1]int{1}`: ok,
	// `v := [1]int{0}`: ok,

	// Maps.
	`v := map[string]string{}`:           ok,
	`v := map[string]string{"k1": "v1"}`: ok,
	`v := map[string]string{2: "v1"}`:    `cannot use 2 (type int) as type string in map key`,
	// `v := map[string]string{"k1": 2}`:    `cannot use 2 (type int) as type string in map value`,

	// Structs.
	`v := pointInt{}`:      ok,
	`v := pointInt{1}`:     `too few values in pointInt literal`,
	`v := pointInt{1,2,3}`: `too many values in pointInt literal`,
	// `v := pointInt{1,2}`:   ok,

	// For statements.
	`for 3 { }`:               "non-bool 3 (type int) used as for condition",
	`for i := 10; i; i++ { }`: "non-bool i (type int) used as for condition",
	// `for i := 0; i < 10; i++ { }`: "",

	// Switch statements.
	`switch 1 { case 1: }`:       ok,
	`switch 1 + 2 { case 3: }`:   ok,
	`switch true { case true: }`: ok,
	// `switch 1 + 2 { case "3": }`: `invalid case "3" in switch on 1 + 2 (mismatched types string and int)`,
}

func TestCheckerStatements(t *testing.T) {
	builtinsScope := typeCheckerScope{
		"true":     &ast.TypeInfo{Type: reflect.TypeOf(false)},
		"false":    &ast.TypeInfo{Type: reflect.TypeOf(false)},
		"int":      &ast.TypeInfo{Properties: ast.PropertyIsType, Type: reflect.TypeOf(0)},
		"string":   &ast.TypeInfo{Properties: ast.PropertyIsType, Type: reflect.TypeOf("")},
		"pointInt": &ast.TypeInfo{Properties: ast.PropertyIsType, Type: reflect.TypeOf(struct{ X, Y int }{})},
	}
	for src, expectedError := range checkerStmts {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*Error); ok {
						if expectedError == "" {
							t.Errorf("source: %q should be 'ok' but got error: %q", src, err)
						} else if !strings.Contains(err.Error(), expectedError) {
							t.Errorf("source: %q should return error: %q but got: %q", src, expectedError, err)
						}
					} else {
						panic(r)
					}
				} else {
					if expectedError != ok {
						t.Errorf("source: %q expecting error: %q, but no errors have been returned by type-checker", src, expectedError)
					}
				}
			}()
			tree, err := ParseSource([]byte(src), ast.ContextNone)
			if err != nil {
				t.Error(err)
			}
			checker := &typechecker{scopes: []typeCheckerScope{builtinsScope, typeCheckerScope{}}}
			checker.checkNodes(tree.Nodes)
		}()
	}
}
