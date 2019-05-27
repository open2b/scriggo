// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"strings"

	"scrigo/internal/compiler/ast"
)

type PackageImporter interface{}

// ParseProgram parses a program reading its sources from packages.
func ParseProgram(packages []PackageImporter) (*ast.Tree, GlobalsDependencies, map[string]*PredefinedPackage, error) {
	p := &programParser{
		trees:    &cache{},
		packages: packages,
	}
	tree, deps, predefinedPackages, err := p.parse("/main")
	if err != nil {
		return nil, nil, nil, err
	}
	return tree, deps, predefinedPackages, nil
}

// programParser implements a programParser that reads the tree from a Reader and expands
// the nodes Extends, Import and Include. The trees are Cached so only one
// call per combination of path and context is made to the reader even if
// several goroutines parse the same paths at the same time.
//
// Returned trees can only be transformed if the programParser is no longer used,
// because it would be the Cached trees to be transformed and a data race can
// occur. In case, use the function Clone in the astutil package to create a
// clone of the tree and then transform the clone.
type programParser struct {
	trees *cache
	// TODO (Gianluca): does packageInfos need synchronized access?
	packageInfos map[string]*PackageInfo // key is path.
	typeCheck    bool
	packages     []PackageImporter
}

// parse reads the source at path, with the reader, in the ctx context,
// expands the nodes Extends, Import and Include and returns the expanded tree.
//
// parse is safe for concurrent use.
func (p *programParser) parse(path string) (*ast.Tree, GlobalsDependencies, map[string]*PredefinedPackage, error) {

	// Path must be absolute.
	if path == "" {
		return nil, nil, nil, ErrInvalidPath
	}
	if path[0] == '/' {
		path = path[1:]
	}
	// Cleans the path by removing "..".
	path, err := ToAbsolutePath("/", path)
	if err != nil {
		return nil, nil, nil, err
	}

	pp := &programExpansion{
		trees:          p.trees,
		packages:       p.packages,
		predefinedPkgs: map[string]*PredefinedPackage{},
		paths:          []string{},
	}

	tree, deps, err := pp.parsePath(path)
	if err != nil {
		if err2, ok := err.(*SyntaxError); ok && err2.Path == "" {
			err2.Path = path
		} else if err2, ok := err.(CycleError); ok {
			err = CycleError(path + "\n\t" + string(err2))
		}
		return nil, nil, nil, err
	}
	if len(tree.Nodes) == 0 {
		return nil, nil, nil, &SyntaxError{"", ast.Position{1, 1, 0, 0}, fmt.Errorf("expected 'package' or script, found 'EOF'")}
	}

	return tree, deps, pp.predefinedPkgs, nil
}

// TypeCheckInfos returns the type-checking infos collected during
// type-checking.
func (p *programParser) typeCheckInfos() map[string]*PackageInfo {
	return p.packageInfos
}

// programExpansion is an programExpansion state.
type programExpansion struct {
	// reader Reader
	trees *cache
	// packages map[string]*PredefinedPackage
	packages       []PackageImporter
	predefinedPkgs map[string]*PredefinedPackage
	paths          []string
}

// abs returns path as absolute.
func (pp *programExpansion) abs(path string) (string, error) {
	var err error
	if path[0] == '/' {
		path, err = ToAbsolutePath("/", path[1:])
	} else {
		parent := pp.paths[len(pp.paths)-1]
		dir := parent[:strings.LastIndex(parent, "/")+1]
		path, err = ToAbsolutePath(dir, path)
	}
	return path, err
}

// parsePath parses the source at path in context ctx. path must be absolute
// and cleared.
func (pp *programExpansion) parsePath(path string) (*ast.Tree, GlobalsDependencies, error) {

	// Checks if there is a cycle.
	for _, p := range pp.paths {
		if p == path {
			return nil, nil, CycleError(path)
		}
	}

	// Checks if it has already been parsed.
	if tree, ok := pp.trees.Get(path, ast.ContextGo); ok {
		return tree, nil, nil
	}
	defer pp.trees.Done(path, ast.ContextGo)

	src, ok := lookupSource(pp.packages, path)
	if !ok {
		panic("not found")
	}

	tree, deps, err := ParseSource(src, true, false)
	if err != nil {
		return nil, nil, err
	}
	tree.Path = path

	// Expands the nodes.
	pp.paths = append(pp.paths, path)
	expandedDeps, err := pp.expand(tree.Nodes)
	if err != nil {
		if e, ok := err.(*SyntaxError); ok && e.Path == "" {
			e.Path = path
		}
		return nil, nil, err
	}
	for k, v := range expandedDeps {
		deps[k] = v
	}
	pp.paths = pp.paths[:len(pp.paths)-1]

	// Adds the tree to the Cache.
	pp.trees.Add(path, ast.ContextGo, tree)

	return tree, deps, nil
}

// expand expands the nodes parsing the sub-trees in context ctx.
func (pp *programExpansion) expand(nodes []ast.Node) (GlobalsDependencies, error) {

	allDeps := GlobalsDependencies{}

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

			pkg, foundPredef := lookupPredefined(pp.packages, n.Path)
			if foundPredef {
				pp.predefinedPkgs[n.Path] = pkg
				continue
			}
			if _, foundSrc := lookupSource(pp.packages, n.Path); foundSrc || foundPredef {
				continue
			}

			var deps GlobalsDependencies
			n.Tree, deps, err = pp.parsePath(absPath + ".go")
			if err != nil {
				if err == ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == ErrNotExist {
					err = &SyntaxError{"", *(n.Pos()), fmt.Errorf("cannot find package \"%s\"", n.Path)}
				} else if err2, ok := err.(CycleError); ok {
					err = CycleError("imports " + string(err2))
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

// lookupSource searches for a source located at path in packages.
func lookupSource(packages []PackageImporter, path string) ([]byte, bool) {
	for _, pi := range packages {
		switch pi := pi.(type) {
		case Reader:
			data, err := pi.Read(path)
			if err == nil {
				return data, true
			}
		case map[string][]byte:
			if data, ok := pi[path]; ok {
				return data, true
			}
		case func(path string) ([]byte, bool):
			data, ok := pi(path)
			if ok {
				return data, true
			}
		case map[string]*PredefinedPackage:
			// Nothing to do.
		case func(path string) (*PredefinedPackage, bool):
			// Nothing to do.
		default:
			panic(fmt.Sprintf("unsupported type %T", pi)) // TODO(Gianluca): to review.
		}
	}
	return nil, false
}

// lookupPredefined searches for a predefined package located at path in packages.
func lookupPredefined(packages []PackageImporter, path string) (*PredefinedPackage, bool) {
	for _, pi := range packages {
		switch pi := pi.(type) {
		case func(path string) (*PredefinedPackage, bool):
			pkg, ok := pi(path)
			if ok {
				return pkg, true
			}
		case map[string]*PredefinedPackage:
			pkg, ok := pi[path]
			if ok {
				return pkg, true
			}
		case Reader:
			// Nothing to do.
		case map[string][]byte:
			// Nothing to do.
		default:
			panic(fmt.Sprintf("unsupported type %T", pi)) // TODO(Gianluca): to review.
		}
	}
	return nil, false
}
