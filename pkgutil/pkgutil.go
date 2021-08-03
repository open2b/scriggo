// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkgutil

import (
	"github.com/open2b/scriggo"
)

// CombinedLoader combines multiple loaders into one loader.
type CombinedLoader []scriggo.PackageLoader

// Load calls each loader's Load methods and returns as soon as a loader
// returns a package.
func (loaders CombinedLoader) Load(path string) (interface{}, error) {
	for _, loader := range loaders {
		p, err := loader.Load(path)
		if p != nil || err != nil {
			return p, err
		}
	}
	return nil, nil
}

// Packages implements scriggo.PackageLoader with a map of scriggo.Package
// values.
type Packages map[string]scriggo.Package

// Load returns a scriggo.Package or the source of a package, as io.Reader.
// It returns nil if there is no package with the given name.
func (pp Packages) Load(path string) (interface{}, error) {
	if p, ok := pp[path]; ok {
		return p, nil
	}
	return nil, nil
}

// MapPackage implements scriggo.Package given a package name and a map of
// declarations.
type MapPackage struct {
	// Package name.
	PkgName string
	// Package declarations.
	Declarations map[string]interface{}
}

// Name returns the package name.
func (p MapPackage) Name() string {
	return p.PkgName
}

// Lookup returns the declaration declName in the package or nil if no such
// declaration exists.
func (p MapPackage) Lookup(declName string) interface{} {
	return p.Declarations[declName]
}

// DeclarationNames returns returns all package declaration names.
func (p MapPackage) DeclarationNames() []string {
	declarations := make([]string, 0, len(p.Declarations))
	for name := range p.Declarations {
		declarations = append(declarations, name)
	}
	return declarations
}

// CombinedPackage implements scriggo.Package by combining multiple packages
// into one package with name the name of the first package and as
// declarations the declarations of all packages.
//
// The Lookup method calls the Lookup methods of each package in order and
// returns as soon as a package returns a declaration.
type CombinedPackage []scriggo.Package

// Name returns the name of the first combined package.
func (packages CombinedPackage) Name() string {
	if len(packages) == 0 {
		return ""
	}
	return packages[0].Name()
}

// Lookup calls the Lookup methods of each package in order and returns as
// soon as a combined package returns a declaration.
func (packages CombinedPackage) Lookup(declName string) interface{} {
	for _, pkg := range packages {
		if decl := pkg.Lookup(declName); decl != nil {
			return decl
		}
	}
	return nil
}

// DeclarationNames returns all declaration names in all packages.
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
