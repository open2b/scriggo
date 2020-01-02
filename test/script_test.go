// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"scriggo"
)

var scriptCases = map[string]struct {
	src  string
	pkgs scriggo.Packages
	init map[string]interface{}

	out string
}{
	"Empty script": {
		src: ``,
	},

	"Don't use anything but Scriggo builtins": {
		src: `println("hi!")`,
	},

	"Use a function from external main definition": {
		src: `Print("hi!")`,
		out: "hi!",
	},

	"Variable declarations": {
		src: `
			A := 10
			B := "hey"
			Print(A, B)
		`,
		out: "10hey",
	},

	"Importing a package using the 'import' statement": {
		src: `
			import "pkg"
			pkg.F()
		`,
		pkgs: scriggo.Packages{
			"pkg": &scriggo.MapPackage{
				PkgName: "pkg",
				Declarations: map[string]interface{}{
					"F": func() {
						scriptStdout.WriteString("pkg.F called!")
					},
				},
			},
		},
		out: "pkg.F called!",
	},

	"Read variables declared in predefined package main": {
		src: `
			Print("A is ", A)
		`,
		out: "A is 0",
		pkgs: scriggo.Packages{
			"main": &scriggo.MapPackage{
				PkgName: "main",
				Declarations: map[string]interface{}{
					"A": (*int)(nil),
				},
			},
		},
	},

	"Read and write variables declared in external main": {
		src: `
			Print("default: ", A, ", ")
			A = 20
			Print("new: ", A)
		`,
		out: "default: 0, new: 20",
		pkgs: scriggo.Packages{
			"main": &scriggo.MapPackage{
				PkgName: "main",
				Declarations: map[string]interface{}{
					"A": (*int)(nil),
				},
			},
		},
	},

	// "Overwriting predefined main variable using init": {
	// 	src: `
	// 		Print(A)
	// 	`,
	// 	pkgs: scriggo.Packages{
	// 		"main": {
	// 			Name: "main",
	// 			Declarations: map[string]interface{}{
	// 				"A": (*int)(nil),
	// 			},
	// 		},
	// 	},
	// 	init: map[string]interface{}{
	// 		"A": reflect.ValueOf(&scriptTestA).Elem(),
	// 	},
	// 	out: "5",
	// },

	"Usage test: using a script to perfom a sum of numbers": {
		src: `
			for i := 0; i < 10; i++ {
				Sum += i
			}
			Print(Sum)
		`,
		pkgs: scriggo.Packages{
			"main": &scriggo.MapPackage{
				PkgName: "main",
				Declarations: map[string]interface{}{
					"Sum": (*int)(nil),
				},
			},
		},
		out: "45",
	},

	"Auto imported package": {
		src: `
			name := strings.ToLower("HELLO")
			Print(name)
			Print(math.MaxInt8)
			v := math.MaxInt8 * 2
			Print(v)
		`,
		pkgs: scriggo.Packages{
			"main": &scriggo.MapPackage{
				PkgName: "main",
				Declarations: map[string]interface{}{
					"strings": &scriggo.MapPackage{
						PkgName: "strings",
						Declarations: map[string]interface{}{
							"ToLower": strings.ToLower,
						},
					},
					"math": &scriggo.MapPackage{
						PkgName: "math",
						Declarations: map[string]interface{}{
							"MaxInt8": math.MaxInt8,
						},
					},
				},
			},
		},
		out: "hello127254",
	},

	// https://github.com/open2b/scriggo/issues/317
	// "Function definitions": {
	// 	src: `
	// 	func F() {
	// 		Print("i'm f")
	// 	}
	// 	F()
	// 	`,
	// },
}

// Holds scripts output.
var scriptStdout strings.Builder

func TestScript(t *testing.T) {
	for name, cas := range scriptCases {
		t.Run(name, func(t *testing.T) {
			if cas.pkgs == nil {
				cas.pkgs = scriggo.Packages{}
			}
			if _, ok := cas.pkgs["main"]; !ok {
				cas.pkgs["main"] = &scriggo.MapPackage{Declarations: make(map[string]interface{})}
			}
			cas.pkgs["main"].(*scriggo.MapPackage).Declarations["Print"] = func(args ...interface{}) {
				for _, a := range args {
					scriptStdout.WriteString(fmt.Sprint(a))
				}
			}
			loadOpts := &scriggo.LoadOptions{}
			loadOpts.Unspec.PackageLess = true
			script, err := scriggo.Load(strings.NewReader(cas.src), cas.pkgs, loadOpts)
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			runOpts := &scriggo.RunOptions{}
			runOpts.Unspec.Builtins = cas.init
			err = script.Run(runOpts)
			if err != nil {
				t.Fatalf("execution error: %s", err)
			}
			out := scriptStdout.String()
			scriptStdout.Reset()
			if out != cas.out {
				t.Fatalf("expecting output %q, got %q", cas.out, out)
			}
		})
	}
}

func TestScriptSum(t *testing.T) {
	src := `for i := 0; i < 10; i++ { Sum += i }`
	Sum := 0
	pkgs := scriggo.Packages{
		"main": &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"Sum": (*int)(nil),
			},
		},
	}
	init := map[string]interface{}{"Sum": &Sum}
	loadOpts := &scriggo.LoadOptions{}
	loadOpts.Unspec.PackageLess = true
	script, err := scriggo.Load(strings.NewReader(src), pkgs, loadOpts)
	if err != nil {
		t.Fatalf("unable to load script: %s", err)
	}
	runOpts := &scriggo.RunOptions{}
	runOpts.Unspec.Builtins = init
	err = script.Run(runOpts)
	if err != nil {
		t.Fatalf("run: %s", err)
	}
	if Sum != 45 {
		t.Fatalf("Sum should be %d, got %d", 45, Sum)
	}
}

func TestScriptChainMessages(t *testing.T) {
	src1 := `Message = Message + "script1,"`
	src2 := `Message = Message + "script2"`
	Message := "external,"
	pkgs := scriggo.Packages{
		"main": &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"Message": (*string)(nil),
			},
		},
	}
	loadOpts := &scriggo.LoadOptions{}
	loadOpts.Unspec.PackageLess = true
	init := map[string]interface{}{"Message": &Message}
	script1, err := scriggo.Load(strings.NewReader(src1), pkgs, loadOpts)
	if err != nil {
		t.Fatalf("unable to load script 1: %s", err)
	}
	script2, err := scriggo.Load(strings.NewReader(src2), pkgs, loadOpts)
	if err != nil {
		t.Fatalf("unable to load script 2: %s", err)
	}
	runOpts := &scriggo.RunOptions{}
	runOpts.Unspec.Builtins = init
	err = script1.Run(runOpts)
	if err != nil {
		t.Fatalf("run: %s", err)
	}
	if Message != "external,script1," {
		t.Fatalf("Message should be %q, got %q", "external,script1,", Message)
	}
	err = script2.Run(runOpts)
	if err != nil {
		t.Fatalf("run: %s", err)
	}
	if Message != "external,script1,script2" {
		t.Fatalf("Message should be %q, got %q", "external,script1,script2", Message)
	}
}
