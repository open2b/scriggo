// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"INTERPRETER\nVARIABLE a":             `cannot use variable with interpreters`,
		"INTERPRETER\nEMBEDDED":               `cannot use both INTERPRETER and EMBEDDED`,
		"EMBEDDED\nINTERPRETER":               `cannot use both INTERPRETER and EMBEDDED`,
		"INTERPRETER plugin":                  `unexpected option plugin for INTERPRETER`,
		"INTERPRETER\nIMPORT a UNCAPITALIZED": `UNCAPITALIZED can appear only after 'AS main'`,
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			_, got := parseScriggofile(strings.NewReader(input))
			if got == nil {
				t.Fatalf("%s: expected error %q, got nothing", input, expected)
			}
			if got.Error() != expected {
				t.Fatalf("%s: expected error %q, got %q", input, expected, got.Error())
			}
		})
	}
}

func TestParse(t *testing.T) {
	cases := map[string]*scriggofile{
		"EMBEDDED":                                                     {comment: fileComment{embedded: true}},
		"EMBEDDED\nGOOS linux darwin":                                  {comment: fileComment{embedded: true, goos: []string{"linux", "darwin"}}},
		"EMBEDDED\nVARIABLE pkgs":                                      {comment: fileComment{embedded: true, varName: "pkgs"}},
		"INTERPRETER":                                                  {comment: fileComment{template: true, script: true, program: true}},
		"INTERPRETER PROGRAM":                                          {comment: fileComment{program: true}},
		"INTERPRETER SCRIPT PROGRAM":                                   {comment: fileComment{script: true, program: true}},
		"INTERPRETER SCRIPT TEMPLATE PROGRAM":                          {comment: fileComment{script: true, program: true, template: true}},
		"INTERPRETER SCRIPT":                                           {comment: fileComment{script: true}},
		"INTERPRETER TEMPLATE":                                         {comment: fileComment{template: true}},
		"EMBEDDED\nIMPORT a":                                           {comment: fileComment{embedded: true}, imports: []importDescriptor{{path: "a"}}},
		"EMBEDDED\nIMPORT a AS main":                                   {comment: fileComment{embedded: true}, imports: []importDescriptor{{path: "a", comment: importComment{main: true}}}},
		"EMBEDDED\nIMPORT a AS main UNCAPITALIZED":                     {comment: fileComment{embedded: true}, imports: []importDescriptor{{"a", importComment{main: true, uncapitalize: true}}}},
		"EMBEDDED\nIMPORT a EXPORTING Sleep":                           {comment: fileComment{embedded: true}, imports: []importDescriptor{{"a", importComment{export: []string{"Sleep"}}}}},
		"EMBEDDED\nIMPORT a AS main EXPORTING Sleep":                   {comment: fileComment{embedded: true}, imports: []importDescriptor{{"a", importComment{main: true, export: []string{"Sleep"}}}}},
		"EMBEDDED\nIMPORT a AS main UNCAPITALIZED EXPORTING Sleep":     {comment: fileComment{embedded: true}, imports: []importDescriptor{{"a", importComment{main: true, uncapitalize: true, export: []string{"Sleep"}}}}},
		"EMBEDDED\nIMPORT a EXPORTING Sleep Duration":                  {comment: fileComment{embedded: true}, imports: []importDescriptor{{"a", importComment{export: []string{"Sleep", "Duration"}}}}},
		"EMBEDDED\nIMPORT a AS main UNCAPITALIZED NOT EXPORTING Sleep": {comment: fileComment{embedded: true}, imports: []importDescriptor{{"a", importComment{main: true, uncapitalize: true, notexport: []string{"Sleep"}}}}},
		"EMBEDDED\nIMPORT a AS test":                                   {comment: fileComment{embedded: true}, imports: []importDescriptor{{"a", importComment{newPath: "test", newName: "test"}}}},
		"EMBEDDED\nIMPORT a AS newpath EXPORTING Sleep":                {comment: fileComment{embedded: true}, imports: []importDescriptor{{"a", importComment{newPath: "newpath", newName: "newpath", export: []string{"Sleep"}}}}},
		"EMBEDDED\nIMPORT a AS path/to/pkg EXPORTING Sleep":            {comment: fileComment{embedded: true}, imports: []importDescriptor{{"a", importComment{newPath: "path/to/pkg", newName: "pkg", export: []string{"Sleep"}}}}},
		"EMBEDDED\nIMPORT a AS path/to/test EXPORTING Sleep":           {comment: fileComment{embedded: true}, imports: []importDescriptor{{"a", importComment{newPath: "path/to/test", newName: "test", export: []string{"Sleep"}}}}},
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			got, err := parseScriggofile(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("input: %s:\nwanted\t%#v\ngot\t\t%#v", input, want, got)
			}
		})
	}
}
