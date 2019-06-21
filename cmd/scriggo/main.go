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

const help = `Scriggo is a tool for managing Scriggo interpreters and loaders

Usage:
	scriggo <command> [arguments]

The commands are:
	
	gen		generate an interpreter or a loader

Use "scriggo help <command>" for more information about a command.`

func main() {
	if len(os.Args) == 1 {
		fmt.Println(help)
		os.Exit(0)
	}
	cmd := os.Args[1]
	os.Args = append(os.Args[:1], os.Args[2:]...)
	switch cmd {
	case "gen":
		defaultGOOS := os.Getenv("GOOS")
		if defaultGOOS == "" {
			defaultGOOS = runtime.GOOS
		}
		script := flag.Bool("s", false, "generate a script interpreter")
		template := flag.Bool("t", false, "generate a template interpreter")
		program := flag.Bool("p", false, "generate a program interpreter")
		loader := flag.Bool("l", false, "generate a package loader")
		goossArg := flag.String("goos", defaultGOOS, "Target GOOSs (separated by commas). If not provided, tries to set from 1) GOOS environment variable 2) scriggob runtime's GOOS")
		variable := flag.String("variable", "packages", "Custom variable name")
		outputDir := flag.String("o", "scriggo-interpreter", "Custom variable name")
		flag.Parse()
		if !*script && !*template && !*program && !*loader {
			commandLineError("specify what to build")
			os.Exit(1)
		}
		if *loader && (*script || *template || *program) {
			panic("error")
		}
		if len(flag.Args()) == 0 {
			panic("file name needed")
		}
		inputFile := flag.Arg(0)
		gooss := strings.Split(*goossArg, ",")
		packages, pkgName, err := extractImports(inputFile)
		if err != nil {
			panic(err)
		}
		if pkgName == "" {
			panic("pkg name cant be empty")
		}
		if *loader {
			for _, goos := range gooss {
				out := generatePackages(packages, inputFile, *variable, pkgName, goos)
				importsFileBase := filepath.Base(inputFile)
				importsFileBaseWithoutExtension := strings.TrimSuffix(importsFileBase, filepath.Ext(importsFileBase))
				var newBase string
				newBase = importsFileBaseWithoutExtension + "_" + goBaseVersion(runtime.Version()) + "_" + goos + filepath.Ext(importsFileBase)
				outPath := filepath.Join(filepath.Dir(inputFile), newBase)
				f, err := os.Create(outPath)
				if err != nil {
					printErrorAndQuit(err)
				}
				_, err = f.WriteString(out)
				if err != nil {
					printErrorAndQuit(err)
				}
				err = goImports(outPath)
				if err != nil {
					printErrorAndQuit(err)
				}
			}
			return
		}
		if *template || *script || *program {
			if *outputDir == "" {
				commandLineError(fmt.Sprintf("an output dir must be specified when running in...")) // TODO(Gianluca).
			}
			err := os.MkdirAll(*outputDir, dirPerm)
			if err != nil {
				panic(err)
			}
			for _, goos := range gooss {
				out := generatePackages(packages, inputFile, "packages", "main", goos)
				pkgsFile := filepath.Join(*outputDir, "pkgs_"+goBaseVersion(runtime.Version())+"_"+goos+".go")
				err = ioutil.WriteFile(pkgsFile, []byte(out), filePerm)
				if err != nil {
					panic(err)
				}
				err = goImports(pkgsFile)
				if err != nil {
					panic(err)
				}
			}
			err = ioutil.WriteFile(filepath.Join(*outputDir, "main.go"), []byte(skel), filePerm)
			if err != nil {
				panic(err)
			}
			return
		}
	case "help":
	case "version":
		fmt.Print(`Scriggo module version:  1.0.5
Scriggo tool version:     1.0.4
`) // TODO(Gianluca).
	default:
		commandLineError(fmt.Sprintf("command %q not supported", cmd))
	}
}

// func mainold() {

// 	// TODO(Gianluca): check if `go generate` version has the same GOROOT used
// 	// in package lookup. If not, panic.

// 	flag.Usage = func() {
// 		u := `Scriggo is a tool for managing Scriggo source code

// Usage:
// 	scriggo <command> [arguments]

// The commands are:

// 	gen		generate an interpreter or a loader

// Use "scriggo help <command>" for more information about a command.
// `
// 		fmt.Fprint(os.Stderr, u)
// 		flag.PrintDefaults()
// 	}

