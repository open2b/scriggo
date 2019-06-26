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

// exit prints msg on stderr with a bold red color and exits with status code 1.
func exit(format string, a ...interface{}) {
	msg := fmt.Errorf(format, a...)
	stderr("\033[1;31m"+msg.Error()+"\033[0m", `exit status 1`)
	os.Exit(1)
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
			`usage: scriggo gen filename.go`,
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

	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(0)
	}
	if len(flag.Args()) > 1 {
		stderr(`bad number of arguments`)
		flag.Usage()
		os.Exit(1)
	}

	inputFile := flag.Arg(0)

	// Reads informations from inputFile.
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		exit(err.Error())
	}
	sd, err := parseScriggoDescriptor(data)
	if err != nil {
		exit("file %q: %s", inputFile, err)
	}
	sd.filepath = inputFile
	if len(sd.fileComment.goos) == 0 {
		defaultGOOS := os.Getenv("GOOS")
		if defaultGOOS == "" {
			defaultGOOS = runtime.GOOS
		}
		sd.fileComment.goos = []string{defaultGOOS}
	}

	// Generates an embeddable loader.
	if sd.fileComment.embedded {
		if sd.fileComment.embeddedVariable == "" {
			sd.fileComment.embeddedVariable = "packages"
		}
		inputFileBase := filepath.Base(inputFile)
		inputFileBaseNoExt := strings.TrimSuffix(inputFileBase, filepath.Ext(inputFileBase))
		for _, goos := range sd.fileComment.goos {
			if sd.containsMain() {
				newMainBase := inputFileBaseNoExt + "_main_" + goBaseVersion(runtime.Version()) + "_" + goos + filepath.Ext(inputFileBase)
				main, err := renderPackageMain(sd, goos)
				if err != nil {
					exit("rendering package main: %s", err)
				}
				mainFile := filepath.Join(filepath.Dir(inputFile), newMainBase)
				err = ioutil.WriteFile(mainFile, []byte(main), filePerm)
				if err != nil {
					exit("writing package main: %s", err)
				}
				err = goImports(mainFile)
				if err != nil {
					exit("goimports on file %q: %s", mainFile, err)
				}
			}
			data, hasContent, err := renderPackages(sd, sd.fileComment.embeddedVariable, goos)
			if err != nil {
				panic(err)
			}
			// Data has been generated but has no content (only has a
			// "skeleton"): do not write file.
			if !hasContent {
				continue
			}
			newPackagesBase := inputFileBaseNoExt + "_" + goBaseVersion(runtime.Version()) + "_" + goos + filepath.Ext(inputFileBase)
			out := filepath.Join(filepath.Dir(inputFile), newPackagesBase)
			ioutil.WriteFile(out, []byte(data), filePerm)
			if err != nil {
				exit("writing packages file: %s", err)
			}
			err = goImports(out)
			if err != nil {
				exit("goimports on file %q: %s", out, err)
			}
		}
		os.Exit(0)
	}

	// Generates sources for a new interpreter.
	if sd.fileComment.template || sd.fileComment.script || sd.fileComment.program {
		if sd.fileComment.output == "" {
			sd.fileComment.output = strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "-interpreter"
		}
		err := os.MkdirAll(sd.fileComment.output, dirPerm)
		if err != nil {
			panic(err)
		}
		for _, goos := range sd.fileComment.goos {
			sd.pkgName = "main"
			if sd.containsMain() {
				if !sd.fileComment.template && !sd.fileComment.script {
					panic("cannot have main if not making a template or script interpreter") // TODO(Gianluca).
				}
				main, err := renderPackageMain(sd, goos)
				if err != nil {
					exit("rendering package main: %s", err)
				}
				mainFile := filepath.Join(sd.fileComment.output, "main_"+goBaseVersion(runtime.Version())+"_"+goos+".go")
				err = ioutil.WriteFile(mainFile, []byte(main), filePerm)
				if err != nil {
					exit("writing package main: %s", err)
				}
				err = goImports(mainFile)
				if err != nil {
					exit("goimports on file %q: %s", mainFile, err)
				}
			} else { // pd does not contain main, so has packages (or is empty).
				if len(sd.imports) > 0 {
					if sd.fileComment.template && !sd.fileComment.script && !sd.fileComment.program {
						panic("cannot have packages if making a template interpreter") // TODO(Gianluca).
					}
				}
			}
			data, hasContent, err := renderPackages(sd, "packages", goos)
			if err != nil {
				exit("rendering packages: %s", err)
			}
			// Data has been generated but has no content (only has a
			// "skeleton"): do not write file.
			if !hasContent {
				continue
			}
			outPkgsFile := filepath.Join(sd.fileComment.output, "pkgs_"+goBaseVersion(runtime.Version())+"_"+goos+".go")
			err = ioutil.WriteFile(outPkgsFile, []byte(data), filePerm)
			if err != nil {
				exit("writing packages file: %s", err)
			}
			err = goImports(outPkgsFile)
			if err != nil {
				exit("goimports on file %q: %s", outPkgsFile, err)
			}
		}
		mainPath := filepath.Join(sd.fileComment.output, "main.go")
		err = ioutil.WriteFile(mainPath, makeInterpreterSkeleton(sd.fileComment.program, sd.fileComment.script, sd.fileComment.template), filePerm)
		if err != nil {
			exit("writing interpreter file: %s", err)
		}
		err = goImports(mainPath)
		if err != nil {
			exit("goimports on file %q: %s", mainPath, err)
		}
		os.Exit(0)
	}
}
