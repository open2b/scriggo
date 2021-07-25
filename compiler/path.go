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

	"github.com/open2b/scriggo/compiler/ast"
)

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
