// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/open2b/scriggo/compiler/ast"
)

// ParseTemplate parses the template file with the given path and written in
// language lang, reading the template files from the reader. path, if not
// absolute, is relative to the root of the template. lang can be Text, HTML,
// CSS or JavaScript.
//
// ParseTemplate expands the nodes Extends, Import and ShowPartial parsing the
// relative trees.
//
// The parsed trees are cached so only one call per combination of path and
// context is made to the reader.
func ParseTemplate(path string, reader FileReader, lang ast.Language, packages PackageLoader) (*ast.Tree, error) {

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
		reader:   reader,
		packages: packages,
		trees:    &cache{},
		paths:    []string{},
	}

	tree, err := pp.parseFile(path, lang)
	if err != nil {
		if err2, ok := err.(*SyntaxError); ok && err2.path == "" {
			err2.path = path
		} else if e, ok := err.(*CycleError); ok {
			e.msg = "file " + path + e.msg + ": cycle not allowed"
		} else if os.IsNotExist(err) {
			err = ErrNotExist
		}
		return nil, err
	}

	return tree, nil
}

// templateExpansion represents the state of a template expansion.
type templateExpansion struct {
	reader   FileReader
	trees    *cache
	packages PackageLoader
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

// parseFile parses the file, written in language lang, with the given name.
// name must be absolute and cleared.
func (pp *templateExpansion) parseFile(name string, lang ast.Language) (*ast.Tree, error) {

	// Check if there is a cycle.
	for _, p := range pp.paths {
		if p == name {
			return nil, &CycleError{path: p}
		}
	}

	// Check if it has already been parsed.
	if tree, ok := pp.trees.Get(name, lang); ok {
		return tree, nil
	}
	defer pp.trees.Done(name, lang)

	src, err := pp.reader.ReadFile(name)
	if err != nil {
		return nil, err
	}

	tree, err := ParseTemplateSource(src, lang)
	if err != nil {
		if se, ok := err.(*SyntaxError); ok {
			se.path = name
		}
		return nil, err
	}
	tree.Path = name

	// Expand the nodes.
	pp.paths = append(pp.paths, name)
	err = pp.expand(tree.Nodes)
	pp.paths = pp.paths[:len(pp.paths)-1]
	if err != nil {
		if e, ok := err.(*SyntaxError); ok && e.path == "" {
			e.path = name
		}
		return nil, err
	}

	// Add the tree to the cache.
	pp.trees.Add(name, lang, tree)

	return tree, nil
}

// expand expands the nodes parsing the sub-trees.
func (pp *templateExpansion) expand(nodes []ast.Node) error {

	for _, node := range nodes {

		switch n := node.(type) {

		case *ast.If:

			for {
				err := pp.expand(n.Then.Nodes)
				if err != nil {
					return err
				}
				switch e := n.Else.(type) {
				case *ast.If:
					n = e
					continue
				case *ast.Block:
					err := pp.expand(e.Nodes)
					if err != nil {
						return err
					}
				}
				break
			}

		case *ast.For:

			err := pp.expand(n.Body)
			if err != nil {
				return err
			}

		case *ast.ForRange:

			err := pp.expand(n.Body)
			if err != nil {
				return err
			}

		case *ast.Switch:

			var err error
			for _, c := range n.Cases {
				err = pp.expand(c.Body)
				if err != nil {
					return err
				}
			}

		case *ast.TypeSwitch:

			var err error
			for _, c := range n.Cases {
				err = pp.expand(c.Body)
				if err != nil {
					return err
				}
			}

		case *ast.Select:

			var err error
			for _, c := range n.Cases {
				err = pp.expand(c.Body)
				if err != nil {
					return err
				}
			}

		case *ast.Macro:

			err := pp.expand(n.Body)
			if err != nil {
				return err
			}

		case *ast.Extends:

			if len(pp.paths) > 1 {
				return syntaxError(n.Pos(), "extended, imported and shown paths can not have extends")
			}
			absPath, err := pp.abs(n.Path)
			if err != nil {
				return err
			}
			n.Tree, err = pp.parseFile(absPath, ast.Language(n.Context))
			if err != nil {
				if err == ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if os.IsNotExist(err) {
					err = syntaxError(n.Pos(), "extends path %q does not exist", absPath)
				} else if e, ok := err.(*CycleError); ok {
					e.msg = "\n\textends " + absPath + e.msg
					if e.path == pp.paths[len(pp.paths)-1] {
						e.pos = *(n.Pos())
					}
				}
				return err
			}

		case *ast.Import:

			if ext := filepath.Ext(n.Path); ext == "" {
				// Import a precompiled package (the path has no extension).
				if pp.packages == nil {
					return syntaxError(n.Pos(), "cannot find package %q", n.Path)
				}
				pkg, err := pp.packages.Load(n.Path)
				if err != nil {
					return err
				}
				switch pkg := pkg.(type) {
				case predefinedPackage:
				case nil:
					return syntaxError(n.Pos(), "cannot find package %q", n.Path)
				default:
					return fmt.Errorf("scriggo: unexpected type %T returned by the package loader", pkg)
				}
			} else {
				// Import a template file (the path has an extension).
				absPath, err := pp.abs(n.Path)
				if err != nil {
					return err
				}
				n.Tree, err = pp.parseFile(absPath, ast.Language(n.Context))
				if err != nil {
					if err == ErrInvalidPath {
						err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
					} else if os.IsNotExist(err) {
						err = syntaxError(n.Pos(), "import path %q does not exist", absPath)
					} else if e, ok := err.(*CycleError); ok {
						e.msg = "\n\timports " + absPath + e.msg
						if e.path == pp.paths[len(pp.paths)-1] {
							e.pos = *(n.Pos())
						}
					}
					return err
				}
			}

		case *ast.ShowPartial:

			absPath, err := pp.abs(n.Path)
			if err != nil {
				return err
			}
			n.Tree, err = pp.parseFile(absPath, ast.Language(n.Context))
			if err != nil {
				if err == ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if os.IsNotExist(err) {
					err = syntaxError(n.Pos(), "shown path %q does not exist", absPath)
				} else if e, ok := err.(*CycleError); ok {
					e.msg = "\n\tshows   " + absPath + e.msg
					if e.path == pp.paths[len(pp.paths)-1] {
						e.pos = *(n.Pos())
					}
				}
				return err
			}

		case *ast.Statements:

			err := pp.expand(n.Nodes)
			if err != nil {
				return err
			}

		case *ast.Label:

			err := pp.expand([]ast.Node{n.Statement})
			if err != nil {
				return err
			}

		}

	}

	return nil
}
