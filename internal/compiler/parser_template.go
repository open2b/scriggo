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

func ParseTemplate(path string, reader Reader, main *Package, ctx ast.Context) (*ast.Tree, error) {
	p := &templateParser{
		reader:       reader,
		trees:        &cache{},
		packageInfos: make(map[string]*PackageInfo),
	}

	tree, err := p.parse(path, main, ctx)
	if err != nil {
		return nil, convertError(err)
	}
	return tree, nil
}

func convertError(err error) error {
	if err == ErrInvalidPath {
		return ErrInvalidPath
	}
	if err == ErrNotExist {
		return ErrNotExist
	}
	return err
}

// templateParser implements a templateParser that reads the tree from a Reader and expands
// the nodes Extends, Import and Include. The trees are cached so only one
// call per combination of path and context is made to the reader even if
// several goroutines parse the same paths at the same time.
//
// Returned trees can only be transformed if the templateParser is no longer used,
// because it would be the cached trees to be transformed and a data race can
// occur. In case, use the function Clone in the astutil package to create a
// clone of the tree and then transform the clone.
type templateParser struct {
	reader Reader
	trees  *cache
	// TODO (Gianluca): does packageInfos need synchronized access?

	// TODO(Gianluca): deprecated, remove.
	packageInfos map[string]*PackageInfo // key is path.
}

// parse reads the source at path, with the reader, in the ctx context,
// expands the nodes Extends, Import and Include and returns the expanded tree.
//
// Parse is safe for concurrent use.
func (p *templateParser) parse(path string, main *Package, ctx ast.Context) (*ast.Tree, error) {

	// Path must be absolute.
	if path == "" {
		return nil, ErrInvalidPath
	}
	if path[0] == '/' {
		path = path[1:]
	}
	// Cleans the path by removing "..".
	path, err := toAbsolutePath("/", path)
	if err != nil {
		return nil, err
	}

	pp := &templateExpansion{p.reader, p.trees, map[string]*Package{"main": main}, []string{}}

	tree, err := pp.parsePath(path, ctx)
	if err != nil {
		if err2, ok := err.(*SyntaxError); ok && err2.Path == "" {
			err2.Path = path
		} else if err2, ok := err.(cycleError); ok {
			err = cycleError(path + "\n\t" + string(err2))
		}
		return nil, err
	}

	if len(tree.Nodes) == 0 {
		return nil, &SyntaxError{"", ast.Position{1, 1, 0, 0}, fmt.Errorf("??????????, found 'EOF'")}
	}

	return tree, nil
}

// templateExpansion is a template expansion state.
type templateExpansion struct {
	reader   Reader
	trees    *cache
	packages map[string]*Package
	paths    []string
}

// abs returns path as absolute.
func (pp *templateExpansion) abs(path string) (string, error) {
	var err error
	if path[0] == '/' {
		path, err = toAbsolutePath("/", path[1:])
	} else {
		parent := pp.paths[len(pp.paths)-1]
		dir := parent[:strings.LastIndex(parent, "/")+1]
		path, err = toAbsolutePath(dir, path)
	}
	return path, err
}

// parsePath parses the source at path in context ctx. path must be absolute
// and cleared.
func (pp *templateExpansion) parsePath(path string, ctx ast.Context) (*ast.Tree, error) {

	// Checks if there is a cycle.
	for _, p := range pp.paths {
		if p == path {
			return nil, cycleError(path)
		}
	}

	// Checks if it has already been parsed.
	if tree, ok := pp.trees.Get(path, ctx); ok {
		return tree, nil
	}
	defer pp.trees.Done(path, ctx)

	src, err := pp.reader.Read(path)
	if err != nil {
		return nil, err
	}

	var tree *ast.Tree
	if ctx == ast.ContextGo {
		tree, _, err = ParseSource(src, true, false)
	} else {
		tree, err = ParseTemplateSource(src, ctx)
	}
	if err != nil {
		return nil, err
	}
	tree.Path = path

	// Expands the nodes.
	pp.paths = append(pp.paths, path)
	err = pp.expand(tree.Nodes, ctx)
	if err != nil {
		if e, ok := err.(*SyntaxError); ok && e.Path == "" {
			e.Path = path
		}
		return nil, err
	}
	pp.paths = pp.paths[:len(pp.paths)-1]

	// Adds the tree to the cache.
	pp.trees.Add(path, ctx, tree)

	return tree, nil
}

// expand expands the nodes parsing the sub-trees in context ctx.
func (pp *templateExpansion) expand(nodes []ast.Node, ctx ast.Context) error {

	for _, node := range nodes {

		switch n := node.(type) {

		case *ast.If:

			for {
				err := pp.expand(n.Then.Nodes, ctx)
				if err != nil {
					return err
				}
				switch e := n.Else.(type) {
				case *ast.If:
					n = e
					continue
				case *ast.Block:
					err := pp.expand(e.Nodes, ctx)
					if err != nil {
						return err
					}
				}
				break
			}

		case *ast.For:

			err := pp.expand(n.Body, ctx)
			if err != nil {
				return err
			}

		case *ast.ForRange:

			err := pp.expand(n.Body, ctx)
			if err != nil {
				return err
			}

		case *ast.Switch:

			var err error
			for _, c := range n.Cases {
				err = pp.expand(c.Body, ctx)
				if err != nil {
					return err
				}
			}

		case *ast.TypeSwitch:

			var err error
			for _, c := range n.Cases {
				err = pp.expand(c.Body, ctx)
				if err != nil {
					return err
				}
			}

		case *ast.Select:

			var err error
			for _, c := range n.Cases {
				err = pp.expand(c.Body, ctx)
				if err != nil {
					return err
				}
			}

		case *ast.Macro:

			err := pp.expand(n.Body, ctx)
			if err != nil {
				return err
			}

		case *ast.Extends:

			if len(pp.paths) > 1 {
				return &SyntaxError{"", *(n.Pos()), fmt.Errorf("extended, imported and included paths can not have extends")}
			}
			absPath, err := pp.abs(n.Path)
			if err != nil {
				return err
			}
			n.Tree, err = pp.parsePath(absPath, n.Context)
			if err != nil {
				if err == ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == ErrNotExist {
					err = &SyntaxError{"", *(n.Pos()), fmt.Errorf("extends path %q does not exist", absPath)}
				} else if err2, ok := err.(cycleError); ok {
					err = cycleError("imports " + string(err2))
				}
				return err
			}

		case *ast.Import:

			absPath, err := pp.abs(n.Path)
			if err != nil {
				return err
			}
			n.Tree, err = pp.parsePath(absPath, n.Context)
			if err != nil {
				if err == ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == ErrNotExist {
					err = &SyntaxError{"", *(n.Pos()), fmt.Errorf("import path %q does not exist", absPath)}
				} else if err2, ok := err.(cycleError); ok {
					err = cycleError("imports " + string(err2))
				}
				return err
			}

		case *ast.Include:

			absPath, err := pp.abs(n.Path)
			if err != nil {
				return err
			}
			n.Tree, err = pp.parsePath(absPath, n.Context)
			if err != nil {
				if err == ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == ErrNotExist {
					err = &SyntaxError{"", *(n.Pos()), fmt.Errorf("included path %q does not exist", absPath)}
				} else if err2, ok := err.(cycleError); ok {
					err = cycleError("include " + string(err2))
				}
				return err
			}

		case *ast.Text:
			// Nothing to do.

		case *ast.Show:
			// Nothing to do.

		// TODO: to remove.
		default:
			panic(fmt.Errorf("unexpected node %s", node))

		}

	}

	return nil
}
