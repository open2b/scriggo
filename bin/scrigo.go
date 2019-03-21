// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"scrigo"

	"scrigo/ast"
	"scrigo/parser"
)

var packages map[string]*parser.GoPackage

func main() {

	if len(os.Args) != 2 {
		fmt.Printf("usage: %s filename\n", os.Args[0])
		os.Exit(-1)
	}

	file := os.Args[1]
	ext := filepath.Ext(file)
	if ext != ".go" && ext != ".sgo" {
		fmt.Printf("%s: extension must be \".go\" for main packages and \".sgo\" for scripts\n", file)
		os.Exit(-1)
	}

	absFile, err := filepath.Abs(file)
	if err != nil {
		fmt.Printf("%s: %s\n", file, err)
		os.Exit(-1)
	}
	r := parser.DirReader(filepath.Dir(absFile))

	p := parser.New(r, packages, true)
	tree, err := p.Parse(filepath.Base(file), ast.ContextNone)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	pkgs := make(map[string]*scrigo.Package, len(packages))
	for n, pkg := range packages {
		pkgs[n] = &scrigo.Package{Name: pkg.Name, Declarations: pkg.Declarations}
	}

	if ext == ".sgo" {
		err = scrigo.RunScriptTree(tree, nil)
	} else {
		err = scrigo.RunPackageTree(tree, pkgs, p.TypeCheckInfo("/main"))
	}
	if err != nil {
		fmt.Println(err)
	}

}
