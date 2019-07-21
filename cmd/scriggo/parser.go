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
	"unicode/utf8"
)

type Command int

const (
	commandEmbed Command = iota
	commandGenerate
	commandInstall
)

type Target int

const (
	targetPrograms Target = 1 << (3 - 1 - iota)
	targetScripts
	targetTemplates
)

const targetAll = targetPrograms | targetScripts | targetTemplates

// scriggofile represents the content of a Scriggofile.
type scriggofile struct {
	pkgName  string           // name of the package to be generated.
	target   Target           // target.
	variable string           // variable name for embedded packages.
	output   string           // output path.
	goos     []string         // target GOOSs.
	imports  []*importCommand // list of imports defined in file.
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
func parseScriggofile(src io.Reader, goos string) (*scriggofile, error) {

	sf := scriggofile{
		pkgName:  "main",
		variable: "packages",
	}

	scanner := bufio.NewScanner(src)
	ln := 0

	for scanner.Scan() {

		line := scanner.Text()
		if ln == 0 {
			// Remove UTF-8 BOM.
			line = strings.TrimPrefix(line, "0xEF0xBB0xBF")
		}
		ln++
		if !utf8.ValidString(line) {
			return nil, fmt.Errorf("invalid UTF-8 character at line %d", ln)
		}

		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}

		switch strings.ToUpper(tokens[0]) {
		case "TARGET":
			if len(tokens) == 1 {
				return nil, fmt.Errorf("after %s expecting PROGRAMS, SCRIPTS or TEMPLATES at line %d", tokens[0], ln)
			}
			for _, tok := range tokens[1:] {
				target := strings.ToUpper(tok)
				switch target {
				case "PROGRAMS":
					if sf.target&targetPrograms != 0 {
						return nil, fmt.Errorf("repeated target %s at line %d", target, ln)
					}
					sf.target |= targetPrograms
				case "SCRIPTS":
					if sf.target&targetScripts != 0 {
						return nil, fmt.Errorf("repeated target %s at line %d", target, ln)
					}
					sf.target |= targetScripts
				case "TEMPLATES":
					if sf.target&targetTemplates != 0 {
						return nil, fmt.Errorf("repeated target %s at line %d", target, ln)
					}
					sf.target |= targetTemplates
				default:
					return nil, fmt.Errorf("unexpected %q as TARGET at line %d", tok, ln)
				}
			}
		case "SET":
			if len(tokens) == 1 {
				return nil, fmt.Errorf("expecting VARIABLE or PACKAGE after %s at line %d", tokens[0], ln)
			}
			switch strings.ToUpper(tokens[1]) {
			case "VARIABLE":
				if len(tokens) == 2 {
					return nil, fmt.Errorf("missing variable name at line %d", ln)
				}
				if len(tokens) > 3 {
					return nil, fmt.Errorf("too many variable names at line %d", ln)
				}
				variable := string(tokens[2])
				err := checkIdentifierName(variable)
				if err != nil {
					return nil, err
				}
				sf.variable = variable
			case "PACKAGE":
				if len(tokens) == 2 {
					return nil, fmt.Errorf("missing package name at line %d", ln)
				}
				if len(tokens) > 3 {
					return nil, fmt.Errorf("too many packages names at line %d", ln)
				}
				pkgName := string(tokens[2])
				err := checkIdentifierName(pkgName)
				if err != nil {
					return nil, err
				}
				sf.pkgName = pkgName
			default:
				return nil, fmt.Errorf("unexpected %s %s, expecteding %s VARIABLE or %s PACKAGE at line %d",
					tokens[0], tokens[1], tokens[0], tokens[0], ln)
			}
		case "GOOS":
			if len(tokens) == 1 {
				return nil, fmt.Errorf("missing os after %s at line %d", tokens[0], ln)
			}
			if sf.goos == nil {
				sf.goos = make([]string, 0, len(tokens)-1)
			}
			for _, tok := range tokens[1:] {
				os := string(tok)
				err := checkGOOS(os)
				if err != nil {
					return nil, err
				}
				sf.goos = append(sf.goos, os)
			}
		case "IMPORT":
			if len(tokens) == 1 {
				return nil, fmt.Errorf("missing package path at line %d", ln)
			}
			path := string(tokens[1])
			if len(tokens) > 2 && strings.EqualFold(path, "STANDARD") && strings.EqualFold(tokens[2], "LIBRARY") {
				for _, imp := range sf.imports {
					if imp.stdlib {
						return nil, fmt.Errorf("command %s %s %s is repeated at line %d", tokens[0], tokens[1], tokens[2], ln)
					}
				}
				if len(tokens) > 3 {
					return nil, fmt.Errorf("unexpected %q after %s %s %s at line %d", tokens[3], tokens[0], tokens[1], tokens[2], ln)
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
						return nil, fmt.Errorf("repeated option %s at line %d", tok, ln)
					}
					if len(tokens) == 1 {
						return nil, fmt.Errorf("missing package path after AS at line %d", ln)
					}
					path := string(tokens[1])
					err := checkPackagePath(path)
					if err != nil {
						return nil, err
					}
					if hasStdlibPrefix(path) {
						return nil, fmt.Errorf("invalid path %q (prefix conflicts with Go standard library)", path)
					}
					imp.asPath = path
					parsedAs = true
					tokens = tokens[2:]
				case "INCLUDING":
					if len(tokens) == 1 {
						return nil, fmt.Errorf("missing names after INCLUDING at line %d", ln)
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
						return nil, fmt.Errorf("missing names after EXCLUDING at line %d", ln)
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
							return nil, fmt.Errorf("unexpected %s, expecting %s CAPITALIZED at line %d", tok, tok, ln)
						}
						return nil, fmt.Errorf("unexpected %s at line %d", tok, ln)
					}
					if strings.ToUpper(tokens[1]) != "CAPITALIZED" {
						if imp.asPath == "main" {
							return nil, fmt.Errorf("unexpected %s %s, expecting %s CAPITALIZED at line %d", tok, tokens[1], tok, ln)
						}
						return nil, fmt.Errorf("unexpected %s", tok)
					}
					if imp.asPath != "main" {
						return nil, fmt.Errorf("%s %s can appear only after 'AS main' at line %d", tok, tokens[1], ln)
					}
					imp.notCapitalized = true
					tokens = tokens[2:]
				default:
					return nil, fmt.Errorf("unexpected option %s for IMPORT at line %d", tok, ln)
				}
			}
			sf.imports = append(sf.imports, &imp)
		default:
			return nil, fmt.Errorf("unknown command %s at line %d", tokens[0], ln)
		}

	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(sf.goos) > 0 {
		found := false
		for _, os := range sf.goos {
			if os == goos {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("GOOS %s not supported in Scriggofile", goos)
		}
	}

	// When making an interpreter that reads only template sources, sf
	// cannot contain only packages.
	if sf.target == targetTemplates && len(sf.imports) > 0 {
		for _, imp := range sf.imports {
			if imp.asPath != "main" {
				return nil, fmt.Errorf("cannot have packages if making a template interpreter")
			}
		}
	}

	if sf.target == 0 {
		sf.target = targetAll
	}

	return &sf, nil
}
