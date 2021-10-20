// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package misc

import (
	"context"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/internal/fstest"
	"github.com/open2b/scriggo/native"
)

type TypeStruct struct{}

func (t TypeStruct) Method() {}

// TestIssue403 executes a test the issue
// https://github.com/open2b/scriggo/issues/403. This issue cannot be tested
// with usual tests as it would require an execution environment that would be
// hard to reproduce without writing host code
func TestIssue403(t *testing.T) {
	t.Run("Method call on predefined variable", func(t *testing.T) {
		packages := native.CombinedImporter{
			native.Packages{
				"pkg": native.Package{
					Name: "pkg",
					Declarations: native.Declarations{
						"Value": &TypeStruct{},
					},
				},
			},
		}
		main := `
		package main

		import "pkg"

		func main() {
			pkg.Value.Method()
		}`
		fsys := fstest.Files{"main.go": main}
		program, err := scriggo.Build(fsys, &scriggo.BuildOptions{Packages: packages})
		if err != nil {
			t.Fatal(err)
		}
		err = program.Run(nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Method call on not-predefined variable", func(t *testing.T) {
		packages := native.CombinedImporter{
			native.Packages{
				"pkg": native.Package{
					Name: "pkg",
					Declarations: native.Declarations{
						"Type": reflect.TypeOf(new(TypeStruct)).Elem(),
					},
				},
			},
		}
		main := `
		package main

		import "pkg"
		
		func main() {
			t := pkg.Type{}
			t.Method()
		}`
		fsys := fstest.Files{"main.go": main}
		program, err := scriggo.Build(fsys, &scriggo.BuildOptions{Packages: packages})
		if err != nil {
			t.Fatal(err)
		}
		err = program.Run(nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Redeclared package name", func(t *testing.T) {
		packages := native.CombinedImporter{
			native.Packages{
				"pkg": native.Package{
					Name:         "pkg",
					Declarations: native.Declarations{"C": 5},
				},
			},
		}
		main := `
		package main

		import (
			"pkg"
			"pkg"
		)

		func main() {
			_ = pkg.C
		}`
		fsys := fstest.Files{"main.go": main}
		_, err := scriggo.Build(fsys, &scriggo.BuildOptions{Packages: packages})
		if err == nil {
			t.Fatalf("expected build error, got no error")
		}
		const expected = "main:6:4: pkg redeclared as imported package name\n\tmain:5:4: previous declaration"
		if s := err.Error(); s != expected {
			t.Fatalf("expected error %q, got %q", expected, s)
		}
	})
	t.Run("Function that takes a struct as argument", func(t *testing.T) {
		packages := native.CombinedImporter{
			native.Packages{
				"pkg": native.Package{
					Name: "pkg",
					Declarations: native.Declarations{
						"F": func(s struct{}) {},
					},
				},
			},
		}
		main := `
	
		package main

		import "pkg"
		
		func main() {
			t := struct{}{}
			pkg.F(t)
		}
		`
		fsys := fstest.Files{"main.go": main}
		program, err := scriggo.Build(fsys, &scriggo.BuildOptions{Packages: packages})
		if err != nil {
			t.Fatal(err)
		}
		err = program.Run(nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Function taking an array", func(t *testing.T) {
		packages := native.CombinedImporter{
			native.Packages{
				"pkg": native.Package{
					Name: "pkg",
					Declarations: native.Declarations{
						"F": func(s [3]int) {},
					},
				},
			},
		}
		main := `
	
		package main

		import "pkg"
		
		func main() {
			a := [3]int{}
			pkg.F(a)
		}
		`
		fsys := fstest.Files{"main.go": main}
		program, err := scriggo.Build(fsys, &scriggo.BuildOptions{Packages: packages})
		if err != nil {
			t.Fatal(err)
		}
		err = program.Run(nil)
		if err != nil {
			t.Fatal(err)
		}
	})
}

// TestIssue309 executes a test the issue
// https://github.com/open2b/scriggo/issues/309.
func TestIssue309(t *testing.T) {
	t.Run("Add right position to 'imported and not used' errors", func(t *testing.T) {
		packages := native.CombinedImporter{
			native.Packages{
				"pkg": native.Package{
					Name: "pkg",
					Declarations: native.Declarations{
						"Value": &TypeStruct{},
					},
				},
			},
		}
		main := `
        package main

		import (
			"pkg"
		)

		func main() { }`
		fsys := fstest.Files{"main.go": main}
		_, err := scriggo.Build(fsys, &scriggo.BuildOptions{Packages: packages})
		if err == nil {
			t.Fatal("unexpected nil error")
		}
		err2, ok := err.(*scriggo.BuildError)
		if !ok {
			t.Fatalf("unexpected error %s, expecting a compiler error", err)
		}
		expectedPosition := scriggo.Position{Line: 5, Column: 4, Start: 37, End: 41}
		if err2.Position() != expectedPosition {
			t.Fatalf("unexpected position %#v, expecting %#v", err2.Position(), expectedPosition)
		}
	})
}

var compositeStructLiteralTests = []struct {
	fsys fs.FS
	pass bool
	err  string
}{{
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{5} }`,
		"p/p.go":  `package p; type S struct{ F int }`,
	},
	pass: true,
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{3, 5} }`,
		"p/p.go":  `package p; type S struct{ F, f int }`,
	},
	err: "main:1:56: implicit assignment of unexported field 'f' in p.S literal",
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{F: 3} }`,
		"p/p.go":  `package p; type T struct { F int }; type S struct{ T }`,
	},
	err: "main:1:53: cannot use promoted field T.F in struct literal of type p.S",
}}

// TestCompositeStructLiterals tests composite struct literals when the struct
// is defined in another package.
func TestCompositeStructLiterals(t *testing.T) {
	for _, cas := range compositeStructLiteralTests {
		_, err := scriggo.Build(cas.fsys, nil)
		if cas.pass {
			if err != nil {
				t.Fatalf("unexpected error: %q", err)
			}
		} else {
			if err == nil {
				t.Fatalf("expected error %q, got no error", cas.err)
			}
			if cas.err != err.Error() {
				t.Fatalf("expected error %q, got error %q", cas.err, err)
			}
		}
	}
}

var structFieldSelectorTests = []struct {
	fsys fs.FS
	pass bool
	err  string
}{{
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{}.F }`,
		"p/p.go":  `package p; type S struct{ F int }`,
	},
	pass: true,
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{}.f }`,
		"p/p.go":  `package p; type S struct{ f int }`,
	},
	err: "main:1:54: p.S{}.f undefined (cannot refer to unexported field or method f)",
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{}.G }`,
		"p/p.go":  `package p; type S struct{ F int }`,
	},
	err: "main:1:54: p.S{}.G undefined (type S has no field or method G)",
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{}.g }`,
		"p/p.go":  `package p; type S struct{ F int }`,
	},
	err: "main:1:54: p.S{}.g undefined (type S has no field or method g)",
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{}.F }`,
		"p/p.go":  `package p; type ( T struct { F int }; V struct { V int }; S struct{ T; V } )`,
	},
	pass: true,
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{}.F }`,
		"p/p.go":  `package p; type ( T struct { F int }; V struct { F int }; S struct{ T; V } )`,
	},
	err: "main:1:54: ambiguous selector p.S{}.F",
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{}.F }`,
		"p/p.go":  `package p; type ( T struct { F int }; V struct { F int }; S struct{ T; V } )`,
	},
	err: "main:1:54: ambiguous selector p.S{}.F",
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{}.f }`,
		"p/p.go":  `package p; type ( T struct { f int }; V struct { f int }; S struct{ T; V } )`,
	},
	err: "main:1:54: p.S{}.f undefined (type S has no field or method f)", // TODO(marco): S should be p.S
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{}.F }`,
		"p/p.go":  `package p; type ( t struct { F int }; S struct{ t } )`,
	},
	pass: true,
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{}.t.F }`,
		"p/p.go":  `package p; type ( t struct { F int }; S struct{ t } )`,
	},
	err: "main:1:54: p.S{}.t undefined (cannot refer to unexported field or method t)",
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { var _ string = p.S{}.F }`,
		"p/p.go":  `package p; type ( T struct { F int }; A = T; V struct { T; F string; A }; S struct{ V } )`,
	},
	pass: true,
}}

