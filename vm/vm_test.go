// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"text/tabwriter"

	"scrigo/parser"
)

// TODO (Gianluca): currently unable to test:
//
//   a | b         --> parsing error
//   a := 1234
//   a := new(int)
//   a := []int{1,2,3,4}
//   f(3)          --> arguments must be defined in function scope
//   func f(int) {

var stmtTests = []struct {
	name         string
	src          string
	disassembled []string
	registers    []reg
}{

	// Assignment.

	{"Single assignment",
		`
		package main

		func main() {
			a := 10
			_ = a
			c := "hi"
			_ = c
			return
		}
		`,
		[]string{
			`Package main`,
			``,
			`Func main()`,
			`	// regs(1,0,2,0)`,
			`	MoveInt 10 R1`,
			`     MoveString "hi" R2`,
			`     MoveString R2 R1`,
			`	Return`,
		},
		[]reg{
			{TypeInt, 1, int64(10)}, // a
			{TypeString, 1, "hi"},   // c
		}},

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
		}},

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
		},
		nil},

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
		}},

	{"Function call assignment (2 to 1) - Native function with two return values",
		`
		package main

		import "testpkg"

		func main() {
			a := 5
			b, c := testpkg.Pair()
			_ = a
			_ = b
			_ = c
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(5)},
			{TypeInt, 2, int64(42)},
			{TypeInt, 3, int64(33)},
		}},

	// Expressions - composite literals.

	{"Empty int slice composite literal",
		`
		package main

		func main() {
			a := []int{};
			_ = a
		}
		`,
		nil,
		[]reg{
			{TypeIface, 1, []int{}}, // a
		}},

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
		}},

	{"Empty byte slice composite literal",
		`
		package main

		func main() {
			a := []int{};
			b := []byte{};
			_ = a; _ = b
		}
		`,
		nil,
		[]reg{
			{TypeIface, 1, []int{}},  // a
			{TypeIface, 2, []byte{}}, // b
		}},

	{"Empty map composite literal",
		`
		package main

		func main() {
			m := map[string]int{}
			_ = m
		}
		`,
		[]string{
			`Package main`,
			``,
			`Func main()`,
			`		// regs(0,0,0,1)`,
			`		MakeMap map[string]int 0 R1`,
		},
		[]reg{
			{TypeIface, 1, map[string]int{}},
		}},

	// Expressions - conversions.

	{"Converting from int to string",
		`
		package main

		func main() {
			a := 97
			b := string(a)
			_ = b
			return
		}
		`,
		nil,
		[]reg{
			{TypeIface, 1, "a"},
		},
	},

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
		},
	},

	// Expressions - misc.

	{"Go functions as expressions",
		`
		package main

		import "fmt"

		func main() {
			f := fmt.Println
			print(f)
		}
		`,
		[]string{
			`Package main`,
			``,
			`Import "fmt"`,
			``,
			`Func main()`,
			`		// regs(1,0,0,1)`,
			`		GetFunc fmt.Println R1`,
			`		Move R1 R1`,
			`		Print R1`,
		},
		nil,
	},

	{"String concatenation (constant)",
		`
		package main

		func main() {
			a := "s";
			_ = a;
			b := "ee" + "ff";
			_ = b
		}
		`,
		nil,
		[]reg{
			{TypeString, 1, "s"},    // a
			{TypeString, 3, "eeff"}, // b
		}},

	{"String concatenation (non constant)",
		`
		package main

		func main() {
			s1 := "hello"
			s2 := "world"
			s3 := s1 + " " + s2
			_ = s3
		}
		`,
		nil,
		[]reg{
			{TypeString, 1, "hello"},       // s1
			{TypeString, 2, "hello"},       // "hello" constant
			{TypeString, 3, "world"},       // s2
			{TypeString, 4, "world"},       // "worlds" constant
			{TypeString, 6, " "},           // " " constant
			{TypeString, 5, "hello world"}, // should be R3?
		},
	},

	// Type assertion.

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
		[]reg{
			{TypeIface, 1, int64(10)}, // a
			{TypeInt, 1, int64(10)},   // interface{}(10) argument
			{TypeIface, 2, int64(10)}, // n
			{TypeInt, 3, int64(1)},    // ok (true)
		},
	},

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
		}},

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
		}},

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
		}},

	// Statements - For.

	{"For statement",
		`
		package main

		func main() {
			  sum := 0
			  for i := 0; i < 10; i++ {
				    sum = sum + 2
			  }
		}
		`,
		[]string{
			`Package main`,
			``,
			`Func main()`,
			`		// regs(2,0,0,0)`,
			`		MoveInt 0 R1`,
			`		MoveInt 0 R2`,
			`1:      IfInt R2 Less 10`,
			`		Goto 2`,
			`		AddInt R2 1 R2`,
			`		MoveInt R1 R1`,
			`		AddInt R1 2 R1`,
			`		Goto 1`,
		},
		[]reg{
			{TypeInt, 1, int64(20)}, // sum
			{TypeInt, 2, int64(10)}, // i
		},
	},

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
		}},

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
		}},

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
		}},

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
		}},

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
		nil,
		nil},

	{"Package function 'inc'",
		`
		package main

		func inc(n int) int {
			return n+1
		}
		
		func main() {
			a := 2
			res := inc(8)
			b := 10
			c := a + b + res
			_ = c
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(2)},  // a
			{TypeInt, 2, int64(9)},  // res
			{TypeInt, 3, int64(9)},  // inc(8) return value
			{TypeInt, 4, int64(8)},  // inc(8) argument
			{TypeInt, 5, int64(10)}, // b
			{TypeInt, 6, int64(21)}, // c
		}},

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
		}},

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
		}},

	{"Package function with many return values (different types)",
		`
		package main

		func pair() (int, float64) {
			return 42, 33.0
		}

		func main() {
			a := 2
			b, c := pair()
			d, e := 11, 12
			_ = a + b + d + e
			_ = c
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(2)},        // a
			{TypeInt, 2, int64(42)},       // b
			{TypeFloat, 1, float64(33.0)}, // c
			{TypeInt, 4, int64(11)},       // d
			{TypeInt, 5, int64(12)},       // e
		}},

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
		}},

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
		}},

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
		}},

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
			`// regs(4,0,2,0)`,
			`	MoveInt 42 R1`,
			`	Print R1`,
			`	MoveString "hello" R1`,
			`	MoveString R1 R2`,
			`	Print R2`,
			`	MoveInt 10 R3`,
			`	MoveInt R3 R4`,
			`	Print R4`,
		},
		nil},

	{"Builtin make - map",
		`
		package main

		func main() {
			m := make(map[string]int, 2)
			_ = m
		}
		`,
		[]string{
			`Package main`,
			``,
			`Func main()`,
			`	// regs(0,0,0,1)`,
			`	MakeMap map[string]int 2 R1`,
		},
		[]reg{
			{TypeIface, 1, map[string]int{}},
		}},

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
		nil},

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
		}},

	{"Native function call (1 in, 0 out)",
		`
		package main

		import "testpkg"

		func main() {
			testpkg.F10(50)
			return
		}
		`,
		nil,
		nil},

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
		}},

	{"Native function call (2 in, 1 out) (with surrounding variables)",
		`
		package main

		import "testpkg"
		
		func main() {
			a := 2 + 1  // first arg.
			b := 3 + 10 // second arg.
			e := 4      // unused, just takes space.
			_ = e
			c := testpkg.Sum(a, b)
			d := c // return value assigned to variable.
			_ = d
			return
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(3)},  // a
			{TypeInt, 2, int64(13)}, // b
			{TypeInt, 3, int64(4)},  // e
			{TypeInt, 4, int64(16)}, // c
			{TypeInt, 8, int64(16)}, // d // TODO (Gianluca): d should be allocated in register 5, which is no longer used by function call.
		}},

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
		}},

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
			b := 1
			d := b + 2
			s1 := repeat("z", 4)
			f := "hello"
			s2 := repeat(f, d)
			_ = s1 + s2
		}
		`,
		nil,
		[]reg{
			{TypeInt, 1, int64(1)},             // b
			{TypeInt, 2, int64(3)},             // d
			{TypeString, 1, "zzzz"},            // s1
			{TypeString, 2, "zzzz"},            // repeat#1 return
			{TypeString, 3, "z"},               // repeat#1 arg#1
			{TypeInt, 3, int64(4)},             // repeat#1 arg#2
			{TypeString, 8, "hellohellohello"}, // s2
		},
	},
}

