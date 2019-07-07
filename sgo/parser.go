// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// scriggofile represents the content of a Scriggofile.
type scriggofile struct {
	pkgName   string           // name of the package to be generated.
	filepath  string           // filepath of the parsed file.
	embedded  bool             // generating embedded.
	programs  bool             // generating program interpreter.
	templates bool             // generating template interpreter.
	scripts   bool             // generating script interpreter.
	variable  string           // variable name for embedded packages.
	output    string           // output path.
	goos      []string         // target GOOSs.
	imports   []*importCommand // list of imports defined in file.
}

// importCommand represents an IMPORT command in a Scriggofile.
type importCommand struct {
	stdlib         bool
	path           string
	asPath         string // import asPath asPath in Scriggo.
	notCapitalized bool   // exported names must not be capitalized.
	including      []string
	excluding      []string
}

// parseScriggofile parses a Scriggofile and returns its commands.
func parseScriggofile(src io.Reader) (*scriggofile, error) {

	sf := scriggofile{}
	scanner := bufio.NewScanner(src)
	ln := 0

	hasMake := false

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

		switch strings.ToUpper(tokens[0]) {
		case "MAKE":
			if hasMake {
				return nil, fmt.Errorf("repeated command MAKE")
			}
			if len(tokens) == 1 {
				return nil, fmt.Errorf("expecting EMBEDDED or INTERPRETER after %s", tokens[0])
			}
			switch strings.ToUpper(tokens[1]) {
			case "EMBEDDED":
				if len(tokens) > 2 {
					return nil, fmt.Errorf("unknown %q after %s %s", tokens[0], tokens[1], tokens[1])
				}
				sf.embedded = true
			case "INTERPRETER":
				if len(tokens) > 2 {
					if tok := strings.ToUpper(tokens[2]); tok != "FOR" {
						switch tok {
						case "PROGRAMS", "SCRIPTS", "TEMPLATES":
							return nil, fmt.Errorf("unexpected %s %s %s, expecting %s %s FOR %s",
								tokens[0], tokens[1], tokens[2], tokens[0], tokens[1], tokens[2])
						}
						return nil, fmt.Errorf("unexpected %s %s %q, expecting %s %s FOR",
							tokens[0], tokens[1], tokens[2], tokens[0], tokens[1])
					}
					if len(tokens) == 3 {
						return nil, fmt.Errorf("expecting PROGRAMS, SCRIPTS or TEMPLATES AFTER %s %s %s",
							tokens[0], tokens[1], tokens[2])
					}
					for _, tok := range tokens[3:] {
						typ := strings.ToUpper(tok)
						switch typ {
						case "PROGRAMS":
							sf.programs = true
						case "SCRIPTS":
							sf.scripts = true
						case "TEMPLATES":
							sf.templates = true
						default:
							return nil, fmt.Errorf("unexpected %q after %s %s %s",
								tok, tokens[0], tokens[1], tokens[2])
						}
					}
				} else {
					sf.programs = true
					sf.scripts = true
					sf.templates = true
				}
			}
			hasMake = true
		case "SET":
			if !hasMake {
				return nil, fmt.Errorf("missing MAKE before %s", tokens[0])
			}
			if len(tokens) == 1 {
				return nil, fmt.Errorf("expecting VARIABLE or PACKAGE after %s", tokens[0])
			}
			switch strings.ToUpper(tokens[1]) {
			case "VARIABLE":
				if !sf.embedded {
					return nil, fmt.Errorf("cannot use SET VARIABLE with interpreters")
				}
				if len(tokens) == 2 {
					return nil, fmt.Errorf("missing variable name")
				}
				if len(tokens) > 3 {
					return nil, fmt.Errorf("too many variable names")
				}
				varName := string(tokens[2])
				err := checkIdentifierName(varName)
				if err != nil {
					return nil, err
				}
				sf.variable = varName
			case "PACKAGE":
				if len(tokens) == 2 {
					return nil, fmt.Errorf("missing package name")
				}
				if len(tokens) > 3 {
					return nil, fmt.Errorf("too many packages names")
				}
				pkgName := string(tokens[1])
				err := checkIdentifierName(pkgName)
				if err != nil {
					return nil, err
				}
				sf.pkgName = pkgName
			default:
				return nil, fmt.Errorf("unexpected %s %s, expecteding %s VARIABLE or %s PACKAGE",
					tokens[0], tokens[1], tokens[0], tokens[0])
			}
		case "REQUIRE":
			if !hasMake {
				return nil, fmt.Errorf("missing MAKE before %s", tokens[0])
			}
			if len(tokens) == 1 {
				return nil, fmt.Errorf("expected GOOS after %s", tokens[0])
			}
			if !strings.EqualFold(tokens[1], "GOOS") {
				return nil, fmt.Errorf("unexpected %s %q, expected %s GOOS", tokens[0], tokens[1], tokens[0])
			}
			if len(tokens) == 2 {
				return nil, fmt.Errorf("missing os after %s %s", tokens[0], tokens[1])
			}
			if sf.goos == nil {
				sf.goos = make([]string, 0, len(tokens)-2)
			}
			for _, tok := range tokens[2:] {
				os := string(tok)
				err := checkGOOS(os)
				if err != nil {
					return nil, err
				}
				sf.goos = append(sf.goos, os)
			}
		case "IMPORT":
			if !hasMake {
				return nil, fmt.Errorf("missing MAKE before %s", tokens[0])
			}
			if len(tokens) == 1 {
				return nil, fmt.Errorf("missing package path")
			}
			path := string(tokens[1])
			if len(tokens) > 2 && strings.EqualFold(path, "STANDARD") && strings.EqualFold(tokens[2], "LIBRARY") {
				for _, imp := range sf.imports {
					if imp.stdlib {
						return nil, fmt.Errorf("command %s %s %s is repeated", tokens[0], tokens[1], tokens[2])
					}
				}
				if len(tokens) > 3 {
					return nil, fmt.Errorf("unexpected %q after %s %s %s", tokens[3], tokens[0], tokens[1], tokens[2])
				}
				sf.imports = append(sf.imports, &importCommand{stdlib: true})
				continue
			} else {
				err := checkPackagePath(path)
				if err != nil {
					return nil, err
				}
			}
			imp := importCommand{path: path}
			parsedAs := false
			tokens = tokens[2:]
			for len(tokens) > 0 {
				switch tok := strings.ToUpper(tokens[0]); tok {
				case "AS":
					if parsedAs {
						return nil, fmt.Errorf("repeated option %s", tok)
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
				case "INCLUDING":
					if len(tokens) == 1 {
						return nil, fmt.Errorf("missing names after INCLUDING")
					}
					imp.including = make([]string, len(tokens)-1)
					for i, tok := range tokens[1:] {
						name := string(tok)
						err := checkExportedName(name)
						if err != nil {
							return nil, err
						}
						imp.including[i] = name
					}
					tokens = nil
				case "EXCLUDING":
					if len(tokens) == 1 {
						return nil, fmt.Errorf("missing names after EXCLUDING")
					}
					imp.excluding = make([]string, len(tokens)-1)
					for i, tok := range tokens[1:] {
						name := string(tok)
						err := checkExportedName(name)
						if err != nil {
							return nil, err
						}
						imp.excluding[i] = name
					}
					tokens = nil
				case "NOT":
					if len(tokens) == 1 {
						if imp.asPath == "main" {
							return nil, fmt.Errorf("unexpected %s, expecting %s CAPITALIZED", tok, tok)
						}
						return nil, fmt.Errorf("unexpected %s", tok)
					}
					if strings.ToUpper(tokens[1]) != "CAPITALIZED" {
						if imp.asPath == "main" {
							return nil, fmt.Errorf("unexpected %s %s, expecting %s CAPITALIZED", tok, tokens[1], tok)
						}
						return nil, fmt.Errorf("unexpected %s", tok)
					}
					if imp.asPath != "main" {
						return nil, fmt.Errorf("%s %s can appear only after 'AS main'", tok, tokens[1])
					}
					imp.notCapitalized = true
					tokens = tokens[2:]
				default:
					return nil, fmt.Errorf("unexpected option %s for IMPORT", tok)
				}
			}
			sf.imports = append(sf.imports, &imp)
		default:
			return nil, fmt.Errorf("unknown command %s", tokens[0])
		}

	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &sf, nil
}
