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
	"math/big"
	"os"
	"path/filepath"
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

	explicitImports := []struct{ Name, Path string }{}
	for _, imp := range sf.imports {
		uniqueName := uniquePackageName(imp.path)
		if uniqueName != imp.path { // TODO: uniqueName should be compared to the package name and not to the package path.
			explicitImports = append(explicitImports, struct{ Name, Path string }{uniqueName, imp.path})
		} else {
			explicitImports = append(explicitImports, struct{ Name, Path string }{"", imp.path})
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
			return err
		}
		if !refToImport {
			explicitImports[len(explicitImports)-1].Name = "_"
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
						// TODO(Gianluca): If the signature is the same of Go
						//  'print' and 'println', shadowing is allowed. Else, an
						//  error must be returned.
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
				name = filepath.Base(path)
			}
		}

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

	// TODO(Gianluca): package main must be first.

	// Skeleton for a package group.
	const pkgsSkeleton = `// Code generated by scriggo command. DO NOT EDIT.
//+build {{.GOOS}},{{.BaseVersion}},!{{.NextGoVersion}}

package {{.PkgName}}

import (
{{- range .ExplicitImports}}
	{{if ne .Name ""}}{{.Name}} {{end}}"{{.Path}}"
{{- end}}
)

import . "scriggo"
{{ if not .ImportReflect}}import "reflect"{{end}}

func init() {
	{{.Variable}} = make(Packages, {{len .PkgContent}})
	var decs map[string]interface{}

	{{- range .PkgContent}}
	// {{.Path}}
	decs = make(map[string]interface{}, {{len .Declarations}})
	{{- range .Declarations}}
	decs["{{.Name}}"] = {{.Value}}
	{{- end}}
	{{.Variable}}[{{.Path}}] = &MapPackage{
		PkgName: {{.Name}},
		Declarations: decs,
	}

	{{- end}}
}
`

	pkgOutput := map[string]interface{}{
		"GOOS":            goos,
		"BaseVersion":     goBaseVersion(runtime.Version()),
		"NextGoVersion":   nextGoVersion(runtime.Version()),
		"PkgName":         sf.pkgName,
		"ExplicitImports": explicitImports,
		"ImportReflect":   importReflect,
		"Variable":        sf.variable,
		"PkgContent":      allPkgsContent,
	}

	t := template.Must(template.New("packages").Parse(pkgsSkeleton))
	err := t.Execute(w, pkgOutput)

	return err
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
				if val.Kind() == constant.Float && strings.Contains(s, "/") {
					if rat, ok := new(big.Rat).SetString(s); ok {
						s2 := rat.FloatString(512)
						if rat2, ok := new(big.Rat).SetString(s2); ok && rat.Cmp(rat2) == 0 {
							if f, ok := new(big.Float).SetPrec(512).SetString(s2); ok {
								s = f.Text('g', -1)
							}
						}
					}
				}
				decl[v.Name()] = "UntypedConstant(" + strconv.Quote(s) + ")"
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
		}
	}

	refToImport = len(decl) > numUntyped

	return name, decl, refToImport, nil
}
