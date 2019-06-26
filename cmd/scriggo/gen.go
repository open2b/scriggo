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
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/loader"
)

// renderPackages renders a package definition. It also returns a boolean
// indicating if content contains packages or if it's empty.
func renderPackages(pd scriggoDescriptor, pkgsVariableName, goos string) (string, bool, error) {

	// Remove main packages from pd; they must be handled externally.
	tmp := []importDescriptor{}
	for _, imp := range pd.imports {
		if !imp.comment.main {
			tmp = append(tmp, imp)
		}
	}
	pd.imports = tmp

	explicitImports := strings.Builder{}
	for _, imp := range pd.imports {
		uniqueName := uniquePackageName(imp.path)
		if uniqueName != imp.path {
			explicitImports.WriteString(uniqueName + ` "` + imp.path + `"` + "\n")
		} else {
			explicitImports.WriteString(`"` + imp.path + `"` + "\n")
		}
	}

	pkgs := map[string]string{}
	for _, imp := range pd.imports {
		pkgName, decls, err := parseGoPackage(imp.path, goos)
		if err != nil {
			panic(err) // TODO(Gianluca).
		}
		// No declarations at path: move on to next import path.
		if len(decls) == 0 {
			continue
		}
		if len(imp.comment.export) > 0 {
			decls, err = filterIncluding(decls, imp.comment.export)
			if err != nil {
				return "", false, err
			}
		} else if len(imp.comment.notexport) > 0 {
			decls, err = filterExcluding(decls, imp.comment.notexport)
			if err != nil {
				return "", false, err
			}
		}
		// Sorts declarations.
		names := make([]string, 0, len(decls))
		for name := range decls {
			names = append(names, name)
		}
		sort.Strings(names)

		// Writes all declarations.
		pkgContent := strings.Builder{}
		for _, name := range names {
			pkgContent.WriteString(`"` + name + `": ` + decls[name] + ",\n")
		}

		// Determines which import path use: default (the one specified in
		// source, used by Go) or a new one, indicated in Scriggo comments.
		// TODO(Gianluca): check path collision before starting rendering.
		path := imp.path
		if imp.comment.newPath != "" {
			path = imp.comment.newPath
		}

		// Check if import path already exists.
		if _, ok := pkgs[path]; ok {
			return "", false, fmt.Errorf("path collision: %q", path)
		}

		// Determines which package name use: default (the one specified in
		// source, used by Go) or a new one, indicated in Scriggo comment.
		name := pkgName
		if imp.comment.newName != "" {
			name = imp.comment.newName
		}

		out := `{
			Name: "[pkg.Name()]",
			Declarations: map[string]interface{}{
				[pkgContent]
			},
		},`
		out = strings.ReplaceAll(out, "[pkg.Name()]", name)
		out = strings.ReplaceAll(out, "[pkgContent]", pkgContent.String())
		pkgs[path] = out
	}

	// If no packages have been declared, just return.
	if len(pkgs) == 0 {
		return "", false, nil
	}

	allPkgsContent := strings.Builder{}
	for path, content := range pkgs {
		allPkgsContent.WriteString("\n" + `"` + path + `": ` + "" + content)
	}

	// Skeleton for a package group.
	const pkgsSkeleton = `package [pkgName]

		import (
			[explicitImports]
		)

		import . "scriggo"
		import "reflect"

		func init() {
			[customVariableName] = Packages{
				[pkgContent]
			}
		}`

	pkgOutput := strings.NewReplacer(
		"[pkgName]", pd.pkgName,
		"[explicitImports]", explicitImports.String(),
		"[customVariableName]", pkgsVariableName,
		"[pkgContent]", allPkgsContent.String(),
	).Replace(pkgsSkeleton)
	pkgOutput = genHeader(pd, goos) + pkgOutput
	return pkgOutput, true, nil
}

// renderPackageMain renders a package main.
func renderPackageMain(pd scriggoDescriptor, goos string) (string, error) {

	// Filters all imports extracting only those that refer to package main.
	mains := []importDescriptor{}
	for _, imp := range pd.imports {
		if imp.comment.main {
			mains = append(mains, imp)
		}
	}
	allMainDecls := map[string]string{}

	explicitImports := strings.Builder{}
	for _, imp := range pd.imports {
		uniqueName := uniquePackageName(imp.path)
		if uniqueName != imp.path {
			explicitImports.WriteString(uniqueName + ` "` + imp.path + `"` + "\n")
		} else {
			explicitImports.WriteString(`"` + imp.path + `"` + "\n")
		}

	}

	for _, imp := range mains {

		// Parses path.
		_, decls, err := parseGoPackage(imp.path, goos)
		if err != nil {
			return "", err
		}

		// Checks if only certain declarations must be included or excluded.
		if len(imp.comment.export) > 0 {
			decls, err = filterIncluding(decls, imp.comment.export)
			if err != nil {
				return "", err
			}
		} else if len(imp.comment.notexport) > 0 {
			decls, err = filterExcluding(decls, imp.comment.notexport)
			if err != nil {
				return "", err
			}
		}

		// Converts all declaration name to "unexported" if requested.
		if imp.comment.uncapitalize {
			tmp := map[string]string{}
			for name, decl := range decls {
				tmp[uncapitalize(name)] = decl
			}
			decls = tmp
		}

		// Adds parsed declarations to list of declarations of the package main.
		for k, v := range decls {
			_, ok := allMainDecls[k]
			if ok {
				panic("already defined!") // TODO(Gianluca).
			}
			allMainDecls[k] = v
		}
	}

	// Sorts declarations in package main.
	names := make([]string, 0, len(allMainDecls))
	for name := range allMainDecls {
		names = append(names, name)
	}
	sort.Strings(names)

	pkgContent := strings.Builder{}
	for _, name := range names {
		pkgContent.WriteString(`"` + name + `": ` + allMainDecls[name] + ",\n")
	}

	out := `
		package main

		import (
			[explicitImports]
		)

		import . "scriggo"
		import "reflect"

		func init() {
			Main = &scriggo.Package{
				Name: "main",
				Declarations: map[string]interface{}{
					[pkgContent]
				},
			}
		}`
	out = strings.ReplaceAll(out, "[pkgContent]", pkgContent.String())
	out = strings.ReplaceAll(out, "[explicitImports]", explicitImports.String())
	out = genHeader(pd, goos) + out

	return out, nil
}

// parseGoPackage parses pkgPath and returns the package name and a map
// containing the exported declarations.
func parseGoPackage(pkgPath, goos string) (string, map[string]string, error) {
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
					continue
				}
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
