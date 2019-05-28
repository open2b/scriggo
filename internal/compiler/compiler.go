// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"scrigo/internal/compiler/ast"
)

type Options struct {
	AllowImports   bool
	NotUsedError   bool
	IsPackage      bool
	DisallowGoStmt bool
}

func Typecheck(opts *Options, tree *ast.Tree, main *PredefinedPackage, imports map[string]*PredefinedPackage, deps GlobalsDependencies, customBuiltins TypeCheckerScope) (_ map[string]*PackageInfo, err error) {
	if opts.IsPackage && main != nil {
		panic("cannot have package main with option IsPackage enabled")
	}
	if opts.IsPackage && customBuiltins != nil {
		panic("cannot have customBuiltins with option IsPackage enabled")
	}
	if imports != nil && !opts.IsPackage {
		panic("cannot have imports when checking a non-package")
	}
	if deps != nil && !opts.IsPackage {
		panic("cannot have deps when checking a non-package")
	}
	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(*Error); ok {
				err = rerr
			} else {
				panic(r)
			}
		}
	}()
	tc := NewTypechecker(tree.Path, true, opts.DisallowGoStmt)
	tc.Universe = Universe
	if customBuiltins != nil {
		tc.Scopes = append(tc.Scopes, customBuiltins)
	}
	if main != nil {
		tc.Scopes = append(tc.Scopes, ToTypeCheckerScope(main))
	}
	if opts.IsPackage {
		pkgInfos := map[string]*PackageInfo{}
		err := CheckPackage(tree, deps, imports, pkgInfos, opts.DisallowGoStmt)
		if err != nil {
			return nil, err
		}
		return pkgInfos, nil
	}
	tc.CheckNodesInNewScope(tree.Nodes)
	mainPkgInfo := &PackageInfo{}
	mainPkgInfo.IndirectVars = tc.IndirectVars
	mainPkgInfo.TypeInfo = tc.TypeInfo
	return map[string]*PackageInfo{"main": mainPkgInfo}, nil
}
