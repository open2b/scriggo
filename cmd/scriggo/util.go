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
	"math"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

const (
	dirPerm  = 0775 // default new directory permission.
	filePerm = 0644 // default new file permission.
)

type packageDef struct {
	name     string
	filepath string
	imports  []importDef
}

// containsMain indicates if packageDef contains a "main" package.
func (pd packageDef) containsMain() bool {
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
	main              bool   // declared as "main" package.
	uncapitalize      bool   // exported names must be set "uncapitalized".
	newPath           string // import as newPath in Scriggo.
	newName           string // use as newName in Scriggo.
	export, notexport []string
}

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

// parseCommentTag parses a comment tag.
// See function tests for syntax examples.
func parseCommentTag(c string) (commentTag, error) {

	ct := commentTag{}
	c = strings.TrimSpace(c)

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
	c = strings.TrimSpace(c)

	// Nothing after "scriggo:".
	if len(c) == 0 {
		return ct, nil
	}

	switch {
	case strings.HasPrefix(c, "main"):
		ct.main = true
		c = strings.TrimPrefix(c, "main")
		c = strings.TrimSpace(c)
	case strings.HasPrefix(c, "uncapitalize"):
		return commentTag{}, errors.New("cannot use uncapitalize without main")
	case strings.HasPrefix(c, "export") || strings.HasPrefix(c, "notexport") || strings.HasPrefix(c, "path"):
	default:
		return commentTag{}, fmt.Errorf("bad comment tag %s", c)
	}

	switch {
	case strings.HasPrefix(c, "uncapitalize"):
		ct.uncapitalize = true
		c = strings.TrimPrefix(c, "uncapitalize")
		c = strings.TrimSpace(c)
	}

	tag := reflect.StructTag(c)

	// Parses "export" and "notexport" using reflect.StructTag.Get.
	if export := tag.Get("export"); len(strings.TrimSpace(export)) > 0 {
		for _, e := range strings.Split(export, ",") {
			ct.export = append(ct.export, strings.TrimSpace(e))
		}
	}
	if notexport := tag.Get("notexport"); len(strings.TrimSpace(notexport)) > 0 {
		for _, ne := range strings.Split(notexport, ",") {
			ct.notexport = append(ct.notexport, strings.TrimSpace(ne))
		}
	}
	if len(ct.export) > 0 && len(ct.notexport) > 0 {
		return commentTag{}, errors.New("cannot have export and notexport in same import comment")
	}

	// Parses "path", setting package path and name.
	if path := strings.TrimSpace(tag.Get("path")); len(path) > 0 {
		if ct.main {
			return commentTag{}, errors.New("cannot use both main and path")
		}
		ct.newPath = path
		ct.newName = filepath.Base(path)
	}

	return ct, nil
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

// parseImports returns a list of imports path imported in file filepath and the
// package name specified in src.
func parseImports(src []byte) (packageDef, error) {

	// Parses file.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return packageDef{}, fmt.Errorf("parsing error: %s", err.Error())
	}

	pd := packageDef{
		name: file.Name.Name,
	}

	// Iterates over imports.
	for _, imp := range file.Imports {

		// Imports must have name "_".
		if imp.Name.Name != "_" {
			return packageDef{}, fmt.Errorf("import name %q not allowed", imp.Name)
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
				return packageDef{}, fmt.Errorf("error on comment %q: %s", imp.Comment.Text(), err.Error())
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

// genHeader generates an header using given parameters.
//
// BUG(Gianluca): devel "gc" versions are currently not supported.
//
func genHeader(pd packageDef, goos string) string {
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
