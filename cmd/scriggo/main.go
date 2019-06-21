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

func main() {

	// TODO(Gianluca): check if `go generate` version has the same GOROOT used
	// in package lookup. If not, panic.

	flag.Usage = func() {
		u := `
	Usage:   scriggob [options] mode input_file

Available modes
---------------

	mode = "imports" | "sources" | "build"

		imports:
			Generates a Scriggo package reading imports from SOURCE.
			Requires the "-variable" argument.

		sources:
			Populates DIR with an new intepreter source code which can execute code using the same imports as input_file.
			Requires the "-output" argument.

		build:
			Same as "sources" but also builds the intepreter generated.
			Requires the "-output" argument.

Options
-------
`
		fmt.Fprint(os.Stderr, u)
		flag.PrintDefaults()
	}

	goossArg := flag.String("goos", "", "Target GOOSs (separated by commas). If not provided, tries to set from 1) GOOS environment variable 2) scriggob runtime's GOOS")
	variable := flag.String("variable", "", "Custom variable name. Only effective when running in \"imports\" mode")
	outputDir := flag.String("output", "", "Output directory")

	flag.Parse()

	if len(flag.Args()) < 2 {
		commandLineError("too few arguments")
	}
	if len(flag.Args()) > 2 {
		commandLineError("too many arguments")
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

	packages, pkgName, err := extractImports(inputFile)
	if err != nil {
		panic(err)
	}
	if pkgName == "" {
		panic("pkg name must be specified by command line")
	}

	switch mode := flag.Arg(0); mode {
	case "imports":
		if *variable == "" {
			commandLineError("a custom variable name must be specified when using scriggob in \"imports\" mode")
		}
		if *outputDir != "" {
			panic("TODO: not implemented") // TODO(Gianluca): to implement.
		}
		for _, goos := range gooss {
			out := generatePackages(packages, inputFile, *variable, pkgName, goos)
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
	case "sources", "build":
		if *outputDir == "" {
			commandLineError(fmt.Sprintf("an output dir must be specified when running in %q mode", mode))
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
		if mode == "build" {
			err := goBuild(*outputDir)
			if err != nil {
				panic(err)
			}
		}
	default:
		commandLineError(fmt.Sprintf("mode %q is not valid", flag.Arg(0)))
	}

}

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
