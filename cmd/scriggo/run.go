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
)

const usage = "usage: %s [-S] filename\n"

var packages scriggo.Packages
var Main *scriggo.Package

func renderPanics(p *runtime.Panic) string {
	var msg string
	for ; p != nil; p = p.Next() {
		msg = "\n" + msg
		if p.Recovered() {
			msg = " [recovered]" + msg
		}
		msg = p.String() + msg
		if p.Next() != nil {
			msg = "\tpanic: " + msg
		}
	}
	return msg
}

func run() {

	var asm = flag.Bool("S", false, "print assembly listing")

	flag.Parse()

	var loadOptions = &scriggo.BuildOptions{}

	var args = flag.Args()

	if len(args) != 1 {
		_, _ = fmt.Fprintf(os.Stderr, usage, os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	file := args[0]
	ext := filepath.Ext(file)
	if ext != ".go" {
		fmt.Printf("%s: extension must be \".go\"\n", file)
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
	program, err := scriggo.Build(bytes.NewReader(main), scriggo.Loaders(packages), loadOptions)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "scriggo: %s\n", err)
		os.Exit(2)
	}
	if *asm {
		_, err := program.Disassemble(os.Stdout, "main")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scriggo: %s\n", err)
			os.Exit(2)
		}
	} else {
		code, err := program.Run(nil)
		if err != nil {
			if p, ok := err.(*runtime.Panic); ok {
				panic(renderPanics(p))
			}
			if err == context.DeadlineExceeded {
				err = errors.New("process took too long")
			}
			_, _ = fmt.Fprintf(os.Stderr, "scriggo: %s\n", err)
			os.Exit(2)
		}
		os.Exit(code)
	}
	os.Exit(0)
}
