// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package native

// Package represents a native package.
type Package interface {

	// PackageName returns the name of the package.
	// It must be a Go identifier and cannot be the black identifier.
	PackageName() string

	// Lookup searches for an exported declaration, named name, in the
	// package. If the declaration does not exist, it returns nil.
	Lookup(name string) Declaration

	// DeclarationNames returns the exported declaration names in the package.
	DeclarationNames() []string
}

// PackageLoader represents a package loader; Load returns the native package
// with the given path.
//
// If an error occurs it returns the error, if the package does not exist it
// returns a nil package.
type PackageLoader interface {
	Load(path string) (Package, error)
}

// CombinedLoader combines multiple loaders into one loader.
type CombinedLoader []PackageLoader

// Load calls each loader's Load methods and returns as soon as a loader
// returns a package.
func (loaders CombinedLoader) Load(path string) (Package, error) {
	for _, loader := range loaders {
		p, err := loader.Load(path)
		if p != nil || err != nil {
			return p, err
		}
	}
	return nil, nil
}

// Packages implements PackageLoader using a map of Package.
type Packages map[string]Package

// Load returns a Package.
func (pp Packages) Load(path string) (Package, error) {
	if p, ok := pp[path]; ok {
		return p, nil
	}
	return nil, nil
}

// DeclarationsPackage implements Package given its name and declarations.
type DeclarationsPackage struct {
	// Name of the package.
	Name string
	// Declarations of the package.
	Declarations Declarations
}

// PackageName returns the name of the package.
func (p DeclarationsPackage) PackageName() string {
	return p.Name
}

// Lookup returns the declaration named name in the package or nil if no such
// declaration exists.
func (p DeclarationsPackage) Lookup(name string) Declaration {
	return p.Declarations[name]
}

// DeclarationNames returns all package declaration names.
func (p DeclarationsPackage) DeclarationNames() []string {
	declarations := make([]string, 0, len(p.Declarations))
	for name := range p.Declarations {
		declarations = append(declarations, name)
	}
	return declarations
}

// CombinedPackage implements a Package by combining multiple packages into
// one package with name the name of the first package and as declarations the
// declarations of all packages.
//
// The Lookup method calls the Lookup methods of each package in order and
// returns as soon as a package returns a not nil value.
type CombinedPackage []Package

// Name returns the name of the first combined package.
func (packages CombinedPackage) PackageName() string {
	if len(packages) == 0 {
		return ""
	}
	return packages[0].PackageName()
}

// Lookup calls the Lookup methods of each package in order and returns as
// soon as a combined package returns a declaration.
func (packages CombinedPackage) Lookup(name string) Declaration {
	for _, pkg := range packages {
		if decl := pkg.Lookup(name); decl != nil {
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
