// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"
	"go/constant"
	gotoken "go/token"
	"reflect"
	"strings"
	"testing"

	"scrigo/ast"
)

func tiBool() *ast.TypeInfo { return &ast.TypeInfo{Type: boolType} }
func tiAddrBool() *ast.TypeInfo {
	return &ast.TypeInfo{Type: boolType, Properties: ast.PropertyAddressable}
}
func tiBoolConst(b bool) *ast.TypeInfo {
	return &ast.TypeInfo{Type: boolType, Constant: &ast.Constant{DefaultType: ast.DefaultTypeBool, Bool: b}}
}
func tiUntypedBool() *ast.TypeInfo {
	return &ast.TypeInfo{}
}
func tiUntypedBoolConst(b bool) *ast.TypeInfo {
	return &ast.TypeInfo{Constant: &ast.Constant{DefaultType: ast.DefaultTypeBool, Bool: b}}
}

func tiInt() *ast.TypeInfo { return &ast.TypeInfo{Type: intType} }
func tiAddrInt() *ast.TypeInfo {
	return &ast.TypeInfo{Type: intType, Properties: ast.PropertyAddressable}
}
func tiIntConst(n string) *ast.TypeInfo {
	return &ast.TypeInfo{
		Type: intType,
		Constant: &ast.Constant{
			DefaultType: ast.DefaultTypeInt,
			Number:      constant.MakeFromLiteral(n, gotoken.INT, 0),
		},
	}
}
func tiUntypedIntConst(n string) *ast.TypeInfo {
	return &ast.TypeInfo{
		Constant: &ast.Constant{
			DefaultType: ast.DefaultTypeInt,
			Number:      constant.MakeFromLiteral(n, gotoken.INT, 0),
		},
	}
}

var checkerExprs = []struct {
	src   string
	ti    *ast.TypeInfo
	scope typeCheckerScope
}{
	{`"a" == "b"`, tiUntypedBoolConst(false), nil},
	{`a`, tiAddrInt(), typeCheckerScope{"a": tiAddrInt()}},
	{`b + 10`, tiInt(), typeCheckerScope{"b": tiInt()}},
	{`a`, tiBool(), typeCheckerScope{"a": tiBool()}},
	{`a`, tiAddrBool(), typeCheckerScope{"a": tiAddrBool()}},
	{`a == 1`, tiUntypedBool(), typeCheckerScope{"a": tiInt()}},
	{`a == 1`, tiUntypedBoolConst(true), typeCheckerScope{"a": tiIntConst("1")}},
	{`a == 1`, tiUntypedBoolConst(true), typeCheckerScope{"a": tiUntypedIntConst("1")}},
}

func TestCheckerExpressions(t *testing.T) {
	for _, expr := range checkerExprs {
		var lex = newLexer([]byte(expr.src), ast.ContextNone)
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*Error); ok {
						t.Errorf("source: %q, %s\n", expr.src, err)
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
				t.Errorf("source: %q, unexpected %s, expecting expression\n", expr.src, tok)
				return
			}
			var scopes []typeCheckerScope
			if expr.scope == nil {
				scopes = []typeCheckerScope{universe}
			} else {
				scopes = []typeCheckerScope{universe, expr.scope}
			}
			checker := &typechecker{scopes: scopes}
			ti := checker.checkExpression(node)
			err := equalTypeInfo(expr.ti, ti)
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				if testing.Verbose() {
					t.Logf("\nUnexpected:\n%s\nExpected:\n%s\n", dumpTypeInfo(ti), dumpTypeInfo(expr.ti))
				}
			}
		}()
	}
}

// TODO (Gianluca): add blank identifier ("_") support.

const ok = ""

