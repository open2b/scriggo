// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/fs"
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
// ParseTemplate expands the nodes Extends, Import and Partial parsing the
// relative trees.
func ParseTemplate(fsys fs.FS, name string, packages PackageLoader) (*ast.Tree, error) {

	if name == "." || strings.HasSuffix(name, "/") {
		return nil, os.ErrInvalid
	}

	src, format, err := readFileAndFormat(fsys, name)
	if err != nil {
		return nil, err
	}

	pp := &templateExpansion{
		fsys:     fsys,
		packages: packages,
		trees:    map[string]parsedTree{},
		paths:    []string{},
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
	fsys     fs.FS
	trees    map[string]parsedTree
	packages PackageLoader
	paths    []string
}

// parsedTree represents a parsed tree. parent is the file path and node that
// extends, imports or renders as partial the tree.
type parsedTree struct {
	tree   *ast.Tree
	parent struct {
		path string
		node ast.Node
	}
}

// rooted returns the path of an Extend, Import or Partial node, rooted at
// the root path of the template.
//
// Supposing that a/b/c is the path name of the file that contains the node
//
//   if name is /d/e, the rooted path name is d/e
//   if name is d/e, the rooted path name is a/b/d/e
//   if name is ../d/e, the rooted path name is a/d/e
//   if name is ../../d/e, the rooted path name is d/e
//
func (pp *templateExpansion) rooted(name string) (string, error) {
	if name[0] == '/' {
		return name[1:], nil
	}
	parent := pp.paths[len(pp.paths)-1]
	if p := strings.LastIndex(parent, "/"); p > 0 {
		parent = parent[:p]
	} else {
		parent = ""
	}
	if strings.HasPrefix(name, "..") {
		p := strings.LastIndex(parent, "/")
		if p == -1 {
			return "", os.ErrInvalid
		}
		parent = parent[p-1:]
		name = name[3:]
	}
	if parent == "" {
		return name, nil
	}
	return parent + "/" + name, nil
}

// parseNodeFile parses the file referenced by an Extends, Import or
// Partial node and returns its tree.
func (pp *templateExpansion) parseNodeFile(node ast.Node) (*ast.Tree, error) {

	var err error
	var name string
	var format ast.Format
	var imported bool

	// Get the file's rooted path, its format and if it should be
	// a declarations file.
	switch n := node.(type) {
	case *ast.Extends:
		name = n.Path
		format = ast.Format(n.Context)
	case *ast.Import:
		name = n.Path
		imported = true
	case *ast.Partial:
		name = n.Path
		format = ast.Format(n.Context)
	}
	name, err = pp.rooted(name)
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
			case *ast.Partial:
				return nil, syntaxError(node.Pos(), "partial of file extended at %s:%s", parsed.parent.path, n.Pos())
			}
			if format != parsed.tree.Format {
				if !(format == ast.FormatMarkdown && parsed.tree.Format == ast.FormatHTML) {
					return nil, syntaxError(node.Pos(), "extended file %q is %s instead of %s",
						name, parsed.tree.Format, format)
				}
			}
		case *ast.Import:
			if _, ok := node.(*ast.Partial); ok {
				return nil, syntaxError(node.Pos(), "partial of file imported at %s:%s", parsed.parent.path, n.Pos())
			}
		case *ast.Partial:
			if _, ok := node.(*ast.Import); ok {
				return nil, syntaxError(node.Pos(), "import of file rendered as partial at %s:%s", parsed.parent.path, n.Pos())
			}
			if format != parsed.tree.Format {
				return nil, syntaxError(node.Pos(), "partial file %q is %s instead of %s", name, parsed.tree.Format, format)
			}
		}
		tree = parsed.tree
	}

	if tree == nil {
		// Parse the file.
		src, fo, err := readFileAndFormat(pp.fsys, name)
		if err != nil {
			return nil, err
		}
		if fo != format {
			switch node.(type) {
			case *ast.Extends:
				if !(format == ast.FormatMarkdown && fo == ast.FormatHTML) {
					return nil, syntaxError(node.Pos(), "extended file %q is %s instead of %s",
						name, fo, format)
				}
			case *ast.Partial:
				return nil, syntaxError(node.Pos(), "partial file %q is %s instead of %s", name, fo, format)
			}
		}
		tree, err = pp.parseSource(src, name, fo, imported)
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

// parseSource parses src expanding Extends, Import and Partial nodes.
// path is the path of the file, format is its content format and imported
// indicates whether the file is imported. path must be absolute and cleared.
func (pp *templateExpansion) parseSource(src []byte, path string, format ast.Format, imported bool) (*ast.Tree, error) {

	tree, unexpanded, err := ParseTemplateSource(src, format, imported)
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

			if len(pp.paths) > 1 {
				return syntaxError(n.Pos(), "extended, imported and partial files can not have extends")
			}
			var err error
			n.Tree, err = pp.parseNodeFile(n)
			if err != nil {
				absPath, _ := pp.rooted(n.Path)
				if errors.Is(err, os.ErrNotExist) {
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
					absPath, _ := pp.rooted(n.Path)
					if errors.Is(err, os.ErrNotExist) {
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

		case *ast.Partial:

			var err error
			n.Tree, err = pp.parseNodeFile(n)
			if err != nil {
				absPath, _ := pp.rooted(n.Path)
				if errors.Is(err, os.ErrNotExist) {
					err = syntaxError(n.Pos(), "partial path %q does not exist", absPath)
				} else if e, ok := err.(*CycleError); ok {
					e.msg = "\n\tpartial " + absPath + e.msg
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
