// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

const (
	dirPerm  = 0775 // default new directory permission.
	filePerm = 0644 // default new file permission.
)

// makeExecutableGoMod makes a 'go.mod' file for creating and installing an
// executable.
func makeExecutableGoMod(path string) []byte {
	out := `module scriggo-[moduleName]
	replace scriggo => [scriggoPath]`

	inputFileBase := filepath.Base(path)
	inputBaseNoExt := strings.TrimSuffix(inputFileBase, filepath.Ext(inputFileBase))
	out = strings.ReplaceAll(out, "[moduleName]", inputBaseNoExt)
	goPaths := strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator))
	if len(goPaths) == 0 {
		panic("empty gopath not supported")
	}
	scriggoPath := filepath.Join(goPaths[0], "src/scriggo")
	out = strings.ReplaceAll(out, "[scriggoPath]", scriggoPath)

	// TODO(Gianluca): executable name must have a prefix like 'scriggo-',
	// otherwise go refuses to build it. Find the reason and make executable
	// name '[moduleName]' (without a prefix).
	fmt.Fprintf(os.Stderr, "module name (and executable) is 'scriggo-%s' (instead of '%s'. See 'scriggo' code (function makeExecutableGoMod) for further details)\n", inputBaseNoExt, inputBaseNoExt)

	return []byte(out)
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

// getScriggofile reads a Scriggo descriptor from path. If path is a
// file ending with ".go", reads such file and returns its content. If path is a
// package path, it reads the file "scriggo.go" located at the root of the
// package and returns its content. If "scriggo.go" does not exists at the root
// of the package, a default one is returned, which includes all declarations
// from package only.
//
//		path/to/file.go                         ->  reads path/to/file.go
//		path/to/package   (with    scriggo.go)  ->  reads path/to/package/scriggo.go
//		path/to/package   (without scriggo.go)  ->  return a default scriggo.go
//
func getScriggofile(path string) (io.ReadCloser, error) {

	// path points to a file.
	if strings.HasPrefix(path, "Scriggofile") {
		return os.Open(path)
	}

	// path points to a package.
	pkgs, err := packages.Load(nil, path)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("package not found. Install it in some way (eg. 'go get %q')", path)
	}
	if len(pkgs) > 1 {
		return nil, errors.New("too many packages matching")
	}
	pkg := pkgs[0]
	for _, p := range pkg.GoFiles {
		base := filepath.Base(p)
		if base == "Scriggofile" {
			return os.Open(p)
		}
	}

	if pkg.Name == "" {
		// TODO(Gianluca): why did not previous check catch this error?
		return nil, fmt.Errorf("package not found. Install it in some way (eg. 'go get %q')", path)
	}

	fmt.Fprintf(os.Stderr, "package %q does not provide a scriggo.go file, generating a default\n", path)

	out := `
INTERPRETER
	
PACKAGE [pkgName]
	
IMPORT [pkgPath]
`

	out = strings.ReplaceAll(out, "[pkgName]", pkg.Name)
	out = strings.ReplaceAll(out, "[pkgPath]", path)

	return ioutil.NopCloser(strings.NewReader(out)), nil
}

// isPredeclaredIdentifier indicates if name is a Go predeclared identifier or
// not.
func isPredeclaredIdentifier(name string) bool {
	list := []string{
		"bool", "byte", "complex64", "complex128", "error", "float32", "float64",
		"int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8",
		"uint16", "uint32", "uint64", "uintptr", "true", "false", "iota",
		"nil", "append", "cap", "close", "complex", "copy", "delete", "imag",
		"len", "make", "new", "panic", "print", "println", "real", "recover",
	}
	for _, pred := range list {
		if pred == name {
			return true
		}
	}
	return false
}

func txtToHelp(s string) {
	s = strings.TrimSpace(s)
	stderr(strings.Split(s, "\n")...)
}

// goImports runs the system command "goimports" on path.
func goImports(path string) error {
	_, err := exec.LookPath("goimports")
	if err != nil {
		stderr("Use 'go get golang.org/x/tools/cmd/goimports' to install 'goimports' on your system")
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

// goInstall runs the system command "go install" on dir.
func goInstall(dir string) error {
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

var uniquePackageNameCache = map[string]string{}

// isGoKeyword returns true if w is a Go keyword as specified by
// https://golang.org/ref/spec#Keywords .
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

	pkgName := filepath.Base(pkgPath)
	done := false
	for !done {
		done = true
		cachePath, ok := uniquePackageNameCache[pkgName]
		if ok && cachePath != pkgPath {
			done = false
			pkgName += "_"
		}
	}
	if isGoKeyword(pkgName) {
		pkgName = "_" + pkgName + "_"
	}
	uniquePackageNameCache[pkgName] = pkgPath

	return pkgName
}

// genHeader generates an header using given parameters.
//
// BUG(Gianluca): devel "gc" versions are currently not supported.
//
// TODO(Gianluca): pd.filepath must be validated before being inserted in a
// comment: 1) check that contains only valid unicode characters 2) check that
// does not contain any "newline" characters.
//
func genHeader(pd *scriggofile, goos string) string {
	return "// Code generated by Scriggo sgo command, based on file \"" + pd.filepath + "\". DO NOT EDIT.\n" +
		fmt.Sprintf("//+build %s,%s,!%s\n\n", goos, goBaseVersion(runtime.Version()), nextGoVersion(runtime.Version()))
}

// goBaseVersion returns the go base version for v.
//
//		1.12.5 -> 1.12
//
func goBaseVersion(v string) string {
	if i := strings.Index(v, "beta"); i >= 0 {
		v = v[:i]
	}
	v = v[4:]
	f, err := strconv.ParseFloat(v, 32)
	if err != nil {
		panic(err)
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
	v = goBaseVersion(v)[4:]
	f, err := strconv.ParseFloat(v, 32)
	if err != nil {
		panic(err)
	}
	f = math.Floor(f)
	next := int(f) + 1
	return fmt.Sprintf("go1.%d", next)
}
