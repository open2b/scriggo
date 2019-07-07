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
		"INTERPRETER\nVARIABLE a":               `cannot use variable with interpreters`,
		"INTERPRETER\nEMBEDDED":                 `cannot use both INTERPRETER and EMBEDDED`,
		"EMBEDDED\nINTERPRETER":                 `cannot use both INTERPRETER and EMBEDDED`,
		"INTERPRETER PROGRAM":                   `unexpected INTERPRETER PROGRAM, expecting INTERPRETER FOR PROGRAM`,
		"INTERPRETER plugin":                    `unexpected INTERPRETER plugin, expecting INTERPRETER FOR`,
		"INTERPRETER FOR plugin":                `unexpected plugin after INTERPRETER FOR`,
		"INTERPRETER\nIMPORT a NOT CAPITALIZED": `NOT CAPITALIZED can appear only after 'AS main'`,
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
		"EMBEDDED":                                                   {embedded: true},
		"EMBEDDED\nGOOS linux darwin":                                {embedded: true, goos: []string{"linux", "darwin"}},
		"EMBEDDED\nVARIABLE pkgs":                                    {embedded: true, variable: "pkgs"},
		"INTERPRETER":                                                {template: true, script: true, program: true},
		"INTERPRETER FOR PROGRAM":                                    {program: true},
		"INTERPRETER FOR SCRIPT PROGRAM":                             {script: true, program: true},
		"INTERPRETER FOR SCRIPT TEMPLATE PROGRAM":                    {script: true, program: true, template: true},
		"INTERPRETER FOR SCRIPT":                                     {script: true},
		"INTERPRETER FOR TEMPLATE":                                   {template: true},
		"EMBEDDED\nIMPORT a":                                         {embedded: true, imports: []*importInstruction{{path: "a"}}},
		"EMBEDDED\nIMPORT a AS main":                                 {embedded: true, imports: []*importInstruction{{path: "a", asPath: "main"}}},
		"EMBEDDED\nIMPORT a AS main NOT CAPITALIZED":                 {embedded: true, imports: []*importInstruction{{path: "a", asPath: "main", notCapitalized: true}}},
		"EMBEDDED\nIMPORT a INCLUDING Sleep":                         {embedded: true, imports: []*importInstruction{{path: "a", including: []string{"Sleep"}}}},
		"EMBEDDED\nIMPORT a AS main INCLUDING Sleep":                 {embedded: true, imports: []*importInstruction{{path: "a", asPath: "main", including: []string{"Sleep"}}}},
		"EMBEDDED\nIMPORT a AS main NOT CAPITALIZED INCLUDING Sleep": {embedded: true, imports: []*importInstruction{{path: "a", asPath: "main", notCapitalized: true, including: []string{"Sleep"}}}},
		"EMBEDDED\nIMPORT a INCLUDING Sleep Duration":                {embedded: true, imports: []*importInstruction{{path: "a", including: []string{"Sleep", "Duration"}}}},
		"EMBEDDED\nIMPORT a AS main NOT CAPITALIZED EXCLUDING Sleep": {embedded: true, imports: []*importInstruction{{path: "a", asPath: "main", notCapitalized: true, excluding: []string{"Sleep"}}}},
		"EMBEDDED\nIMPORT a AS test":                                 {embedded: true, imports: []*importInstruction{{path: "a", asPath: "test"}}},
		"EMBEDDED\nIMPORT a AS newpath INCLUDING Sleep":              {embedded: true, imports: []*importInstruction{{path: "a", asPath: "newpath", including: []string{"Sleep"}}}},
		"EMBEDDED\nIMPORT a AS path/to/pkg INCLUDING Sleep":          {embedded: true, imports: []*importInstruction{{path: "a", asPath: "path/to/pkg", including: []string{"Sleep"}}}},
		"EMBEDDED\nIMPORT a AS path/to/test INCLUDING Sleep":         {embedded: true, imports: []*importInstruction{{path: "a", asPath: "path/to/test", including: []string{"Sleep"}}}},
		"EMBEDDED\nIMPORT STANDARD LIBRARY":                          {embedded: true, imports: []*importInstruction{{stdlib: true}}},
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
