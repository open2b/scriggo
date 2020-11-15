// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"testing"

	"github.com/open2b/scriggo/compiler/ast"
)

var intSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(intType), Properties: propertyAddressable}
var intArrayTypeInfo = &typeInfo{Type: reflect.ArrayOf(2, intType), Properties: propertyAddressable}
var stringSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(stringType), Properties: propertyAddressable}
var stringArrayTypeInfo = &typeInfo{Type: reflect.ArrayOf(2, stringType), Properties: propertyAddressable}
var boolSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(boolType), Properties: propertyAddressable}
var boolArrayTypeInfo = &typeInfo{Type: reflect.ArrayOf(2, boolType), Properties: propertyAddressable}
var interfaceSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(emptyInterfaceType), Properties: propertyAddressable}

var stringToIntMapTypeInfo = &typeInfo{Type: reflect.MapOf(stringType, intType), Properties: propertyAddressable}
var intToStringMapTypeInfo = &typeInfo{Type: reflect.MapOf(intType, stringType), Properties: propertyAddressable}
var definedIntToStringMapTypeInfo = &typeInfo{Type: reflect.MapOf(definedIntTypeInfo.Type, stringType), Properties: propertyAddressable}

var definedIntTypeInfo = &typeInfo{Type: reflect.TypeOf(definedInt(0)), Properties: propertyAddressable}
var definedIntSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(definedIntTypeInfo.Type), Properties: propertyAddressable}

var definedStringTypeInfo = &typeInfo{Type: reflect.TypeOf(definedString("")), Properties: propertyAddressable}

