// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/fstest"
	"github.com/open2b/scriggo/native"
)

func init() {
	jsStringerType = reflect.TypeOf((*native.JSStringer)(nil)).Elem()
	jsEnvStringerType = reflect.TypeOf((*native.JSEnvStringer)(nil)).Elem()
	jsonStringerType = reflect.TypeOf((*native.JSONStringer)(nil)).Elem()
	jsonEnvStringerType = reflect.TypeOf((*native.JSONEnvStringer)(nil)).Elem()
}

type html string
type css string
type js string
type json string
type markdown string

var formatTypes = map[ast.Format]reflect.Type{
	ast.FormatHTML:     reflect.TypeOf((*html)(nil)).Elem(),
	ast.FormatCSS:      reflect.TypeOf((*css)(nil)).Elem(),
	ast.FormatJS:       reflect.TypeOf((*js)(nil)).Elem(),
	ast.FormatJSON:     reflect.TypeOf((*json)(nil)).Elem(),
	ast.FormatMarkdown: reflect.TypeOf((*markdown)(nil)).Elem(),
}

var intSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(intType), Properties: propertyAddressable}
var intArrayTypeInfo = &typeInfo{Type: reflect.ArrayOf(2, intType), Properties: propertyAddressable}
var stringSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(stringType), Properties: propertyAddressable}
var stringArrayTypeInfo = &typeInfo{Type: reflect.ArrayOf(2, stringType), Properties: propertyAddressable}
var boolSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(boolType), Properties: propertyAddressable}
var boolArrayTypeInfo = &typeInfo{Type: reflect.ArrayOf(2, boolType), Properties: propertyAddressable}
var interfaceSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(emptyInterfaceType), Properties: propertyAddressable}
var stringToAnyMapTypeInfo = &typeInfo{Type: reflect.MapOf(stringType, emptyInterfaceType), Properties: propertyAddressable}

var stringToIntMapTypeInfo = &typeInfo{Type: reflect.MapOf(stringType, intType), Properties: propertyAddressable}
var intToStringMapTypeInfo = &typeInfo{Type: reflect.MapOf(intType, stringType), Properties: propertyAddressable}
var intToAnyMapTypeInfo = &typeInfo{Type: reflect.MapOf(intType, emptyInterfaceType), Properties: propertyAddressable}
var anyToIntMapTypeInfo = &typeInfo{Type: reflect.MapOf(emptyInterfaceType, intType), Properties: propertyAddressable}
var definedStringToStringMapTypeInfo = &typeInfo{Type: reflect.MapOf(definedStringTypeInfo.Type, stringType), Properties: propertyAddressable}
var definedIntToStringMapTypeInfo = &typeInfo{Type: reflect.MapOf(definedIntTypeInfo.Type, stringType), Properties: propertyAddressable}

var stringToStringToIntMapTypeInfo = &typeInfo{Type: reflect.MapOf(stringType, reflect.MapOf(stringType, intType)), Properties: propertyAddressable}

var definedIntTypeInfo = &typeInfo{Type: reflect.TypeOf(definedInt(0)), Properties: propertyAddressable}
var definedIntSliceTypeInfo = &typeInfo{Type: reflect.SliceOf(definedIntTypeInfo.Type), Properties: propertyAddressable}

var definedStringTypeInfo = &typeInfo{Type: reflect.TypeOf(definedString("")), Properties: propertyAddressable}

func tiHTMLConst(s string) *typeInfo {
	return &typeInfo{Type: formatTypes[ast.FormatHTML], Constant: stringConst(s)}
}

func tiHTML() *typeInfo {
	return &typeInfo{Type: formatTypes[ast.FormatHTML]}
}

func tiMarkdownConst(s string) *typeInfo {
	return &typeInfo{Type: formatTypes[ast.FormatMarkdown], Constant: stringConst(s)}
}

func tiMarkdown() *typeInfo {
	return &typeInfo{Type: formatTypes[ast.FormatMarkdown]}
}

type TF struct {
	F int
}

var stringToTFMapTypeInfo = &typeInfo{Type: reflect.MapOf(stringType, reflect.TypeOf(TF{})), Properties: propertyAddressable}

