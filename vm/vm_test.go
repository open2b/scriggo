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

	// Numbers.
	`3`:    int64(3),
	`-3`:   int64(-3),
	`3000`: int64(3000),
	`4.5`:  float64(4.5),
	`-4.5`: float64(-4.5),

	// Strings.
	`"abc"`: "abc",

	// Composite literals - arrays.
	`[4]int{1,2,3,4}`:     []int{1, 2, 3, 4},  // Internally, arrays are stored as slices.
	`[...]int{1,2,3,4}`:   []int{1, 2, 3, 4},  // Internally, arrays are stored as slices.
	`[3]string{}`:         []string{},         // Internally, arrays are stored as slices.
	`[3]string{"a", "b"}`: []string{"a", "b"}, // Internally, arrays are stored as slices.

	// Composite literals - slices.
	`[]int{}`:                   []int{},
	`[]int{1, 2}`:               []int{1, 2},
	`[]int{0: 1, 1: 3}`:         []int{0: 1, 1: 3},
	`[]int{0: 1, 5: 3}`:         []int{0: 1, 5: 3},
	`[]string{}`:                []string{},
	`[]string{"a", "b"}`:        []string{"a", "b"},
	`[]string{"a", 5: "b"}`:     []string{"a", 5: "b"},
	`[]float64{5, 12, 0}`:       []float64{5, 12, 0},
	`[]float64{5.6, 12.3, 0.4}`: []float64{5.6, 12.3, 0.4},

	// Composite literals - maps.
	`map[int]string{}`:                                    map[int]string{},
	`map[[3]int]string{}`:                                 map[[3]int]string{},
	`map[int]string{1: "one"}`:                            map[int]string{1: "one"},
	`map[int]string{1: "one", 2: "two", 10: "ten"}`:       map[int]string{1: "one", 2: "two", 10: "ten"},
	`map[string]int{"one": 1, "two": 10 / 5, "three": 3}`: map[string]int{"one": 1, "two": 10 / 5, "three": 3},

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
			main, err := comp.CompilePackage("/test.go")
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
			case reflect.Slice, reflect.Map, reflect.Array:
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

// reg represents a register. It's used in tests only.
type reg struct {
	typ   Type
	r     int8
	value interface{}
}

