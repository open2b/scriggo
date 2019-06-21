// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	dirPerm  = 0775 // default new directory permission.
	filePerm = 0644 // default new file permission.
)

type pkgDef struct {
	name     string
	filepath string
	imports  []importDef
}

// containsMain indicates if pkgDef contains a "main" package.
func (pd pkgDef) containsMain() bool {
	for _, imp := range pd.imports {
		if imp.main {
			return true
		}
	}
	return false
}

type importDef struct {
	path string
	commentTag
}

type commentTag struct {
	main             bool // declared as "main" package.
	toLower          bool // exported names must be set to lowercase.
	include, exclude []string
}

// parseCommentTag parses a comment tag.
// See function tests for syntax examples.
func parseCommentTag(c string) (commentTag, error) {
	ct := commentTag{}
	c = strings.TrimSpace(c)

	illegal := func(w string) error {
		return fmt.Errorf("illegal word %s", w)
	}

	// c must start with "//"".
	if !strings.HasPrefix(c, "//") {
		panic("comment must start with //")
	}
	c = c[len("//"):]

	// If c does not start with "scriggo:", returns: not a Scriggo directive.
	if !strings.HasPrefix(c, "scriggo:") {
		return ct, nil
	}
	c = c[len("scriggo:"):]

	words := strings.Fields(c)

	// No words after "scriggo:"
	if len(words) == 0 {
		return ct, nil
	}

	// First field can be either 'main' or 'include'/'exclude'.
	switch words[0] {
	case "main":
		ct.main = true
		words = words[1:]
	case "include", "exclude":
		// Do nothing here.
	default:
		return commentTag{}, illegal(words[0])
	}

	// Just one word: returns.
	if len(words) == 0 {
		return ct, nil
	}

	// Second field can be either 'tolower' or 'include'/'exclude'.
	switch words[0] {
	case "include", "exclude":
		// Do nothing here
	case "tolower":
		ct.toLower = true
		words = words[1:]
	default:
		return commentTag{}, illegal(words[0])
	}

	// Two words: returns.
	if len(words) == 0 {
		return ct, nil
	}

	switch words[0] {
	case "include":
		ct.include = words[1:]
	case "exclude":
		ct.exclude = words[1:]
	default:
		// illegal word
	}

	return ct, nil
}

// parseImports returns a list of imports path imported in file filepath. If
// filepath points to a package, the package name is returned, else an empty
// string is returned.
func parseImports(src []byte) (pkgDef, error) {

	// Parses file.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return pkgDef{}, fmt.Errorf("parsing error: %s", err.Error())
	}

	pd := pkgDef{
		name: file.Name.Name,
	}

	// Iterates over imports.
	for _, imp := range file.Imports {

		// Imports must have name "_".
		if imp.Name.Name != "_" {
			return pkgDef{}, fmt.Errorf("import name %q not allowed", imp.Name)
		}

		// Read import path unquoting it.
		id := importDef{}
		id.path, err = strconv.Unquote(imp.Path.Value)
		if err != nil {
			panic(fmt.Errorf("unquoting error: %s", err.Error()))
		}

		if imp.Comment != nil {
			it, err := parseCommentTag("//" + imp.Comment.Text())
			if err != nil {
				return pkgDef{}, fmt.Errorf("error while parsing comment %s", err.Error())
			}
			id.commentTag = it
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

var uniquePackageName_cache = map[string]string{}

// uniquePackageName generates an unique package name for every package path.
func uniquePackageName(pkgPath string) string {

	// Make a list of reserved go keywords.
	var goKeywords = []string{
		"break", "default", "func", "interface", "select", "case", "defer",
		"go", "map", "struct", "chan", "else", "goto", "package",
		"switch", "const", "fallthrough", "if", "range",
		"type", "continue", "for", "import", "return", "var",
	}
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
	for _, goKwd := range goKeywords {
		if goKwd == pkgName {
			pkgName = "_" + pkgName + "_"
		}
	}
	uniquePackageName_cache[pkgName] = pkgPath

	return pkgName
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
