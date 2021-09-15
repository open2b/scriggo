// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo_test

import (
	"fmt"
	"log"
	"os"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/native"
)

func ExampleBuild() {
	fsys := scriggo.Files{
		"main.go": []byte(`
			package main

			func main() { }
		`),
	}
	_, err := scriggo.Build(fsys, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleProgram_Run() {
	fsys := scriggo.Files{
		"main.go": []byte(`
			package main

			import "fmt"

			func main() {
				fmt.Println("Hello, I'm Scriggo!")
			}
		`),
	}
	opts := &scriggo.BuildOptions{
		Packages: native.Packages{
			"fmt": native.Package{
				Name: "fmt",
				Declarations: native.Declarations{
					"Println": fmt.Println,
				},
			},
		},
	}
	program, err := scriggo.Build(fsys, opts)
	if err != nil {
		log.Fatal(err)
	}
	err = program.Run(nil)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// Hello, I'm Scriggo!
}

func ExampleBuildTemplate() {
	fsys := scriggo.Files{
		"index.html": []byte(`{% name := "Scriggo" %}Hello, {{ name }}!`),
	}
	_, err := scriggo.BuildTemplate(fsys, "index.html", nil)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleTemplate_Run() {
	fsys := scriggo.Files{
		"index.html": []byte(`{% name := "Scriggo" %}Hello, {{ name }}!`),
	}
	template, err := scriggo.BuildTemplate(fsys, "index.html", nil)
	if err != nil {
		log.Fatal(err)
	}
	err = template.Run(os.Stdout, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// Hello, Scriggo!
}

func ExampleHTMLEscape() {
	fmt.Println(scriggo.HTMLEscape("Rock & Roll!"))
	// Output:
	// Rock &amp; Roll!
}

func ExampleBuildError() {
	fsys := scriggo.Files{
		"index.html": []byte(`{{ 42 + "hello" }}`),
	}
	_, err := scriggo.BuildTemplate(fsys, "index.html", nil)
	if err != nil {
		fmt.Printf("Error has type %T\n", err)
		fmt.Printf("Error message is: %s\n", err.(*scriggo.BuildError).Message())
		fmt.Printf("Error path is: %s\n", err.(*scriggo.BuildError).Path())
	}
	// Output:
	// Error has type *scriggo.BuildError
	// Error message is: invalid operation: 42 + "hello" (mismatched types int and string)
	// Error path is: index.html
}
