// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/runtime"
	"github.com/open2b/scriggo/scripts"
	"github.com/open2b/scriggo/templates"
)

//go:generate scriggo embed -v -o predefPkgs.go
var predefPkgs scriggo.Packages

var globals = templates.Declarations{
	"MainSum": func(a, b int) int { return a + b },
}

// In case of success the standard output contains the output of the execution
// of the program or the rendering of the template. Else, in case of error, the
// stderr contains the error.

func main() {

	err := os.Setenv("SCRIGGO", "v0.0.0")
	if err != nil {
		panic(err)
	}

	var disallowGoStmt = flag.Bool("disallowGoStatement", false, "disallow the 'go' statement")

	flag.Parse()

	// Read and validate command and extension arguments.
	switch flag.NArg() {
	case 0:
		_, _ = fmt.Fprint(os.Stderr, "missing 'cmd' argument")
		os.Exit(1)
	case 1:
		_, _ = fmt.Fprint(os.Stderr, "missing 'ext' argument")
		os.Exit(1)
	}
	cmd := flag.Arg(0)
	switch cmd {
	case "build", "run", "rundir":
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown command %s", cmd)
		os.Exit(1)
	}
	ext := flag.Arg(1)
	switch ext {
	case ".go", ".script":
	case ".html", ".css", ".js", ".json":
		if *disallowGoStmt {
			_, _ = fmt.Fprint(os.Stderr, "disallow Go statement not supported for templates")
			os.Exit(1)
		}
	default:
		_, _ = fmt.Fprintf(os.Stderr, "invalid extension %s", cmd)
		os.Exit(1)
	}

	// Execute the command.
	switch ext {
	case ".go":
		var src io.Reader = os.Stdin
		var packages scriggo.PackageLoader = predefPkgs
		if cmd == "rundir" {
			dir := dirLoader(flag.Args()[2])
			main, err := dir.Load("main")
			if err != nil {
				panic(err)
			}
			src = main.(io.Reader)
			packages = scriggo.CombinedLoader{dir, packages}
		}
		loadOpts := &scriggo.LoadOptions{}
		loadOpts.DisallowGoStmt = *disallowGoStmt
		program, err := scriggo.Load(src, packages, loadOpts)
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		if cmd != "build" {
			_, err = program.Run(nil)
			if err != nil {
				panic(convertRunError(err))
			}
		}
	case ".script":
		loadOpts := &scripts.LoadOptions{}
		loadOpts.DisallowGoStmt = *disallowGoStmt
		script, err := scripts.Load(os.Stdin, predefPkgs, loadOpts)
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		if cmd == "run" {
			_, err = script.Run(nil, nil)
			if err != nil {
				panic(convertRunError(err))
			}
		}
	default:
		var r templates.FileReader
		switch cmd {
		case "build", "run":
			src, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				panic(err)
			}
			r = templates.MapReader{"/index" + ext: src}
		case "rundir":
			r = templates.DirReader(flag.Args()[2])
		}
		opts := templates.LoadOptions{
			Globals:  globals,
			Packages: predefPkgs,
		}
		template, err := templates.Load("/index"+ext, r, &opts)
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		if cmd != "build" {
			err = template.Render(os.Stdout, nil, nil)
			if err != nil {
				panic(convertRunError(err))
			}
		}
	}

	return
}

// convertRunError converts an error returned from a Run or Render method of
// the VM, transforming a *runtime.Panic value to its string representation.
func convertRunError(err error) error {
	if p, ok := err.(*runtime.Panic); ok {
		var msg string
		for p != nil {
			msg = "\n" + msg
			if p.Recovered() {
				msg = " [recovered]" + msg
			}
			msg = p.String() + msg
			if p.Next() != nil {
				msg = "\tpanic: " + msg
			}
			p = p.Next()
		}
		return errors.New(msg)
	}
	return err
}

// dirLoader implements the scriggo.PackageLoader interface.
type dirLoader string

func (dl dirLoader) Load(path string) (interface{}, error) {
	if path == "main" {
		main, err := ioutil.ReadFile(filepath.Join(string(dl), "main.go"))
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(main), nil
	}
	data, err := ioutil.ReadFile(filepath.Join(string(dl), path+".go"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return bytes.NewReader(data), nil
}
