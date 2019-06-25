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

// parseImportComment parses an import tag.
func parseImportComment(c string) (importComment, error) {

	// c must start with "//"".
	if !strings.HasPrefix(c, "//") {
		panic("comment must start with //")
	}
	c = c[len("//"):]

	// If c does not start with "scriggo:", returns: not a Scriggo directive.
	if !strings.HasPrefix(c, "scriggo:") {
		return importComment{}, nil
	}
	c = c[len("scriggo:"):]
	c = strings.TrimSpace(c)

	// Nothing after "scriggo:".
	if len(c) == 0 {
		return importComment{}, nil
	}

	ic := importComment{}

	opts, kvs, err := parse(c)
	if err != nil {
		return importComment{}, err
	}

	// Looks for option "main".
	for i, o := range opts {
		if o == "main" {
			ic.main = true
			opts = append(opts[:i], opts[i+1:]...)
			break
		}
	}

	// Looks for option "capitalize".
	for i, o := range opts {
		if o == "uncapitalize" {
			if !ic.main {
				return importComment{}, errors.New("cannot use option uncapitalize without option main")
			}
			ic.uncapitalize = true
			opts = append(opts[:i], opts[i+1:]...)
			break
		}
	}

	if len(opts) > 0 {
		return importComment{}, fmt.Errorf("unknown option %s", opts[0])
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

	return ic, nil
}

// option represents an option in a tag.
type option string

// keyValues represents an key-values pair in a tag.
type keyValues struct {
	Key    string
	Values []string
}

// parse parses str returning a list of Options and KeyValues.
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

// tokenize returns the list of token read from str.
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