// TestStructFieldSelector tests struct field selectors when the struct is
// defined in another package.
func TestStructFieldSelector(t *testing.T) {
	for _, cas := range structFieldSelectorTests {
		_, err := scriggo.Build(cas.fsys, nil)
		if cas.pass {
			if err != nil {
				t.Fatalf("unexpected error: %q", err)
			}
		} else {
			if err == nil {
				t.Fatalf("expected error %q, got no error", cas.err)
			}
			if cas.err != err.Error() {
				t.Fatalf("expected error %q, got error %q", cas.err, err)
			}
		}
	}
}

var importPackageNameTests = []struct {
	ident string
	name  string
	msg   string
}{
	{``, ``, ``},
	{``, `_`, ``},
	{``, `foo`, ``},
	{``, `Foo`, ``},
	{``, `for`, ``},
	{``, `i n v a l i d`, ``},
	{``, `main`, `import "a.b/p" is a program, not an importable package`},
	{``, `init`, `cannot import package as init - init must be a func`},
	{`main`, `foo`, `main redeclared in this block`},
	{`init`, `foo`, `cannot import package as init - init must be a func`},
}

// TestNativePackageName tests the name of a native package in a import statement.
// defined in another package.
func TestImportPackageName(t *testing.T) {
	fsys := fstest.Files{}
	options := &scriggo.BuildOptions{}
	for _, cas := range importPackageNameTests {
		if cas.ident == "" {
			fsys["main.go"] = `package main; import "a.b/p"; func main() { }`
		} else {
			fsys["main.go"] = `package main; import ` + cas.ident + ` "a.b/p"; func main() { }`
		}
		options.Packages = native.Packages{
			"a.b/p": native.Package{Name: cas.name},
		}
		_, err := scriggo.Build(fsys, options)
		if err == nil {
			t.Fatalf("expected error %q, got no error", cas.msg)
		}
		if cas.msg == "" {
			if !strings.Contains(err.Error(), "imported and not used") {
				t.Fatalf("unexpected error: %q", err)
			}
		} else {
			if err, ok := err.(*scriggo.BuildError); ok {
				if cas.msg != err.Message() {
					t.Fatalf("expected error %q, got error %q", cas.msg, err.Message())
				}
			} else {
				t.Fatalf("expected a *scriggo.BuildError, got %T", err)
			}
		}
	}
}

