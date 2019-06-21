// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/build"
	"go/constant"
	"go/types"
	"path/filepath"
	"runtime"
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

// generatePackages generate all packages imported in sourceFile, creating the
// package pkgName and storing them in customVariableName.
func generatePackages(pd pkgDef, sourceFile, customVariableName, goos string) string {
	explicitImports := ""
	for _, p := range pd.imports {
		explicitImports += uniquePackageName(p.path) + ` "` + p.path + `"` + "\n"
	}

	pkgContent := strings.Builder{}
	for _, p := range pd.imports {
		out := generatePackage(p.path, goos)
		pkgContent.WriteString(out)
	}

	r := strings.NewReplacer(
		"[generatedWarning]", "// Code generated by scriggo-generate, based on file \""+sourceFile+"\". DO NOT EDIT.",
		"[buildDirectives]", fmt.Sprintf("//+build %s,%s,!%s", goos, goBaseVersion(runtime.Version()), nextGoVersion(runtime.Version())),
		"[pkgName]", pd.name,
		"[explicitImports]", explicitImports,
		"[customVariableName]", customVariableName,
		"[pkgContent]", pkgContent.String(),
	)
	return r.Replace(outputSkeleton)
}

// generatePackage generates package pkgPath.
func generatePackage(pkgPath, goos string) string {

	pkgName, decls, err := goPackageToDeclarations(pkgPath, goos)
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
		"[pkg.Name()]", pkgName,
	)

	return repl.Replace(skel)
}

// goPackageToDeclarations navigates the package pkgPath and returns the package
// name and a map containing the exported declarations.
func goPackageToDeclarations(pkgPath, goos string) (string, map[string]string, error) {
	if goos == "" {
		panic("bug: goos is \"\"") // TODO(Gianluca): remove.
	}
	fmt.Printf("generating package %q (GOOS=%q)...", pkgPath, goos)
	out := make(map[string]string)
	pkgBase := uniquePackageName(pkgPath)
	config := loader.Config{
		Build: &build.Default,
	}
	if goos != "" {
		config.Build.GOOS = goos
	}
	config.Import(pkgPath)
	program, err := config.Load()
	if err != nil {
		return "", nil, fmt.Errorf("(GOOS=%q): %s", goos, err)
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
			switch v.Val().Kind() {
			case constant.String, constant.Bool:
				out[v.Name()] = fmt.Sprintf("ConstValue(%s.%s)", pkgBase, v.Name())
				continue
			case constant.Int:
				// Most cases fall here.
				if len(v.Val().ExactString()) < 7 {
					out[v.Name()] = fmt.Sprintf("ConstValue(%s.%s)", pkgBase, v.Name())
				}
				continue
			}
			typ := v.Type().String()
			if strings.HasPrefix(typ, "untyped ") {
				typ = "nil"
			} else {
				if strings.Contains(typ, ".") {
					typ = pkgBase + filepath.Ext(typ)
				}
				typ = fmt.Sprintf("reflect.TypeOf(new(%s)).Elem()", typ)
			}
			exact := v.Val().ExactString()
			quoted := strconv.Quote(exact)
			if v.Val().Kind() == constant.String && len(quoted) == len(exact)+4 {
				quoted = "`" + exact + "`"
			}
			out[v.Name()] = fmt.Sprintf("ConstLiteral(%v, %s)", typ, quoted)
		case *types.Func:
			if v.Type().(*types.Signature).Recv() == nil {
				out[v.Name()] = fmt.Sprintf("%s.%s", pkgBase, v.Name())
			}
		case *types.Var:
			if !v.Embedded() && !v.IsField() {
				out[v.Name()] = fmt.Sprintf("&%s.%s", pkgBase, v.Name())
			}
		case *types.TypeName:
			if ss := strings.Split(v.String(), " "); len(ss) >= 3 && strings.HasPrefix(ss[2], "struct{") {
				out[v.Name()] = fmt.Sprintf("reflect.TypeOf(%s.%s{})", pkgBase, v.Name())
				continue
			}
			out[v.Name()] = fmt.Sprintf("reflect.TypeOf(new(%s.%s)).Elem()", pkgBase, v.Name())

		}
	}
	fmt.Printf("done!\n")
	return pkgInfo.Pkg.Name(), out, nil
}

var outputSkeleton = `[generatedWarning]

[buildDirectives]

package [pkgName]

import (
	[explicitImports]
)

import . "scriggo"
import "reflect"

func init() {
	[customVariableName] = Packages{
		[pkgContent]
	}
}
`
