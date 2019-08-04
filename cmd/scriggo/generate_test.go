package main

import (
	"bytes"
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
		sf       *scriggofile
		goos     string
		expected string
	}{
		"Importing fmt with an alternative path": {
			sf: &scriggofile{
				pkgName:  "test",
				variable: "packages",
				imports: []*importCommand{
					{path: "fmt", asPath: "custom/fmt/path"},
				},
			},
			expected: `package test

			import (
				"fmt"
			)

			import . "scriggo"
			import "reflect"

			func init() {
				packages = make(Packages, 1)
				var decs map[string]interface{}
				// "custom/fmt/path"
				decs = make(map[string]interface{}, 25)
				decs["Errorf"] = fmt.Errorf
				decs["Formatter"] = reflect.TypeOf((*fmt.Formatter)(nil)).Elem()
				decs["Fprint"] = fmt.Fprint
				decs["Fprintf"] = fmt.Fprintf
				decs["Fprintln"] = fmt.Fprintln
				decs["Fscan"] = fmt.Fscan
				decs["Fscanf"] = fmt.Fscanf
				decs["Fscanln"] = fmt.Fscanln
				decs["GoStringer"] = reflect.TypeOf((*fmt.GoStringer)(nil)).Elem()
				decs["Print"] = fmt.Print
				decs["Printf"] = fmt.Printf
				decs["Println"] = fmt.Println
				decs["Scan"] = fmt.Scan
				decs["ScanState"] = reflect.TypeOf((*fmt.ScanState)(nil)).Elem()
				decs["Scanf"] = fmt.Scanf
				decs["Scanln"] = fmt.Scanln
				decs["Scanner"] = reflect.TypeOf((*fmt.Scanner)(nil)).Elem()
				decs["Sprint"] = fmt.Sprint
				decs["Sprintf"] = fmt.Sprintf
				decs["Sprintln"] = fmt.Sprintln
				decs["Sscan"] = fmt.Sscan
				decs["Sscanf"] = fmt.Sscanf
				decs["Sscanln"] = fmt.Sscanln
				decs["State"] = reflect.TypeOf((*fmt.State)(nil)).Elem()
				decs["Stringer"] = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
				packages["custom/fmt/path"] = &MapPackage{
					PkgName: "fmt",
					Declarations: decs,
				}
			}`,
		},
		"Importing archive/tar simple": {
			sf: &scriggofile{
				pkgName:  "test",
				variable: "packages",
				imports:  []*importCommand{{path: "archive/tar"}},
			},
			expected: `package test

			import (
				tar "archive/tar"
			)

			import . "scriggo"
			import "reflect"

			func init() {
				packages = make(Packages, 1)
				var decs map[string]interface{}
				// "archive/tar"
				decs = make(map[string]interface{}, 29)
				decs["ErrFieldTooLong"] = &tar.ErrFieldTooLong
				decs["ErrHeader"] = &tar.ErrHeader
				decs["ErrWriteAfterClose"] = &tar.ErrWriteAfterClose
				decs["ErrWriteTooLong"] = &tar.ErrWriteTooLong
				decs["FileInfoHeader"] = tar.FileInfoHeader
				decs["Format"] = reflect.TypeOf((*tar.Format)(nil)).Elem()
				decs["FormatGNU"] = tar.FormatGNU
				decs["FormatPAX"] = tar.FormatPAX
				decs["FormatUSTAR"] = tar.FormatUSTAR
				decs["FormatUnknown"] = tar.FormatUnknown
				decs["Header"] = reflect.TypeOf((*tar.Header)(nil)).Elem()
				decs["NewReader"] = tar.NewReader
				decs["NewWriter"] = tar.NewWriter
				decs["Reader"] = reflect.TypeOf((*tar.Reader)(nil)).Elem()
				decs["TypeBlock"] = UntypedConstant("52")
				decs["TypeChar"] = UntypedConstant("51")
				decs["TypeCont"] = UntypedConstant("55")
				decs["TypeDir"] = UntypedConstant("53")
				decs["TypeFifo"] = UntypedConstant("54")
				decs["TypeGNULongLink"] = UntypedConstant("75")
				decs["TypeGNULongName"] = UntypedConstant("76")
				decs["TypeGNUSparse"] = UntypedConstant("83")
				decs["TypeLink"] = UntypedConstant("49")
				decs["TypeReg"] = UntypedConstant("48")
				decs["TypeRegA"] = UntypedConstant("0")
				decs["TypeSymlink"] = UntypedConstant("50")
				decs["TypeXGlobalHeader"] = UntypedConstant("103")
				decs["TypeXHeader"] = UntypedConstant("120")
				decs["Writer"] = reflect.TypeOf((*tar.Writer)(nil)).Elem()
				packages["archive/tar"] = &MapPackage{
					PkgName: "tar",
					Declarations: decs,
				}
			}`,
		},
		"Importing fmt simple": {
			sf: &scriggofile{
				pkgName:  "test",
				variable: "packages",
				imports:  []*importCommand{{path: "fmt"}},
			},
			expected: `package test

			import (
				"fmt"
			)

			import . "scriggo"
			import "reflect"

			func init() {
				packages = make(Packages, 1)
				var decs map[string]interface{}
				// "fmt"
				decs = make(map[string]interface{}, 25)
				decs["Errorf"] = fmt.Errorf
				decs["Formatter"] = reflect.TypeOf((*fmt.Formatter)(nil)).Elem()
				decs["Fprint"] = fmt.Fprint
				decs["Fprintf"] = fmt.Fprintf
				decs["Fprintln"] = fmt.Fprintln
				decs["Fscan"] = fmt.Fscan
				decs["Fscanf"] = fmt.Fscanf
				decs["Fscanln"] = fmt.Fscanln
				decs["GoStringer"] = reflect.TypeOf((*fmt.GoStringer)(nil)).Elem()
				decs["Print"] = fmt.Print
				decs["Printf"] = fmt.Printf
				decs["Println"] = fmt.Println
				decs["Scan"] = fmt.Scan
				decs["ScanState"] = reflect.TypeOf((*fmt.ScanState)(nil)).Elem()
				decs["Scanf"] = fmt.Scanf
				decs["Scanln"] = fmt.Scanln
				decs["Scanner"] = reflect.TypeOf((*fmt.Scanner)(nil)).Elem()
				decs["Sprint"] = fmt.Sprint
				decs["Sprintf"] = fmt.Sprintf
				decs["Sprintln"] = fmt.Sprintln
				decs["Sscan"] = fmt.Sscan
				decs["Sscanf"] = fmt.Sscanf
				decs["Sscanln"] = fmt.Sscanln
				decs["State"] = reflect.TypeOf((*fmt.State)(nil)).Elem()
				decs["Stringer"] = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
				packages["fmt"] = &MapPackage{
					PkgName: "fmt",
					Declarations: decs,
				}
			}`,
		},
		"Importing only Println from fmt": {
			sf: &scriggofile{
				pkgName:  "test",
				variable: "packages",
				imports: []*importCommand{
					{
						path:      "fmt",
						including: []string{"Println"},
					},
				},
			},
			expected: `package test

			import (
				"fmt"
			)

			import . "scriggo"
			import "reflect"

			func init() {
				packages = make(Packages, 1)
				var decs map[string]interface{}
				// "fmt"
				decs = make(map[string]interface{}, 1)
				decs["Println"] = fmt.Println
				packages["fmt"] = &MapPackage{
					PkgName: "fmt",
					Declarations: decs,
				}
			}`,
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
			b := bytes.Buffer{}
			err := renderPackages(&b, "", c.sf, c.goos, buildFlags{})
			if err != nil {
				t.Fatal(err)
			}
			got := _cleanOutput(b.String())
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
				"Formatter":  "reflect.TypeOf((*fmt.Formatter)(nil)).Elem()",
				"Fprint":     "fmt.Fprint",
				"Fprintf":    "fmt.Fprintf",
				"Fprintln":   "fmt.Fprintln",
				"Fscan":      "fmt.Fscan",
				"Fscanf":     "fmt.Fscanf",
				"Fscanln":    "fmt.Fscanln",
				"GoStringer": "reflect.TypeOf((*fmt.GoStringer)(nil)).Elem()",
				"Print":      "fmt.Print",
				"Printf":     "fmt.Printf",
				"Println":    "fmt.Println",
				"Scan":       "fmt.Scan",
				"ScanState":  "reflect.TypeOf((*fmt.ScanState)(nil)).Elem()",
				"Scanf":      "fmt.Scanf",
				"Scanln":     "fmt.Scanln",
				"Scanner":    "reflect.TypeOf((*fmt.Scanner)(nil)).Elem()",
				"Sprint":     "fmt.Sprint",
				"Sprintf":    "fmt.Sprintf",
				"Sprintln":   "fmt.Sprintln",
				"Sscan":      "fmt.Sscan",
				"Sscanf":     "fmt.Sscanf",
				"Sscanln":    "fmt.Sscanln",
				"State":      "reflect.TypeOf((*fmt.State)(nil)).Elem()",
				"Stringer":   "reflect.TypeOf((*fmt.Stringer)(nil)).Elem()",
			},
		},
		"archive/tar": {
			name: "tar",
			decls: map[string]string{
				"ErrFieldTooLong":    "&tar.ErrFieldTooLong",
				"ErrHeader":          "&tar.ErrHeader",
				"ErrWriteAfterClose": "&tar.ErrWriteAfterClose",
				"ErrWriteTooLong":    "&tar.ErrWriteTooLong",
				"FileInfoHeader":     "tar.FileInfoHeader",
				"Format":             "reflect.TypeOf((*tar.Format)(nil)).Elem()",
				"FormatGNU":          "tar.FormatGNU",
				"FormatPAX":          "tar.FormatPAX",
				"FormatUSTAR":        "tar.FormatUSTAR",
				"FormatUnknown":      "tar.FormatUnknown",
				"Header":             "reflect.TypeOf((*tar.Header)(nil)).Elem()",
				"NewReader":          "tar.NewReader",
				"NewWriter":          "tar.NewWriter",
				"Reader":             "reflect.TypeOf((*tar.Reader)(nil)).Elem()",
				"TypeBlock":          "UntypedConstant(\"52\")",
				"TypeChar":           "UntypedConstant(\"51\")",
				"TypeCont":           "UntypedConstant(\"55\")",
				"TypeDir":            "UntypedConstant(\"53\")",
				"TypeFifo":           "UntypedConstant(\"54\")",
				"TypeGNULongLink":    "UntypedConstant(\"75\")",
				"TypeGNULongName":    "UntypedConstant(\"76\")",
				"TypeGNUSparse":      "UntypedConstant(\"83\")",
				"TypeLink":           "UntypedConstant(\"49\")",
				"TypeReg":            "UntypedConstant(\"48\")",
				"TypeRegA":           "UntypedConstant(\"0\")",
				"TypeSymlink":        "UntypedConstant(\"50\")",
				"TypeXGlobalHeader":  "UntypedConstant(\"103\")",
				"TypeXHeader":        "UntypedConstant(\"120\")",
				"Writer":             "reflect.TypeOf((*tar.Writer)(nil)).Elem()",
			},
		},
	}
	goos := "linux" // paths in this test should be OS-independent.
	for path, expected := range cases {
		t.Run(path, func(t *testing.T) {
			gotName, gotDecls, _, err := loadGoPackage(path, "", goos, buildFlags{}, nil, nil)
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
