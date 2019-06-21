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

// renderPackagesAndMain returns two valid Go sources, where the former contains
// a package loader definition while the latter contains main package
// declarations.
func renderPackagesAndMain(pd pkgDef, sourceFile, pkgsVariableName, goos string) string {

	// Remove main packages from pd; they must be handled externally.
	for i, imp := range pd.imports {
		if imp.main {
			pd.imports = append(pd.imports[:i], pd.imports[i+1:]...)
		}
	}

	explicitImports := ""
	for _, imp := range pd.imports {
		explicitImports += uniquePackageName(imp.path) + ` "` + imp.path + `"` + "\n"
	}

	pkgContent := strings.Builder{}
	for _, imp := range pd.imports {
		out := generatePackage(imp.path, goos)
		pkgContent.WriteString(out)
	}

	commonReplacer := strings.NewReplacer(
		"[generatedWarning]", "// Code generated by scriggo-generate, based on file \""+sourceFile+"\". DO NOT EDIT.",
		"[buildDirectives]", fmt.Sprintf("//+build %s,%s,!%s", goos, goBaseVersion(runtime.Version()), nextGoVersion(runtime.Version())),
		"[pkgName]", pd.name,
		"[explicitImports]", explicitImports,
	)

	pkgsReplacer := strings.NewReplacer(
		"[customVariableName]", pkgsVariableName,
		"[pkgContent]", pkgContent.String(),
	)

	pkgOutput := commonReplacer.Replace(pkgsSkeleton)
	pkgOutput = pkgsReplacer.Replace(pkgOutput)

	return pkgOutput
}

// generatePackage generates package pkgPath.
func generatePackage(pkgPath, goos string) string {

	pkgName, decls, err := goPackageToDeclarations(pkgPath, goos)
	if err != nil {
		panic(err) // TODO(Gianluca).
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

func renderPackageMain(pd pkgDef, goos string) string {
	// TODO(Gianluca): add header to package main.

	mains := []importDef{}
	for _, imp := range pd.imports {
		if imp.main {
			mains = append(mains, imp)
		}
	}
	allMainDecls := map[string]string{}
	for _, imp := range mains {
		_, decls, err := goPackageToDeclarations(imp.path, goos)
		if err != nil {
			panic(err) // TODO(Gianluca).
		}
		if imp.include != nil {
			tmp := map[string]string{}
			for _, name := range imp.include {
				decl, ok := decls[name]
				if !ok {
					panic("doesnt exists!") // TODO(Gianluca).
				}
				tmp[name] = decl
			}
			decls = tmp
		} else if imp.exclude != nil {
			for _, name := range imp.exclude {
				_, ok := decls[name]
				if !ok {
					panic("doesnt exists!") // TODO(Gianluca).
				}
				delete(decls, name)
			}
		}
		if imp.toLower {
			tmp := map[string]string{}
			for name, decl := range decls {
				tmp[strings.ToLower(name)] = decl // TODO(Gianluca): use the real "unexporting" function.
			}
			decls = tmp
		}
		for k, v := range decls {
			_, ok := allMainDecls[k]
			if ok {
				panic("already defined!") // TODO(Gianluca).
			}
			allMainDecls[k] = v
		}
	}
	// Sorts declarations.
	names := make([]string, 0, len(allMainDecls))
	for name := range allMainDecls {
		names = append(names, name)
	}
	sort.Strings(names)

	pkgContent := strings.Builder{}
	for _, name := range names {
		pkgContent.WriteString(`"` + name + `": ` + allMainDecls[name] + ",\n")
	}

	skel := `
		package main

		var Main = scriggo.Package{
			Name: "main",
			Declarations: map[string]interface{}{
				[pkgContent]
			},
		}`

	repl := strings.NewReplacer(
		"[pkgContent]", pkgContent.String(),
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

var pkgsSkeleton = `[generatedWarning]

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