var checkerTemplateExprs = []struct {
	src   string
	ti    *typeInfo
	scope map[string]*typeInfo
}{

	// contains ( slice and array )
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

	// contains ( map )
	{`m contains 5`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m contains 7.0`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m contains 'c'`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m contains int(2)`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m contains a`, tiUntypedBool(), map[string]*typeInfo{"m": intToStringMapTypeInfo, "a": tiInt()}},
	{`m contains b`, tiUntypedBool(), map[string]*typeInfo{"m": definedIntToStringMapTypeInfo, "b": definedIntTypeInfo}},
	{`m contains "a"`, tiUntypedBool(), map[string]*typeInfo{"m": stringToIntMapTypeInfo}},
	{`m contains a`, tiUntypedBool(), map[string]*typeInfo{"m": stringToIntMapTypeInfo, "a": tiString()}},

	// contains ( string and string )
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

	// contains ( string and rune )
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

	// macro type literal
	{`(macro() string)(nil)`, &typeInfo{Type: reflect.TypeOf((func() string)(nil))}, nil},
	{`(macro() html)(nil)`, &typeInfo{Type: reflect.TypeOf((func() html)(nil))}, nil},
	{`(macro() css)(nil)`, &typeInfo{Type: reflect.TypeOf((func() css)(nil))}, nil},
	{`(macro() js)(nil)`, &typeInfo{Type: reflect.TypeOf((func() js)(nil))}, nil},
	{`(macro() json)(nil)`, &typeInfo{Type: reflect.TypeOf((func() json)(nil))}, nil},
	{`(macro() markdown)(nil)`, &typeInfo{Type: reflect.TypeOf((func() markdown)(nil))}, nil},

	// conversion from markdown to html
	{`html(a)`, tiHTMLConst("<h1>title</h1>"), map[string]*typeInfo{"a": tiMarkdownConst("# title")}},
	{`html(a)`, tiHTML(), map[string]*typeInfo{"a": tiMarkdown()}},

	// key selector
	{`m.x`, tiInt(), map[string]*typeInfo{"m": stringToIntMapTypeInfo}},
	{`m.x`, tiString(), map[string]*typeInfo{"m": definedStringToStringMapTypeInfo}},
	{`m.x`, tiInt(), map[string]*typeInfo{"m": anyToIntMapTypeInfo}},
	{`m.x.y`, tiInt(), map[string]*typeInfo{"m": stringToStringToIntMapTypeInfo}},
	{`m.x.y`, tiInterface(), map[string]*typeInfo{"m": stringToAnyMapTypeInfo}},
	{`m.x.y.z`, tiInterface(), map[string]*typeInfo{"m": stringToAnyMapTypeInfo}},
	{`m.x.nil`, tiInterface(), map[string]*typeInfo{"m": stringToAnyMapTypeInfo}},
	{`m.x.F`, tiInt(), map[string]*typeInfo{"m": stringToTFMapTypeInfo}},
}

func TestCheckerTemplateExpressions(t *testing.T) {
	mdConverter := func(src []byte, out io.Writer) error {
		if string(src) != "# title" {
			panic(fmt.Sprintf("unexpected markdown string %q", string(src)))
		}
		_, err := io.WriteString(out, "<h1>title</h1>")
		return err
	}
	options := checkerOptions{mod: templateMod, formatTypes: formatTypes, mdConverter: mdConverter}
	for _, expr := range checkerTemplateExprs {
		var lex = scanTemplate([]byte("{{ "+expr.src+" }}"), ast.FormatText, false)
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
				lex:       lex,
				ancestors: nil,
			}
			p.next() // discard tokenLeftBraces.
			node, tok := p.parseExpr(p.next(), false, false, false, false)
			if node == nil {
				t.Errorf("source: %q, unexpected %s, expecting expression\n", expr.src, tok)
				return
			}
			if tok.typ != tokenRightBraces {
				t.Errorf("source: %q, unexpected %s, expecting }}\n", expr.src, tok)
				return
			}
			compilation := newCompilation(nil)
			tc := newTypechecker(compilation, "", options, nil)
			for name, ti := range expr.scope {
				tc.scopes.Declare(name, ti, nil, nil)
			}
			tc.scopes.Enter(node)
			ti := tc.checkExpr(node)
			tc.scopes.Exit()
			err := equalTypeInfo(expr.ti, ti)
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				if testing.Verbose() {
					t.Logf("\nUnexpected:\n%s\nExpected:\n%s\n", dumpTypeInfo(ti), dumpTypeInfo(expr.ti))
				}
			}
			err = compilation.finalizeUsingStatements(tc)
			if err != nil {
				t.Fatal(err)
			}
		}()
	}
}

