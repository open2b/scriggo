// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"sort"
	"strconv"
	"testing"
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

	"two variables where second depends on first": {
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

	"function with a local variable (so has no depencencies)": {
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

	"example from https://golang.org/ref/spec#Package_initialization": {
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
}

func TestDependencies(t *testing.T) {
	for name, cas := range cases {
		t.Run(name, func(t *testing.T) {
			_, got, err := ParseSource([]byte(cas.src), false, false)
			if err != nil {
				t.Fatalf("parsing error: %s", err)
			}
			gotIdentifiers := map[string][]string{}
			for symbol, deps := range got {
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
