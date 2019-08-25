// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"scriggo/ast"
)

// validatePackagePath validates path at the position pos and panics if path
// is not a valid package path.
func validatePackagePath(path string, pos *ast.Position) {
	if path == "main" {
		return
	}
	if !ValidPath(path) {
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
	return
}

// ValidPath indicates whether path is valid path for Extends, Import and
// Include paths.
func ValidPath(path string) bool {
	return utf8.ValidString(path) &&
		path != "" && path != ".." &&
		path[len(path)-1] != '/' &&
		!strings.Contains(path, "//") &&
		!strings.HasSuffix(path, "/..")
}

// toAbsolutePath combines dir with path to obtain an absolute path.
// dir must be absolute and path must be relative. The parameters are not
// validated, but an error is returned if the resulting path is outside
// the root "/".
func toAbsolutePath(dir, path string) (string, error) {
	if !strings.Contains(path, "..") {
		return dir + path, nil
	}
	var b = []byte(dir + path)
	for i := 0; i < len(b); i++ {
		if b[i] == '/' {
			if b[i+1] == '.' && b[i+2] == '.' {
				if i == 0 {
					return "", fmt.Errorf("scriggo: invalid path %q", path)
				}
				s := bytes.LastIndexByte(b[:i], '/')
				b = append(b[:s+1], b[i+4:]...)
				i = s - 1
			}
		}
	}
	return string(b), nil
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
