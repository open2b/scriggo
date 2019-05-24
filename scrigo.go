// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"fmt"
	"reflect"
	"strings"

	"scrigo/internal/compiler"
	"scrigo/internal/compiler/ast"
	"scrigo/native"
	"scrigo/vm"
)

type Program struct {
	Fn      *vm.ScrigoFunction
	globals []compiler.Global
}

func Compile(path string, reader compiler.Reader, packages map[string]*native.GoPackage, alloc bool) (*Program, error) {
	p := NewParser(reader, packages)
	tree, deps, err := p.Parse(path)
	if err != nil {
		return nil, err
	}

	opts := &compiler.Options{
		IsPackage: true,
	}
	tci, err := compiler.Typecheck(opts, tree, nil, packages, deps, nil)
	if err != nil {
		return nil, err
	}

	typeInfos := map[ast.Node]*compiler.TypeInfo{}
	for _, pkgInfos := range tci {
		for node, ti := range pkgInfos.TypeInfo {
			typeInfos[node] = ti
		}
	}
	pkgMain := compiler.EmitPackageMain(tree.Nodes[0].(*ast.Package), packages, typeInfos, tci[path].IndirectVars, alloc)
	globals := make([]compiler.Global, len(pkgMain.Globals))
	for i, global := range pkgMain.Globals {
		globals[i].Pkg = global.Pkg
		globals[i].Name = global.Name
		globals[i].Type = global.Type
		globals[i].Value = global.Value
	}
	return &Program{Fn: pkgMain.Main, globals: globals}, nil
}

// TODO(Gianluca): second parameter "tf" should be removed: setting the trace
// function must be done in some other way.
func Execute(p *Program, tf *vm.TraceFunc, freeMemory int) error {
	vmm := vm.New()
	if n := len(p.globals); n > 0 {
		globals := make([]interface{}, n)
		for i, global := range p.globals {
			if global.Value == nil {
				globals[i] = reflect.New(global.Type).Interface()
			} else {
				globals[i] = global.Value
			}
		}
		vmm.SetGlobals(globals)
	}
	if tf != nil {
		vmm.SetTraceFunc(*tf)
	}
	vmm.SetFreeMemory(freeMemory)
	_, err := vmm.Run(p.Fn)
	return err
}

// Parser implements a parser that reads the tree from a Reader and expands
// the nodes Extends, Import and Include. The trees are compiler.Cached so only one
// call per combination of path and context is made to the reader even if
// several goroutines parse the same paths at the same time.
//
// Returned trees can only be transformed if the parser is no longer used,
// because it would be the compiler.Cached trees to be transformed and a data race can
// occur. In case, use the function Clone in the astutil package to create a
// clone of the tree and then transform the clone.
type Parser struct {
	reader   compiler.Reader
	packages map[string]*native.GoPackage
	trees    *compiler.Cache
	// TODO (Gianluca): does packageInfos need synchronized access?
	packageInfos map[string]*compiler.PackageInfo // key is path.
	typeCheck    bool
}

// NewParser returns a new Parser that reads the trees from the reader r. typeCheck
// indicates if a type-checking must be done after parsing.
func NewParser(r compiler.Reader, packages map[string]*native.GoPackage) *Parser {
	p := &Parser{
		reader:   r,
		packages: packages,
		trees:    &compiler.Cache{},
	}
	return p
}

// Parse reads the source at path, with the reader, in the ctx context,
// expands the nodes Extends, Import and Include and returns the expanded tree.
//
// Parse is safe for concurrent use.
func (p *Parser) Parse(path string) (*ast.Tree, compiler.GlobalsDependencies, error) {

	// Path must be absolute.
	if path == "" {
		return nil, nil, compiler.ErrInvalidPath
	}
	if path[0] == '/' {
		path = path[1:]
	}
	// Cleans the path by removing "..".
	path, err := compiler.ToAbsolutePath("/", path)
	if err != nil {
		return nil, nil, err
	}

	pp := &expansion{p.reader, p.trees, p.packages, []string{}}

	tree, deps, err := pp.parsePath(path)
	if err != nil {
		if err2, ok := err.(*compiler.SyntaxError); ok && err2.Path == "" {
			err2.Path = path
		} else if err2, ok := err.(compiler.CycleError); ok {
			err = compiler.CycleError(path + "\n\t" + string(err2))
		}
		return nil, nil, err
	}
	if len(tree.Nodes) == 0 {
		return nil, nil, &compiler.SyntaxError{"", ast.Position{1, 1, 0, 0}, fmt.Errorf("expected 'package' or script, found 'EOF'")}
	}

	return tree, deps, nil
}

