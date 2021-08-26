// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/open2b/scriggo/ast"
)

// FormatFS is the interface implemented by a file system that can determine
// the file format from a path name.
type FormatFS interface {
	fs.FS
	Format(name string) (ast.Format, error)
}

// ParseTemplate parses the named template file rooted at the given file
// system. If fsys implements FormatFS, the file format is read from its
// Format method, otherwise it depends on the extension of the file name.
// Any error related to the compilation itself is returned as a CompilerError.
//
// If noParseShow is true, short show statements are not parsed.
//
// ParseTemplate expands the nodes Extends, Import and Render parsing the
// relative trees.
func ParseTemplate(fsys fs.FS, name string, noParseShow, dollarIdentifier bool) (*ast.Tree, error) {

	if name == "." || strings.HasSuffix(name, "/") {
		return nil, os.ErrInvalid
	}

	src, format, err := readFileAndFormat(fsys, name)
	if err != nil {
		return nil, err
	}

	pp := &templateExpansion{
		fsys:             fsys,
		trees:            map[string]parsedTree{},
		paths:            []string{},
		noParseShow:      noParseShow,
		dollarIdentifier: dollarIdentifier,
	}

	tree, err := pp.parseSource(src, name, format, false)
	if err != nil {
		if err2, ok := err.(*SyntaxError); ok && err2.path == "" {
			err2.path = name
		} else if e, ok := err.(*CycleError); ok {
			e.msg = "file " + name + e.msg + ": cycle not allowed"
		}
		return nil, err
	}

	return tree, nil
}

// templateExpansion represents the state of a template expansion.
type templateExpansion struct {
	fsys             fs.FS
	trees            map[string]parsedTree
	paths            []string
	noParseShow      bool
	dollarIdentifier bool
}

// parsedTree represents a parsed tree. parent is the file path and node that
// extends, imports or renders the tree.
type parsedTree struct {
	tree   *ast.Tree
	parent struct {
		path string
		node ast.Node
	}
}

// rooted returns the path of an Extend, Import or Render node, rooted at
// the parent path of the template.
//
// Supposing that a/b/c is the parent path
//
//   if name is /d/e, the rooted path name is d/e
//   if name is d/e, the rooted path name is a/b/d/e
//   if name is ../d/e, the rooted path name is a/d/e
//   if name is ../../d/e, the rooted path name is d/e
//
func rooted(parent, name string) (string, error) {
	if path.IsAbs(name) {
		return name[1:], nil
	}
	r := path.Join(path.Dir(parent), name)
	if strings.HasPrefix(r, "..") {
		return "", os.ErrInvalid
	}
	return r, nil
}

// parseNodeFile parses the file referenced by an Extends, Import or Render
// node and returns its tree.
func (pp *templateExpansion) parseNodeFile(node ast.Node) (*ast.Tree, error) {

	var err error
	var name string
	var imported bool

	// Get the file's rooted path and if it should be a declarations file.
	switch n := node.(type) {
	case *ast.Extends:
		name = n.Path
	case *ast.Import:
		name = n.Path
		imported = true
	case *ast.Render:
		name = n.Path
	}
	parent := pp.paths[len(pp.paths)-1]
	name, err = rooted(parent, name)
	if err != nil {
		return nil, err
	}

	// Check if there is a cycle.
	for _, p := range pp.paths {
		if p == name {
			return nil, &CycleError{path: name}
		}
	}

	// Check if it has already been parsed.
	var tree *ast.Tree
	if parsed, ok := pp.trees[name]; ok {
		switch n := parsed.parent.node.(type) {
		case *ast.Extends:
			switch node.(type) {
			case *ast.Import:
				return nil, syntaxError(node.Pos(), "import of file extended at %s:%s", parsed.parent.path, n.Pos())
			case *ast.Render:
				return nil, syntaxError(node.Pos(), "render of file extended at %s:%s", parsed.parent.path, n.Pos())
			}
		case *ast.Import:
			if _, ok := node.(*ast.Render); ok {
				return nil, syntaxError(node.Pos(), "render of file imported at %s:%s", parsed.parent.path, n.Pos())
			}
		case *ast.Render:
			if _, ok := node.(*ast.Import); ok {
				return nil, syntaxError(node.Pos(), "import of file rendered at %s:%s", parsed.parent.path, n.Pos())
			}
		}
		tree = parsed.tree
	}

	if tree == nil {
		// Parse the file.
		src, format, err := readFileAndFormat(pp.fsys, name)
		if err != nil {
			return nil, err
		}
		tree, err = pp.parseSource(src, name, format, imported)
		if err != nil {
			return nil, err
		}
		parsed := parsedTree{tree: tree}
		parsed.parent.path = pp.paths[len(pp.paths)-1]
		parsed.parent.node = node
		pp.trees[name] = parsed
	}

	return tree, nil
}

