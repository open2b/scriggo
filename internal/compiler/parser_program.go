// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"io"
	"io/ioutil"

	"scriggo/ast"
)

// ParseProgram parses a program reading its sources from loaders.
func ParseProgram(packages PackageLoader) (*ast.Tree, error) {

	trees := map[string]*ast.Tree{}
	predefined := map[string]bool{}

	main := ast.NewImport(nil, nil, "main", ast.ContextGo)

	imports := []*ast.Import{main}

	for len(imports) > 0 {

		last := len(imports) - 1
		n := imports[last]

		// Check if it has already been loaded.
		if tree, ok := trees[n.Path]; ok {
			n.Tree = tree
			imports = imports[:last]
			continue
		}
		if _, ok := predefined[n.Path]; ok {
			imports = imports[:last]
			continue
		}

		// Load the package.
		pkg, err := packages.Load(n.Path)
		if err != nil {
			return nil, err
		}
		if pkg == nil {
			return nil, syntaxError(n.Pos(), "cannot find package %q", n.Path)
		}

		switch pkg := pkg.(type) {
		case predefinedPackage:
			predefined[n.Path] = true
		case io.Reader:
			src, err := ioutil.ReadAll(pkg)
			if r, ok := pkg.(io.Closer); ok {
				_ = r.Close()
			}
			if err != nil {
				return nil, err
			}
			n.Tree, err = ParseSource(src, false, false)
			if err != nil {
				return nil, err
			}
			n.Tree.Path = n.Path
			trees[n.Path] = n.Tree
			declarations := n.Tree.Nodes[0].(*ast.Package).Declarations
			// Parse the import declarations in the tree.
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
							err := "package "
							for i, imp = range imports {
								if i > 0 {
									err += "\n\timports "
								}
								err += imp.Path
							}
							err += "\n\timports " + p.Path
							return nil, cycleError(err)
						}
					}
					imp.Tree = tree
				} else if predefined[imp.Path] {
					// Skip.
				} else {
					// Check if the path is a package path (path is already a valid path).
					if err := validPackagePath(n.Path); err != nil {
						if err == ErrNotCanonicalImportPath {
							return nil, fmt.Errorf("non-canonical import path %q (should be %q)", n.Path, cleanPath(n.Path))
						}
						return nil, fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
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
			}
		default:
			panic("scriggo: unexpected type from package loader")
		}

		if last == len(imports)-1 {
			imports = imports[:last]
		}

	}

	return main.Tree, nil
}

// ParseScript parses a script reading its source from src and the imported
// packages form the loader. shebang reports whether the script can have the
// shebang as first line.
func ParseScript(src io.Reader, loader PackageLoader, shebang bool) (*ast.Tree, error) {

	packages := map[string]bool{}

	// Load package main.
	if loader != nil {
		main, err := loader.Load("main")
		if err != nil {
			return nil, err
		}
		switch main.(type) {
		case nil, predefinedPackage:
		default:
			return nil, fmt.Errorf("scriggo: unexpected type %T for package \"main\"", main)
		}
	}

	// Parse the source.
	buf, err := ioutil.ReadAll(src)
	if r, ok := src.(io.Closer); ok {
		_ = r.Close()
	}
	if err != nil {
		return nil, err
	}
	tree, err := ParseSource(buf, true, shebang)
	if err != nil {
		return nil, err
	}

	// Parse the import declarations in the tree.
	for _, node := range tree.Nodes {
		imp, ok := node.(*ast.Import)
		if !ok {
			break
		}
		// Check if the path is a package path (path is already a valid path).
		if err := validPackagePath(imp.Path); err != nil {
			if err == ErrNotCanonicalImportPath {
				return nil, fmt.Errorf("non-canonical import path %q (should be %q)", imp.Path, cleanPath(imp.Path))
			}
			return nil, fmt.Errorf("invalid path %q at %s", imp.Path, imp.Pos())
		}
		// Load the package.
		if loader == nil {
			return nil, syntaxError(imp.Pos(), "cannot find package %q", imp.Path)
		}
		pkg, err := loader.Load(imp.Path)
		if err != nil {
			return nil, err
		}
		switch pkg := pkg.(type) {
		case predefinedPackage:
			packages[imp.Path] = true
		case nil:
			return nil, syntaxError(imp.Pos(), "cannot find package %q", imp.Path)
		default:
			return nil, fmt.Errorf("scriggo: unexpected type %T package loader", pkg)
		}
	}

	return tree, nil
}
