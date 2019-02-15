// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/importer"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const skel string = `[version]

package [pkgName]

[imports]
var Package = scrigo.Package{
[pkgContent]}
`

// imp represents an import. Key is the import path, value, if not empty, is the
// import alias.
type imp map[string]string

func (i imp) String() string {
	str := ""
	for k, v := range i {
		if v == "" {
			str += "import \"" + k + "\"\n"
		} else {
			str += "import " + v + " \"" + k + "\"\n"
		}
	}
	return str
}

// mapPairs: key, value -> "key": value,\n
func mapPair(key, value string) string {
	return fmt.Sprintf("\t\"%s\": %s,\n", key, value)
}

// isExported indicates if name is exported.
func isExported(name string) bool {
	return string(name[0]) == strings.ToUpper(string(name[0]))
}

// TODO (Gianluca): add support for type aliases
func zero(obj types.Object) string {
	matches := regexp.MustCompile(`type\s(\S+)(?:\s(.*))?`).FindStringSubmatch(obj.String())
	name := "original" + filepath.Ext(matches[1])
	typ := matches[2]
	switch {
	case strings.HasPrefix(typ, "[]"), strings.HasPrefix(typ, "map"):
		return fmt.Sprintf("(%s)(nil)", name)
	case strings.HasPrefix(typ, "interface"):
		return fmt.Sprintf("(*%s)(nil)", name)
	case strings.HasPrefix(typ, "struct"):
		return fmt.Sprintf("%s{}", name)
	case strings.HasPrefix(typ, "func"), strings.HasPrefix(typ, "map"):
		return fmt.Sprintf("(%s)(nil)", name)
	}
	switch typ {
	case "int", "int16", "int32", "int64", "int8", "rune",
		"uint", "uint16", "uint32", "uint64", "uint8", "uintptr",
		"float32", "float64", "byte":
		return fmt.Sprintf("%s(%s(0))", name, typ)
	case "bool":
		return "false"
	case "string":
		return `""`
	case "error":
		return "nil"
	}
	return fmt.Sprintf("// not supported: %s", obj.String())
}

func main() {
	imports := make(imp)
	var scrigoPackageName, pkgPath string

	// Checks command line arguments.
	switch len(os.Args) {
	case 3:
		scrigoPackageName = os.Args[2]
		fallthrough
	case 2:
		pkgPath = os.Args[1]
	default:
		fmt.Fprintf(os.Stderr, "Usage: %s package_to_import [scrigo_package_name] \n", os.Args[0])
		os.Exit(1)
	}

	// Imports package through go/importer.
	pkg, err := importer.Default().Import(pkgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}

	// Declares and initialize some useful variables (1/2).
	pkgName := filepath.Base(pkgPath)
	imports[pkgPath] = "original"
	imports["scrigo"] = ""
	var pkgContent string

	for _, name := range pkg.Scope().Names() {

		if !isExported(name) {
			continue
		}

		// Declares and initializes some useful variables (2/2).
		obj := pkg.Scope().Lookup(name)
		sign := obj.String()
		fullName := "original." + name
		typ := obj.Type().String()

		// Cleans type substituting original package name with "original"
		cleanReg := regexp.MustCompile(`(\w+\/)\w`)
		for _, m := range cleanReg.FindAllStringSubmatch(typ, -1) {
			typ = strings.Replace(typ, m[1], "", 1)
		}
		typ = strings.Replace(typ, pkgName, "original", -1)

		switch {

		// It's a variable.
		case strings.HasPrefix(sign, "var"):
			pkgContent += mapPair(name, "&"+fullName)

		// It's a function.
		case strings.HasPrefix(sign, "func"):
			pkgContent += mapPair(name, fullName)

		// It's a type definition.
		case strings.HasPrefix(sign, "type"):
			var elem string
			if s := strings.Split(sign, " "); len(s) >= 3 && strings.HasPrefix(s[2], "interface") {
				elem = ".Elem()"
			}
			pkgContent += mapPair(name, "reflect.TypeOf("+zero(obj)+")"+elem)
			imports["reflect"] = ""

		// It's a constant.
		case strings.HasPrefix(sign, "const"):
			var t string
			if strings.HasPrefix(typ, "untyped ") {
				t = "nil"
			} else {
				// TODO (Gianluca): constant is untyped, so this should be the
				// type of the constants as specified in package source code.
				t = "nil"
			}
			v := fmt.Sprintf("scrigo.Constant(%s, %s)", fullName, t)
			pkgContent += mapPair(name, v)

		// Unknown package element.
		default:
			log.Fatalf("unknown: %s (sign: %s)\n", name, sign)
		}
	}

	if scrigoPackageName != "" {
		pkgName = scrigoPackageName
	}
	r := strings.NewReplacer(
		"[imports]", imports.String(),
		"[pkgContent]", pkgContent,
		"[pkgName]", pkgName,
		"[pkgPath]", pkgPath,
		"[version]", "// Go version: "+runtime.Version(),
	)
	fmt.Println(r.Replace(skel))
}
