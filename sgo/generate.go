// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"go/constant"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// renderPackages renders a Scriggo descriptor. It also returns a boolean
// indicating if the content contains packages. Ignores all main packages
// contained in the descriptor.
func renderPackages(descriptor *scriggofile, pkgsVariable, goos string) (string, bool, error) {

	type packageType struct {
		name string
		decl map[string]string
	}

	explicitImports := strings.Builder{}
	for _, imp := range descriptor.imports {
		uniqueName := uniquePackageName(imp.path)
		if uniqueName != imp.path {
			explicitImports.WriteString(uniqueName + ` "` + imp.path + `"` + "\n")
		} else {
			explicitImports.WriteString(`"` + imp.path + `"` + "\n")
		}
	}

	pkgs := map[string]*packageType{}
	for _, imp := range descriptor.imports {
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
		// Convert all declaration name to "unexported" if requested.
		// TODO(Gianluca): if uncapitalized name conflicts with a Go builtin
		//  return an error. Note that the builtin functions 'print' and
		//  'println' should be handled as special case:
		if imp.comment.uncapitalize {
			tmp := map[string]string{}
			for name, decl := range decls {
				newName := uncapitalize(name)
				if newName == "main" || newName == "init" {
					return "", false, fmt.Errorf("%q is not a valid identifier: remove 'uncapitalize' or change declaration name in package %q", newName, imp.path)
				}
				if isGoKeyword(newName) {
					return "", false, fmt.Errorf("%q is not a valid identifier as it conflicts with Go keyword %q: remove 'uncapitalize' or change declaration name in package %q", newName, newName, imp.path)
				}
				if isPredeclaredIdentifier(newName) {
					if newName == "print" || newName == "println" {
						// TODO(Gianluca): If the signature is the same of Go
						//  'print' and 'println', shadowing is allowed. Else, an
						//  error must be returned.
					} else {
						return "", false, fmt.Errorf("%q is not a valid identifier as it conflicts with Go predeclared identifier %q: remove 'uncapitalize' or change declaration name in package %q", newName, newName, imp.path)
					}
				}
				tmp[newName] = decl
			}
			decls = tmp
		}

		// Determine which import path use: the default (the one specified in
		// source, used by Go) or a new one indicated in Scriggo comments.
		path := imp.path
		if imp.comment.newPath != "" {
			// TODO(Gianluca): if newPath points to standard lib, return error.
			//  Else, package mixing is allowed.
			path = imp.comment.newPath
		}

		switch imp.comment.main {
		case true: // Add read declarations to package main as builtins.
			if pkgs["main"] == nil {
				pkgs["main"] = &packageType{"main", map[string]string{}}
			}
			for name, decl := range decls {
				if _, ok := pkgs["main"].decl[name]; ok {
					return "", false, fmt.Errorf("declaration name collision in package main: %q", name)
				}
				pkgs["main"].decl[name] = decl
			}
		case false: // Add read declarations to the specified package.
			if pkgs[path] == nil {
				pkgs[path] = &packageType{
					decl: map[string]string{},
				}
			}
			for name, decl := range decls {
				if _, ok := pkgs[path].decl[name]; ok {
					return "", false, fmt.Errorf("declaration name collision: %q in package %q", name, imp.path)
				}
				pkgs[path].decl[name] = decl
				pkgs[path].name = pkgName
				if imp.comment.newName != "" {
					name = imp.comment.newName
				}
			}
		}

	}

	// If no packages have been declared, just return.
	if len(pkgs) == 0 {
		return "", false, nil
	}

	// Render package content.
	allPkgsContent := strings.Builder{}
	paths := make([]string, 0, len(pkgs))
	hasMain := false
	for path := range pkgs {
		if path == "main" {
			hasMain = true
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if hasMain {
		paths = append([]string{"main"}, paths...)
	}
	for _, path := range paths {
		pkg := pkgs[path]
		declarations := strings.Builder{}
		names := make([]string, 0, len(pkg.decl))
		for name := range pkg.decl {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			decl := pkg.decl[name]
			declarations.WriteString(strconv.Quote(name) + ": " + decl + ",\n")
		}
		out := `[path]: {
			Name: [pkgName],
			Declarations: map[string]interface{}{
				[declarations]
			},
		},
		`
		out = strings.Replace(out, "[path]", strconv.Quote(path), 1)
		out = strings.Replace(out, "[pkgName]", strconv.Quote(pkg.name), 1)
		out = strings.Replace(out, "[declarations]", declarations.String(), 1)
		allPkgsContent.WriteString(out)
	}

	// TODO(Gianluca): package main must be first.

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
		"[pkgName]", descriptor.pkgName,
		"[explicitImports]", explicitImports.String(),
		"[customVariableName]", pkgsVariable,
		"[pkgContent]", allPkgsContent.String(),
	).Replace(pkgsSkeleton)
	pkgOutput = genHeader(descriptor, goos) + pkgOutput

	return pkgOutput, true, nil
}

// parseGoPackage parses a package path and returns the package name and a map
// containing the exported declarations in that package.
func parseGoPackage(pkgPath, goos string) (string, map[string]string, error) {

	fmt.Printf("generating package %q (GOOS=%q)...", pkgPath, goos)

	out := make(map[string]string)
	pkgBase := uniquePackageName(pkgPath)

	conf := &packages.Config{
		Mode: 1023,
	}
	if goos != "" {
		// TODO(Gianluca): enable support for different GOOSes.
		//  conf.Env = append(os.Environ(), "GOOS=", goos)
	}

	pkgs, err := packages.Load(conf, pkgPath)
	if err != nil {
		return "", nil, err
	}

	if packages.PrintErrors(pkgs) > 0 {
		return "", nil, errors.New("error")
	}

	if len(pkgs) > 1 {
		return "", nil, errors.New("package query returned more than one package")
	}

	if len(pkgs) != 1 {
		panic("bug")
	}

	pkg := pkgs[0]

	pkgInfo := pkg.TypesInfo
	for _, v := range pkgInfo.Defs {
		// Include only exported names. Do not take into account whether the
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
			if ss := strings.Split(v.String(), " "); len(ss) >= 3 {
				if strings.HasPrefix(ss[2], "struct{") {
					out[v.Name()] = fmt.Sprintf("reflect.TypeOf(%s.%s{})", pkgBase, v.Name())
					continue
				}
			}
			out[v.Name()] = fmt.Sprintf("reflect.TypeOf(new(%s.%s)).Elem()", pkgBase, v.Name())

		}
	}

	fmt.Printf("done!\n")

	return pkg.Name, out, nil
}
