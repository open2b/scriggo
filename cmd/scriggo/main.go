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
	scriggo(os.Args...)
}

// TestEnvironment is true when testing Scriggo, false otherwise.
var TestEnvironment = false

// exit causes the current program to exit with the given status code. If
// running in a test environment, every exit call is a no-op.
func exit(status int) {
	if !TestEnvironment {
		os.Exit(status)
	}
}

// stderr prints lines on stderr.
func stderr(lines ...string) {
	for _, l := range lines {
		fmt.Fprint(os.Stderr, l+"\n")
	}
}

// exitError prints msg on stderr with a bold red color and exits with status
// code 1.
func exitError(format string, a ...interface{}) {
	msg := fmt.Errorf(format, a...)
	stderr("\033[1;31m"+msg.Error()+"\033[0m", `exit status 1`)
	exit(1)
	return
}

// scriggo runs command 'scriggo' with given args. First argument must be
// executable name.
func scriggo(args ...string) {
	flag.Usage = commandsHelp["scriggo"]

	// No command provided.
	if len(args) == 1 {
		flag.Usage()
		exit(0)
		return
	}

	cmdArg := args[1]

	// Used by flag.Parse.
	os.Args = append(args[:1], args[2:]...)

	cmd, ok := commands[cmdArg]
	if !ok {
		stderr(
			fmt.Sprintf("scriggo %s: unknown command", cmdArg),
			`Run 'scriggo help' for usage.`,
		)
		exit(1)
		return
	}
	cmd()
}

// commandsHelp maps a command name to a function that prints help for that
// command.
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
			`	bug            start a bug report`,
			`	gen            generate an interpreter or a loader`,
			`	install        install an interpreter`,
			`	version        print Scriggo version`,
			``,
			`Use "scriggo help <command>" for more information about a command.`,
			``,
			`Additional help topics:`,
			``,
			`	descriptor     syntax of descriptor file`,
		)
		flag.PrintDefaults()
	},
	// Help topics.
	"descriptor": func() {
		stderr(
			`A Scriggo descriptor file consits of a valid Go package source code containing one Scriggo file comment and one or more imports, which may in turn have a Scriggo import comment`,
			``,
			`An example Scriggo descriptor is:`,
			``,
			`	//scriggo: interpreters:"script"`,
			`	`,
			`	package x`,
			`	`,
			`	import (`,
			`		_ "fmt"`,
			`		_ "math" //scriggo: main uncapitalize`,
			`	)`,
			``,
			`This Scriggo descriptor describes a Scriggo interpreter provides package "fmt" (available through an import statement) and package "math" as "builtin", with all names "uncapitalized".`,
			``,
			`Each import statement should have a name _, which prevents tools like goimports from removing import.`,
			``,
			`Options available in the Scriggo file comment are:`,
			``,
			`	interpreters[:targets]  describe an interpreter for targets. Valid targets are "template, "script" and "program". If not targets are specified, it's assumed by default all available interpreters.`,
			`	embedded                describe an embedded packages declaration`,
			`	output                  select output file/directory`,
			`	goos:GOOSs              force GOOS to the specified value. More than one value can be provided`,
			``,
			`Options available as Scriggo import comments are:`,
			``,
			`	main                    import as package main. Only available in scripts an templates`,
			`	uncapitalize            declarations imported as main are "uncapitalized"`,
			`	path                    force an alternative path for import`,
			`	export:names            only export names`,
			`	noexport:names          export everything excluding names`,
			``,
			`Example import comments`,
			``,
			`Default. Makes "fmt" available in Scriggo as would be available in Go:`,
			`	import _ "fmt" //scriggo:`,
			``,
			`Import all declarations from "fmt" in package main, making them accessible without a selector:`,
			`	import _ "fmt" //scriggo: main`,
			``,
			`Import all declarations from "fmt" in package main with uncapitalized names, making them accessible without a selector:`,
			`	import _ "fmt" //scriggo: main uncapitalize`,
			``,
			`Import all declarations from "fmt" excluding "Print" and Println":`,
			`	import _ "fmt" //scriggo: noexport:"Print,Println"`,
		)
	},

	// Commands helps.
	"bug": func() {
		stderr(
			`usage: scriggo bug`,
			`Bug opens the default browser and starts a new bug report.`,
			`The report includes useful system information.`,
		)
	},
	"gen": func() {
		stderr(
			`usage: scriggo gen [target]`,
			`Gen generates a directory containing source code for a new interpreter`,
		)
	},
	"install": func() {
		stderr(
			`usage: scriggo install [target]`,
			`Install installs an executable Scriggo interpreter on system. Output directory is the same used by 'go install' (see 'go help install' for details)`,
		)
	},
	"version": func() {
		stderr(
			`usage: scriggo version`,
		)
	},
}

