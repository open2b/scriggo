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

func commandLineError(err string) {
	fmt.Printf("Command line error: %s\n", err)
	fmt.Println("Use -help to get help on scriggb usage")
	os.Exit(1)
}

func main() {

	// TODO(Gianluca): check if `go generate` version has the same GOROOT used
	// in package lookup. If not, panic.

	flag.Usage = func() {
		u := `
	Usage:   scriggob [OPTIONS] MODE INPUT_FILE

Available modes
---------------

	MODE = "imports" | "sources" | "build"

		imports
			Generate import file starting from SOURCE putting result into DIR
			Require "-variable-name" argument.
		sources
			Populate DIR with an new intepreter source code which can execute code using the same imports as SOURCE
		build
			Same as "sources" but also build the intepreter

Options
-------
	`
		fmt.Fprint(os.Stderr, u)
		flag.PrintDefaults()
	}

	goossArg := flag.String("goos", "", "Target GOOSs (separated by commas). Default to current GOOS")
	variableName := flag.String("variable-name", "", "Custom variable name. Only effective when running in \"imports\" mode")

	flag.Parse()

	if len(flag.Args()) == 0 {
		commandLineError("MODE must be provided as command line argument")
	}

	var gooss []string
	if *goossArg == "" {
		gooss = []string{os.Getenv("GOOS")}
		if gooss[0] == "" {
			gooss[0] = runtime.GOOS
		}
	} else {
		gooss = strings.Split(*goossArg, ",")
	}

	if len(flag.Args()) == 1 {
		commandLineError("INPUT_FILE must be provided as command line argument")
	}
	inputFile := flag.Arg(1)

	var outputDir string
	if len(flag.Args()) == 3 {
		outputDir = flag.Arg(2)
	}

	packages, pkgName, err := extractImports(inputFile)
	if err != nil {
		panic(err)
	}
	if pkgName == "" {
		panic("pkg name must be specified by command line")
	}

	switch flag.Arg(0) {
	case "imports":
		if *variableName == "" {
			commandLineError("a custom variable name must be specified when using scriggob in \"imports\" mode")
		}
		if outputDir != "" {
			panic("TODO: not implemented") // TODO(Gianluca): to implement.
		}
		for _, goos := range gooss {
			out := generatePackages(packages, inputFile, *variableName, pkgName, goos)
			importsFileBase := filepath.Base(inputFile)
			importsFileBaseWithoutExtension := strings.TrimSuffix(importsFileBase, filepath.Ext(importsFileBase))
			var newBase string
			{ // TODO(Gianluca): Remove from here...
				if goos == "" {
					panic("bug")
				}
			} // ...to here.
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
	case "sources":
		if outputDir == "" {
			panic("TODO: not implemented") // TODO(Gianluca): to implement.
		}
		err := os.MkdirAll(outputDir, dirPerm)
		if err != nil {
			panic(err)
		}
		out := generatePackages(packages, inputFile, "packages", "main", "")
		err = ioutil.WriteFile(filepath.Join(outputDir, "packages.go"), []byte(out), filePerm)
		if err != nil {
			panic(err)
		}
		err = goImports(filepath.Join(outputDir, "packages.go"))
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(filepath.Join(outputDir, "main.go"), []byte(skel), filePerm)
		if err != nil {
			panic(err)
		}
	case "build":
		panic("TODO: not implemented") // TODO(Gianluca): to implement.
	default:
		commandLineError(fmt.Sprintf("mode %q is not valid", flag.Arg(0)))
	}

}
