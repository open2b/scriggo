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

	flag.Usage = commandsHelp["sgo"]

	// No command provided.
	if len(os.Args) == 1 {
		flag.Usage()
		exit(0)
		return
	}

	cmdArg := os.Args[1]

	// Used by flag.Parse.
	os.Args = append(os.Args[:1], os.Args[2:]...)

	cmd, ok := commands[cmdArg]
	if !ok {
		stderr(
			fmt.Sprintf("sgo %s: unknown command", cmdArg),
			`Run 'sgo help' for usage.`,
		)
		exit(1)
		return
	}
	cmd()
}

// TestEnvironment is true when testing sgo, false otherwise.
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
		_, _ = fmt.Fprint(os.Stderr, l+"\n")
	}
}

// exitError prints an error message on stderr with a bold red color and exits
// with status code 1.
func exitError(format string, a ...interface{}) {
	msg := fmt.Errorf(format, a...)
	if runtime.GOOS == "linux" {
		stderr("\033[1;31m"+msg.Error()+"\033[0m", `exit status 1`)
	} else {
		stderr(msg.Error(), `exit status 1`)
	}
	exit(1)
	return
}

// commandsHelp maps a command name to a function that prints the help for
// that command.
var commandsHelp = map[string]func(){
	"sgo": func() {
		stderr(
			`sgo is a tool for managing Scriggo interpreters and loaders`,
			``,
			`Usage:`,
			``,
			`	sgo <command> [arguments]`,
			``,
			`The commands are:`,
			``,
			`	bug            start a bug report`,
			`	generate       generate an interpreter or a loader`,
			`	install        install an interpreter`,
			`	version        print sgo/Scriggo version`,
			``,
			`Use "sgo help <command>" for more information about a command.`,
			``,
			`Additional help topics:`,
			``,
			`	descriptor     syntax of descriptor file`,
			``,
		)
		flag.PrintDefaults()
	},
	// Help topics.
	"descriptor": func() {
		txtToHelp(helpDescriptor)
	},

	// Commands helps.
	"bug": func() {
		stderr(
			`usage: sgo bug`,
			`Bug opens the default browser and starts a new bug report.`,
			`The report includes useful system information.`,
		)
	},
	"generate": func() {
		txtToHelp(helpGenerate)
	},
	"install": func() {
		stderr(
			`usage: sgo install [target]`,
			`Install installs an executable Scriggo interpreter on system. Output directory is the same used by 'go install' (see 'go help install' for details)`,
			``,
			`See also: sgo generate`,
		)
	},
	"version": func() {
		stderr(
			`usage: sgo version`,
		)
	},
}

// commands maps a command name to a function that executes that command.
// Commands are called by command-line using:
//
//		sgo command
//
var commands = map[string]func(){
	"bug": func() {
		flag.Usage = commandsHelp["bug"]
		panic("TODO: not implemented") // TODO(Gianluca): to implement.
	},
	"install": func() {
		flag.Usage = commandsHelp["install"]
		generate(true)
	},
	"generate": func() {
		flag.Usage = commandsHelp["generate"]
		generate(false)
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
			_, _ = fmt.Fprintf(os.Stderr, "sgo help %s: unknown help topic. Run 'sgo help'.\n", topic)
			exit(1)
			return
		}
		help()
	},
	"version": func() {
		flag.Usage = commandsHelp["version"]
		fmt.Printf("Scriggo module version:            (TODO) \n") // TODO(Gianluca): use real version.
		fmt.Printf("sgo tool version:                  (TODO) \n") // TODO(Gianluca): use real version.
		fmt.Printf("Go version used to build sgo:      %s\n", runtime.Version())
	},
}

// generate executes the sub commands "generate" and "install":
//
//		sgo generate
//		sgo install
//
// If install is set, the interpreter will be installed as executable and
// the interpreter sources will be removed.
func generate(install bool) {

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

	r, err := getScriggofile(inputPath)
	if err != nil {
		exitError(err.Error())
	}
	defer r.Close()

	sf, err := parseScriggofile(r)
	if err != nil {
		exitError("path %q: %s", inputPath, err)
	}
	sf.filepath = inputPath
	if len(sf.comment.goos) == 0 {
		defaultGOOS := os.Getenv("GOOS")
		if defaultGOOS == "" {
			defaultGOOS = runtime.GOOS
		}
		sf.comment.goos = []string{defaultGOOS}
	}

	// Generate an embeddable loader.
	if sf.comment.embedded {
		if install {
			stderr(`sgo install is not compatible with a Scriggo descriptor that generates embedded packages`)
			flag.Usage()
			exit(1)
			return
		}
		if sf.comment.varName == "" {
			sf.comment.varName = "packages"
		}
		inputFileBase := filepath.Base(inputPath)
		inputBaseNoExt := strings.TrimSuffix(inputFileBase, filepath.Ext(inputFileBase))

		// Iterate over all GOOS.
		for _, goos := range sf.comment.goos {

			// Render all packages, ignoring main.
			data, hasContent, err := renderPackages(sf, sf.comment.varName, goos)
			if err != nil {
				exitError("%s", err)
			}

			// Data has been generated but has no content (only has a
			// "skeleton"): do not write file.
			if !hasContent {
				continue
			}

			newBase := inputBaseNoExt + "_" + goBaseVersion(runtime.Version()) + "_" + goos + filepath.Ext(inputFileBase)
			out := filepath.Join(filepath.Dir(inputPath), newBase)

			// Write the packages on a file and run "goimports" on that file.
			err = ioutil.WriteFile(out, []byte(data), filePerm)
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

	// Generate the sources for a new interpreter.
	if sf.comment.template || sf.comment.script || sf.comment.program {

		if sf.comment.output == "" {
			sf.comment.output = strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
		}

		// Create a temporary directory for interpreter sources. If installing,
		// directory will be lost. If generating sources and no errors occurred,
		// tmpDir will be moved to the correct path.
		tmpDir, err := ioutil.TempDir("", "sgo")
		if err != nil {
			exitError(err.Error())
		}
		tmpDir = filepath.Join(tmpDir, sf.pkgName)

		err = os.MkdirAll(tmpDir, dirPerm)
		if err != nil {
			exitError(err.Error())
		}

		for _, goos := range sf.comment.goos {

			sf.pkgName = "main"

			// When making an interpreter that reads only template sources, sf
			// cannot contain only packages.
			if sf.comment.template && !sf.comment.script && !sf.comment.program && !sf.containsMain() && len(sf.imports) > 0 {
				exitError("cannot have packages if making a template interpreter")
			}

			data, hasContent, err := renderPackages(sf, "packages", goos)
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

		// Write the package main on disk and run "goimports" on it.
		mainPath := filepath.Join(tmpDir, "main.go")
		err = ioutil.WriteFile(mainPath, makeInterpreterSource(sf.comment.program, sf.comment.script, sf.comment.template), filePerm)
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

		// Move the interpreter from tmpDir to the correct dir.
		fis, err := ioutil.ReadDir(tmpDir)
		if err != nil {
			exitError(err.Error())
		}
		err = os.MkdirAll(sf.comment.output, dirPerm)
		if err != nil {
			exitError(err.Error())
		}
		for _, fi := range fis {
			if !fi.IsDir() {
				filePath := filepath.Join(tmpDir, fi.Name())
				newFilePath := filepath.Join(sf.comment.output, fi.Name())
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
	}

	return
}
