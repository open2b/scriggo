// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os/exec"
	"strconv"

	"scriggo/internal/compiler"
	"scriggo/internal/compiler/ast"
)

const (
	dirPerm  = 0775 // default new directory permission.
	filePerm = 0644 // default new file permission.
)

// extractImports returns a list of imports path imported in file filepath. If
// filepath points to a package, the package name is returned, else an empty
// string is returned.
func extractImports(filepath string) ([]string, string, error) {
	src, err := ioutil.ReadFile(filepath)
	if err != nil {
		panic(err)
	}
	tree, _, err := compiler.ParseSource(src, false, false)
	if err != nil {
		panic(err)
	}
	pkgs := []string{}
	if len(tree.Nodes) != 1 {
		return nil, "", errors.New("imports file must be a package definition")
	}
	pkg, ok := tree.Nodes[0].(*ast.Package)
	if !ok {
		return nil, "", errors.New("imports file must be a package definition")
	}
	for _, n := range pkg.Declarations {
		if imp, ok := n.(*ast.Import); ok {
			pkgs = append(pkgs, imp.Path)
		}
	}
	return pkgs, pkg.Name, nil
}

// goImports runs the system command "goimports" on path.
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

// goBuild runs the system command "go build" on path.
func goBuild(path string) error {
	_, err := exec.LookPath("go")
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build")
	cmd.Dir = path
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("go build: %s", stderr.String())
	}
	return nil
}

// goBaseVersion returns the go base version for v.
//
//		1.12.5 -> 1.12
//
func goBaseVersion(v string) string {
	v = v[4:]
	f, err := strconv.ParseFloat(v, 32)
	if err != nil {
		return ""
	}
	f = math.Floor(f)
	next := int(f)
	return fmt.Sprintf("go1.%d", next)
}

// nextGoVersion returns the successive go version of v.
//
//		1.12.5 -> 1.13
//
func nextGoVersion(v string) string {
	v = v[4:]
	f, err := strconv.ParseFloat(v, 32)
	if err != nil {
		return ""
	}
	f = math.Floor(f)
	next := int(f) + 1
	return fmt.Sprintf("go1.%d", next)
}
