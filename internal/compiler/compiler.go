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
	Name string
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

	// TODO(Gianluca): review doc.
	// IsProgram indicate if it's a package. If true, all sources must start
	// with "package" and package-level declarations will be sorted according
	// to Go package inizialization specs.
	IsProgram bool

	IsTemplate bool

	IsScript bool

	// DisallowGoStmt disables the "go" statement.
	DisallowGoStmt bool
}

func Typecheck(tree *ast.Tree, predefinedPkgs map[string]*Package, deps GlobalsDependencies, opts *Options) (map[string]*PackageInfo, error) {
	// TODO(Gianluca): review.
	tc := newTypechecker(tree.Path, opts.IsScript, opts.IsTemplate, opts.DisallowGoStmt)
	tc.Universe = universe
	if main, ok := predefinedPkgs["main"]; ok {
		tc.Scopes = append(tc.Scopes, ToTypeCheckerScope(main))
	}
	if opts.IsProgram { // TODO(Gianluca): move this code before creating a newTypechecker.
		pkgInfos := map[string]*PackageInfo{}
		err := checkPackage(tree.Nodes[0].(*ast.Package), tree.Path, deps, predefinedPkgs, pkgInfos, opts.IsTemplate, opts.DisallowGoStmt)
		if err != nil {
			return nil, err
		}
		return pkgInfos, nil
	}
	if opts.IsTemplate {
		if extends, ok := tree.Nodes[0].(*ast.Extends); ok {
			for _, d := range tree.Nodes[1:] {
				if m, ok := d.(*ast.Macro); ok {
					f := macroToFunc(m)
					tc.filePackageBlock[f.Ident.Name] = scopeElement{t: &TypeInfo{Type: tc.typeof(f.Type, noEllipses).Type}}
				}
			}
			err := tc.CheckNodesInNewScopeCatchingPanics(extends.Tree.Nodes)
			if err != nil {
				return nil, err
			}
			err = tc.templateToPackage(tree)
			if err != nil {
				return nil, err
			}
			pkgInfos := map[string]*PackageInfo{}
			err = checkPackage(tree.Nodes[0].(*ast.Package), tree.Path, deps, nil, pkgInfos, true, true)
			if err != nil {
				return nil, err
			}
			mainPkgInfo := &PackageInfo{}
			mainPkgInfo.IndirectVars = tc.IndirectVars
			mainPkgInfo.TypeInfo = tc.TypeInfo
			for _, pkgInfo := range pkgInfos {
				for k, v := range pkgInfo.TypeInfo {
					mainPkgInfo.TypeInfo[k] = v
				}
				for k, v := range pkgInfo.IndirectVars {
					mainPkgInfo.IndirectVars[k] = v
				}
			}
			return map[string]*PackageInfo{"main": mainPkgInfo}, nil
		}
	}
	tc.predefinedPkgs = predefinedPkgs
	err := tc.CheckNodesInNewScopeCatchingPanics(tree.Nodes)
	if err != nil {
		return nil, err
	}
	mainPkgInfo := &PackageInfo{}
	mainPkgInfo.IndirectVars = tc.IndirectVars
	mainPkgInfo.TypeInfo = tc.TypeInfo
	return map[string]*PackageInfo{"main": mainPkgInfo}, nil
}
