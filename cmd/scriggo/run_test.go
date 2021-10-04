// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"reflect"
	"sort"
	"testing"

	"github.com/open2b/scriggo/native"
)

var parseConstantsTests = []struct {
	constants    string
	declarations native.Declarations
	err          error
}{
	{"", nil, nil},
	{`t1=false`, native.Declarations{"t1": native.UntypedBooleanConst(false)}, nil},
	{`t2=true`, native.Declarations{"t2": native.UntypedBooleanConst(true)}, nil},
	{`s1=""`, native.Declarations{"s1": native.UntypedStringConst("")}, nil},
	{`s2="foo boo"`, native.Declarations{"s2": native.UntypedStringConst(`foo boo`)}, nil},
	{`s3="\"foo boo\" \\"`, native.Declarations{"s3": native.UntypedStringConst(`"foo boo" \`)}, nil},
	{"s4=``", native.Declarations{"s4": native.UntypedStringConst("")}, nil},
	{"s5=`foo boo`", native.Declarations{"s5": native.UntypedStringConst("foo boo")}, nil},
	{`n1=5`, native.Declarations{"n1": native.UntypedNumericConst(`5`)}, nil},
	{`n2=23.89`, native.Declarations{"n2": native.UntypedNumericConst(`23.89`)}, nil},
	{` a =  1 `, native.Declarations{"a": native.UntypedNumericConst(`1`)}, nil},
	{` b  = "foo" `, native.Declarations{"b": native.UntypedStringConst(`foo`)}, nil},
	{"c = \"foo\" d= `boo` e =6 f= false", native.Declarations{
		"c": native.UntypedStringConst(`foo`),
		"d": native.UntypedStringConst(`boo`),
		"e": native.UntypedNumericConst("6"),
		"f": native.UntypedBooleanConst(false),
	}, nil},
}

// TestParseConstants tests the parseConstants function.
func TestParseConstants(t *testing.T) {
	for _, test := range parseConstantsTests {
		decls := native.Declarations{}
		err := parseConstants(test.constants, decls)
		if err != nil {
			if test.err == nil {
				t.Fatal(err)
			}
			if err != test.err {
				t.Fatalf("expected error %q, got %q", test.err, err)
			}
			continue
		}
		for name, expected := range test.declarations {
			got, ok := decls[name]
			if !ok {
				t.Fatalf("missing constant %s", name)
			}
			et := reflect.TypeOf(expected)
			gt := reflect.TypeOf(got)
			if et != gt {
				t.Fatalf("expected type %s for constant %s, got %s", et, name, gt)
			}
			ev := reflect.ValueOf(expected)
			gv := reflect.ValueOf(got)
			if ev.String() != gv.String() {
				t.Fatalf("expected %s for constant %s, got %s", ev.String(), name, gv.String())
			}
			delete(decls, name)
		}
		if len(decls) > 0 {
			var names []string
			for name := range decls {
				names = append(names, name)
			}
			sort.Strings(names)
			t.Fatalf("unexpected constant %s", names[0])
		}
	}

}
