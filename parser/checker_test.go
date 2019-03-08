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

var checkerExprs = []struct {
	src   string
	ti    *ast.TypeInfo
	scope typeCheckerScope
}{
	{`true`, tiUntypedBoolConst(true), nil},
	{`false`, tiUntypedBoolConst(false), nil},
	{`0`, tiUntypedIntConst("0"), nil},
	{`15`, tiUntypedIntConst("15"), nil},
	{`15/3`, tiUntypedIntConst("5"), nil},
	{`15/3.0`, tiUntypedFloatConst("5.0"), nil},
	{`"a" == "b"`, tiUntypedBoolConst(false), nil},
	{`a`, tiAddrInt(), typeCheckerScope{"a": tiAddrInt()}},
	{`b + 10`, tiInt(), typeCheckerScope{"b": tiInt()}},
	{`a`, tiBool(), typeCheckerScope{"a": tiBool()}},
	{`a`, tiAddrBool(), typeCheckerScope{"a": tiAddrBool()}},
	{`a == 1`, tiUntypedBool(), typeCheckerScope{"a": tiInt()}},
	{`a == 1`, tiUntypedBoolConst(true), typeCheckerScope{"a": tiIntConst(1)}},
	{`a == 1`, tiUntypedBoolConst(true), typeCheckerScope{"a": tiUntypedIntConst("1")}},

	// Index.
	{`"a"[0]`, tiByte(), nil},
	{`a[0]`, tiByte(), typeCheckerScope{"a": tiUntypedStringConst("a")}},
	{`a[0]`, tiByte(), typeCheckerScope{"a": tiStringConst("a")}},
	{`a[0]`, tiByte(), typeCheckerScope{"a": tiAddrString()}},
	{`a[0]`, tiByte(), typeCheckerScope{"a": tiString()}},
	{`"a"[0.0]`, tiByte(), nil},
	{`"aa"[1.0]`, tiByte(), nil},
	{`"aaa"[1+1]`, tiByte(), nil},
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
const missingReturn = "missing return at end of function"

// checkerStmts contains some Scrigo snippets with expected type-checker error
// (or empty string if type-checking is valid). Error messages are based upon Go
// 1.12.
var checkerStmts = map[string]string{

	// Var declarations.
	`var a = 3`:               ok,
	`var a, b = 1, 2`:         ok,
	`var a, b = 1`:            "assignment mismatch: 2 variable but 1 values",
	`var a, b, c, d = 1, 2`:   "assignment mismatch: 4 variable but 2 values",
	`var a int = 1`:           ok,
	`var a, b int = 1, "2"`:   `cannot use "2" (type string) as type int in assignment`,
	`var a int = "s"`:         `cannot use "s" (type string) as type int in assignment`,
	`var a int; _ = a`:        ok,
	`var a int; a = 3; _ = a`: ok,

	// Const declarations.
	`const a = 2`:        ok,
	`const a int = 2`:    ok,
	`const a string = 2`: `cannot use 2 (type int) as type string in assignment`, // TODO (Gianluca): Go returns error: cannot convert 2 (type untyped number) to type string

	// Expression errors.
	`v := 1 + "s"`:       "mismatched types int and string",
	`v := 5 + 8.9 + "2"`: `invalid operation: (5 + 8.9) + "2" (mismatched types float64 and string)`, // TODO (Gianluca): should not have parenthesis

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
	`_ = []int{}`:      ok,
	`_ = []int{1,2,3}`: ok,
	`_ = []int{-3: 9}`: `index must be non-negative integer constant`,
	`_ = []int{"a"}`:   `cannot convert "a" (type untyped string) to type int`,
	`_ = [][]string{[]string{"a", "f"}, []string{"g", "h"}}`: ok,
	// `_ = []int{1:10, 1:20}`:                                  `duplicate index in array literal: 1`,
	// `_ = [][]int{[]string{"a", "f"}, []string{"g", "h"}}`:    `cannot use []string literal (type []string) as type []int in array or slice literal`,

	// Arrays.
	`_ = [1]int{1}`:        ok,
	`_ = [5 + 6]int{}`:     ok,
	`_ = [5.0]int{}`:       ok,
	`_ = [0]int{1}`:        `array index 0 out of bounds [0:0]`,
	`_ = [1]int{10:2}`:     `array index 10 out of bounds [0:1]`,
	`a := 4; _ = [a]int{}`: `non-constant array bound a`,
	`_ = [-2]int{}`:        `array bound must be non-negative`,
	// `_ = [3]int{1:10, 1:20}`: `duplicate index in array literal: 1`,
	// `_ = [5.3]int{}`:       `constant 5.3 truncated to integer`,

	// Maps.
	`_ = map[string]string{}`:           ok,
	`_ = map[string]string{"k1": "v1"}`: ok,
	`_ = map[string]string{2: "v1"}`:    `cannot use 2 (type int) as type string in map key`,
	// `_ = map[string]string{"k1": 2}`:    `cannot use 2 (type int) as type string in map value`,

	// Structs.
	`_ = pointInt{}`:           ok,
	`_ = pointInt{1}`:          `too few values in pointInt literal`,
	`_ = pointInt{1,2,3}`:      `too many values in pointInt literal`,
	`_ = pointInt{1,2}`:        ok,
	`_ = pointInt{1.0,2.0}`:    ok,
	`_ = pointInt{X: 1, Y: 2}`: ok,
	`_ = pointInt{X: 1, 2}`:    `mixture of field:value and value initializers`,
	`_ = pointInt{1, Y: 2}`:    `mixture of field:value and value initializers`,
	// `_ = pointInt{1.2,2.0}`: `constant 1.2 truncated to integer`,

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
	`for 3 { }`:                     "non-bool 3 (type int) used as for condition",
	`for i := 10; i; i++ { }`:       "non-bool i (type int) used as for condition",
	`for i := 0; i < 10; i++ { }`:   "",
	`for i := 0; i < 10; {}`:        ok,
	`for i := 0; i < 10; _ = 2 {}`:  ok,
	`for i := 0; i < 10; i = "" {}`: `cannot use "" (type string) as type int in assignment`,

	// Switch statements.
	`switch 1 { case 1: }`:                  ok,
	`switch 1 + 2 { case 3: }`:              ok,
	`switch true { case true: }`:            ok,
	`switch 1 + 2 { case "3": }`:            `invalid case "3" in switch on 1 + 2 (mismatched types string and int)`,
	`a := false; switch a { case true: }`:   ok,
	`a := false; switch a { case 4 > 10: }`: ok,
	`a := false; switch a { case a: }`:      ok,
	`a := 3; switch a { case a: }`:          ok,
	// `a := 3; switch a { case a > 2: }`:      `invalid case a > 2 in switch on a (mismatched types bool and int)`,

	// Functions literal definitions.
	`_ = func(     )         {                                             }`: ok,
	`_ = func(     )         { return                                      }`: ok,
	`_ = func(int  )         {                                             }`: ok,
	`_ = func(     )         { a                                           }`: `undefined: a`,
	`_ = func(     )         { 7 == "hey"                                  }`: `invalid operation: 7 == "hey" (mismatched types int and string)`,
	`_ = func(     )         { if true { }; { a := 10; { _ = a } ; _ = a } }`: ok,
	`_ = func(     )         { if true { }; { a := 10; { _ = b } ; _ = a } }`: `undefined: b`,
	`_ = func(     ) (s int) { s := 0; return 0                            }`: `no new variables on left side of :=`,
	`_ = func(s int)         { s := 0; _ = s                               }`: `no new variables on left side of :=`,

	// Terminating statements - https://golang.org/ref/spec#Terminating_statements
	`_ = func() int { return 1                                          }`: ok,            // (1)
	`_ = func() int { { return 0 }                                      }`: ok,            // (3)
	`_ = func() int { { }                                               }`: missingReturn, // (3) non terminating block
	`_ = func() int { if true { return 1 } else { return 2 }            }`: ok,            // (4)
	`_ = func() int { if true { return 1 } else { }                     }`: missingReturn, // (4) no else
	`_ = func() int { if true { } else { }                              }`: missingReturn, // (4) no then, no else
	`_ = func() int { if true { } else { return 1 }                     }`: missingReturn, // (4) no then
	`_ = func() int { for { }                                           }`: ok,            // (5)
	`_ = func() int { for { break }                                     }`: missingReturn, // (5) has break
	`_ = func() int { for { { break } }                                 }`: missingReturn, // (5) has break
	`_ = func() int { for true { }                                      }`: missingReturn, // (5) has loop condition
	`_ = func() int { for i := 0; i < 10; i++ { }                       }`: missingReturn, // (5) has loop condition
	`_ = func() int { switch { case true: return 0; default: return 0 } }`: ok,            // (6)
	`_ = func() int { switch { case true: fallthrough; default: }       }`: ok,            // (6)
	`_ = func() int { switch { }                                        }`: missingReturn, // (6) no default
	`_ = func() int { switch { case true: return 0; default:  }         }`: missingReturn, // (6) non terminating default
	`_ = func() int { a := 2; _ = a                                     }`: missingReturn,
	`_ = func() int {                                                   }`: missingReturn,

	// Return statements with named result parameters.
	`_ = func() (a int)           { return             }`: ok,
	`_ = func() (a int, b string) { return             }`: ok,
	`_ = func() (a int, b string) { return 0, ""       }`: ok,
	`_ = func() (s int)           { { s := 0; return } }`: `s is shadowed during return`,
	`_ = func() (a int)           { return ""          }`: `cannot use "" (type string) as type int in return argument`,
	`_ = func() (a int, b string) { return "", ""      }`: `cannot use "" (type string) as type int in return argument`,
	`_ = func() (a int)           { return 0, 0        }`: "too many arguments to return\n\thave (int, int)\n\twant (int)",      // TODO (Gianluca): should be "number", not "int"
	`_ = func() (a int, b string) { return 0           }`: "not enough arguments to return\n\thave (int)\n\twant (int, string)", // TODO (Gianluca): should be "number", not "int"

	// Result statements with non-named result parameters.
	`_ = func() int { return 0 }`:              ok,
	`_ = func() int { return "" }`:             `cannot use "" (type string) as type int in return argument`,
	`_ = func() (int, string) { return 0 }`:    "not enough arguments to return\n\thave (int)\n\twant (int, string)",         // TODO (Gianluca): should be "number", not "int"
	`_ = func() (int, int) { return 0, 0, 0}`:  "too many arguments to return\n\thave (int, int, int)\n\twant (int, int)",    // TODO (Gianluca): should be "number", not "int"
	`_ = func() (int, int) { return 0, "", 0}`: "too many arguments to return\n\thave (int, string, int)\n\twant (int, int)", // TODO (Gianluca): should be "number", not "int"

	// Function literal calls.
	`f := func() { }; f()`:     ok,
	`f := func(int) { }; f(0)`: ok,
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
			checker := &typechecker{hasBreak: map[ast.Node]bool{}, scopes: []typeCheckerScope{builtinsScope, typeCheckerScope{}}}
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
	if t1.Value == nil && t2.Value != nil {
		return fmt.Errorf("unexpected non-constant")
	}
	if t1.Value != nil && t2.Value == nil {
		return fmt.Errorf("unexpected constant")
	}
	if t1.Value != nil {
		v1 := t1.Value
		v2 := t2.Value
		if u1, ok := v1.(*ast.UntypedValue); ok {
			u2, ok := v2.(*ast.UntypedValue)
			if !ok {
				return fmt.Errorf("unexpected value %T, expecting untyped value", v2)
			}
			if u1.DefaultType != u2.DefaultType {
				return fmt.Errorf("unexpected default type %s, expecting %s", u2.DefaultType, u1.DefaultType)
			}
			switch u1.DefaultType {
			case reflect.Bool:
				if u1.Bool != u2.Bool {
					return fmt.Errorf("unexpected bool %t, expecting %t", u2.Bool, u1.Bool)
				}
			case reflect.String:
				if u1.String != u2.String {
					return fmt.Errorf("unexpected string %q, expecting %q", u2.String, u1.String)
				}
			default:
				if u1.Number.ExactString() != u2.Number.ExactString() {
					return fmt.Errorf("unexpected number %s, expecting %s", u2.Number.ExactString(), u1.Number.ExactString())
				}
			}
		} else if v1 != v2 {
			if reflect.TypeOf(v1) != reflect.TypeOf(v2) {
				return fmt.Errorf("unexpected value type %T, expecting %T", v2, v1)
			}
			return fmt.Errorf("unexpected value %v, expecting %v", v2, v1)
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
	s += "\n\tValue:"
	if ti.Value != nil {
		if v, ok := ti.Value.(*ast.UntypedValue); ok {
			switch dt := v.DefaultType; dt {
			case reflect.Int, reflect.Int32, reflect.Float64:
				s += fmt.Sprintf(" %s (%s)", v.Number.ExactString(), dt)
			case reflect.String:
				s += fmt.Sprintf(" %s (%s)", v.String, dt)
			case reflect.Bool:
				s += fmt.Sprintf(" %t (%s)", v.Bool, dt)
			}
		} else {
			s += fmt.Sprintf(" %v", ti.Value)
		}
	}
	s += "\n\tPackage:"
	if ti.Package != nil {
		s += " " + ti.Package.Name
	}
	return s
}

// bool type infos.
func tiUntypedBoolConst(b bool) *ast.TypeInfo {
	return &ast.TypeInfo{Value: &ast.UntypedValue{DefaultType: reflect.Bool, Bool: b}}
}

func tiBool() *ast.TypeInfo { return &ast.TypeInfo{Type: boolType} }

func tiAddrBool() *ast.TypeInfo {
	return &ast.TypeInfo{Type: boolType, Properties: ast.PropertyAddressable}
}

func tiBoolConst(b bool) *ast.TypeInfo { return &ast.TypeInfo{Type: boolType, Value: b} }

func tiUntypedBool() *ast.TypeInfo { return &ast.TypeInfo{} }

// float type infos.

func tiUntypedFloatConst(n string) *ast.TypeInfo {
	return &ast.TypeInfo{
		Value: &ast.UntypedValue{
			DefaultType: reflect.Float64,
			Number:      constant.MakeFromLiteral(n, gotoken.FLOAT, 0),
		},
	}
}

func tiFloat32() *ast.TypeInfo { return &ast.TypeInfo{Type: universe["float32"].Type} }

func tiFloat64() *ast.TypeInfo { return &ast.TypeInfo{Type: universe["float64"].Type} }

func tiAddrFloat32() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["float32"].Type, Properties: ast.PropertyAddressable}
}

func tiAddrFloat64() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["float64"].Type, Properties: ast.PropertyAddressable}
}

