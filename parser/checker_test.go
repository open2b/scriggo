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
	{`"a" == 3`, nil},
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

	// Assignments.
	`v := 1`:                          ok,
	`v = 1`:                           "undefined: v",
	`v := 1 + 2`:                      ok,
	`v := "s" + "s"`:                  ok,
	`v = 1 + "s"`:                     "mismatched types int and string",
	`v := 1; v = 2`:                   ok,
	`v := 1; v := 2`:                  "no new variables on left side of :=",
	`v := 1 + 2; v = 3 + 4`:           ok,
	`v1 := 0; v2 := 1; v3 := v2 + v1`: ok,

	// Blocks.
	`{ a := 1; a = 10 }`:     ok,
	`{ a := 1; { a = 10 } }`: ok,
	`{ a := 1; a := 2 }`:     "no new variables on left side of :=",

	// Switch.
	`switch 1 { case 1: }`:     ok,
	`switch 1 + 2 { case 3: }`: ok,

	// Waiting for some changes in type-checker.

	// `for i := 0; i < 10; i++ { }`: "",
	// `for i := 10; i; i++ { }`: "non-bool i (type int) used as for condition",
	// `if 1 { }`: "non-bool 1 (type int) used as if condition",
	// `switch 1 + 2 { case "3": }`: `invalid case "3" in switch on 1 + 2 (mismatched types string and int)`,
	// `if true { }`: "",
	// `switch true { case true: }`: ok,
	// `v1 := "a";       v2 := 1;      v1 = v2`: false,
}

func TestCheckerStatements(t *testing.T) {
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
			checker := &typechecker{scopes: []typeCheckerScope{typeCheckerScope{}}}
			checker.checkNodes(tree.Nodes)
		}()
	}
}
