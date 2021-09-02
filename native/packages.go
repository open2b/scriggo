// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package native

import "errors"

// StopLookup is used as return value from a LookupFunc function to indicate
// that the lookup should be stopped.
var StopLookup = errors.New("stop lookup")

// LookupFunc is the type of the function called by
// ImportablePackage.LookupFunc to read each package declaration. If the
// function returns an error, ImportablePackage.LookupFunc stops and returns
// the error or nil if the error is StopLookup.
type LookupFunc func(name string, decl Declaration) error

// ImportablePackage represents an importable package.
type ImportablePackage interface {

	// PackageName returns the name of the package.
	// It is a Go identifier but not the empty identifier.
	PackageName() string

	// Lookup searches for an exported declaration, named name, in the
	// package. If the declaration does not exist, it returns nil.
	Lookup(name string) Declaration

	// LookupFunc calls f for each package declaration stopping if f returns
	// an error. Lookup order is undefined.
	LookupFunc(f LookupFunc) error
}

// Importer represents a package importer; Import returns the native package
// with the given package path.
//
// If an error occurs it returns the error, if the package does not exist it
// returns nil and nil.
type Importer interface {
	Import(path string) (ImportablePackage, error)
}

// CombinedImporter combines multiple importers into one importer.
type CombinedImporter []Importer

// Import calls the Import method of each importer and returns as soon as an
// importer returns a package.
func (importers CombinedImporter) Import(path string) (ImportablePackage, error) {
	for _, importer := range importers {
		p, err := importer.Import(path)
		if p != nil || err != nil {
			return p, err
		}
	}
	return nil, nil
}

// Packages implements Importer using a map of ImportablePackage.
type Packages map[string]ImportablePackage

// Import returns an ImportablePackage.
func (pp Packages) Import(path string) (ImportablePackage, error) {
	if p, ok := pp[path]; ok {
		return p, nil
	}
	return nil, nil
}

// Package implements ImportablePackage given its name and declarations.
type Package struct {
	// Name of the package.
	Name string
	// Declarations of the package.
	Declarations Declarations
}

// PackageName returns the name of the package.
func (p Package) PackageName() string {
	return p.Name
}

// Lookup returns the declaration named name in the package or nil if no such
// declaration exists.
func (p Package) Lookup(name string) Declaration {
	return p.Declarations[name]
}

// LookupFunc calls f for each package declaration stopping if f returns an
// error. Lookup order is undefined.
func (p Package) LookupFunc(f LookupFunc) error {
	var err error
	for n, d := range p.Declarations {
		if err := f(n, d); err != nil {
			break
		}
	}
	if err == StopLookup {
		err = nil
	}
	return err
}

// CombinedPackage implements an ImportablePackage by combining multiple
// packages into one package with name the name of the first package and as
// declarations the declarations of all packages.
//
// The Lookup method calls the Lookup methods of each package in order and
// returns as soon as a package returns a not nil value.
type CombinedPackage []ImportablePackage

// PackageName returns the package name of the first combined package.
func (packages CombinedPackage) PackageName() string {
	if len(packages) == 0 {
		return ""
	}
	return packages[0].PackageName()
}

// Lookup calls the Lookup method of each package in order and returns as soon
// as a combined package returns a declaration.
func (packages CombinedPackage) Lookup(name string) Declaration {
	for _, pkg := range packages {
		if decl := pkg.Lookup(name); decl != nil {
			return decl
		}
	}
	return nil
}

// LookupFunc calls the LookupFunc method of each package in order. As soon as
// f returns StopLookup, LookupFunc returns. If the same declaration name is
// in multiple packages, f is only called with its first occurrence.
func (packages CombinedPackage) LookupFunc(f LookupFunc) error {
	var err error
	names := map[string]struct{}{}
	w := func(name string, decl Declaration) error {
		if _, ok := names[name]; !ok {
			err = f(name, decl)
			names[name] = struct{}{}
		}
		return err
	}
	for _, pkg := range packages {
		_ = pkg.LookupFunc(w)
		if err != nil {
			break
		}
	}
	if err == StopLookup {
		err = nil
	}
	return err
}
