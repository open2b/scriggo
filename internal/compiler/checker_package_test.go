// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"testing"

	"scriggo/ast"
)

func TestInitializationLoop(t *testing.T) {
	cases := map[string]struct {
		src      string
		expected string
	}{
		"#1": {
			`package main
		
			var A = 10
			var B = 20`,
			"",
		},
		"#2": {
			`package main
			
			var A = B
			var B = A`,
			"3:4: typechecking loop involving var A = B\n\t3:8: A\n\t3:12: B\n\t4:12: A\n",
		},
		"#3": {
			`package main
			
			const C1 = C2
			const C2 = C3
			const C3 = C1`,
			"3:4: constant definition loop\n\t3:10: C1 uses C2\n\t3:15: C2 uses C3\n\t4:15: C3 uses C1\n",
		},
	}
	for name, cas := range cases {
		t.Run(name, func(t *testing.T) {
			tree, err := ParseSource([]byte(cas.src), false, false)
			if err != nil {
				t.Fatalf("parsing error: %s", err)
			}
			deps := AnalyzeTree(tree, ProgramSyntax)
			pkg := tree.Nodes[0].(*ast.Package)
			vars := []*ast.Var{}
			consts := []*ast.Const{}
			for _, d := range pkg.Declarations {
				switch d := d.(type) {
				case *ast.Var:
					vars = append(vars, d)
				case *ast.Const:
					consts = append(consts, d)
				}
			}
			got := ""
			err = detectConstantsLoop(consts, deps)
			if err != nil {
				got += err.Error()
			}
			err = detectVarsLoop(vars, deps)
			if err != nil {
				got += err.Error()
			}
			if cas.expected != got {
				t.Fatalf("expecting error %q, got %q", cas.expected, got)
			}
		})
	}
}

func TestPackageOrdering(t *testing.T) {
	cases := map[string]struct {
		src      string
		expected string
	}{
		"two independent variables": {
			`package pkg

			var A = 10
			var B = 20
			`,
			"A,B,",
		},

		"first variable depending on second one": {
			`package pkg

			var A = B
			var B = 20`,
			"B,A,",
		},

		"constants and variables (unrelated)": {
			`package pkg

			const  C1  =  10
			var    A   =  30
			const  C2  =  20
			var    B   =  40`,
			"C1,C2,A,B,",
		},

		"constants and variables (related)": {
			`package pkg

			const  C1  =  10
			var    B   =  C1
			var    A   =  C2
			const  C2  =  20`,
			"C1,C2,B,A,",
		},

		"complex variables": {
			`package pkg

			var (
				A = B
				B = 10
				C = D
				D = A
				E = A + B
			)`,
			"B,A,D,C,E,",
		},

		"types dependencies must be ignored": {
			`package pkg

			var A = int(20)`,
			"A,",
		},

		"imported symbols dependencies must be ignored": {
			`package pkg

			import "x"

			var A = B
			var B = x.F()`,
			"B,A,",
		},

		"functions, constants and variables": {
			`package main

			func   F1()    {}
			var    A       = E
			const  C1      = C2
			const  C2      = 11
			var    E       = C1 + C2
			func   F2()    {}
			var    D       = 2
			func   main()  {}
			`,
			"C2,C1,E,A,D,F1,F2,main,",
		},

		"function assignment": {
			`package pkg

			var E = A + B
			var A, B, C = F()
			var D = A

			func F() {}`,
			"A,B,C,E,D,F,",
		},

		"complex test": {
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
			}`, "m,value,ok,notOk,doubleValue,k,main,",
		},

		"two constants where first depends on second": {
			`package main

			import "fmt"

			const A = B
			const B = 20

			func main() {
				fmt.Println(A)
				fmt.Println(B)
			}`, "B,A,main,",
		},
		"two blank identifiers": {
			`package main

			var _ = A
			var _ = 10
			var A = 20

			func main() {}
			`, "_,A,_,main,",
		},
		"one alias declaration": {
			`package main

			type A = int

			func main() {}`,
			"A,main,",
		},
		"two alias declarations": {
			`package main

			type A = int
			type B = string

			func main() {}`,
			"A,B,main,",
		},
		"two (dependent) alias declarations": {
			`package main

			type A = int
			type B = A

			func main() {}`,
			"A,B,main,",
		},
		"two (dependent) alias declarations where first depends on second": {
			`package main

			type B = A
			type A = int

			func main() {}`,
			"A,B,main,",
		},
		"alias declarations with funcs, vars and consts": {
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
			"A,B,C,C2,V1,V2,F,main,",
		},
		"package variable with a type alias declaration as type": {
			`package main

			type T = int

			var v1 T = 10

			func main() {
			}
			`,
			"T,v1,main,",
		},
	}
	for name, cas := range cases {
		t.Run(name, func(t *testing.T) {
			tree, err := ParseSource([]byte(cas.src), false, false)
			deps := AnalyzeTree(tree, ProgramSyntax)
			if err != nil {
				t.Fatalf("parsing error: %s", err)
			}
			pkg := tree.Nodes[0].(*ast.Package)
			sortDeclarations(pkg, deps)
			got := ""
			for _, d := range pkg.Declarations {
				switch d := d.(type) {
				case *ast.Var:
					for _, left := range d.Lhs {
						got += left.Name + ","
					}
				case *ast.Const:
					got += d.Lhs[0].Name + ","
				case *ast.Func:
					got += d.Ident.Name + ","
				case *ast.TypeDeclaration:
					got += d.Identifier.Name + ","
				}
			}
			if cas.expected != got {
				t.Fatalf("expecting %q, got %q", cas.expected, got)
			}
		})
	}
}
