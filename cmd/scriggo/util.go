// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"math"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"unicode"
)

const (
	dirPerm  = 0775 // default new directory permission.
	filePerm = 0644 // default new file permission.
)

// uncapitalize "uncapitalizes" n.
//
// 	Name        ->  name
//	DoubleWord  ->  doubleWord
//  AbC         ->  abC
//
func uncapitalize(n string) string {
	isUp := unicode.IsUpper
	toLow := unicode.ToLower
	if n == "" {
		return n
	}
	runes := []rune(n)
	if len(runes) == 1 {
		return string(toLow(runes[0]))
	}
	if !isUp(runes[0]) {
		return n
	}
	b := bytes.Buffer{}
	b.Grow(len(n))
	var i int
	b.WriteRune(toLow(runes[0]))
	for i = 1; i < len(runes)-1; i++ {
		if isUp(runes[i]) && isUp(runes[i+1]) {
			b.WriteRune(toLow(runes[i]))
		} else {
			break
		}
	}
	for ; i < len(runes)-1; i++ {
		b.WriteRune(runes[i])
	}
	if isUp(runes[i]) && isUp(runes[i-1]) {
		b.WriteRune(toLow(runes[i]))
	} else {
		b.WriteRune(runes[i])
	}
	return b.String()
}

// filterIncluding filters decls including only names specified in include.
// Returns a copy of the map.
func filterIncluding(decls map[string]string, include []string) (map[string]string, error) {
	tmp := map[string]string{}
	for _, name := range include {
		decl, ok := decls[name]
		if !ok {
			return nil, fmt.Errorf("cannot include declaration %s: doesn't exist.", name)
		}
		tmp[name] = decl
	}
	return tmp, nil
}

// filterExcluding filters decls excluding all names specified in include.
// Returns a copy of the map.
func filterExcluding(decls map[string]string, exclude []string) (map[string]string, error) {
	tmp := map[string]string{}
	for k, v := range decls {
		tmp[k] = v
	}
	for _, name := range exclude {
		_, ok := tmp[name]
		if !ok {
			return nil, fmt.Errorf("cannot exclude declaration %s: doesn't exist.", name)
		}
		delete(tmp, name)
	}
	return tmp, nil
}

// parseScriggoDescriptor returns a list of imports path imported in file
// filepath and the package name specified in src.
func parseScriggoDescriptor(src []byte) (scriggoDescriptor, error) {

	// Parses file.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return scriggoDescriptor{}, fmt.Errorf("parsing error: %s", err.Error())
	}

	pos := func(pos token.Pos) token.Position {
		return fset.File(pos).Position(pos)
	}

	pd := scriggoDescriptor{
		pkgName: file.Name.Name,
	}

	found := false
	for _, c := range file.Comments {
		if pos(c.Pos()).Column == 1 {
			if _, ok := isScriggoComment("//" + c.Text()); ok {
				if found {
					return scriggoDescriptor{}, fmt.Errorf("line: %d: just one Scriggo comment is allowed per file", pos(c.Pos()).Line)
				}
				pd.comment, err = parseFileComment("//" + c.Text())
				if err != nil {
					return scriggoDescriptor{}, fmt.Errorf("line %d: %s", pos(c.Pos()).Line, err)
				}
				found = true
			}
		}
	}
	if !found {
		return scriggoDescriptor{}, errors.New("missing Scriggo file comment")
	}

	// Iterates over imports.
	for _, imp := range file.Imports {

		// Imports must have name "_".
		if imp.Name.Name != "_" {
			return scriggoDescriptor{}, fmt.Errorf("import name %q not allowed", imp.Name)
		}

		// Read import path unquoting it.
		id := importDescriptor{}
		id.path, err = strconv.Unquote(imp.Path.Value)
		if err != nil {
			panic(fmt.Errorf("unquoting error: %s", err.Error()))
		}

		if imp.Comment != nil {
			it, err := parseImportComment("//" + imp.Comment.Text())
			if err != nil {
				return scriggoDescriptor{}, fmt.Errorf("error on comment %q: %s", imp.Comment.Text(), err.Error())
			}
			id.comment = it
		}

		pd.imports = append(pd.imports, id)
	}

	return pd, nil
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

func goInstall(dir string) error {
	log.Printf("█ scriggo install is currently supported only inside GOPATH █") // TODO(Gianluca): remove.
	_, err := exec.LookPath("go")
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "install")
	cmd.Dir = dir
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("go build: %s", stderr.String())
	}
	return nil
}

// goBuild runs the system command "go build" on path.
// TODO(Gianluca): obsolete, remove.
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

var uniquePackageName_cache = map[string]string{}

func isGoKeyword(w string) bool {
	goKeywords := []string{
		"break", "default", "func", "interface", "select", "case", "defer",
		"go", "map", "struct", "chan", "else", "goto", "package",
		"switch", "const", "fallthrough", "if", "range",
		"type", "continue", "for", "import", "return", "var",
	}
	for _, gw := range goKeywords {
		if w == gw {
			return true
		}
	}
	return false
}

// uniquePackageName generates an unique package name for every package path.
func uniquePackageName(pkgPath string) string {

	// Make a list of reserved go keywords.

	pkgName := filepath.Base(pkgPath)
	done := false
	for !done {
		done = true
		cachePath, ok := uniquePackageName_cache[pkgName]
		if ok && cachePath != pkgPath {
			done = false
			pkgName += "_"
		}
	}
	if isGoKeyword(pkgName) {
		pkgName = "_" + pkgName + "_"
	}
	uniquePackageName_cache[pkgName] = pkgPath

	return pkgName
}

// genHeader generates an header using given parameters.
//
// BUG(Gianluca): devel "gc" versions are currently not supported.
//
func genHeader(pd scriggoDescriptor, goos string) string {
	return "// Code generated by scriggo-generate, based on file \"" + pd.filepath + "\". DO NOT EDIT.\n" +
		fmt.Sprintf("//+build %s,%s,!%s\n\n", goos, goBaseVersion(runtime.Version()), nextGoVersion(runtime.Version()))
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
