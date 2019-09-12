// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo

import (
	"strings"
)

// Package represents a predefined package.
type Package interface {

	// Name returns the package's name.
	Name() string

	// Lookup searches for an exported declaration, named declName, in the
	// package. If the declaration does not exist, it returns nil.
	//
	// For a variable returns a pointer to the variable, for a function
	// returns the function, for a type returns the reflect.Type and for a
	// constant returns its value or a Constant.
	Lookup(declName string) interface{}

	// DeclarationNames returns the exported declaration names in the package.
	DeclarationNames() []string
}

// PackageLoader is implemented by package loaders. Given a package path, Load
// returns a *Package value or a package source as io.Reader.
//
// If the package does not exist it returns nil and nil.
// If the package exists but there was an error while loading the package, it
// returns nil and the error.
//
// If Load returns an io.Reader that implements io.Closer, the Close method
// will be called after a Read returns either EOF or an error.
type PackageLoader interface {
	Load(path string) (interface{}, error)
}

// MapStringLoader implements PackageLoader that returns the source of a
// package. Package paths and sources are respectively the keys and the values
// of the map.
type MapStringLoader map[string]string

func (r MapStringLoader) Load(path string) (interface{}, error) {
	if src, ok := r[path]; ok {
		return strings.NewReader(src), nil
	}
	return nil, nil
}

// CombinedLoader combines more loaders in one loader. Load calls in order the
// Load methods of each loader and returns as soon as a loader returns a
// package.
type CombinedLoader []PackageLoader

func (loaders CombinedLoader) Load(path string) (interface{}, error) {
	for _, loader := range loaders {
		p, err := loader.Load(path)
		if p != nil || err != nil {
			return p, err
		}
	}
	return nil, nil
}

// Loaders returns a CombinedLoader that combine loaders.
func Loaders(loaders ...PackageLoader) PackageLoader {
	return CombinedLoader(loaders)
}

// Packages is a Loader that load packages from a map where the key is a
// package path and the value is a *Package value.
type Packages map[string]Package

func (pp Packages) Load(path string) (interface{}, error) {
	if p, ok := pp[path]; ok {
		return p, nil
	}
	return nil, nil
}

type MapPackage struct {
	// Package name.
	PkgName string
	// Package declarations.
	Declarations map[string]interface{}
}

func (p *MapPackage) Name() string {
	return p.PkgName
}

func (p *MapPackage) Lookup(declName string) interface{} {
	return p.Declarations[declName]
}

func (p *MapPackage) DeclarationNames() []string {
	declarations := make([]string, 0, len(p.Declarations))
	for name := range p.Declarations {
		declarations = append(declarations, name)
	}
	return declarations
}

// CombinedPackage combines more packages in one package with name the name of
// the first package and declarations the declarations of all the packages.
//
// Lookup method calls in order the Lookup methods of each package and returns
// as soon as a package returns a declaration.
type CombinedPackage []Package

func (packages CombinedPackage) Name() string {
	if len(packages) == 0 {
		return ""
	}
	return packages[0].Name()
}

func (packages CombinedPackage) Lookup(declName string) interface{} {
	for _, pkg := range packages {
		if decl := pkg.Lookup(declName); decl != nil {
			return decl
		}
	}
	return nil
}

func (packages CombinedPackage) DeclarationNames() []string {
	if len(packages) == 0 {
		return []string{}
	}
	var names []string
	for i, pkg := range packages {
		if i == 0 {
			names = pkg.DeclarationNames()
			continue
		}
		for _, name := range pkg.DeclarationNames() {
			exists := false
			for _, n := range names {
				if n == name {
					exists = true
					break
				}
			}
			if !exists {
				names = append(names, name)
			}
		}
	}
	return names
}
