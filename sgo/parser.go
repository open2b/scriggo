// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// scriggoDescriptor represents a descriptor of a Scriggo loader or interpreter.
// A scriggoDescriptor can be read from a file using a parsing function.
type scriggoDescriptor struct {
	pkgName  string // name of the package to be generated.
	filepath string // filepath of the parsed file.
	comment  fileComment
	imports  []importDescriptor // list of imports defined in file.
}

// containsMain reports whether a descriptor contains at least one package
// "main". It ignores all non-main packages contained in the descriptor.
func (descriptor scriggoDescriptor) containsMain() bool {
	for _, imp := range descriptor.imports {
		if imp.comment.main {
			return true
		}
	}
	return false
}

// importDescriptor is a single import descriptor.
// An example import descriptor is:
//
//		import _ "fmt" //scriggo: main uncapitalize export:"Println"
//
type importDescriptor struct {
	path    string
	comment importComment
}

// importComment is a comment of an import descriptor. Import comments are
// built from Scriggo comments, such as:
//
//		//scriggo: main uncapitalize export:"Println"
//
type importComment struct {
	main              bool   // declared as "main" package.
	uncapitalize      bool   // exported names must be set "uncapitalized".
	newPath           string // import as newPath in Scriggo.
	newName           string // use as newName in Scriggo.
	export, notexport []string
}

// fileComment is the comment of a Scriggo descriptor. A file comment can be
// generated from a line as:
//
//  //scriggo: embedded variable:"pkgs" goos:"linux,darwin"
//
// TODO(Gianluca): use output.
type fileComment struct {
	embedded bool     // generating embedded.
	varName  string   // variable name for embedded packages.
	template bool     // generating template interpreter.
	script   bool     // generating script interpreter.
	program  bool     // generating program interpreter.
	output   string   // output path.
	goos     []string // target GOOSs.
}

// isScriggoComment reports whether c is a valid Scriggo comment, that is
// starts with:
//
//   //scriggo:
//
// and returns the comment without "//scriggo:", ready to be parsed.
//
func isScriggoComment(c string) (string, bool) {

	// c must start with "//"".
	if !strings.HasPrefix(c, "//") {
		panic("comment must start with //")
	}
	c = c[len("//"):]

	// If c does not start with "scriggo:", returns: not a Scriggo comment.
	if !strings.HasPrefix(c, "scriggo:") {
		return "", false
	}
	c = c[len("scriggo:"):]
	c = strings.TrimSpace(c)

	if len(c) > 0 && strings.Contains(c[:len(c)-1], "\n") {
		return "", false
	}

	return c, true
}

// parseFileComment parses a file comment.
func parseFileComment(c string) (fileComment, error) {

	c, isScriggoComment := isScriggoComment(c)
	if !isScriggoComment {
		return fileComment{}, nil
	}

	fc := fileComment{}

	opts, kvs, err := parse(c)
	if err != nil {
		return fileComment{}, err
	}
	for i, o := range opts {
		if o == "embedded" {
			fc.embedded = true
			opts = append(opts[:i], opts[i+1:]...)
			break
		}
	}

	for i, o := range opts {
		if o == "interpreter" {
			fc.template = true
			fc.script = true
			fc.program = true
			opts = append(opts[:i], opts[i+1:]...)
			break
		}
	}

	for _, o := range opts {
		if o == "interpreters" {
			return fileComment{}, fmt.Errorf("interpreters must have at least one value; use 'interpreter' for generating a default interpreter")
		}
	}

	if len(opts) > 0 {
		return fileComment{}, fmt.Errorf("bad option %s", opts[0])
	}

	for _, kv := range kvs {
		switch kv.Key {
		case "interpreter":
			return fileComment{}, fmt.Errorf("cannot specify values for interpreters: use 'interpreters' instead")
		case "interpreters":
			if fc.template || fc.script || fc.program {
				return fileComment{}, fmt.Errorf("cannot use interpreter with interpreters")
			}
			for _, v := range kv.Values {
				switch v {
				case "template":
					fc.template = true
				case "script":
					fc.script = true
				case "program":
					fc.program = true
				default:
					return fileComment{}, fmt.Errorf("unknown interpreter %q", v)
				}
			}
		case "variable":
			if len(kv.Values) != 1 {
				return fileComment{}, errors.New("expecting one name as variable name")
			}
			if kv.Values[0] == "" {
				return fileComment{}, errors.New("invalid variable name")
			}
			fc.varName = kv.Values[0]
		case "output":
			if len(kv.Values) != 1 {
				return fileComment{}, errors.New("expecting one path as output")
			}
			if kv.Values[0] == "" {
				return fileComment{}, errors.New("invalid path")
			}
			fc.output = kv.Values[0]
		case "goos":
			for _, v := range kv.Values {
				if v == "" {
					return fileComment{}, errors.New("invalid goos")
				}
			}
			fc.goos = kv.Values
		default:
			return fileComment{}, fmt.Errorf("unknown key %s", kv.Key)
		}
	}

	if fc.varName != "" && (fc.template || fc.script || fc.program) {
		return fileComment{}, fmt.Errorf("cannot use variable with interpreters")
	}

	if fc.embedded && (fc.template || fc.script || fc.program) {
		return fileComment{}, fmt.Errorf("cannot use embedded with interpreters")
	}

	if !(fc.embedded || fc.template || fc.script || fc.program) {
		return fileComment{}, fmt.Errorf("specify what to do") // TODO(Gianluca).
	}

	return fc, nil

}

