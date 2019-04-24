// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"text/tabwriter"

	"scrigo/parser"
)

var exprTests = map[string]interface{}{
	// Composite literals - slices.
	`[]int{}`:               []int{},
	`[]int{1, 2}`:           []int{1, 2},
	`[]int{0: 1, 1: 3}`:     []int{0: 1, 1: 3},
	`[]int{0: 1, 5: 3}`:     []int{0: 1, 5: 3},
	`[]string{}`:            []string{},
	`[]string{"a", "b"}`:    []string{"a", "b"},
	`[]string{"a", 5: "b"}`: []string{"a", 5: "b"},
	`[]float64{5, 12, 0}`:   []float64{5, 12, 0},
	// `[]float64{5.6, 12.3, 0.4}`: []float64{5.6, 12.3, 0.4},

	// Builtin 'make'.
	`make([]int, 0, 0)`:       []int{},
	`make([]int, 0, 5)`:       []int{},
	`make([]int, 3, 5)`:       []int{0, 0, 0},
	`make([]string, 1, 5)`:    []string{""},
	`make(map[string]int, 1)`: map[string]int{},
	`make([]float64, 3, 3)`:   []float64{0, 0, 0},
}

func TestVMExpressions(t *testing.T) {
	DebugTraceExecution = false
	for src, expected := range exprTests {
		t.Run(src, func(t *testing.T) {
			r := parser.MapReader{"/test.go": []byte("package main; func main() { a := " + src + "; _ = a }")}
			comp := NewCompiler(r, goPackages)
			main, err := comp.CompileFunction()
			if err != nil {
				t.Errorf("test %q, compiler error: %s", src, err)
				return
			}
			vm := New()
			_, err = vm.Run(main)
			if err != nil {
				t.Errorf("test %q, execution error: %s", src, err)
				return
			}
			kind := reflect.TypeOf(expected).Kind()
			var got interface{}
			switch kind {
			case reflect.Int, reflect.Bool, reflect.Int64:
				got = vm.int(1)
			case reflect.Float32, reflect.Float64:
				got = vm.float(1)
			case reflect.String:
				got = vm.string(1)
			case reflect.Slice, reflect.Map:
				got = vm.general(1)
			default:
				panic("bug")
			}
			if !reflect.DeepEqual(expected, got) {
				t.Errorf("test %q, expected %v (type %T), got %v (type %T)", src, expected, expected, got, got)
			}
		})
	}
}

// TODO (Gianluca): currently unable to test:
//
//   a | b         --> parsing error
//   a := 1234
//   a := new(int)
//   a := []int{1,2,3,4}
//   func f(int) {

// TODO (Gianluca): test fmt.Printf

// reg represents a register and it's used in tests only.
type reg struct {
	typ   Type
	r     int8
	value interface{}
}

