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
		trees:    map[string]parsedTree{},
		paths:    []string{},
	}

	tree, err := pp.parseFile(path, lang, false)
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
	trees    map[string]parsedTree
	packages PackageLoader
	paths    []string
}

// parsedTree represents a parsed tree. parent is the file path and node that
// extends, imports or shows the tree.
type parsedTree struct {
	tree   *ast.Tree
	parent struct {
		path string
		node ast.Node
	}
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

// parseNodeFile parses the file referenced by an Extends, Import or
// ShowPartial node and returns its tree.
func (pp *templateExpansion) parseNodeFile(node ast.Node) (*ast.Tree, error) {

	var err error
	var path string
	var lang ast.Language
	var declarationsFile bool

	// Get the file's absolute path, its language and if it should be
	// a declarations file.
	switch n := node.(type) {
	case *ast.Extends:
		path = n.Path
		lang = ast.Language(n.Context)
	case *ast.Import:
		path = n.Path
		lang = ast.Language(n.Context)
		declarationsFile = true
	case *ast.ShowPartial:
		path = n.Path
		lang = ast.Language(n.Context)
	}
	path, err = pp.abs(path)
	if err != nil {
		return nil, err
	}

	// Check if there is a cycle.
	for _, p := range pp.paths {
		if p == path {
			return nil, &CycleError{path: path}
		}
	}

	// Check if it has already been parsed.
	var tree *ast.Tree
	if parsed, ok := pp.trees[path]; ok {
		switch n := parsed.parent.node.(type) {
		case *ast.Extends:
			switch node.(type) {
			case *ast.Import:
				return nil, syntaxError(node.Pos(), "import of file extended at %s:%s", parsed.parent.path, n.Pos())
			case *ast.ShowPartial:
				return nil, syntaxError(node.Pos(), "show of file extended at %s:%s", parsed.parent.path, n.Pos())
			}
		case *ast.Import:
			if _, ok := node.(*ast.ShowPartial); ok {
				return nil, syntaxError(node.Pos(), "show of file imported at %s:%s", parsed.parent.path, n.Pos())
			}
		case *ast.ShowPartial:
			if _, ok := node.(*ast.Import); ok {
				return nil, syntaxError(node.Pos(), "import of file showed at %s:%s", parsed.parent.path, n.Pos())
			}
		}
		tree = parsed.tree
	}

	if tree == nil {
		// Parse the file.
		tree, err = pp.parseFile(path, lang, declarationsFile)
		if err != nil {
			return nil, err
		}
		parsed := parsedTree{tree: tree}
		parsed.parent.path = pp.paths[len(pp.paths)-1]
		parsed.parent.node = node
		pp.trees[path] = parsed
	}

	return tree, nil
}

// parseFile parses the file with the given path, written in language lang.
// declarationsFile indicates whether src should be a declarations file.
// path must be absolute and cleared.
func (pp *templateExpansion) parseFile(path string, lang ast.Language, declarationsFile bool) (*ast.Tree, error) {

	src, err := pp.reader.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tree, err := ParseTemplateSource(src, lang, declarationsFile)
	if err != nil {
		if se, ok := err.(*SyntaxError); ok {
			se.path = path
		}
		return nil, err
	}
	tree.Path = path

	// Expand the nodes.
	pp.paths = append(pp.paths, path)
	err = pp.expand(tree.Nodes)
	pp.paths = pp.paths[:len(pp.paths)-1]
	if err != nil {
		if e, ok := err.(*SyntaxError); ok && e.path == "" {
			e.path = path
		}
		return nil, err
	}

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
			var err error
			n.Tree, err = pp.parseNodeFile(n)
			if err != nil {
				absPath, _ := pp.abs(n.Path)
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
				var err error
				n.Tree, err = pp.parseNodeFile(n)
				if err != nil {
					absPath, _ := pp.abs(n.Path)
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

			var err error
			n.Tree, err = pp.parseNodeFile(n)
			if err != nil {
				absPath, _ := pp.abs(n.Path)
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
