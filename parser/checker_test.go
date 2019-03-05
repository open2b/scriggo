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

func TestChecker(t *testing.T) {
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