var checkerTemplateExprErrors = []struct {
	src   string
	err   *CheckingError
	scope map[string]*typeInfo
}{

	// contains
	{`[]byte{} contains "a"`, tierr(1, 13, `invalid operation: []byte{} contains "a" (cannot convert "a" (type untyped string) to type uint8)`), nil},
	{`[]int{} contains int32(5)`, tierr(1, 12, `invalid operation: []int{} contains int32(5) (mismatched types int and rune)`), nil},
	{`[]int{} contains i`, tierr(1, 12, `invalid operation: []int{} contains i (mismatched types int and compiler.definedInt)`), map[string]*typeInfo{"i": definedIntTypeInfo}},
	{`[2]int{0,1} contains rune('a')`, tierr(1, 16, `invalid operation: [2]int{...} contains rune('a') (mismatched types int and rune)`), nil},

	// macro type literal
	{`(macro() css)(nil)`, tierr(1, 13, `invalid macro result type css`), map[string]*typeInfo{"css": {Type: reflect.TypeOf(0), Properties: propertyIsType}}},
	{`(macro() html)(nil)`, tierr(1, 13, `invalid macro result type html`), map[string]*typeInfo{"html": {Type: reflect.TypeOf(definedInt(0)), Properties: propertyIsType}}},
	{`(macro() markdown)(nil)`, tierr(1, 13, `invalid macro result type markdown`), map[string]*typeInfo{"markdown": {Type: reflect.TypeOf(js("")), Properties: propertyIsType}}},

	// slicing of a format type
	{`a[1:2]`, tierr(1, 5, `invalid operation a[1:2] (slice of compiler.html)`), map[string]*typeInfo{"a": tiHTMLConst("<b>a</b>")}},
	{`a[1:2]`, tierr(1, 5, `invalid operation a[1:2] (slice of compiler.html)`), map[string]*typeInfo{"a": tiHTML()}},

	// key selector
	{`m.x`, tierr(1, 5, `invalid operation: cannot select m.x (type map[int]string does not support key selection)`), map[string]*typeInfo{"m": intToStringMapTypeInfo}},
	{`m.x`, tierr(1, 5, `invalid operation: cannot select m.x (type map[int]interface {} does not support key selection)`), map[string]*typeInfo{"m": intToAnyMapTypeInfo}},
	{`m.x.y`, tierr(1, 7, `m.x.y undefined (type int has no field or method y)`), map[string]*typeInfo{"m": stringToIntMapTypeInfo}},
	{`m.x.y.z`, tierr(1, 7, `m.x.y undefined (type int has no field or method y)`), map[string]*typeInfo{"m": stringToIntMapTypeInfo}},
	{`nil.x.y.z`, tierr(1, 4, `use of untyped nil`), nil},
	{`m._`, tierr(1, 5, `cannot refer to blank field or method`), map[string]*typeInfo{"m": stringToAnyMapTypeInfo}},
}

func TestCheckerTemplateExpressionErrors(t *testing.T) {
	options := checkerOptions{mod: templateMod, formatTypes: formatTypes}
	for _, expr := range checkerTemplateExprErrors {
		var lex = scanTemplate([]byte("{{ "+expr.src+" }}"), ast.FormatText, false)
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
				lex:       lex,
				ancestors: nil,
			}
			p.next() // discard tokenLeftBraces.
			node, tok := p.parseExpr(p.next(), false, false, false, false)
			if node == nil {
				t.Errorf("source: %q, unexpected %s, expecting expression\n", expr.src, tok)
				return
			}
			if tok.typ != tokenRightBraces {
				t.Errorf("source: %q, unexpected %s, expecting }}\n", expr.src, tok)
				return
			}
			compilation := newCompilation(nil)
			tc := newTypechecker(compilation, "", options, nil)
			for name, ti := range expr.scope {
				tc.scopes.Declare(name, ti, nil, nil)
			}
			tc.scopes.Enter(node)
			ti := tc.checkExpr(node)
			tc.scopes.Exit()
			t.Errorf("source: %s, unexpected %s, expecting error %q\n", expr.src, ti, expr.err)
			err := compilation.finalizeUsingStatements(tc)
			if err != nil {
				t.Fatal(err)
			}
		}()
	}
}