func tiFloat32Const(n float32) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["float32"].Type, Value: n}
}

func tiFloat64Const(n float64) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["float64"].Type, Value: n}
}

// string type infos.

func tiUntypedStringConst(s string) *ast.TypeInfo {
	return &ast.TypeInfo{
		Value: &ast.UntypedValue{
			DefaultType: reflect.String,
			String:      s,
		},
	}
}

func tiString() *ast.TypeInfo { return &ast.TypeInfo{Type: universe["string"].Type} }

func tiAddrString() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["string"].Type, Properties: ast.PropertyAddressable}
}

func tiStringConst(s string) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["string"].Type, Value: s}
}

// int type infos.

func tiUntypedIntConst(n string) *ast.TypeInfo {
	return &ast.TypeInfo{
		Value: &ast.UntypedValue{
			DefaultType: reflect.Int,
			Number:      constant.MakeFromLiteral(n, gotoken.INT, 0),
		},
	}
}

func tiInt() *ast.TypeInfo    { return &ast.TypeInfo{Type: universe["int"].Type} }
func tiInt64() *ast.TypeInfo  { return &ast.TypeInfo{Type: universe["int64"].Type} }
func tiInt32() *ast.TypeInfo  { return &ast.TypeInfo{Type: universe["int32"].Type} }
func tiInt16() *ast.TypeInfo  { return &ast.TypeInfo{Type: universe["int16"].Type} }
func tiInt8() *ast.TypeInfo   { return &ast.TypeInfo{Type: universe["int8"].Type} }
func tiUint() *ast.TypeInfo   { return &ast.TypeInfo{Type: universe["uint"].Type} }
func tiUint64() *ast.TypeInfo { return &ast.TypeInfo{Type: universe["uint64"].Type} }
func tiUint32() *ast.TypeInfo { return &ast.TypeInfo{Type: universe["uint32"].Type} }
func tiUint16() *ast.TypeInfo { return &ast.TypeInfo{Type: universe["uint16"].Type} }
func tiUint8() *ast.TypeInfo  { return &ast.TypeInfo{Type: universe["uint8"].Type} }

