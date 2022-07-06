// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/native"
)

//go:generate scriggo import -v -o packages.go
var packages native.Packages

var globals = native.Declarations{
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
	case ".go":
	case ".html", ".css", ".js", ".json", ".md":
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
		opts := &scriggo.BuildOptions{}
		opts.AllowGoStmt = !*disallowGoStmt
		opts.Packages = packages
		var fsys fs.FS
		if cmd == "rundir" {
			fsys = os.DirFS(flag.Arg(2))
		} else {
			data, err := io.ReadAll(src)
			if err != nil {
				panic(err)
			}
			fsys = scriggo.Files{"main.go": data}
		}
		program, err := scriggo.Build(fsys, opts)
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		if cmd != "build" {
			err = program.Run(nil)
			if err != nil {
				panic(convertRunError(err))
			}
		}
	default:
		var fsys fs.FS
		switch cmd {
		case "build", "run":
			src, err := io.ReadAll(os.Stdin)
			if err != nil {
				panic(err)
			}
			fsys = scriggo.Files{"index" + ext: src}
		case "rundir":
			fsys = os.DirFS(flag.Arg(2))
		}
		opts := scriggo.BuildOptions{
			Globals:           globals,
			Packages:          packages,
			MarkdownConverter: markdownConverter,
		}
		template, err := scriggo.BuildTemplate(fsys, "index"+ext, &opts)
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		if cmd != "build" {
			err = template.Run(os.Stdout, nil, nil)
			if err != nil {
				panic(convertRunError(err))
			}
		}
	}

	return
}

// convertRunError converts an error returned from a Run method, transforming
// a *scriggo.PanicError value to its string representation.
func convertRunError(err error) error {
	if p, ok := err.(*scriggo.PanicError); ok {
		return errors.New(p.Error())
	}
	return err
}

var mdStart = []byte("--- start Markdown ---\n")
var mdEnd = []byte("--- end Markdown ---\n")

// markdownConverter is a scriggo.Converter that it used to check that the
// markdown converter is called. To do this, markdownConverter does not
// convert but only wraps the Markdown code.
func markdownConverter(src []byte, out io.Writer) error {
	_, err := out.Write(mdStart)
	if err == nil {
		_, err = out.Write(src)
	}
	if err == nil {
		_, err = out.Write(mdEnd)
	}
	return err
}
