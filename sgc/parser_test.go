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
		"MAKE INTERPRETER\nSET VARIABLE a":           `cannot use SET VARIABLE with interpreters`,
		"MAKE INTERPRETER\nMAKE EMBEDDED":            `repeated command MAKE`,
		"MAKE EMBEDDED\nMAKE INTERPRETER":            `repeated command MAKE`,
		"MAKE INTERPRETER PROGRAMS":                  `unexpected MAKE INTERPRETER PROGRAMS, expecting MAKE INTERPRETER FOR PROGRAMS`,
		"MAKE INTERPRETER plugin":                    `unexpected MAKE INTERPRETER "plugin", expecting MAKE INTERPRETER FOR`,
		"MAKE INTERPRETER FOR plugin":                `unexpected "plugin" after MAKE INTERPRETER FOR`,
		"MAKE INTERPRETER\nIMPORT a NOT CAPITALIZED": `NOT CAPITALIZED can appear only after 'AS main'`,
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
		"MAKE EMBEDDED": {embedded: true, variable: "packages"},
		"MAKE EMBEDDED\nREQUIRE GOOS linux darwin":                        {embedded: true, goos: []string{"linux", "darwin"}, variable: "packages"},
		"MAKE EMBEDDED\nSET VARIABLE pkgs":                                {embedded: true, variable: "pkgs"},
		"MAKE INTERPRETER":                                                {templates: true, scripts: true, programs: true, variable: "packages"},
		"MAKE INTERPRETER FOR PROGRAMS":                                   {programs: true, variable: "packages"},
		"MAKE INTERPRETER FOR SCRIPTS PROGRAMS":                           {scripts: true, programs: true, variable: "packages"},
		"MAKE INTERPRETER FOR SCRIPTS TEMPLATES PROGRAMS":                 {scripts: true, programs: true, templates: true, variable: "packages"},
		"MAKE INTERPRETER FOR SCRIPTS":                                    {scripts: true, variable: "packages"},
		"MAKE INTERPRETER FOR TEMPLATES":                                  {templates: true, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT a":                                         {embedded: true, imports: []*importCommand{{path: "a"}}, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT a AS main":                                 {embedded: true, imports: []*importCommand{{path: "a", asPath: "main"}}, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT a AS main NOT CAPITALIZED":                 {embedded: true, imports: []*importCommand{{path: "a", asPath: "main", notCapitalized: true}}, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT a INCLUDING Sleep":                         {embedded: true, imports: []*importCommand{{path: "a", including: []string{"Sleep"}}}, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT a AS main INCLUDING Sleep":                 {embedded: true, imports: []*importCommand{{path: "a", asPath: "main", including: []string{"Sleep"}}}, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT a AS main NOT CAPITALIZED INCLUDING Sleep": {embedded: true, imports: []*importCommand{{path: "a", asPath: "main", notCapitalized: true, including: []string{"Sleep"}}}, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT a INCLUDING Sleep Duration":                {embedded: true, imports: []*importCommand{{path: "a", including: []string{"Sleep", "Duration"}}}, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT a AS main NOT CAPITALIZED EXCLUDING Sleep": {embedded: true, imports: []*importCommand{{path: "a", asPath: "main", notCapitalized: true, excluding: []string{"Sleep"}}}, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT a AS test":                                 {embedded: true, imports: []*importCommand{{path: "a", asPath: "test"}}, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT a AS newpath INCLUDING Sleep":              {embedded: true, imports: []*importCommand{{path: "a", asPath: "newpath", including: []string{"Sleep"}}}, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT a AS path/to/pkg INCLUDING Sleep":          {embedded: true, imports: []*importCommand{{path: "a", asPath: "path/to/pkg", including: []string{"Sleep"}}}, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT a AS path/to/test INCLUDING Sleep":         {embedded: true, imports: []*importCommand{{path: "a", asPath: "path/to/test", including: []string{"Sleep"}}}, variable: "packages"},
		"MAKE EMBEDDED\nIMPORT STANDARD LIBRARY":                          {embedded: true, imports: []*importCommand{{stdlib: true}}, variable: "packages"},
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
