package vm

import (
	"bytes"
	"fmt"
	"scrigo/parser"
	"strings"
	"testing"
	"text/tabwriter"
)

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

func removeTabs(s string) string {
	return strings.ReplaceAll(s, "\t", "    ")
}

var expr_tests = []struct {
	src      string
	expected []string
}{

	{
		`a := 10; _ = a`,
		[]string{
			"Package main",
			"",
			"Func main()",
			"	// regs(1,0,0,0)",
			"	MoveInt 10 R0",
			"	Return",
		},
	},
	{
		`a := 4 + 5; _ = a`,
		[]string{
			"Package main",
			"",
			"Func main()",
			"	// regs(1,0,0,0)",
			"	MoveInt 9 R0",
			"	Return",
		},
	},
	{
		`a := 0; c := 0; if a < 20 { c = 1 } else { c = 2 }; _ = c`,
		[]string{
			"Package main",
			"",
			"Func main()",
			"      // regs(2,0,0,0)",
			"      MoveInt 0 R0",
			"      MoveInt 0 R1",
			"      IfInt R0 Less 20",
			"      Goto 1",
			"      MoveInt 1 R1",
			"      Goto 2",
			"1:	 MoveInt 2 R1",
			"2:    Return",
		},
	},
}

func TestCompiler(t *testing.T) {
	for _, cas := range expr_tests {
		fullSrc := "package main\nfunc main(){\n" + cas.src + "\nreturn\n\n}\n"
		r := parser.MapReader{"/test.go": []byte(fullSrc)}
		comp := NewCompiler(r, nil)
		pkg, err := comp.Compile("/test.go")
		if err != nil {
			t.Errorf("source: %q, compiler error: %s", cas.src, err)
			continue
		}
		got := &bytes.Buffer{}
		_, err = Disassemble(got, pkg)
		if err != nil {
			t.Errorf("source: %q, disassemble error: %s", cas.src, err)
			continue
		}
		gotLines := []string{}
		for _, line := range strings.Split(strings.TrimSpace(got.String()), "\n") {
			gotLines = append(gotLines, line)
		}
		if diff := equal(cas.expected, gotLines); diff >= 0 {
			if !testing.Verbose() {
				t.Errorf("disassembler output doesn't match for source %q (run tests in verbose mode for further details)", cas.src)
			} else {
				out := &bytes.Buffer{}
				const padding = 3
				w := tabwriter.NewWriter(out, 0, 0, padding, ' ', tabwriter.Debug)
				fmt.Fprintf(w, "expected\t  got\t\n")
				fmt.Fprintf(w, "--------\t  ---\t\n")
				longest := len(cas.expected)
				if len(gotLines) > longest {
					longest = len(gotLines)
				}
				for i := 0; i < longest; i++ {
					e := " "
					g := " "
					if i <= len(cas.expected)-1 {
						e = cas.expected[i]
					}
					if i <= len(gotLines)-1 {
						g = gotLines[i]
					}
					e = removeTabs(e)
					g = removeTabs(g)
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
}