// TypeCheckInfos returns the type-checking infos collected during
// type-checking.
func (p *Parser) TypeCheckInfos() map[string]*compiler.PackageInfo {
	return p.packageInfos
}

// expansion is an expansion state.
type expansion struct {
	reader   compiler.Reader
	trees    *compiler.Cache
	packages map[string]*native.GoPackage
	paths    []string
}

// abs returns path as absolute.
func (pp *expansion) abs(path string) (string, error) {
	var err error
	if path[0] == '/' {
		path, err = compiler.ToAbsolutePath("/", path[1:])
	} else {
		parent := pp.paths[len(pp.paths)-1]
		dir := parent[:strings.LastIndex(parent, "/")+1]
		path, err = compiler.ToAbsolutePath(dir, path)
	}
	return path, err
}

// parsePath parses the source at path in context ctx. path must be absolute
// and cleared.
func (pp *expansion) parsePath(path string) (*ast.Tree, compiler.GlobalsDependencies, error) {

	// Checks if there is a cycle.
	for _, p := range pp.paths {
		if p == path {
			return nil, nil, compiler.CycleError(path)
		}
	}

	// Checks if it has already been parsed.
	if tree, ok := pp.trees.Get(path, ast.ContextNone); ok {
		return tree, nil, nil
	}
	defer pp.trees.Done(path, ast.ContextNone)

	src, err := pp.reader.Read(path, ast.ContextNone)
	if err != nil {
		return nil, nil, err
	}

	tree, deps, err := compiler.ParseSource(src, true, ast.ContextNone)
	if err != nil {
		return nil, nil, err
	}
	tree.Path = path

	// Expands the nodes.
	pp.paths = append(pp.paths, path)
	expandedDeps, err := pp.expand(tree.Nodes)
	if err != nil {
		if e, ok := err.(*compiler.SyntaxError); ok && e.Path == "" {
			e.Path = path
		}
		return nil, nil, err
	}
	for k, v := range expandedDeps {
		deps[k] = v
	}
	pp.paths = pp.paths[:len(pp.paths)-1]

	// Adds the tree to the compiler.Cache.
	pp.trees.Add(path, ast.ContextNone, tree)

	return tree, deps, nil
}

// expand expands the nodes parsing the sub-trees in context ctx.
func (pp *expansion) expand(nodes []ast.Node) (compiler.GlobalsDependencies, error) {

	allDeps := compiler.GlobalsDependencies{}

	for _, node := range nodes {

		switch n := node.(type) {

		case *ast.Package:

			deps, err := pp.expand(n.Declarations)
			if err != nil {
				return nil, err
			}
			for k, v := range deps {
				allDeps[k] = v
			}

		case *ast.Import:

			absPath, err := pp.abs(n.Path)
			if err != nil {
				return nil, err
			}
			found := false
			for path := range pp.packages {
				if path == n.Path {
					found = true
					break
				}
			}
			if found {
				continue
			}
			var deps compiler.GlobalsDependencies
			n.Tree, deps, err = pp.parsePath(absPath + ".go")
			if err != nil {
				if err == compiler.ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == compiler.ErrNotExist {
					err = &compiler.SyntaxError{"", *(n.Pos()), fmt.Errorf("cannot find package \"%s\"", n.Path)}
				} else if err2, ok := err.(compiler.CycleError); ok {
					err = compiler.CycleError("imports " + string(err2))
				}
				return nil, err
			}
			for k, v := range deps {
				allDeps[k] = v
			}

		}

	}

	return allDeps, nil
}
