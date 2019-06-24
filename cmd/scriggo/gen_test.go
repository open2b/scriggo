package main

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func BenchmarkGeneratePackage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, v, err := parseGoPackage("fmt", "")
		if err != nil {
			b.Fatal(err)
		}
		_ = v
	}
}

func Test_renderPackages(t *testing.T) {
	// NOTE: these tests ignores whitespaces, imports and comments.
	cases := map[string]struct {
		pd               packageDef
		pkgsVariableName string
		goos             string
		expected         string
	}{
		"Importing fmt simple": {
			pd: packageDef{
				name: "test",
				imports: []importDef{
					importDef{path: "fmt"},
				},
			},
			expected: `package test

			import (
				fmt "fmt"
			)
			
			import . "scriggo"
			import "reflect"
			
			func init() {
				packages = Packages{
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
				}
			}`,
		},
		"Importing only Println from fmt": {
			pd: packageDef{
				name: "test",
				imports: []importDef{
					importDef{
						path: "fmt",
						commentTag: commentTag{
							export: []string{"Println"},
						},
					},
				},
			},
			expected: `package test

			import (
				fmt "fmt"
			)
			
			import . "scriggo"
			
			func init() {
				packages = Packages{
					"fmt": {
						Name: "fmt",
						Declarations: map[string]interface{}{
							"Println": fmt.Println,
						},
					},
				}
			}
			`,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if c.goos == "" {
				c.goos = os.Getenv("GOOS")
				if c.goos == "" {
					c.goos = runtime.GOOS
				}
			}
			if c.pkgsVariableName == "" {
				c.pkgsVariableName = "packages"
			}
			got, _, err := renderPackages(c.pd, c.pkgsVariableName, c.goos)
			if err != nil {
				t.Fatal(err)
			}
			clean := func(s string) string {
				lines := []string{}
				for _, l := range strings.Split(s, "\n") {
					l := strings.TrimSpace(l)
					if l != "" && !strings.HasPrefix(l, "//") && !strings.HasPrefix(l, "import ") {
						l := strings.Join(strings.Fields(l), " ")
						lines = append(lines, l)
					}
				}
				return strings.Join(lines, "\n")
			}
			got = clean(got)
			c.expected = clean(c.expected)
			if got != c.expected {
				if testing.Verbose() {
					t.Fatalf("expecting:\n\n%s\n\ngot:\n\n%s", c.expected, got)
				}
				t.Fatalf("expecting %q, got %q", c.expected, got)
			}
		})
	}
}
