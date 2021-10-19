// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"go/constant"
	"go/types"
	"io"
	"math/big"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/template"

	pkgs "golang.org/x/tools/go/packages"
)

// renderPackages renders a Scriggofile. It also returns a boolean indicating
// if the content contains packages. Ignores all main packages contained in
// the Scriggofile.
func renderPackages(w io.Writer, dir string, sf *scriggofile, goos string, flags buildFlags) error {

	type packageType struct {
		name string
		decl map[string]string
	}
	cache := newPackageNameCache()

	// Import the packages of the Go standard library.
	for i, imp := range sf.imports {
		if imp.stdlib {
			paths := stdLibPaths()
			imports := make([]*importCommand, len(sf.imports)+len(paths)-1)
			copy(imports[:i], sf.imports[:i])
			for j, path := range paths {
				imports[i+j] = &importCommand{path: path}
			}
			copy(imports[i+len(paths):], sf.imports[i+1:])
			sf.imports = imports
		}
	}

	alreadyImportsReflect := false
	explicitImports := []struct{ Name, Path string }{}

	mustImportReflect := false
	packages := map[string]*packageType{}
	for i, imp := range sf.imports {
		if flags.v {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", imp.path)
		}
		pkgName, decls, refToImport, refToReflect, err := loadGoPackage(imp.path, dir, goos, flags, imp.including, imp.excluding, cache)
		if err != nil {
			return err
		}
		uniqueName := cache.uniquePackageName(imp.path, pkgName)
		if uniqueName != pkgName {
			explicitImports = append(explicitImports, struct{ Name, Path string }{uniqueName, imp.path})
		} else {
			explicitImports = append(explicitImports, struct{ Name, Path string }{pkgName, imp.path})
		}
		if imp.path == "reflect" {
			alreadyImportsReflect = true
		}

		mustImportReflect = mustImportReflect || refToReflect
		if !refToImport {
			explicitImports[i].Name = "_"
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
					return fmt.Errorf("%q is not a valid identifier: remove 'uncapitalized' or change declaration name in package %q", newName, imp.path)
				}
				if isGoKeyword(newName) {
					return fmt.Errorf("%q is not a valid identifier as it conflicts with Go keyword %q: remove 'uncapitalized' or change declaration name in package %q", newName, newName, imp.path)
				}
				if isPredeclaredIdentifier(newName) {
					if newName == "print" || newName == "println" {
						// https://github.com/open2b/scriggo/issues/361.
					} else {
						return fmt.Errorf("%q is not a valid identifier as it conflicts with Go predeclared identifier %q: remove 'uncapitalized' or change declaration name in package %q", newName, newName, imp.path)
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
					return fmt.Errorf("declaration name collision in package main: %q", name)
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
					return fmt.Errorf("declaration name collision: %q in package %q", name, imp.path)
				}
				packages[path].decl[name] = decl
				packages[path].name = pkgName
			}
		}

	}

	// If the 'reflect' package has already been imported from the Scriggofile,
	// it must not be added twice, even if the value of some declarations of
	// another package refers to it.
	if alreadyImportsReflect {
		mustImportReflect = false
	}

	if w == nil {
		return nil
	}

	// If no packages have been declared, just return.
	if len(packages) == 0 {
		return nil
	}

	// Render package content.
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
	type declaration struct {
		Name  string
		Value string
	}
	allPkgsContent := make([]*struct {
		Variable     string
		Path, Name   string
		Declarations []declaration
	}, len(paths))
	for i, path := range paths {
		pkg := packages[path]
		names := make([]string, 0, len(pkg.decl))
		for name := range pkg.decl {
			names = append(names, name)
		}
		sort.Strings(names)
		declarations := make([]declaration, len(pkg.decl))
		for j, name := range names {
			declarations[j] = declaration{
				Name:  name,
				Value: pkg.decl[name],
			}
		}
		allPkgsContent[i] = &struct {
			Variable     string
			Path, Name   string
			Declarations []declaration
		}{
			Variable:     sf.variable,
			Path:         strconv.Quote(path),
			Name:         strconv.Quote(pkg.name),
			Declarations: declarations,
		}
	}

	// Skeleton for a package group.
	const pkgsSkeleton = `// Code generated by scriggo command. DO NOT EDIT.
//go:build {{.GOOS}} && {{.BaseVersion}} && !{{.NextGoVersion}}
// +build {{.GOOS}},{{.BaseVersion}},!{{.NextGoVersion}}

package {{.Name}}

import (
{{- range .ExplicitImports}}
	{{if ne .Name ""}}{{.Name}} {{end}}"{{.Path}}"
{{- end}}
)

import "github.com/open2b/scriggo/native"
{{- if .MustImportReflect}}
import "reflect"
{{- end}}

func init() {
	{{.Variable}} = make(native.Packages, {{len .PkgContent}})
	var decs native.Declarations

	{{- range .PkgContent}}
	// {{.Path}}
	decs = make(native.Declarations, {{len .Declarations}})
	{{- range .Declarations}}
	decs["{{.Name}}"] = {{.Value}}
	{{- end}}
	{{.Variable}}[{{.Path}}] = native.Package{
		Name:         {{.Name}},
		Declarations: decs,
	}

	{{- end}}
}
`

	pkgOutput := map[string]interface{}{
		"GOOS":              goos,
		"BaseVersion":       goBaseVersion(runtime.Version()),
		"NextGoVersion":     nextGoVersion(runtime.Version()),
		"Name":              sf.pkgName,
		"ExplicitImports":   explicitImports,
		"MustImportReflect": mustImportReflect,
		"Variable":          sf.variable,
		"PkgContent":        allPkgsContent,
	}

	t := template.Must(template.New("packages").Parse(pkgsSkeleton))
	err := t.Execute(w, pkgOutput)

	return err
}

