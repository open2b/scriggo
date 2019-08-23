// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"testing"

	"scriggo"
)

func TestProgramImport(t *testing.T) {
	cases := map[string]scriggo.MapStringLoader{
		`Just package "main", no imports`: {
			"main": `package main
				func main() {
				}`},

		`"main" importing "pkg"`: {
			"main": `package main
				import "pkg"
				func main() {
					pkg.F()
				}`,
			"pkg": `package pkg
				func F() {
					print("called pkg.F()")
				}`},

		`"main" importing "pkg1" and "pkg2`: {
			"main": `package main
				import "pkg1"
				import "pkg2"
				func main() {
					pkg1.F1()
					pkg2.F2()
				}`,
			"pkg1": `package pkg1
				func F1() {
					print("called pkg1.F1()")
				}`,
			"pkg2": `package pkg2
				func F2() {
					print("called pkg2.F2()")
				}`},

		`"main" importing "pkg1" importing "pkg2" (1)`: {
			"main": `package main
				import "pkg1"
				func main() {
					pkg1.F()
				}`,
			"pkg1": `package pkg1
				import "pkg2"
				func F() {
					pkg2.G()
				}`,
			"pkg2": `package pkg2
				func G() {
					print("called pkg2.G()")
				}`},

		`"main" importing "pkg1" importing "pkg2" (dot import)`: {
			"main": `package main
				import . "pkg1"
				func main() {
					F()
				}`,
			"pkg1": `package pkg1
				import p2 "pkg2"
				func F() {
					p2.G()
				}`,
			"pkg2": `package pkg2
				func G() {
					print("called pkg1.G()")
				}`},

		`"main" importing a package variable from "pkg1`: {
			"main": `package main
				import "pkg1"
				func main() {
					v := pkg1.V
					_ = v
				}`,
			"pkg1": `package pkg1
				var V = 10`,
		},
	}
	for name, loader := range cases {
		t.Run(name, func(t *testing.T) {
			program, err := scriggo.Load(loader, &scriggo.LoadOptions{LimitMemorySize: true})
			if err != nil {
				t.Errorf("compiling error: %s", err)
				return
			}
			err = program.Run(&scriggo.RunOptions{MaxMemorySize: 1000000})
			if err != nil {
				t.Errorf("execution error: %s", err)
				return
			}
		})
	}
}