var stmtTests = []struct {
	name         string
	src          string
	disassembled []string
	registers    []reg
	output       string
}{

	{"Recycling of registers",
		`
		package main

		import "fmt"

		func main() {
			a := 10
			b := a + 32
			c := 88
			fmt.Println(a, b, c)
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(10)},
			{TypeInt, 2, int64(42)},
			{TypeInt, 3, int64(88)},
		},
		"10 42 88\n"},

	// Assignments.

	{"Single assignment",
		`
		package main

		func main() {
			var a int
			var c string

			a = 10
			c = "hi"

			_ = a
			_ = c

			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(10)}, // a
			{TypeString, 1, "hi"},   // c
		}, ""},

	{"Multiple assignment",
		`
		package main

		func main() {
			a, b := 6, 7
			_, _ = a, b
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(6)}, // a
			{TypeInt, 2, int64(7)}, // b
		}, ""},

	{"Assignment with constant int value (addition)",
		`
		package main

		func main() {
			a := 4 + 5;
			_ = a
			return
		}
		`,
		[]string{
			`Package main`,
			``,
			`Func main()`,
			`	// regs(1,0,0,0)`,
			`	MoveInt 9 R1`,
			`	Return`,
		}, nil, ""},

	{"Assignment with addition value (non constant)",
		`
		package main

		func main() {
			a := 5
			b := 10
			c := a + 2 + b + 30
			_ = c
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(5)},  // a
			{TypeInt, 2, int64(10)}, // b
			{TypeInt, 3, int64(47)}, // c
		}, ""},

	{"Assignment with math expression (non constant)",
		`
		package main

		func main() {
			var a, b, c int

			a = 3
			b = 5
			c = a + 4*b

			_ = c
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(3)},  // a
			{TypeInt, 2, int64(5)},  // b
			{TypeInt, 3, int64(23)}, // c
		}, ""},

	{"Function call assignment (2 to 1) - Native function with two return values",
		`
		package main

		import (
			"fmt"
			"testpkg"
		)

		func main() {
			a := 5
			b, c := testpkg.Pair()
			fmt.Println(a, b, c)
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(5)},  // a
			{TypeInt, 2, int64(42)}, // b
			{TypeInt, 3, int64(33)}, // c
		}, "5 42 33\n"},

	{"Addition (+=) and subtraction (-=) assignments",
		`
		package main

		func main() {
			var a, b, c, d int

			a = 10
			b = a
			a += 40
			c = a
			a -= 5
			d = a

			_ = b
			_ = c
			_ = d
		}
		`,
		[]string{
			`Package main`,
			``,
			`Func main()`,
			`		// regs(4,0,0,0)`,
			`		MoveInt 0 R1`,
			`		MoveInt 0 R2`,
			`		MoveInt 0 R3`,
			`		MoveInt 0 R4`,
			`		MoveInt 10 R1`,
			`		MoveInt R1 R2`,
			`		AddInt R1 40 R1`,
			`		MoveInt R1 R3`,
			`		SubInt R1 5 R1`,
			`		MoveInt R1 R4`,
		},
		[]reg{
			{TypeInt, 1, int64(45)}, // a
			{TypeInt, 2, int64(10)}, // b
			{TypeInt, 3, int64(50)}, // c
			{TypeInt, 4, int64(45)}, // d
		}, ""},

	{"Variable declaration with 'var'",
		`
		package main

		func pair() (int, string) {
			return 10, "zzz"
		}

		func main() {
			var a, b = 1, 2
			var c, d int
			var e, f = pair()

			_ = a
			_ = b
			_ = c
			_ = d
			_ = e
			_ = f
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(1)},  // a
			{TypeInt, 2, int64(2)},  // b
			{TypeInt, 3, int64(0)},  // c
			{TypeInt, 4, int64(0)},  // d
			{TypeInt, 5, int64(10)}, // e
			// {TypeString, 6, "zzz"},  // f // TODO (Gianluca):
		},
		"",
	},

	{"Slice index assignment",
		`
		package main

		import "fmt"

		func main() {
			var s1 []int
			var s2 []string

			s1 = []int{1, 2, 3}
			s1[0] = 2

			s2 = []string{"a", "b", "c", "d"}
			s2[2] = "d"

			fmt.Println(s1)
			fmt.Println(s2)
			return
		}
		`, nil, []reg{
			{TypeIface, 1, []int{2, 2, 3}},               // s1
			{TypeIface, 2, []string{"a", "b", "d", "d"}}, // s2
		}, "[2 2 3]\n[a b d d]\n"},

	// Expressions - composite literals.

	{"Empty int slice composite literal",
		`
		package main

		import "fmt"

		func main() {
			a := []int{};
			fmt.Println(a)
		}
		`,
		nil,
		[]reg{
			// {TypeIface, 1, []int{}}, // a
		}, "[]\n"},

	{"Empty string slice composite literal",
		`
		package main

		func main() {
			a := []string{};
			_ = a
		}
		`,
		nil,
		[]reg{
			{TypeIface, 1, []string{}}, // a
		}, ""},

	{"Empty byte slice composite literal",
		`
		package main

		func main() {
			var a []int
			var b []byte

			a = []int{};
			b = []byte{};

			_ = a
			_ = b
			return
		}
		`,
		nil,
		[]reg{
			{TypeIface, 1, []int{}},  // a
			{TypeIface, 2, []byte{}}, // b
		}, ""},

	{"Empty map composite literal",
		`
		package main

		func main() {
			m := map[string]int{}
			_ = m
		}
		`,
		nil,
		[]reg{
			{TypeIface, 1, map[string]int{}},
		}, ""},

	// Expressions - function literals.

	{"Function literal definition - (0 in, 0 out)",
		`
		package main

		func main() {
			a := 10
			f := func() {
				a := 10
				_ = a
			}
			b := 20
			_ = a
			_ = b
			_ = f
		} 
		`,
		[]string{
			`Package main`,
			``,
			`Func main()`,
			`		// regs(2,0,0,2)`,
			`		MoveInt 10 R1`,
			`		Func R2 ()`,
			`			// regs(0,0,0,0)`,
			`			MoveInt 10 R1`,
			`		Move R2 R1`,
			`		MoveInt 20 R2`,
		}, nil, ""},

	{"Function literal definition - (0 in, 1 out)",
		`
		package main

		func main() {
			f := func() int {
				a := 20
				return a
			}
			_ = f
		}
		`,
		[]string{
			`Package main`,
			``,
			`Func main()`,
			`		// regs(0,0,0,2)`,
			`		Func R2 () (1 int)`,
			`			// regs(0,0,0,0)`,
			`			MoveInt 20 R2`,
			`			MoveInt R2 R1`,
			`			Return`,
			`		Move R2 R1`,
		}, nil, ""},

	{"Function literal definition - (1 in, 0 out)",
		`
		package main

		func main() {
			f := func(a int) {
				b := a + 20
				_ = b
			}
			_ = f
		}
		`,
		[]string{
			`Package main`,
			``,
			`Func main()`,
			`		// regs(0,0,0,2)`,
			`		Func R2 (1 int)`,
			`			// regs(0,0,0,0)`,
			`			MoveInt R1 R4`,
			`			AddInt R4 20 R3`,
			`			MoveInt R3 R2`,
			`		Move R2 R1`,
		}, nil, ""},

	// Expressions - conversions.

	{"Converting from int to string",
		`
		package main

		func main() {
			var a int
			var b string

			a = 97
			b = string(a)

			_ = b
			return
		}
		`,
		nil,
		[]reg{
			// TODO (Gianluca):
			// {TypeInt, 1, int64(97)}, // a
			// {TypeString, 1, "a"},    // b
		}, ""},

	{"Converting from int to interface{}",
		`
		package main

		func main() {
			a := 97
			b := interface{}(a)
			_ = b
			return
		}
		`,
		nil,
		[]reg{
			{TypeIface, 1, int64(97)},
		}, ""},

	// Expressions - boolean.

	{"Constant boolean",
		`
		package main

		func main() {
			var a, b int
			var c, d bool
			a = 1
			b = 2
			c = false
			d = 4 < 5
			_ = a
			_ = b
			_ = c
			_ = d
			return
		}

		`,
		[]string{
			`Package main`,
			``,
			`Func main()`,
			`		// regs(4,0,0,0)`,
			`           MoveInt 0 R1`,
			`           MoveInt 0 R2`,
			`           MoveInt 0 R3`,
			`           MoveInt 0 R4`,
			`           MoveInt 1 R1`,
			`           MoveInt 2 R2`,
			`           MoveInt 0 R3`,
			`           MoveInt 1 R4`,
			`		Return`,
		}, nil, ""},

	{"Comparison operators",
		`
		package main

		func main() {
			var a, b int
			var l, g, le, ge, e, ne bool

			a = 1
			b = 2

			l = a < b
			g = a > b
			le = a <= b
			ge = a >= b
			e = a == b
			ne = a != b

			_ = l
			_ = g
			_ = le
			_ = ge
			_ = e
			_ = ne
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(1)}, // a
			{TypeInt, 2, int64(2)}, // b
			{TypeInt, 3, int64(1)}, // l   =  a < b
			{TypeInt, 4, int64(0)}, // g   =  a > b
			{TypeInt, 5, int64(1)}, // le  =  a <= b
			{TypeInt, 6, int64(0)}, // ge  =  a >= b
			{TypeInt, 7, int64(0)}, // e   =  a == b
			{TypeInt, 8, int64(1)}, // ne  =  a != b
		}, ""},

	{"Logic AND and OR operators",
		`
		package main

		func main() {
			T := true
			F := false

			var a, b, c, d, e, f, g, h bool

			a = T && T
			b = T && F
			c = F && T
			d = F && F

			e = T || T
			f = T || F
			g = F || T
			h = F || F

			_ = a
			_ = b
			_ = c
			_ = d
			_ = e
			_ = f
			_ = g
			_ = h
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(1)},  // T
			{TypeInt, 2, int64(0)},  // F
			{TypeInt, 3, int64(1)},  // a := T && T
			{TypeInt, 4, int64(0)},  // b := T && F
			{TypeInt, 5, int64(0)},  // c := F && T
			{TypeInt, 6, int64(0)},  // d := F && F
			{TypeInt, 7, int64(1)},  // e := T || T
			{TypeInt, 8, int64(1)},  // f := T || F
			{TypeInt, 9, int64(1)},  // g := F || T
			{TypeInt, 10, int64(0)}, // h := F || F
		}, ""},

	{"Short-circuit evaluation",
		`
		package main

		func p() bool {
			panic("error")
		}

		func main() {
			T := true
			F := false

			v := false

			v = F && p()
			v = T || p()
			v = (F || F) && p()
			v = (T && T) || p()
			v = F && p() && p()
			v = F || T || p()
			v = T && F && p()

			_ = v
		}
		`, nil, nil, ""},

	{"Operator ! (not)",
		`
		package main

		func main() {
			T := true
			F := false

			var a, b, c, d bool

			a = !T
			b = !F
			c = !!F
			d = !!T

			_ = a
			_ = b
			_ = c
			_ = d
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(1)}, // T
			{TypeInt, 2, int64(0)}, // F
			{TypeInt, 3, int64(0)}, // a = !T
			{TypeInt, 4, int64(1)}, // b = !F
			{TypeInt, 5, int64(0)}, // c = !!F
			{TypeInt, 6, int64(1)}, // d = !!T
		}, ""},

	// Expressions - misc.

	{"Number negation (unary operator '-')",
		`
		package main

		import "fmt"

		func main() {
			var a, b int

			a = 45
			b = -a

			fmt.Print(b)
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(45)},  // a
			{TypeInt, 2, int64(-45)}, // b
		},
		"-45"},

	{"Go functions as expressions",
		`
		package main

		import "fmt"

		func main() {
			f := fmt.Println
			print(f)
		}
		`, nil, nil, ""},

	{"String concatenation (constant)",
		`
		package main

		import "fmt"

		func main() {
			a := "s";
			b := "ee" + "ff";

			fmt.Println(a)
			fmt.Print(b)
		}
		`,
		nil,
		[]reg{
			{TypeString, 1, "s"},    // a
			{TypeString, 3, "eeff"}, // b
		}, "s\neeff"},

	{"String concatenation (non constant)",
		`
		package main

		import "fmt"

		func main() {
			var s1, s2, s3 string

			s1 = "hello"
			s2 = "world"
			s3 = s1 + " " + s2
			fmt.Print(s3)
		}
		`,
		nil,
		[]reg{
			{TypeString, 1, "hello"},       // s1
			{TypeString, 2, ""},            // (empty string used in s1 initialization)
			{TypeString, 3, "world"},       // s2
			{TypeString, 4, ""},            // (empty string used in s2 initialization)
			{TypeString, 5, "hello world"}, // s3
		}, "hello world"},

	// // TODO (Gianluca): add slice, map and array indexing tests.
	// {"Indexing",
	// 	`
	// 		package main

	// 		func main() {
	// 			var a string
	// 			var b byte

	// 			a = "abcde"
	// 			b = a[2]

	// 			_ = b
	// 			return
	// 		}
	// 		`,
	// 	nil,
	// 	nil,
	// 	"",
	// },

	// Type assertion.

	{"Type assertion as expression (single value context)",
		`
		package main

		func main() {
			i := interface{}(int64(5))
			a := 7 + i.(int64)
			_ = a
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(5)},  // argument to int64
			{TypeInt, 2, int64(12)}, // a
		},
		"",
	},

	{"Type assertion - Int on interface{}",
		`
		package main

		func main() {
			a := interface{}(10)
			n, ok := a.(int)

			_ = n
			_ = ok
			return
		}
		`,
		nil,
		[]reg{}, ""},
	// Statements - If.

	{"If statement with else",
		`
		package main

		func main() {
			a := 10
			c := 0
			if a > 5 {
				c = 1
			} else {
				c = 2
			};
			_ = c
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(10)}, // a
			{TypeInt, 2, int64(1)},  // c
		}, ""},

	{"If statement with else",
		`
		package main

		func main() {
			a := 10
			c := 0
			if a <= 5 {
				c = 1
			} else {
				c = 2
			}
			_ = c
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(10)}, // a
			{TypeInt, 2, int64(2)},  // c
		}, ""},

	{"If with init assignment",
		`
		package main

		func main() {
			c := 0
			if x := 1; x == 1 {
				c = 1
			} else {
				c = 2
			}
			_ = c
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(1)}, // c
		}, ""},

	// Statements - For.

	{"For statement",
		`
		package main

		func main() {
			  sum := 0
		        i := 0
			  for i = 0; i < 10; i++ {
				    sum = sum + 2
			  }
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(20)}, // sum
			{TypeInt, 2, int64(10)}, // i
		}, ""},

	// Statements - Switch.

	{"Switch statement",
		`
		package main

		func main() {
			a := 0
			switch 1 + 1 {
			case 1:
				a = 10
			case 2:
				a = 20
			case 3:
				a = 30
			}
			_ = a
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(20)},
		}, ""},

	{"Switch statement with fallthrough",
		`
		package main

		func main() {
			a := 0
			switch 1 + 1 {
			case 1:
				a = 10
			case 2:
				a = 20
				fallthrough
			case 3:
				a = 30
			}
			_ = a
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(30)},
		}, ""},

	{"Switch statement with default",
		`
		package main

		func main() {
			a := 0
			switch 10 + 10 {
			case 1:
				a = 10
			case 2:
				a = 20
			default:
				a = 80
			case 3:
				a = 30
			}
			_ = a
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(80)},
		}, ""},

	{"Switch statement with default and fallthrough",
		`
		package main

		func main() {
			a := 0
			switch 2 + 2 {
			case 1:
				a = 10
			default:
				a = 80
				fallthrough
			case 2:
				a = 1
				fallthrough
			case 3:
				a = 30
			case 40:
				a = 3
			}
			_ = a
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(30)},
		}, ""},

	// Statements - Type switch.

	{"Type switch statement",
		`
		package main

		import "fmt"

		func main() {
			i := interface{}(int64(5))
			v := 0
			switch i.(type) {
			case string:
				v = 10
			case int64:
				v = 20
			default:
				v = 30
			}

			fmt.Print(v)
			return
		}
		`,
		nil,
		[]reg{
			// {TypeInt, 2, int64(20)}, // v
		}, "20"},

	// Function literal calls.

	{"Function literal call (0 in, 0 out)",
		`
		package main

		func main() {
			f := func() {}
			f()
		}
		`,
		[]string{
			`Package main`,
			``,
			`Func main()`,
			`	// regs(0,0,0,2)`,
			`	Func R2 ()`,
			`		// regs(0,0,0,0)`,
			`	Move R2 R1`,
			`	CallIndirect R1    // Stack shift: 0, 0, 0, 2`,
		}, nil, ""},

	{"Function literal call (0 in, 1 out)",
		`
		package main

		func main() {
			a := 0
			f := func() int {
				a := 10
				b := 15
				return a + b
			}
			a = f()
			_ = a
			return
		}
		`,
		nil, []reg{
			{TypeInt, 1, int64(25)}, // a
		}, ""},

	{"Function literal call (1 in, 1 out)",
		`
		package main

		func main() {
			a := 41
			inc := func(a int) int {
				b := a
				b++
				return b
			}
			a = inc(a)
			return
		}
		`, nil,
		[]reg{
			{TypeInt, 1, int64(42)}, // a
		}, ""},

	{"Function literal call - Shadowing package level function",
		`
		package main

		func f() {
			panic("")
		}

		func main() {
			f := func() {
				// No panics.
			}
			f()
			return
		}
		`, nil, nil, ""},

	// Scrigo function calls.

	{"Package function call",
		`
		package main

		func a() {

		}

		func main() {
			a()
			return
		}
		`,
		nil, nil, ""},

	{"Package function 'inc'",
		`
		package main

		func inc(n int) int {
			return n+1
		}

		func main() {
			var a, res, b, c int

			a = 2
			res = inc(8)
			b = 10
			c = a + b + res

			_ = c
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(2)},  // a
			{TypeInt, 2, int64(9)},  // res
			{TypeInt, 3, int64(10)}, // b
			{TypeInt, 4, int64(21)}, // c
		}, ""},

	{"Package function with one return value",
		`
		package main

		func five() int {
			return 5
		}

		func main() {
			a := five()
			_ = a
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(5)},
		}, ""},

	{"Package function with many return values (same type)",
		`
		package main

		func pair() (int, int) {
			return 42, 33
		}

		func main() {
			a := 2
			b, c := pair()
			d, e := 11, 12
			_ = a + b + c + d + e
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(2)},  // a
			{TypeInt, 2, int64(42)}, // b
			{TypeInt, 3, int64(33)}, // c
			{TypeInt, 6, int64(11)}, // d
			{TypeInt, 7, int64(12)}, // e
		}, ""},

	// TODO (Gianluca): find bug.
	// {"Package function with many return values (different types)",
	// 	`
	// 	package main

	// 	func pair() (int, float64) {
	// 		return 42, 33.0
	// 	}

	// 	func main() {
	// 		a := 2
	// 		b, c := pair()
	// 		d, e := 11, 12
	// 		_ = a + b + d + e
	// 		_ = c
	// 		return
	// 	}
	// 	`,
	// 	nil,
	// 	[]reg{
	// 		{TypeInt, 1, int64(2)},        // a
	// 		{TypeInt, 2, int64(42)},       // b
	// 		{TypeFloat, 1, float64(33.0)}, // c
	// 		{TypeInt, 4, int64(11)},       // d
	// 		{TypeInt, 5, int64(12)},       // e
	// 	}, ""},

	{"Package function with one parameter (not used)",
		`
		package main

		func f(a int) {
			return
		}

		func main() {
			a := 2
			f(3)
			b := 10
			c := a + b
			_ = c
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(2)},  // a
			{TypeInt, 3, int64(10)}, // b
			{TypeInt, 4, int64(12)}, // c
		}, ""},

	{"Package function with arguments that need evaluation",
		`
		package main

		import "fmt"

		func f(a, b, c int) {
			fmt.Println("a, b, c: ", a, b, c)
			return
		}

		func g(slice []int) {
			l := len(slice) // TODO (Gianluca): move inline inside fmt.Println
			fmt.Println(slice, "has len", l)
			return
		}

		func main() {
			a := 3
			b := 2
			c := 5
			f(a, (b*2)+a-a, c)
			g([]int{a,b,c})
		}
		`, nil, nil,
		"a, b, c:  3 4 5\n[3 2 5] has len 3\n"},

	// {"Variadic package functions",
	// 	`
	// 	package main

	// 	import "fmt"

	// 	func f(a ...int) {
	// 		fmt.Println("variadic:", a)
	// 		return
	// 	}

	// 	func g(a, b string, c... int) {
	// 		fmt.Println("strings:", a, b)
	// 		fmt.Println("variadic:", c)
	// 		return
	// 	}

	// 	func h(a string, b... string) int {
	// 		fmt.Println("a:", a)
	// 		return len(b)
	// 	}

	// 	func main() {
	// 		f(1, 2, 3)
	// 		g("x", "y", 3, 23, 11, 12)
	// 		fmt.Println("h:", h("a", "b", "c", "d"))
	// 	}
	// 	`, nil, nil,
	// 	"variadic: [1 2 3]\nstrings: x y\nvariadic: [3 23 11 12]\na: a\nh: 3\n"},

	// Builtin function calls.

	{"Builtin len (with a constant argument)",
		`
		package main

		func main() {
			a := len("abc");
			_ = a
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(3)}, // a
		}, ""},

	{"Builtin len (with a variable argument)",
		`
		package main

		func main() {
			a := "a string"
			b := len(a)
			_ = b
		}
		`,
		nil,
		[]reg{
			{TypeString, 1, "a string"}, // a
			{TypeInt, 1, int64(8)},      // b
		}, ""},

	{"Builtin print",
		`
		package main

		func main() {
			print(42)
			print("hello")
			a := 10
			print(a)
		}
		`,
		[]string{
			`Package main`,
			``,
			`Func main()`,
			`    // regs(3,0,2,3)`,
			`    Move fromInt 42 R1`,
			`    Print R1`,
			`    MoveString "hello" R1`,
			`    Move fromString R1 R2`,
			`    Print R2`,
			`    MoveInt 10 R2`,
			`    Move fromInt R2 R3`,
			`    Print R3`,
		}, nil, ""},

	{"Builtin make - map",
		`
		package main

		func main() {
			var m map[string]int
			m = make(map[string]int, 2)
			_ = m
		}
		`,
		nil,
		[]reg{
			{TypeIface, 1, map[string]int{}},
		}, ""},

	{"Builtin copy",
		`
		package main

		func main() {
			var src, dst []int
			var n int

			src = []int{10, 20, 30}
			dst = []int{}
			n = copy(dst, src)

			_ = n
			return
		}
		`, nil, nil, "",
	},

	// Native (Go) function calls.

	{"Native function call (0 in, 0 out)",
		`
		package main

		import "testpkg"

		func main() {
			testpkg.F00()
			return
		}
		`,
		[]string{
			`Package main`,
			``,
			`Import "testpkg"`,
			``,
			`Func main()`,
			`	// regs(0,0,0,0)`,
			`	CallFunc testpkg.F00    // Stack shift: 0, 0, 0, 0`,
			`     Return`,
		},
		nil, ""},

	{"Native function call (0 in, 1 out)",
		`
		package main

		import "testpkg"

		func main() {
			v := testpkg.F01()
			_ = v
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(40)},
		}, ""},

	{"Native function call (1 in, 0 out)",
		`
		package main

		import "testpkg"

		func main() {
			testpkg.F10(50)
			return
		}
		`,
		nil, nil, ""},

	{"Native function call (1 in, 1 out)",
		`
		package main

		import "testpkg"

		func main() {
			a := 2
			a = testpkg.F11(9)
			_ = a
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(42)},
		}, ""},

	{"Native function call (2 in, 1 out) (with surrounding variables)",
		`
		package main

		import "testpkg"

		func main() {
			var a, b, e, c, d int

			a = 2 + 1
			b = 3 + 10
			e = 4
			c = testpkg.Sum(a, b)
			d = c

			_ = d
			_ = e
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(3)},  // a
			{TypeInt, 2, int64(13)}, // b
			{TypeInt, 3, int64(4)},  // e
			{TypeInt, 4, int64(16)}, // c
			{TypeInt, 5, int64(16)}, // d // TODO (Gianluca): d should be allocated in register 5, which is no longer used by function call.
		}, ""},

	{"Native function call of StringLen",
		`
		package main

		import "testpkg"

		func main() {
			a := testpkg.StringLen("zzz")
			_ = a
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(3)},
		}, ""},

	{"Native function call of fmt.Println",
		`
		package main

		import "fmt"

		func main() {
			fmt.Println("hello, world!")
			fmt.Println(42)
			a := "hi!"
			fmt.Println(a)
			fmt.Println(1, 2, 3)
			fmt.Println(a, a, []int{3,4,5})
		}
		`, nil, nil, "hello, world!\n42\nhi!\n1 2 3\nhi! hi! [3 4 5]\n"},

	// Native (Go) variables.

	{"Reading a native int variable",
		`
		package main

		import "testpkg"

		func main() {
			a := testpkg.A
			testpkg.PrintInt(a)
		}
		`, nil, nil,
		"20"},

	{"Writing a native int variable",
		`
		package main

		import "testpkg"

		func main() {
			testpkg.PrintInt(testpkg.B)
			testpkg.B = 7
			testpkg.PrintString("->")
			testpkg.PrintInt(testpkg.B)
		}
		`, nil, nil,
		"42->7"},

	// Complex tests.

	{"Scrigo function 'repeat(string, int) string'",
		`
		package main

		func repeat(s string, n int) string {
			out := ""
			for i := 0; i < n; i++ {
				out = out + s
			}
			return out
		}

		func main() {
			var b, d int
			var s1, f, s2 string

			b = 1
			d = b + 2
			s1 = repeat("z", 4)
			f = "hello"
			s2 = repeat(f, d)

			_ = s1 + s2
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(1)},             // b
			{TypeInt, 2, int64(3)},             // d
			{TypeString, 1, "zzzz"},            // s1
			{TypeString, 5, "hellohellohello"}, // s2  // TODO (Gianluca): should be register 2, not 5.
		}, ""},

	{"Slice of int slices assignment",
		`
		package main

		func main() {
			var ss [][]int
			
			ss = [][]int{
				[]int{10, 20},
				[]int{},
				[]int{30, 40, 50},
			}
			ss[1] = []int{25, 26}

			_ = ss
			return
		}
		`,
		nil,
		[]reg{
			{TypeIface, 1, [][]int{[]int{10, 20}, []int{25, 26}, []int{30, 40, 50}}},
		},
		""},

	{"Multiple native function calls",
		`
		package main

		import "testpkg"

		func main() {
			var a, b, c int
			
			a = testpkg.Inc(5)
			b = testpkg.Dec(a)
			c = testpkg.Inc(0) + testpkg.Dec(testpkg.Dec(testpkg.Dec(b)))

			_, _, _ = a, b, c
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(6)}, // a
			{TypeInt, 2, int64(5)}, // b
			{TypeInt, 3, int64(3)}, // c
		}, ""},

	{"Native function 'Swap'",
		`
		package main

		import "testpkg"

		func main() {
			var i1, i2 int
			var s1, s2 string

			i1 = 3
			s1 = "hey"

			s1, i1 = testpkg.Swap(i1, s1)
			s2, i2 = testpkg.Swap(i1, s1)

			_ = s2
			_ = i2
		}
		`, nil, []reg{
			{TypeInt, 1, int64(3)},
			{TypeInt, 2, int64(3)},
			{TypeString, 1, "hey"},
			{TypeString, 3, "hey"},
		}, ""},

	{"Many Scrigo functions (swap, sum, fact)",
		`
		package main

		import "testpkg"

		func swap(a, b int) (int, int) {
			return b, a
		}

		func sum(a, b, c int) int {
			s := 0
			s += a
			s += b
			s += c
			return s
		}

		func fact(n int) int {
			switch n {
			case 0:
				return 1
			case 1:
				return 1
			default:
				return n * fact(n-1)
			}
		}

		func main() {
			var a, b, c, d, e, f int

			a = 2
			b = 6
			a, b = swap(a, b)
			c, d = swap(a, b)
			e = sum(a, b, sum(c, d, 0))
			f = fact(d+2) + sum(a, b, c)

			_ = a
			_ = b
			_ = c
			_ = d
			_ = e
			_ = f

			testpkg.PrintString("a:")
			testpkg.PrintInt(a)
			testpkg.PrintString(",")
			testpkg.PrintString("b:")
			testpkg.PrintInt(b)
			testpkg.PrintString(",")
			testpkg.PrintString("c:")
			testpkg.PrintInt(c)
			testpkg.PrintString(",")
			testpkg.PrintString("d:")
			testpkg.PrintInt(d)
			testpkg.PrintString(",")
			testpkg.PrintString("e:")
			testpkg.PrintInt(e)
			testpkg.PrintString(",")
			testpkg.PrintString("f:")
			testpkg.PrintInt(f)
			testpkg.PrintString(",")

			return
		}
		`,
		nil,
		nil,
		"a:6,b:2,c:2,d:6,e:16,f:40330,",
	},
}

