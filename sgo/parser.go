// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// scriggofile represents the content of a Scriggofile.
type scriggofile struct {
	pkgName  string              // name of the package to be generated.
	filepath string              // filepath of the parsed file.
	embedded bool                // generating embedded.
	program  bool                // generating program interpreter.
	template bool                // generating template interpreter.
	script   bool                // generating script interpreter.
	variable string              // variable name for embedded packages.
	output   string              // output path.
	goos     []string            // target GOOSs.
	imports  []importInstruction // list of imports defined in file.
}

// containsMain reports whether a descriptor contains at least one package
// "main". It ignores all non-main packages contained in the descriptor.
func (file scriggofile) containsMain() bool {
	for _, imp := range file.imports {
		if imp.asPath == "main" {
			return true
		}
	}
	return false
}

// importInstruction represents an IMPORT instruction in a Scriggofile.
type importInstruction struct {
	path          string
	asPath        string // import asPath asPath in Scriggo.
	uncapitalized bool   // exported names must be set "uncapitalized".
	export        []string
	notExport     []string
}

// parseScriggofile parses a Scriggofile and returns its instructions.
func parseScriggofile(src io.Reader) (*scriggofile, error) {

	parsed := map[string]bool{
		"EMBEDDED":    false,
		"INTERPRETER": false,
		"PACKAGE":     false,
		"VARIABLE":    false,
		"GOOS":        false,
	}

	sf := scriggofile{}
	scanner := bufio.NewScanner(src)
	ln := 0

	for scanner.Scan() {

		line := scanner.Text()
		if ln == 0 {
			// Remove UTF-8 BOM.
			line = strings.TrimPrefix(line, "0xEF0xBB0xBF")
		}
		ln++
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}

		instr := strings.ToUpper(tokens[0])
		if parsed[instr] {
			return nil, fmt.Errorf("instruction %s repeated", tokens[0])
		} else {
			parsed[instr] = true
		}

		switch instr {
		case "EMBEDDED":
			if parsed["INTERPRETER"] {
				return nil, fmt.Errorf("cannot use both INTERPRETER and EMBEDDED")
			}
			if len(tokens) > 1 {
				return nil, fmt.Errorf("unknown %q after EMBEDDED", tokens[1])
			}
			sf.embedded = true
		case "INTERPRETER":
			if parsed["EMBEDDED"] {
				return nil, fmt.Errorf("cannot use both INTERPRETER and EMBEDDED")
			}
			if len(tokens) > 1 {
				for _, tok := range tokens[1:] {
					typ := strings.ToUpper(tok)
					switch typ {
					case "PROGRAM":
						sf.program = true
					case "SCRIPT":
						sf.script = true
					case "TEMPLATE":
						sf.template = true
					default:
						return nil, fmt.Errorf("unexpected option %s for INTERPRETER", tok)
					}
				}
			} else {
				sf.program = true
				sf.script = true
				sf.template = true
			}
		case "PACKAGE":
			if !parsed["INTERPRETER"] && !parsed["EMBEDDED"] {
				return nil, fmt.Errorf("missing INTERPRETER or EMBEDDED before %s", tokens[0])
			}
			if len(tokens) == 1 {
				return nil, fmt.Errorf("missing package name")
			}
			if len(tokens) > 2 {
				return nil, fmt.Errorf("too many packages names")
			}
			pkgName := string(tokens[1])
			err := checkIdentifierName(pkgName)
			if err != nil {
				return nil, err
			}
			sf.pkgName = pkgName
		case "VARIABLE":
			if !parsed["INTERPRETER"] && !parsed["EMBEDDED"] {
				return nil, fmt.Errorf("missing EMBEDDED before %s", tokens[0])
			}
			if !sf.embedded {
				return nil, fmt.Errorf("cannot use variable with interpreters")
			}
			if len(tokens) == 1 {
				return nil, fmt.Errorf("missing variable name")
			}
			if len(tokens) > 2 {
				return nil, fmt.Errorf("too many variable names")
			}
			varName := string(tokens[1])
			err := checkIdentifierName(varName)
			if err != nil {
				return nil, err
			}
			sf.variable = varName
		case "GOOS":
			if !parsed["INTERPRETER"] && !parsed["EMBEDDED"] {
				return nil, fmt.Errorf("missing INTERPRETER or EMBEDDED before %s", tokens[0])
			}
			if len(tokens) == 1 {
				return nil, fmt.Errorf("missing os")
			}
			sf.goos = make([]string, len(tokens)-1)
			for i, tok := range tokens[1:] {
				os := string(tok)
				err := checkGOOS(os)
				if err != nil {
					return nil, err
				}
				sf.goos[i] = os
			}
		case "IMPORT":
			if !parsed["INTERPRETER"] && !parsed["EMBEDDED"] {
				return nil, fmt.Errorf("missing INTERPRETER or EMBEDDED before %s", tokens[0])
			}
			if len(tokens) == 1 {
				return nil, fmt.Errorf("missing package path")
			}
			path := string(tokens[1])
			err := checkPackagePath(path)
			if err != nil {
				return nil, err
			}
			imp := importInstruction{path: path}
			parsedAs := false
			tokens = tokens[2:]
			for len(tokens) > 0 {
				switch option := strings.ToUpper(tokens[0]); option {
				case "AS":
					if parsedAs {
						return nil, fmt.Errorf("repeated option %s", option)
					}
					if len(tokens) == 1 {
						return nil, fmt.Errorf("missing package path after AS")
					}
					path := string(tokens[1])
					err := checkPackagePath(path)
					if err != nil {
						return nil, err
					}
					imp.asPath = path
					parsedAs = true
					tokens = tokens[2:]
				case "UNCAPITALIZED":
					if imp.asPath != "main" {
						return nil, fmt.Errorf("%s can appear only after 'AS main'", option)
					}
					imp.uncapitalized = true
					tokens = tokens[1:]
				case "EXPORTING":
					if len(tokens) == 1 {
						return nil, fmt.Errorf("missing export names after EXPORTING")
					}
					imp.export = make([]string, len(tokens)-1)
					for i, tok := range tokens[1:] {
						name := string(tok)
						err := checkExportedName(name)
						if err != nil {
							return nil, err
						}
						imp.export[i] = name
					}
					tokens = nil
				case "NOT":
					if len(tokens) == 1 {
						return nil, fmt.Errorf("unexpected NOT, expecting NOT EXPORTING")
					}
					if strings.ToUpper(tokens[1]) != "EXPORTING" {
						return nil, fmt.Errorf("unexpected NOT %s, expecting NOT EXPORTING", tokens[1])
					}
					if len(tokens) == 2 {
						return nil, fmt.Errorf("missing export names after NOT EXPORTING")
					}
					imp.notExport = make([]string, len(tokens)-2)
					for i, tok := range tokens[2:] {
						name := string(tok)
						err := checkExportedName(name)
						if err != nil {
							return nil, err
						}
						imp.notExport[i] = name
					}
					tokens = nil
				default:
					return nil, fmt.Errorf("unexpected option %s for IMPORT", option)
				}
			}
			sf.imports = append(sf.imports, imp)
		}

	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &sf, nil
}

// checkIdentifierName checks that name is a valid not blank identifier name.
func checkIdentifierName(name string) error {
	if name == "_" {
		return fmt.Errorf("cannot use the blank identifier")
	}
	switch name {
	case "break", "case", "chan", "const", "continue", "default", "defer", "else",
		"fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map",
		"package", "range", "return", "struct", "select", "switch", "type", "var":
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