func tiAddrInt() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["int"].Type, Properties: ast.PropertyAddressable}
}
func tiAddrInt64() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["int64"].Type, Properties: ast.PropertyAddressable}
}
func tiAddrInt32() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["int32"].Type, Properties: ast.PropertyAddressable}
}
func tiAddrInt16() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["int16"].Type, Properties: ast.PropertyAddressable}
}
func tiAddrInt8() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["int8"].Type, Properties: ast.PropertyAddressable}
}
func tiAddrUint() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["uint"].Type, Properties: ast.PropertyAddressable}
}
func tiAddrUint64() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["uint64"].Type, Properties: ast.PropertyAddressable}
}
func tiAddrUint32() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["uint32"].Type, Properties: ast.PropertyAddressable}
}
func tiAddrUint16() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["uint16"].Type, Properties: ast.PropertyAddressable}
}
func tiAddrUint8() *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["uint8"].Type, Properties: ast.PropertyAddressable}
}

func tiIntConst(n int) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["int"].Type, Value: n}
}
func tiInt64Const(n int64) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["int64"].Type, Value: n}
}
func tiInt32Const(n int32) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["int32"].Type, Value: n}
}
func tiInt16Const(n int16) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["int16"].Type, Value: n}
}
func tiInt8Const(n int8) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["int8"].Type, Value: n}
}
func tiUintConst(n uint) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["uint"].Type, Value: n}
}
func tiUint64Const(n uint64) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["uint64"].Type, Value: n}
}
func tiUint32Const(n uint32) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["uint32"].Type, Value: n}
}
func tiUint16Const(n uint16) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["uint16"].Type, Value: n}
}
func tiUint8Const(n uint8) *ast.TypeInfo {
	return &ast.TypeInfo{Type: universe["uint8"].Type, Value: n}
}