func TestVM(t *testing.T) {
	DebugTraceExecution = false
	for _, cas := range stmtTests {
		t.Run(cas.name, func(t *testing.T) {
			registers := cas.registers
			r := parser.MapReader{"/test.go": []byte(cas.src)}
			comp := NewCompiler(r, goPackages)
			main, err := comp.CompileFunction()
			if err != nil {
				t.Errorf("test %q, compiler error: %s", cas.name, err)
				return
			}
			backupStdout := os.Stdout
			vm := New()
			backupStderr := os.Stderr
			reader, writer, err := os.Pipe()
			if err != nil {
				panic(err)
			}
			os.Stdout = writer
			os.Stderr = writer
			defer func() {
				os.Stdout = backupStdout
				os.Stderr = backupStderr
			}()
			out := make(chan string)
			wg := new(sync.WaitGroup)
			wg.Add(1)
			go func() {
				var buf bytes.Buffer
				wg.Done()
				io.Copy(&buf, reader)
				out <- buf.String()
			}()
			wg.Wait()
			_, err = vm.Run(main)
			if err != nil {
				t.Errorf("test %q, execution error: %s", cas.name, err)
				return
			}
			writer.Close()

			output := <-out

			// Tests if disassembler output matches.

			// TODO (Gianluca): to review.
			if false && cas.disassembled != nil {
				assembler, err := Disassemble(main)
				if err != nil {
					t.Errorf("test %q, disassemble error: %s", cas.name, err)
					return
				}
				got := assembler["main"]
				gotLines := []string{}
				for _, line := range strings.Split(strings.TrimSpace(got), "\n") {
					gotLines = append(gotLines, line)
				}
				if diff := equal(cas.disassembled, gotLines); diff >= 0 {
					if !testing.Verbose() {
						t.Errorf("disassembler output doesn't match for test %q (run tests in verbose mode for further details)", cas.name)
					} else {
						out := &bytes.Buffer{}
						const padding = 3
						w := tabwriter.NewWriter(out, 0, 0, padding, ' ', tabwriter.Debug)
						fmt.Fprintf(w, "expected\t  got\t\n")
						fmt.Fprintf(w, "--------\t  ---\t\n")
						longest := len(cas.disassembled)
						if len(gotLines) > longest {
							longest = len(gotLines)
						}
						for i := 0; i < longest; i++ {
							e := " "
							g := " "
							if i <= len(cas.disassembled)-1 {
								e = cas.disassembled[i]
							}
							if i <= len(gotLines)-1 {
								g = gotLines[i]
							}
							e = tabsToSpaces(e)
							g = tabsToSpaces(g)
							if diff == i+1 {
								fmt.Fprintf(w, "%s\t  %s\t <<< difference here\n", e, g)
							} else {
								fmt.Fprintf(w, "%s\t  %s\t\n", e, g)
							}
						}
						w.Flush()
						t.Errorf("test %q:\n%s", cas.name, out.String())
					}
				}
			}

			// Tests if registers match.

			for _, reg := range registers {
				var got interface{}
				switch reg.typ {
				case TypeFloat:
					got = vm.float(reg.r)
				case TypeIface:
					got = vm.general(reg.r)
				case TypeInt:
					got = vm.int(reg.r)
				case TypeString:
					got = vm.string(reg.r)
				}
				if !reflect.DeepEqual(reg.value, got) {
					t.Errorf("test %q, register %s[%d]: expecting %#v (type %T), got %#v (type %T)", cas.name, reg.typ, reg.r, reg.value, reg.value, got, got)
				}
			}

			// // Tests if output matches.
			if cas.output != output {
				t.Errorf("test %q: expecting output %q, got %q", cas.name, cas.output, output)
			}
		})
	}
}

