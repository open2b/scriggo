// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The compiler implements parsing, type-checking and emitting of sources.
//
// Parsing
//
// Parsing is done using
//
//	ParseTemplate(...)
//	ParseProgram(...)
//	ParseScript(...) (currently not implemented)
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
//    EmitSingle(...)
//    EmitPackageMain(...)
//
package compiler

import (
	"reflect"

	"scrigo/internal/compiler/ast"
)

type Package struct {
	Name         string
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

func Typecheck(opts *Options, tree *ast.Tree, packages map[string]*Package, imports map[string]*Package, deps GlobalsDependencies, customBuiltins TypeCheckerScope) (_ map[string]*PackageInfo, err error) {
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
	if customBuiltins != nil {
		tc.Scopes = append(tc.Scopes, customBuiltins)
	}
	if main, ok := packages["main"]; ok {
		tc.Scopes = append(tc.Scopes, ToTypeCheckerScope(main))
	}
	if opts.IsPackage {
		pkgInfos := map[string]*PackageInfo{}
		err := checkPackage(tree, deps, imports, pkgInfos, opts.DisallowGoStmt)
		if err != nil {
			return nil, err
		}
		return pkgInfos, nil
	}
	tc.packages = packages
	tc.CheckNodesInNewScope(tree.Nodes)
	mainPkgInfo := &PackageInfo{}
	mainPkgInfo.IndirectVars = tc.IndirectVars
	mainPkgInfo.TypeInfo = tc.TypeInfo
	return map[string]*PackageInfo{"main": mainPkgInfo}, nil
}
