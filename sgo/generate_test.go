package main

import (
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func Test_renderPackages(t *testing.T) {
	// NOTE: these tests ignores whitespaces, imports and comments.
	cases := map[string]struct {
		pd           *scriggofile
		pkgsVariable string
		goos         string
		expected     string
	}{
		"Importing fmt with an alternative path": {
			pd: &scriggofile{
				pkgName: "test",
				imports: []*importCommand{
					{path: "fmt", asPath: "custom/fmt/path"},
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
					"custom/fmt/path": {
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
		"Importing archive/tar simple": {
			pd: &scriggofile{
				pkgName: "test",
				imports: []*importCommand{{path: "archive/tar"}},
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
			pd: &scriggofile{
				pkgName: "test",
				imports: []*importCommand{{path: "fmt"}},
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
			pd: &scriggofile{
				pkgName: "test",
				imports: []*importCommand{
					{
						path:      "fmt",
						including: []string{"Println"},
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
			if c.pkgsVariable == "" {
				c.pkgsVariable = "packages"
			}
			got, content, err := renderPackages(c.pd, c.pkgsVariable, c.goos)
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

func Test_parseGoPackage(t *testing.T) {
	cases := map[string]struct {
		name  string            // package name.
		decls map[string]string // package declarations.
	}{
		"fmt": {
			name: "fmt",
			decls: map[string]string{
				"Errorf":     "fmt.Errorf",
				"Formatter":  "reflect.TypeOf(new(fmt.Formatter)).Elem()",
				"Fprint":     "fmt.Fprint",
				"Fprintf":    "fmt.Fprintf",
				"Fprintln":   "fmt.Fprintln",
				"Fscan":      "fmt.Fscan",
				"Fscanf":     "fmt.Fscanf",
				"Fscanln":    "fmt.Fscanln",
				"GoStringer": "reflect.TypeOf(new(fmt.GoStringer)).Elem()",
				"Print":      "fmt.Print",
				"Printf":     "fmt.Printf",
				"Println":    "fmt.Println",
				"Scan":       "fmt.Scan",
				"ScanState":  "reflect.TypeOf(new(fmt.ScanState)).Elem()",
				"Scanf":      "fmt.Scanf",
				"Scanln":     "fmt.Scanln",
				"Scanner":    "reflect.TypeOf(new(fmt.Scanner)).Elem()",
				"Sprint":     "fmt.Sprint",
				"Sprintf":    "fmt.Sprintf",
				"Sprintln":   "fmt.Sprintln",
				"Sscan":      "fmt.Sscan",
				"Sscanf":     "fmt.Sscanf",
				"Sscanln":    "fmt.Sscanln",
				"State":      "reflect.TypeOf(new(fmt.State)).Elem()",
				"Stringer":   "reflect.TypeOf(new(fmt.Stringer)).Elem()"},
		},
		"archive/tar": {
			name: "tar",
			decls: map[string]string{
				"ErrFieldTooLong":    "&tar.ErrFieldTooLong",
				"ErrHeader":          "&tar.ErrHeader",
				"ErrWriteAfterClose": "&tar.ErrWriteAfterClose",
				"ErrWriteTooLong":    "&tar.ErrWriteTooLong",
				"FileInfoHeader":     "tar.FileInfoHeader",
				"Format":             "reflect.TypeOf(new(tar.Format)).Elem()",
				"FormatGNU":          "ConstValue(tar.FormatGNU)",
				"FormatPAX":          "ConstValue(tar.FormatPAX)",
				"FormatUSTAR":        "ConstValue(tar.FormatUSTAR)",
				"FormatUnknown":      "ConstValue(tar.FormatUnknown)",
				"Header":             "reflect.TypeOf(tar.Header{})",
				"NewReader":          "tar.NewReader",
				"NewWriter":          "tar.NewWriter",
				"Reader":             "reflect.TypeOf(tar.Reader{})",
				"TypeBlock":          "ConstValue(tar.TypeBlock)",
				"TypeChar":           "ConstValue(tar.TypeChar)",
				"TypeCont":           "ConstValue(tar.TypeCont)",
				"TypeDir":            "ConstValue(tar.TypeDir)",
				"TypeFifo":           "ConstValue(tar.TypeFifo)",
				"TypeGNULongLink":    "ConstValue(tar.TypeGNULongLink)",
				"TypeGNULongName":    "ConstValue(tar.TypeGNULongName)",
				"TypeGNUSparse":      "ConstValue(tar.TypeGNUSparse)",
				"TypeLink":           "ConstValue(tar.TypeLink)",
				"TypeReg":            "ConstValue(tar.TypeReg)",
				"TypeRegA":           "ConstValue(tar.TypeRegA)",
				"TypeSymlink":        "ConstValue(tar.TypeSymlink)",
				"TypeXGlobalHeader":  "ConstValue(tar.TypeXGlobalHeader)",
				"TypeXHeader":        "ConstValue(tar.TypeXHeader)",
				"Writer":             "reflect.TypeOf(tar.Writer{})",
			},
		},
	}
	goos := "linux" // paths in this test should be OS-independent.
	for path, expected := range cases {
		t.Run(path, func(t *testing.T) {
			gotName, gotDecls, err := parseGoPackage(path, goos)
			if err != nil {
				t.Fatal(err)
			}
			if gotName != expected.name {
				t.Fatalf("path %q: expecting name %q, got %q", path, expected.name, gotName)
			}
			if len(gotDecls) != len(expected.decls) {
				t.Fatalf("path %q: expecting %#v, %#v", path, expected.decls, gotDecls)
			}
			if !reflect.DeepEqual(gotDecls, expected.decls) {
				t.Fatalf("path %q: expecting %#v, %#v", path, expected.decls, gotDecls)
			}
		})
	}
}