var checkerContainsExprs = []struct {
	src   string
	ti    *typeInfo
	scope map[string]*typeInfo
}{

	// Slice and array.
	{`s contains 5`, tiUntypedBool(), map[string]*typeInfo{"s": intSliceTypeInfo}},
	{`s contains 7.0`, tiUntypedBool(), map[string]*typeInfo{"s": intSliceTypeInfo}},
	{`s contains 'c'`, tiUntypedBool(), map[string]*typeInfo{"s": intSliceTypeInfo}},
	{`s contains int(2)`, tiUntypedBool(), map[string]*typeInfo{"s": intSliceTypeInfo}},
	{`s contains a`, tiUntypedBool(), map[string]*typeInfo{"s": intSliceTypeInfo, "a": tiInt()}},
	{`s contains b`, tiUntypedBool(), map[string]*typeInfo{"s": definedIntSliceTypeInfo, "b": definedIntTypeInfo}},
	{`s contains -2`, tiUntypedBool(), map[string]*typeInfo{"s": intArrayTypeInfo}},
	{`s contains ""`, tiUntypedBool(), map[string]*typeInfo{"s": stringSliceTypeInfo}},
	{`s contains "a"`, tiUntypedBool(), map[string]*typeInfo{"s": stringArrayTypeInfo}},
	{`s contains a`, tiUntypedBool(), map[string]*typeInfo{"s": stringArrayTypeInfo, "a": tiString()}},
	{`s contains b`, tiUntypedBool(), map[string]*typeInfo{"s": stringArrayTypeInfo, "b": tiStringConst("b")}},
	{`s contains true`, tiUntypedBool(), map[string]*typeInfo{"s": boolSliceTypeInfo}},
	{`s contains false`, tiUntypedBool(), map[string]*typeInfo{"s": boolArrayTypeInfo}},
	{`s contains bool(false)`, tiUntypedBool(), map[string]*typeInfo{"s": boolArrayTypeInfo}},
	{`s contains a`, tiUntypedBool(), map[string]*typeInfo{"s": boolArrayTypeInfo, "a": tiBool()}},
	{`s contains nil`, tiUntypedBool(), map[string]*typeInfo{"s": interfaceSliceTypeInfo}},

	// Map
	{`m contains 5`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m contains 7.0`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m contains 'c'`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m contains int(2)`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m contains a`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo, "a": tiInt()}},
	{`m contains b`, tiUntypedBool(), map[string]*typeInfo{"m": definedIntToStringMapTypeInfo, "b": definedIntTypeInfo}},
	{`m contains "a"`, tiUntypedBool(), map[string]*typeInfo{"m": stringToIntMapTypeInfo}},
	{`m contains a`, tiUntypedBool(), map[string]*typeInfo{"m": stringToIntMapTypeInfo, "a": tiString()}},

	// String and string
	{`"ab" contains "a"`, tiUntypedBoolConst(true), nil},
	{`"ab" contains "c"`, tiUntypedBoolConst(false), nil},
	{`ab contains a`, tiUntypedBoolConst(true), map[string]*typeInfo{"ab": tiUntypedStringConst("ab"), "a": tiUntypedStringConst("a")}},
	{`ab contains b`, tiUntypedBoolConst(true), map[string]*typeInfo{"ab": tiStringConst("ab"), "b": tiStringConst("b")}},
	{`ab contains c`, tiUntypedBoolConst(false), map[string]*typeInfo{"ab": tiStringConst("ab"), "c": tiStringConst("c")}},
	{`ab contains d`, tiUntypedBoolConst(false), map[string]*typeInfo{"ab": tiUntypedStringConst("ab"), "d": tiStringConst("d")}},
	{`ab contains e`, tiUntypedBoolConst(false), map[string]*typeInfo{"ab": tiStringConst("ab"), "e": tiUntypedStringConst("e")}},
	{`ab contains f`, tiUntypedBool(), map[string]*typeInfo{"ab": tiString(), "f": tiStringConst("f")}},
	{`ab contains g`, tiUntypedBool(), map[string]*typeInfo{"ab": tiStringConst("ab"), "g": tiString()}},
	{`ab contains h`, tiUntypedBool(), map[string]*typeInfo{"ab": tiString(), "h": tiString()}},
	{`ab contains i`, tiUntypedBool(), map[string]*typeInfo{"ab": tiString(), "i": tiUntypedStringConst("i")}},
	{`ab contains j`, tiUntypedBool(), map[string]*typeInfo{"ab": tiUntypedStringConst("ab"), "j": tiString()}},
	{`ab contains k`, tiUntypedBool(), map[string]*typeInfo{"ab": definedStringTypeInfo, "k": definedStringTypeInfo}},
	{`ab contains l`, tiUntypedBool(), map[string]*typeInfo{"ab": definedStringTypeInfo, "l": tiUntypedStringConst("l")}},
	{`ab contains m`, tiUntypedBool(), map[string]*typeInfo{"ab": tiUntypedStringConst("ab"), "m": definedStringTypeInfo}},

	// String and rune
	{`"àb" contains 'à'`, tiUntypedBoolConst(true), nil},
	{`"àb" contains 224`, tiUntypedBoolConst(true), nil},
	{`"àb" contains 'ù'`, tiUntypedBoolConst(false), nil},
	{`"àb" contains 249`, tiUntypedBoolConst(false), nil},
	{`àb contains 'à'`, tiUntypedBoolConst(true), map[string]*typeInfo{"àb": tiUntypedStringConst("àb")}},
	{`àb contains 'à'`, tiUntypedBoolConst(true), map[string]*typeInfo{"àb": tiStringConst("àb")}},
	{`àb contains 'à'`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString()}},
	{`àb contains à`, tiUntypedBoolConst(true), map[string]*typeInfo{"àb": tiUntypedStringConst("àb"), "à": tiUntypedRuneConst('à')}},
	{`àb contains à`, tiUntypedBoolConst(true), map[string]*typeInfo{"àb": tiUntypedStringConst("àb"), "à": tiRuneConst('à')}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiUntypedStringConst("àb"), "à": tiRune()}},
	{`àb contains à`, tiUntypedBoolConst(true), map[string]*typeInfo{"àb": tiStringConst("àb"), "à": tiUntypedRuneConst('à')}},
	{`àb contains à`, tiUntypedBoolConst(true), map[string]*typeInfo{"àb": tiStringConst("àb"), "à": tiRuneConst('à')}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiStringConst("àb"), "à": tiRune()}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString(), "à": tiUntypedRuneConst('à')}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString(), "à": tiRuneConst('à')}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString(), "à": tiRune()}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString(), "à": tiUntypedIntConst("224")}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString(), "à": tiIntConst(224)}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": tiString(), "à": tiInt()}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": definedStringTypeInfo, "à": tiUntypedRuneConst('à')}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": definedStringTypeInfo, "à": tiRuneConst('à')}},
	{`àb contains à`, tiUntypedBool(), map[string]*typeInfo{"àb": definedStringTypeInfo, "à": tiRune()}},
}

