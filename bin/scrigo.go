// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"scrigo"
	"scrigo/internal/compiler"
	"scrigo/template"
	"scrigo/vm"
)

const usage = "usage: %s [-S] [-mem 250K] [-time 50ms] [-trace] filename\n"

var packages map[string]*scrigo.PredefinedPackage

func main() {

	var asm = flag.Bool("S", false, "print assembly listing")
	var timeout = flag.String("time", "", "`limit` the execution time; zero is no limit")
	var mem = flag.String("mem", "", "`limit` the allocable memory; zero is no limit")
	var trace = flag.Bool("trace", false, "print an execution trace")

	flag.Parse()

	var loadOptions scrigo.Option
	var runOptions scrigo.RunOptions

	if *timeout != "" {
		d, err := time.ParseDuration(*timeout)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, usage, os.Args[0])
			flag.PrintDefaults()
			os.Exit(-1)
		}
		if d != 0 {
			var cancel context.CancelFunc
			runOptions.Context, cancel = context.WithTimeout(context.Background(), d)
			defer cancel()
		}
	}

	if *mem != "" {
		loadOptions = scrigo.LimitMemorySize
		var unit = (*mem)[len(*mem)-1]
		if unit == 'K' || unit == 'M' {
			*mem = (*mem)[:len(*mem)-1]
		}
		var err error
		runOptions.MaxMemorySize, err = strconv.Atoi(*mem)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, usage, os.Args[0])
			flag.PrintDefaults()
			os.Exit(-1)
		}
		if unit == 'K' {
			runOptions.MaxMemorySize *= 1024
		} else if unit == 'M' {
			runOptions.MaxMemorySize *= 1024 * 1024
		} else if unit == 'G' {
			runOptions.MaxMemorySize *= 1024 * 1024 * 1024
		}
	}

	if *trace {
		runOptions.TraceFunc = func(fn *vm.Function, pc uint32, regs vm.Registers) {
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
		_, _ = fmt.Fprintf(os.Stderr, usage, os.Args[0])
		flag.PrintDefaults()
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
		// TODO(Gianluca): disassembling is currently not available for
		// scripts in interpreter. Find a solution.
		r, err := os.Open(absFile)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
			os.Exit(2)
		}
		script, err := scrigo.LoadScript(r, nil, loadOptions)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
			os.Exit(2)
		}
		if *asm {
			_, err := script.Disassemble(os.Stdout)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		} else {
			err = script.Run(nil, runOptions)
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
		program, err := scrigo.Load(path, r, packages, loadOptions)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
			os.Exit(2)
		}
		if *asm {
			_, err := program.Disassemble(os.Stdout, "main")
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		} else {
			err = program.Run(runOptions)
			if err != nil {
				if err == context.DeadlineExceeded {
					err = errors.New("process took too long")
				}
				_, _ = fmt.Fprintf(os.Stderr, "scrigo: %s\n", err)
				os.Exit(2)
			}
		}
	case ".html":
		r := template.DirReader(filepath.Dir(absFile))
		path := "/" + filepath.Base(absFile)
		page, err := template.Compile(path, r, nil, template.ContextHTML, false)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		err = page.Render(os.Stdout, nil, 0)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	}

}
