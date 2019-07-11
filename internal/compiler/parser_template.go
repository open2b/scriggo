// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"strings"

	"scriggo/ast"
)

// ParseTemplate parses the template with the given path, reading the template
// files from the reader, in context ctx.
//
// ParseTemplate expands the nodes Extends, Import and Include parsing the
// relative trees. The parsed trees are cached so only one call per
// combination of path and context is made to the reader.
func ParseTemplate(path string, reader Reader, ctx ast.Context) (*ast.Tree, error) {

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

	pp := &templateExpansion{
		reader: reader,
		trees:  &cache{},
		paths:  []string{},
	}

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

// templateExpansion represents the state of a template expansion.
type templateExpansion struct {
	reader Reader
	trees  *cache
	paths  []string
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

// parsePath parses the source at the given path in context ctx. path must be
// absolute and cleared.
func (pp *templateExpansion) parsePath(path string, ctx ast.Context) (*ast.Tree, error) {

	// Check if there is a cycle.
	for _, p := range pp.paths {
		if p == path {
			return nil, cycleError(path)
		}
	}

	// Check if it has already been parsed.
	if tree, ok := pp.trees.Get(path, ctx); ok {
		return tree, nil
	}
	defer pp.trees.Done(path, ctx)

	src, err := pp.reader.Read(path)
	if err != nil {
		return nil, err
	}

	tree, _, err := ParseTemplateSource(src, ctx)
	if err != nil {
		if se, ok := err.(*SyntaxError); ok {
			se.Path = path
		}
		return nil, err
	}
	tree.Path = path

	// Expand the nodes.
	pp.paths = append(pp.paths, path)
	err = pp.expand(tree.Nodes, ctx)
	if err != nil {
		if e, ok := err.(*SyntaxError); ok && e.Path == "" {
			e.Path = path
		}
		return nil, err
	}
	pp.paths = pp.paths[:len(pp.paths)-1]

	// Add the tree to the cache.
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

		case *ast.Label:

			err := pp.expand([]ast.Node{n.Statement}, ctx)
			if err != nil {
				return err
			}

		}

	}

	return nil
}
