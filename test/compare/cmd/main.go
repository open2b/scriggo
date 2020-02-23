// This executable is not meant to be human friendly or to have a nice CLI
// interface. It just aims to simplicity, speed and robustness.

// In case of success the standard output contains the output of the execution
// of the program or the rendering of the template. Else, in case of error, the
// stderr contains the error.

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"scriggo"
	"scriggo/runtime"
)

//go:generate scriggo embed -v -o predefPkgs.go
var predefPkgs scriggo.Packages

type stdinLoader struct {
	file *os.File
}

func (b stdinLoader) Load(path string) (interface{}, error) {
	if path == "main" {
		return b.file, nil
	}
	return nil, nil
}

type dirLoader string

func (dl dirLoader) Load(path string) (interface{}, error) {
	if path == "main" {
		main, err := ioutil.ReadFile(filepath.Join(string(dl), "main.go"))
		if err != nil {
			panic(err)
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

func renderPanics(p *runtime.Panic) string {
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
	return msg
}

var templateMain = &scriggo.MapPackage{
	PkgName: "main",
	Declarations: map[string]interface{}{
		"MainSum": func(a, b int) int { return a + b },
	},
}

func main() {

	err := os.Setenv("SCRIGGO", "v0.0.0")
	if err != nil {
		panic(err)
	}

	var timeoutArg = flag.String("time", "", "limit the execution time; zero is no limit; the argument is parsed using function time.ParseDuration")
	var memArg = flag.String("mem", "", "limit the allocable memory; zero is no limit; suffixes [BKMG] are supported")
	var disallowGoStmt = flag.Bool("disallowGoStatement", false, "disallow the 'go' statement")

	flag.Parse()

	// TODO: this is a copy-paste from cmd/scriggo/interpreter_skel.go. When
	// these code will be implemented as a support function, call it.
	var timeout context.Context
	if *timeoutArg != "" {
		d, err := time.ParseDuration(*timeoutArg)
		if err != nil {
			panic(err)
		}
		if d != 0 {
			var cancel context.CancelFunc
			timeout, cancel = context.WithTimeout(context.Background(), d)
			defer cancel()
		}
	}

	// TODO: this is a copy-paste from cmd/scriggo/interpreter_skel.go. When
	// these code will be implemented as a support function, call it.
	var memoryLimiter runtime.MemoryLimiter
	if *memArg != "" {
		var unit = (*memArg)[len(*memArg)-1]
		if unit > 'Z' {
			unit -= 'z' - 'Z'
		}
		switch unit {
		case 'B', 'K', 'M', 'G':
			*memArg = (*memArg)[:len(*memArg)-1]
		}
		max, err := strconv.Atoi(*memArg)
		if err != nil {
			panic(err)
		}
		switch unit {
		case 'K':
			max *= 1024
		case 'M':
			max *= 1024 * 1024
		case 'G':
			max *= 1024 * 1024 * 1024
		}
		memoryLimiter = scriggo.NewSingleMemoryLimiter(max)
	}

	switch flag.Args()[0] {
	case "compile program":
		if timeout != nil {
			panic("timeout not supported when compiling a program")
		}
		loadOpts := &scriggo.LoadOptions{LimitMemorySize: memoryLimiter != nil}
		loadOpts.OutOfSpec.DisallowGoStmt = *disallowGoStmt
		_, err := scriggo.Load(os.Stdin, predefPkgs, loadOpts)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	case "compile script":
		if timeout != nil {
			panic("timeout not supported when compiling a package-less program")
		}
		loadOpts := &scriggo.LoadOptions{LimitMemorySize: memoryLimiter != nil}
		loadOpts.OutOfSpec.DisallowGoStmt = *disallowGoStmt
		loadOpts.OutOfSpec.PackageLess = true
		_, err = scriggo.Load(os.Stdin, predefPkgs, loadOpts)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	case "run program":
		loadOpts := &scriggo.LoadOptions{LimitMemorySize: memoryLimiter != nil}
		loadOpts.OutOfSpec.DisallowGoStmt = *disallowGoStmt
		program, err := scriggo.Load(os.Stdin, predefPkgs, loadOpts)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		runOpts := &scriggo.RunOptions{
			Context:       timeout,
			MemoryLimiter: memoryLimiter,
		}
		_, err = program.Run(runOpts)
		if err != nil {
			if p, ok := err.(*runtime.Panic); ok {
				panic(renderPanics(p))
			}
			panic(err)
		}
	case "run script":
		loadOpts := &scriggo.LoadOptions{LimitMemorySize: memoryLimiter != nil}
		loadOpts.OutOfSpec.DisallowGoStmt = *disallowGoStmt
		loadOpts.OutOfSpec.PackageLess = true
		script, err := scriggo.Load(os.Stdin, predefPkgs, loadOpts)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		runOpts := &scriggo.RunOptions{
			Context:       timeout,
			MemoryLimiter: memoryLimiter,
		}
		_, err = script.Run(runOpts)
		if err != nil {
			if p, ok := err.(*runtime.Panic); ok {
				panic(renderPanics(p))
			}
			panic(err)
		}
	case "run program directory":
		loadOpts := &scriggo.LoadOptions{LimitMemorySize: memoryLimiter != nil}
		loadOpts.OutOfSpec.DisallowGoStmt = *disallowGoStmt
		dirPath := flag.Args()[1]
		dl := dirLoader(dirPath)
		main, err := dl.Load("main")
		if err != nil {
			panic(err)
		}
		prog, err := scriggo.Load(main.(io.Reader), scriggo.CombinedLoader{dl, predefPkgs}, loadOpts)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		runOpts := &scriggo.RunOptions{
			Context:       timeout,
			MemoryLimiter: memoryLimiter,
		}
		_, err = prog.Run(runOpts)
		if err != nil {
			if p, ok := err.(*runtime.Panic); ok {
				panic(renderPanics(p))
			}
			panic(err)
		}
	case "render html":
		if *disallowGoStmt {
			panic("disallow Go statement not supported when rendering a html page")
		}
		src, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		r := mapReader{"/index.html": src}
		templ, err := compileTemplate(r, memoryLimiter != nil)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		err = templ.render(timeout, memoryLimiter)
		if err != nil {
			if p, ok := err.(*runtime.Panic); ok {
				panic(renderPanics(p))
			}
			panic(err)
		}
	case "render html directory":
		if *disallowGoStmt {
			panic("disallow Go statement not supported when rendering a html directory")
		}
		dirPath := flag.Args()[1]
		r := dirReader(dirPath)
		templ, err := compileTemplate(r, memoryLimiter != nil)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		err = templ.render(timeout, memoryLimiter)
		if err != nil {
			if p, ok := err.(*runtime.Panic); ok {
				panic(renderPanics(p))
			}
			panic(err)
		}
	case "compile html":
		if *disallowGoStmt {
			panic("disallow Go statement not supported when compiling a html page")
		}
		src, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		if timeout != nil {
			panic("timeout not supported when compiling a html page")
		}
		r := mapReader{"/index.html": src}
		_, err = compileTemplate(r, memoryLimiter != nil)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	default:
		panic("invalid argument: %s" + flag.Args()[0])
	}
}
