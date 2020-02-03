// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

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
