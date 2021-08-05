// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/internal/fstest"
)

func TestProgramImport(t *testing.T) {
	cases := map[string]fstest.Files{
		`Just package "main", no imports`: {
			"main.go": `package main
				func main() {
				}`},

		`"main" importing "pkg"`: {
			"go.mod": "module a.b",
			"main.go": `package main
				import "a.b/pkg"
				func main() {
					pkg.F()
				}`,
			"pkg/pkg.go": `package pkg
				func F() {
					print("called pkg.F()")
				}`},

		`"main" importing "pkg1" and "pkg2`: {
			"go.mod": "module a.b",
			"main.go": `package main
				import "a.b/pkg1"
				import "a.b/pkg2"
				func main() {
					pkg1.F1()
					pkg2.F2()
				}`,
			"pkg1/pk1.go": `package pkg1
				func F1() {
					print("called pkg1.F1()")
				}`,
			"pkg2/pkg2.go": `package pkg2
				func F2() {
					print("called pkg2.F2()")
				}`},

		`"main" importing "pkg1" importing "pkg2" (1)`: {
			"go.mod": "module a.b",
			"main.go": `package main
				import "a.b/pkg1"
				func main() {
					pkg1.F()
				}`,
			"pkg1/pk1.go": `package pkg1
				import "a.b/pkg2"
				func F() {
					pkg2.G()
				}`,
			"pkg2/pkg2.go": `package pkg2
				func G() {
					print("called pkg2.G()")
				}`},

		`"main" importing "pkg1" importing "pkg2" (dot import)`: {
			"go.mod": "module a.b",
			"main.go": `package main
				import . "a.b/pkg1"
				func main() {
					F()
				}`,
			"pkg1/pkg1.go": `package pkg1
				import p2 "a.b/pkg2"
				func F() {
					p2.G()
				}`,
			"pkg2/pkg2.go": `package pkg2
				func G() {
					print("called pkg1.G()")
				}`},

		`"main" importing a package variable from "pkg1`: {
			"go.mod": "module a.b",
			"main.go": `package main
				import "a.b/pkg1"
				func main() {
					v := pkg1.V
					_ = v
				}`,
			"pkg1/pkg1.go": `package pkg1
				var V = 10`,
		},
	}
	for name, fsys := range cases {
		t.Run(name, func(t *testing.T) {
			program, err := scriggo.Build(fsys, nil)
			if err != nil {
				t.Errorf("compiling error: %s", err)
				return
			}
			_, err = program.Run(nil)
			if err != nil {
				t.Errorf("execution error: %s", err)
				return
			}
		})
	}
}
