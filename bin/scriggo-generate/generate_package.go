// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/constant"
	"go/importer"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/loader"
)

var goKeywords = []string{
	"break", "default", "func", "interface", "select", "case", "defer",
	"go", "map", "struct", "chan", "else", "goto", "package",
	"switch", "const", "fallthrough", "if", "range",
	"type", "continue", "for", "import", "return", "var",
}

var pkgNamesToPkgPaths = map[string]string{}

// uniquePackageName generates an unique package name for every package path.
func uniquePackageName(pkgPath string) string {
	pkgName := filepath.Base(pkgPath)
	done := false
	for !done {
		done = true
		cachePath, ok := pkgNamesToPkgPaths[pkgName]
		if ok && cachePath != pkgPath {
			done = false
			pkgName += "_"
		}
	}
	for _, goKwd := range goKeywords {
		if goKwd == pkgName {
			pkgName = "_" + pkgName + "_"
		}
	}
	pkgNamesToPkgPaths[pkgName] = pkgPath
	return pkgName
}

// goPackageToDeclarations navigates the package pkgPath and returns a map
// containing the exported declarations.
func goPackageToDeclarations(pkgPath string) (map[string]string, error) {
	fmt.Printf("generating package %q...", pkgPath)
	out := make(map[string]string)
	pkgBase := uniquePackageName(pkgPath)
	config := loader.Config{}
	config.Import(pkgPath)
	program, err := config.Load()
	if err != nil {
		return nil, err
	}
	pkgInfo := program.Package(pkgPath)
	for _, v := range pkgInfo.Defs {
		// Include only exported names. It doesn't take into account whether the
		// object is in a local (function) scope or not.
		if v == nil || !v.Exported() {
			continue
		}
		// Include only package-level names.
		if v.Parent() == nil || v.Parent().Parent() != types.Universe {
			continue
		}
		switch v := v.(type) {
		case *types.Const:
			typ := v.Type().String()
			if strings.HasPrefix(typ, "untyped ") {
				typ = "nil"
			} else {
				typ = fmt.Sprintf("reflect.TypeOf(new(%s)).Elem()", typ)
			}
			exact := v.Val().ExactString()
			quoted := strconv.Quote(exact)
			if v.Val().Kind() == constant.String && len(quoted) == len(exact)+4 {
				quoted = "`" + exact + "`"
			}
			out[v.Name()] = fmt.Sprintf("scriggo.ConstLiteral(%v, %s)", typ, quoted)
		case *types.Func:
			if v.Type().(*types.Signature).Recv() == nil {
				out[v.Name()] = fmt.Sprintf("%s.%s", pkgBase, v.Name())
			}
		case *types.Var:
			if !v.Embedded() && !v.IsField() {
				out[v.Name()] = fmt.Sprintf("&%s.%s", pkgBase, v.Name())
			}
		case *types.TypeName:
			out[v.Name()] = fmt.Sprintf("reflect.TypeOf(new(%s.%s)).Elem()", pkgBase, v.Name())

		}
	}
	fmt.Printf("done!\n")
	return out, nil
}

var outputSkeleton = `[generatedWarning]

package [pkgName]

import (
	[explicitImports]
)

import "scriggo"

func init() {
	[customVariableName] = scriggo.Packages{
		[pkgContent]
	}
}
`

// generatePackages generate all packages imported in sourceFile, creating the
// package pkgName and storing them in customVariableName.
func generatePackages(pkgs []string, sourceFile, customVariableName, pkgName string) string {
	explicitImports := ""
	for _, p := range pkgs {
		explicitImports += uniquePackageName(p) + `"` + p + `"` + "\n"
	}

	pkgContent := strings.Builder{}
	for _, p := range pkgs {
		out := generatePackage(p)
		pkgContent.WriteString(out)
	}

	r := strings.NewReplacer(
		"[generatedWarning]", "// Code generated by scriggo-generate, based on file \""+sourceFile+"\". DO NOT EDIT.",
		"[pkgName]", pkgName,
		"[explicitImports]", explicitImports,
		"[customVariableName]", customVariableName,
		"[pkgContent]", pkgContent.String(),
	)
	return r.Replace(outputSkeleton)
}

// generatePackage generates package pkgPath.
func generatePackage(pkgPath string) string {
	pkg, err := importer.Default().Import(pkgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "importer error: %s\n", err)
		return ""
	}

	decls, err := goPackageToDeclarations(pkgPath)
	if err != nil {
		panic(err)
	}

	// Sorts declarations.
	names := make([]string, 0, len(decls))
	for name := range decls {
		names = append(names, name)
	}
	sort.Strings(names)

	pkgContent := strings.Builder{}
	for _, name := range names {
		pkgContent.WriteString(`"` + name + `": ` + decls[name] + ",\n")
	}

	skel := `
		"[pkgPath]": {
			Name: "[pkg.Name()]",
			Declarations: map[string]interface{}{
				[pkgContent]
			},
		},`

	repl := strings.NewReplacer(
		"[pkgPath]", pkgPath,
		"[pkgContent]", pkgContent.String(),
		"[pkg.Name()]", pkg.Name(),
	)

	return repl.Replace(skel)
}
