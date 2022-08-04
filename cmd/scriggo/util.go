// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error {
	return nil
}

func getOutputFlag(output string) (io.WriteCloser, error) {
	if output == "" {
		return nopCloser{os.Stdout}, nil
	}
	if output == os.DevNull {
		return nil, nil
	}
	dir, file := filepath.Split(output)
	if file == "" {
		exitError("%q cannot be a directory", output)
	}
	if dir != "" {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			return nil, err
		}
	}
	return os.OpenFile(output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
}

// uncapitalize "uncapitalizes" n.
//
//	Name        ->  name
//	DoubleWord  ->  doubleWord
//	AbC         ->  abC
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

var predeclaredIdentifier = []string{
	"bool", "byte", "complex64", "complex128", "error", "float32", "float64",
	"int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8",
	"uint16", "uint32", "uint64", "uintptr", "true", "false", "iota",
	"nil", "append", "cap", "close", "complex", "copy", "delete", "imag",
	"len", "make", "new", "panic", "print", "println", "real", "recover",
}

// isPredeclaredIdentifier reports whether name is a Go predeclared
// identifier.
func isPredeclaredIdentifier(name string) bool {
	for _, pred := range predeclaredIdentifier {
		if name == pred {
			return true
		}
	}
	return false
}

func txtToHelp(s string) {
	s = strings.TrimSpace(s)
	stderr(strings.Split(s, "\n")...)
}

var goKeywords = []string{
	"break", "case", "chan", "const", "continue", "default", "defer", "else",
	"fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map",
	"package", "range", "return", "struct", "select", "switch", "type", "var",
}

// isGoKeyword reports whether a string is a Go keyword.
func isGoKeyword(s string) bool {
	for _, k := range goKeywords {
		if s == k {
			return true
		}
	}
	return false
}

type packageNameCache struct {
	cache map[string]string
}

func newPackageNameCache() packageNameCache {
	return packageNameCache{
		cache: map[string]string{},
	}
}

// packageNameUsed is a simple wrapper that determines if a package name is already in use.
func (u packageNameCache) packageNameUsed(pkgName string) bool {
	for _, v := range u.cache {
		if v == pkgName {
			return true
		}
	}
	return false
}

// uniquePackageName generates an unique package name for every package path,
// this will ensure that even if package names collide we return a valid unique package name.
func (u packageNameCache) uniquePackageName(pkgPath, pkgName string) string {

	//check if the package path has already been resolved
	if cachePath, ok := u.cache[pkgPath]; ok {
		return cachePath //package path to name has already been set
	}

	//check if the package name is available
	if u.packageNameUsed(pkgName) {
		//iterate on the package name until we get a free package name
		i := 2
		for {
			pkgNameTemp := fmt.Sprintf("%s_%d", pkgName, i)
			if !u.packageNameUsed(pkgNameTemp) {
				pkgName = pkgNameTemp
				break
			}
			i++
		}
	}
	u.cache[pkgPath] = pkgName
	return pkgName
}

// goBaseVersion returns the go base version for v.
//
//	1.15.5 -> 1.15
func goBaseVersion(v string) string {
	// When updating, also update test/compare/run.go.
	if strings.HasPrefix(v, "devel ") {
		v = v[len("devel "):]
		if i := strings.Index(v, "-"); i >= 0 {
			v = v[:i]
		}
	}
	if i := strings.Index(v, "beta"); i >= 0 {
		v = v[:i]
	}
	if i := strings.Index(v, "rc"); i >= 0 {
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

// hasStdlibPrefix reports whether the prefix of path conflicts with the path of
// a package of the Go standard library.
func hasStdlibPrefix(path string) bool {
	stdlibPrefixes := []string{
		"archive", "bufio", "bytes", "compress", "container",
		"context", "crypto", "database", "debug", "embed", "encoding",
		"errors", "expvar", "flag", "fmt", "go", "hash", "html", "image",
		"index", "io", "log", "math", "mime", "net", "os",
		"path", "plugin", "reflect", "regexp", "runtime", "sort",
		"strconv", "strings", "sync", "syscall", "testing", "text",
		"time", "unicode", "unsafe",
	}
	first := strings.Split(path, "/")[0]
	for _, pref := range stdlibPrefixes {
		if pref == first {
			return true
		}
	}
	return false
}

// nextGoVersion returns the successive go version of v.
//
//	1.15.5 -> 1.16
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

// checkIdentifierName checks that name is a valid not blank identifier name.
func checkIdentifierName(name string) error {
	if name == "_" {
		return fmt.Errorf("cannot use the blank identifier")
	}
	if isGoKeyword(name) {
		return fmt.Errorf("invalid variable name")
	}
	first := true
	for _, r := range name {
		if !unicode.IsLetter(r) && (first || !unicode.IsDigit(r)) {
			return fmt.Errorf("invalid identifier name")
		}
		first = false
	}
	return nil
}

// checkGOOS checks that os is a valid GOOS value.
func checkGOOS(os string) error {
	switch os {
	case "darwin", "dragonfly", "js", "linux", "android", "solaris",
		"freebsd", "nacl", "netbsd", "openbsd", "plan9", "windows", "aix":
		return nil
	}
	return fmt.Errorf("unknown os %q", os)
}

// checkPackagePath checks that a given package path is valid.
//
// This function must be in sync with the function validPackagePath in the
// file "scriggo/compiler/path".
func checkPackagePath(path string) error {
	if path == "main" {
		return nil
	}
	for _, r := range path {
		if !unicode.In(r, unicode.L, unicode.M, unicode.N, unicode.P, unicode.S) {
			return fmt.Errorf("invalid path path %q", path)
		}
		switch r {
		case '!', '"', '#', '$', '%', '&', '\'', '(', ')', '*', ':', ';', '<',
			'=', '>', '?', '[', '\\', ']', '^', '`', '{', '|', '}', '\uFFFD':
			return fmt.Errorf("invalid path path %q", path)
		}
	}
	if ss := strings.Split(path, "/"); len(ss) > 0 && ss[0] == "internal" {
		return fmt.Errorf("use of internal package %q not allowed", path)
	}
	if cleaned := cleanPath(path); path != cleaned {
		return fmt.Errorf("invalid path path %q", path)
	}
	return nil
}

// checkExportedName checks that name is a valid exported identifier name.
func checkExportedName(name string) error {
	err := checkIdentifierName(name)
	if err != nil {
		return err
	}
	if fc, _ := utf8.DecodeRuneInString(name); !unicode.Is(unicode.Lu, fc) {
		return fmt.Errorf("cannot refer to unexported name %s", name)
	}
	return nil
}

// cleanPath cleans a path and returns the path in its canonical form.
// path must be already a valid path.
//
// This function must be in sync with the function cleanPath in the file
// "scriggo/compiler/path".
func cleanPath(path string) string {
	if !strings.Contains(path, "..") {
		return path
	}
	var b = []byte(path)
	for i := 0; i < len(b); i++ {
		if b[i] == '/' {
			if b[i+1] == '.' && b[i+2] == '.' {
				s := bytes.LastIndexByte(b[:i], '/')
				b = append(b[:s+1], b[i+4:]...)
				i = s - 1
			}
		}
	}
	return string(b)
}