// equal checks if expected is equal to got, and returns:
// -1 if they are equal;
// 0  if they have different length;
// n  if they differ from (at least) line n;
func equal(expected, got []string) int {
	if len(expected) != len(got) {
		return 0
	}
	for i, expectedLine := range expected {
		if strings.Join(strings.Fields(expectedLine), " ") != strings.Join(strings.Fields(got[i]), " ") {
			return i + 1
		}
	}
	return -1
}

func tabsToSpaces(s string) string {
	return strings.ReplaceAll(s, "\t", "    ")
}

// oneLine puts src in just one line, returning an (as much as possibile) human
// readable representation.
func oneLine(src string) string {
	src = strings.Join(strings.Split(src, "\n"), " ")
	src = strings.ReplaceAll(src, "\t", "")
	return src
}

// NoTestMakeExpressionTests renders the list of tests.
//
// Usage:
//
//   1) Be sure TestVM passes 100%
//   2) Rename this test removing "No" from it's name
//   3) Run this test and copy result from stdout
//   4) Paste it in this file
//   5) Use your diff tool to discard unwanted changes (eg. registers overwriting)
//
// TODO (Gianluca):
// func NoTestMakeExpressionTests(t *testing.T) {
// 	out := strings.Builder{}
// 	out.WriteString("\n")
// 	for _, cas := range stmtTests {
// 		r := parser.MapReader{"/test.go": []byte(cas.src)}
// 		comp := NewCompiler(r, goPackages)
// 		pkg, err := comp.CompileFunction()
// 		if err != nil {
// 			panic(fmt.Errorf("unexpected error: source: %q, compiler error: %s", oneLine(cas.src), err))
// 		}
// 		got := &bytes.Buffer{}
// 		_, err = DisassembleFunction(got, pkg)
// 		if err != nil {
// 			panic(fmt.Errorf("unexpected error: source: %q, disassemble error: %s", oneLine(cas.src), err))
// 		}

