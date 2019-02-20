package main

import (
	"fmt"
	"go/importer"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"unicode"
)

func mapEntry(key, value string) string {
	return fmt.Sprintf("\t\"%s\": %s,\n", key, value)
}

func isExported(name string) bool {
	return unicode.Is(unicode.Lu, []rune(name)[0])
}

// TODO (Gianluca): add support for type aliases.
func zero(obj types.Object, pkgName string) string {
	matches := regexp.MustCompile(`type\s(\S+)(?:\s(.*))?`).FindStringSubmatch(obj.String())
	name := pkgName + filepath.Ext(matches[1])
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

// needsToBeGenerated checks if pkg needs to be generated.
//
// TODO (Gianluca): since imports list can be created manually, this function
// should be obsolete?
func needsToBeGenerated(pkg string) bool {
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

func generateMultiplePackages(pkgs []string, customVariableName string) string {
	s := strings.Builder{}
	s.WriteString("package main\n\n")
	for _, p := range pkgs {
		s.WriteString("import " + `"` + p + `"` + "\n")
	}
	s.WriteString("\n")
	s.WriteString("func init() {")
	s.WriteString("    " + customVariableName + " = map[string]scrigo.Package{")
	for _, p := range pkgs {
		if !needsToBeGenerated(p) {
			continue
		}
		s.WriteString(generatePackage(p))
	}
	s.WriteString("    }")
	s.WriteString("}")
	return s.String()
}

func generatePackage(pkgPath string) string {
	pkg, err := importer.Default().Import(pkgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "importer error: %s\n", err)
		return ""
	}
	pkgBase := filepath.Base(pkgPath)
	var pkgContent string
	for _, name := range pkg.Scope().Names() {
		if !isExported(name) {
			continue
		}
		obj := pkg.Scope().Lookup(name)
		objSign := obj.String()
		objPath := pkgBase + "." + name
		switch {

		// It's a variable.
		case strings.HasPrefix(objSign, "var"):

			pkgContent += mapEntry(name, "&"+objPath)

		// It's a function.
		case strings.HasPrefix(objSign, "func"):

			pkgContent += mapEntry(name, objPath)

		// It's a type definition.
		case strings.HasPrefix(objSign, "type"):

			var elem string
			if s := strings.Split(objSign, " "); len(s) >= 3 && strings.HasPrefix(s[2], "interface") {
				elem = ".Elem()"
			}
			z := zero(obj, pkgBase)
			if z == "" {
				continue
			}
			pkgContent += mapEntry(name, "reflect.TypeOf("+z+")"+elem)
			// imports["reflect"] = ""

		// It's a constant.
		case strings.HasPrefix(objSign, "const"):

			{
				// TODO (Gianluca): this manages the special case MaxUint64.
				// Find a better and general way to do this.
				if pkgPath == "math" && strings.Contains(objPath, "MaxUint64") {
					pkgContent += mapEntry(name, "scrigo.Constant(uint64(math.MaxUint64), nil)")
					continue
				}
				if pkgPath == "hash/crc64" {
					continue
				}
			}
			var t string
			if strings.HasPrefix(obj.Type().String(), "untyped ") {
				t = "nil"
			} else {
				// TODO (Gianluca): constant is untyped, so this should be the
				// type of the constants as specified in package source code.
				t = "nil"
			}
			v := fmt.Sprintf("scrigo.Constant(%s, %s)", objPath, t)
			pkgContent += mapEntry(name, v)

		// Unknown package element.
		default:

			fmt.Fprintf(os.Stderr, "unknown: %s (obj: %s)\n", name, obj.String())
		}
	}

	r := strings.NewReplacer(
		// "[imports]", imports.String(),
		"[pkgContent]", pkgContent,
		"[pkgFullPath]", pkgPath,
		"[pkgName]", pkgBase,
		"[pkgPath]", pkgPath,
		"[version]", "// Go version: "+runtime.Version(),
	)
	const skel string = `

	"[pkgFullPath]": scrigo.Package{

[pkgContent]
},
`
	return r.Replace(skel)
}