// checkerStmts contains some Scrigo snippets with expected type-checker error
// (or empty string if type-checking is valid). Error messages are based upon Go
// 1.12.
var checkerStmts = map[string]string{

	// Var declarations.
	`var a = 3`:             ok,
	`var a, b = 1, 2`:       ok,
	`var a, b = 1`:          "assignment mismatch: 2 variable but 1 values",
	`var a, b, c, d = 1, 2`: "assignment mismatch: 4 variable but 2 values",
	`var a int = 1`:         ok,
	`var a, b int = 1, "2"`: `cannot use "2" (type string) as type int in assignment`,
	`var a int = "s"`:       `cannot use "s" (type string) as type int in assignment`,
	// `var a int; _ = a`:        ok,
	// `var a int; a = 3; _ = a`: ok,

	// Const declarations.
	// `const a = 2`:     ok,
	// `const a int = 2`: ok,

	// Expression errors.
	`v := 1 + "s"`: "mismatched types int and string",
	// `v := 5 + 8.9 + "2"`: `invalid operation: 5 + 8.9 + "2" (mismatched types float64 and string)`,

	// Assignments.
	`_ = 1`:                           ok,
	`v := 1`:                          ok,
	`v = 1`:                           "undefined: v",
	`v := 1 + 2`:                      ok,
	`v := "s" + "s"`:                  ok,
	`v := 1; v = 2`:                   ok,
	`v := 1; v := 2`:                  "no new variables on left side of :=",
	`v := 1 + 2; v = 3 + 4`:           ok,
	`v1 := 0; v2 := 1; v3 := v2 + v1`: ok,
	`v1 := 1; v2 := "a"; v1 = v2`:     `cannot use v2 (type string) as type int in assignment`,

	// Increments and decrements.
	`a := 1; a++`:   ok,
	`a := ""; a++`:  `invalid operation: a++ (non-numeric type string)`,
	`b++`:           `undefined: b`,
	`a := 5.0; a--`: ok,
	`a := ""; a--`:  `invalid operation: a-- (non-numeric type string)`,
	`b--`:           `undefined: b`,

	// "Compact" assignments.
	`a := 1; a += 1`: ok,
	`a := 1; a *= 2`: ok,
	// `a := ""; a /= 6`: `invalid operation: a /= 6 (mismatched types string and int)`,

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
							t.Errorf("source: '%s' should be 'ok' but got error: %q", src, err)
						} else if !strings.Contains(err.Error(), expectedError) {
							t.Errorf("source: '%s' should return error: %q but got: %q", src, expectedError, err)
						}
					} else {
						panic(r)
					}
				} else {
					if expectedError != ok {
						t.Errorf("source: '%s' expecting error: %q, but no errors have been returned by type-checker", src, expectedError)
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

// tiEquals checks that t1 and t2 are identical.
func equalTypeInfo(t1, t2 *ast.TypeInfo) error {
	if t1.Type == nil && t2.Type != nil {
		return fmt.Errorf("unexpected type %s, expecting untyped", t2.Type)
	}
	if t1.Type != nil && t2.Type == nil {
		return fmt.Errorf("unexpected untyped, expecting type %s", t1.Type)
	}
	if t1.Type != nil && t1.Type != t2.Type {
		return fmt.Errorf("unexpected type %s, expecting %s", t2.Type, t1.Type)
	}
	if t1.Nil() && !t2.Nil() {
		return fmt.Errorf("unexpected non-predeclared nil")
	}
	if !t1.Nil() && t2.Nil() {
		return fmt.Errorf("unexpected predeclared nil")
	}
	if t1.IsType() && !t2.IsType() {
		return fmt.Errorf("unexpected non-type")
	}
	if !t1.IsType() && t2.IsType() {
		return fmt.Errorf("unexpected type")
	}
	if t1.IsBuiltin() && !t2.IsBuiltin() {
		return fmt.Errorf("unexpected non-builtin")
	}
	if !t1.IsBuiltin() && t2.IsBuiltin() {
		return fmt.Errorf("unexpected builtin")
	}
	if t1.Addressable() && !t2.Addressable() {
		return fmt.Errorf("unexpected not addressable")
	}
	if !t1.Addressable() && t2.Addressable() {
		return fmt.Errorf("unexpected addressable")
	}
	if t1.Constant == nil && t2.Constant != nil {
		return fmt.Errorf("unexpected non-constant")
	}
	if t1.Constant != nil && t2.Constant == nil {
		return fmt.Errorf("unexpected constant")
	}
	if t1.Constant != nil {
		c1 := t1.Constant
		c2 := t2.Constant
		if c1.DefaultType != c2.DefaultType {
			return fmt.Errorf("unexpected default type %s, expecting %s", c2.DefaultType, c1.DefaultType)
		}
		switch c1.DefaultType {
		case ast.DefaultTypeBool:
			if c1.Bool != c2.Bool {
				return fmt.Errorf("unexpected bool %t, expecting %t", c2.Bool, c1.Bool)
			}
		case ast.DefaultTypeString:
			if c1.String != c2.String {
				return fmt.Errorf("unexpected string %q, expecting %q", c2.String, c1.String)
			}
		default:
			if c1.Number.ExactString() != c2.Number.ExactString() {
				return fmt.Errorf("unexpected number %s, expecting %s", c2.Number.ExactString(), c1.Number.ExactString())
			}
		}
	}
	if t1.Package != nil && t2.Package == nil {
		return fmt.Errorf("unexpected package")
	}
	if t1.Package == nil && t2.Package != nil {
		return fmt.Errorf("unexpected non-package, expecting a package")
	}
	if t1.Package != nil && t1.Package != t2.Package {
		return fmt.Errorf("unexpected package %s, expecting %s", t2.Package.Name, t1.Package.Name)
	}
	return nil
}

func dumpTypeInfo(ti *ast.TypeInfo) string {
	s := "\tType:"
	if ti.Type != nil {
		s += " " + ti.Type.String()
	}
	s += "\n\tProperties:"
	if ti.Nil() {
		s += " nil"
	}
	if ti.IsType() {
		s += " isType"
	}
	if ti.IsBuiltin() {
		s += " isBuiltin"
	}
	if ti.Addressable() {
		s += " addressable"
	}
	s += "\n\tConstant:"
	if ti.Constant != nil {
		switch dt := ti.Constant.DefaultType; dt {
		case ast.DefaultTypeInt, ast.DefaultTypeRune, ast.DefaultTypeFloat64:
			s += fmt.Sprintf(" %s (%s)", ti.Constant.Number.ExactString(), dt)
		case ast.DefaultTypeString:
			s += fmt.Sprintf(" %s (%s)", ti.Constant.String, dt)
		case ast.DefaultTypeBool:
			s += fmt.Sprintf(" %t (%s)", ti.Constant.Bool, dt)
		}
	}
	s += "\n\tPackage:"
	if ti.Package != nil {
		s += " " + ti.Package.Name
	}
	return s
}
