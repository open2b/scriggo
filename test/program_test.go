// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"os"
	"reflect"
	"testing"

	"scriggo"
)

type TypeStruct struct{}

func (t TypeStruct) Method() {}

// TestIssue403 executes a test the issue
// https://github.com/open2b/scriggo/issues/403. This issue cannot be tested
// with usual tests as it would require an execution environment that would be
// hard to reproduce without writing host code
func TestIssue403(t *testing.T) {
	t.Run("Method call on predefined variable", func(t *testing.T) {
		loaders := scriggo.CombinedLoader{
			scriggo.Packages{
				"pkg": &scriggo.MapPackage{
					PkgName: "pkg",
					Declarations: map[string]interface{}{
						"Value": &TypeStruct{},
					},
				},
			},
			&scriggo.MapStringLoader{
				"main": `
					package main

					import "pkg"
		
					func main() {
						pkg.Value.Method()
					}`,
			},
		}
		program, err := scriggo.Load(loaders, nil)
		if err != nil {
			t.Fatal(err)
		}
		err = program.Run(nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Method call on not-predefined variable", func(t *testing.T) {
		loaders := scriggo.CombinedLoader{
			scriggo.Packages{
				"pkg": &scriggo.MapPackage{
					PkgName: "pkg",
					Declarations: map[string]interface{}{
						"Type": reflect.TypeOf(new(TypeStruct)).Elem(),
					},
				},
			},
			&scriggo.MapStringLoader{
				"main": `
				package main
	
				import "pkg"
				
				func main() {
					t := pkg.Type{}
					t.Method()
				}`,
			},
		}
		program, err := scriggo.Load(loaders, nil)
		if err != nil {
			t.Fatal(err)
		}
		err = program.Run(nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Function that takes a struct as argument", func(t *testing.T) {
		loaders := scriggo.CombinedLoader{
			scriggo.Packages{
				"pkg": &scriggo.MapPackage{
					PkgName: "pkg",
					Declarations: map[string]interface{}{
						"F": func(s struct{}) {},
					},
				},
			},
			&scriggo.MapStringLoader{
				"main": `
	
				package main
	
				import "pkg"
				
				func main() {
					t := struct{}{}
					pkg.F(t)
				}
				`,
			},
		}
		program, err := scriggo.Load(loaders, nil)
		if err != nil {
			t.Fatal(err)
		}
		err = program.Run(nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Function taking an array", func(t *testing.T) {
		loaders := scriggo.CombinedLoader{
			scriggo.Packages{
				"pkg": &scriggo.MapPackage{
					PkgName: "pkg",
					Declarations: map[string]interface{}{
						"F": func(s [3]int) {},
					},
				},
			},
			&scriggo.MapStringLoader{
				"main": `
	
				package main
	
				import "pkg"
				
				func main() {
					a := [3]int{}
					pkg.F(a)
				}
				`,
			},
		}
		program, err := scriggo.Load(loaders, nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = program.Disassemble(os.Stdout, "main")
		if err != nil {
			t.Fatal(err)
		}
		err = program.Run(nil)
		if err != nil {
			t.Fatal(err)
		}
	})
}