// parseImportComment parses an import comment.
func parseImportComment(c string) (importComment, error) {

	c, isScriggoComment := isScriggoComment(c)
	if !isScriggoComment {
		return importComment{}, nil
	}

	// Nothing after "scriggo:".
	if len(c) == 0 {
		return importComment{}, nil
	}

	ic := importComment{}

	opts, kvs, err := parse(c)
	if err != nil {
		return importComment{}, err
	}

	// Look for option "main".
	for i, o := range opts {
		if o == "main" {
			ic.main = true
			opts = append(opts[:i], opts[i+1:]...)
			break
		}
	}

	// Look for option "capitalize".
	for i, o := range opts {
		if o == "uncapitalize" {
			ic.uncapitalize = true
			opts = append(opts[:i], opts[i+1:]...)
			break
		}
	}

	if len(opts) > 0 {
		return importComment{}, fmt.Errorf("bad option %s", opts[0])
	}

	for _, kv := range kvs {
		switch kv.Key {
		case "export":
			ic.export = kv.Values
		case "notexport":
			ic.notexport = kv.Values
		case "path":
			if len(kv.Values) != 1 {
				return importComment{}, errors.New("expecting one path as value for key path")
			}
			ic.newPath = kv.Values[0]
			ic.newName = filepath.Base(kv.Values[0])
		default:
			return importComment{}, fmt.Errorf("unknown key %s", kv.Key)
		}
	}

	if len(ic.export) > 0 && len(ic.notexport) > 0 {
		return importComment{}, errors.New("cannot have export and notexport in same import comment")
	}
	if ic.uncapitalize && !ic.main {
		return importComment{}, errors.New("cannot use option uncapitalize without option main")
	}

	return ic, nil
}

// option represents an option in a Scriggo comment.
type option string

// keyValues represents an key-values pair in a Scriggo comment.
type keyValues struct {
	Key    string
	Values []string
}

// parse parses str returning a list of Options and KeyValues.
// parse is a low-level parsing function and should not be used directly.
func parse(str string) ([]option, []keyValues, error) {
	toks, err := tokenize(str)
	if err != nil {
		return nil, nil, err
	}
	if len(toks) == 0 {
		return nil, nil, nil
	}
	waitingForValue := false
	opts := []option{}
	kvs := []keyValues{}
	for i := 0; i < len(toks); i++ {
		if i != len(toks)-1 && toks[i+1] == ":" {
			waitingForValue = true
			kvs = append(kvs, keyValues{Key: toks[i]})
			i++ // jumps colon.
		} else if waitingForValue {
			vs := strings.Split(toks[i], ",")
			for _, v := range vs {
				kvs[len(kvs)-1].Values = append(kvs[len(kvs)-1].Values, strings.TrimSpace(v))
			}
			waitingForValue = false
		} else {
			opts = append(opts, option(toks[i]))
		}
	}
	return opts, kvs, nil
}

// tokenize returns the list of token read from str. tokenize is a low-level
// parsing function and should not be used directly.
func tokenize(str string) ([]string, error) {
	tokens := []string{}
	inQuotes := false
loop:
	for {
		tok := ""
		for _, r := range str {
			switch r {
			case ' ':
				if inQuotes {
					tok += " "
					str = str[1:]
					if len(str) == 0 {
						tokens = append(tokens, tok)
					}
				} else {
					if len(tok) == 0 && len(tokens) > 0 && tokens[len(tokens)-1] == ":" {
						return nil, errors.New("unexpected space after colon")
					}
					if tok != "" {
						tokens = append(tokens, tok)
						tok = ""
					}
					str = str[1:]
					continue loop
				}
			case ':':
				if len(tok) == 0 {
					return nil, errors.New("unexpected colon at beginning of line, expecting word or space")
				}
				if len(tokens) > 0 && tokens[len(tokens)-1] == `"` {
					return nil, errors.New("unexpected colon after quote, expecting word")
				}
				tokens = append(tokens, tok)
				tokens = append(tokens, ":")
				str = str[1:]
				if len(str) == 0 {
					return nil, errors.New("unexpected EOL after colon, expecting quote or word")
				}
				continue loop
			case '"':
				if !inQuotes {
					if len(tok) == 0 && len(tokens) == 0 {
						return nil, errors.New("unexpected quote at beginning of line, expecting word or space")
					}
					if len(tok) > 0 {
						return nil, fmt.Errorf("unexpected quote after %s", tok)
					}
					if len(tokens) > 0 && tokens[len(tokens)-1] != ":" {
						return nil, fmt.Errorf("unexpected quote after %s", tokens[len(tokens)-1])
					}
					str = str[1:]
					inQuotes = true
				} else {
					inQuotes = false
					tokens = append(tokens, tok)
					str = str[1:]
					continue loop
				}
			default:
				tok += string(r)
				str = str[1:]
				if len(str) == 0 {
					tokens = append(tokens, tok)
				}
			}
		}
		if inQuotes {
			return nil, errors.New("unexpected EOL, expecting quote")
		}
		break
	}
	return tokens, nil
}
