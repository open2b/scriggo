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
	"scrigo/compiler"
	"scrigo/vm"
)

var packages map[string]*compiler.GoPackage

func main() {

	var asm = flag.Bool("S", false, "print assembly listing")
	var trace = flag.Bool("trace", false, "print an execution trace")

	flag.Parse()

	var tf vm.TraceFunc
	if *trace {
		tf = func(fn *vm.ScrigoFunction, pc uint32, regs vm.Registers) {
			funcName := fn.Name
			if funcName != "" {
				funcName += ":"
			}
			_, _ = fmt.Fprintf(os.Stderr, "i%v f%v\t%s\t", regs.Int, regs.Float, funcName)
			_, _ = compiler.DisassembleInstruction(os.Stderr, fn, pc)
			println()
		}
	}

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
		r := compiler.DirReader(filepath.Dir(absFile))
		comp := compiler.NewCompiler(r, packages)
		main, err := comp.CompileScript(path)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
			os.Exit(2)
		}
		if *asm {
			_, err = compiler.DisassembleFunction(os.Stdout, main)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		} else {
			v := vm.New()
			if *trace {
				v.SetTraceFunc(tf)
			}
			_, err = v.Run(main)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		}
	case ".go":
		path := "/" + filepath.Base(absFile)
		r := compiler.DirReader(filepath.Dir(absFile))
		comp := compiler.NewCompiler(r, packages)
		main, err := comp.CompilePackage(path)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
			os.Exit(2)
		}
		if *asm {
			funcs, err := compiler.Disassemble(main)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
			fmt.Print(funcs["main"])
		} else {
			v := vm.New()
			if *trace {
				v.SetTraceFunc(tf)
			}
			_, err = v.Run(main)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		}
	case ".html":
		r := compiler.DirReader(filepath.Dir(absFile))
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
