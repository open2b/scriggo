package vm

import (
	"bytes"
	"scrigo/parser"
	"strings"
	"testing"
)

func TestCompiler(t *testing.T) {
	cases := map[string]string{

		`a := 10; _ = a`: `
		Package main
		Func main()
			// regs(1,0,0,0)
			MoveInt 10 R0`,

		`a := 4 + 5; _ = a`: `
		Package main
		Func main()
			// regs(1,0,0,0)
			MoveInt 9 R0
		`,
	}
	for src, expected := range cases {
		clean := func(s string) string {
			s = strings.ReplaceAll(s, "\n", "")
			s = strings.ReplaceAll(s, "\t", "")
			return s
		}
		fullSrc := "package main\nfunc main(){\n" + src + "\n}\n"
		r := parser.MapReader{"/test.go": []byte(fullSrc)}
		comp := NewCompiler(r, nil)
		pkg, err := comp.Compile("/test.go")
		if err != nil {
			t.Errorf("source: %q, compiler error: %s", src, err)
			continue
		}
		got := &bytes.Buffer{}
		_, err = Disassemble(got, pkg)
		if err != nil {
			t.Errorf("source: %q, disassemble error: %s", src, err)
			continue
		}
		if clean(got.String()) != clean(expected) {
			if testing.Verbose() {
				expected = strings.ReplaceAll(expected, "\t\t", "")
				t.Errorf("source: %q:\nexpected\n----------\n%s\ngot\n----------\n%s", src, strings.TrimSpace(expected), got.String())
			} else {
				t.Errorf("source: %q, expected: %q, got %q", src, clean(expected), clean(got.String()))
			}
		}
	}
}
