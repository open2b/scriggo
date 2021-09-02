// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/native"
)

var (
	ErrNoGoFiles      = errors.New("no Go files")
	ErrTooManyGoFiles = errors.New("too many Go files")
)

// ParseProgram parses a program.
func ParseProgram(fsys fs.FS) (*ast.Tree, error) {

	modPath, err := readModulePath(fsys)
	if err != nil {
		return nil, err
	}
	modPrefix := modPath + "/"

	trees := map[string]*ast.Tree{}
	main := ast.NewImport(nil, nil, "main", nil)
	imports := []*ast.Import{main}

	for len(imports) > 0 {

		last := len(imports) - 1
		n := imports[last]

		// Check if it has already been parsed.
		if tree, ok := trees[n.Path]; ok {
			n.Tree = tree
			imports = imports[:last]
			continue
		}

		// Parse the package.
		dir := "."
		if n.Path != "main" {
			dir = strings.TrimPrefix(n.Path, modPrefix)
		}
		n.Tree, err = parsePackage(fsys, dir)
		if err != nil {
			return nil, err
		}
		if n.Tree == nil {
			if n.Path == "main" {
				return nil, errors.New("cannot find main package")
			}
			return nil, syntaxError(n.Position, "cannot find package %q", n.Path)
		}
		n.Tree.Path = n.Path
		trees[n.Path] = n.Tree

		if modPath == "" {
			return main.Tree, nil
		}

		// Parse the import declarations within the module.
		declarations := n.Tree.Nodes[0].(*ast.Package).Declarations
		for _, decl := range declarations {
			imp, ok := decl.(*ast.Import)
			if !ok {
				break
			}
			if tree, ok := trees[imp.Path]; ok {
				// Check if there is a cycle.
				for i, p := range imports {
					if p.Path == imp.Path {
						// There is a cycle.
						err := &CycleError{
							path: p.Path,
							pos:  *(imp.Pos()),
						}
						err.msg = "package "
						for i, imp = range imports {
							if i > 0 {
								err.msg += "\n\timports "
							}
							err.msg += imp.Path
						}
						err.msg += "\n\timports " + p.Path + ": import cycle not allowed"
						return nil, err
					}
				}
				imp.Tree = tree
				continue
			}
			if !strings.HasPrefix(imp.Path, modPrefix) {
				continue
			}
			// Append the imports in reverse order.
			if last == len(imports)-1 {
				imports = append(imports, imp)
			} else {
				imports = append(imports, nil)
				copy(imports[last+2:], imports[last+1:])
				imports[last+1] = imp
			}
		}

		if last == len(imports)-1 {
			imports = imports[:last]
		}

	}

	return main.Tree, nil
}

// parsePackage parses a package at the given directory in fsys.
func parsePackage(fsys fs.FS, dir string) (*ast.Tree, error) {
	files, err := fs.ReadDir(fsys, dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = nil
		}
		return nil, err
	}
	var name string
	for _, file := range files {
		if file.Type().IsRegular() && strings.HasSuffix(file.Name(), ".go") {
			if name != "" {
				return nil, ErrTooManyGoFiles
			}
			if dir != "." {
				name = dir + "/"
			}
			name += file.Name()
		}
	}
	if name == "" {
		return nil, ErrNoGoFiles
	}
	fi, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	src, err := io.ReadAll(fi)
	_ = fi.Close()
	if err != nil {
		return nil, err
	}
	tree, err := parseSource(src, false)
	if err != nil {
		return nil, err
	}
	return tree, nil
}

// ParseScript parses a script reading its source from src and the imported
// packages form the importer.
func ParseScript(src io.Reader, importer native.Importer) (*ast.Tree, error) {

	// Parse the source.
	buf, err := io.ReadAll(src)
	if r, ok := src.(io.Closer); ok {
		_ = r.Close()
	}
	if err != nil {
		return nil, err
	}
	tree, err := parseSource(buf, true)
	if err != nil {
		return nil, err
	}

	// Parse the import declarations in the tree.
	for _, node := range tree.Nodes {
		imp, ok := node.(*ast.Import)
		if !ok {
			break
		}
		// Import the package.
		if importer == nil {
			return nil, syntaxError(imp.Pos(), "cannot find package %q", imp.Path)
		}
		pkg, err := importer.Import(imp.Path)
		if err != nil {
			return nil, err
		}
		switch pkg := pkg.(type) {
		case native.ImportablePackage:
		case nil:
			return nil, syntaxError(imp.Pos(), "cannot find package %q", imp.Path)
		default:
			return nil, fmt.Errorf("scriggo: unexpected type %T returned by the package importer", pkg)
		}
	}

	return tree, nil
}

func readModulePath(fsys fs.FS) (string, error) {
	fi, err := fsys.Open("go.mod")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	src, err := io.ReadAll(fi)
	_ = fi.Close()
	if err != nil {
		return "", err
	}
	path := modulePath(src)
	if path == "" {
		return "", &GoModError{path: "go.mod", pos: ast.Position{1, 1, 0, 0}, msg: "no module declaration in go.mod"}
	}
	if !validModulePath(path) {
		return "", &GoModError{path: "go.mod", pos: ast.Position{1, 1, 0, 0}, msg: "invalid module path in go.mod"}
	}
	return path, nil
}
