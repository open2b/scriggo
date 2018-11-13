//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"
)

// toAbsolutePath combines dir with path to obtain an absolute path.
// dir must be absolute and relative path. The parameters are not validated,
// but an error is returned if the resulting path is outside the root "/".
func toAbsolutePath(dir, path string) (string, error) {
	if !strings.Contains(path, "..") {
		return dir + path, nil
	}
	var b = []byte(dir + path)
	for i := 0; i < len(b); i++ {
		if b[i] == '/' {
			if b[i+1] == '.' && b[i+2] == '.' {
				if i == 0 {
					return "", fmt.Errorf("template: invalid path %q", path)
				}
				s := bytes.LastIndexByte(b[:i], '/')
				b = append(b[:s+1], b[i+4:]...)
				i = s - 1
			}
		}
	}
	return string(b), nil
}

func isValidDirName(name string) bool {
	// must be at least one character long and less than 256
	if name == "" || utf8.RuneCountInString(name) >= 256 {
		return false
	}
	// should not be '.' and must not contain '..'
	if name == "." || strings.Contains(name, "..") {
		return false
	}
	// the first and last character should not be spaces
	if name[0] == ' ' || name[len(name)-1] == ' ' {
		return false
	}
	// must not contain characters from NUL to US and the characters
	// " * / : < > ? \ | DEL
	for _, c := range name {
		if ('\x00' <= c && c <= '\x1f') || c == '\x22' || c == '\x2a' || c == '\x2f' ||
			c == '\x3a' || c == '\x3c' || c == '\x3e' || c == '\x3f' || c == '\x5c' ||
			c == '\x7c' || c == '\x7f' {
			return false
		}
	}
	// it does not have to be a reserved name for Windows
	name = strings.ToLower(name)
	if name == "con" || name == "prn" || name == "aux" || name == "nul" ||
		(len(name) > 3 && name[0:3] == "com" && '0' <= name[3] && name[3] <= '9') ||
		(len(name) > 3 && name[0:3] == "lpt" && '0' <= name[3] && name[3] <= '9') {
		if len(name) == 4 || name[4] == '.' {
			return false
		}
	}
	return true
}

func isValidFileName(name string) bool {
	// it must be at least 3 characters long and less than 256
	var length = utf8.RuneCountInString(name)
	if length <= 2 || length >= 256 {
		return false
	}
	// the first and the last character can not be a point
	if name[0] == '.' || name[len(name)-1] == '.' {
		return false
	}
	// the extension must be present
	var dot = strings.LastIndexByte(name, '.')
	name = strings.ToLower(name)
	var ext = name[dot+1:]
	if strings.IndexByte(ext, '.') >= 0 {
		return false
	}
	// the first and last character should not be spaces
	if name[0] == ' ' || name[len(name)-1] == ' ' {
		return false
	}
	// must not contain characters from NUL to US and characters
	// " * / : < > ? \ | DEL
	for _, c := range name {
		if ('\x00' <= c && c <= '\x1f') || c == '\x22' || c == '\x2a' || c == '\x2f' ||
			c == '\x3a' || c == '\x3c' || c == '\x3e' || c == '\x3f' || c == '\x5c' ||
			c == '\x7c' || c == '\x7f' {
			return false
		}
	}
	// it does not have to be a reserved name for Windows
	if name == "con" || name == "prn" || name == "aux" || name == "nul" ||
		(len(name) > 3 && name[0:3] == "com" && '0' <= name[3] && name[3] <= '9') ||
		(len(name) > 3 && name[0:3] == "lpt" && '0' <= name[3] && name[3] <= '9') {
		if len(name) == 4 || name[4] == '.' {
			return false
		}
	}
	return true
}

// isValidFilePath indicates whether path is valid as an include or show path.
// They are valid paths: '/a', '/a/a', 'a', 'a/a', 'a.a', '../a', 'a/../b'.
// These are invalid paths: '', '/', 'a/', '..', 'a/..'.
func isValidFilePath(path string) bool {
	// must have at least one character and do not end with '/'
	if path == "" || path[len(path)-1] == '/' {
		return false
	}
	// splits the path in the various names
	var names = strings.Split(path, "/")
	// the first names must be directories or '..'
	for i, name := range names[:len(names)-1] {
		// if the first name is empty...
		if i == 0 && name == "" {
			// ...then path starts with '/'
			continue
		}
		if name != ".." && !isValidDirName(name) {
			return false
		}
	}
	// the last name must be a file
	return isValidFileName(names[len(names)-1])
}
