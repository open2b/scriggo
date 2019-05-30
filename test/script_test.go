package test

import (
	"bytes"
	"fmt"
	"scrigo"
	"testing"
)

func TestScript(t *testing.T) {
	var simpleMain = &scrigo.PredefinedPackage{
		Name: "main",
		Declarations: map[string]interface{}{
			"Println": fmt.Println,
			"Print":   fmt.Print,
		},
	}
	cases := map[string]struct {
		src  string
		main *scrigo.PredefinedPackage
		init map[string]interface{}
	}{
		"Don't use anything but Go builtins": {
			src: `println("hi!")`,
		},
		"Use external main definition": {
			src:  `Println("hi!")`,
			main: simpleMain,
		},
		"Function definitions": {
			src: `
			func F() {
				println("i'm f")
			}
			F()
			`,
		},
	}
	for name, cas := range cases {
		t.Run(name, func(t *testing.T) {
			script, err := scrigo.LoadScript(bytes.NewReader([]byte(cas.src)), cas.main, scrigo.Option(0))
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			err = script.Run(cas.init, scrigo.RunOptions{})
			if err != nil {
				t.Fatalf("execution error: %s", err)
			}
		})
	}
}