var stmtTests = []struct {
	name         string   // name of the test.
	src          string   // source code which must be executed.
	disassembled []string // expected disassembler output. Can be nil.
	registers    []reg    // list of expected registers. Can be nil.
	output       string   // expected stdout/stderr output.
}{
	{"Return register is determined by function type, not value type",
		`package main

		import (
			  "fmt"
		)
		
		func f() interface{} {
			  return "hey"
		}
		
		func main() {
			  a := f()
			  fmt.Println(a)
		}`, nil, nil, "hey\n"},

	{"Package function with many return values (different types)",
		`
		package main

		import "fmt"

		func pair() (int, float64) {
			return 42, 33.0
		}

		func main() {
			a := 2
			b, c := pair()
			d, e := 11, 12
			fmt.Print(a, b, c, d, e)
			return
		}
		`, nil, nil, "2 42 33 11 12"},

	{"Correct order in assignments",
		`package main

		import "fmt"

		func f() []int {
			fmt.Printf("f")
			return []int{1, 2, 3}
		}

		func g() int {
			fmt.Printf("g")
			return 10
		}

		func main() {
			f()[1] = g()
		}
		`, nil, nil, "fg"},

	{"Issue #76",
		`package main

		import "fmt"

		func typeof(v interface{}) string {
			return fmt.Sprintf("%T", v)
		}
	
		
		func main() {
			{
				var a []int
				var b []byte
			
				a = []int{}
				b = []byte{}
			
				fmt.Printf("%T", a)
				fmt.Printf("%T", b)
			}
			{
				var a []int
				var b []byte

				a = []int{}
				b = []byte{}

				fmt.Println(a, typeof(a))
				fmt.Println(b, typeof(b))
			}
		}`,
		nil, nil, "[]int[]uint8[] []int\n[] []uint8\n"},

	{"Increment assignment should evaluate expression only once",
		`package main

		import (
			"fmt"
		)
		
		func i() int {
			fmt.Print("i()")
			return 0
		}
		
		func main() {
			s := []int{1,2,3}
			s[i()] += 5
			fmt.Println(s)
		}
		`,
		nil, nil, "i()[6 2 3]\n",
	},

	{"Package function with many return values (same type)",
		`package main

		import "fmt"

		func pair() (int, int) {
			return 42, 33
		}

		func main() {
			a := 2
			b, c := pair()
			d, e := 11, 12
			fmt.Print(a + b + c + d + e)
			return
		}
		`, nil, nil, "100"},

	{"Issue #67",
		`package main

		func f() {
			f := "hi!"
			_ = f
		}
		
		func main() {
		
		}`, nil, nil, ""},

	{"Issue #75",
		`package main

		import "fmt"
		
		func f(flag bool) {
			if flag {
				fmt.Println("flag is true")
			} else {
				fmt.Println("flag is false")
			}
		}
		
		func main() {
			f(true)
			f(false)
		}`, nil, nil, "flag is true\nflag is false\n"},

	{"Functions with strings as input/output arguments",
		`package main

		import "fmt"
		
		func f(s1, s2 string) string {
			return s1 + s2
		}
		
		func g(s string) int {
			return len(s)
		}
		
		func main() {
			b := "hey"
			fmt.Print(
				g(
					f("a", b),
				),
			)
		}`,
		nil, nil, "4"},

	{"Named return parameters",
		`package main
		
		import "fmt"
		
		func f1() (a int, b int) {
			a = 10
			b = 20
			return
		}
		
		func f2() (a int, b int) {
			return 30, 40
		}
		
		func f3(bool) (a int, b int, c string) {
			a = 70
			b = 80
			c = "str2"
			return
		}
		
		func f4() (a int, b int, c float64) {
			a = 90
			c = 100
			return
		}
		
		func main() {
			v11, v12 := f1()
			v21, v22 := f2()
			v31, v32, v33 := f3(true)
			v41, v42, v43 := f3(false)
			v51, v52, v53 := f4()
			fmt.Println(v11, v12)
			fmt.Println(v21, v22)
			fmt.Println(v31, v32, v33)
			fmt.Println(v41, v42, v43)
			fmt.Println(v51, v52, v53)
		}
		`, nil, nil, "10 20\n30 40\n70 80 str2\n70 80 str2\n90 0 100\n"},

	{"Anonymous function arguments",
		`package main

		import "fmt"
		
		func f(int) { }
		
		func g(string, string) int {
			return 10
		}
		
		func main() {
			f(3)
			v := g("a", "b")
			fmt.Println(v)
		}
		`, nil, nil, "10\n"},

	{"Bit operators & (And) and | (Or)",
		`package main

		import (
			"fmt"
		)
		
		func main() {
			a := 5
			b := 11
			or := a | b
			and := a & b
			fmt.Println(or, and)
		}
		`, nil, nil, "15 1\n"},

	{"Bit operators ^ (Xor) and &^ (bit clear = and not)",
		`package main

		import (
			"fmt"
		)
		
		func main() {
			a := 5
			b := 11
			xor := a ^ b
			bitclear := a &^ b
			fmt.Println(xor, bitclear)
		}
		`, nil, nil, "14 4\n"},

	{"Channels - Reading and writing",
		`package main

		import "fmt"
		
		func main() {
			ch := make(chan int, 4)
			ch <- 5
			v := <-ch
			fmt.Print(v)
		}
		`, nil, nil, "5"},

	{"Variable swapping",
		`package main

		import "fmt"

		func main() {
			a := 10
			b := 20
			fmt.Println(a, b)
			a, b = a, b
			fmt.Println(a, b)
			a, b = b, a
			fmt.Println(a, b)
			b, a = a, b
			fmt.Println(a, b)
		}`,
		nil,
		nil,
		"10 20\n10 20\n20 10\n10 20\n"},

	{"Some maths",
		`package main

		import (
			"fmt"
		)

		func eval(v int) {
			fmt.Print(v, ",")
		}

		func main() {
			a := 33
			b := 42
			eval(a + b)
			eval(a - b)
			eval(a * b)
			eval(a / b)
			eval(a % b)
			eval(a + b*b)
			eval(a*b + a - b)
			eval(a - b - b + a*b)
			eval(a/3 + b/10)
		}
		`, nil, nil,
		"75,-9,1386,0,33,1797,1377,1335,15,"},

	{"Selector (native struct)",
		`package main

		import (
			"fmt"
			"testpkg"
		)
		
		func main() {
			center := testpkg.Center
			fmt.Println(center)
		}
		`, nil, nil, "{5 42}\n"},
	{"Implicit return",
		`package main

		import "fmt"

		func f() {
			fmt.Print("f")
		}

		func g() {
			fmt.Print("g")
		}

		func main() {
			f()
			g()
		}`,
		nil, nil, "fg"},

	{"Recover",
		`package main
		
		func main() {
			recover()
			v := recover()
			_ = v
		}
		`, nil, nil, ""},

	{"Defer - Calling f ang g (package level functions)",
		`package main

		import "fmt"
		
		func f() {
			fmt.Println("f")
		}
		
		func g() {
			fmt.Println("g")
		}
		
		func main() {
			defer f()
			defer g()
		}`,
		nil,
		nil,
		"g\nf\n"},

	{"Dot import (native)",
		`package main
		
		import . "fmt"
		
		func main() {
			Println("hey")
		}`,
		nil, nil, "hey\n"},

	{"Import (native) with explicit package name",
		`package main
		
		import f "fmt"
		
		func main() {
			f.Println("hey")
		}`,
		nil, nil, "hey\n"},

	{"Import (native) with explicit package name (two packages)",
		`package main
		
		import f "fmt"
		import f2 "fmt"
		
		func main() {
			f.Println("hey")
			f2.Println("oh!")
		}`,
		nil, nil, "hey\noh!\n"},

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
		// TODO(Gianluca): add a register test.
		nil,
		"10 42 88\n"},

	{"Assignment to _ evaluates expression",
		`
		package main

		import "fmt"

		func f() int {
			fmt.Print("f")
			return 0
		}

		func main() {
			_ = 4
			_ = f()
			_ = []int{f(), f(), 4, 5}
			_ = f() + f()
			_ = -f()
		}
		`,
		nil,
		nil,
		"ffffff"},

	{"Order of evaluation - Composite literal values",
		`
		package main

		import (
			"fmt"
		)

		func a() int {
			fmt.Print("a")
			return 0
		}

		func b() int {
			fmt.Print("b")
			return 0
		}
		func c() int {
			fmt.Print("c")
			return 0
		}
		func d() int {
			fmt.Print("d")
			return 0
		}
		func e() int {
			fmt.Print("e")
			return 0
		}

		func main() {
			s := []int{a(), b(), c(), d(), e()}
			_ = s
		}
		`, nil, nil,
		"abcde"},

	{"Single assignment",
		`
		package main

		import "fmt"

		func main() {
			var a int
			var c string

			a = 10
			c = "hi"

			fmt.Print(a, c)
		}
		`, nil, nil, "10hi"},

	{"Multiple assignment",
		`package main

		import "fmt"
		
		func main() {
			a, b := 6, 7
			fmt.Print(a, b)
		}
		`,
		nil, nil, "6 7"},

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
		`package main

		import "fmt"
		
		func main() {
			a := 5
			b := 10
			c := a + 2 + b + 30
			fmt.Print(c)
		}`,
		nil, nil, "47"},

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
		`, nil, nil, "5 42 33\n"},
	{"Type assertion assignment",
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
		`, nil, nil,
		"[2 2 3]\n[a b d d]\n"},

	{"Arrays - Composite literal, len",
		`
		package main

		import (
			"fmt"
		)

		func main() {
			a := [3]int{1, 2, 3}
			b := [...]string{"a", "b", "c", "d"}
			fmt.Println(a, b)
			fmt.Print(len(a), len(b))
		}
		`, nil, nil, "[1 2 3] [a b c d]\n3 4"},

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

		import "fmt"

		func main() {
			var a []int
			var b []byte

			a = []int{}
			b = []byte{}

			fmt.Print(a, b)
			return
		}
		`,
		nil, nil, "[] []"},

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
		`package main

		import "fmt"
		
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
		
			fmt.Print(l)
			fmt.Print(g)
			fmt.Print(le)
			fmt.Print(ge)
			fmt.Print(e)
			fmt.Print(ne)
		}`,
		nil, nil,
		"truefalsetruefalsefalsetrue"},

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
		nil,
		// TODO(Gianluca): test output.
		""},

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
		// TODO(Gianluca): test output.
		nil,
		""},

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
		nil,
		"s\neeff"},

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
		nil, nil, "hello world"},

	{"Indexing",
		`package main

		import "fmt"

		func main() {
			s := []int{1, 42, 3}
			second := s[1]
			fmt.Println(second)
		}`, nil, nil, "42\n"},
	{"If statement with else",
		`package main

		import "fmt"
		
		func main() {
			a := 10
			c := 0
			if a > 5 {
				fmt.Print("then")
				c = 1
			} else {
				fmt.Print("else")
				c = 2
			}
			fmt.Print("c=", c)
		}
		`, nil, nil, "thenc=1"},

	{"If statement with else",
		`package main

		import "fmt"
		
		func main() {
			a := 10
			c := 0
			if a <= 5 {
				c = 10
				fmt.Print("then")
			} else {
				c = 20
				fmt.Print("else")
			}
			fmt.Print(c)
		}
		`, nil, nil, "else20"},

	{"If with init assignment",
		`package main

		import "fmt"
		
		func main() {
			c := 0
			if x := 1; x == 1 {
				c = 1
				fmt.Print("then")
				fmt.Print("x is ", x)
			} else {
				c = 2
				fmt.Print("else")
			}
			fmt.Print(c)
		}
		`, nil, nil, "thenx is 11"},

	{"For statement",
		// TODO(Gianluca): this is the correct output:
		// `, nil, nil, "i=0,i=1,i=2,i=3,i=4,i=5,i=6,i=7,i=8,i=9,sum=20"},
		`package main

		import "fmt"
		
		func main() {
			sum := 0
			i := 0
			for i = 0; i < 10; i++ {
				fmt.Print("i=", i, ",")
				sum = sum + 2
			}
			fmt.Print("sum=",sum)
		}
		`, nil, nil, "i=1,i=2,i=3,i=4,i=5,i=6,i=7,i=8,i=9,i=10,sum=20"},

	{"Switch statement",
		`package main

		import "fmt"
		
		func main() {
			a := 0
			switch 1 + 1 {
			case 1:
				fmt.Print("case 1")
				a = 10
			case 2:
				fmt.Print("case 1")
				a = 20
			case 3:
				fmt.Print("case 1")
				a = 30
			}
			fmt.Print("a=", a)
			_ = a
		}
		`, nil, nil, "case 1a=20"},

	{"Switch statement with fallthrough",
		`package main

		import "fmt"
		
		func main() {
			a := 0
			switch 1 + 1 {
			case 1:
				fmt.Print("case 1")
				a = 10
			case 2:
				fmt.Print("case 2")
				a = 20
				fallthrough
			case 3:
				fmt.Print("case 3")
				a = 30
			}
			fmt.Print(a)
		}
		`, nil, nil, "case 2case 330"},

	{"Switch statement with default",
		`package main

		import "fmt"
		
		func main() {
			a := 0
			switch 10 + 10 {
			case 1:
				fmt.Print("case 1")
				a = 10
			case 2:
				fmt.Print("case 2")
				a = 20
			default:
				fmt.Print("default")
				a = 80
			case 3:
				fmt.Print("case 3")
				a = 30
			}
			fmt.Print(a)
		}
		`, nil, nil, "default80"},

	{"Switch statement with default and fallthrough",
		`package main

		import "fmt"
		
		func main() {
			a := 0
			switch 2 + 2 {
			case 1:
				fmt.Print("case 1")
				a = 10
			default:
				fmt.Print("default")
				a = 80
				fallthrough
			case 2:
				fmt.Print("case 2")
				a = 1
				fallthrough
			case 3:
				fmt.Print("case 3")
				a = 30
			case 40:
				fmt.Print("case 4")
				a = 3
			}
			fmt.Print(a)
		}
		`, nil, nil, "defaultcase 2case 330"},

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
		`, nil, nil, "20"},

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

		import "fmt"

		func main() {
			a := 0
			f := func() int {
				a := 10
				b := 15
				return a + b
			}
			a = f()
			fmt.Print(a)
			return
		}
		`, nil, nil, "25"},

	{"Function literal call - Shadowing package level function",
		// TODO(Gianluca): test output.
		`
		package main

		func f() {
			panic("")
		}

		func main() {
			f := func() {
				// No panic.
			}
			f()
			return
		}
		`, nil, nil, ""},

	{"Package function call",
		`
		package main

		import "fmt"

		func a() {
			fmt.Printf("a")
		}

		func main() {
			a()
			return
		}
		`,
		nil, nil, "a"},

	{"Package function 'inc'",
		`package main

		import "fmt"
		
		func inc(n int) int {
			return n + 1
		}
		
		func main() {
			var a, res, b, c int
		
			a = 2
			res = inc(8)
			b = 10
			c = a + b + res
		
			fmt.Print(c)
			return
		}
		`, nil, nil, "21"},

	{"Package function with one return value",
		`package main

		import "fmt"

		func five() int {
			return 5
		}

		func main() {
			a := five()
			fmt.Print(a)
		}`, nil, nil, "5"},

	{"Package function - 'floats'",
		`package main

		import (
			"fmt"
		)
		
		func floats() (float32, float64) {
			a := float32(5.6)
			b := float64(-432.12)
			return a, b
		}
		
		func main() {
			f1, f2 := floats()
			fmt.Println(f1, f2)
		}`,
		nil, nil, "5.6 -432.12\n"},
	{"Package function with one parameter (not used)",
		`
		package main

		import "fmt"

		func f(a int) {
			return
		}

		func main() {
			a := 2
			f(3)
			b := 10
			c := a + b
			fmt.Print(c)
			return
		}
		`, nil, nil, "12"},

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

	{"Builtin len (with a constant argument)",
		`
		package main

		import "fmt"

		func main() {
			a := len("abc");
			fmt.Println(a)
		}
		`, nil, nil, "3\n"},

	{"Builtin len",
		`package main

		import "fmt"
		
		func main() {
			a := "hello"
			b := []int{1, 2, 3}
			c := []string{"", "x", "xy", "xyz"}
			l1 := len(a)
			l2 := len(b)
			l3 := len(c)
			fmt.Print(l1)
			fmt.Print(l2)
			fmt.Print(l3)
		}
		`,
		nil, nil, "534"},

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
			{TypeIface, 3, map[string]int{}},
		}, ""},

	{"Builtin copy",
		`package main

		import "fmt"
		
		func main() {
			src := []int{10, 20, 30}
			dst := []int{1, 2, 3}
			n := copy(dst, src)
			fmt.Println("dst:", dst, "n:", n)
			dst2 := []int{1, 2}
			copy(dst2, src)
			fmt.Println("dst2:", dst2)
		}
		`, nil, nil, "dst: [10 20 30] n: 3\ndst2: [10 20]\n"},
	{"Function which calls both Scrigo and native functions",
		`package main

		import "fmt"
		
		func scrigoFunc() {
			fmt.Println("scrigoFunc()")
		}

		func main() {
			scrigoFunc()
			fmt.Println("main()")
		}`,
		nil,
		nil,
		"scrigoFunc()\nmain()\n"},
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
		}`,
		nil, nil, ""},

	{"Native function call (1 in, 1 out)",
		`
		package main

		import (
			"fmt"
			"testpkg"
		)

		func main() {
			a := 2
			a = testpkg.F11(9)
			fmt.Print(a)
			return
		}
		`, nil, nil, "42"},

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

	{"Native function call f(g())",
		`package main

		import (
			"fmt"
		)
		
		func f(a int, b int) {
			fmt.Println("a is", a, "and b is", b)
			return
		}
		
		func g() (int, int) {
			return 42, 33
		}
		
		func main() {
			f(g())
		}
		`,
		nil, nil,
		"a is 42 and b is 33\n"},

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

	{"Pointer declaration (nil int pointer)",
		`package main

		import (
			"fmt"
		)
		
		func main() {
			var a *int
			fmt.Print(a)
		}
		`, nil, nil, "<nil>"},

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
		nil, nil, ""},

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
			{TypeIface, 3, [][]int{[]int{10, 20}, []int{25, 26}, []int{30, 40, 50}}},
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
		import "fmt"

		func main() {
			var i1, i2 int
			var s1, s2 string

			i1 = 3
			s1 = "hey"

			s1, i1 = testpkg.Swap(i1, s1)
			s2, i2 = testpkg.Swap(i1, s1)

			fmt.Print(s1, s2, i1, i2)
		}
		`, nil, nil, "heyhey3 3"},

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
		"a:6,b:2,c:2,d:6,e:16,f:40330,"},
	{"f(g()) call (advanced test)",
		`package main

		import (
			"fmt"
		)
		
		func f(a int, b int) {
			fmt.Println("a is", a, "and b is", b)
		}
		
		func g(s string) (int, int) {
			return len(s), len(s) * (len(s) + 7)
		}
		
		func main() {
			f(g("a string!"))
		}`,
		nil,
		nil,
		"a is 9 and b is 144\n"},

	//------------------------------------
	// TODO(Gianluca): disabled tests:

	// {"Variable declaration with 'var'",
	// 	`package main

	// 	import "fmt"

	// 	func pair() (int, string) {
	// 		return 10, "zzz"
	// 	}

	// 	func main() {
	// 		var a, b = 1, 2
	// 		var c, d int
	// 		var e, f = pair()
	// 		fmt.Print(a, b, c, d, e, f)
	// 	}
	// 	`,
	// 	nil,
	// 	nil,
	// 	"1 2 0 0 10zzz"},

	//------------------------------------

	// {"Defer",
	// 	`
	// 	package main

	// 	import "fmt"

	// 	func mark(m string) {
	// 		fmt.Print(m + ",")
	// 	}

	// 	func f() {
	// 		mark("f")
	// 		defer func() {
	// 			mark("f.1")
	// 			defer func() {

	// 			}()
	// 		}()
	// 		defer func() {
	// 			mark("f.2")
	// 			defer func() {
	// 				mark("f.2.1")
	// 			}()
	// 			h()
	// 		}()
	// 		g()
	// 		mark("ret f")
	// 	}

	// 	func g() {
	// 		mark("g")
	// 	}

	// 	func h() {
	// 		mark("h")
	// 	}

	// 	func main() {
	// 		mark("main")
	// 		defer func() {
	// 			mark("main.1")
	// 			h()
	// 			mark("main.1")
	// 		}()
	// 		f()
	// 		mark("ret main")
	// 	}
	// 	`,
	// 	nil, nil, "main,f,g,ret f,f.2,h,f.2.1,f.1,ret main,main.1,h,main.1,"},

	//------------------------------------

	// {"Multiple assignment to slices",
	// 	`package main

	// 	import "fmt"

	// 	func triplet() (int, string, string) {
	// 		return 20, "new1", "new2"
	// 	}

	// 	func main() {
	// 		ss := []string{"old1", "old2", "old3"}
	// 		is := []int{1,2,3}
	// 		fmt.Println(ss, is)
	// 		is[0], ss[1], ss[0] = triplet()
	// 		fmt.Println(ss, is)
	// 	}
	// 	`, nil, nil, "[old1 old2 old3] [1 2 3]\n[new2 new1 old3] [20 2 3]\n"},

	//------------------------------------

	// TODO (Gianluca):
	// {"Operator address (&)",
	// 	`
	// 	package main

	// 	import "fmt"

	// 	func main() {
	// 		a := 1
	// 		b := &a
	// 		c := &a
	// 		fmt.Println(b == c)
	// 	}
	// 	`, nil, nil, ""},

	//------------------------------------

	// TODO(Gianluca):
	// {"Reading an int variable",
	// 	`package main

	// 	import "fmt"

	// 	var A int = 40

	// 	func main() {
	// 		fmt.Println("A is ", A)
	// 	}
	// 	`,
	// 	nil, nil,
	// 	"A is 40\n"},

	//------------------------------------

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

	//------------------------------------

	// TODO(Gianluca): index out of range.
	// {"Function literal call (1 in, 1 out)",
	// 	`package main

	// 	import "fmt"

	// 	func main() {
	// 		a := 41
	// 		fmt.Print(a)
	// 		inc := func(a int) int {
	// 			fmt.Print("inc:", a)
	// 			b := a
	// 			b++
	// 			fmt.Print("inc:", b)
	// 			return b
	// 		}
	// 		a = inc(a)
	// 		fmt.Print(a)
	// 		return
	// 	}
	// 	`, nil, nil, "41inc:41inc:4242"},

	//------------------------------------

	// TODO(Gianluca): conversion always puts result into "general", so this
	// test cannot pass.
	// {"Type assertion as expression (single value context)",
	// 	`package main

	// 	import "fmt"

	// 	func main() {
	// 		i := interface{}(int64(5))
	// 		a := 7 + i.(int64)
	// 		fmt.Print(i, ", ", a)
	// 		return
	// 	}
	// 	`,
	// 	nil, nil,
	// 	"5, 12",
	// },

	//------------------------------------

	// TODO (Gianluca):
	// {"(Native) struct composite literal (empty)",
	// 	`package main

	// 	import (
	// 		"fmt"
	// 		"testpkg"
	// 	)

	// 	func main() {
	// 		t1 := testpkg.TestPointInt{}
	// 		fmt.Println(t1)
	// 	}
	// 	`,
	// 	nil,
	// 	nil,
	// 	"",
	// },
}

func TestVM(t *testing.T) {
	DebugTraceExecution = false
	for _, cas := range stmtTests {
		t.Run(cas.name, func(t *testing.T) {
			registers := cas.registers
			r := parser.MapReader{"/test.go": []byte(cas.src)}
			comp := NewCompiler(r, goPackages)
			main, err := comp.CompilePackage("/test.go")
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
				_, err := io.Copy(&buf, reader)
				if err != nil {
					panic(err)
				}
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
				gotLines := strings.Split(strings.TrimSpace(got), "\n")
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
			"A":            &A,
			"B":            &B,
			"TestPointInt": reflect.TypeOf(new(TestPointInt)).Elem(),
			"GetPoint":     GetPoint,
			"Center":       &TestPointInt{A: 5, B: 42},
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

type TestPointInt struct {
	A, B int
}

func GetPoint() TestPointInt {
	return TestPointInt{A: 5, B: 42}
}