var checkerTemplateStmts = []struct {
	src      string
	expected string
}{

	// Misc.
	{
		src:      `Just test`,
		expected: ok,
	},
	{
		src:      `{{ a }}`,
		expected: "undefined: a",
	},
	// Macro definitions.
	{
		src:      `{% macro M %}{% end %}`,
		expected: ok,
	},
	{
		src:      `{% macro M() %}{% end %}`,
		expected: ok,
	},
	{
		src:      `{% macro M(a, b int) %}{% end %}`,
		expected: ok,
	},
	{
		src:      `{% macro M(int, int, string) %}{% end %}`,
		expected: ok,
	},
	{
		src:      `{% macro M(a) %}{% end %}`,
		expected: "undefined: a",
	},
	{
		src:      `{% macro M %}{% end %}{% macro M %}{% end %}`,
		expected: "1:32: M redeclared in this block\n\tprevious declaration at 1:10",
	},

	// Show macro.
	{
		src:      `{% macro M %}{% end %}         {% show M() %}`,
		expected: ok,
	},
	{
		src:      `{% macro M %}{% end %}         {% show M(1) %}`,
		expected: "too many arguments in call to M\n\thave (number)\n\twant ()",
	},
	{
		src:      `{% macro M(int) %}{% end %}    {% show M("s") %}`,
		expected: "cannot use \"s\" (type untyped string) as type int in argument to M",
	},

	{
		src:      `{% macro M %}{% end %}    {% show M() %}`,
		expected: ok,
	},

	{
		src:      `{% show M() %}`,
		expected: `undefined: M`,
	},

	{
		src:      `{% a := 10 %}{% a %}`,
		expected: `a evaluated but not used`,
	},

	{
		src:      `{% a := 20 %}{{ a and a }}`,
		expected: ok,
	},

	{
		src:      `{% a := 20 %}{{ a or a }}`,
		expected: ok,
	},

	{
		src:      `{% a := 20 %}{% b := "" %}{{ a or b and (not b) }}`,
		expected: ok,
	},

	{
		src:      `{% a := 20 %}{{ 3 and a }}`,
		expected: ok,
	},

	{
		src:      `{% a := 20 %}{{ 3 or a }}`,
		expected: ok,
	},

	{
		src:      `{% const a = 20 %}{{ not a }}`,
		expected: ok,
	},

	{
		src:      `{% a := true %}{% b := true %}{{ a and b or b and b }}`,
		expected: ok,
	},

	{
		src:      `{% n := 10 %}{% var a bool = not n %}`,
		expected: ok,
	},

	{
		src:      `{% a := []int(nil) %}{% if a %}{% end %}`,
		expected: ``,
	},

	{
		src:      `{% if 20 %}{% end %}`,
		expected: ``,
	},

	{
		src:      `{{ true and nil }}`,
		expected: `invalid operation: true and nil (operator 'and' not defined on nil)`,
	},

	{
		src:      `{{ nil and false }}`,
		expected: `invalid operation: nil and false (operator 'and' not defined on nil)`,
	},

	{
		src:      `{{ true or nil }}`,
		expected: `invalid operation: true or nil (operator 'or' not defined on nil)`,
	},

	{
		src:      `{{ not nil }}`,
		expected: `invalid operation: not nil (operator 'not' not defined on nil)`,
	},

	{
		src:      `{% v := 10 %}{{ v and nil }}`,
		expected: `invalid operation: v and nil (operator 'and' not defined on nil)`,
	},

	{
		// Check that the 'and' operator returns an untyped bool even if its two
		// operands are both typed booleans. The same applies to the 'or' and
		// 'not' operators.
		src: `
			{% type Bool bool %}
			{% var _ bool = Bool(true) and Bool(false) %}
		`,
		expected: ok,
	},

	{
		// Check that a format type value can be explicitly converted to
		// string.
		src: `
			{%%
				var s1 html
				var s2 css
				var s3 js
				var s4 json
				var s5 markdown
				_ = string(s1)
				_ = string(s2)
				_ = string(s3)
				_ = string(s4)
				_ = string(s5)
			%%}
		`,
		expected: ok,
	},

	{
		// Check that an untyped constant string value can be converted to a
		// format type.
		src: `
			{%%
				 const s = "a" 
				_ = html(s)
				_ = css(s)
				_ = js(s)
				_ = json(s)
				_ = markdown(s)
			%%}
		`,
		expected: ok,
	},

	{
		src:      `{% s := "a" %}{% _ = html(s) %}`,
		expected: `cannot convert s (type string) to type compiler.html`,
	},

	{
		src:      `{% s := "a" %}{% _ = css(s) %}`,
		expected: `cannot convert s (type string) to type compiler.css`,
	},

	{
		src:      `{% s := "a" %}{% _ = js(s) %}`,
		expected: `cannot convert s (type string) to type compiler.js`,
	},

	{
		src:      `{% s := "a" %}{% _ = json(s) %}`,
		expected: `cannot convert s (type string) to type compiler.json`,
	},

	{
		src:      `{% s := "a" %}{% _ = markdown(s) %}`,
		expected: `cannot convert s (type string) to type compiler.markdown`,
	},

	{
		// Check that an typed format constant can be converted to the same
		// format type.
		src: `
			{%%
				const s1 html = "a"
				const s2 css = "a"
				const s3 js = "a"
				const s4 json = "a"
				const s5 markdown = "a"
				_ = html(s1)
				_ = css(s2)
				_ = js(s3)
				_ = json(s4)
				_ = markdown(s5)
			%%}
		`,
		expected: ok,
	},

	{
		// Check that a non-constant format value can be converted to the same
		// format type.
		src: `
			{%%
				var s1 html
				var s2 css
				var s3 js
				var s4 json
				var s5 markdown
				_ = html(s1)
				_ = css(s2)
				_ = js(s3)
				_ = json(s4)
				_ = markdown(s5)
			%%}
		`,
		expected: ok,
	},

	// 'for in' statements.
	{src: `{%% for v in "abc" { var _ rune = v } %%}`, expected: ok},
	{src: `{%% for _ in "abc" { } %%}`, expected: ok},
	{src: `{%% for v in ([...]int{}) { var _ int = v } %%}`, expected: ok},
	{src: `{%% for k in map[float64]string{} { var _ float64 = k } %%}`, expected: ok},
	{src: `{%% for _ in (&[...]int{}) { } %%}`, expected: ok},
	{src: `{%% for a in make(<-chan string) { var _ string = a } %%}`, expected: ok},
	{src: `{%% for _ in 0 { } %%}`, expected: `cannot range over 0 (type untyped number)`},
	{src: `{%% for _ in (&[]int{}) { } %%}`, expected: `cannot range over &[]int{} (type *[]int)`},
	{src: `{%% for a, b in "" { } %%}`, expected: `unexpected in, expecting := or = or comma`}, // should be better 'too many variables in range'.
	{src: `{%% for a in nil { } %%}`, expected: `cannot range over nil`},
	{src: `{%% for a in _ { } %%}`, expected: `cannot use _ as value`},
	{src: `{%% for a in make(chan<- int) { } %%}`, expected: `invalid operation: range make(chan<- int) (receive from send-only type chan<- int)`},
	{src: `{%% for a in ([]int{1,2,3}) { } else { } %%}`, expected: ok},
	{src: `{%% for a in ([]int{1,2,3}) { } else { _ = a } %%}`, expected: `undefined: a`},

	// 'show' statements.
	{src: `{% show "a" %}`, expected: ok},
	{src: `{% show "a", 7, true %}`, expected: ok},
	{src: `{% show render "partial.html" %}`, expected: ok},
	{src: `{% show render "partial.html", render "partial.html" %}`, expected: ok},

	// Variable declarations and assignments with 'default' expression.
	{src: `{% var a = I default 5 %}`, expected: ok},
	{src: `{% var a = J default 5 %}`, expected: ok},
	{src: `{% var a = I default "" %}`, expected: `cannot use I (type int) as type string in assignment`},
	{src: `{% var a int = I default 5 %}`, expected: ok},
	{src: `{% var a int = J default 5 %}`, expected: ok},
	{src: `{% var a string = I default "" %}`, expected: `cannot use I (type int) as type string in assignment`},
	{src: `{% var a string = S default 5 %}`, expected: `cannot use 5 (type untyped int) as type string in assignment`},
	{src: `{% var a interface{} = S default 5 %}`, expected: ok},
	{src: `{% a := I default 5 %}`, expected: ok},
	{src: `{% a := J default 5 %}`, expected: ok},
	{src: `{% a := I default "" %}`, expected: `cannot use I (type int) as type string in assignment`},
	{src: `{% var a int %}{% a = I default 5 %}`, expected: ok},
	{src: `{% var a int %}{% a = J default 5 %}`, expected: ok},
	{src: `{% var a string %}{% a = I default "" %}`, expected: `cannot use I (type int) as type string in assignment`},
	{src: `{% var a interface{} %}{% a = I default "" %}`, expected: ok},
	{src: `{% a := 5 %}{% var b = a default 0 %}`, expected: `use of non-builtin a on left side of default`},
	{src: `{% var a = _ default 0 %}`, expected: `cannot use _ as value`},
	{src: `{% var a = p default 0 %}`, expected: `use of package p without selector`},
	{src: `{% var a = nil default 0 %}`, expected: `use of untyped nil`},
	{src: `{% var a = len default 0 %}`, expected: `use of builtin len not in function call`},
	{src: `{% var a = true default false %}`, expected: ok},
	{src: `{% var loc int %}{% var a = loc default 0 %}`, expected: `use of non-builtin loc on left side of default`},
	{src: `{% var a = T default 0 %}`, expected: `unexpected type on left side of default`},

	// Constant declaration with 'default' expression.
	{src: `{% const c = Ui default 3 %}`, expected: ok},
	{src: `{% const c = D default 3 %}`, expected: ok},
	{src: `{% const c = R default 3 %}`, expected: `cannot use typed const R in untyped const initializer`},
	{src: `{% const c = Uf default 3 %}`, expected: `mismatched kinds floating-point and integer in untyped const initializer`},
	{src: `{% const c int = Ci default 3 %}`, expected: ok},
	{src: `{% const c int = D default int(3) %}`, expected: ok},
	{src: `{% const c int = Ui default int(3) %}`, expected: ok},
	{src: `{% const c int = Ui default 3 %}`, expected: ok},
	{src: `{% const c string = Ui default "" %}`, expected: `cannot use Ui (type untyped int) as type string in assignment`},
	{src: `{% const c = iota default 0 %}`, expected: ok},

	// Other default expression uses.
	{src: `{{ 5 + ( x default 3 ) - 2 }}`, expected: `cannot use default expression in this context`},
	{src: `{{ -x default 3 }}`, expected: `cannot use default expression in this context`},

	// Labels.
	{src: `{% L: for %}{% break L %}{% end %}`, expected: ok},
	//{src: `{% L: for %}{% continue L %}{% end %}`, expected: ok}, TODO: panic "panic: TODO(Gianluca): not implemented"
	{src: `{% L: switch %}{% default %}{% break L %}{% end %}`, expected: ok},
	{src: `{% L: select %}{% default %}{% break L %}{% end %}`, expected: ok},
	{src: `{% _ = func() { L: goto L } %}`, expected: ok},
	{src: `{% L: for %}{% _ = func() { goto L } %}{% end %}`, expected: `label L not defined`},

	// Key selector.
	{src: `{% m := map[string]int{} %}{% m.x = 5 %}{% m.x += 1 %}{% m.x++ %}{{ m.x + 2 }}`, expected: ok},
	{src: `{% m := map[DS]int{} %}{% m.x = 5 %}{% m.x += 1 %}{% m.x++ %}{{ m.x + 2 }}`, expected: ok},
	{src: `{% m := map[string]bool{} %}{% m.nil = true %}{{ m.nil && false }}`, expected: ok},
	{src: `{% m := map[interface{}]string{} %}{% m.x = "a" %}{{ m.a + "b" }}`, expected: ok},
	{src: `{% m := map[string]interface{}{} %}{% m.x = 6.89 %}{{ m.x.(float64) - 1.4 }}`, expected: ok},
	{src: `{% m := map[string]interface{}{} %}{% m.x = map[string]interface{}{} %}{% m.x.y = true %}`, expected: `cannot index m.x (map index expression of type interface{})`},
	{src: `{% m := map[interface{}]interface{}{} %}{% m.x = map[string]interface{}{} %}{% m.x.y = 'v' %}`, expected: `cannot index m.x (map index expression of type interface{})`},
	{src: `{% m := map[string]map[string]int{} %}{% m.x = map[string]int{} %}{% m.x.y = 3 %}{% m.x.y += 1 %}{% m.x.y++ %}{{ m.x.y * 2 }}`, expected: ok},
	{src: `{% m := map[string]map[string]interface{}{} %}{% m.x = map[string]interface{}{} %}{% m.x.y = "a" %}{{ m.x.y.(string) + "b" }}`, expected: ok},
	{src: `{% m := map[string]int{} %}{% m.x = "a" %}`, expected: `cannot use "a" (type untyped string) as type int in assignment`},
	{src: `{% m := map[interface{}]int{} %}{% m.x = "a" %}`, expected: `cannot use "a" (type untyped string) as type int in assignment`},
	{src: `{% m := map[string]int{} %}{% m.x := "a" %}`, expected: `non-name m.x on left side of :=`},
	{src: `{% m := map[interface{}]int{} %}{% m.x := "a" %}`, expected: `non-name m.x on left side of :=`},
	{src: `{% m := map[string]int{} %}{{ m.x + "a" }}`, expected: `invalid operation: m.x + "a" (cannot convert "a" (type untyped string) to type int)`},
	{src: `{% m := map[interface{}]int{} %}{{ m.x + "a" }}`, expected: `invalid operation: m.x + "a" (cannot convert "a" (type untyped string) to type int)`},
}

