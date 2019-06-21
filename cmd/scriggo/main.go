// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func printErrorAndQuit(err interface{}) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func main() {
	flag.Usage = func() {
		f := fmt.Fprintf
		f(os.Stderr, "Scriggo is a tool for managing Scriggo interpreters and loaders\n")
		f(os.Stderr, "\n")
		f(os.Stderr, "Usage:\n")
		f(os.Stderr, "\n")
		f(os.Stderr, "\tscriggo <command> [arguments]\n")
		f(os.Stderr, "\n")
		f(os.Stderr, "The commands are:\n")
		f(os.Stderr, "\n")
		f(os.Stderr, "\tgen		generate an interpreter or a loader\n")
		f(os.Stderr, "\n")
		f(os.Stderr, "Use \"scriggo help <command>\" for more information about a command.`)\n")
		flag.PrintDefaults()
	}
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}
	cmd := os.Args[1]
	os.Args = append(os.Args[:1], os.Args[2:]...)
	switch cmd {
	case "gen":
		scriggoGen()
	case "help":
		flag.Usage()
		os.Exit(0)
	case "version":
		fmt.Printf("Scriggo module version:  1.0.5\n")
		fmt.Printf("Scriggo tool version:    1.0.4\n")
	default:
		fmt.Fprintf(os.Stderr, "scriggo %s: unknown command\n", cmd)
		fmt.Fprintf(os.Stderr, "Run 'scriggo help' for usage.\n")
		os.Exit(1)
	}
}

// scriggoGen handles 'scriggo gen'.
func scriggoGen() {

	// Sets usage.
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `usage: scriggo gen [-l] [-t] [-s] [-p] [-goos GOOSs] [-variable variable] [-o output] filename`)
		fmt.Fprintf(os.Stderr, `Run 'scriggo help gen' for details`)
	}

	// Finds the default GOOS.
	defaultGOOS := os.Getenv("GOOS")
	if defaultGOOS == "" {
		defaultGOOS = runtime.GOOS
	}

	// Sets command line variables.
	script := flag.Bool("s", false, "generate a script interpreter")
	template := flag.Bool("t", false, "generate a template interpreter")
	program := flag.Bool("p", false, "generate a program interpreter")
	loader := flag.Bool("l", false, "generate a package loader")
	goossArg := flag.String("goos", defaultGOOS, "Target GOOSs (separated by commas). If not provided, tries to set from 1) GOOS environment variable 2) scriggob runtime's GOOS")
	loaderVarName := flag.String("variable", "packages", "Custom variable name")
	outputDir := flag.String("o", "scriggo-interpreter", "Custom variable name")

	// CLI arguments validation and parsing.
	flag.Parse()
	if !*script && !*template && !*program && !*loader {
		fmt.Fprintf(os.Stderr, "no gen type specified\n")
		flag.Usage()
		os.Exit(1)
	}
	if *loader && (*script || *template || *program) {
		fmt.Fprintf(os.Stderr, "cannot use -l in conjuction with other gen type (-s, -t or -p)\n")
		flag.Usage()
		os.Exit(1)
	}
	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "no filename has been specified\n")
		flag.Usage()
		os.Exit(1)
	}
	inputFile := flag.Arg(0)
	gooss := strings.Split(*goossArg, ",")

	// Reads informations from inputFile.
	data, err := ioutil.ReadFile(inputFile)
	pd, err := parseImports(data)
	if err != nil {
		panic(err)
	}

	// Generates a package loader.
	if *loader {
		for _, goos := range gooss {
			data := generatePackages(pd, inputFile, *loaderVarName, goos)
			inputFileBase := filepath.Base(inputFile)
			inputFileBaseNoExt := strings.TrimSuffix(inputFileBase, filepath.Ext(inputFileBase))
			newBase := inputFileBaseNoExt + "_" + goBaseVersion(runtime.Version()) + "_" + goos + filepath.Ext(inputFileBase)
			out := filepath.Join(filepath.Dir(inputFile), newBase)
			f, err := os.Create(out)
			if err != nil {
				printErrorAndQuit(err)
			}
			_, err = f.WriteString(data)
			if err != nil {
				printErrorAndQuit(err)
			}
			err = goImports(out)
			if err != nil {
				printErrorAndQuit(err)
			}
		}
		os.Exit(0)
	}

	// Generates sources for a new interpreter.
	if *template || *script || *program {
		err := os.MkdirAll(*outputDir, dirPerm)
		if err != nil {
			panic(err)
		}
		for _, goos := range gooss {
			data := generatePackages(pd, inputFile, "packages", goos)
			outPkgsFile := filepath.Join(*outputDir, "pkgs_"+goBaseVersion(runtime.Version())+"_"+goos+".go")
			err = ioutil.WriteFile(outPkgsFile, []byte(data), filePerm)
			if err != nil {
				panic(err)
			}
			err = goImports(outPkgsFile)
			if err != nil {
				panic(err)
			}
		}
		err = ioutil.WriteFile(filepath.Join(*outputDir, "main.go"), []byte(skel), filePerm)
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}
}
