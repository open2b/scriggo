// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"scriggo"
	"scriggo/vm"
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
	`[4]int{1,2,3,4}`:     []int{1, 2, 3, 4},      // Internally, arrays are stored as slices.
	`[...]int{1,2,3,4}`:   []int{1, 2, 3, 4},      // Internally, arrays are stored as slices.
	`[3]string{}`:         []string{"", "", ""},   // Internally, arrays are stored as slices.
	`[3]string{"a", "b"}`: []string{"a", "b", ""}, // Internally, arrays are stored as slices.

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
	if goPackages == nil {
		panic("goPackages declared but not initialized. Run 'go generate scriggo/test' with the latest version of scriggo installed to create the package declaration file.")
	}
	for src, expected := range exprTests {
		t.Run(src, func(t *testing.T) {
			r := scriggo.MapStringLoader{"main": "package main; func main() { a := " + src + "; _ = a }"}
			program, err := scriggo.LoadProgram(scriggo.Loaders(r, goPackages), &scriggo.LoadOptions{LimitMemorySize: true})
			if err != nil {
				t.Errorf("test %q, compiler error: %s", src, err)
				return
			}
			var registers vm.Registers
			tf := func(_ *vm.Function, _ uint32, regs vm.Registers) {
				registers = regs
			}
			err = program.Run(&scriggo.RunOptions{MaxMemorySize: 1000000, TraceFunc: tf})
			if err != nil {
				t.Errorf("test %q, execution error: %s", src, err)
				return
			}
			kind := reflect.TypeOf(expected).Kind()
			var got interface{}
			switch kind {
			case reflect.Int, reflect.Bool, reflect.Int64:
				got = registers.Int[0]
			case reflect.Float32, reflect.Float64:
				got = registers.Float[0]
			case reflect.String:
				got = registers.String[0]
			case reflect.Slice, reflect.Map, reflect.Array:
				got = registers.General[0]
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
	typ   vm.Type
	r     int8
	value interface{}
}

var stmtTests = []struct {
	name       string      // name of the test.
	src        string      // source code which must be executed.
	out        string      // expected stdout/stderr output.
	err        interface{} // error.
	freeMemory int         // free memory in bytes, set to zero if there is no limit.
}{

	{
		name: "Type assertions",
		src: `package main

		import "fmt"
		
		func main() {
			fmt.Print(interface{}(1).(int))
			i := interface{}(2).(int)
			fmt.Print(i)
			fmt.Print(interface{}(5).(int) * interface{}(7).(int))
		}
		`,
		out: `1235`,
	},

	{
		name: "Issue #175",
		src: `package main

		func main() {
			switch u := interface{}(2).(type) {
			case int:
				_ = u * 2
			}
		}
		`,
		out: ``,
	},

	{
		name: "Issue #176 (2)",
		src: `package main

		import (
			"fmt"
			"unicode"
		)
		
		func main() {
			f := func() {
				fmt.Println(unicode.Cc)
			}
			f()
		}
		`,
		out: "&{[{0 31 1} {127 159 1}] [] 2}\n",
	},

	{
		name: "Issue #176",
		src: `package main

		import (
			"unicode"
		)
		
		func main() {
			f := func() {
				_ = unicode.Cc
			}
			_ = f
		}
		`,
		out: "",
	},

	{
		name: "appending to a new slice every iteration",
		src: `package main

		import "fmt"
		
		func main() {
			for i := range []int{1, 2} {
				s := []int(nil)
				s = append(s, 1)
				fmt.Printf("iteration #%d: s = %v\n", i, s)
			}
		}
		`,
		out: "iteration #0: s = [1]\niteration #1: s = [1]\n",
	},

	{
		name: "Issue #188",
		src: `package main

		import "fmt"
		
		var p = fmt.Println
		
		func main() {
			p("hello, world")
		}
		`,
		out: "hello, world\n",
	},

	{
		name: "Issue #112",
		src: `package main

		import "fmt"
		
		func main() {
			fmt.Printf("%#v ", [3]int{1, 2, 3})
			a := [4]string{"a", "b", "c", "d"}
			fmt.Printf("%#v", a)
		}
		`,
		out: `[3]int{1, 2, 3} [4]string{"a", "b", "c", "d"}`,
	},

	{
		name: "Issue #186",
		src: `package main

		import "fmt"
		
		func main() {
			fmt.Printf("%v ", [3]int{1, 2, 3})
			a := [10]string{"a", "b", "c"}
			fmt.Printf("%v", a)
		}
		`,
		out: `[1 2 3] [a b c       ]`,
	},

	{
		name: "Issue #181",
		src: `package main

		import "fmt"
		
		func main() {
			ss := [][]int{}
			ss = append(ss, []int{10, 20, 30})
			fmt.Print(ss)
		}
		`,
		out: `[[10 20 30]]`,
	},

	{
		name: "Builtin append (non variadic)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			s := []int{}
			fmt.Print(s)
			s = append(s, 1)
			fmt.Print(s)
			s = append(s, 2, 3)
			fmt.Print(s)
			s = append(s)
			fmt.Print(s)
		}
		`,
		out: `[][1][1 2 3][1 2 3]`,
	},

	{
		name: "Appending a slice to a slice (variadic call to builtin append)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			s := []int{}
			fmt.Print(s)
			s = append(s, []int{1, 2, 3}...)
			fmt.Print(s)
			ns := []int{4, 5, 6}
			s = append(s, ns...)
			fmt.Print(s)
		}
		`,
		out: `[][1 2 3][1 2 3 4 5 6]`,
	},

	{
		name: "Issue #174 (2)",
		src: `package main

		import "fmt"

		func f(a ...int) {
			fmt.Println("variadic:", a)
			return
		}

		func g(a, b string, c... int) {
			fmt.Println("strings:", a, b)
			fmt.Println("variadic:", c)
			return
		}

		func h(a string, b... string) int {
			fmt.Println("a:", a)
			return len(b)
		}

		func main() {
			f(1, 2, 3)
			g("x", "y", 3, 23, 11, 12)
			fmt.Println("h:", h("a", "b", "c", "d"))
		}
		`,
		out: "variadic: [1 2 3]\nstrings: x y\nvariadic: [3 23 11 12]\na: a\nh: 3\n",
	},

	{
		name: "Issue #174",
		src: `package main

		import "fmt"
		
		func f(a string, slice ...int) {
			fmt.Print(slice)
			return
		}
		
		func main() {
			f("a", 10, 20, 30)
		}
		`,
		out: `[10 20 30]`,
	},

	{
		name: "Issue #132",
		src: `package main

		import "fmt"
		
		var (
			_ = d
			_ = f("_", c, b)
			a = f("a", 0, 0)
			b = f("b", 0, 0)
			c = f("c", 0, 0)
			d = f("d", 0, 0)
		)
		
		func f(s string, a int, b int) int {
			fmt.Print(s, a, b)
			return 0
		}
		
		func main() {}
		`,
		out: `a0 0b0 0c0 0_0 0d0 0`,
	},

	{
		name: "Function call with function call as argument",
		src: `package main

		import "fmt"
		
		func main() {
			fmt.Print(int(0))
			fmt.Print(int(int(1)))
			fmt.Print(int(2), int(-3))
		}
		`,
		out: `012 -3`,
	},

	{
		name: "local alias declaration which shadowes package alias declaration",
		src: `package main

		import "fmt"
		
		type T = int
		
		var v1 T = 10
		
		func main() {
			fmt.Printf("%v-%T", v1, v1)
			var v2 T = 20
			fmt.Printf("%v-%T", v2, v2)
			type T = string
			var v3 T = "hey"
			fmt.Printf("%v-%T", v3, v3)
		}
		`,
		out: `10-int20-inthey-string`,
	},

	{
		name: "local alias declaration",
		src: `package main

		import "fmt"
		
		func main() {
			type Int = int
			var i Int = 20
			fmt.Print(i)
		}
		`,
		out: `20`,
	},

	{
		name: "alias declaration used in function declaration",
		src: `package main

		import "fmt"
		
		type Int = int
		
		func zero() Int {
			var z int
			return z
		}
		
		func main() {
			fmt.Print(zero())
			var z1 Int = zero()
			var z2 int = zero()
			fmt.Print(z1, z2)
		}
		`,
		out: `00 0`,
	},

	{
		name: "alias declaration where first alias depends on second",
		src: `package main

		import "fmt"
		
		type Int2 = Int1
		type Int1 = int
		
		func main() {
			var i1 Int1 = Int1(10)
			var i2 Int2 = Int2(30)
			fmt.Print(i1, i2)
		}
		`,
		out: `10 30`,
	},

	{
		name: "alias declaration used in a composite literal",
		src: `package main

		import "fmt"
		
		type SliceInt = []int
		
		func main() {
			si := SliceInt{10, 20, 30}
			fmt.Print(len(si))
			fmt.Print(si)
		}
		`,
		out: `3[10 20 30]`,
	},

	{
		name: "alias declaration",
		src: `package main

		import "fmt"
		
		type Int = int
		
		func main() {
			var i Int = 10
			fmt.Print(i)
		}
		`,
		out: `10`,
	},

	{
		name: "Upvars referring to a multiple assignment",
		src: `package main

		func main() {
			a, b := 1, 1
			_ = a + b
			_ = func() {
				_ = a + b
			}
		}
		`,
		out: "",
	},

	{
		name: "Issue #185 (3)",
		src: `package main

		import "fmt"
		
		func main() {
			var s = make([]func(int) int, 1)
			s[0] = func(x int) int { return x + 1 }
			res := s[0](3)
			fmt.Println(res)
		}`,
		out: "4\n",
	},

	{
		name: "Issue #185 (2)",
		src: `package main

		import "fmt"
		
		func main() {
			var s = make([]func() int, 1)
			s[0] = func() int { return 10 }
			res := s[0]()
			fmt.Println(res)
		}`,
		out: "10\n",
	},

	{
		name: "Issue #185",
		src: `package main

		import "fmt"
		
		func main() {
			var s = make([]func(), 1)
			s[0] = func() { fmt.Println("called") }
			s[0]()
		}`,
		out: "called\n",
	},

	{
		name: "Unary operator -",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
		
			// Constants.
			var i interface{} = -4
			fmt.Print(i)
			fmt.Print(-329)
			a := -43
			fmt.Print(a)
		
			// Non constants.
			var i2 = 4
			var interf interface{} = -i2
			fmt.Print(interf)
			var b = 329
			fmt.Print(-b)
		}`,
		out: `-4-329-43-4-329`,
	},

	{
		name: "Floating point constant converted to interface{}",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			const d = 20.0
			fmt.Printf("%T", d)
			i := interface{}(d)
			fmt.Printf("%T", d)
			fmt.Print(i)
		}
		`,
		out: `float64float6420`,
	},

	{
		name: "Integer constant converted to interface{}",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			i := interface{}(20)
			fmt.Print(i)
		}
		`,
		out: `20`,
	},

	{
		name: "Issue #179 - Part one",
		src: `package main

		import "fmt"

		func main() {
			const d = 3e20 / 500000000
			fmt.Printf("d has type %T", d)
			// _ = int64(d)
		}`,
		out: `d has type float64`,
	},

	{
		name: "Issue #179 - Part two",
		src: `package main

		import "fmt"

		func main() {
			const d = 3e20 / 500000000
			fmt.Printf("d has type %T", d)
			_ = int64(d)
		}`,
		out: `d has type float64`,
	},

	{
		name: "For statements with break",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			for {
				fmt.Print("a")
				break
			}
			for i := 0; ; i = i + 2 {
				fmt.Print("b")
				if i > 10 {
					break
				}
			}
			for {
				fmt.Print("c1")
				for {
					fmt.Print("c2")
					break
					panic("")
				}
				break
				panic("")
			}
		}
		`,
		out: `abbbbbbbc1c2`,
	},

	{
		name: "For statements",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			for i := 0; i < 5; i++ {
				fmt.Print(i)
			}
			fmt.Print(",")
			for i := 0; i < 5; {
				fmt.Print(i)
				i++
			}
			fmt.Print(",")
			for i := 0; i < 5; {
				i++
				fmt.Print(i)
			}
			fmt.Print(",")
			i := 0
			for ; i < 5; i++ {
				fmt.Print(i)
			}
			fmt.Print(",")
			for i := 0; i < 5; i++ {
				i++
				fmt.Print(i)
			}
		}
		`,
		out: `01234,01234,12345,01234,135`,
	},

	{
		name: "Indexing operation with a constant expression",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			c0 := "variable"[0]
			fmt.Print(c0, "inline"[1])
		}
		`,
		out: `118 110`,
	},

	{
		name: "Shift operators on constants",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print(128 << 2)
			fmt.Print(2 >> 1)
		}
		`,
		out: `5121`,
	},

	{
		name: "Shift operators",
		src: `package main

		import (
			"fmt"
		)

		func main() {
			var a int = 20
			fmt.Print(a << 2)
			fmt.Print(a >> 1)
			var b int8 = 12
			fmt.Print(b << 3)
			fmt.Print(b >> 1)
			fmt.Print((b << 5) >> 5)
		}
		`,
		out: `8010966-4`,
	},

	{
		src: `package main

		import (
			"fmt"
			"testpkg"
		)
		
		func main() {
			c1 := testpkg.Complex128(1 + 2i)
			c2 := testpkg.Complex128(3281 - 12 - 32i + 1.43i)
			c3 := c1 + c2
			c4 := c1 - c2
			c5 := -(c1 * c2)
			fmt.Printf("%v %T, ", c3, c3)
			fmt.Printf("%v %T, ", c4, c4)
			fmt.Printf("%v %T", c5, c5)
		}
		`,
		out: `(3270-28.57i) testpkg.Complex128, (-3268+32.57i) testpkg.Complex128, (-3330.14-6507.43i) testpkg.Complex128`,
	},

	{
		name: "Complex defined type",
		src: `package main

		import (
			"fmt"
			"testpkg"
		)
		
		func main() {
			c1 := testpkg.Complex128(1 + 2i)
			fmt.Print(c1)
			fmt.Printf("%T", c1)
		}
		`,
		out: "(1+2i)testpkg.Complex128",
	},

	{
		name: "Binary operator on complex which are returned by a function call",
		src: `package main

		import (
			"fmt"
		)
		
		func doubleComplex(c complex128) complex128 {
			return 2 * c
		}
		
		func main() {
			fmt.Print(3+8i + doubleComplex(-2+0.2i))
			fmt.Print(doubleComplex(3+8i) + doubleComplex(-2+0.2i))
		}
		`,
		out: `(-1+8.4i)(2+16.4i)`,
	},

	{
		name: "Complex numbers declared with var",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			var x complex128 = complex(1, 2)
			var y complex128 = complex(3, 4)
			fmt.Print(x * y)
		}
		`,
		out: `(-5+10i)`,
	},

	{
		name: "Many complex operations",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print(3 + 3i)
			c1 := 3 + 3 + 19 - 3i - 120i
			fmt.Print(c1)
			fmt.Print(-c1)
			fmt.Print(c1 * c1)
			c2 := 321.43 + 3i - 32.129i
			fmt.Print(c2)
			fmt.Print(-c2)
			fmt.Print(c2 * c2)
			fmt.Print(c1 * c2)
		}
		`,
		out: `(3+3i)(25-123i)(-25+123i)(-14504-6150i)(321.43-29.129i)(-321.43+29.129i)(102468.746259-18725.86894i)(4452.883-40264.115i)`,
	},

	{
		name: "Negation of a complex number",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			c1 := 5 + 6i
			nc1 := -c1
			fmt.Print(c1, nc1)
		}
		`,
		out: `(5+6i) (-5-6i)`,
	},

	{
		name: "Complex number multiplied by an integer and by a float",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			c1 := 5 + 6i
			fmt.Print(c1)
			fmt.Print(10 * c1)
			fmt.Print(c1 * 0.4)
		}
		`,
		out: `(5+6i)(50+60i)(2+2.4000000000000004i)`,
	},

	{
		name: "Complex numbers operations assigned directly to interface{}",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			c1 := 5 + 6i
			c2 := -4 + 2i
			fmt.Print(c1 + c2)
			fmt.Print(c1 - c2)
			fmt.Print(c1 * c2)
			fmt.Print(c1 / c2)
		}
		`,
		out: `(1+8i)(9+4i)(-32-14i)(-0.4-1.7i)`,
	},

	{
		name: "Division of complex numbers",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			c1 := 5 + 6i
			c2 := -4 + 2i
			c3 := c1 / c2
			fmt.Print(c1, c2, c3)
		}
		`,
		out: `(5+6i) (-4+2i) (-0.4-1.7i)`,
	},

	{
		name: "Multiplication of complex numbers",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			c1 := 5 + 6i
			c2 := -4 + 2i
			c3 := c1 * c2
			fmt.Print(c1, c2, c3)
		}
		`,
		out: `(5+6i) (-4+2i) (-32-14i)`,
	},

	{
		name: "Subtraction of complex numbers",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			c1 := 5 + 6i
			c2 := -4 + 2i
			c3 := c1 - c2
			fmt.Print(c1, c2, c3)
		}
		`,
		out: `(5+6i) (-4+2i) (9+4i)`,
	},

	{
		name: "Addition of complex numbers",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			c1 := 5 + 6i
			c2 := -4 + 2i
			c3 := c1 + c2
			fmt.Print(c1, c2, c3)
		}
		`,
		out: `(5+6i) (-4+2i) (1+8i)`,
	},

	{
		name: "Complex number constants",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			c1 := 5 + 6i
			c2 := -2 - 2 - 1i
			c3 := c1
			fmt.Print(c1, c2, c3)
		}
		`,
		out: `(5+6i) (-4-1i) (5+6i)`,
	},

	{
		name: "Complex number constant",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			c1 := 5 + 6i
			fmt.Print(c1)
		}
		`,
		out: `(5+6i)`,
	},

	{
		name: "Issue #171",
		src: `package main

		import "fmt"
		
		func main() {
			slice := []string{"abc", "def", "ghi"}
			fmt.Print(slice[0][1])
			fmt.Print(slice[0])
			fmt.Print(slice[1])
			out := slice[1][2]
			fmt.Print(out)
		}`,
		out: `98abcdef102`,
	},

	{
		name: "Issue #155",
		src: `package main

		import "fmt"
		
		func main() {
			s := [][]int{[]int{1, 2, 3}, []int{4, 5, 6}, []int{7, 8, 9}}
			fmt.Print(s)
			fmt.Print(s[1][2])
		}`,
		out: `[[1 2 3] [4 5 6] [7 8 9]]6`,
	},

	{
		name: "Assignment to index of slice index",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			s := [][]int{[]int{0, 1}, []int{2, 3}}
			fmt.Print(s, ", ")
			s[0][0] = 5
			fmt.Print(s)
		}
		`,
		out: `[[0 1] [2 3]], [[5 1] [2 3]]`,
	},

	{
		name: "Binary operator",
		src: `package main

		import "fmt"
		
		func main() {
			a := 1
			b := 20
			fmt.Print(a == b)
			fmt.Print(bool(a == b))
			fmt.Print(bool(a != b))
			fmt.Print(a + b)
			fmt.Print(int(a) + int(b))
		}`,
		out: `falsefalsetrue2121`,
	},

	{
		name: "References equality (w/o an intermediate variable)",
		src: `package main

		import "fmt"
		
		func main() {
			a := 1
			b := &a
			c := &a
			fmt.Println(b == c)
		}
		`,
		out: "true\n",
	},

	{
		name: "References equality (assigning to an intermediate variable)",
		src: `package main

		import "fmt"
		
		func main() {
			a := 1
			b := &a
			c := &a
			equal := b == c
			fmt.Println(equal)
		}
		`,
		out: "true\n",
	},

	{
		name: "Multiplication of two constant integers with value = 0",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print(0 * 0)
		}
		`,
		out: "0",
	},

	{
		name: "Constants assigned to an interface type",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print(10)
			fmt.Print(34.51)
			fmt.Print("hello")
			i := interface{}(20)
			fmt.Print(i)
		}
		`,
		out: `1034.51hello20`,
	},

	{
		name: "Unary operator *",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			a := "message"
			b := &a
			fmt.Print("*b: ", *b)
			c := *b
			fmt.Print(", c: ", c)
		}`,
		out: "*b: message, c: message",
	},

	{
		name: "Unary operator !",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print(true)
			fmt.Print(!true)
			v := !false
			fmt.Print(v)
		}
		`,
		out: `truefalsetrue`,
	},

	// {
	// 	name: "Assigning a predefined function to a variable and calling it",
	// 	src: `package main

	// 	import (
	// 		"fmt"
	// 	)

	// 	func main() {
	// 		p := fmt.Print
	// 		p("hello")
	// 	}
	// 	`,
	// 	out: "hello",
	// },

	{
		name: "Binary operators && and ||",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			t := true
			f := false
			fmt.Print(t && f)
			fmt.Print(t && f || f && t)
			fmt.Print(t || f)
		}
		`,
		out: "falsefalsetrue",
	},

	{
		name: "Builtin copy",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			src := []int{1, 2, 3}
			dst := []int{0, 0, 0}
			n := copy(dst, src)
			fmt.Print("n is ", n)
			fmt.Print(", n is now ", copy(dst, src))
		}
		`,
		out: `n is 3, n is now 3`,
	},

	// TODO(Gianluca):
	// {
	// 	name: "Go calling a function literal defined in Scriggo",
	// 	src: `package main

	// 	import (
	// 		"fmt"
	// 		"strings"
	// 		"unicode"
	// 	)

	// 	func main() {
	// 		f := func(c rune) bool {
	// 			return unicode.Is(unicode.Han, c)
	// 		}
	// 		fmt.Print(strings.IndexFunc("Hello, 世界\n", f))
	// 		fmt.Print(strings.IndexFunc("Hello, world\n", f))
	// 	}
	// 	`,
	// 	out: "7-1",
	// },

	{
		name: "Conversions assigning values to variables of different kind",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			i64 := int64(100)
			i := int(i64)
			interf := interface{}(i64)
			fmt.Print(i64, i, interf)
			fmt.Printf("%T %T %T", i64, i, interf)
		}
		`,
		out: "100 100 100int64 int int64",
	},

	{
		name: "Binary boolean operator puts result in an empty interface context",
		src: `package main

		import (
			"fmt"
		)

		func main() {
			fmt.Print(true && false)
			a := true
			b := true
			fmt.Print(a && b)
			var i interface{} = true
			fmt.Print(i)
		}
		`,
		out: `falsetruetrue`,
	},

	{
		name: "Type assertion on empty interface containing an int value",
		src: `package main

		import "fmt"
		
		func main() {
			i := interface{}(5)
			a := 7 + i.(int)
			fmt.Print(i, ", ", a)
			return
		}
		`,
		out: "5, 12",
	},
	{
		name: "Variable declaration with 'var'",
		src: `package main

		import "fmt"

		func pair() (int, string) {
			return 10, "zzz"
		}

		func main() {
			var a, b = 1, 2
			var c, d int
			var e, f = pair()
			fmt.Print(a, b, c, d, e, f)
		}
		`,
		out: "1 2 0 0 10zzz",
	},
	{
		name: "Function literal call (1 in, 1 out)",
		src: `package main

		import "fmt"

		func main() {
			a := 41
			fmt.Print(a)
			inc := func(a int) int {
				fmt.Print("inc:", a)
				b := a
				b++
				fmt.Print("inc:", b)
				return b
			}
			a = inc(a)
			fmt.Print(a)
			return
		}
		`,
		out: "41inc:41inc:4242",
	},
	{
		name: "(Predefined) struct composite literal (empty)",
		src: `package main

		import (
			"fmt"
			"testpkg"
		)

		func main() {
			t1 := testpkg.TestPointInt{}
			fmt.Println(t1)
		}
		`,
		out: "{0 0}\n",
	},
	{
		name: "Reading an int variable",
		src: `package main

		import "fmt"

		var A int = 40

		func main() {
			fmt.Println("A is", A)
		}
		`,
		out: "A is 40\n",
	},
	{
		name: "blank identifier as global variable",
		src: `package main

		import "fmt"
		
		var _ = f()
		
		func f() int {
			fmt.Println("f called!")
			return 0
		}
		
		func main() {
		}
		`,
		out: "f called!\n",
	},
	{
		name: "Nil: predeclared nil as function parameter",
		src: `package main

		import (
			"fmt"
		)
		
		func f(s []int) {
			fmt.Printf("s: %v, len(s): %d, type(s): %T\n", s, len(s), s)
		}
		
		func main() {
			f([]int{10,20,30})
			f(nil)
		}
		`,
		out: "s: [10 20 30], len(s): 3, type(s): []int\ns: [], len(s): 0, type(s): []int\n",
	},
	{
		name: "Nil: declaring a nil slice with var and explicitly assigning 'nil' to it",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			var s []int = nil
			fmt.Print("s: ", s, ", len(s): ", len(s))
		}
		`,
		out: `s: [], len(s): 0`,
	},
	{
		name: "Predefined constants",
		src: `package main

		import (
			"fmt"
			"math"
			"time"
		)
		
		func main() {
			phi := math.Phi
			fmt.Println("phi:", phi)
		
			ansic := time.ANSIC
			fmt.Println("ansic:", ansic)
		
			nanosecond := time.Nanosecond
			fmt.Println("nanosecond:", nanosecond)
		}
		`,
		out: "phi: 1.618033988749895\nansic: Mon Jan _2 15:04:05 2006\nnanosecond: 1ns\n",
	},

	{
		name: "Predeclared identifier nil as function argument",
		src: `package main

		import (
			"fmt"
		)
		
		func f(s []int) {
			fmt.Print(s)
		}
		
		func main() {
			f([]int{1, 2, 3})
			f(nil)
			f([]int{})
		}
		`,
		out: `[1 2 3][][]`,
	},

	{
		name: "Float exponent as constant",
		src: `package main

		import "fmt"
		
		func main() {
			const d = 3e20
			fmt.Println(d)
		}
		`,
		out: "3e+20\n",
	},

	{
		name: "Reserving the right register for a function body with a variadic argument",
		src: `package main

		import "fmt"
		
		func sum(nums ...int) {
			fmt.Print(nums, " ")
			nums[0] = -2
			fmt.Print(nums, " ")
		}
		
		func main() {
			sum(1, 2)
			sum(1, 2, 3)
		}
		`,
		out: `[1 2] [-2 2] [1 2 3] [-2 2 3] `,
	},

	{
		name: "Make slice with different combinations of len and cap",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			var s []int
			var l, c int
			s = make([]int, 10)
			fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
			s = make([]int, 4, 6)
			fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
			s = make([]int, 0, 3)
			fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
			l = 2
			s = make([]int, l)
			fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
			c = 5
			s = make([]int, l, c)
			fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
			s = make([]int, l, 20)
			fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
			s = make([]int, 0, c)
			fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
		}
		`,
		out: "[0 0 0 0 0 0 0 0 0 0], len: 10, cap: 10\n[0 0 0 0], len: 4, cap: 6\n[], len: 0, cap: 3\n[0 0], len: 2, cap: 2\n[0 0], len: 2, cap: 5\n[0 0], len: 2, cap: 20\n[], len: 0, cap: 5\n",
	},

	{
		name: "Builtin make - len and cap of a slice",
		src: `package main

		import "fmt"
		
		func main() {
			s := make([]int, 5, 10)
			fmt.Println("s:", s)
			fmt.Println("len:", len(s))
			fmt.Println("cap:", cap(s))
		}
		`,
		out: "s: [0 0 0 0 0]\nlen: 5\ncap: 10\n",
	},

	{
		name: "Type assertion in assignment",
		src: `package main

		import (
			"fmt"
		)

		func main() {
			var i interface{} = "hello"
			s := i.(string)
			s, ok := i.(string)
			fmt.Println(s, ok)
		}
		`,
		out: "hello true\n",
	},

	{
		name: "Converting a float to uint",
		src: `package main

		import "fmt"
		
		func main() {
				var f float64 = 1.3
				u := uint(f)
				fmt.Println(f, u)
		}`,
		out: "1.3 1\n",
	},

	{
		name: "Builtin len call as function (interface{}) parameter",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print(len("hello"))
			fmt.Print(len([]int{2,3,4}))
		}				
		`,
		out: "53",
	},

	{
		name: "Struct composite literal - side effects when not assigned",
		src: `package main

		import (
			"fmt"
			"os/exec"
		)
		
		func s() string {
			fmt.Print("s()")
			return ""
		}
		
		func main() {
			_ = exec.Error{Name: s()}
		}
		`,
		out: `s()`,
	},

	{
		name: "Struct composite literal with implicit fields",
		src: `package main

		import (
			"errors"
			"fmt"
			"os/exec"
		)
		
		func main() {
			e := exec.Error{"errorName", errors.New("error")}
			fmt.Println(e)
		}
		`,
		out: "{errorName error}\n",
	},

	{
		name: "Struct composite literal with two strings and one []string field",
		src: `package main

		import (
			"fmt"
			"os/exec"
		)
		
		func main() {
			c := exec.Cmd{
				Dir:  "/home/user/",
				Path: "git/prg",
				Args: []string{"arg1", "arg2", "arg3"},
			}
			fmt.Println(c)
			fmt.Println(c.Dir)
			fmt.Println(c.Path)
			fmt.Println(c.Args)
		}
		`,
		out: "{git/prg [arg1 arg2 arg3] [] /home/user/ <nil> <nil> <nil> [] <nil> <nil> <nil> <nil> <nil> false [] [] [] [] <nil> <nil>}\n/home/user/\ngit/prg\n[arg1 arg2 arg3]\n",
	},

	{
		name: "Struct composite literal with two explicit string fields",
		src: `package main

		import (
			"fmt"
			"os/exec"
		)
		
		func main() {
			c := exec.Cmd{
					Dir: "/home/user/",
					Path: "git/prg",
			}
			fmt.Println(c)
			fmt.Println(c.Dir)
			fmt.Println(c.Path)
		}
		
		`,
		out: "{git/prg [] [] /home/user/ <nil> <nil> <nil> [] <nil> <nil> <nil> <nil> <nil> false [] [] [] [] <nil> <nil>}\n/home/user/\ngit/prg\n",
	},

	{
		name: "Assignment to (two) struct fields",
		src: `package main

		import (
			"fmt"
			"os/exec"
		)
		
		func main() {
			c := exec.Cmd{}
			c.Dir = "/home/user/"
			c.Path = "git/prg"
			fmt.Println(c)
			fmt.Println(c.Dir)
			fmt.Println(c.Path)
		}
		`,
		out: "{git/prg [] [] /home/user/ <nil> <nil> <nil> [] <nil> <nil> <nil> <nil> <nil> false [] [] [] [] <nil> <nil>}\n/home/user/\ngit/prg\n",
	},

	{
		name: "Assignment to struct field",
		src: `package main

		import (
			"fmt"
			"os/exec"
		)
		
		func main() {
			c := exec.Cmd{
				Path: "oldPath",
			}
			fmt.Print(c.Path, ",")
			c.Path = "newPath"
			fmt.Print(c.Path)
		}
		`,
		out: "oldPath,newPath",
	},

	{
		name: "Composite literal with struct type (one string field #3)",
		src: `package main

		import (
			"fmt"
			"os/exec"
		)
		
		func main() {
			c := exec.Cmd{
				Path: "aPath",
			}
			fmt.Printf("Path is %q", c.Path)
		}
		`,
		out: `Path is "aPath"`,
	},

	{
		name: "Composite literal with struct type (one string field #2)",
		src: `package main

		import (
			"fmt"
			"os/exec"
		)
		
		func main() {
			c := exec.Cmd{
				Path: "aPath",
			}
			path := c.Path
			fmt.Printf("Path is %q", path)
		}
		`,
		out: `Path is "aPath"`,
	},

	{
		name: "Composite literal with struct type (one string field #1)",
		src: `package main

		import (
			"fmt"
			"os/exec"
		)
		
		func main() {
			c := exec.Cmd{
				Path: "aPath",
			}
			fmt.Printf("%+v", c)
		}
		`,
		out: "{Path:aPath Args:[] Env:[] Dir: Stdin:<nil> Stdout:<nil> Stderr:<nil> ExtraFiles:[] SysProcAttr:<nil> Process:<nil> ProcessState:<nil> ctx:<nil> lookPathErr:<nil> finished:false childFiles:[] closeAfterStart:[] closeAfterWait:[] goroutine:[] errch:<nil> waitDone:<nil>}",
	},

	{
		name: "Composite literal with struct type (empty)",
		src: `package main

		import (
			"fmt"
			"os/exec"
		)
		
		func main() {
			c := exec.Cmd{}
			fmt.Printf("%+v", c)
		}
		`,
		out: "{Path: Args:[] Env:[] Dir: Stdin:<nil> Stdout:<nil> Stderr:<nil> ExtraFiles:[] SysProcAttr:<nil> Process:<nil> ProcessState:<nil> ctx:<nil> lookPathErr:<nil> finished:false childFiles:[] closeAfterStart:[] closeAfterWait:[] goroutine:[] errch:<nil> waitDone:<nil>}",
	},

	{
		name: "Side effect when creating composite literals w/o assigning them",
		src: `package main

		import (
			"fmt"
		)
		
		func f(s string) int {
			fmt.Print(s, " ", "side-effect, ")
			return 10
		}
		
		func main() {
			_ = []int{f("slice"), f("slice"), 5, f("slice")}
			_ = [...]int{10: f("array")}
			_ = map[string]int{"f": f("map")}
		}
		`,
		out: "slice side-effect, slice side-effect, slice side-effect, array side-effect, map side-effect, ",
	},

	{
		name: "Conversion as function call argument with different register kind",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Println(string([]byte{97, 98, 99}))
		}
		`,
		out: "abc\n",
	},

	{
		name: "Method expression (call) on interface receiver",
		src: `package main

		import (
			"fmt"
			"time"
		)
		
		func main() {
			d, _ := time.ParseDuration("20m42s")
			s := fmt.Stringer(d)
			ss := fmt.Stringer.String(s)
			fmt.Print("s.String() is ", ss)
		}`,
		out: "s.String() is 20m42s",
	},

	{
		name: "Method expression (value) on interface receiver",
		src: `package main

		import (
			"fmt"
			"time"
		)
		
		func main() {
			d, _ := time.ParseDuration("20m42s")
			s := fmt.Stringer(d)
			m := fmt.Stringer.String
			ss := m(s)
			fmt.Print("s.String() is ", ss)
		}`,
		out: "s.String() is 20m42s",
	},

	{
		name: "Method value on interface receiver",
		src: `package main

		import (
			"fmt"
			"time"
		)
		
		func main() {
			d, _ := time.ParseDuration("20m42s")
			s := fmt.Stringer(d)
			m := s.String
			ss := m()
			fmt.Print("s.String() is ", ss)
		}
		`,
		out: "s.String() is 20m42s",
	},

	{
		name: "Method value on concrete receiver",
		src: `package main

		import (
			"fmt"
			"time"
		)

		func main() {
			var d time.Duration
			d += 7200000000000
			mv := d.Hours
			s := mv()
			fmt.Print(s)
		}`,
		out: `2`,
	},

	{
		name: "Method call on interface receiver (method defined on value receiver)",
		src: `package main

		import (
			"fmt"
			"time"
		)
		
		func main() {
			d, _ := time.ParseDuration("10m")
			s := fmt.Stringer(d)
			ss := s.String()
			fmt.Print("s.String() is ", ss)
		}`,
		out: "s.String() is 10m0s",
	},

	{
		name: "Method expression value on concrete receiver (method defined on value receiver)",
		src: `package main

		import (
			"fmt"
			"time"
		)
		
		func main() {
			var d time.Duration = 7200000000
			expr := time.Duration.Hours
			h := expr(d)
			fmt.Print(h)
		}
		`,
		out: "0.002",
	},

	{
		name: "Method expression call on concrete receiver (method defined on value receiver)",
		src: `package main

		import (
			"fmt"
			"time"
		)
		
		func main() {
			var d time.Duration = 7200000000
			h := time.Duration.Hours(d)
			fmt.Print(h)
		}		
		`,
		out: `0.002`,
	},

	{
		name: "Method call on concrete value (T* rcv on T method)",
		src: `package main

		import (
			"fmt"
			"time"
		)
		
		func main() {
			var d time.Duration
			d += 7200000000000
			dp := &d
			s := dp.Hours()
			fmt.Print(s)
		}		
		`,
		out: "2",
	},

	{
		name: "Method call on concrete value (T rcv on T method)",
		src: `package main

		import (
			"fmt"
			"time"
		)
		
		func main() {
			var d time.Duration
			d += 7200000000000
			s := d.Hours()
			fmt.Print(s)
		}
		`,
		out: "2",
	},

	{
		name: "Method call on concrete value (T rcv on *T method (an implicit & is added)",
		src: `package main

		import (
			"bytes"
			"fmt"
		)
		
		func main() {
			var b = *bytes.NewBufferString("content of (indirect) buffer")
			s := b.String()
			fmt.Print(s)
		}
		`,
		out: "content of (indirect) buffer",
	},

	{
		name: "Method call on concrete value (T* rcv on *T method (explicit &)",
		src: `package main

		import (
			"bytes"
			"fmt"
		)
		
		func main() {
			var b = *bytes.NewBufferString("content of buffer")
			s := (&b).String()
			fmt.Print(s)
		}
		`,
		out: "content of buffer",
	},

	{
		name: "Method call on concrete value (*T rcv on *T method)",
		src: `package main

		import (
			"bytes"
			"fmt"
		)
		
		func main() {
			b := bytes.NewBuffer([]byte{1, 2, 3, 4, 5})
			l := b.Len()
			fmt.Print(l)
		}
		`,
		out: "5",
	},

	{
		name: "Converting uint16 -> string",
		src: `package main

		import (
			"fmt"
		)

		func main() {
			var f uint16 = 99
			var s string = string(f)
			fmt.Print(f, ", ", s)
		}`,
		out: "99, c",
	},

	{
		name: "Converting float64 -> int (truncating)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			var f float64 = 60.001
			var i int = int(f)
			fmt.Print(f, i)
		}
		`,
		out: "60.001 60",
	},
	{
		name: "Converting int -> float64",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			var i int = 10
			var f float64 = float64(i)
			fmt.Print(i, f)
		}
		`,
		out: "10 10",
	},
	{
		name: "Converting float64 -> float32",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			var f64 float64 = 123456789.98765321
			var f32 float32 = float32(f64)
			fmt.Print(f64, f32)
		}
		`,
		out: "1.2345678998765321e+08 1.2345679e+08",
	},
	{
		name: "Converting string -> []byte",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			var str = string("hello!")
			var slice = []byte(str)
			fmt.Print("str: ", str, ", slice: ", slice)
		}
		`,
		out: `str: hello!, slice: [104 101 108 108 111 33]`,
	},
	{
		name: "Converting []byte -> string",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			var s []byte = []byte{97, 98, 99}
			var c string = string(s)
			fmt.Print("s: ", s, ", c: ", c)
		}`,
		out: `s: [97 98 99], c: abc`,
	},
	{
		name: "Converting int -> string",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			var i int = 97
			var c string = string(i)
			fmt.Print("i: ", i, ", c: ", c)
		}
		`,
		out: "i: 97, c: a",
	},
	{
		name: "Converting int -> int8 -> int (truncating result)",
		src: `package main

		import (
			"fmt"
		)

		func main() {
			var a int = 400
			var b int8 = int8(a)
			var c int = int(b)
			fmt.Print(a, b, c)
		}
		`,
		out: "400 -112 -112",
	},
	{
		name: "Slice expressions",
		src: `package main

		import (
			"fmt"
		)

		func main() {
			var s = []int{1,2,3,4,5}
			slices := [][]int{
				s[:],
				s[:0],
				s[0:0],
				s[:3],
				s[0:3],
				s[0:5],
				s[2:5],
				s[2:4],
				s[2:2],
				s[0:0:0],
				s[0:1:2],
				s[2:3:5],
				s[5:5:5],
			}
			for _, slice := range slices {
				fmt.Print(slice, len(slice), cap(slice), "\t")
			}
		}`,
		out: "[1 2 3 4 5] 5 5\t[] 0 5\t[] 0 5\t[1 2 3] 3 5\t[1 2 3] 3 5\t[1 2 3 4 5] 5 5\t[3 4 5] 3 3\t[3 4] 2 3\t[] 0 3\t[] 0 0\t[1] 1 2\t[3] 1 3\t[] 0 0\t",
	},
	{
		name: "Passing a function to Go",
		src: `package main

		import (
			"fmt"
		)

		func main() {
			f := func() {}
			fmt.Printf("%T", f)
		}`,
		out: "func()",
	},
	{
		name: "Issue #130",
		src: `package main

		import (
			"fmt"
		)

		func main() {
			a := "hello"
			a = a + " "
			a += "world!"
			fmt.Println(a)
		}
		`,
		out: "hello world!\n",
	},
	{
		name: "If statement inside function literal",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			f := func() {
				if true == false {
					fmt.Print("paradox")
				} else {
					fmt.Print("ok")
				}
			}
			f()
		}
		`,
		out: "ok",
	},
	{
		name: "Implicit repetition on global declaration using iota",
		src: `package main

		import (
			"fmt"
		)
		
		const (
			A = iota
		)
		
		const (
			B = iota
			C
		)
		
		func main() {
			fmt.Print(A, B, C)
		}
		`,
		out: "0 0 1",
	},
	{
		name: "Implicit repetition on local declaration using iota (multiple iota on same line)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			const (
				A = 10 + iota + (iota * 2)
				B
				C
			)
			fmt.Print(A, B, C)
		}
		`,
		out: "10 13 16",
	},
	{
		name: "Implicit repetition on local declaration using iota",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			const (
				A = 2 << iota
				B
				C
			)
			fmt.Print(A, B, C)
		}
		`,
		out: "2 4 8",
	},
	{
		name: "Implicit repetition on local declaration",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			const (
				A = 10
				B
			)
			fmt.Print(A, B)
		}
		`,
		out: "10 10",
	},
	{
		name: "Implicit repetition on global declaration",
		src: `package main

		import (
			"fmt"
		)
		
		const (
			A = 10
			B
		)
		
		func main() {
			fmt.Print(A, B)
		}	
		`,
		out: "10 10",
	},
	{
		name: "Iota - local constant in math expression",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			const (
				A = iota * 4
				B = iota * 2
				C = (iota + 3) * 2
			)
			fmt.Print(A, B, C)
		}
		`,
		out: "0 2 10",
	},
	{
		name: "Iota - global constant in math expression",
		src: `package main

		import (
			"fmt"
		)

		const (
			A = iota * 4
			B = iota * 2
			C = (iota + 3) * 2
		)

		func main() {
			fmt.Print(A, B, C)
		}
		`,
		out: "0 2 10",
	},
	{
		name: "Iota - local constant",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			const A = iota
			const B = iota
			const (
				C = iota
				D = iota
			)
			fmt.Print(A, B, C, D)
		}
		`,
		out: "0 0 0 1",
	},
	{
		name: "Iota - global constant",
		src: `package main

		import "fmt"
		
		const (
			A = iota
			B = iota
			C = iota
		)
		
		func main() {
			fmt.Print(A, B, C)
		}
		`,
		out: "0 1 2",
	},
	{
		name: "Identifiers",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			a := 1
			_x9 := 2
			ThisVariableIsExported := 3
			αβ := 4
			fmt.Print(a, _x9, ThisVariableIsExported, αβ)
		}
		`,
		out: "1 2 3 4",
	},
	{
		name: "Comment inside string",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			s := "/* not a comment */"
			fmt.Print(s)
		}
		`,
		out: "/* not a comment */",
	},
	{
		name: "Issue #57",
		src: `package main

		import "fmt"
		
		func main() {
			v := interface{}(3)
			switch u := v.(type) {
			default:
				fmt.Println(u)
			}
		}
		`,
		out: "3\n",
	},
	{
		name: "Builtin append - single element",
		src: `package main

			import (
				"fmt"
			)

			func main() {
				s1 := []int{10, 20, 30}
				s2 := append(s1, 40)
				s3 := append(s1, 50)
				fmt.Print(s1, s2, s3)
			}
			`,
		out: "[10 20 30] [10 20 30 40] [10 20 30 50]",
	},
	{
		name: "Builtin append on slice (variadic)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			s1 := []int{1,2,3}
			s2 := []int{4,5,6}
			s3 := append(s1, s2...)
			fmt.Print(s1,s2,s3)
		}`,
		out: "[1 2 3] [4 5 6] [1 2 3 4 5 6]",
	},

	{
		name: "Binary operation on global expression",
		src: `package main

		import (
			"fmt"
		)
		
		var A = true
		var B = false
		var OR = A || B
		
		func main() {
			fmt.Print(A, B, OR)
		}`,
		out: "true false true"},

	{
		name: "Issue #113",
		src: `package main

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
		}`,
		out: "map[20:12]\nfalse\n24\n"},

	{
		name: "Boolean global variables",
		src: `package main

		import (
			"fmt"
		)
		
		var A = true
		var B = false
		var C = A
		
		func main() {
			fmt.Print(A, B, C)
		}
		`,
		out: "true false true"},

	{
		name: "Double pointer indirection",
		src: `package main

		import "fmt"
		
		func main() {
			a := "hello"
			b := &a
			c := &b
			fmt.Print(**c)
		}
		`,
		out: "hello"},

	{
		name: "Break statement in type-switch statements",
		src: `package main

		import (
			  "fmt"
		)
		
		func main() {
			  fmt.Print("switch,")
			  switch interface{}("hey").(type) {
			  case string:
				    fmt.Print("case string,")
				    break
				    fmt.Print("???")
			  default:
				    fmt.Print("???")
			  }
			  fmt.Print("done")
		}`,
		out: "switch,case string,done"},

	{
		name: "Break statement in switch statements",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print("switch,")
			switch {
			case true:
				fmt.Print("case true,")
				break
				fmt.Print("???")
			default:
				fmt.Print("???")
			}
			fmt.Print("done")
		}
		`,
		out: "switch,case true,done"},

	{
		name: "Break statement in for statements",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Printf("a")
			for {
				fmt.Printf("b")
				break
				fmt.Printf("???")
			}
			fmt.Printf("c")
		}
		`,
		out: "abc"},

	{
		name: "Const declaration inside function body",
		src: `package main

		import "fmt"
		
		func main() {
			const A = 10
			fmt.Println(A)
		}`,
		out: "10\n"},

	{
		name: "Issue #78",
		src: `package main

		func main() {
			a := 0
			f := func() { _ = a }
			_ = f
		}
		`,
		out: ""},

	{
		name: "Counter based on gotos/labels",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			count := 0
		loop:
			fmt.Print(",")
			if count > 9 {
				goto end
			}
			fmt.Print(count)
			count++
			goto loop
		end:
		}
		`,
		out: ",0,1,2,3,4,5,6,7,8,9,"},

	{
		name: "Goto label - Forward jump out of scope",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print("out1,")
			{
				fmt.Print("goto,")
				goto out
				fmt.Print("error")
			}
			out:
			fmt.Print("out2")
		}
		`,
		out: "out1,goto,out2"},

	{
		name: "Goto label - Simple forwaring goto",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print("a")
			goto L
			fmt.Print("???")
		L:
			fmt.Print("b")
		}
		`,
		out: "ab"},

	{
		name: "Goto label - Simple one-step forwarding goto",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print("a")
			goto L
		L:
			fmt.Print("b")
		}
		`,
		out: "ab"},

	{
		name: "Package variable shadowing in closure body",
		src: `package main

		import (
			"fmt"
		)
		
		var A = 1
		
		func main() {
			fmt.Print(A)
			func() {
				fmt.Print(A)
				A := 20
				fmt.Print(A)
			}()
			fmt.Print(A)
		}
		`,
		out: "11201"},

	{
		name: "Package variable in closure",
		src: `package main

		import (
			"fmt"
		)
		
		var A = 1
		
		func main() {
			fmt.Print(A)
			A = 2
			func() {
				fmt.Print(A)
				A = 3
				fmt.Print(A)
				f := func() {
					A = 4
					fmt.Print(A)
				}
				fmt.Print(A)
				f()
			}()
			fmt.Print(A)
		}
		`,
		out: "123344"},

	{
		name: "Package with both predefined and not predefined variables",
		src: `package main

		import (
			"fmt"
			"testpkg"
		)
		
		var Center = "msg"
		
		func main() {
			predefinedC := testpkg.Center
			fmt.Println(predefinedC)
			c := Center
			fmt.Println(c)
		}
		`,
		out: "{5 42}\nmsg\n"},

	{
		name: "Function literal call",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print("start,")
			func() {
				fmt.Print("f,")
			}()
			fmt.Print("end")
		}
		`,
		out: "start,f,end"},

	{
		name: "Builtin close",
		src: `package main

		import "fmt"
		
		func main() {
			c := make(chan int)
			fmt.Print("closing: ")
			close(c)
			fmt.Print("ok")
		}
		`,
		out: "closing: ok"},

	{
		name: "Builtin make - with and w/o cap argument",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			s1 := make([]int, 2)
			fmt.Print("s1: ", len(s1), cap(s1))
			s2 := make([]int, 2, 5)
			fmt.Print("/s2: ", len(s2), cap(s2))
		}
		`,
		out: "s1: 2 2/s2: 2 5"},

	{
		name: "Switch without expression",
		src: `package main

		import "fmt"
		
		func main() {
			fmt.Print("switch: ")
			switch {
			case true:
				fmt.Print("true")
			case false:
				fmt.Print("false")
			}
			fmt.Print(".")
		}
		`,
		out: "switch: true."},

	{
		name: "Package variable - call to a predefined function with package variable as argument",
		src: `package main

		import "fmt"
		
		var A = 3
		
		func main() {
			fmt.Print(A)
		}
		`,
		out: "3"},

	{
		name: "Package variable - int variable set by a package function",
		src: `package main

		import "fmt"
		
		var A = 3
		
		func SetA() {
			A = 4
		}
		
		func main() {
			a := A
			fmt.Print(a)
			SetA()
			a = A
			fmt.Print(a)
		}
		`,
		out: "34"},

	{
		name: "Package variable - int variable with math expression as value",
		src: `package main

		import "fmt"
		
		var A = F() + 3 + (2 * F())
		
		func F() int {
			return 42
		}
		
		func main() {
			a := A
			fmt.Print(a)
		}
		`,
		out: "129"},

	{
		name: "Package variable - int variable with package function as value",
		src: `package main

		import "fmt"
		
		var A = F()
		
		func F() int {
			return 42
		}
		
		func main() {
			a := A
			fmt.Print(a)
		}
		`,
		out: "42"},

	{
		name: "Package variable - int variable and string variable",
		src: `package main

		import "fmt"
		
		var A = 3
		var B = "hey!"
		
		func main() {
			a := A
			fmt.Print(a)
			b := B
			fmt.Print(b)
		
		}
		`,
		out: "3hey!"},

	{
		name: "Package variable - one int variable",
		src: `package main

		import "fmt"
		
		var A = 3
		
		func main() {
		
			a := A
			fmt.Print(a)
		
		}
		`,
		out: "3"},

	{
		name: "Break - For range (no label)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print("start, ")
			for _, v := range []int{1, 2, 3} {
				fmt.Print("v: ", v, ", ")
				if v == 2 {
					fmt.Print("break, ")
					break
				}
				fmt.Print("no break, ")
			}
			fmt.Print("end")
		}
		`,
		out: "start, v: 1, no break, v: 2, break, end"},

	{
		name: "Continue - For range (no label)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			for _, v := range []int{1, 2, 3} {
				fmt.Print(v, ":cont?")
				if v == 2 {
					fmt.Print("yes!")
					continue
					panic("?")
				}
				fmt.Print("no!")
			}
		}
		`,
		out: "1:cont?no!2:cont?yes!3:cont?no!"},

	{
		name: "Init function with package variables",
		src: `package main

		import (
			"fmt"
		)
		
		func init() {
			fmt.Print("init1!")
		}
		
		var A = F()
		
		func F() int {
			fmt.Print("F!")
			return 42
		}
		
		func main() {
			fmt.Print("main!")
		}
		`,
		out: "F!init1!main!"},

	{
		name: "Init function - one",
		src: `package main

		import (
			"fmt"
		)
		
		func init() {
			fmt.Print("init!")
		}
		
		func main() {
			fmt.Print("main!")
		}
		`,
		out: "init!main!"},

	{
		name: "Init function - three",
		src: `package main

		import (
			"fmt"
		)
		
		func init() {
			fmt.Print("init1!")
		}
		
		func main() {
			fmt.Print("main!")
		}
		
		func init() {
			fmt.Print("init2!")
		}
		
		func init() {
			fmt.Print("init3!")
		}
		`,
		out: "init1!init2!init3!main!"},

	{
		name: "Package variables initialization",
		src: `package main

		import "fmt"
		
		var A = F()
		
		func F() int {
			fmt.Print("F has been called!")
			return 200
		}
		
		func main() {
			fmt.Print("main!")
		}
		`,
		out: "F has been called!main!"},

	{
		name: "Closure - complex tests (multiple clojures)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			A := 10
			B := 20
			f1 := func() {
				fmt.Print(A)
				A = 10
			}
			f2 := func() {
				fmt.Print(B)
				A = B + 2
			}
			f3 := func() {
				fmt.Print(A + B)
				B = A + B
			}
			f1()
			f2()
			f3()
		}
		`,
		out: "102042"},

	{
		name: "Closure - one level and two levels variable writing (int)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			A := 5
			fmt.Print("main:", A)
			f := func() {
				fmt.Print(", f:", A)
				A = 20
				g := func() {
					fmt.Print(", g:", A)
					A = 30
					fmt.Print(", g:", A)
				}
				fmt.Print(", f:", A)
				g()
				fmt.Print(", f:", A)
			}
			fmt.Print(", main:", A)
			f()
			fmt.Print(", main:", A)
		}
		`,
		out: "main:5, main:5, f:5, f:20, g:20, g:30, f:30, main:30"},

	{
		name: "Closure - one level variable writing (int)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			A := 5
			fmt.Print("main:", A)
			f := func() {
				fmt.Print(", f:", A)
				A = 20
				fmt.Print(", f:", A)
			}
			fmt.Print(", main:", A)
			f()
			fmt.Print(", main:", A)
		}
		`,
		out: "main:5, main:5, f:5, f:20, main:20"},

	{
		name: "Closure - one level and two levels variable reading (int)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			A := 10
			f := func() {
				fmt.Print("f: ", A, ", ")
				g := func() {
					fmt.Print("g: ", A)
				}
				g()
			}
			f()
		}`,
		out: "f: 10, g: 10"},

	{
		name: "Closure - one level variable reading (int and string)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			A := 10
			B := "hey"
			f := func() {
				a := A
				fmt.Print("f: ", a, ",")
				b := B
				fmt.Print("f: ", b)
			}
			f()
		}
		`,
		out: "f: 10,f: hey"},

	{
		name: "Closure - one level variable reading (int)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			A := 10
			f := func() {
				a := A
				fmt.Print("f: ", a)
			}
			f()
		}
		`,
		out: "f: 10"},

	{
		name: "Closure - one level variable reading (interface{})",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			A := interface{}(5)
			f := func() {
				fmt.Printf("%v (%T)", A, A)
			}
			f()
		}
		`,
		out: "5 (int)"},

	{
		name: "For range on interface{} slice",
		src: `package main

		import "fmt"
		
		func main() {
		
			mess := []interface{}{3, "hey", 5.6, 2}
			for e, v := range mess {
				fmt.Print(e, v, ",")
			}
		
		}
		`,
		out: "0 3,1hey,2 5.6,3 2,"},

	{
		name: "For range on empty int slice",
		src: `package main

		import "fmt"
		
		func main() {
		
			fmt.Printf("start-")
			for _ = range []int{} {
				fmt.Print("looping!")
			}
			fmt.Printf("-end")
		
		}
		`,
		out: "start--end"},

	{
		name: "For range on int slice, taking only values",
		src: `package main

		import "fmt"
		
		func main() {
		
			for _, v := range []int{3, 4, 5} {
				fmt.Print(v)
			}
		
		}
		`,
		out: "345"},

	{
		name: "For range on int slice, taking only indexes",
		src: `package main

		import "fmt"
		
		func main() {
		
			for i := range []int{3, 4, 5} {
				fmt.Print(i)
			}
		
		}
		`,
		out: "012"},

	{
		name: "For range with simple assignment (no declaration)",
		src: `package main

		import "fmt"
		
		func main() {
		
			i := 0
			e := 0
			for i, e = range []int{3, 4, 5} {
				fmt.Print(i, e, ",")
			}
		}
		`,
		out: "0 3,1 4,2 5,"},

	{
		name: "For range with no assignments",
		src: `package main

		import "fmt"
		
		func main() {
		
			for _ = range []int{1, 2, 3} {
				fmt.Print("?")
			}
		}
		`,
		out: "???"},

	{
		name: "For range on map[string]int",
		src: `package main

		import "fmt"
		
		func main() {
		
			m := map[string]int{"twelve": 12}
			for k, v := range m {
				fmt.Print(k, " is ", v, ",")
			}
		
		}
		`,
		out: "twelve is 12,"},

	{
		name: "For range on int slice",
		src: `package main

		import "fmt"
		
		func main() {
		
			for i, v := range []int{4, 5, 6} {
				fmt.Print(i, v, ",")
			}
		
		}
		`,
		out: "0 4,1 5,2 6,"},

	{
		name: "For range on string",
		src: `package main

		import "fmt"
		
		func main() {
		
			for i, c := range "Hello" {
				fmt.Print(i, c, ",")
			}
		
		}
		`,
		out: "0 72,1 101,2 108,3 108,4 111,"},

	{
		name: "Issue #92",
		src: `package main

		import "fmt"
		
		func main() {
		
			fmt.Println("a", 10)
			fmt.Println("b", 2.5)
		
		}`,
		out: "a 10\nb 2.5\n"},

	{
		name: "Builtin len on maps",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			m := map[string]int{}
			fmt.Print(len(m))
			m["one"] = 1
			fmt.Print(len(m))
			m["one"] = 1
			m["one"] = 2
			fmt.Print(len(m))
			m["one"] = 1
			m["two"] = 2
			m["three"] = 3
			m["five"] = 4
			fmt.Print(len(m))
		}
		`,
		out: "0114"},

	{
		name: "Builtin function calls can be used directly as function arguments",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print(len([]int{1,2,3}))
			fmt.Print(cap(make([]int, 1, 10)))
		}
		`,
		out: "310"},

	{
		name: "Checking if map has keys",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			m := map[string]int{"one": 1, "three": 0}
			v1, ok1 := m["one"]
			v2, ok2 := m["two"]
			v3, ok3 := m["three"]
			fmt.Print(v1, v2, v3, ",")
			fmt.Print(ok1, ok2, ok3)
		}
		`,
		out: "1 0 0,true false true"},

	{
		name: "Map index assignment",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			m := map[string]int{}
			fmt.Print(m, ",")
			m["one"] = 1
			m["four"] = 4
			fmt.Print(m, ",")
			m["one"] = 10
			fmt.Print(m)
		}
		`,
		out: "map[],map[four:4 one:1],map[four:4 one:10]"},

	{
		name: "Builtin delete",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			m := map[string]string{"key": "value", "key2": "value2"}
			fmt.Print(m, ",")
			delete(m, "key")
			fmt.Print(m)
		}
		`,
		out: "map[key:value key2:value2],map[key2:value2]"},

	{
		name: "Multiple assignment with blank identifier",
		src: `package main

		import "fmt"
		
		func main() {
		
			_, a := 1, 2
			fmt.Println(a)
		
		}
		`,
		out: "2\n"},

	{
		name: "Builtin make - Map with no size",
		src: `package main

		import "fmt"
		
		func main() {
		
			m := make(map[string]int)
			fmt.Printf("%v (%T)", m, m)
		}
		`,
		out: "map[] (map[string]int)"},

	{
		name: "Builtin new - creating a string pointer and updating it's value",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			pi := new(string)
			pv := *pi
			fmt.Print("*pi:", pv, ",")
			*pi = "newv"
			pv = *pi
			fmt.Print("*pi:", pv)
		}`,
		out: "*pi:,*pi:newv"},

	{
		name: "Builtin new - creating two int pointers and comparing their values",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			pi := new(int)
			fmt.Printf("pi: %T,", pi)
			pi2 := new(int)
			pi2 = pi
			equal := pi == pi2
			fmt.Print("equal?", equal)
		}
		`,
		out: "pi: *int,equal?true"},

	{
		name: "Builtin new - creating an int pointer",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			pi := new(int)
			fmt.Printf("pi: %T", pi)
		}
		`,
		out: "pi: *int"},

	{
		name: "Pointer indirection as expression",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			a := "a"
			b := &a
			c := *b
			fmt.Print(c)
		}
		`,
		out: "a"},

	{
		name: "Pointer indirection assignment",
		src: `package main

		import "fmt"
		
		func main() {
		
			a := "a"
			b := "b"
			c := "c"
		
			pb := &b
			*pb = "newb"
		
			fmt.Print(a, b, c)
		}`,
		out: "anewbc"},

	{
		name: "Address operator on strings declared with 'var'",
		src: `package main

		import "fmt"
		
		func main() {
		
			var s1 = "hey"
			var s2 = "hey"
			var p1 = &s1
			var p2 = &s1
			var p3 = &s2
			c1 := p1 == p2
			c2 := p2 == p3
			c3 := p1 == p3
			fmt.Println(c1)
			fmt.Println(c2)
			fmt.Println(c3)
		
		}
		`,
		out: "true\nfalse\nfalse\n"},

	{
		name: "Address operator on strings declared with ':='",
		src: `package main

		import "fmt"

		func main() {

			s1 := "hey"
			s2 := "hoy"
			p1 := &s1
			p2 := &s1
			p3 := &s2
			c1 := p1 == p2
			c2 := p2 == p3
			c3 := p1 == p3
			fmt.Println(c1)
			fmt.Println(c2)
			fmt.Println(c3)

		}
		`,
		out: "true\nfalse\nfalse\n"},

	{
		name: "Switch with non-immediate expression in a case",
		src: `package main

		import "fmt"
		
		func main() {
			a := 6
			b := 7
			switch 42 {
			case a * b:
				fmt.Print("a * b")
			default:
				fmt.Print("default")
			}
		}
		`,
		out: "a * b"},

	{
		name: "Issue #86",
		src: `package main

		import "fmt"
		
		func main() {
			a := "ext"
			switch a := 3; a {
			case 3:
				fmt.Println(a)
				a := "internal"
				fmt.Println(a)
			}
			fmt.Println(a)
		
		}`,
		out: "3\ninternal\next\n"},

	{
		name: "Builtin 'cap'",
		src: `package main

		import (
			"fmt"
		)

		func main() {
			s1 := make([]int, 2, 10)
			c1 := cap(s1)
			fmt.Print(c1, ",")
			s2 := []int{1, 2, 3}
			c2 := cap(s2)
			fmt.Print(c2)
		}
		`,
		out: "10,3"},

	// {"Package function calling a post-declared package function",
	// 	`package main

	// 	func f() { g() }

	// 	func g() { }

	// 	func main() { }
	// 	`,
	// output: ""},

	{
		name: "Function literal assigned to underscore",
		src: `package main
		
		func main() {
			_ = func() { }
		}`,
		out: ""},
	{
		name: "Return register is determined by function type, not value type",
		src: `package main

		import (
			  "fmt"
		)
		
		func f() interface{} {
			  return "hey"
		}
		
		func main() {
			  a := f()
			  fmt.Println(a)
		}`,
		out: "hey\n"},

	{
		name: "Package function with many return values (different types)",
		src: `
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
		`,
		out: "2 42 33 11 12"},

	{
		name: "Correct order in assignments",
		src: `package main

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
		`,
		out: "fg"},

	{
		name: "Issue #76",
		src: `package main

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
		out: "[]int[]uint8[] []int\n[] []uint8\n"},

	{
		name: "Increment assignment should evaluate expression only once",
		src: `package main

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
		out: "i()[6 2 3]\n"},

	{
		name: "Package function with many return values (same type)",
		src: `package main

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
		`,
		out: "100"},

	{
		name: "Issue #67",
		src: `package main

		func f() {
			f := "hi!"
			_ = f
		}
		
		func main() {
		
		}`,
		out: ""},

	{
		name: "Issue #75",
		src: `package main

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
		}`,
		out: "flag is true\nflag is false\n"},

	{
		name: "Functions with strings as input/output arguments",
		src: `package main

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
		out: "4"},

	// {"Named return parameters",
	// 	`package main

	// 	import "fmt"

	// 	func f1() (a int, b int) {
	// 		a = 10
	// 		b = 20
	// 		return
	// 	}

	// 	func f2() (a int, b int) {
	// 		return 30, 40
	// 	}

	// 	func f3(bool) (a int, b int, c string) {
	// 		a = 70
	// 		b = 80
	// 		c = "str2"
	// 		return
	// 	}

	// 	func f4() (a int, b int, c float64) {
	// 		a = 90
	// 		c = 100
	// 		return
	// 	}

	// 	func main() {
	// 		v11, v12 := f1()
	// 		v21, v22 := f2()
	// 		v31, v32, v33 := f3(true)
	// 		v41, v42, v43 := f3(false)
	// 		v51, v52, v53 := f4()
	// 		fmt.Println(v11, v12)
	// 		fmt.Println(v21, v22)
	// 		fmt.Println(v31, v32, v33)
	// 		fmt.Println(v41, v42, v43)
	// 		fmt.Println(v51, v52, v53)
	// 	}
	// 	`,
	// output: "10 20\n30 40\n70 80 str2\n70 80 str2\n90 0 100\n"},

	{
		name: "Anonymous function arguments",
		src: `package main

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
		`,
		out: "10\n"},

	{
		name: "Bit operators & (And) and | (Or)",
		src: `package main

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
		`,
		out: "15 1\n"},

	{
		name: "Bit operators ^ (Xor) and &^ (bit clear = and not)",
		src: `package main

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
		`,
		out: "14 4\n"},

	{
		name: "Channels - Reading and writing",
		src: `package main

		import "fmt"
		
		func main() {
			ch := make(chan int, 4)
			ch <- 5
			v := <-ch
			fmt.Print(v)
		}
		`,
		out: "5"},

	{
		name: "Variable swapping",
		src: `package main

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
		out: "10 20\n10 20\n20 10\n10 20\n"},

	{
		name: "Some maths",
		src: `package main

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
		`,
		out: "75,-9,1386,0,33,1797,1377,1335,15,"},

	{
		name: "Selector (predefined struct)",
		src: `package main

		import (
			"fmt"
			"testpkg"
		)
		
		func main() {
			center := testpkg.Center
			fmt.Println(center)
		}
		`,
		out: "{5 42}\n"},
	{
		name: "Implicit return",
		src: `package main

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
		out: "fg"},

	{
		name: "Recover",
		src: `package main
		
		func main() {
			recover()
			v := recover()
			_ = v
		}
		`,
		out: ""},

	{
		name: "Defer - Calling f ang g (package level functions)",
		src: `package main

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
		out: "g\nf\n"},

	{
		name: "Dot import (predefined)",
		src: `package main
		
		import . "fmt"
		
		func main() {
			Println("hey")
		}`,
		out: "hey\n"},

	{
		name: "Import (predefined) with explicit package name",
		src: `package main
		
		import f "fmt"
		
		func main() {
			f.Println("hey")
		}`,
		out: "hey\n"},

	{
		name: "Import (predefined) with explicit package name (two packages)",
		src: `package main
		
		import f "fmt"
		import f2 "fmt"
		
		func main() {
			f.Println("hey")
			f2.Println("oh!")
		}`,
		out: "hey\noh!\n"},

	{
		name: "Recycling of registers",
		src: `
		package main

		import "fmt"

		func main() {
			a := 10
			b := a + 32
			c := 88
			fmt.Println(a, b, c)
		}
		`,
		out: "10 42 88\n"},

	{
		name: "Assignment to _ evaluates expression",
		src: `
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
		out: "ffffff"},

	{
		name: "Order of evaluation - Composite literal values",
		src: `
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
		`,
		out: "abcde"},

	{
		name: "Single assignment",
		src: `
		package main

		import "fmt"

		func main() {
			var a int
			var c string

			a = 10
			c = "hi"

			fmt.Print(a, c)
		}
		`,
		out: "10hi"},

	{
		name: "Multiple assignment",
		src: `package main

		import "fmt"
		
		func main() {
			a, b := 6, 7
			fmt.Print(a, b)
		}
		`,
		out: "6 7"},

	{
		name: "Assignment with constant int value (addition)",
		src: `package main

		import "fmt"
		
		func main() {
			a := 4 + 5
			fmt.Print(a)
		}
		`,
		out: "9"},

	{
		name: "Assignment with addition value (non constant)",
		src: `package main

		import "fmt"
		
		func main() {
			a := 5
			b := 10
			c := a + 2 + b + 30
			fmt.Print(c)
		}`,
		out: "47"},

	{
		name: "Assignment with math expression (non constant)",
		src: `
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
		out: ""},

	{
		name: "Function call assignment (2 to 1) - Predefined function with two return values",
		src: `
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
		out: "5 42 33\n"},
	{
		name: "Type assertion assignment",
		src: `
		package main

		func main() {
			a := interface{}(10)
			n, ok := a.(int)

			_ = n
			_ = ok
			return
		}
		`,
		out: ""},
	{
		name: "Addition (+=) and subtraction (-=) assignments",
		src: `package main

		import "fmt"
		
		func main() {
			var a, b, c, d int
			fmt.Print(a, b, c, d)
			a = 10
			b = a
			fmt.Print(a, b)
			a += 40
			fmt.Print(a)
			c = a
			fmt.Print(a, b, c)
			a -= 5
			fmt.Print(a)
			d = a
			fmt.Print(a, b, c, d)
		}
		`,
		out: "0 0 0 010 105050 10 504545 10 50 45"},

	{
		name: "Other forms of assignment",
		src: `
		package main

		import "fmt"

		func print(v interface{}) {
			fmt.Print(v, ",")
		}

		func main() {

			a := 27
			print(a) // 27
			a *= 6
			print(a) // 162
			a /= 3
			print(a) // 54
			a %= 16
			print(a) // 6
			a &= 10
			print(a) // 2
			a |= 21
			print(a) // 23
			a <<= 3
			print(a) // 184
			a >>= 4	
			print(a) // 11

			print("-")

			a = 5
			print(a) // 5
			a += 3 
			print(a) // 8
			a -= 2
			print(a) // 6
			a *= 6
			print(a) // 36
			a /= 3
			print(a) // 12
			a &= 7
			print(a) // 4
			a |= 3
			print(a) // 7
			a &^= 1
			print(a) // 6
			a <<= 2
			print(a) // 24
			a >>= 1
			print(a) // 12

		}
		`,
		out: "27,162,54,6,2,23,184,11,-,5,8,6,36,12,4,7,6,24,12,"},

	{
		name: "Slice index assignment",
		src: `
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
		`,
		out: "[2 2 3]\n[a b d d]\n"},

	{
		name: "Arrays - Composite literal, len",
		src: `
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
		`,
		out: "[1 2 3] [a b c d]\n3 4"},

	{
		name: "Empty int slice composite literal",
		src: `
		package main

		import "fmt"

		func main() {
			a := []int{};
			fmt.Println(a)
		}
		`,
		out: "[]\n"},

	{
		name: "Empty string slice composite literal",
		src: `
		package main

		func main() {
			a := []string{};
			_ = a
		}
		`,
		out: ""},

	{
		name: "Empty byte slice composite literal",
		src: `
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
		out: "[] []"},

	{
		name: "Empty map composite literal",
		src: `
		package main

		func main() {
			m := map[string]int{}
			_ = m
		}
		`,
		out: ""},

	{
		name: "Function literal definition - (0 in, 0 out)",
		src: `
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
		out: ""},

	{
		name: "Function literal definition - (0 in, 1 out)",
		src: `package main

		import "fmt"
		
		func main() {
			f := func() int {
				a := 20
				fmt.Print("f called")
				return a
			}
			_ = f
		}
		`,
		out: ""},

	{
		name: "Function literal definition - (1 in, 0 out)",
		src: `package main

		import "fmt"
		
		func main() {
			f := func(a int) {
				fmt.Print("f called")
				b := a + 20
				_ = b
			}
			_ = f
		}
		`,
		out: ""},

	{
		name: "Converting from int to string",
		src: `
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
		out: ""},

	{
		name: "Converting from int to interface{}",
		src: `
		package main

		func main() {
			a := 97
			b := interface{}(a)
			_ = b
			return
		}
		`,
		out: ""},

	{
		name: "Constant boolean",
		src: `package main

		import "fmt"
		
		func main() {
			var a, b int
			var c, d bool
			fmt.Print(a, b, c, d)
			a = 1
			b = 2
			fmt.Print(a, b, c, d)
			c = false
			d = 4 < 5
			fmt.Print(a, b, c, d)
		}
		`,
		out: "0 0 false false1 2 false false1 2 false true"},

	{
		name: "Comparison operators",
		src: `package main

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
		out: "truefalsetruefalsefalsetrue"},

	{
		name: "Logic AND and OR operators",
		src: `
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
		out: ""},

	{
		name: "Short-circuit evaluation",
		src: `
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
		`,
		out: ""},

	{
		name: "Operator ! (not)",
		src: `
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
		out: ""},

	{
		name: "Number negation (unary operator '-')",
		src: `
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
		out: "-45"},

	{
		name: "Go functions as expressions",
		src: `
		package main

		import "fmt"

		func main() {
			f := fmt.Println
			print(f)
		}
		`,
		out: ""},

	{
		name: "String concatenation (constant)",
		src: `
		package main

		import "fmt"

		func main() {
			a := "s";
			b := "ee" + "ff";

			fmt.Println(a)
			fmt.Print(b)
		}
		`,
		out: "s\neeff"},

	{
		name: "String concatenation (non constant)",
		src: `
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
		out: "hello world"},

	{
		name: "Indexing",
		src: `package main

		import "fmt"

		func main() {
			s := []int{1, 42, 3}
			second := s[1]
			fmt.Println(second)
		}`,
		out: "42\n"},
	{
		name: "If statement with else",
		src: `package main

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
		`,
		out: "thenc=1"},

	{
		name: "If statement with else",
		src: `package main

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
		`,
		out: "else20"},

	{
		name: "If with init assignment",
		src: `package main

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
		`,
		out: "thenx is 11"},

	{
		name: "For statement",
		src: `package main

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
		`,
		out: "i=0,i=1,i=2,i=3,i=4,i=5,i=6,i=7,i=8,i=9,sum=20"},

	{
		name: "Switch statement",
		src: `package main

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
		`,
		out: "case 1a=20"},

	{
		name: "Switch statement with fallthrough",
		src: `package main

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
		`,
		out: "case 2case 330"},

	{
		name: "Switch statement with default",
		src: `package main

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
		`,
		out: "default80"},

	{
		name: "Switch statement with default and fallthrough",
		src: `package main

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
		`,
		out: "defaultcase 2case 330"},

	{
		name: "Type switch statement",
		src: `
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
		out: "20"},

	{
		name: "Function literal call (0 in, 0 out)",
		src: `package main

		import "fmt"
		
		func main() {
			f := func() { fmt.Print("f called") }
			f()
		}
		`,
		out: "f called"},

	{
		name: "Function literal call (0 in, 1 out)",
		src: `
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
		`,
		out: "25"},

	{
		name: "Function literal call - Shadowing package level function",
		src: `package main

		import "fmt"
		
		func f() {
			panic("")
		}
		
		func main() {
			f := func() {
				fmt.Print("local function")
				// No panic.
			}
			f()
			return
		}
		`,
		out: "local function"},

	{
		name: "Package function call",
		src: `
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
		out: "a"},

	{
		name: "Package function 'inc'",
		src: `package main

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
		`,
		out: "21"},

	{
		name: "Package function with one return value",
		src: `package main

		import "fmt"

		func five() int {
			return 5
		}

		func main() {
			a := five()
			fmt.Print(a)
		}`,
		out: "5"},

	{
		name: "Package function - 'floats'",
		src: `package main

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
		out: "5.6 -432.12\n"},
	{
		name: "Package function with one parameter (not used)",
		src: `
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
		`,
		out: "12"},

	{
		name: "Package function with arguments that need evaluation",
		src: `
		package main

		import "fmt"

		func f(a, b, c int) {
			fmt.Println("a, b, c: ", a, b, c)
			return
		}

		func g(slice []int) {
			fmt.Println(slice, "has len", len(slice))
			return
		}

		func main() {
			a := 3
			b := 2
			c := 5
			f(a, (b*2)+a-a, c)
			g([]int{a,b,c})
		}
		`,
		out: "a, b, c:  3 4 5\n[3 2 5] has len 3\n"},

	{
		name: "Builtin len (with a constant argument)",
		src: `
		package main

		import "fmt"

		func main() {
			a := len("abc");
			fmt.Println(a)
		}
		`,
		out: "3\n"},

	{
		name: "Builtin len",
		src: `package main

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
		out: "534"},

	{
		name: "Builtin print",
		src: `
		package main

		func main() {
			print(42)
			print("hello")
			a := 10
			print(a)
		}
		`,
		out: ""},

	{
		name: "Builtin make - map",
		src: `
		package main

		func main() {
			var m map[string]int
			m = make(map[string]int, 2)
			_ = m
		}
		`,
		out: ""},

	{
		name: "Builtin copy",
		src: `package main

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
		`,
		out: "dst: [10 20 30] n: 3\ndst2: [10 20]\n"},
	{
		name: "Function which calls both predefined and not predefined functions",
		src: `package main

		import "fmt"
		
		func scriggoFunc() {
			fmt.Println("scriggoFunc()")
		}

		func main() {
			scriggoFunc()
			fmt.Println("main()")
		}`,
		out: "scriggoFunc()\nmain()\n"},
	{
		name: "Predefined function call (0 in, 0 out)",
		src: `
		package main

		import "testpkg"

		func main() {
			testpkg.F00()
			return
		}
		`,
		out: ""},

	{
		name: "Predefined function call (0 in, 1 out)",
		src: `
		package main

		import "testpkg"

		func main() {
			v := testpkg.F01()
			_ = v
			return
		}
		`,
		out: ""},

	{
		name: "Predefined function call (1 in, 0 out)",
		src: `
		package main

		import "testpkg"

		func main() {
			testpkg.F10(50)
			return
		}`,
		out: ""},

	{
		name: "Predefined function call (1 in, 1 out)",
		src: `
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
		`,
		out: "42"},

	{
		name: "Predefined function call (2 in, 1 out) (with surrounding variables)",
		src: `
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
		out: ""},

	{
		name: "Predefined function call of StringLen",
		src: `
		package main

		import "testpkg"

		func main() {
			a := testpkg.StringLen("zzz")
			_ = a
		}
		`,
		out: ""},

	{
		name: "Predefined function call of fmt.Println",
		src: `
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
		`,
		out: "hello, world!\n42\nhi!\n1 2 3\nhi! hi! [3 4 5]\n"},

	{
		name: "Predefined function call f(g())",
		src: `package main

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
		out: "a is 42 and b is 33\n"},

	{
		name: "Reading a predefined int variable",
		src: `
		package main

		import "testpkg"

		func main() {
			a := testpkg.A
			testpkg.PrintInt(a)
		}
		`,
		out: "20"},

	{
		name: "Writing a predefined int variable",
		src: `
		package main

		import "testpkg"

		func main() {
			testpkg.PrintInt(testpkg.B)
			testpkg.B = 7
			testpkg.PrintString("->")
			testpkg.PrintInt(testpkg.B)
		}
		`,
		out: "42->7"},

	{
		name: "Pointer declaration (nil int pointer)",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			var a *int
			fmt.Print(a)
		}
		`,
		out: "<nil>"},

	{
		name: "Scriggo function 'repeat(string, int) string'",
		src: `
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
		out: ""},

	{
		name: "Slice of int slices assignment",
		src: `
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
		out: ""},

	{
		name: "Multiple predefined function calls",
		src: `
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
		out: ""},

	{
		name: "Predefined function 'Swap'",
		src: `
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
		`,
		out: "heyhey3 3"},

	{
		name: "Many Scriggo functions (swap, sum, fact)",
		src: `
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
		out: "a:6,b:2,c:2,d:6,e:16,f:40330,"},
	{
		name: "f(g()) call (advanced test)",
		src: `package main

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
		out: "a is 9 and b is 144\n",
	},

	{
		name: "Iota - local constant in math expression (with float numbers)",
		src: `package main

		import (
			"fmt"
		)

		func main() {
			const (
				A = (iota * iota) + 10
				B = A * (iota * 2.341)
				C = A + B + iota / 2.312
			)
			fmt.Print(A, B, C)
		}
		`,
		out: "10 23.41 34.27505190311419",
	},

	{
		name: "Out of memory: OpAppend ",
		src: `package main
	
		import "fmt"
	
		func main() {
			fmt.Println("12345678901234567890" + "12345678901234567890")
		}`,
		out: "1234567890123456789012345678901234567890\n",
	},

	{
		name: "Multiple assignment to slices",
		src: `package main

		import "fmt"

		func triplet() (int, string, string) {
			return 20, "new1", "new2"
		}

		func main() {
			ss := []string{"old1", "old2", "old3"}
			is := []int{1,2,3}
			fmt.Println(ss, is)
			is[0], ss[1], ss[0] = triplet()
			fmt.Println(ss, is)
		}
		`,
		out: "[old1 old2 old3] [1 2 3]\n[new2 new1 old3] [20 2 3]\n",
	},
}