// loadGoPackage loads the Go package with the given path and returns its name
// and its exported declarations.
//
// refToImport reports whether at least one declaration refers to the import
// path directly; for example when importing a package with no declarations or
// where all declarations are constant literals refToImport is false.
//
// refToScriggo reports whether at least one of the declarations refers to the
// package 'scriggo', while refToReflect reports whether at least one of the
// declarations refers to the package 'reflect'.
func loadGoPackage(path, dir, goos string, flags buildFlags, including, excluding []string, cache packageNameCache) (name string, decl map[string]string, refToImport, refToReflect bool, err error) {

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

	conf := &pkgs.Config{
		Mode: 1023,
	}
	if goos != "" {
		// https://github.com/open2b/scriggo/issues/388
		//  conf.Env = append(os.Environ(), "GOOS=", goos)
	}

	if flags.x {
		_, _ = fmt.Fprintf(os.Stderr, "go list -json -find %s\n", path)
	}
	if dir != "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", nil, false, false, fmt.Errorf("scriggo: can't get current directory: %s", err)
		}
		err = os.Chdir(dir)
		if err != nil {
			return "", nil, false, false, fmt.Errorf("scriggo: can't change current directory: %s", err)
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
		return "", nil, false, false, err
	}

	if pkgs.PrintErrors(packages) > 0 {
		return "", nil, false, false, errors.New("error")
	}

	if len(packages) > 1 {
		return "", nil, false, false, errors.New("package query returned more than one package")
	}

	if len(packages) != 1 {
		panic("bug")
	}

	name = packages[0].Name
	pkgBase := cache.uniquePackageName(path, name)

	numUntyped := 0

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
		if !allowed(v.Name()) {
			continue
		}
		switch v := v.(type) {
		case *types.Const:
			t := v.Type()
			if basic, ok := t.(*types.Basic); ok && basic.Info()&types.IsUntyped != 0 {
				val := v.Val()
				s := val.ExactString()
				var value string
				switch val.Kind() {
				case constant.String:
					value = "native.UntypedStringConst(" + s + ")"
				case constant.Bool:
					value = "native.UntypedBooleanConst(" + s + ")"
				case constant.Int:
					value = "native.UntypedNumericConst(" + strconv.Quote(s) + ")"
				case constant.Float:
					if strings.Contains(s, "/") {
						if rat, ok := new(big.Rat).SetString(s); ok {
							s2 := rat.FloatString(512)
							if rat2, ok := new(big.Rat).SetString(s2); ok && rat.Cmp(rat2) == 0 {
								if f, ok := new(big.Float).SetPrec(512).SetString(s2); ok {
									s = f.Text('g', -1)
								}
							}
						}
					} else if !strings.Contains(s, ".") {
						s += ".0"
					}
					value = "native.UntypedNumericConst(" + strconv.Quote(s) + ")"
				case constant.Complex:
					s = strings.ReplaceAll(s[1:len(s)-1], " ", "")
					value = "native.UntypedNumericConst(" + strconv.Quote(s) + ")"
				default:
					panic(fmt.Sprintf("Unexpected constant kind %d", val.Kind()))
				}
				decl[v.Name()] = value
				numUntyped++
			} else {
				decl[v.Name()] = fmt.Sprintf("%s.%s", pkgBase, v.Name())
			}
		case *types.Func:
			if v.Type().(*types.Signature).Recv() == nil {
				decl[v.Name()] = fmt.Sprintf("%s.%s", pkgBase, v.Name())
			}
		case *types.Var:
			if !v.Embedded() && !v.IsField() {
				decl[v.Name()] = fmt.Sprintf("&%s.%s", pkgBase, v.Name())
			}
		case *types.TypeName:
			decl[v.Name()] = fmt.Sprintf("reflect.TypeOf((*%s.%s)(nil)).Elem()", pkgBase, v.Name())
			refToReflect = true
		}
	}

	refToImport = len(decl) > numUntyped

	return name, decl, refToImport, refToReflect, nil
}
