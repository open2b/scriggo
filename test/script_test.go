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

	expectedOutput string
}{
	"Don't use anything but Go builtins": {
		src: `println("hi!")`,
	},

	"Use external main definition": {
		src:            `Print("hi!")`,
		expectedOutput: "hi!",
	},

	"Variable declarations": {
		src: `
			A := 10
			B := "hey"
			Print(A, B)
		`,
		expectedOutput: "10hey",
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

// Handles script output.
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
			script, err := scrigo.LoadScript(bytes.NewReader([]byte(cas.src)), cas.main, scrigo.Option(0))
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			err = script.Run(cas.init, scrigo.RunOptions{})
			if err != nil {
				t.Fatalf("execution error: %s", err)
			}
			output := scriptStdout.String()
			scriptStdout.Reset()
			if output != cas.expectedOutput {
				t.Fatalf("expecting output %q, got %q", cas.expectedOutput, output)
			}
		})
	}
}