//go:generate scriggo embed -f packages.Scriggofile -v -o packages_test.go
var goPackages scriggo.PackageLoader

func TestProgram(t *testing.T) {
	if goPackages == nil {
		panic("goPackages declared but not initialized. Run 'go generate scriggo/test' with the latest version of scriggo installed to create the package declaration file.")
	}
	for _, cas := range stmtTests {
		if cas.name == "" {
			cas.name = fmt.Sprintf("(unnamed test with source %q)", cas.src)
		}
		t.Run(cas.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r != nil {
					panic(fmt.Errorf("%s panicked: %s", cas.name, r))
				}
			}()
			r := scriggo.MapStringLoader{"main": cas.src}
			program, err := scriggo.LoadProgram(scriggo.Loaders(r, goPackages), &scriggo.LoadOptions{LimitMemorySize: true})
			if err != nil {
				t.Errorf("test %q, compiler error: %s", cas.name, err)
				return
			}
			backupStdout := os.Stdout
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
			err = program.Run(&scriggo.RunOptions{MaxMemorySize: 1000000})
			if err == nil {
				if cas.err != nil {
					t.Errorf("test %q, expecting error %#v, got not error", cas.name, cas.err)
				}
			} else {
				if cas.err == nil {
					t.Errorf("test %q, execution error: %s", cas.name, err)
				} else if err != cas.err {
					t.Errorf("test %q, expecting error %#v, got error %#v", cas.name, cas.err, err)
				}
				return
			}
			writer.Close()

			output := <-out

			// // Tests if output matches.
			if cas.out != output {
				t.Errorf("test %q: expecting output %q, got %q", cas.name, cas.out, output)
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