// 		out.WriteString("{\n")
// 		out.WriteString("\t\"" + cas.name + "\",\n")
// 		out.WriteString("\t`" + cas.src + "`,\n")
// 		out.WriteString("\t[]string{\n")
// 		for _, line := range strings.Split(strings.TrimSpace(got.String()), "\n") {
// 			out.WriteString("\t\t`" + line + "`,\n")
// 		}
// 		out.WriteString("\t},\n")
// 		out.WriteString("\t[]reg{\n")
// 		out.WriteString("\t\t// Registers overwritten (use your diff tool to restore original ones)\n")
// 		out.WriteString("\t},\n")
// 		out.WriteString("},\n")
// 	}
// 	t.Error(out.String())
// }

var goPackages = map[string]*parser.GoPackage{
	"fmt": &parser.GoPackage{
		Name: "fmt",
		Declarations: map[string]interface{}{
			"Errorf":     fmt.Errorf,
			"Formatter":  reflect.TypeOf(new(fmt.Formatter)).Elem(),
			"Fprint":     fmt.Fprint,
			"Fprintf":    fmt.Fprintf,
			"Fprintln":   fmt.Fprintln,
			"Fscan":      fmt.Fscan,
			"Fscanf":     fmt.Fscanf,
			"Fscanln":    fmt.Fscanln,
			"GoStringer": reflect.TypeOf(new(fmt.GoStringer)).Elem(),
			"Print":      fmt.Print,
			"Printf":     fmt.Printf,
			"Println":    fmt.Println,
			"Scan":       fmt.Scan,
			"ScanState":  reflect.TypeOf(new(fmt.ScanState)).Elem(),
			"Scanf":      fmt.Scanf,
			"Scanln":     fmt.Scanln,
			"Scanner":    reflect.TypeOf(new(fmt.Scanner)).Elem(),
			"Sprint":     fmt.Sprint,
			"Sprintf":    fmt.Sprintf,
			"Sprintln":   fmt.Sprintln,
			"Sscan":      fmt.Sscan,
			"Sscanf":     fmt.Sscanf,
			"Sscanln":    fmt.Sscanln,
			"State":      reflect.TypeOf(new(fmt.State)).Elem(),
			"Stringer":   reflect.TypeOf(new(fmt.Stringer)).Elem(),
		},
	},
	"testpkg": &parser.GoPackage{
		Name: "testpkg",
		Declarations: map[string]interface{}{
			"F00": func() {},
			"F01": func() int { return 40 },
			"F10": func(a int) {},
			"F11": func(a int) int { return a + 33 },
			"Sum": func(a, b int) int {
				return a + b
			},
			"StringLen": func(s string) int {
				return len(s)
			},
			"Pair": func() (int, int) {
				return 42, 33
			},
			"Inc": func(a int) int {
				return a + 1
			},
			"Dec": func(a int) int {
				return a - 1
			},
			"Swap": func(a int, b string) (string, int) {
				return b, a
			},
			"PrintString": func(s string) {
				fmt.Print(s)
			},
			"PrintInt": func(i int) {
				fmt.Print(i)
			},
			"A": &A,
			"B": &B,
		},
	},
}

// Used in tests.
var A int
var B int

func init() {
	A = 20
	B = 42
}
