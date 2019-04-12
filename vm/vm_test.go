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

var stmtTests = []struct {
	name         string
	src          string
	disassembled []string
	registers    []reg
}{

	{"Simple assignment",
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
			`	MoveInt 10 R0`,
			`     MoveString "hi" R1`,
			`     MoveString R1 R0`,
			`	Return`,
		},
		[]reg{
			{TypeInt, 0, int64(10)}, // a
			{TypeString, 0, "hi"},   // c
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
			`	MoveInt 9 R0`,
			`	Return`,
		},
		nil},
	{"Assignment with constant int value (addition)",
		`
			package main

			func main() {
				a := 10 + 20 - 4
				_ = a
			}
		`,
		nil,
		[]reg{
			{TypeInt, 0, int64(26)}, // a
		}},
	{"Assignment with addition value (non constant)",
		`
			package main

			func main() {
				a := 5
				b := 10
				c := a + b
				_ = c
			}
		`,
		nil,
		[]reg{
			{TypeInt, 0, int64(5)},  // a
			{TypeInt, 1, int64(10)}, // b
			{TypeInt, 2, int64(15)}, // c
		}},
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
			{TypeInt, 0, int64(10)}, // a
			{TypeInt, 1, int64(1)},  // c
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
			{TypeInt, 0, int64(10)}, // a
			{TypeInt, 1, int64(2)},  // c
		}},
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
			{TypeInt, 0, int64(1)}, // c
		}},
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
			{TypeString, 0, "s"},    // a
			{TypeString, 2, "eeff"}, // b
		}},
	{"Empty int slice",
		`
			package main

			func main() {
				a := []int{};
				_ = a
			}
		`,
		nil,
		[]reg{
			{TypeIface, 0, []int{}}, // a
		}},
	{"Empty string slice",
		`
			package main

			func main() {
				a := []string{};
				_ = a
			}
		`,
		nil,
		[]reg{
			{TypeIface, 0, []string{}}, // a
		}},
	{"Empty byte slice",
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
			{TypeIface, 0, []int{}},  // a
			{TypeIface, 1, []byte{}}, // b
		}},
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
			{TypeInt, 0, int64(3)}, // a
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
			{TypeString, 0, "a string"}, // a
			{TypeInt, 0, int64(8)},      // b
		}},
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
			{TypeInt, 0, int64(20)},
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
			{TypeInt, 0, int64(30)},
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
			{TypeInt, 0, int64(80)},
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
			{TypeInt, 0, int64(30)},
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
		}
		`,
		[]string{
			`Package main`,
			``,
			`Func five()`,
			`	// regs(1,0,0,0)`,
			`	MoveInt 5 R0`,
			`	Return`,
			``,
			`Func main()`,
			`	// regs(1,0,0,1)`,
			`	Call main.five	// Stack shift: 1, 0, 0, 0`,
		},
		nil},
}

func TestVM(t *testing.T) {
	DebugTraceExecution = false
	for _, cas := range stmtTests {
		t.Run(cas.name, func(t *testing.T) {
			src := cas.src
			registers := cas.registers
			r := parser.MapReader{"/test.go": []byte(cas.src)}
			comp := NewCompiler(r, nil)
			pkg, err := comp.Compile("/test.go")
			if err != nil {
				t.Errorf("source: %q, compiler error: %s", src, err)
				return
			}
			vm := New(pkg)
			_, err = vm.Run("main")
			if err != nil {
				t.Errorf("source: %q, execution error: %s", src, err)
				return
			}

			// Tests if disassembler output matches.

			if cas.disassembled != nil {
				got := &bytes.Buffer{}
				_, err = Disassemble(got, pkg)
				if err != nil {
					t.Errorf("source: %q, disassemble error: %s", cas.src, err)
					return
				}
				gotLines := []string{}
				for _, line := range strings.Split(strings.TrimSpace(got.String()), "\n") {
					gotLines = append(gotLines, line)
				}
				if diff := equal(cas.disassembled, gotLines); diff >= 0 {
					if !testing.Verbose() {
						t.Errorf("disassembler output doesn't match for source %q (run tests in verbose mode for further details)", cas.src)
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
						t.Errorf("error on source %q:\n%s", cas.src, out.String())
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
					t.Errorf("source %q, register %s[%d]: expecting %#v (type %T), got %#v (type %T)", src, reg.typ, reg.r, reg.value, reg.value, got, got)
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
		comp := NewCompiler(r, nil)
		pkg, err := comp.Compile("/test.go")
		if err != nil {
			panic(fmt.Errorf("unexpected error: source: %q, compiler error: %s", cas.src, err))
		}
		got := &bytes.Buffer{}
		_, err = Disassemble(got, pkg)
		if err != nil {
			panic(fmt.Errorf("unexpected error: source: %q, disassemble error: %s", cas.src, err))
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
