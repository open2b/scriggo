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

	"github.com/open2b/scriggo/native"
	"github.com/open2b/scriggo/scripts"
)

var scriptTests = map[string]struct {
	src     string
	pkgs    native.Packages
	init    map[string]interface{}
	globals native.Declarations

	out string
}{
	"Empty script": {
		src: ``,
	},

	"Don't use anything but builtins": {
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
		pkgs: native.Packages{
			"pkg": native.DeclarationsPackage{
				Name: "pkg",
				Declarations: native.Declarations{
					"F": func() {
						scriptsStdout.WriteString("pkg.F called!")
					},
				},
			},
		},
		out: "pkg.F called!",
	},

	"Read variables declared as global": {
		src: `
			Print("A is ", A)
		`,
		out: "A is 0",
		globals: native.Declarations{
			"A": (*int)(nil),
		},
	},

	"Read and write variables declared as globals": {
		src: `
			Print("default: ", A, ", ")
			A = 20
			Print("new: ", A)
		`,
		out: "default: 0, new: 20",
		globals: native.Declarations{
			"A": (*int)(nil),
		},
	},

	//"Overwriting predefined main variable using init": {
	//	src: `
	//		Print(A)
	//	`,
	//	pkgs: native.Packages{
	//		"main": native.DeclarationsPackage{
	//			Name: "main",
	//			Declarations: native.Declarations{
	//				"A": (*int)(nil),
	//			},
	//		},
	//	},
	//	init: native.Declarations{
	//		"A": reflect.ValueOf(&scriptTestA).Elem(),
	//	},
	//	out: "5",
	//},

	"Usage test: using a script to perform a sum of numbers": {
		src: `
			for i := 0; i < 10; i++ {
				Sum += i
			}
			Print(Sum)
		`,
		globals: native.Declarations{
			"Sum": (*int)(nil),
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
		globals: native.Declarations{

			"strings": native.DeclarationsPackage{
				Name: "strings",
				Declarations: native.Declarations{
					"ToLower": strings.ToLower,
				},
			},
			"math": native.DeclarationsPackage{
				Name: "math",
				Declarations: native.Declarations{
					"MaxInt8": math.MaxInt8,
				},
			},
		},
		out: "hello127254",
	},

	// https://github.com/open2b/scriggo/issues/317
	"Function definitions": {
		src: `
		func F() {
			Print("i'm f")
		}
		F()
		`,
		out: "i'm f",
	},

	"Function literal body refers to a closure variable": {
		src: `
		func F() {
			var V = 42
			func() {
				Print(V);
			}()
		}
		F()
		`,
		out: "42",
	},

	// https://github.com/open2b/scriggo/issues/659
	"Accessing a global variable from a function literal": {
		src: `
		func F() {
			_ = V
		}
		F()
		`,
		globals: native.Declarations{
			"V": (*int)(nil),
		},
	},
}

// Holds output of scriptTests.
var scriptsStdout strings.Builder

func TestScripts(t *testing.T) {
	for name, cas := range scriptTests {
		t.Run(name, func(t *testing.T) {
			globals := cas.globals
			if globals == nil {
				globals = native.Declarations{}
			}
			globals["Print"] = func(args ...interface{}) {
				for _, a := range args {
					scriptsStdout.WriteString(fmt.Sprint(a))
				}
			}
			options := &scripts.BuildOptions{
				Globals:  globals,
				Packages: cas.pkgs,
			}
			script, err := scripts.Build(strings.NewReader(cas.src), options)
			if err != nil {
				t.Fatalf("build error: %s", err)
			}
			err = script.Run(cas.init, nil)
			if err != nil {
				t.Fatalf("run error: %s", err)
			}
			out := scriptsStdout.String()
			scriptsStdout.Reset()
			if out != cas.out {
				t.Fatalf("expecting output %q, got %q", cas.out, out)
			}
		})
	}
}

func TestScriptSum(t *testing.T) {
	src := `for i := 0; i < 10; i++ { Sum += i }`
	Sum := 0
	init := map[string]interface{}{"Sum": &Sum}
	options := &scripts.BuildOptions{
		Globals: native.Declarations{
			"Sum": (*int)(nil),
		},
	}
	script, err := scripts.Build(strings.NewReader(src), options)
	if err != nil {
		t.Fatalf("unable to load script: %s", err)
	}
	err = script.Run(init, nil)
	if err != nil {
		t.Fatalf("run: %s", err)
	}
	if Sum != 45 {
		t.Fatalf("Sum should be %d, got %d", 45, Sum)
	}
}

func TestScriptsChainMessages(t *testing.T) {
	src1 := `Message = Message + "script1,"`
	src2 := `Message = Message + "script2"`
	Message := "external,"
	options := &scripts.BuildOptions{
		Globals: native.Declarations{
			"Message": (*string)(nil),
		},
	}
	init := map[string]interface{}{"Message": &Message}
	script1, err := scripts.Build(strings.NewReader(src1), options)
	if err != nil {
		t.Fatalf("unable to load script 1: %s", err)
	}
	script2, err := scripts.Build(strings.NewReader(src2), options)
	if err != nil {
		t.Fatalf("unable to load script 2: %s", err)
	}
	err = script1.Run(init, nil)
	if err != nil {
		t.Fatalf("run: %s", err)
	}
	if Message != "external,script1," {
		t.Fatalf("Message should be %q, got %q", "external,script1,", Message)
	}
	err = script2.Run(init, nil)
	if err != nil {
		t.Fatalf("run: %s", err)
	}
	if Message != "external,script1,script2" {
		t.Fatalf("Message should be %q, got %q", "external,script1,script2", Message)
	}
}