// nil type info.

func tiNil() *ast.TypeInfo { return &ast.TypeInfo{Properties: ast.PropertyNil} }

// byte type info.

func tiByte() *ast.TypeInfo { return &ast.TypeInfo{Type: universe["byte"].Type} }

func TestTypechecker_MaxIndex(t *testing.T) {
	cases := map[string]int{
		"[]T{}":              noEllipses,
		"[]T{x}":             0,
		"[]T{x, x}":          1,
		"[]T{4:x}":           4,
		"[]T{3:x, x}":        4,
		"[]T{x, x, x, 9: x}": 9,
		"[]T{x, 9: x, x, x}": 11,
	}
	tc := &typechecker{}
	for src, expected := range cases {
		tree, err := ParseSource([]byte(src), ast.ContextNone)
		if err != nil {
			t.Error(err)
		}
		got := tc.maxIndex(tree.Nodes[0].(*ast.CompositeLiteral))
		if got != expected {
			t.Errorf("src '%s': expected: %v, got: %v", src, expected, got)
		}
	}
}

func TestTypechecker_IsAssignableTo(t *testing.T) {
	stringType := universe["string"].Type
	float64Type := universe["float64"].Type
	intSliceType := reflect.TypeOf([]int{})
	emptyInterfaceType := reflect.TypeOf(&[]interface{}{interface{}(nil)}[0]).Elem()
	weirdInterfaceType := reflect.TypeOf(&[]interface{ F() }{interface{ F() }(nil)}[0]).Elem()
	byteType := reflect.TypeOf(byte(0))
	type myInt int
	myIntType := reflect.TypeOf(myInt(0))
	type myIntSlice []int
	myIntSliceType := reflect.TypeOf(myIntSlice(nil))
	type myIntSlice2 []int
	myIntSliceType2 := reflect.TypeOf(myIntSlice2(nil))
	cases := []struct {
		x          *ast.TypeInfo
		T          reflect.Type
		assignable bool
	}{
		// From https://golang.org/ref/spec#Assignability

		// «x's type is identical to T»
		{x: tiInt(), T: intType, assignable: true},
		{x: tiString(), T: stringType, assignable: true},
		{x: tiFloat64(), T: float64Type, assignable: true},
		{x: tiFloat64(), T: stringType, assignable: false},
		{x: &ast.TypeInfo{Type: myIntType}, T: myIntType, assignable: true},

		// «x's type V and T have identical underlying types and at least one of
		// V or T is not a defined type.»
		{x: &ast.TypeInfo{Type: intSliceType}, T: myIntSliceType, assignable: true},     // x is not a defined type, but T is
		{x: &ast.TypeInfo{Type: myIntSliceType}, T: intSliceType, assignable: true},     // x is a defined type, but T is not
		{x: &ast.TypeInfo{Type: myIntSliceType}, T: myIntSliceType2, assignable: false}, // x and T are both defined types

		// «T is an interface type and x implements T.»
		{x: tiInt(), T: emptyInterfaceType, assignable: true},
		{x: tiInt(), T: weirdInterfaceType, assignable: false},
		{x: tiString(), T: emptyInterfaceType, assignable: true},
		{x: tiString(), T: weirdInterfaceType, assignable: false},

		// «x is the predeclared identifier nil and T is a pointer, function,
		// slice, map, channel, or interface type»
		{x: tiNil(), T: intSliceType, assignable: true},
		{x: tiNil(), T: emptyInterfaceType, assignable: true},
		{x: tiNil(), T: weirdInterfaceType, assignable: true},
		{x: tiNil(), T: intType, assignable: false},

		// «x is an untyped constant representable by a value of type T.»
		{x: tiUntypedBoolConst(false), T: boolType, assignable: true},
		{x: tiUntypedIntConst("0"), T: boolType, assignable: false},
		{x: tiUntypedIntConst("0"), T: intType, assignable: true},
		{x: tiUntypedIntConst("10"), T: float64Type, assignable: true},
		{x: tiUntypedIntConst("10"), T: byteType, assignable: true},
		// {x: tiUntypedIntConst("300"), T: byteType, assignable: false},
	}
	tc := &typechecker{}
	for _, c := range cases {
		got := tc.isAssignableTo(c.x, c.T)
		if c.assignable && !got {
			t.Errorf("%s should be assignable to %s, but isAssignableTo returned false", c.x, c.T)
		}
		if !c.assignable && got {
			t.Errorf("%s should not be assignable to %s, but isAssignableTo returned true", c.x, c.T)
		}
	}
}

