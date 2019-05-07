// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"scrigo"
	"scrigo/parser"
	"scrigo/vm"
)

var packages map[string]*parser.GoPackage

func main() {

	var asm = flag.Bool("S", false, "print assembly listing")
	var trace = flag.Bool("trace", false, "print an execution trace")

	flag.Parse()

	vm.DebugTraceExecution = *trace

	var args = flag.Args()

	if len(args) != 1 {
		fmt.Printf("usage: %s [-S] [--trace] filename\n", os.Args[0])
		os.Exit(-1)
	}

	file := args[0]
	ext := filepath.Ext(file)
	if ext != ".go" && ext != ".gos" && ext != ".html" {
		fmt.Printf("%s: extension must be \".go\" for main packages, \".gos\" for scripts and \".html\" for template pages\n", file)
		os.Exit(-1)
	}

	absFile, err := filepath.Abs(file)
	if err != nil {
		fmt.Printf("%s: %s\n", file, err)
		os.Exit(-1)
	}

	switch ext {
	case ".gos":
		path := "/" + filepath.Base(absFile)
		r := parser.DirReader(filepath.Dir(absFile))
		compiler := vm.NewCompiler(r, packages)
		main, err := compiler.CompileScript(path)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
			os.Exit(2)
		}
		if *asm {
			_, err = vm.DisassembleFunction(os.Stdout, main)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		} else {
			_, err = vm.New().Run(main)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		}
	case ".go":
		path := "/" + filepath.Base(absFile)
		r := parser.DirReader(filepath.Dir(absFile))
		compiler := vm.NewCompiler(r, packages)
		main, err := compiler.CompilePackage(path)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
			os.Exit(2)
		}
		if *asm {
			_, err = vm.DisassembleFunction(os.Stdout, main)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		} else {
			_, err = vm.New().Run(main)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		}
	case ".html":
		r := parser.DirReader(filepath.Dir(absFile))
		template := scrigo.NewTemplate(r)
		path := "/" + filepath.Base(absFile)
		page, err := template.Compile(path, nil, scrigo.ContextHTML)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		err = scrigo.Render(os.Stdout, page, nil)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	}

}
