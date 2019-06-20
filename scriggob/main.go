// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"scriggo/internal/compiler"
	"scriggo/internal/compiler/ast"
)

func printErrorAndQuit(err interface{}) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

// goImports runs "goimports" on path.
func goImports(path string) error {
	_, err := exec.LookPath("goimports")
	if err != nil {
		return err
	}
	cmd := exec.Command("goimports", "-w", path)
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("goimports: %s", stderr.String())
	}
	return nil
}

func extractImportsFromFile(filepath string) ([]string, string) {
	src, err := ioutil.ReadFile(filepath)
	if err != nil {
		panic(err)
	}
	tree, _, err := compiler.ParseSource(src, false, false)
	if err != nil {
		panic(err)
	}
	packages := []string{}
	if len(tree.Nodes) != 1 {
		printErrorAndQuit("imports file must be a package definition")
	}
	pkg, ok := tree.Nodes[0].(*ast.Package)
	if !ok {
		printErrorAndQuit("imports file must be a package definition")
	}
	for _, n := range pkg.Declarations {
		imp, ok := n.(*ast.Import)
		if !ok {
			// TODO(Gianluca):
			printErrorAndQuit(fmt.Errorf("only imports are allowed in imports file %s", filepath))
		}
		packages = append(packages, imp.Path)
	}
	return packages, pkg.Name
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
	Usage:   scriggob [OPTIONS] MODE INPUT

Available modes
---------------

	MODE = "imports" | "sources" | "build"

		imports
			Generate import file starting from SOURCE putting result into DIR
			Require "-variable-name" argument.
		sources
			Populate DIR with an new intepreter source code which can execute code using the same imports as SOURCE
		build
			Same as "sources" but also build intepreter

Options
-------
	`
		fmt.Fprint(os.Stderr, u)
		flag.PrintDefaults()
	}

	gooses := flag.String("GOOSs", "", "Target GOOSs (separated by commas). Default to current GOOS")
	variableName := flag.String("variable-name", "cd", "Custom variable name. Only effective when running in \"imports\" mode")

	flag.Parse()

	if len(flag.Args()) == 0 {
		commandLineError("a mode must be specified")
	}

	var gooss []string
	if *gooses == "" {
		gooss = []string{os.Getenv("GOOS")}
	} else {
		gooss = strings.Split(*gooses, ",")
	}

	if len(flag.Args()) == 1 {
		commandLineError("INPUT has not been passed as argument")
	}
	importsFile := flag.Arg(1)

	packages, pkgName := extractImportsFromFile(importsFile)
	if pkgName == "" {
		panic("pkg name must be specified by command line")
	}

	switch flag.Arg(0) {
	case "imports":
		if *variableName == "" {
			commandLineError("a custom variable name must be specified when using scriggob in \"imports\" mode")
		}
		for _, goos := range gooss {
			out := generatePackages(packages, importsFile, *variableName, pkgName, goos)
			importsFileBase := filepath.Base(importsFile)
			importsFileBaseWithoutExtension := strings.TrimSuffix(importsFileBase, filepath.Ext(importsFileBase))
			newBase := importsFileBaseWithoutExtension + "_generated" + "_" + goos + filepath.Ext(importsFileBase)
			outPath := filepath.Join(filepath.Dir(importsFile), newBase)
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
		panic("TODO: not implemented") // TODO(Gianluca): to implement.
	case "build":
		panic("TODO: not implemented") // TODO(Gianluca): to implement.
	default:
		commandLineError(fmt.Sprintf("mode %q is not valid", flag.Arg(0)))
	}

}
