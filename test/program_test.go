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
	"scriggo/runtime"
)

var exprTests = map[string]interface{}{

	// Numbers.
	`3`:    int64(3),
	`-3`:   int64(-3),
	`3000`: int64(3000),
	`4.5`:  float64(4.5),
	`-4.5`: float64(-4.5),

	// Strings.
	`"abc"`: "abc",

	// Composite literals - arrays.
	`[4]int{1,2,3,4}`:     [4]int{1, 2, 3, 4},
	`[...]int{1,2,3,4}`:   [4]int{1, 2, 3, 4},
	`[3]string{}`:         [3]string{"", "", ""},
	`[3]string{"a", "b"}`: [3]string{"a", "b", ""},

	// Composite literals - slices.
	`[]int{}`:                   []int{},
	`[]int{1, 2}`:               []int{1, 2},
	`[]int{0: 1, 1: 3}`:         []int{0: 1, 1: 3},
	`[]int{0: 1, 5: 3}`:         []int{0: 1, 5: 3},
	`[]string{}`:                []string{},
	`[]string{"a", "b"}`:        []string{"a", "b"},
	`[]string{"a", 5: "b"}`:     []string{"a", 5: "b"},
	`[]float64{5, 12, 0}`:       []float64{5, 12, 0},
	`[]float64{5.6, 12.3, 0.4}`: []float64{5.6, 12.3, 0.4},

	// Composite literals - maps.
	`map[int]string{}`:                                    map[int]string{},
	`map[[3]int]string{}`:                                 map[[3]int]string{},
	`map[int]string{1: "one"}`:                            map[int]string{1: "one"},
	`map[int]string{1: "one", 2: "two", 10: "ten"}`:       map[int]string{1: "one", 2: "two", 10: "ten"},
	`map[string]int{"one": 1, "two": 10 / 5, "three": 3}`: map[string]int{"one": 1, "two": 10 / 5, "three": 3},

	// Builtin 'make'.
	`make([]int, 0, 0)`:       []int{},
	`make([]int, 0, 5)`:       []int{},
	`make([]int, 3, 5)`:       []int{0, 0, 0},
	`make([]string, 1, 5)`:    []string{""},
	`make(map[string]int, 1)`: map[string]int{},
	`make([]float64, 3, 3)`:   []float64{0, 0, 0},
}

func TestVMExpressions(t *testing.T) {
	for src, expected := range exprTests {
		t.Run(src, func(t *testing.T) {
			r := scriggo.MapStringLoader{"main": "package main; func main() { a := " + src + "; _ = a }"}
			program, err := scriggo.Load(r, &scriggo.LoadOptions{LimitMemorySize: true})
			if err != nil {
				t.Errorf("test %q, compiler error: %s", src, err)
				return
			}
			var registers runtime.Registers
			tf := func(_ *runtime.Function, _ runtime.Addr, regs runtime.Registers) {
				registers = regs
			}
			err = program.Run(&scriggo.RunOptions{MaxMemorySize: 1000000, TraceFunc: tf})
			if err != nil {
				t.Errorf("test %q, execution error: %s", src, err)
				return
			}
			kind := reflect.TypeOf(expected).Kind()
			var got interface{}
			switch kind {
			case reflect.Int, reflect.Bool, reflect.Int64:
				got = registers.Int[0]
			case reflect.Float32, reflect.Float64:
				got = registers.Float[0]
			case reflect.String:
				got = registers.String[0]
			case reflect.Slice, reflect.Map, reflect.Array:
				got = registers.General[0]
			default:
				panic("bug")
			}
			if !reflect.DeepEqual(expected, got) {
				t.Errorf("test %q, expected %v (type %T), got %v (type %T)", src, expected, expected, got, got)
			}
		})
	}
}

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