func TestVM(t *testing.T) {
	DebugTraceExecution = false
	for _, cas := range stmtTests {
		t.Run(cas.name, func(t *testing.T) {
			registers := cas.registers
			r := parser.MapReader{"/test.go": []byte(cas.src)}
			comp := NewCompiler(r, goPackages)
			pkg, err := comp.Compile("/test.go")
			if err != nil {
				t.Errorf("test %q, compiler error: %s", cas.name, err)
				return
			}
			vm := New(pkg)
			_, err = vm.Run("main")
			if err != nil {
				t.Errorf("test %q, execution error: %s", cas.name, err)
				return
			}

			// Tests if disassembler output matches.

			if cas.disassembled != nil {
				got := &bytes.Buffer{}
				_, err = Disassemble(got, pkg)
				if err != nil {
					t.Errorf("test %q, disassemble error: %s", cas.name, err)
					return
				}
				gotLines := []string{}
				for _, line := range strings.Split(strings.TrimSpace(got.String()), "\n") {
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

// reg represents a register and it's used in tests only.
type reg struct {
	typ   Type
	r     int8
	value interface{}
}

// ptrTo returns a pointer to a copy of v.
func ptrTo(v interface{}) interface{} {
	rv := reflect.New(reflect.TypeOf(v))
	rv.Elem().Set(reflect.ValueOf(v))
	return rv.Interface()
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
func NoTestMakeExpressionTests(t *testing.T) {
	out := strings.Builder{}
	out.WriteString("\n")
	for _, cas := range stmtTests {
		r := parser.MapReader{"/test.go": []byte(cas.src)}
		comp := NewCompiler(r, goPackages)
		pkg, err := comp.Compile("/test.go")
		if err != nil {
			panic(fmt.Errorf("unexpected error: source: %q, compiler error: %s", oneLine(cas.src), err))
		}
		got := &bytes.Buffer{}
		_, err = Disassemble(got, pkg)
		if err != nil {
			panic(fmt.Errorf("unexpected error: source: %q, disassemble error: %s", oneLine(cas.src), err))
		}

		out.WriteString("{\n")
		out.WriteString("\t\"" + cas.name + "\",\n")
		out.WriteString("\t`" + cas.src + "`,\n")
		out.WriteString("\t[]string{\n")
		for _, line := range strings.Split(strings.TrimSpace(got.String()), "\n") {
			out.WriteString("\t\t`" + line + "`,\n")
		}
		out.WriteString("\t},\n")
		out.WriteString("\t[]reg{\n")
		out.WriteString("\t\t// Registers overwritten (use your diff tool to restore original ones)\n")
		out.WriteString("\t},\n")
		out.WriteString("},\n")
	}
	t.Error(out.String())
}

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
		},
	},
}
