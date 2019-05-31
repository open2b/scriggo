package test

import (
	"bytes"
	"fmt"
	"scrigo"
	"strings"
	"testing"
)

var scriptCases = map[string]struct {
	src  string
	main *scrigo.PredefinedPackage
	init map[string]interface{}

	out string
}{
	"Don't use anything but Go builtins": {
		src: `println("hi!")`,
	},

	"Use external main definition": {
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

	// "Read variables declared in external main": {
	// 	src: `
	// 		Print("A is ", A)
	// 	`,
	// 	out: "A is 2",
	// 	main: &scrigo.PredefinedPackage{
	// 		Name: "main",
	// 		Declarations: map[string]interface{}{
	// 			"A": (*int)(nil),
	// 		},
	// 	},
	// },

	// "Read and write variables declared in external main": {
	// 	src: `
	// 		Print("default: ", A, ", ")
	// 		A = 20
	// 		Print("new: ", A)
	// 	`,
	// 	out: "default: 0, new: 20",
	// 	main: &scrigo.PredefinedPackage{
	// 		Name: "main",
	// 		Declarations: map[string]interface{}{
	// 			"A": (*int)(nil),
	// 		},
	// 	},
	// },

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
			if cas.main == nil {
				cas.main = &scrigo.PredefinedPackage{}
				cas.main.Declarations = make(map[string]interface{})
			}
			cas.main.Declarations["Print"] = func(args ...interface{}) {
				for _, a := range args {
					scriptStdout.WriteString(fmt.Sprint(a))
				}
			}
			packages := []scrigo.PkgImporter{
				map[string]*scrigo.PredefinedPackage{
					"main": cas.main,
				},
			}
			script, err := scrigo.LoadScript(bytes.NewReader([]byte(cas.src)), packages, scrigo.Option(0))
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
