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
	"strconv"

	"scrigo"
	"scrigo/internal/compiler"
	"scrigo/native"
	"scrigo/script"
	"scrigo/template"
	"scrigo/vm"
)

var packages map[string]*native.GoPackage

func main() {

	var asm = flag.Bool("S", false, "print assembly listing")
	var trace = flag.Bool("trace", false, "print an execution trace")
	var mem = flag.String("mem", "", "maximum allocable memory; set to zero for no limit")

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

	freeMemory := 0
	if *mem != "" {
		var unit = (*mem)[len(*mem)-1]
		if unit == 'K' || unit == 'M' {
			*mem = (*mem)[:len(*mem)-1]
		}
		var err error
		freeMemory, err = strconv.Atoi(*mem)
		if err != nil {
			fmt.Printf("usage: %s [-S] [--trace] [--mem=size[K|M|G]] filename\n", os.Args[0])
			os.Exit(-1)
		}
		if unit == 'K' {
			freeMemory *= 1024
		} else if unit == 'M' {
			freeMemory *= 1024 * 1024
		} else if unit == 'G' {
			freeMemory *= 1024 * 1024 * 1024
		}
	}

	var args = flag.Args()

	if len(args) != 1 {
		fmt.Printf("usage: %s [-S] [--trace] [--mem=size[K|M|G]] filename\n", os.Args[0])
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
		if *mem == "" {
			freeMemory = 16 * 1024 * 1024
		}
		// TODO(Gianluca): disassembling is currently not available for
		// scripts in interpreter. Find a solution.
		r, err := os.Open(absFile)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
			os.Exit(2)
		}
		s, err := script.Compile(r, nil, freeMemory > 0)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
			os.Exit(2)
		}
		if *asm {
			funcs, err := compiler.Disassemble(s.Fn)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
			fmt.Print(funcs["main"])
		} else {
			err = script.Execute(s, nil, freeMemory)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		}
	case ".go":
		// TODO(Gianluca): use program.Execute: vm shall not be imported by
		// interpreter. Note that disassembling and tracing support must be
		// keep.
		path := "/" + filepath.Base(absFile)
		r := compiler.DirReader(filepath.Dir(absFile))
		program, err := scrigo.Compile(path, r, packages, freeMemory > 0)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
			os.Exit(2)
		}
		if *asm {
			funcs, err := compiler.Disassemble(program.Fn)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
			fmt.Print(funcs["main"])
		} else {
			if *trace {
				err = scrigo.Execute(program, &tf, freeMemory)
			} else {
				err = scrigo.Execute(program, nil, freeMemory)
			}
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		}

	case ".html":
		if *mem == "" {
			freeMemory = 16 * 1024 * 1024
		}
		r := template.DirReader(filepath.Dir(absFile))
		path := "/" + filepath.Base(absFile)
		page, err := template.Compile(path, r, nil, template.ContextHTML, freeMemory > 0)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		err = page.Render(os.Stdout, nil, freeMemory)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	}

}
