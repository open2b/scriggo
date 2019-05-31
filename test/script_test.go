package test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"scrigo"
)

var scriptCases = map[string]struct {
	src  string
	pkgs map[string]*scrigo.PredefinedPackage
	init map[string]interface{}

	out string
}{
	"Don't use anything but Scrigo builtins": {
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
		pkgs: map[string]*scrigo.PredefinedPackage{
			"pkg": &scrigo.PredefinedPackage{
				Name: "pkg",
				Declarations: map[string]interface{}{
					"F": func() {
						scriptStdout.WriteString("pkg.F called!")
					},
				},
			},
		},
		out: "pkg.F called!",
	},

	"Read variables declared in predeclared package main": {
		src: `
			Print("A is ", A)
		`,
		out: "A is 0",
		pkgs: map[string]*scrigo.PredefinedPackage{
			"main": &scrigo.PredefinedPackage{
				Name: "main",
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
		pkgs: map[string]*scrigo.PredefinedPackage{
			"main": &scrigo.PredefinedPackage{
				Name: "main",
				Declarations: map[string]interface{}{
					"A": (*int)(nil),
				},
			},
		},
	},

	"Overwriting predeclared main variable using init": {
		src: `
			Print(A)
		`,
		pkgs: map[string]*scrigo.PredefinedPackage{
			"main": &scrigo.PredefinedPackage{
				Name: "main",
				Declarations: map[string]interface{}{
					"A": (*int)(nil),
				},
			},
		},
		init: map[string]interface{}{
			"A": &[]int{5}[0],
		},
		out: "5",
	},

	"Usage test: using a script to perfom a sum of numbers": {
		src: `
			for i := 0; i < 10; i++ {
				Sum += i
			}
			Print(Sum)
		`,
		pkgs: map[string]*scrigo.PredefinedPackage{
			"main": &scrigo.PredefinedPackage{
				Name: "main",
				Declarations: map[string]interface{}{
					"Sum": (*int)(nil),
				},
			},
		},
		out: "45",
	},

	// TODO(Gianluca): panics.
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
				cas.pkgs = map[string]*scrigo.PredefinedPackage{}
			}
			if _, ok := cas.pkgs["main"]; !ok {
				cas.pkgs["main"] = &scrigo.PredefinedPackage{}
				cas.pkgs["main"].Declarations = make(map[string]interface{})
			}
			cas.pkgs["main"].Declarations["Print"] = func(args ...interface{}) {
				for _, a := range args {
					scriptStdout.WriteString(fmt.Sprint(a))
				}
			}
			script, err := scrigo.LoadScript(bytes.NewReader([]byte(cas.src)), []scrigo.PkgImporter{cas.pkgs}, scrigo.Option(0))
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			err = script.Run(cas.init, scrigo.RunOptions{})
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
	pkgs := []scrigo.PkgImporter{
		map[string]*scrigo.PredefinedPackage{
			"main": &scrigo.PredefinedPackage{
				Name: "main",
				Declarations: map[string]interface{}{
					"Sum": (*int)(nil),
				},
			},
		},
	}
	init := map[string]interface{}{"Sum": &Sum}
	script, err := scrigo.LoadScript(bytes.NewReader([]byte(src)), pkgs, scrigo.Option(0))
	if err != nil {
		t.Fatalf("unable to load script: %s", err)
	}
	err = script.Run(init, scrigo.RunOptions{})
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
	pkgs := []scrigo.PkgImporter{
		map[string]*scrigo.PredefinedPackage{
			"main": &scrigo.PredefinedPackage{
				Name: "main",
				Declarations: map[string]interface{}{
					"Message": (*string)(nil),
				},
			},
		},
	}
	init := map[string]interface{}{"Message": &Message}
	script1, err := scrigo.LoadScript(bytes.NewReader([]byte(src1)), pkgs, scrigo.Option(0))
	if err != nil {
		t.Fatalf("unable to load script 1: %s", err)
	}
	script2, err := scrigo.LoadScript(bytes.NewReader([]byte(src2)), pkgs, scrigo.Option(0))
	if err != nil {
		t.Fatalf("unable to load script 2: %s", err)
	}
	err = script1.Run(init, scrigo.RunOptions{})
	if err != nil {
		t.Fatalf("run: %s", err)
	}
	if Message != "external,script1," {
		t.Fatalf("Message should be %q, got %q", "external,script1,", Message)
	}
	err = script2.Run(init, scrigo.RunOptions{})
	if err != nil {
		t.Fatalf("run: %s", err)
	}
	if Message != "external,script1,script2" {
		t.Fatalf("Message should be %q, got %q", "external,script1,script2", Message)
	}
}
