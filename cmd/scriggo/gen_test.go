package main

import (
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func Test_renderPackages(t *testing.T) {
	// NOTE: these tests ignores whitespaces, imports and comments.
	cases := map[string]struct {
		pd               scriggoDescriptor
		pkgsVariableName string
		goos             string
		expected         string
	}{
		"Importing archive/tar simple": {
			pd: scriggoDescriptor{
				pkgName: "test",
				imports: []importDescriptor{
					importDescriptor{path: "archive/tar"},
				},
			},
			expected: `package test

			import (
				tar "archive/tar"
			)
			
			import . "scriggo"
			import "reflect"
			
			func init() {
				packages = Packages{
			
					"archive/tar": {
						Name: "tar",
						Declarations: map[string]interface{}{
							"ErrFieldTooLong":    &tar.ErrFieldTooLong,
							"ErrHeader":          &tar.ErrHeader,
							"ErrWriteAfterClose": &tar.ErrWriteAfterClose,
							"ErrWriteTooLong":    &tar.ErrWriteTooLong,
							"FileInfoHeader":     tar.FileInfoHeader,
							"Format":             reflect.TypeOf(new(tar.Format)).Elem(),
							"FormatGNU":          ConstValue(tar.FormatGNU),
							"FormatPAX":          ConstValue(tar.FormatPAX),
							"FormatUSTAR":        ConstValue(tar.FormatUSTAR),
							"FormatUnknown":      ConstValue(tar.FormatUnknown),
							"Header":             reflect.TypeOf(tar.Header{}),
							"NewReader":          tar.NewReader,
							"NewWriter":          tar.NewWriter,
							"Reader":             reflect.TypeOf(tar.Reader{}),
							"TypeBlock":          ConstValue(tar.TypeBlock),
							"TypeChar":           ConstValue(tar.TypeChar),
							"TypeCont":           ConstValue(tar.TypeCont),
							"TypeDir":            ConstValue(tar.TypeDir),
							"TypeFifo":           ConstValue(tar.TypeFifo),
							"TypeGNULongLink":    ConstValue(tar.TypeGNULongLink),
							"TypeGNULongName":    ConstValue(tar.TypeGNULongName),
							"TypeGNUSparse":      ConstValue(tar.TypeGNUSparse),
							"TypeLink":           ConstValue(tar.TypeLink),
							"TypeReg":            ConstValue(tar.TypeReg),
							"TypeRegA":           ConstValue(tar.TypeRegA),
							"TypeSymlink":        ConstValue(tar.TypeSymlink),
							"TypeXGlobalHeader":  ConstValue(tar.TypeXGlobalHeader),
							"TypeXHeader":        ConstValue(tar.TypeXHeader),
							"Writer":             reflect.TypeOf(tar.Writer{}),
						},
					},
				}
			}`,
		},
		"Importing fmt simple": {
			pd: scriggoDescriptor{
				pkgName: "test",
				imports: []importDescriptor{
					importDescriptor{path: "fmt"},
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
							"Formatter":  reflect.TypeOf(fmt.Formatter(nil)),
							"Fprint":     fmt.Fprint,
							"Fprintf":    fmt.Fprintf,
							"Fprintln":   fmt.Fprintln,
							"Fscan":      fmt.Fscan,
							"Fscanf":     fmt.Fscanf,
							"Fscanln":    fmt.Fscanln,
							"GoStringer": reflect.TypeOf(fmt.GoStringer(nil)),
							"Print":      fmt.Print,
							"Printf":     fmt.Printf,
							"Println":    fmt.Println,
							"Scan":       fmt.Scan,
							"ScanState":  reflect.TypeOf(fmt.ScanState(nil)),
							"Scanf":      fmt.Scanf,
							"Scanln":     fmt.Scanln,
							"Scanner":    reflect.TypeOf(fmt.Scanner(nil)),
							"Sprint":     fmt.Sprint,
							"Sprintf":    fmt.Sprintf,
							"Sprintln":   fmt.Sprintln,
							"Sscan":      fmt.Sscan,
							"Sscanf":     fmt.Sscanf,
							"Sscanln":    fmt.Sscanln,
							"State":      reflect.TypeOf(fmt.State(nil)),
							"Stringer":   reflect.TypeOf(fmt.Stringer(nil)),
						},
					},
				}
			}`,
		},
		"Importing only Println from fmt": {
			pd: scriggoDescriptor{
				pkgName: "test",
				imports: []importDescriptor{
					importDescriptor{
						path: "fmt",
						comment: importComment{
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
			got, content, err := renderPackages(c.pd, c.pkgsVariableName, c.goos)
			if err != nil {
				t.Fatal(err)
			}
			if !content {
				t.Fatalf("no content generated")
			}
			got = _cleanOutput(got)
			c.expected = _cleanOutput(c.expected)
			if got != c.expected {
				if testing.Verbose() {
					t.Fatalf("expecting:\n\n%s\n\ngot:\n\n%s", c.expected, got)
				}
				t.Fatalf("expecting %q, got %q", c.expected, got)
			}
		})
	}
}

func Test_renderPackageMain(t *testing.T) {
	// NOTE: these tests ignores whitespaces, imports and comments.
	cases := map[string]struct {
		pd               scriggoDescriptor
		pkgsVariableName string
		goos             string
		expected         string
	}{
		"println e print taken from fmt": {
			pd: scriggoDescriptor{
				imports: []importDescriptor{
					importDescriptor{
						path: "fmt",
						comment: importComment{
							main:         true,
							uncapitalize: true,
							export:       []string{"Print", "Println"},
						},
					},
				},
			},
			expected: `package main

			import (
				"fmt"
			)
			
			func init() {
				Main = &Package{
					Name: "main",
					Declarations: map[string]interface{}{
						"print":   fmt.Print,
						"println": fmt.Println,
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
			got, err := renderPackageMain(c.pd, c.goos)
			if err != nil {
				t.Fatal(err)
			}

			got = _cleanOutput(got)
			c.expected = _cleanOutput(c.expected)
			if got != c.expected {
				if testing.Verbose() {
					t.Fatalf("expecting:\n\n%s\n\ngot:\n\n%s", c.expected, got)
				}
				t.Fatalf("expecting %q, got %q", c.expected, got)
			}
		})
	}
}

func _cleanOutput(s string) string {
	re := regexp.MustCompile(`(?s)import \(.*?\)`)
	s = re.ReplaceAllString(s, "")
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
