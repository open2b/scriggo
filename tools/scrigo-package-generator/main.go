// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/importer"
	"log"
	"os"
	"strings"
)

func isExported(name string) bool {
	return string(name[0]) == strings.ToUpper(string(name[0]))
}

const skel string = `
// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import "[pkgName]"
[reflectImport]
var _[pkgName] = Package {
[pkgContent]}
`

func mapEntry(key, value string) string {
	return fmt.Sprintf("\t\"%s\": %s,\n", key, value)
}

func reflectTypeOf(s string, isInterface bool) string {
	if isInterface {
		return fmt.Sprintf("reflect.TypeOf((*%s)(nil)).Elem()", s)
	}
	return fmt.Sprintf("reflect.TypeOf(%s{})", s)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s package_name\n", os.Args[0])
		os.Exit(1)
	}
	pkgName := os.Args[1]
	pkg, err := importer.Default().Import(pkgName)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}
	usingReflect := false
	pkgContent := ""
	for _, name := range pkg.Scope().Names() {
		if isExported(name) {
			obj := pkg.Scope().Lookup(name)
			sign := obj.String()
			fullName := pkgName + "." + name
			switch {
			case strings.HasPrefix(sign, "func"):
				pkgContent += mapEntry(name, fullName)
			case strings.HasPrefix(sign, "type"):
				usingReflect = true
				isInterface := false
				if strings.Contains(sign, "interface") {
					isInterface = true
				}
				pkgContent += mapEntry(name, reflectTypeOf(fullName, isInterface))
			case strings.HasPrefix(sign, "var"):
				pkgContent += mapEntry(name, "&"+fullName)
			case strings.HasPrefix(sign, "const"):
				pkgContent += mapEntry(name, fullName)
			default:
				log.Fatalf("unknown: %s (sign: %s)\n", name, sign)
			}
		}
	}
	skel := strings.Replace(skel, "[pkgName]", pkgName, -1)
	skel = strings.Replace(skel, "[pkgContent]", pkgContent, -1)
	if usingReflect && pkgName != "reflect" {
		skel = strings.Replace(skel, "[reflectImport]", "import \"reflect\"\n", -1)
	} else {
		skel = strings.Replace(skel, "[reflectImport]", "", -1)
	}
	fmt.Printf(skel)
}
