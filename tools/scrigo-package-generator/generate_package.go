// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/importer"
	"go/types"
	"io"
	"io/ioutil"
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
	fmt.Fprintf(os.Stdout, "not supported: %s\n", obj)
	return ""
}

func isValidPkg(pkg string) bool {
	if strings.TrimSpace(pkg) == "" {
		return false
	}
	if pkg == "log/syslog" && runtime.GOOS == "windows" {
		return false
	}
	if pkg == "runtime/race" || pkg == "runtime" {
		return false
	}
	parts := strings.Split(pkg, "/")
	if len(parts) > 0 {
		switch parts[0] {
		case "database", "cmd", "builtin", "debug", "plugin", "testing", "reflect", "unsafe", "syscall":
			return false
		}
		if parts[len(parts)-1] == "cgo" {
			return false
		}
		for _, p := range parts {
			if p == "internal" {
				return false
			}
		}
	}
	return true
}

func generateMultiplePackages(pkgs []string, dir string) {
	// Checks that dir is valid and empty.
	{
		if dir == "" {
			error("dir cannot be nil")
		}
		fi, err := ioutil.ReadDir(dir)
		if err != nil {
			error(err)
		}
		if len(fi) > 0 {
			error("dir must be empty")
		}
	}
	for _, p := range pkgs {
		if !isValidPkg(p) {
			continue
		}
		targetDir := filepath.Clean(dir + "/" + p)
		err := os.MkdirAll(targetDir, 0755)
		if err != nil {
			error(err)
		}
		targetFile, err := os.Create(filepath.Join(targetDir, filepath.Base(p)+".go"))
		if err != nil {
			error(err)
		}
		generatePackage(targetFile, p, "")
	}
}

func generatePackage(w io.Writer, pkgPath string, scrigoPackageName string) {
	imports := make(imp)

	// Imports package through go/importer.
	pkg, err := importer.Default().Import(pkgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "importer error: %s\n", err)
		return
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
			z := zero(obj)
			if z == "" {
				continue
			}
			pkgContent += mapPair(name, "reflect.TypeOf("+z+")"+elem)
			imports["reflect"] = ""

		// It's a constant.
		case strings.HasPrefix(sign, "const"):
			{
				// TODO (Gianluca): this manages the special case MaxUint64.
				// Find a better and general way to do this.
				if pkgPath == "math" && strings.Contains(fullName, "MaxUint64") {
					pkgContent += mapPair(name, "scrigo.Constant(uint64(original.MaxUint64), nil)")
				}
				continue
			}
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
			fmt.Fprintf(os.Stderr, "unknown: %s (obj: %s)\n", name, obj.String())
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
	_, err = w.Write([]byte(r.Replace(skel)))
	if err != nil {
		error(err)
	}
}