// 	goossArg := flag.String("goos", "", "Target GOOSs (separated by commas). If not provided, tries to set from 1) GOOS environment variable 2) scriggob runtime's GOOS")
// 	variable := flag.String("variable", "", "Custom variable name. Only effective when running in \"imports\" mode")
// 	outputDir := flag.String("output", "", "Output directory")

// 	flag.Parse()

// 	if len(flag.Args()) < 2 {
// 		commandLineError("too few arguments")
// 	}
// 	if len(flag.Args()) > 2 {
// 		commandLineError("too many arguments")
// 	}

// 	var gooss []string
// 	if *goossArg == "" {
// 		gooss = []string{os.Getenv("GOOS")}
// 		if gooss[0] == "" {
// 			gooss[0] = runtime.GOOS
// 		}
// 	} else {
// 		gooss = strings.Split(*goossArg, ",")
// 	}

// 	if len(flag.Args()) == 1 {
// 		commandLineError("INPUT_FILE must be provided as command line argument")
// 	}
// 	inputFile := flag.Arg(1)

// 	packages, pkgName, err := extractImports(inputFile)
// 	if err != nil {
// 		panic(err)
// 	}
// 	if pkgName == "" {
// 		panic("pkg name cant be empty")
// 	}

// 	switch mode := flag.Arg(0); mode {
// 	case "imports":
// 		if *variable == "" {
// 			commandLineError("a custom variable name must be specified when using scriggob in \"imports\" mode")
// 		}
// 		if *outputDir != "" {
// 			panic("TODO: not implemented") // TODO(Gianluca): to implement.
// 		}
// 		for _, goos := range gooss {
// 			out := generatePackages(packages, inputFile, *variable, pkgName, goos)
// 			importsFileBase := filepath.Base(inputFile)
// 			importsFileBaseWithoutExtension := strings.TrimSuffix(importsFileBase, filepath.Ext(importsFileBase))
// 			var newBase string
// 			{ // TODO(Gianluca): Remove from here...
// 				if goos == "" {
// 					panic("bug")
// 				}
// 			} // ...to here.
// 			newBase = importsFileBaseWithoutExtension + "_" + goBaseVersion(runtime.Version()) + "_" + goos + filepath.Ext(importsFileBase)
// 			outPath := filepath.Join(filepath.Dir(inputFile), newBase)
// 			f, err := os.Create(outPath)
// 			if err != nil {
// 				printErrorAndQuit(err)
// 			}
// 			_, err = f.WriteString(out)
// 			if err != nil {
// 				printErrorAndQuit(err)
// 			}
// 			err = goImports(outPath)
// 			if err != nil {
// 				printErrorAndQuit(err)
// 			}
// 		}
// 	case "sources", "build":
// 		if *outputDir == "" {
// 			commandLineError(fmt.Sprintf("an output dir must be specified when running in %q mode", mode))
// 		}
// 		err := os.MkdirAll(*outputDir, dirPerm)
// 		if err != nil {
// 			panic(err)
// 		}
// 		for _, goos := range gooss {
// 			out := generatePackages(packages, inputFile, "packages", "main", goos)
// 			pkgsFile := filepath.Join(*outputDir, "pkgs_"+goBaseVersion(runtime.Version())+"_"+goos+".go")
// 			err = ioutil.WriteFile(pkgsFile, []byte(out), filePerm)
// 			if err != nil {
// 				panic(err)
// 			}
// 			err = goImports(pkgsFile)
// 			if err != nil {
// 				panic(err)
// 			}
// 		}
// 		err = ioutil.WriteFile(filepath.Join(*outputDir, "main.go"), []byte(skel), filePerm)
// 		if err != nil {
// 			panic(err)
// 		}
// 		if mode == "build" {
// 			err := goBuild(*outputDir)
// 			if err != nil {
// 				panic(err)
// 			}
// 		}
// 	default:
// 		commandLineError(fmt.Sprintf("mode %q is not valid", flag.Arg(0)))
// 	}

// }

func printErrorAndQuit(err interface{}) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func commandLineError(err string) {
	fmt.Printf(ansi_red+"Error: %s"+ansi_reset+"\n", err)
	flag.Usage()
	os.Exit(1)
}

const (
	ansi_red   = "\033[1;31m"
	ansi_reset = "\033[0m"
)