func TestCheckerTemplatesStatements(t *testing.T) {
	var I = 3
	var S = "s"
	p := native.Package{
		Name:         "p",
		Declarations: native.Declarations{},
	}
	options := Options{
		FormatTypes: formatTypes,
		Globals: native.Declarations{
			"p":  p,
			"T":  reflect.TypeOf(int(0)),
			"I":  &I,
			"S":  &S,
			"Ci": 5,
			"Ui": native.UntypedNumericConst("5"),
			"Uf": native.UntypedNumericConst("5.0"),
			"R":  'r',
			"DS": reflect.TypeOf(definedString("")),
		},
	}
	for _, cas := range checkerTemplateStmts {
		src := cas.src
		expected := cas.expected
		t.Run(src, func(t *testing.T) {
			fsys := fstest.Files{"index.html": src, "partial.html": "x"}
			_, err := BuildTemplate(fsys, "index.html", options)
			switch {
			case expected == "" && err != nil:
				t.Fatalf("unexpected error: %q", err)
			case expected != "" && err == nil:
				t.Fatalf("expecting error %q, got nothing", expected)
			case expected != "" && err != nil && !strings.Contains(err.Error(), expected):
				t.Fatalf("expecting error %q, got %q", expected, err.Error())
			}
		})
	}
}

