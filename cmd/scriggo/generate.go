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
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	pkgs "golang.org/x/tools/go/packages"
)

// renderPackages renders a Scriggofile. It also returns a boolean indicating
// if the content contains packages. Ignores all main packages contained in
// the Scriggofile.
func renderPackages(w io.Writer, dir string, sf *scriggofile, goos string, flags buildFlags) (int, error) {

	type packageType struct {
		name string
		decl map[string]string
	}

	// Import the packages of the Go standard library.
	for i, imp := range sf.imports {
		if imp.stdlib {
			imports := make([]*importCommand, len(sf.imports)+len(stdlibPaths)-1)
			copy(imports[:i], sf.imports[:i])
			for j, path := range stdlibPaths {
				imports[i+j] = &importCommand{path: path}
			}
			copy(imports[i+len(stdlibPaths):], sf.imports[i+1:])
			sf.imports = imports
		}
	}

	importReflect := false

	explicitImports := [][2]string{} // first element is the import name, second is import path..
	for _, imp := range sf.imports {
		uniqueName := uniquePackageName(imp.path)
		if uniqueName != imp.path { // TODO: uniqueName should be compared to the package name and not to the package path.
			explicitImports = append(explicitImports, [2]string{uniqueName, imp.path})
		} else {
			explicitImports = append(explicitImports, [2]string{"", imp.path})
		}
		if imp.path == "reflect" {
			importReflect = true
		}
	}

	packages := map[string]*packageType{}
	for _, imp := range sf.imports {
		if flags.v {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", imp.path)
		}
		pkgName, decls, refToImport, err := loadGoPackage(imp.path, dir, goos, flags, imp.including, imp.excluding)
		if err != nil {
			return 0, err
		}
		if !refToImport {
			explicitImports[len(explicitImports)-1][0] = "_"
		}
		// No declarations at path: move on to next import path.
		if len(decls) == 0 {
			continue
		}
		if imp.notCapitalized {
			tmp := map[string]string{}
			for name, decl := range decls {
				newName := uncapitalize(name)
				if newName == "main" || newName == "init" {
					return 0, fmt.Errorf("%q is not a valid identifier: remove 'uncapitalized' or change declaration name in package %q", newName, imp.path)
				}
				if isGoKeyword(newName) {
					return 0, fmt.Errorf("%q is not a valid identifier as it conflicts with Go keyword %q: remove 'uncapitalized' or change declaration name in package %q", newName, newName, imp.path)
				}
				if isPredeclaredIdentifier(newName) {
					if newName == "print" || newName == "println" {
						// TODO(Gianluca): If the signature is the same of Go
						//  'print' and 'println', shadowing is allowed. Else, an
						//  error must be returned.
					} else {
						return 0, fmt.Errorf("%q is not a valid identifier as it conflicts with Go predeclared identifier %q: remove 'uncapitalized' or change declaration name in package %q", newName, newName, imp.path)
					}
				}
				tmp[newName] = decl
			}
			decls = tmp
		}

		// Determine which import path use: the default (the one specified in
		// source, used by Go) or a new one indicated in Scriggo comments.
		path := imp.path
		if imp.asPath != "" {
			path = imp.asPath
		}

		switch imp.asPath {
		case "main": // Add read declarations to package main as builtins.
			if packages["main"] == nil {
				packages["main"] = &packageType{"main", map[string]string{}}
			}
			for name, decl := range decls {
				if _, ok := packages["main"].decl[name]; ok {
					return 0, fmt.Errorf("declaration name collision in package main: %q", name)
				}
				packages["main"].decl[name] = decl
			}
		default: // Add read declarations to the specified package.
			if packages[path] == nil {
				packages[path] = &packageType{
					decl: map[string]string{},
				}
			}
			for name, decl := range decls {
				if _, ok := packages[path].decl[name]; ok {
					return 0, fmt.Errorf("declaration name collision: %q in package %q", name, imp.path)
				}
				packages[path].decl[name] = decl
				packages[path].name = pkgName
				name = filepath.Base(path)
			}
		}

	}

	if w == nil {
		return 0, nil
	}

	// If no packages have been declared, just return.
	if len(packages) == 0 {
		return 0, nil
	}

	// Render package content.
	allPkgsContent := strings.Builder{}
	paths := make([]string, 0, len(packages))
	hasMain := false
	for path := range packages {
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
	const spaces = "                              "
	for i, path := range paths {
		var out string
		if i > 0 {
			out = "\n\t\t"
		}
		pkg := packages[path]
		declarations := strings.Builder{}
		names := make([]string, 0, len(pkg.decl))
		var maxLen int
		for name := range pkg.decl {
			names = append(names, name)
			if l := len(name); l > maxLen {
				maxLen = l
			}
		}
		sort.Strings(names)
		for j, name := range names {
			if j > 0 {
				declarations.WriteString("\n\t\t\t\t")
			}
			decl := pkg.decl[name]
			declarations.WriteByte('"')
			declarations.WriteString(name)
			declarations.WriteString("\":")
			if n := maxLen - len(name) + 1; n <= len(spaces) {
				declarations.WriteString(spaces[:n])
			} else {
				declarations.WriteString(strings.Repeat(" ", n))
			}
			declarations.WriteString(decl)
			declarations.WriteString(",")
		}
		out += `[path]: &MapPackage{
			PkgName: [pkgName],
			Declarations: map[string]interface{}{
				[declarations]
			},
		},`
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
[reflectImport]

func init() {
	[variable] = Packages{
		[pkgContent]
	}
}
`

	var reflectImport string
	if !importReflect {
		reflectImport = `import "reflect"`
	}

	explicitImportsString := ""
	for i, imp := range explicitImports {
		if i > 0 {
			explicitImportsString = "\n\t"
		}
		name := imp[0]
		path := imp[1]
		if name == "" {
			explicitImportsString += `"` + path + `"`
		} else {
			explicitImportsString += name + ` "` + path + `"`
		}
	}

	pkgOutput := strings.NewReplacer(
		"[pkgName]", sf.pkgName,
		"[explicitImports]", explicitImportsString,
		"[reflectImport]", reflectImport,
		"[variable]", sf.variable,
		"[pkgContent]", allPkgsContent.String(),
	).Replace(pkgsSkeleton)
	pkgOutput = genHeader(sf, goos) + pkgOutput

	return io.WriteString(w, pkgOutput)
}

// loadGoPackage loads the Go package with the given path and returns its name
// and its exported declarations.
//
// refToImport reports whether at least one declaration refers to the import
// path directly; for example when loading a package with no declarations or
// where all declarations are constant literals refToImport is false.
//
func loadGoPackage(path, dir, goos string, flags buildFlags, including, excluding []string) (name string, decl map[string]string, refToImport bool, err error) {

	allowed := func(n string) bool {
		if len(including) > 0 {
			for _, inc := range including {
				if inc == n {
					return true
				}
			}
			return false
		}
		if len(excluding) > 0 {
			for _, exc := range excluding {
				if exc == n {
					return false
				}
			}
			return true
		}
		return true
	}

	decl = map[string]string{}
	// TODO(marco): remove the global cache of package names.
	pkgBase := uniquePackageName(path)

	conf := &pkgs.Config{
		Mode: 1023,
	}
	if goos != "" {
		// TODO(Gianluca): enable support for different GOOSes.
		//  conf.Env = append(os.Environ(), "GOOS=", goos)
	}

	if flags.x {
		_, _ = fmt.Fprintf(os.Stderr, "go list -json -find %s\n", path)
	}
	if dir != "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", nil, false, fmt.Errorf("scriggo: can't get current directory: %s", err)
		}
		err = os.Chdir(dir)
		if err != nil {
			return "", nil, false, fmt.Errorf("scriggo: can't change current directory: %s", err)
		}
		defer func() {
			err = os.Chdir(cwd)
			if err != nil {
				name = ""
				decl = nil
				err = fmt.Errorf("scriggo: can't change current directory: %s", err)
			}
		}()
	}
	packages, err := pkgs.Load(conf, path)
	if err != nil {
		return "", nil, false, err
	}

	if pkgs.PrintErrors(packages) > 0 {
		return "", nil, false, errors.New("error")
	}

	if len(packages) > 1 {
		return "", nil, false, errors.New("package query returned more than one package")
	}

	if len(packages) != 1 {
		panic("bug")
	}

	name = packages[0].Name

	for _, v := range packages[0].TypesInfo.Defs {
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
			if !allowed(v.Name()) {
				continue
			}
			switch v.Val().Kind() {
			case constant.String, constant.Bool:
				decl[v.Name()] = fmt.Sprintf("ConstValue(%s.%s)", pkgBase, v.Name())
				refToImport = true
				continue
			case constant.Int:
				// Most cases fall here.
				if len(v.Val().ExactString()) < 7 {
					decl[v.Name()] = fmt.Sprintf("ConstValue(%s.%s)", pkgBase, v.Name())
					refToImport = true
					continue
				}
			}
			typ := v.Type().String()
			if strings.HasPrefix(typ, "untyped ") {
				typ = "nil"
			} else {
				if strings.Contains(typ, ".") {
					typ = pkgBase + filepath.Ext(typ)
					refToImport = true
				}
				typ = fmt.Sprintf("reflect.TypeOf(new(%s)).Elem()", typ)
			}
			exact := v.Val().ExactString()
			quoted := strconv.Quote(exact)
			if v.Val().Kind() == constant.String && len(quoted) == len(exact)+4 {
				quoted = "`" + exact + "`"
			}
			decl[v.Name()] = fmt.Sprintf("ConstLiteral(%v, %s)", typ, quoted)
		case *types.Func:
			if !allowed(v.Name()) {
				continue
			}
			if v.Type().(*types.Signature).Recv() == nil {
				decl[v.Name()] = fmt.Sprintf("%s.%s", pkgBase, v.Name())
				refToImport = true
			}
		case *types.Var:
			if !allowed(v.Name()) {
				continue
			}
			if !v.Embedded() && !v.IsField() {
				decl[v.Name()] = fmt.Sprintf("&%s.%s", pkgBase, v.Name())
				refToImport = true
			}
		case *types.TypeName:
			if !allowed(v.Name()) {
				continue
			}
			if parts := strings.Split(v.String(), " "); len(parts) >= 3 {
				if strings.HasPrefix(parts[2], "struct{") {
					decl[v.Name()] = fmt.Sprintf("reflect.TypeOf(%s.%s{})", pkgBase, v.Name())
					refToImport = true
					continue
				}
			}
			decl[v.Name()] = fmt.Sprintf("reflect.TypeOf(new(%s.%s)).Elem()", pkgBase, v.Name())
			refToImport = true
		}
	}

	return name, decl, refToImport, nil
}
