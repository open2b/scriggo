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
	AllowShebangLine   bool
	DisallowGoStmt     bool
	LimitMemorySize    bool
	PackageLess        bool
	TemplateContext    ast.Context
	TemplateFailOnTODO bool
	TreeTransformer    func(*ast.Tree) error
}

func CompileProgram(r io.Reader, importer PackageLoader, opts Options) (*Code, error) {
	var tree *ast.Tree

	// Parse the source code.
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
	checkerOpts.SyntaxType = ProgramSyntax
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
	emitterOpts.MemoryLimit = opts.LimitMemorySize
	if opts.PackageLess {
		code = EmitPackageLessProgram(tree, typeInfos, tci["main"].IndirectVars, emitterOpts)
	} else {
		code = EmitPackageMain(tree.Nodes[0].(*ast.Package), typeInfos, tci["main"].IndirectVars, emitterOpts)
	}

	return code, nil
}

func CompileTemplate(r Reader, path string, main PackageLoader, opts Options) (*Code, error) {

	var tree *ast.Tree

	// Parse the source code.
	var err error
	tree, err = ParseTemplate(path, r, ast.Context(opts.TemplateContext))
	if err != nil {
		return nil, err
	}

	// Transform the tree.
	if opts.TreeTransformer != nil {
		err := opts.TreeTransformer(tree)
		if err != nil {
			return nil, err
		}
	}

	// Type check the tree.
	checkerOpts := CheckerOptions{
		AllowNotUsed:   true,
		DisallowGoStmt: opts.DisallowGoStmt,
		FailOnTODO:     opts.TemplateFailOnTODO,
		PackageLess:    opts.PackageLess,
		SyntaxType:     TemplateSyntax,
	}
	tci, err := Typecheck(tree, main, checkerOpts)
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
	emitterOpts := EmitterOptions{}
	emitterOpts.MemoryLimit = opts.LimitMemorySize
	code := EmitTemplate(tree, typeInfos, tci["main"].IndirectVars, emitterOpts)

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