type S []V
type V []S
type L struct {
	M M
	C complex64
}
type M struct {
	L *L
}

type N struct {
	Slice []N
	Map   map[int]N
}

func TestJSCycles(t *testing.T) {
	for i := 0; i < 1; i++ {
		err := checkShowJS(reflect.TypeOf(S{}), nil)
		if err != nil {
			t.Fatalf("unexpected error %q showing S", err)
		}
		err = checkShowJS(reflect.TypeOf(L{}), nil)
		if err == nil {
			t.Fatalf("unexpected nil error showing L")
		}
		if err.Error() != "cannot show struct containing complex64 as JavaScript" {
			t.Fatalf("unexpected error %q showing L", err)
		}
		err = checkShowJS(reflect.TypeOf(N{}), nil)
		if err != nil {
			t.Fatalf("unexpected error %q showing N", err)
		}
	}
}

func TestJSONCycles(t *testing.T) {
	for i := 0; i < 1; i++ {
		err := checkShowJSON(reflect.TypeOf(S{}), nil)
		if err != nil {
			t.Fatalf("unexpected error %q showing S", err)
		}
		err = checkShowJSON(reflect.TypeOf(L{}), nil)
		if err == nil {
			t.Fatalf("unexpected nil error showing L")
		}
		if err.Error() != "cannot show struct containing complex64 as JSON" {
			t.Fatalf("unexpected error %q showing L", err)
		}
		err = checkShowJSON(reflect.TypeOf(N{}), nil)
		if err != nil {
			t.Fatalf("unexpected error %q showing N", err)
		}
	}
}
