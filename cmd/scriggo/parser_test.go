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
	cases := []struct {
		cmd  Command
		src  string
		want string
	}{
		{commandInstall, "TARGET plugin", `unexpected "plugin" as TARGET at line 1`},
		{commandInstall, "GOOS linux", `GOOS windows not supported in Scriggofile`},
		{commandInstall, "IMPORT a NOT CAPITALIZED", `NOT CAPITALIZED can appear only after 'AS main' at line 1`},
	}
	for _, cas := range cases {
		t.Run(cas.src, func(t *testing.T) {
			_, got := parseScriggofile(strings.NewReader(cas.src), "windows")
			if got == nil {
				t.Fatalf("%s: expected error %q, got nothing", cas.src, cas.want)
			}
			if got.Error() != cas.want {
				t.Fatalf("%s: expected error %q, got %q", cas.src, cas.want, got.Error())
			}
		})
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		cmd  Command
		src  string
		want *scriggofile
	}{
		{commandEmbed, "", &scriggofile{pkgName: "main", target: targetAll, variable: "packages"}},
		{commandEmbed, "GOOS linux darwin", &scriggofile{pkgName: "main", target: targetAll, goos: []string{"linux", "darwin"}, variable: "packages"}},
		{commandEmbed, "SET VARIABLE pkgs", &scriggofile{pkgName: "main", target: targetAll, variable: "pkgs"}},
		{commandInstall, "", &scriggofile{pkgName: "main", target: targetAll, variable: "packages"}},
		{commandGenerate, "TARGET PROGRAMS", &scriggofile{pkgName: "main", target: targetPrograms, variable: "packages"}},
		{commandInstall, "TARGET SCRIPTS PROGRAMS", &scriggofile{pkgName: "main", target: targetPrograms | targetScripts, variable: "packages"}},
		{commandGenerate, "TARGET SCRIPTS TEMPLATES PROGRAMS", &scriggofile{pkgName: "main", target: targetAll, variable: "packages"}},
		{commandInstall, "TARGET SCRIPTS", &scriggofile{pkgName: "main", target: targetScripts, variable: "packages"}},
		{commandGenerate, "TARGET TEMPLATES", &scriggofile{pkgName: "main", target: targetTemplates, variable: "packages"}},
		{commandEmbed, "IMPORT a", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{path: "a"}}, variable: "packages"}},
		{commandEmbed, "IMPORT a AS main", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{path: "a", asPath: "main"}}, variable: "packages"}},
		{commandEmbed, "IMPORT a AS main NOT CAPITALIZED", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{path: "a", asPath: "main", notCapitalized: true}}, variable: "packages"}},
		{commandEmbed, "IMPORT a INCLUDING Sleep", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{path: "a", including: []string{"Sleep"}}}, variable: "packages"}},
		{commandEmbed, "IMPORT a AS main INCLUDING Sleep", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{path: "a", asPath: "main", including: []string{"Sleep"}}}, variable: "packages"}},
		{commandEmbed, "IMPORT a AS main NOT CAPITALIZED INCLUDING Sleep", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{path: "a", asPath: "main", notCapitalized: true, including: []string{"Sleep"}}}, variable: "packages"}},
		{commandEmbed, "IMPORT a INCLUDING Sleep Duration", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{path: "a", including: []string{"Sleep", "Duration"}}}, variable: "packages"}},
		{commandEmbed, "IMPORT a AS main NOT CAPITALIZED EXCLUDING Sleep", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{path: "a", asPath: "main", notCapitalized: true, excluding: []string{"Sleep"}}}, variable: "packages"}},
		{commandEmbed, "IMPORT a AS test", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{path: "a", asPath: "test"}}, variable: "packages"}},
		{commandEmbed, "IMPORT a AS newpath INCLUDING Sleep", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{path: "a", asPath: "newpath", including: []string{"Sleep"}}}, variable: "packages"}},
		{commandEmbed, "IMPORT a AS path/to/pkg INCLUDING Sleep", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{path: "a", asPath: "path/to/pkg", including: []string{"Sleep"}}}, variable: "packages"}},
		{commandEmbed, "IMPORT a AS path/to/test INCLUDING Sleep", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{path: "a", asPath: "path/to/test", including: []string{"Sleep"}}}, variable: "packages"}},
		{commandEmbed, "IMPORT STANDARD LIBRARY", &scriggofile{pkgName: "main", target: targetAll, imports: []*importCommand{{stdlib: true}}, variable: "packages"}},
	}
	for _, cas := range cases {
		t.Run(cas.src, func(t *testing.T) {
			got, err := parseScriggofile(strings.NewReader(cas.src), "linux")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, cas.want) {
				t.Fatalf("input: %s:\nwanted\t%#v\ngot\t\t%#v", cas.src, cas.want, got)
			}
		})
	}
	// Generate and install.
}
