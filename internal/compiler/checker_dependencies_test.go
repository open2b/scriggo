// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/open2b/scriggo/ast"
)

var cases = map[string]struct {
	src      string
	expected map[string][]string
}{
	"two independent variables": {
		`package pkg

		var A = 10
		var B = 20`,
		map[string][]string{
			"A": {},
			"B": {},
		},
	},

	"variable with a type and a value": {
		`package pkg
		
		var A int = 20`,
		map[string][]string{
			"A": {"int"},
		},
	},

	"triple var assignment with no dependencies": {
		`package pkg

			var A, B, C = 1, 2, 3`,
		map[string][]string{
			"A": {},
			"B": {},
			"C": {},
		},
	},

	"triple var assignment with dependency from another var": {
		`package pkg

			var C = 10
			var A, B, C = 1, C, 3`,
		map[string][]string{
			"A": {},
			"B": {"C"},
			"C": {},
		},
	},

	"two variables where the second depends on first": {
		`package pkg

			var A = 10
			var B = A`,
		map[string][]string{
			"A": {},
			"B": {"A"},
		},
	},

	"two variables initialization loop": {
		`package pkg

			var A = B
			var B = A`,
		map[string][]string{
			"A": {"B"},
			"B": {"A"},
		},
	},

	"three variables initialization loop": {
		`package pkg

			var A = B
			var B = C
			var C = A`,
		map[string][]string{
			"A": {"B"},
			"B": {"C"},
			"C": {"A"},
		},
	},

	"three variables": {
		`package pkg

			var A = B + C
			var B = 4 + 12
			var C = B`,
		map[string][]string{
			"A": {"B", "C"},
			"B": {},
			"C": {"B"},
		},
	},

	"two independent constants": {
		`package pkg

			const C1 = 80
			const C2 = 100`,
		map[string][]string{
			"C1": {},
			"C2": {},
		},
	},

	"two constants where second depends on first": {
		`package pkg

			var C1 = "str"
			var C2 = C1`,
		map[string][]string{
			"C1": {},
			"C2": {"C1"},
		},
	},

	"two constants initialization loop": {
		`package pkg

			var C1 = C2
			var C2 = C1`,
		map[string][]string{
			"C1": {"C2"},
			"C2": {"C1"},
		},
	},

	"variable depending on undefined symbol": {
		`package pkg

			var A = undef`,
		map[string][]string{
			"A": {"undef"},
		},
	},

	"variable depending on function": {
		`package pkg

			var A = F()

			func F() int { return 0 }`,
		map[string][]string{
			"A": {"F"},
			"F": {"int"},
		},
	},

	"function depending on a variable": {
		`package pkg

			var A = 20

			func F() {
				_ = A
			}`,
		map[string][]string{
			"A": {},
			"F": {"A"},
		},
	},

	"function depending on a variable depending on a function - initialization loop": {
		`package main

			var A = F()
			func F() int {
				return A
			}`,
		map[string][]string{
			"A": {"F"},
			"F": {"int", "A"},
		},
	},

	"two variables depending on the same constant": {
		`package main

			const C1 = 10
			var A = C1
			var B = C1`,
		map[string][]string{
			"C1": {},
			"A":  {"C1"},
			"B":  {"C1"},
		},
	},

	"two functions referencing each other": {
		`package main

			func F() {
				G()
			}

			func G() {
				F()
			}`,
		map[string][]string{
			"F": {"G"},
			"G": {"F"},
		},
	},

	"variable referenced more than once": {
		`package main

			var A = B + B * B
			var B = 20`,
		map[string][]string{
			"A": {"B"},
			"B": {},
		},
	},

	"function with a local variable (so has no dependencies)": {
		`package main

			func F() {
				var A = 20
				_ = A
			}`,
		map[string][]string{
			"F": {},
		},
	},

	"function with a local variable which goes out of scope (so depends on a global variable)": {
		`package main

			var A = 10

			func F() {
				{
					var A = 20
					_ = A
				}
				_ = A
			}`,
		map[string][]string{
			"A": {},
			"F": {"A"},
		},
	},

	"function with if statement": {
		`package main

			var A = 10
			var B = 20

			func F() {
				_ = A
				if true {
					B := 20
					var C = 30
					_ = B
					_ = C
				}
				_ = B
			}`,
		map[string][]string{
			"A": {},
			"B": {},
			"F": {"A", "B", "true"},
		},
	},

	"multiple assignment with function: var a, b, c = f()": {
		`package main

			var a, b, c = f()

			func f() (int, int, string) {
				return 0, 1, "str"
			}`,
		map[string][]string{
			"a": {"f"},
			"b": {"f"},
			"c": {"f"},
			"f": {"int", "string"},
		},
	},

	"example from https://go.dev/ref/spec#Package_initialization": {
		`package pkg

			var (
				a = c + b
				b = f()
				c = f()
				d = 3
			)

			func f() int {
				d++
				return d
			}`,
		map[string][]string{
			"a": {"c", "b"},
			"b": {"f"},
			"c": {"f"},
			"d": {},
			"f": {"int", "d"},
		},
	},

	"function uses both global and local variable with same name": {
		`
			package pkg

			var A = 20

			func F() {
				_ = A
				A := 10
				_ = A
			}`,
		map[string][]string{
			"A": {},
			"F": {"A"},
		},
	},

	"complex test #1": {
		`package main

			import (
				"fmt"
			)

			var m = map[string]int{"20": 12}
			var notOk = !ok
			var doubleValue = value * 2
			var value, ok = m[k()]

			func k() string {
				a := 20
				return fmt.Sprintf("%d", a)
			}

			func main() {
				fmt.Println(m)
				fmt.Println(notOk)
				fmt.Println(doubleValue)
			}
			`,
		map[string][]string{
			"m":           {"string", "int"},
			"notOk":       {"ok"},
			"doubleValue": {"value"},
			"value":       {"m", "k"},
			"ok":          {"m", "k"},
			"k":           {"string", "fmt"},
			"main":        {"fmt", "m", "notOk", "doubleValue"},
		},
	},

	"complex test #2": {
		`package main

		import "fmt"
		
		var A = B
		var B = 100
		var i string
		
		func main() {
			switch A {
			case B:
			case 200:
				for i := 0; i < 10; i++ {
					fmt.Println(i)
				}
			}
		}
		`,
		map[string][]string{
			"A":    {"B"},
			"B":    {},
			"i":    {"string"},
			"main": {"A", "B", "fmt"},
		},
	},

	"type switch in function main": {
		`package main

		func main() {
			var i = interface{}(10)
			switch i.(type) {
			case int:
			}
		}
		`,
		map[string][]string{
			"main": {"int"},
		},
	},

	"local variable with same name as package variable (var A = A)": {
		`package main

		var A = 1
		
		func main() {
			var A = A // <- this A is not local!
			_ = A
		}
		`,
		map[string][]string{
			"A":    {},
			"main": {"A"},
		},
	},

	"local variable with same name as package variable (A := A)": {
		`package main

		var A = 1
		
		func main() {
			A := A // <- this A is not local!
			_ = A
		}
		`,
		map[string][]string{
			"A":    {},
			"main": {"A"},
		},
	},

	"local constant with same name as package constant (const C = C)": {
		`package main

		const C = 1
		
		func main() {
			const C = C // <- this C is not local!
		}`,
		map[string][]string{
			"C":    {},
			"main": {"C"},
		},
	},

	"complex test #3": {
		`package main

		import (
			"fmt"
		)
		
		var A []int
		var m map[string]interface{} = map[string]interface{}{
			"a": 4,
			"b": -20,
		}
		
		func main() {
			fmt.Println(m)
		}
		`,
		map[string][]string{
			"A":    {"int"},
			"m":    {"string"},
			"main": {"fmt", "m"},
		},
	},

	"Local functions that don't depend on package variables": {
		`package main

		var A int
		var B int
		
		func main() {
			F := func(A int) {
				_ = A
			}
			_ = F
			G := func() (B int) {
				B = 100
				return 0
			}
			_ = G
		}
		`,
		map[string][]string{
			"A":    {"int"},
			"B":    {"int"},
			"main": {"int"},
		},
	},

	"Function that has parameters with same name as package variables": {
		`package main

		var A int
		var B int
		
		func F(A int) {
		}
		
		func G() (B int) {
			return 0
		}
		
		func H(a int) {
			A = a
		}
		
		func I() (b int) {
			B = b
			return 0
		}
		
		func main() {}
		`,
		map[string][]string{
			"A":    {"int"},
			"B":    {"int"},
			"F":    {"int"},
			"G":    {"int"},
			"H":    {"int", "A"},
			"I":    {"int", "B"},
			"main": {},
		},
	},

	"https://github.com/golang/go/issues/22326": {
		`package main

		var (
			_ = d
			_ = f("_", c, b)
			a = f("a")
			b = f("b")
			c = f("c")
			d = f("d")
		)
		
		func f(s string, rest ...int) int {
			print(s)
			return 0
		}
		
		func main() {
			println()
		}
		`,
		map[string][]string{
			"underscore_at_line_4": {"d"},
			"underscore_at_line_5": {"f", "c", "b"},
			"a":                    {"f"},
			"b":                    {"f"},
			"c":                    {"f"},
			"d":                    {"f"},
			"f":                    {"string", "int", "print"},
			"main":                 {"println"},
		},
	},
	"two blank identifiers": {
		`package main

		var _ = A
		var _ = 10
		var A = 20
		
		func main() {}
		`,
		map[string][]string{
			"underscore_at_line_3": {"A"},
			"underscore_at_line_4": {},
			"A":                    {},
			"main":                 {},
		},
	},
	"single global variable with type": {
		`
		package pkg
		
		var A int`,
		map[string][]string{
			"A": {"int"},
		},
	},
	"one alias declarations": {
		`package pkg
		
		type Int = int`,
		map[string][]string{
			"Int": {"int"},
		},
	},
	"two independent alias declarations": {
		`package pkg
		
		type Int = int
		type String = string`,
		map[string][]string{
			"Int":    {"int"},
			"String": {"string"},
		},
	},
	"two alias declarations where second depends on first": {
		`package pkg
		
		type A = int
		type B = A`,
		map[string][]string{
			"A": {"int"},
			"B": {"A"},
		},
	},
	"two alias declarations where second depends on first (more complex)": {
		`package pkg
		
		type A = int
		type B = map[A][]string`,
		map[string][]string{
			"A": {"int"},
			"B": {"A", "string"},
		},
	},
	"function declaration referring to a type declaration": {
		`package pkg
		
		type Int = int
		type String = string
		type Mix = map[Int]String
		
		func F(i Int, s String) Mix {
			return Mix{}
		}`,
		map[string][]string{
			"Int":    {"int"},
			"String": {"string"},
			"Mix":    {"Int", "String"},
			"F":      {"Int", "String", "Mix"},
		},
	},
	"alias declarations, constants, variables and functions": {
		`package main

		type A = int
		type B = A

		const C A = 10
		const C2 B = C

		var V1 A
		var V2 B

		func F(a, b, c A) B {
			return B(0)
		}

		func main() {}`,
		map[string][]string{
			"A":    {"int"},
			"B":    {"A"},
			"C":    {"A"},
			"C2":   {"B", "C"},
			"V1":   {"A"},
			"V2":   {"B"},
			"F":    {"A", "B"},
			"main": {},
		},
	},
	"select - no cases": {
		`package main
		
		func main() {
			select {}
		}`,
		map[string][]string{
			"main": {},
		},
	},
	"select - one case that receives from a package level channel": {
		`package main

		var ch = make(chan int, 3)
		
		func main() {
			select {
			case <-ch:
			}
		}
		`,
		map[string][]string{
			"main": {"ch"},
			"ch":   {"int", "make"},
		},
	},
	"select - three receive operations": {
		`package main

		var ch = make(chan int, 3)
		
		var v1, v2 int
		
		func main() { 
			select {
			case <-ch:
			case v1 = <-ch:
				_ = v1
			case v2, ok := <-ch:
				_ = v2
				_ = ok
			}
		}
		`,
		map[string][]string{
			"main": {"ch", "v1"}, // does not depend on v2, which is locally declared.
			"ch":   {"make", "int"},
			"v1":   {"int"},
			"v2":   {"int"},
		},
	},
}

