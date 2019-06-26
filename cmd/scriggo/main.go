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
	flag.Usage = commandsHelp["scriggo"]

	// No command provided.
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	cmd := os.Args[1]
	os.Args = append(os.Args[:1], os.Args[2:]...)
	command, ok := commands[cmd]
	if !ok {
		stderr(
			fmt.Sprintf("scriggo %s: unknown command", cmd),
			`Run 'scriggo help' for usage.`,
		)
		os.Exit(1)
	}
	command()
}

func stderr(lines ...string) {
	for _, l := range lines {
		fmt.Fprint(os.Stderr, l+"\n")
	}
}

var commandsHelp = map[string]func(){
	"scriggo": func() {
		stderr(
			`Scriggo is a tool for managing Scriggo interpreters and loaders`,
			``,
			`Usage:`,
			``,
			`	   scriggo <command> [arguments]`,
			``,
			`The commands are:`,
			``,
			`	   bug         start a bug report`,
			`	   gen         generate an interpreter or a loader`,
			`	   version     print Scriggo version`,
			``,
			`Use "scriggo help <command>" for more information about a command.`,
		)
		flag.PrintDefaults()
	},
	"bug": func() {
		stderr(
			`usage: scriggo bug`,
			`Bug opens the default browser and starts a new bug report.`,
			`The report includes useful system information.`,
		)
	},
	"gen": func() {
		stderr(
			`usage: scriggo gen [-l] [-t] [-s] [-p] [-goos GOOSs] [-variable variable] [-o output] filename`,
			`Run 'scriggo help gen' for details`,
		)
	},
	"version": func() {
		stderr(
			`usage: scriggo version`,
		)
	},
}

var commands = map[string]func(){
	"bug": func() {
		panic("TODO: not implemented") // TODO(Gianluca): to implement.
	},
	"gen": func() {
		flag.Usage = commandsHelp["gen"]
		scriggoGen()
	},
	"help": func() {
		if len(os.Args) == 1 {
			flag.Usage()
			os.Exit(0)
		}
		topic := os.Args[1]
		help, ok := commandsHelp[topic]
		if !ok {
			fmt.Fprintf(os.Stderr, "scriggo help %s: unknown help topic. Run 'scriggo help'.\n", topic)
			os.Exit(1)
		}
		help()
	},
	"version": func() {
		fmt.Printf("Scriggo module version:            (TODO) \n") // TODO(Gianluca): use real version.
		fmt.Printf("Scriggo tool version:              (TODO) \n") // TODO(Gianluca): use real version.
		fmt.Printf("Go version used to build Scriggo:  %s\n", runtime.Version())
	},
}

// scriggoGen handles 'scriggo gen'.
func scriggoGen() {

	// Finds the default GOOS.
	defaultGOOS := os.Getenv("GOOS")
	if defaultGOOS == "" {
		defaultGOOS = runtime.GOOS
	}

	flag.Parse()

	inputFile := flag.Arg(0)

	// Reads informations from inputFile.
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	pd, err := parseImports(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error on file %q: %s\n", inputFile, err)
		os.Exit(1)
	}
	pd.filepath = inputFile
	if len(pd.fileComment.goos) == 0 {
		pd.fileComment.goos = []string{defaultGOOS}
	}

	// Generates a package loader.
	if pd.fileComment.embedded {
		if pd.containsMain() {
			panic("TODO: not implemented") // TODO(Gianluca): to implement.
		}
		if pd.fileComment.embeddedVariable == "" {
			pd.fileComment.embeddedVariable = "packages"
		}
		for _, goos := range pd.fileComment.goos {
			data, hasContent, err := renderPackages(pd, pd.fileComment.embeddedVariable, goos)
			if err != nil {
				panic(err)
			}
			// Data has been generated but has no content (only has a
			// "skeleton"): do not write file.
			if !hasContent {
				continue
			}
			inputFileBase := filepath.Base(inputFile)
			inputFileBaseNoExt := strings.TrimSuffix(inputFileBase, filepath.Ext(inputFileBase))
			newBase := inputFileBaseNoExt + "_" + goBaseVersion(runtime.Version()) + "_" + goos + filepath.Ext(inputFileBase)
			out := filepath.Join(filepath.Dir(inputFile), newBase)
			ioutil.WriteFile(out, []byte(data), filePerm)
			if err != nil {
				panic(err)
			}
			err = goImports(out)
			if err != nil {
				panic(err)
			}
		}
		os.Exit(0)
	}

	// Generates sources for a new interpreter.
	if pd.fileComment.template || pd.fileComment.script || pd.fileComment.program {
		if pd.fileComment.output == "" {
			pd.fileComment.output = strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "-interpreter"
		}
		err := os.MkdirAll(pd.fileComment.output, dirPerm)
		if err != nil {
			panic(err)
		}
		for _, goos := range pd.fileComment.goos {
			pd.pkgName = "main"
			if pd.containsMain() {
				if !pd.fileComment.template && !pd.fileComment.script {
					panic("cannot have main if not making a template or script interpreter") // TODO(Gianluca).
				}
				main, err := renderPackageMain(pd, goos)
				if err != nil {
					panic(err)
				}
				mainFile := filepath.Join(pd.fileComment.output, "main_"+goBaseVersion(runtime.Version())+"_"+goos+".go")
				err = ioutil.WriteFile(mainFile, []byte(main), filePerm)
				if err != nil {
					panic(err)
				}
				err = goImports(mainFile)
				if err != nil {
					panic(err)
				}
			} else { // pd does not contain main, so has packages (or is empty).
				if len(pd.imports) > 0 {
					if pd.fileComment.template && !pd.fileComment.script && !pd.fileComment.program {
						panic("cannot have packages if making a template interpreter") // TODO(Gianluca).
					}
				}
			}
			data, hasContent, err := renderPackages(pd, "packages", goos)
			if err != nil {
				panic(err)
			}
			// Data has been generated but has no content (only has a
			// "skeleton"): do not write file.
			if !hasContent {
				continue
			}
			outPkgsFile := filepath.Join(pd.fileComment.output, "pkgs_"+goBaseVersion(runtime.Version())+"_"+goos+".go")
			err = ioutil.WriteFile(outPkgsFile, []byte(data), filePerm)
			if err != nil {
				panic(err)
			}
			err = goImports(outPkgsFile)
			if err != nil {
				panic(err)
			}
		}
		mainPath := filepath.Join(pd.fileComment.output, "main.go")
		err = ioutil.WriteFile(mainPath, makeInterpreterSkeleton(pd.fileComment.program, pd.fileComment.script, pd.fileComment.template), filePerm)
		if err != nil {
			panic(err)
		}
		err = goImports(mainPath)
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}
}
