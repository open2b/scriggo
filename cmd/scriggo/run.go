// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/runtime"
	"github.com/open2b/scriggo/scripts"
)

const usage = "usage: %s [-S] filename\n"

var packages scriggo.Packages

func run() {

	var asm = flag.Bool("S", false, "print assembly listing")

	flag.Parse()

	var args = flag.Args()

	if len(args) != 1 {
		_, _ = fmt.Fprintf(os.Stderr, usage, os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	file := args[0]
	ext := filepath.Ext(file)
	if ext != ".go" && ext != ".ggo" {
		fmt.Printf("%s: extension must be \".go\" for programs and \".ggo\" for scripts\n", file)
		os.Exit(1)
	}

	absFile, err := filepath.Abs(file)
	if err != nil {
		fmt.Printf("%s: %s\n", file, err)
		os.Exit(1)
	}

	main, err := ioutil.ReadFile(absFile)
	if err != nil {
		panic(err)
	}

	if ext == ".go" {

		fsys := scriggo.NewFileFS(filepath.Base(file), main)
		program, err := scriggo.Build(fsys, &scriggo.BuildOptions{Packages: packages})
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scriggo: %s\n", err)
			os.Exit(2)
		}
		if *asm {
			asm, _ := program.Disassemble("main")
			_, err := os.Stdout.Write(asm)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scriggo: %s\n", err)
				os.Exit(2)
			}
		} else {
			code, err := program.Run(nil)
			if err != nil {
				if p, ok := err.(*runtime.Panic); ok {
					panic(p)
				}
				if err == context.DeadlineExceeded {
					err = errors.New("process took too long")
				}
				_, _ = fmt.Fprintf(os.Stderr, "scriggo: %s\n", err)
				os.Exit(2)
			}
			os.Exit(code)
		}

	} else {

		script, err := scripts.Build(bytes.NewReader(main), &scripts.BuildOptions{Packages: packages})
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scriggo: %s\n", err)
			os.Exit(2)
		}
		if *asm {
			asm := script.Disassemble()
			_, err := os.Stdout.Write(asm)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scriggo: %s\n", err)
				os.Exit(2)
			}
		} else {
			code, err := script.Run(nil, nil)
			if err != nil {
				if p, ok := err.(*runtime.Panic); ok {
					panic(p)
				}
				if err == context.DeadlineExceeded {
					err = errors.New("process took too long")
				}
				_, _ = fmt.Fprintf(os.Stderr, "scriggo: %s\n", err)
				os.Exit(2)
			}
			os.Exit(code)
		}

	}

	os.Exit(0)
}