// parseSource parses src expanding Extends, Import and Render nodes.
// path is the path of the file, format is its content format and imported
// indicates whether the file is imported. path must be absolute and cleared.
func (pp *templateExpansion) parseSource(src []byte, path string, format ast.Format, imported bool) (*ast.Tree, error) {

	tree, unexpanded, err := ParseTemplateSource(src, format, imported, pp.noParseShow, pp.dollarIdentifier)
	if err != nil {
		if se, ok := err.(*SyntaxError); ok {
			se.path = path
		}
		return nil, err
	}
	tree.Path = path

	// Expand the nodes.
	pp.paths = append(pp.paths, path)
	err = pp.expand(unexpanded)
	pp.paths = pp.paths[:len(pp.paths)-1]
	if err != nil {
		if e, ok := err.(*SyntaxError); ok && e.path == "" {
			e.path = path
		}
		return nil, err
	}

	return tree, nil
}

// expand expands nodes parsing the sub-trees.
func (pp *templateExpansion) expand(nodes []ast.Node) error {

	for _, node := range nodes {

		switch n := node.(type) {

		case *ast.Extends:
			// extends "path"

			if len(pp.paths) > 1 {
				return syntaxError(n.Pos(), "extended, imported and rendered files can not have extends")
			}
			var err error
			n.Tree, err = pp.parseNodeFile(n)
			if err != nil {
				parent := pp.paths[len(pp.paths)-1]
				rootedPath, _ := rooted(parent, n.Path)
				if errors.Is(err, os.ErrNotExist) {
					err = syntaxError(n.Pos(), "extends path %q does not exist", rootedPath)
				} else if e, ok := err.(*CycleError); ok {
					e.msg = "\n\textends " + rootedPath + e.msg
					if e.path == pp.paths[len(pp.paths)-1] {
						e.pos = *(n.Pos())
					}
				}
				return err
			}
			if n.Format != n.Tree.Format {
				if !(n.Format == ast.FormatMarkdown && n.Tree.Format == ast.FormatHTML) {
					return syntaxError(node.Pos(), "extended file %q is %s instead of %s",
						n.Tree.Path, n.Tree.Format, n.Format)
				}
			}

		case *ast.Import:
			// import "path"

			// Try to import the path as a template file
			var err error
			n.Tree, err = pp.parseNodeFile(n)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				if e, ok := err.(*CycleError); ok {
					parent := pp.paths[len(pp.paths)-1]
					rootedPath, _ := rooted(parent, n.Path)
					e.msg = "\n\timports " + rootedPath + e.msg
					if e.path == pp.paths[len(pp.paths)-1] {
						e.pos = *(n.Pos())
					}
				}
				return err
			}

		default:
			// render "path

			var r *ast.Render
			var special bool

			switch n := node.(type) {
			case *ast.Render:
				r = n
			case *ast.Default:
				r = n.Expr1.(*ast.Render)
				special = true
			default:
				panic("unexpected node")
			}

			var err error
			r.Tree, err = pp.parseNodeFile(r)
			if err != nil && (!special || !errors.Is(err, os.ErrNotExist)) {
				parent := pp.paths[len(pp.paths)-1]
				rootedPath, _ := rooted(parent, r.Path)
				if errors.Is(err, os.ErrNotExist) {
					err = syntaxError(n.Pos(), "render path %q does not exist", rootedPath)
				} else if e, ok := err.(*CycleError); ok {
					e.msg = "\n\trenders " + rootedPath + e.msg
					if e.path == pp.paths[len(pp.paths)-1] {
						e.pos = *(n.Pos())
					}
				}
				return err
			}

		}

	}

	return nil
}

// readFileAndFormat reads the file with the given path name from fsys and
// returns its content and format. If fsys implements FormatFS, it calls
// its Format method, otherwise it determines the format from the file name
// extension.
func readFileAndFormat(fsys fs.FS, name string) ([]byte, ast.Format, error) {
	src, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, 0, err
	}
	format := ast.FormatText
	if ff, ok := fsys.(FormatFS); ok {
		format, err = ff.Format(name)
		if err != nil {
			return nil, 0, err
		}
		if format < ast.FormatText || format > ast.FormatMarkdown {
			return nil, 0, fmt.Errorf("unkonwn format %d", format)
		}
	} else {
		switch path.Ext(name) {
		case ".html":
			format = ast.FormatHTML
		case ".css":
			format = ast.FormatCSS
		case ".js":
			format = ast.FormatJS
		case ".json":
			format = ast.FormatJSON
		case ".md", ".mkd", ".mkdn", ".mdown", ".markdown":
			format = ast.FormatMarkdown
		}
	}
	return src, format, nil
}
