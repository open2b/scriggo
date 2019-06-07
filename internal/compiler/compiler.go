// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package compiler implements parsing, type-checking and emitting of sources.
//
// Parsing
//
// Parsing is done using
//
//	ParseTemplate(..)
//	ParseProgram(..)
//	ParseScript(..)
//
// Typechecking
//
// When parsing is done, tree can be type-checked by:
//
// 	Typecheck(...)
//
// Emitting
//
// To emit a type-checked tree, use:
//
//  EmitTemplate(..)
//  EmitPackageMain(..)
//  EmitScript(..)
//
package compiler

import (
	"reflect"

	"scriggo/internal/compiler/ast"
)

// Package represents a predefined package.
// A package can contain a function, a variable, a constant or a type.
//
//		Function: assign function value to Declarations as is.
//		Variable: assign the address of value to Declarations.
//		Constant: TODO(Gianluca).
//		Type:     assign the reflect.TypeOf of type to Declarations.
type Package struct {
	// Package name.
	Name         string
	// Package declarations.
	Declarations map[string]interface{}
}

type ConstantValue struct {
	value interface{}
	typ   reflect.Type // nil for untyped constants.
}

// Constant returns a predefined constant given its type and value. Can be
// used as a declaration of a precompiled package. For untyped constants the
// type is nil.
func Constant(typ reflect.Type, value interface{}) ConstantValue {
	return ConstantValue{value: value, typ: typ}
}

// PackageLoader is implemented by package loaders. Load returns a predefined
// package as *Package or the source of a non predefined package as
// an io.Reader.
//
// If the package does not exist it returns nil and nil.
// If the package exists but there was an error while loading the package, it
// returns nil and the error.
type PackageLoader interface {
	Load(pkgPath string) (interface{}, error)
}

type Options struct {

	// AllowImports makes import statements available.
	AllowImports bool

	// NotUsedError returns a checking error if a variable is declared and not
	// used or a package is imported and not used.
	NotUsedError bool

	// IsPackage indicate if it's a package. If true, all sources must start
	// with "package" and package-level declarations will be sorted according
	// to Go package inizialization specs.
	IsPackage bool

	// DisallowGoStmt disables the "go" statement.
	DisallowGoStmt bool
}

func Typecheck(tree *ast.Tree, predefinedPkgs map[string]*Package, deps GlobalsDependencies, opts *Options) (_ map[string]*PackageInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(*CheckingError); ok {
				err = rerr
			} else {
				panic(r)
			}
		}
	}()
	tc := newTypechecker(tree.Path, true, opts.DisallowGoStmt)
	tc.Universe = universe
	if main, ok := predefinedPkgs["main"]; ok {
		tc.Scopes = append(tc.Scopes, ToTypeCheckerScope(main))
	}
	if opts.IsPackage {
		pkgInfos := map[string]*PackageInfo{}
		err := checkPackage(tree, deps, predefinedPkgs, pkgInfos, opts.DisallowGoStmt)
		if err != nil {
			return nil, err
		}
		return pkgInfos, nil
	}
	tc.predefinedPkgs = predefinedPkgs
	tc.CheckNodesInNewScope(tree.Nodes)
	mainPkgInfo := &PackageInfo{}
	mainPkgInfo.IndirectVars = tc.IndirectVars
	mainPkgInfo.TypeInfo = tc.TypeInfo
	return map[string]*PackageInfo{"main": mainPkgInfo}, nil
}