// commands maps a command name to a function that executes that command.
// Commands are called by command-line using:
//
//		scriggo command
//
var commands = map[string]func(){
	"bug": func() {
		flag.Usage = commandsHelp["bug"]
		panic("TODO: not implemented") // TODO(Gianluca): to implement.
	},
	"install": func() {
		flag.Usage = commandsHelp["install"]
		scriggoGen(true)
	},
	"gen": func() {
		flag.Usage = commandsHelp["gen"]
		scriggoGen(false)
	},
	"help": func() {
		if len(os.Args) == 1 {
			flag.Usage()
			exit(0)
			return
		}
		topic := os.Args[1]
		help, ok := commandsHelp[topic]
		if !ok {
			fmt.Fprintf(os.Stderr, "scriggo help %s: unknown help topic. Run 'scriggo help'.\n", topic)
			exit(1)
			return
		}
		help()
	},
	"version": func() {
		flag.Usage = commandsHelp["version"]
		fmt.Printf("Scriggo module version:            (TODO) \n") // TODO(Gianluca): use real version.
		fmt.Printf("Scriggo tool version:              (TODO) \n") // TODO(Gianluca): use real version.
		fmt.Printf("Go version used to build Scriggo:  %s\n", runtime.Version())
	},
}

// scriggoGen executes command:
//
//		scriggo gen
//		scriggo install
//
// If install is set, interpreter will be installed as executable and
// interpreter sources will be removed.
func scriggoGen(install bool) {

	flag.Parse()

	// No arguments provided: this is not an error.
	if len(flag.Args()) == 0 {
		flag.Usage()
		exit(0)
		return
	}

	// Too many arguments provided.
	if len(flag.Args()) > 1 {
		stderr(`bad number of arguments`)
		flag.Usage()
		exit(1)
		return
	}

	inputPath := flag.Arg(0)

	data, err := getScriggoDescriptorData(inputPath)
	if err != nil {
		exitError(err.Error())
	}

	sd, err := parseScriggoDescriptor(data)
	if err != nil {
		exitError("path %q: %s", inputPath, err)
	}
	sd.filepath = inputPath
	if len(sd.comment.goos) == 0 {
		defaultGOOS := os.Getenv("GOOS")
		if defaultGOOS == "" {
			defaultGOOS = runtime.GOOS
		}
		sd.comment.goos = []string{defaultGOOS}
	}

	// Generates an embeddable loader.
	if sd.comment.embedded {
		if install {
			stderr(`scriggo install is not compatible with a Scriggo descriptor that generates embedded packages`)
			flag.Usage()
			exit(1)
			return
		}
		if sd.comment.varName == "" {
			sd.comment.varName = "packages"
		}
		inputFileBase := filepath.Base(inputPath)
		inputBaseNoExt := strings.TrimSuffix(inputFileBase, filepath.Ext(inputFileBase))

		// Iterates over all GOOS.
		for _, goos := range sd.comment.goos {

			// scriggoDescriptor contains at least one main.
			if sd.containsMain() {
				newMainBase := inputBaseNoExt + "_main_" + goBaseVersion(runtime.Version()) + "_" + goos + filepath.Ext(inputFileBase)
				main, err := renderPackageMain(sd, goos)
				if err != nil {
					exitError("rendering package main: %s", err)
				}
				mainFile := filepath.Join(filepath.Dir(inputPath), newMainBase)
				err = ioutil.WriteFile(mainFile, []byte(main), filePerm)
				if err != nil {
					exitError("writing package main: %s", err)
				}
				err = goImports(mainFile)
				if err != nil {
					exitError("goimports on file %q: %s", mainFile, err)
				}
			}

			// Render all packages, ignoring main.
			data, hasContent, err := renderPackages(sd, sd.comment.varName, goos)
			if err != nil {
				panic(err)
			}

			// Data has been generated but has no content (only has a
			// "skeleton"): do not write file.
			if !hasContent {
				continue
			}

			newBase := inputBaseNoExt + "_" + goBaseVersion(runtime.Version()) + "_" + goos + filepath.Ext(inputFileBase)
			out := filepath.Join(filepath.Dir(inputPath), newBase)

			// Writes packages on disk and runs "goimports" on that file.
			ioutil.WriteFile(out, []byte(data), filePerm)
			if err != nil {
				exitError("writing packages file: %s", err)
			}
			err = goImports(out)
			if err != nil {
				exitError("goimports on file %q: %s", out, err)
			}

		}
		exit(0)
		return
	}

	// Generates sources for a new interpreter.
	if sd.comment.template || sd.comment.script || sd.comment.program {
		if sd.comment.output == "" {
			sd.comment.output = strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
		}

		// Creates a temporary directory for interpreter sources. If installing,
		// directory will be lost. If generating sources and no errors occurred,
		// tmpDir will be moved to the correct path.
		tmpDir, err := ioutil.TempDir("", "scriggo")
		if err != nil {
			exitError(err.Error())
		}
		tmpDir = filepath.Join(tmpDir, sd.pkgName)

		err = os.MkdirAll(tmpDir, dirPerm)
		if err != nil {
			exitError(err.Error())
		}

		for _, goos := range sd.comment.goos {
			sd.pkgName = "main"
			if sd.containsMain() {
				if !sd.comment.template && !sd.comment.script {
					exitError("cannot have main if not making a template or script interpreter")
				}
				main, err := renderPackageMain(sd, goos)
				if err != nil {
					exitError("rendering package main: %s", err)
				}
				mainFile := filepath.Join(tmpDir, "main_"+goBaseVersion(runtime.Version())+"_"+goos+".go")
				err = ioutil.WriteFile(mainFile, []byte(main), filePerm)
				if err != nil {
					exitError("writing package main: %s", err)
				}
				err = goImports(mainFile)
				if err != nil {
					exitError("goimports on file %q: %s", mainFile, err)
				}
			}

			// When making an interpreter that reads only template sources, sd
			// cannot contain only packages.
			if sd.comment.template && !sd.comment.script && !sd.comment.program && !sd.containsMain() && len(sd.imports) > 0 {
				exitError("cannot have packages if making a template interpreter")
			}

			data, hasContent, err := renderPackages(sd, "packages", goos)
			if err != nil {
				exitError("rendering packages: %s", err)
			}
			// Data has been generated but has no content (only has a
			// "skeleton"): do not write file.
			if !hasContent {
				continue
			}
			outPkgsFile := filepath.Join(tmpDir, "pkgs_"+goBaseVersion(runtime.Version())+"_"+goos+".go")
			err = ioutil.WriteFile(outPkgsFile, []byte(data), filePerm)
			if err != nil {
				exitError("writing packages file: %s", err)
			}
			err = goImports(outPkgsFile)
			if err != nil {
				exitError("goimports on file %q: %s", outPkgsFile, err)
			}
		}

		// Write package main on disk and run "goimports" on it.
		mainPath := filepath.Join(tmpDir, "main.go")
		err = ioutil.WriteFile(mainPath, makeInterpreterSource(sd.comment.program, sd.comment.script, sd.comment.template), filePerm)
		if err != nil {
			exitError("writing interpreter file: %s", err)
		}
		goModPath := filepath.Join(tmpDir, "go.mod")
		err = ioutil.WriteFile(goModPath, makeExecutableGoMod(inputPath), filePerm)
		if err != nil {
			exitError("writing interpreter file: %s", err)
		}
		err = goImports(mainPath)
		if err != nil {
			exitError("goimports on file %q: %s", mainPath, err)
		}

		if install {
			err = goInstall(tmpDir)
			if err != nil {
				exitError("goimports on dir %q: %s", tmpDir, err)
			}
			exit(0)
			return
		}

		// Move interpeter from tmpDir to correct dir.
		fis, err := ioutil.ReadDir(tmpDir)
		if err != nil {
			exitError(err.Error())
		}
		err = os.MkdirAll(sd.comment.output, dirPerm)
		if err != nil {
			exitError(err.Error())
		}
		for _, fi := range fis {
			if !fi.IsDir() {
				filePath := filepath.Join(tmpDir, fi.Name())
				newFilePath := filepath.Join(sd.comment.output, fi.Name())
				data, err := ioutil.ReadFile(filePath)
				if err != nil {
					exitError(err.Error())
				}
				err = ioutil.WriteFile(newFilePath, data, filePerm)
				if err != nil {
					exitError(err.Error())
				}
			}
		}
		exit(0)
		return
	}
}
