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
	"strings"
)

const skel string = `package [pkgName]

[imports]
var Package = scrigo.Package {
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

// zero returns the zero for obj and the list of imported packages used in type
// definition.
//
// TODO (Gianluca): additional imports are added for function definitions only,
// but may occur for every type definition.
//
// TODO (Gianluca): find a better implementation of this function.
//
// TODO (Gianluca): type aliases (eg. "type MyInt = int") is currently not
// supported.
//
func zero(obj types.Object) (string, []string) {
	reg := regexp.MustCompile(`type\s(\S+)(?:\s(.*))?`)
	match := reg.FindStringSubmatch(obj.String())
	name := "original" + filepath.Ext(match[1])
	typ := match[2]
	switch {
	case strings.HasPrefix(typ, "[]"), strings.HasPrefix(typ, "map"):
		return fmt.Sprintf("(%s)(nil)", name), []string{}
	case strings.HasPrefix(typ, "interface"):
		return fmt.Sprintf("(*%s)(nil)", name), []string{}
	case strings.HasPrefix(typ, "struct"):
		return fmt.Sprintf("%s{}", name), []string{}
	case strings.HasPrefix(typ, "func"), strings.HasPrefix(typ, "map"):
		{
			reg := regexp.MustCompile(`(?:func)?[\W\*]?(\w+/)`)
			matches := reg.FindAllStringSubmatch(typ, -1)
			for _, m := range matches {
				typ = strings.Replace(typ, m[1], "", 1)
			}
		}
		reg := regexp.MustCompile(`(\w+)\.\w+`)
		importsInFunctionSign := map[string]bool{}
		for _, imp := range reg.FindAllStringSubmatch(typ, -1) {
			for _, pkgImp := range obj.Pkg().Imports() {
				if imp[1] == pkgImp.Name() {
					importsInFunctionSign[pkgImp.Path()] = true
				}
			}
		}
		imports := make([]string, 0, len(importsInFunctionSign))
		for imp := range importsInFunctionSign {
			imports = append(imports, imp)
		}
		return fmt.Sprintf("(%s)(nil)", typ), imports
	}
	switch typ {
	case "int", "int16", "int32", "int64", "int8", "rune",
		"uint", "uint16", "uint32", "uint64", "uint8", "uintptr",
		"float32", "float64", "byte":
		return fmt.Sprintf("%s(%s(0))", name, typ), []string{}
	case "bool":
		return "false", []string{}
	case "string":
		return `""`, []string{}
	case "error":
		return "nil", []string{}
	}
	return fmt.Sprintf("// not supported: %s", obj.String()), []string{}
}

func mapEntry(key, value string) string {
	return fmt.Sprintf("\t\"%s\": %s,\n", key, value)
}

func reflectTypeOf(s string) string {
	return fmt.Sprintf("reflect.TypeOf(%s)", s)
}

func isExported(name string) bool {
	return string(name[0]) == strings.ToUpper(string(name[0]))
}

func main() {
	imports := make(imp)
	var scrigoPackageName, pkgPath string
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
	pkg, err := importer.Default().Import(pkgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}
	pkgName := filepath.Base(pkgPath)
	imports[pkgPath] = "original"
	pkgContent := ""
	for _, name := range pkg.Scope().Names() {
		if isExported(name) {
			obj := pkg.Scope().Lookup(name)
			sign := obj.String()
			fullName := "original." + name
			switch {
			case strings.HasPrefix(sign, "func"):
				pkgContent += mapEntry(name, fullName)
			case strings.HasPrefix(sign, "type"):
				z, imps := zero(obj)
				for _, imp := range imps {
					imports[imp] = ""
				}
				z = strings.Replace(z, pkgName, "original", -1)
				imports["reflect"] = ""
				pkgContent += mapEntry(name, reflectTypeOf(z))
			case strings.HasPrefix(sign, "var"):
				pkgContent += mapEntry(name, "&"+fullName)
			case strings.HasPrefix(sign, "const"):
				pkgContent += mapEntry(name, fullName)
			default:
				log.Fatalf("unknown: %s (sign: %s)\n", name, sign)
			}
		}
	}
	if scrigoPackageName != "" {
		pkgName = scrigoPackageName
	}
	r := strings.NewReplacer(
		"[pkgName]", pkgName,
		"[pkgPath]", pkgPath,
		"[pkgContent]", pkgContent,
		"[imports]", imports.String(),
	)
	fmt.Print(r.Replace(skel))
}
