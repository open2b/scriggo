// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"io"
	"io/ioutil"

	"scriggo/ast"
)

type Options struct {
	AllowShebangLine bool
	DisallowGoStmt   bool
	LimitMemorySize  bool
	PackageLess      bool
	TreeTransformer  func(*ast.Tree) error
	Template         struct {
		Path    string
		Context ast.Context
	}
}

// Compiles compiles a program or a template returning the compiled code. If
// compiling a program then r must be an io.Reader; otherwise, when compiling a
// template, r must be a Reader.
// Importer is a package loader that makes available packages for the import;
// importer can also have a package with path "main" that holds the definitions
// of the package level declarations of a package-less program or template.
// In case of compilation error such error is returned.
func Compile(r interface{}, importer PackageLoader, opts Options) (*Code, error) {

	var tree *ast.Tree
	var isTemplate bool

	// Parse the source code.
	switch r := r.(type) {
	case io.Reader:
		isTemplate = false
		if opts.PackageLess {
			var err error
			tree, err = ParsePackageLessProgram(r, importer, opts.AllowShebangLine)
			if err != nil {
				return nil, err
			}
		} else {
			mainSrc, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			tree, err = ParseProgram(mainCombiner{mainSrc, importer})
			if err != nil {
				return nil, err
			}
		}
	case Reader:
		isTemplate = true
		var err error
		tree, err = ParseTemplate(opts.Template.Path, r, ast.Context(opts.Template.Context))
		if err != nil {
			return nil, err
		}
	default:
		panic("invalid type for argument r: expecting an io.Reader or a Reader")
	}

	// Transform the tree.
	if opts.TreeTransformer != nil {
		err := opts.TreeTransformer(tree)
		if err != nil {
			return nil, err
		}
	}

	// Type check the tree.
	checkerOpts := CheckerOptions{PackageLess: opts.PackageLess}
	checkerOpts.DisallowGoStmt = opts.DisallowGoStmt
	if isTemplate {
		checkerOpts.SyntaxType = TemplateSyntax
		checkerOpts.AllowNotUsed = true
	} else {
		checkerOpts.SyntaxType = ProgramSyntax
	}
	tci, err := Typecheck(tree, importer, checkerOpts)
	if err != nil {
		return nil, err
	}
	typeInfos := map[ast.Node]*TypeInfo{}
	for _, pkgInfos := range tci {
		for node, ti := range pkgInfos.TypeInfos {
			typeInfos[node] = ti
		}
	}

	// Emit the code.
	var code *Code
	emitterOpts := EmitterOptions{}
	checkerOpts.DisallowGoStmt = opts.DisallowGoStmt
	emitterOpts.MemoryLimit = opts.LimitMemorySize
	switch {
	case isTemplate:
		code = EmitTemplate(tree, typeInfos, tci["main"].IndirectVars, emitterOpts)
	case opts.PackageLess:
		code = EmitPackageLessProgram(tree, typeInfos, tci["main"].IndirectVars, emitterOpts)
	default:
		code = EmitPackageMain(tree.Nodes[0].(*ast.Package), typeInfos, tci["main"].IndirectVars, emitterOpts)
	}

	return code, nil

}

type mainCombiner struct {
	mainSrc      []byte
	otherImports PackageLoader
}

func (ml mainCombiner) Load(path string) (interface{}, error) {
	if path == "main" {
		return bytes.NewReader(ml.mainSrc), nil
	}
	return ml.otherImports.Load(path)
}