func TestFunctionUpvalues(t *testing.T) {
	cases := map[string][]string{
		`_ = func() { }`:                              nil,           // no variables.
		`a := 1; _ = func() { }`:                      nil,           // a declared outside but not used.
		`a := 1; _ = func() { _ = a }`:                []string{"a"}, // a declared outside and used.
		`_ = func() { a := 1; _ = a }`:                nil,           // a declared inside and used.
		`a := 1; _ = a; _ = func() { a := 1; _ = a }`: nil,           // a declared both outside and inside, used.

		`a, b := 1, 1; _ = a + b; _ = func() { _ = a + b }`:               []string{"a", "b"},
		`a, b := 1, 1; _ = a + b; _ = func() { b := 1; _ = a + b }`:       []string{"a"},
		`a, b := 1, 1; _ = a + b; _ = func() { a, b := 1, 1; _ = a + b }`: nil,
	}
	for src, expected := range cases {
		tc := &typechecker{scopes: []typeCheckerScope{typeCheckerScope{}}}
		tree, err := ParseSource([]byte(src), ast.ContextNone)
		if err != nil {
			t.Error(err)
		}
		tc.checkNodes(tree.Nodes)
		got := tree.Nodes[len(tree.Nodes)-1].(*ast.Assignment).Values[0].(*ast.Func).Upvalues
		if len(got) != len(expected) {
			t.Errorf("bad upvalues for src: '%s': expected: %s, got: %s", src, expected, got)
			continue
		}
		for i := range got {
			if got[i] != expected[i] {
				t.Errorf("bad upvalues for src: '%s': expected: %s, got: %s", src, expected, got)
			}
		}
	}
}
