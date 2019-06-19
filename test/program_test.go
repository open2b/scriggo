// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"text/tabwriter"
	"time"

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
	for src, expected := range exprTests {
		t.Run(src, func(t *testing.T) {
			r := scriggo.MapStringLoader{"main": "package main; func main() { a := " + src + "; _ = a }"}
			program, err := scriggo.LoadProgram(scriggo.Loaders(r, goPackages), scriggo.LimitMemorySize)
			if err != nil {
				t.Errorf("test %q, compiler error: %s", src, err)
				return
			}
			var registers vm.Registers
			tf := func(_ *vm.Function, _ uint32, regs vm.Registers) {
				registers = regs
			}
			err = program.Run(scriggo.RunOptions{MaxMemorySize: 1000000, TraceFunc: tf})
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
	name         string      // name of the test.
	src          string      // source code which must be executed.
	disassembled []string    // expected disassembler output. Can be nil.
	registers    []reg       // list of expected registers. Can be nil.
	output       string      // expected stdout/stderr output.
	err          interface{} // error.
	freeMemory   int         // free memory in bytes, set to zero if there is no limit.
}{
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
		output: "phi: 1.618033988749895\nansic: Mon Jan _2 15:04:05 2006\nnanosecond: 1ns\n",
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
		output: `[1 2 3][][]`,
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
		output: "3e+20\n",
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
		output: `[1 2] [-2 2] [1 2 3] [-2 2 3] `,
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
		output: "[0 0 0 0 0 0 0 0 0 0], len: 10, cap: 10\n[0 0 0 0], len: 4, cap: 6\n[], len: 0, cap: 3\n[0 0], len: 2, cap: 2\n[0 0], len: 2, cap: 5\n[0 0], len: 2, cap: 20\n[], len: 0, cap: 5\n",
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
		output: "s: [0 0 0 0 0]\nlen: 5\ncap: 10\n",
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
		output: "hello true\n",
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
		output: "1.3 1\n",
	},

	{
		name: "Builtin len as function (interface{}) parameter",
		src: `package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print(len("hello"))
			fmt.Print(len([]int{2,3,4}))
		}				
		`,
		output: "53",
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
		output: `s()`,
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
		output: "{errorName error}\n",
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
		output: "{git/prg [arg1 arg2 arg3] [] /home/user/ <nil> <nil> <nil> [] <nil> <nil> <nil> <nil> <nil> false [] [] [] [] <nil> <nil>}\n/home/user/\ngit/prg\n[arg1 arg2 arg3]\n",
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
		output: "{git/prg [] [] /home/user/ <nil> <nil> <nil> [] <nil> <nil> <nil> <nil> <nil> false [] [] [] [] <nil> <nil>}\n/home/user/\ngit/prg\n",
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
		output: "{git/prg [] [] /home/user/ <nil> <nil> <nil> [] <nil> <nil> <nil> <nil> <nil> false [] [] [] [] <nil> <nil>}\n/home/user/\ngit/prg\n",
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
		output: "oldPath,newPath",
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
		output: `Path is "aPath"`,
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
		output: `Path is "aPath"`,
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
		output: "{Path:aPath Args:[] Env:[] Dir: Stdin:<nil> Stdout:<nil> Stderr:<nil> ExtraFiles:[] SysProcAttr:<nil> Process:<nil> ProcessState:<nil> ctx:<nil> lookPathErr:<nil> finished:false childFiles:[] closeAfterStart:[] closeAfterWait:[] goroutine:[] errch:<nil> waitDone:<nil>}",
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
		output: "{Path: Args:[] Env:[] Dir: Stdin:<nil> Stdout:<nil> Stderr:<nil> ExtraFiles:[] SysProcAttr:<nil> Process:<nil> ProcessState:<nil> ctx:<nil> lookPathErr:<nil> finished:false childFiles:[] closeAfterStart:[] closeAfterWait:[] goroutine:[] errch:<nil> waitDone:<nil>}",
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
		output: "slice side-effect, slice side-effect, slice side-effect, array side-effect, map side-effect, ",
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
		output: "abc\n",
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
		output: "s.String() is 20m42s",
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
		output: "s.String() is 20m42s",
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
		output: "s.String() is 20m42s",
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
		output: `2`,
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
		output: "s.String() is 10m0s",
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
		output: "0.002",
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
		output: `0.002`,
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
		output: "2",
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
		output: "2",
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
		output: "content of (indirect) buffer",
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
		output: "content of buffer",
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
		output: "5",
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
		output: "99, c",
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
		output: "60.001 60",
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
		output: "10 10",
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
		output: "1.2345678998765321e+08 1.2345679e+08",
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
		output: `str: hello!, slice: [104 101 108 108 111 33]`,
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
		output: `s: [97 98 99], c: abc`,
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
		output: "i: 97, c: a",
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
		output: "400 -112 -112",
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
		output: "[1 2 3 4 5] 5 5\t[] 0 5\t[] 0 5\t[1 2 3] 3 5\t[1 2 3] 3 5\t[1 2 3 4 5] 5 5\t[3 4 5] 3 3\t[3 4] 2 3\t[] 0 3\t[] 0 0\t[1] 1 2\t[3] 1 3\t[] 0 0\t",
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
		output: "func()",
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
		output: "hello world!\n",
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
		output: "ok",
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
		output: "0 0 1",
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
		output: "10 13 16",
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
		output: "2 4 8",
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
		output: "10 10",
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
		output: "10 10",
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
		output: "0 2 10",
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
		output: "0 2 10",
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
		output: "0 0 0 1",
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
		output: "0 1 2",
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
		output: "1 2 3 4",
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
		output: "/* not a comment */",
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
		output: "3\n",
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
		output: "[10 20 30] [10 20 30 40] [10 20 30 50]",
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
		output: "[1 2 3] [4 5 6] [1 2 3 4 5 6]",
	},

	{"Binary operation on global expression",
		`package main

		import (
			"fmt"
		)
		
		var A = true
		var B = false
		var OR = A || B
		
		func main() {
			fmt.Print(A, B, OR)
		}`, nil, nil, "true false true", nil, 0},

	{"Issue #113",
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
		}`, nil, nil, "map[20:12]\nfalse\n24\n", nil, 0},

	{"Boolean global variables",
		`package main

		import (
			"fmt"
		)
		
		var A = true
		var B = false
		var C = A
		
		func main() {
			fmt.Print(A, B, C)
		}
		`, nil, nil, "true false true", nil, 0},

	{"Double pointer indirection",
		`package main

		import "fmt"
		
		func main() {
			a := "hello"
			b := &a
			c := &b
			fmt.Print(**c)
		}
		`, nil, nil, "hello", nil, 0},

	{"Break statement in type-switch statements",
		`package main

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
		}`, nil, nil, "switch,case string,done", nil, 0},

	{"Break statement in switch statements",
		`package main

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
		`, nil, nil, "switch,case true,done", nil, 0},

	{"Break statement in for statements",
		`package main

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
		`, nil, nil, "abc", nil, 0},

	{"Const declaration inside function body",
		`package main

		import "fmt"
		
		func main() {
			const A = 10
			fmt.Println(A)
		}`, nil, nil, "10\n", nil, 0},

	{"Issue #78",
		`package main

		func main() {
			a := 0
			f := func() { _ = a }
			_ = f
		}
		`, nil, nil, "", nil, 0},

	{"Counter based on gotos/labels",
		`package main

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
		`, nil, nil, ",0,1,2,3,4,5,6,7,8,9,", nil, 0},

	{"Goto label - Forward jump out of scope",
		`package main

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
		`, nil, nil, "out1,goto,out2", nil, 0},

	{"Goto label - Simple forwaring goto",
		`package main

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
		`, nil, nil, "ab", nil, 0},

	{"Goto label - Simple one-step forwarding goto",
		`package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print("a")
			goto L
		L:
			fmt.Print("b")
		}
		`, nil, nil, "ab", nil, 0},

	{"Package variable shadowing in closure body",
		`package main

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
		`, nil, nil, "11201", nil, 0},

	{"Package variable in closure",
		`package main

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
		`, nil, nil, "123344", nil, 0},

	{"Package with both predefined and not predefined variables",
		`package main

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
		`, nil, nil, "{5 42}\nmsg\n", nil, 0},

	{"Function literal call",
		`package main

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
		`, nil, nil, "start,f,end", nil, 0},

	{"Builtin close",
		`package main

		import "fmt"
		
		func main() {
			c := make(chan int)
			fmt.Print("closing: ")
			close(c)
			fmt.Print("ok")
		}
		`, nil, nil, "closing: ok", nil, 0},

	{"Builtin make - with and w/o cap argument",
		`package main

		import (
			"fmt"
		)
		
		func main() {
			s1 := make([]int, 2)
			fmt.Print("s1: ", len(s1), cap(s1))
			s2 := make([]int, 2, 5)
			fmt.Print("/s2: ", len(s2), cap(s2))
		}
		`, nil, nil, "s1: 2 2/s2: 2 5", nil, 0},

	{"Switch without expression",
		`package main

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
		`, nil, nil, "switch: true.", nil, 0},

	{"Package variable - call to a predefined function with package variable as argument",
		`package main

		import "fmt"
		
		var A = 3
		
		func main() {
			fmt.Print(A)
		}
		`, nil, nil, "3", nil, 0},

	{"Package variable - int variable set by a package function",
		`package main

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
		`, nil, nil, "34", nil, 0},

	{"Package variable - int variable with math expression as value",
		`package main

		import "fmt"
		
		var A = F() + 3 + (2 * F())
		
		func F() int {
			return 42
		}
		
		func main() {
			a := A
			fmt.Print(a)
		}
		`, nil, nil, "129", nil, 0},

	{"Package variable - int variable with package function as value",
		`package main

		import "fmt"
		
		var A = F()
		
		func F() int {
			return 42
		}
		
		func main() {
			a := A
			fmt.Print(a)
		}
		`, nil, nil, "42", nil, 0},

	{"Package variable - int variable and string variable",
		`package main

		import "fmt"
		
		var A = 3
		var B = "hey!"
		
		func main() {
			a := A
			fmt.Print(a)
			b := B
			fmt.Print(b)
		
		}
		`, nil, nil, "3hey!", nil, 0},

	{"Package variable - one int variable",
		`package main

		import "fmt"
		
		var A = 3
		
		func main() {
		
			a := A
			fmt.Print(a)
		
		}
		`, nil, nil, "3", nil, 0},

	{"Break - For range (no label)",
		`package main

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
		`, nil, nil, "start, v: 1, no break, v: 2, break, end", nil, 0},

	{"Continue - For range (no label)",
		`package main

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
		`, nil, nil, "1:cont?no!2:cont?yes!3:cont?no!", nil, 0},

	{"Init function with package variables",
		`package main

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
		`, nil, nil, "F!init1!main!", nil, 0},

	{"Init function - one",
		`package main

		import (
			"fmt"
		)
		
		func init() {
			fmt.Print("init!")
		}
		
		func main() {
			fmt.Print("main!")
		}
		`, nil, nil, "init!main!", nil, 0},

	{"Init function - three",
		`package main

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
		`, nil, nil, "init1!init2!init3!main!", nil, 0},

	{"Package variables initialization",
		`package main

		import "fmt"
		
		var A = F()
		
		func F() int {
			fmt.Print("F has been called!")
			return 200
		}
		
		func main() {
			fmt.Print("main!")
		}
		`, nil, nil, "F has been called!main!", nil, 0},

	{"Closure - complex tests (multiple clojures)",
		`package main

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
		`, nil, nil, "102042", nil, 0},

	{"Closure - one level and two levels variable writing (int)",
		`package main

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
		`, nil, nil, "main:5, main:5, f:5, f:20, g:20, g:30, f:30, main:30", nil, 0},

	{"Closure - one level variable writing (int)",
		`package main

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
		`, nil, nil, "main:5, main:5, f:5, f:20, main:20", nil, 0},

	{"Closure - one level and two levels variable reading (int)",
		`package main

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
		}`, nil, nil, "f: 10, g: 10", nil, 0},

	{"Closure - one level variable reading (int and string)",
		`package main

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
		`, nil, nil, "f: 10,f: hey", nil, 0},

	{"Closure - one level variable reading (int)",
		`package main

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
		`, nil, nil, "f: 10", nil, 0},

	{"For range on interface{} slice",
		`package main

		import "fmt"
		
		func main() {
		
			mess := []interface{}{3, "hey", 5.6, 2}
			for e, v := range mess {
				fmt.Print(e, v, ",")
			}
		
		}
		`, nil, nil, "0 3,1hey,2 5.6,3 2,", nil, 0},

	{"For range on empty int slice",
		`package main

		import "fmt"
		
		func main() {
		
			fmt.Printf("start-")
			for _ = range []int{} {
				fmt.Print("looping!")
			}
			fmt.Printf("-end")
		
		}
		`, nil, nil, "start--end", nil, 0},

	{"For range on int slice, taking only values",
		`package main

		import "fmt"
		
		func main() {
		
			for _, v := range []int{3, 4, 5} {
				fmt.Print(v)
			}
		
		}
		`, nil, nil, "345", nil, 0},

	{"For range on int slice, taking only indexes",
		`package main

		import "fmt"
		
		func main() {
		
			for i := range []int{3, 4, 5} {
				fmt.Print(i)
			}
		
		}
		`, nil, nil, "012", nil, 0},

	{"For range with simple assignment (no declaration)",
		`package main

		import "fmt"
		
		func main() {
		
			i := 0
			e := 0
			for i, e = range []int{3, 4, 5} {
				fmt.Print(i, e, ",")
			}
		}
		`, nil, nil, "0 3,1 4,2 5,", nil, 0},

	{"For range with no assignments",
		`package main

		import "fmt"
		
		func main() {
		
			for _ = range []int{1, 2, 3} {
				fmt.Print("?")
			}
		}
		`, nil, nil, "???", nil, 0},

	{"For range on map[string]int",
		`package main

		import "fmt"
		
		func main() {
		
			m := map[string]int{"twelve": 12}
			for k, v := range m {
				fmt.Print(k, " is ", v, ",")
			}
		
		}
		`, nil, nil, "twelve is 12,", nil, 0},

	{"For range on int slice",
		`package main

		import "fmt"
		
		func main() {
		
			for i, v := range []int{4, 5, 6} {
				fmt.Print(i, v, ",")
			}
		
		}
		`, nil, nil, "0 4,1 5,2 6,", nil, 0},

	{"For range on string",
		`package main

		import "fmt"
		
		func main() {
		
			for i, c := range "Hello" {
				fmt.Print(i, c, ",")
			}
		
		}
		`, nil, nil, "0 72,1 101,2 108,3 108,4 111,", nil, 0},

	{"Issue #92",
		`package main

		import "fmt"
		
		func main() {
		
			fmt.Println("a", 10)
			fmt.Println("b", 2.5)
		
		}`, nil, nil, "a 10\nb 2.5\n", nil, 0},

	{"Builtin len on maps",
		`package main

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
		`, nil, nil, "0114", nil, 0},

	{"Builtin function calls can be used directly as function arguments",
		`package main

		import (
			"fmt"
		)
		
		func main() {
			fmt.Print(len([]int{1,2,3}))
			fmt.Print(cap(make([]int, 1, 10)))
		}
		`, nil, nil, "310", nil, 0},

	{"Checking if map has keys",
		`package main

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
		`, nil, nil, "1 0 0,true false true", nil, 0},

	{"Map index assignment",
		`package main

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
		`, nil, nil, "map[],map[four:4 one:1],map[four:4 one:10]", nil, 0},

	{"Builtin delete",
		`package main

		import (
			"fmt"
		)
		
		func main() {
			m := map[string]string{"key": "value", "key2": "value2"}
			fmt.Print(m, ",")
			delete(m, "key")
			fmt.Print(m)
		}
		`, nil, nil, "map[key:value key2:value2],map[key2:value2]", nil, 0},

	{"Multiple assignment with blank identifier",
		`package main

		import "fmt"
		
		func main() {
		
			_, a := 1, 2
			fmt.Println(a)
		
		}
		`, nil, nil, "2\n", nil, 0},

	{"Builtin make - Map with no size",
		`package main

		import "fmt"
		
		func main() {
		
			m := make(map[string]int)
			fmt.Printf("%v (%T)", m, m)
		}
		`, nil, nil, "map[] (map[string]int)", nil, 0},

	{"Builtin new - creating a string pointer and updating it's value",
		`package main

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
		}`, nil, nil, "*pi:,*pi:newv", nil, 0},

	{"Builtin new - creating two int pointers and comparing their values",
		`package main

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
		`, nil, nil, "pi: *int,equal?true", nil, 0},

	{"Builtin new - creating an int pointer",
		`package main

		import (
			"fmt"
		)
		
		func main() {
			pi := new(int)
			fmt.Printf("pi: %T", pi)
		}
		`, nil, nil, "pi: *int", nil, 0},

	{"Pointer indirection as expression",
		`package main

		import (
			"fmt"
		)
		
		func main() {
			a := "a"
			b := &a
			c := *b
			fmt.Print(c)
		}
		`, nil, nil, "a", nil, 0},

	{"Pointer indirection assignment",
		`package main

		import "fmt"
		
		func main() {
		
			a := "a"
			b := "b"
			c := "c"
		
			pb := &b
			*pb = "newb"
		
			fmt.Print(a, b, c)
		}`, nil, nil, "anewbc", nil, 0},

	{"Address operator on strings declared with 'var'",
		`package main

		import "fmt"
		
		func main() {
		
			var s1 = "hey"
			var s2 = "hoy" // TODO: should be "hey"
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
		`, nil, nil, "true\nfalse\nfalse\n", nil, 0},

	{"Address operator on strings declared with ':='",
		`package main

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
		`, nil, nil, "true\nfalse\nfalse\n", nil, 0},

	{"Switch with non-immediate expression in a case",
		`package main

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
		`, nil, nil, "a * b", nil, 0},

	{"Issue #86",
		`package main

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
		
		}`, nil, nil, "3\ninternal\next\n", nil, 0},

	{"Builtin 'cap'",
		`package main

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
		`, nil, nil, "10,3", nil, 0},

	// {"Package function calling a post-declared package function",
	// 	`package main

	// 	func f() { g() }

	// 	func g() { }

	// 	func main() { }
	// 	`, nil, nil, "", nil, 0},

	{"Function literal assigned to underscore",
		`package main
		
		func main() {
			_ = func() { }
		}`, nil, nil, "", nil, 0},
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
		}`, nil, nil, "hey\n", nil, 0},

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
		`, nil, nil, "2 42 33 11 12", nil, 0},

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
		`, nil, nil, "fg", nil, 0},

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
		nil, nil, "[]int[]uint8[] []int\n[] []uint8\n", nil, 0},

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
		nil, nil, "i()[6 2 3]\n", nil, 0},

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
		`, nil, nil, "100", nil, 0},

	{"Issue #67",
		`package main

		func f() {
			f := "hi!"
			_ = f
		}
		
		func main() {
		
		}`, nil, nil, "", nil, 0},

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
		}`, nil, nil, "flag is true\nflag is false\n", nil, 0},

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
		nil, nil, "4", nil, 0},

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
		`, nil, nil, "10 20\n30 40\n70 80 str2\n70 80 str2\n90 0 100\n", nil, 0},

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
		`, nil, nil, "10\n", nil, 0},

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
		`, nil, nil, "15 1\n", nil, 0},

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
		`, nil, nil, "14 4\n", nil, 0},

	{"Channels - Reading and writing",
		`package main

		import "fmt"
		
		func main() {
			ch := make(chan int, 4)
			ch <- 5
			v := <-ch
			fmt.Print(v)
		}
		`, nil, nil, "5", nil, 0},

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
		"10 20\n10 20\n20 10\n10 20\n", nil, 0},

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
		"75,-9,1386,0,33,1797,1377,1335,15,", nil, 0},

	{"Selector (predefined struct)",
		`package main

		import (
			"fmt"
			"testpkg"
		)
		
		func main() {
			center := testpkg.Center
			fmt.Println(center)
		}
		`, nil, nil, "{5 42}\n", nil, 0},
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
		nil, nil, "fg", nil, 0},

	{"Recover",
		`package main
		
		func main() {
			recover()
			v := recover()
			_ = v
		}
		`, nil, nil, "", nil, 0},

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
		"g\nf\n", nil, 0},

	{"Dot import (predefined)",
		`package main
		
		import . "fmt"
		
		func main() {
			Println("hey")
		}`,
		nil, nil, "hey\n", nil, 0},

	{"Import (predefined) with explicit package name",
		`package main
		
		import f "fmt"
		
		func main() {
			f.Println("hey")
		}`,
		nil, nil, "hey\n", nil, 0},

	{"Import (predefined) with explicit package name (two packages)",
		`package main
		
		import f "fmt"
		import f2 "fmt"
		
		func main() {
			f.Println("hey")
			f2.Println("oh!")
		}`,
		nil, nil, "hey\noh!\n", nil, 0},

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
		"10 42 88\n", nil, 0},

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
		"ffffff", nil, 0},

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
		"abcde", nil, 0},

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
		`, nil, nil, "10hi", nil, 0},

	{"Multiple assignment",
		`package main

		import "fmt"
		
		func main() {
			a, b := 6, 7
			fmt.Print(a, b)
		}
		`,
		nil, nil, "6 7", nil, 0},

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
		}, nil, "", nil, 0},

	{"Assignment with addition value (non constant)",
		`package main

		import "fmt"
		
		func main() {
			a := 5
			b := 10
			c := a + 2 + b + 30
			fmt.Print(c)
		}`,
		nil, nil, "47", nil, 0},

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
		nil, "", nil, 0},

	{"Function call assignment (2 to 1) - Predefined function with two return values",
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
		`, nil, nil, "5 42 33\n", nil, 0},
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
		[]reg{}, "", nil, 0},
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
			// {vmp.TypeInt, 1, int64(45)}, // a
			// {vmp.TypeInt, 2, int64(10)}, // b
			// {vmp.TypeInt, 3, int64(50)}, // c
			// {vmp.TypeInt, 4, int64(45)}, // d
		}, "", nil, 0},

	{"Other forms of assignment",
		`
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
		nil, nil, "27,162,54,6,2,23,184,11,-,5,8,6,36,12,4,7,6,24,12,", nil, 0},

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
		"[2 2 3]\n[a b d d]\n", nil, 0},

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
		`, nil, nil, "[1 2 3] [a b c d]\n3 4", nil, 0},

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
			// {TypeGeneral, 1, []int{}}, // a
		}, "[]\n", nil, 0},

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
			// {vmp.TypeGeneral, 1, []string{}}, // a
		}, "", nil, 0},

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
		nil, nil, "[] []", nil, 0},

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
			// {vmp.TypeGeneral, 1, map[string]int{}},
		}, "", nil, 0},

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
		}, nil, "", nil, 0},

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
		}, nil, "", nil, 0},

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
		}, nil, "", nil, 0},

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
		}, "", nil, 0},

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
			// {vmp.TypeGeneral, 1, int64(97)},
		}, "", nil, 0},

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
		}, nil, "", nil, 0},

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
		"truefalsetruefalsefalsetrue", nil, 0},

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
		"", nil, 0},

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
		`, nil, nil, "", nil, 0},

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
		"", nil, 0},

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
			// {vmp.TypeInt, 1, int64(45)},  // a
			// {vmp.TypeInt, 2, int64(-45)}, // b
		},
		"-45", nil, 0},

	{"Go functions as expressions",
		`
		package main

		import "fmt"

		func main() {
			f := fmt.Println
			print(f)
		}
		`, nil, nil, "", nil, 0},

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
		"s\neeff", nil, 0},

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
		nil, nil, "hello world", nil, 0},

	{"Indexing",
		`package main

		import "fmt"

		func main() {
			s := []int{1, 42, 3}
			second := s[1]
			fmt.Println(second)
		}`, nil, nil, "42\n", nil, 0},
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
		`, nil, nil, "thenc=1", nil, 0},

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
		`, nil, nil, "else20", nil, 0},

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
		`, nil, nil, "thenx is 11", nil, 0},

	{"For statement",
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
		`, nil, nil, "i=0,i=1,i=2,i=3,i=4,i=5,i=6,i=7,i=8,i=9,sum=20", nil, 0},

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
		`, nil, nil, "case 1a=20", nil, 0},

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
		`, nil, nil, "case 2case 330", nil, 0},

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
		`, nil, nil, "default80", nil, 0},

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
		`, nil, nil, "defaultcase 2case 330", nil, 0},

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
		`, nil, nil, "20", nil, 0},

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
		}, nil, "", nil, 0},

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
		`, nil, nil, "25", nil, 0},

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
		`, nil, nil, "", nil, 0},

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
		nil, nil, "a", nil, 0},

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
		`, nil, nil, "21", nil, 0},

	{"Package function with one return value",
		`package main

		import "fmt"

		func five() int {
			return 5
		}

		func main() {
			a := five()
			fmt.Print(a)
		}`, nil, nil, "5", nil, 0},

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
		nil, nil, "5.6 -432.12\n", nil, 0},
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
		`, nil, nil, "12", nil, 0},

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
		"a, b, c:  3 4 5\n[3 2 5] has len 3\n", nil, 0},

	{"Builtin len (with a constant argument)",
		`
		package main

		import "fmt"

		func main() {
			a := len("abc");
			fmt.Println(a)
		}
		`, nil, nil, "3\n", nil, 0},

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
		nil, nil, "534", nil, 0},

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
		}, nil, "", nil, 0},

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
			// {vmp.TypeGeneral, 3, map[string]int{}},
		}, "", nil, 0},

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
		`, nil, nil, "dst: [10 20 30] n: 3\ndst2: [10 20]\n", nil, 0},
	{"Function which calls both predefined and not predefined functions",
		`package main

		import "fmt"
		
		func scriggoFunc() {
			fmt.Println("scriggoFunc()")
		}

		func main() {
			scriggoFunc()
			fmt.Println("main()")
		}`,
		nil,
		nil,
		"scriggoFunc()\nmain()\n", nil, 0},
	{"Predefined function call (0 in, 0 out)",
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
		nil, "", nil, 0},

	{"Predefined function call (0 in, 1 out)",
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
			// {vmp.TypeInt, 1, int64(40)},
		}, "", nil, 0},

	{"Predefined function call (1 in, 0 out)",
		`
		package main

		import "testpkg"

		func main() {
			testpkg.F10(50)
			return
		}`,
		nil, nil, "", nil, 0},

	{"Predefined function call (1 in, 1 out)",
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
		`, nil, nil, "42", nil, 0},

	{"Predefined function call (2 in, 1 out) (with surrounding variables)",
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
			// {vmp.TypeInt, 1, int64(3)},  // a
			// {vmp.TypeInt, 2, int64(13)}, // b
			// {vmp.TypeInt, 3, int64(4)},  // e
			// {vmp.TypeInt, 4, int64(16)}, // c
			// {vmp.TypeInt, 5, int64(16)}, // d // TODO (Gianluca): d should be allocated in register 5, which is no longer used by function call.
		}, "", nil, 0},

	{"Predefined function call of StringLen",
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
			// {vmp.TypeInt, 1, int64(3)},
		}, "", nil, 0},

	{"Predefined function call of fmt.Println",
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
		`, nil, nil, "hello, world!\n42\nhi!\n1 2 3\nhi! hi! [3 4 5]\n", nil, 0},

	{"Predefined function call f(g())",
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
		"a is 42 and b is 33\n", nil, 0},

	{"Reading a predefined int variable",
		`
		package main

		import "testpkg"

		func main() {
			a := testpkg.A
			testpkg.PrintInt(a)
		}
		`, nil, nil,
		"20", nil, 0},

	{"Writing a predefined int variable",
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
		"42->7", nil, 0},

	{"Pointer declaration (nil int pointer)",
		`package main

		import (
			"fmt"
		)
		
		func main() {
			var a *int
			fmt.Print(a)
		}
		`, nil, nil, "<nil>", nil, 0},

	{"Scriggo function 'repeat(string, int) string'",
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
		nil, nil, "", nil, 0},

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
			// {vmp.TypeGeneral, 3, [][]int{[]int{10, 20}, []int{25, 26}, []int{30, 40, 50}}},
		},
		"", nil, 0},

	{"Multiple predefined function calls",
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
			// {vmp.TypeInt, 1, int64(6)}, // a
			// {vmp.TypeInt, 2, int64(5)}, // b
			// {vmp.TypeInt, 3, int64(3)}, // c
		}, "", nil, 0},

	{"Predefined function 'Swap'",
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
		`, nil, nil, "heyhey3 3", nil, 0},

	{"Many Scriggo functions (swap, sum, fact)",
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
		"a:6,b:2,c:2,d:6,e:16,f:40330,", nil, 0},
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
		"a is 9 and b is 144\n", nil, 0},

	//------------------------------------
	// TODO(Gianluca): disabled tests:

	// {
	// 	name: "x.f convertion to (*x).f",
	// 	src: `package main

	// 	import (
	// 		"fmt"
	// 		"os/exec"
	// 	)

	// 	func main() {
	// 		e := &exec.Error{Name: "errorName"}
	// 		fmt.Println(e.Name)
	// 	}`,
	// 	output: "errorName\n",
	// },

	// {
	// 	name: "Method value (assignment and call)",
	// 	src: `package main

	// 	import (
	// 		"bytes"
	// 		"fmt"
	// 	)

	// 	func main() {
	// 		b := bytes.NewBuffer([]byte{97, 98, 99})
	// 		lenMethod := b.Len
	// 		l := lenMethod()
	// 		fmt.Print("l is ", l)
	// 	}
	// 	`,
	// 	output: "l is 3",
	// },

	// {
	// 	name: "Method (with non-pointer receiver) called on a pointer receiver",
	// 	src: `package main

	// 	import "testpkg"

	// 	func main() {
	// 		t := testpkg.NewT(-20)
	// 		tp := &t
	// 		testpkg.T.M(tp)
	// 	}
	// 	`,
	// 	output: `t is -20`,
	// },

	// {
	// 	name: "Calling a method defined on pointer on a non-pointer value",
	// 	src: `package main

	// 	import (
	// 		"bytes"
	// 		"fmt"
	// 	)

	// 	func main() {
	// 		b := *bytes.NewBuffer([]byte{97, 98, 99})
	// 		fmt.Printf("b has type %T", b)
	// 		l := b.Len()
	// 		fmt.Print(", l is ", l)
	// 	}
	// 	`,
	// 	output: `b has type bytes.Buffer, l is 3`,
	// }

	//{"Out of memory: OpAppend ",
	//	`package main
	//
	//	import "fmt"
	//
	//	func main() {
	//		fmt.Println("12345678901234567890" + "12345678901234567890")
	//	}`,
	//	nil,
	//	nil,
	//	"a is 9 and b is 144\n", nil, 0},

	// {
	// 	name: "Iota - local constant in math expression (with float numbers)",
	// 	src: `package main

	// 	import (
	// 		"fmt"
	// 	)

	// 	func main() {
	// 		const (
	// 			A = (iota * iota) + 10
	// 			B = A * (iota * 2.341)
	// 			C = A + B + iota / 2.312
	// 		)
	// 		fmt.Print(A, B, C)
	// 	}
	// 	`,
	// 	output: "10 23.41 34.27505190311419",
	// },

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
	// {"(Predefined) struct composite literal (empty)",
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
	for _, cas := range stmtTests {
		t.Run(cas.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r != nil {
					panic(fmt.Errorf("%s panicked: %s", cas.name, r))
				}
			}()
			regs := cas.registers
			r := scriggo.MapStringLoader{"main": cas.src}
			program, err := scriggo.LoadProgram(scriggo.Loaders(r, goPackages), scriggo.LimitMemorySize)
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
			var registers vm.Registers
			tf := func(_ *vm.Function, _ uint32, regs vm.Registers) {
				registers = regs
			}
			err = program.Run(scriggo.RunOptions{MaxMemorySize: 1000000, TraceFunc: tf})
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

			// Tests if disassembler output matches.

			// TODO (Gianluca): to review.
			if false && cas.disassembled != nil {
				asm := strings.Builder{}
				_, err := program.Disassemble(&asm, "main")
				if err != nil {
					t.Errorf("test %q, disassemble error: %s", cas.name, err)
					return
				}
				got := asm.String()
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

			for _, reg := range regs {
				var got interface{}
				switch reg.typ {
				case vm.TypeFloat:
					got = registers.Float[reg.r-1]
				case vm.TypeGeneral:
					got = registers.General[reg.r-1]
				case vm.TypeInt:
					got = registers.Int[reg.r-1]
				case vm.TypeString:
					got = registers.String[reg.r-1]
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

var goPackages = scriggo.Packages{
	"fmt": {
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
	"testpkg": {
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
			"NewT":         NewT,
			"T":            reflect.TypeOf(T(0)),
		},
	},
	"math": {
		Name: "math",
		Declarations: map[string]interface{}{
			"Phi": scriggo.ConstLiteral(nil, "1.61803398874989484820458683436563811772030917980576286213544862"),
		},
	},
	"bytes": {
		Name: "bytes",
		Declarations: map[string]interface{}{
			"NewBuffer":       bytes.NewBuffer,
			"NewBufferString": bytes.NewBufferString,
			"Buffer":          reflect.TypeOf(new(bytes.Buffer)).Elem(),
		},
	},
	"time": {
		Name: "time",
		Declarations: map[string]interface{}{
			"Duration":      reflect.TypeOf(new(time.Duration)).Elem(),
			"ParseDuration": time.ParseDuration,
			"ANSIC":         scriggo.ConstLiteral(nil, "\"Mon Jan _2 15:04:05 2006\""),
			"Nanosecond":    scriggo.ConstLiteral(reflect.TypeOf(new(time.Duration)).Elem(), "1"),
		},
	},
	"os/exec": {
		Name: "exec",
		Declarations: map[string]interface{}{
			"Cmd":   reflect.TypeOf(new(exec.Cmd)).Elem(),
			"Error": reflect.TypeOf(new(exec.Error)).Elem(),
		},
	},
	"errors": {
		Name: "errors",
		Declarations: map[string]interface{}{
			"New": errors.New,
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

type T int

func NewT(a int) T {
	return T(a)
}
func (t T) M() {
	fmt.Print("t is ", t)
}
