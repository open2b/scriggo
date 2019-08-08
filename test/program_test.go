// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"reflect"
	"testing"

	"scriggo"
	"scriggo/vm"
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
	`[4]int{1,2,3,4}`:     []int{1, 2, 3, 4},      // Internally, arrays are stored as slices.
	`[...]int{1,2,3,4}`:   []int{1, 2, 3, 4},      // Internally, arrays are stored as slices.
	`[3]string{}`:         []string{"", "", ""},   // Internally, arrays are stored as slices.
	`[3]string{"a", "b"}`: []string{"a", "b", ""}, // Internally, arrays are stored as slices.

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
			program, err := scriggo.LoadProgram(r, &scriggo.LoadOptions{LimitMemorySize: true})
			if err != nil {
				t.Errorf("test %q, compiler error: %s", src, err)
				return
			}
			var registers vm.Registers
			tf := func(_ *vm.Function, _ uint32, regs vm.Registers) {
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