func TestDependencies(t *testing.T) {
	for name, cas := range cases {
		t.Run(name, func(t *testing.T) {
			tree, err := parseSource([]byte(cas.src), false)
			if err != nil {
				t.Fatalf("parsing error: %s", err)
			}
			got := analyzeTree(tree.Nodes[0].(*ast.Package))
			gotIdentifiers := map[string][]string{}
			for symbol, deps := range got {
				if symbol.Name == "_" {
					symbol.Name = "underscore_at_line_" + strconv.Itoa(symbol.Pos().Line)
				}
				gotIdentifiers[symbol.Name] = []string{}
				for _, dep := range deps {
					gotIdentifiers[symbol.Name] = append(gotIdentifiers[symbol.Name], dep.Name)
				}
			}
			for name := range cas.expected {
				sort.Strings(cas.expected[name])
			}
			for name := range gotIdentifiers {
				sort.Strings(gotIdentifiers[name])
			}
			if !reflect.DeepEqual(cas.expected, gotIdentifiers) {
				t.Fatalf("%s: expecting %s, got %s", name, fmtDeps(cas.expected), fmtDeps(gotIdentifiers))
			}
		})
	}
}

func fmtDeps(symbolDeps map[string][]string) string {
	s := "{"
	i := 0
	for symbol, deps := range symbolDeps {
		s += strconv.Quote(symbol)
		s += " → ["
		for i, dep := range deps {
			s += strconv.Quote(dep)
			if i < len(deps)-1 {
				s += ", "
			}
		}
		s += "]"
		if i < len(symbolDeps)-1 {
			s += ", "
		}
		i++
	}
	s += "}"
	return s
}
