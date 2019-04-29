// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"scrigo"
	"scrigo/parser"
	"scrigo/vm"
)

var packages map[string]*parser.GoPackage

func main() {

	var args = os.Args

	var useVM bool
	var asm bool

	if useVM = args[1] == "-vm"; useVM {
		args = args[1:]
		if asm = args[1] == "-S"; asm {
			args = args[1:]
		}
	}

	if len(args) != 2 {
		fmt.Printf("usage: %s filename\n", args[0])
		os.Exit(-1)
	}

	file := args[1]
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

	if useVM {
		// TODO(Gianluca): to review.
		// path := "/" + filepath.Base(absFile)
		r := parser.DirReader(filepath.Dir(absFile))
		vm.DebugTraceExecution = true
		compiler := vm.NewCompiler(r, packages)
		main, err := compiler.CompileFunction()
		if err != nil {
			fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
			os.Exit(2)
		}
		if asm {
			_, err = vm.DisassembleFunction(os.Stdout, main)
			if err != nil {
				fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		} else {
			print("(vm) ")
			_, err = vm.New().Run(main)
			if err != nil {
				fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		}
		return
	}

	switch ext {
	case ".gos":
		src, err := ioutil.ReadFile(absFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		r := bytes.NewReader(src)
		s, err := scrigo.CompileScript(r, &parser.GoPackage{})
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		_, err = scrigo.ExecuteScript(s, nil)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	case ".go":
		r := parser.DirReader(filepath.Dir(absFile))
		compiler := scrigo.NewCompiler(r, packages)
		f, err := os.Open(file)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		program, err := compiler.Compile(f)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		f.Close()
		err = scrigo.Execute(program)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
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
