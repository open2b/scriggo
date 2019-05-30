package scrigo

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
	_ = simpleMain
	cases := map[string]struct {
		src  string
		main *scrigo.PredefinedPackage
		init map[string]interface{}
	}{
		"A script which doesn't use anything but Go builtins": {
			src: `println("hi!")`,
		},
		// TODO(Gianluca):
		// "A script using external main definition": {
		// 	src:  `Println("hi!")`,
		// 	main: simpleMain,
		// },
	}
	for name, cas := range cases {
		t.Run(name, func(t *testing.T) {
			script, err := scrigo.LoadScript(bytes.NewReader([]byte(cas.src)), nil, scrigo.Option(0))
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			err = script.Run(nil, scrigo.RunOptions{})
			if err != nil {
				t.Fatalf("execution error: %s", err)
			}
		})
	}
}
