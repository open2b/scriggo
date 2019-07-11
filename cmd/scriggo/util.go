// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

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
	"runtime"
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

// makeExecutableGoMod makes a 'go.mod' file for creating and installing an
// executable.
func makeExecutableGoMod(name string) []byte {
	out := `module cmd/[moduleName]
	replace scriggo => [scriggoPath]`

	out = strings.ReplaceAll(out, "[moduleName]", name)
	goPaths := strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator))
	if len(goPaths) == 0 {
		panic("empty gopath not supported")
	}
	scriggoPath := filepath.Join(goPaths[0], "src/scriggo")
	out = strings.ReplaceAll(out, "[scriggoPath]", scriggoPath)

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

var uniquePackageNameCache = map[string]string{}

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
// TODO(Gianluca): sf.filepath must be validated before being inserted in a
// comment: 1) check that contains only valid unicode characters 2) check that
// does not contain any "newline" characters.
//
func genHeader(sf *scriggofile, goos string) string {
	return "// Code generated by scriggo command, based on file \"" + sf.filepath + "\". DO NOT EDIT.\n" +
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
	return fmt.Errorf("unkown os %q", os)
}

// checkPackagePath checks that a given package path is valid.
//
// This function must be in sync with the function validPackagePath in the
// file "scriggo/internal/compiler/path".
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
// "scriggo/internal/compiler/path".
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

// isGoStdlibPath reports whether path is a Go standard library package path or not.
func isGoStdlibPath(path string) bool {
	// TODO(Gianluca): find a dynamic way to determine the list of packages of
	// the Go standard library.
	stdlibPaths := []string{
		"archive/tar", "archive/zip", "bufio", "bytes",
		"compress/bzip2", "compress/flate", "compress/gzip",
		"compress/lzw", "compress/zlib", "container/heap",
		"container/list", "container/ring", "context", "crypto",
		"crypto/aes", "crypto/cipher", "crypto/des", "crypto/dsa",
		"crypto/ecdsa", "crypto/ed25519",
		"crypto/ed25519/internal/edwards25519", "crypto/elliptic",
		"crypto/hmac", "crypto/internal/randutil",
		"crypto/internal/subtle", "crypto/md5", "crypto/rand",
		"crypto/rc4", "crypto/rsa", "crypto/sha1", "crypto/sha256",
		"crypto/sha512", "crypto/subtle", "crypto/tls", "crypto/x509",
		"crypto/x509/pkix", "database/sql", "database/sql/driver",
		"debug/dwarf", "debug/elf", "debug/gosym", "debug/macho",
		"debug/pe", "debug/plan9obj", "encoding", "encoding/ascii85",
		"encoding/asn1", "encoding/base32", "encoding/base64",
		"encoding/binary", "encoding/csv", "encoding/gob",
		"encoding/hex", "encoding/json", "encoding/pem",
		"encoding/xml", "errors", "expvar", "flag", "fmt", "go/ast",
		"go/build", "go/constant", "go/doc", "go/format",
		"go/importer", "go/internal/gccgoimporter",
		"go/internal/gcimporter", "go/internal/srcimporter",
		"go/parser", "go/printer", "go/scanner", "go/token",
		"go/types", "hash", "hash/adler32", "hash/crc32", "hash/crc64",
		"hash/fnv", "html", "html/template", "image", "image/color",
		"image/color/palette", "image/draw", "image/gif",
		"image/internal/imageutil", "image/jpeg", "image/png",
		"index/suffixarray", "internal/bytealg", "internal/cpu",
		"internal/fmtsort", "internal/goroot", "internal/goversion",
		"internal/lazyregexp", "internal/lazytemplate",
		"internal/nettrace", "internal/oserror", "internal/poll",
		"internal/race", "internal/reflectlite",
		"internal/singleflight", "internal/syscall/unix",
		"internal/testenv", "internal/testlog", "internal/trace",
		"internal/xcoff", "io", "io/ioutil", "log", "log/syslog",
		"math", "math/big", "math/bits", "math/cmplx", "math/rand",
		"mime", "mime/multipart", "mime/quotedprintable", "net",
		"net/http", "net/http/cgi", "net/http/cookiejar",
		"net/http/fcgi", "net/http/httptest", "net/http/httptrace",
		"net/http/httputil", "net/http/internal", "net/http/pprof",
		"net/internal/socktest", "net/mail", "net/rpc",
		"net/rpc/jsonrpc", "net/smtp", "net/textproto", "net/url",
		"os", "os/exec", "os/signal", "os/signal/internal/pty",
		"os/user", "path", "path/filepath", "plugin", "reflect",
		"regexp", "regexp/syntax", "runtime", "runtime/cgo",
		"runtime/debug", "runtime/internal/atomic",
		"runtime/internal/math", "runtime/internal/sys",
		"runtime/pprof", "runtime/pprof/internal/profile",
		"runtime/race", "runtime/trace", "sort", "strconv", "strings",
		"sync", "sync/atomic", "syscall", "testing",
		"testing/internal/testdeps", "testing/iotest", "testing/quick",
		"text/scanner", "text/tabwriter", "text/template",
		"text/template/parse", "time", "unicode", "unicode/utf16",
		"unicode/utf8", "unsafe",
	}
	for _, stdlibPath := range stdlibPaths {
		if stdlibPath == path {
			return true
		}
	}
	return false
}
