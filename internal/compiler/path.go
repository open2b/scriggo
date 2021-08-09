// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"io/fs"
	"strings"
	"unicode"

	"github.com/open2b/scriggo/ast"
)

// validModulePath reports whether path is a valid module path.
// See https://golang.org/ref/mod#go-mod-file-ident
func validModulePath(path string) bool {
	if len(path) == 0 || path[0] == '-' {
		return false
	}
	i := 0 // first element index
	j := 0
	for {
		for j < len(path) {
			c := path[j]
			if c == '/' {
				break
			}
			if c < '-' || '9' < c && c < 'A' || 'Z' < c && c != '_' && c < 'a' || 'z' < c && c != '~' {
				return false
			}
			j++
		}
		if i == j || path[i] == '.' || path[j-1] == '.' || specialWindowsName(path[i:j]) {
			return false
		}
		if j == len(path) {
			return true
		}
		j++
		i = j
	}
}

// specialWindowsName reports whether name is a special Windows name that
// cannot be used as an element of module path. name is special if it is
// similar to a Windows short-name or if it is a reserved Windows name.
func specialWindowsName(name string) bool {
	// Shorten the element to the first dot.
	if h := strings.Index(name, "."); h >= 0 {
		name = name[:h]
	}
	// Check if name is similar to a Windows short-name.
	if h := strings.LastIndex(name, "~"); 0 <= h && h != len(name)-1 {
		for i := len(name) - 1; i > h; i-- {
			if c := name[i]; c < '0' || c > '9' {
				return false
			}
		}
		return true
	}
	// Check if name is a reserved Windows name.
	// See https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file
	reserved := "CON PRN AUX NUL"
	if len(name) == 4 {
		if n := name[3]; n < '0' || n > '9' {
			return false
		}
		name = name[:3]
		reserved = "COM LPT"
	}
	if len(name) == 3 {
		for i := 0; i < len(reserved); i += 4 {
			if strings.EqualFold(name, reserved[i:i+3]) {
				return true
			}
		}
	}
	return false
}

// validatePackagePath validates path at the position pos and panics if path
// is not a valid package path.
func validatePackagePath(path string, pos *ast.Position) {
	if path == "main" {
		return
	}
	if !ValidTemplatePath(path) {
		panic(syntaxError(pos, "invalid import path: %q", path))
	}
	for _, r := range path {
		if !unicode.In(r, unicode.L, unicode.M, unicode.N, unicode.P, unicode.S) {
			panic(syntaxError(pos, "invalid import path: %q", path))
		}
		switch r {
		case '!', '"', '#', '$', '%', '&', '\'', '(', ')', '*', ':', ';', '<',
			'=', '>', '?', '[', '\\', ']', '^', '`', '{', '|', '}', '\uFFFD':
			panic(syntaxError(pos, "invalid import path: %q", path))
		}
	}
	if cleaned := cleanPath(path); path != cleaned {
		panic(syntaxError(pos, "non-canonical import path %q (should be %q)", path, cleaned))
	}
}

// ValidTemplatePath indicates whether path is a valid template path. A valid
// template path is a valid file system path but "." is not valid and it can
// start with '/' or alternatively can start with one o more ".." elements.
func ValidTemplatePath(path string) bool {
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	} else {
		for strings.HasPrefix(path, "../") {
			path = path[3:]
		}
	}
	if path == "." {
		return false
	}
	return fs.ValidPath(path)
}

// cleanPath cleans a path and returns the path in its canonical form.
// path must be already a valid path.
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