// TestImportPackageName2 tests that a double import of a native package with
// an invalid package name results in an "imported and not used" error.
func TestImportPackageName2(t *testing.T) {
	fsys := fstest.Files{
		"main.go": `package main; import ( "a.b/p"; "a.b/p" ); func main() { }`,
	}
	options := &scriggo.BuildOptions{
		Packages: native.Packages{
			"a.b/p": native.Package{Name: "$"},
		},
	}
	_, err := scriggo.Build(fsys, options)
	if err == nil {
		t.Fatalf("expected error, got no error")
	}
	if err, ok := err.(*scriggo.BuildError); ok {
		const expected = "imported and not used: \"a.b/p\""
		if msg := err.Message(); msg == "" {
			t.Fatalf("expected error %q, got no error", expected)
		} else if msg != expected {
			t.Fatalf("expected error %q, got error %q", expected, msg)
		}
	} else {
		t.Fatalf("expected a *scriggo.BuildError, got %T", err)
	}
}

func TestIssue523(t *testing.T) {
	// See https://github.com/open2b/scriggo/issues/523.
	src := `package main

	import "fmt"

	func main() {
		fmt.Println("hello")
	}`
	fsys := fstest.Files{"main.go": src}
	_, _ = scriggo.Build(fsys, nil)
}

var ctxCancellationTests = []string{
	`
		package main
		func main() {
			for {}
		}`,
	`
		package main
		func main() {
			select {}
		}`,
	`
		package main
		func main() {
			ch := make(chan int)
			ch <- 1
		}`,
	`
		package main
		func main() {
			ch := make(chan int)
			_ = <-ch
		}`,
}

// TestContextCancellation tests the cancellation of the context.
func TestContextCancellation(t *testing.T) {
	for _, src := range ctxCancellationTests {
		fsys := fstest.Files{"main.go": src}
		program, err := scriggo.Build(fsys, nil)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		opts := &scriggo.RunOptions{Context: ctx}
		err = program.Run(opts)
		if err == nil {
			t.Fatal("expected stop error, got no error", err)
		}
		if err != context.DeadlineExceeded {
			t.Fatalf("expected deadline error, got %s", err)
		}
	}
}

// https://github.com/open2b/scriggo/issues/855
func TestIssue855(t *testing.T) {
	fsys := fstest.Files{
		"index.go": `
			package main

			import . "test/imported1"

			func main() {
				_ = V2
			}
		`,
		"imported1/imported1.go": `
			package imported1

			import . "test/imported2"

			var _ = V2
		`,
		"imported2/imported2.go": `
			package imported2

			var V2 = 2
		`,

		"go.mod": `module test`,
	}
	var gotErr string
	_, err := scriggo.Build(fsys, nil)
	if err != nil {
		gotErr = err.Error()
	}
	expectedErr := "main:7:9: undefined: V2"
	if gotErr != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, gotErr)
	}
}