func TestCheckerContainsExpressions(t *testing.T) {
	for _, expr := range checkerContainsExprs {
		var lex = scanTemplate([]byte("{{ "+expr.src+" }}"), ast.LanguageText)
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*CheckingError); ok {
						t.Errorf("source: %q, %s\n", expr.src, err)
					} else {
						panic(r)
					}
				}
			}()
			var p = &parsing{
				lex:            lex,
				language:       ast.LanguageText,
				ancestors:      nil,
				extendedSyntax: true,
			}
			p.next() // discard tokenLeftBraces.
			node, tok := p.parseExpr(p.next(), false, false, false)
			if node == nil {
				t.Errorf("source: %q, unexpected %s, expecting expression\n", expr.src, tok)
				return
			}
			if tok.typ != tokenRightBraces {
				t.Errorf("source: %q, unexpected %s, expecting }}\n", expr.src, tok)
				return
			}
			scope := make(typeCheckerScope, len(expr.scope))
			for k, v := range expr.scope {
				scope[k] = scopeElement{t: v}
			}
			var scopes []typeCheckerScope
			if expr.scope == nil {
				scopes = []typeCheckerScope{}
			} else {
				scopes = []typeCheckerScope{scope}
			}
			tc := newTypechecker(newCompilation(), "", checkerOptions{RelaxedBoolean: true}, nil)
			tc.scopes = scopes
			tc.enterScope()
			ti := tc.checkExpr(node)
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

var checkerContainsExprErrors = []struct {
	src   string
	err   *CheckingError
	scope map[string]*typeInfo
}{
	{`[]byte{} contains "a"`, tierr(1, 13, `invalid operation: []byte literal contains "a" (cannot convert a (type untyped string) to type uint8)`), nil},
	{`[]int{} contains int32(5)`, tierr(1, 12, `invalid operation: []int literal contains int32(5) (mismatched types int and rune)`), nil},
	{`[]int{} contains i`, tierr(1, 12, `invalid operation: []int literal contains i (mismatched types int and compiler.definedInt)`), map[string]*typeInfo{"i": definedIntTypeInfo}},
	{`[2]int{0,1} contains rune('a')`, tierr(1, 16, `invalid operation: [2]int literal contains rune('a') (mismatched types int and rune)`), nil},
}

func TestCheckerContainsExpressionErrors(t *testing.T) {
	for _, expr := range checkerContainsExprErrors {
		var lex = scanTemplate([]byte("{{ "+expr.src+" }}"), ast.LanguageText)
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*CheckingError); ok {
						err := sameTypeCheckError(err, expr.err)
						if err != nil {
							t.Errorf("source: %q, %s\n", expr.src, err)
							return
						}
					} else {
						panic(r)
					}
				}
			}()
			var p = &parsing{
				lex:            lex,
				language:       ast.LanguageText,
				ancestors:      nil,
				extendedSyntax: true,
			}
			p.next() // discard tokenLeftBraces.
			node, tok := p.parseExpr(p.next(), false, false, false)
			if node == nil {
				t.Errorf("source: %q, unexpected %s, expecting expression\n", expr.src, tok)
				return
			}
			if tok.typ != tokenRightBraces {
				t.Errorf("source: %q, unexpected %s, expecting }}\n", expr.src, tok)
				return
			}
			scope := make(typeCheckerScope, len(expr.scope))
			for k, v := range expr.scope {
				scope[k] = scopeElement{t: v}
			}
			var scopes []typeCheckerScope
			if expr.scope == nil {
				scopes = []typeCheckerScope{}
			} else {
				scopes = []typeCheckerScope{scope}
			}
			tc := newTypechecker(newCompilation(), "", checkerOptions{RelaxedBoolean: true}, nil)
			tc.scopes = scopes
			tc.enterScope()
			ti := tc.checkExpr(node)
			t.Errorf("source: %s, unexpected %s, expecting error %q\n", expr.src, ti, expr.err)
		}()
	}
}
