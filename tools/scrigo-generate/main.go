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

	"scrigo/compiler"
	"scrigo/compiler/ast"
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

func usage() string {
	return "scrigo-generate imports-file variable-name"
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		printErrorAndQuit(usage())
	}
	importsFile := flag.Arg(0)
	customVariableName := flag.Arg(1)
	if importsFile == "" || customVariableName == "" {
		printErrorAndQuit(usage())
	}
	src, err := ioutil.ReadFile(importsFile)
	if err != nil {
		panic(err)
	}
	tree, err := compiler.ParseSource(src, ast.ContextNone)
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
			printErrorAndQuit(fmt.Errorf("only imports are allowed in imports file %s", importsFile))
		}
		packages = append(packages, imp.Path)
	}

	out := generatePackages(packages, importsFile, customVariableName, pkg.Name)

	importsFileBase := filepath.Base(importsFile)
	importsFileBaseWithoutExtension := strings.TrimSuffix(importsFileBase, filepath.Ext(importsFileBase))
	newBase := importsFileBaseWithoutExtension + "_generated" + filepath.Ext(importsFileBase)

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
