// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"io"
	"io/ioutil"

	"scrigo/internal/compiler/ast"
)

// ParseProgram parses a program reading its sources from loaders.
func ParseProgram(packages PackageLoader) (*ast.Tree, GlobalsDependencies, map[string]*PredefinedPackage, error) {

	trees := map[string]*ast.Tree{}
	predefined := map[string]*PredefinedPackage{}
	dependencies := GlobalsDependencies{}

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
			return nil, nil, nil, err
		}
		if pkg == nil {
			return nil, nil, nil, &SyntaxError{"", *(n.Pos()), fmt.Errorf("cannot find package %q", n.Path)}
		}

		switch pkg := pkg.(type) {
		case *PredefinedPackage:
			predefined[n.Path] = pkg
		case io.Reader:
			src, err := ioutil.ReadAll(pkg)
			if r, ok := pkg.(io.Closer); ok {
				_ = r.Close()
			}
			if err != nil {
				return nil, nil, nil, err
			}
			var deps GlobalsDependencies
			n.Tree, deps, err = ParseSource(src, true, false)
			if err != nil {
				return nil, nil, nil, err
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
							return nil, nil, nil, cycleError(err)
						}
					}
					imp.Tree = tree
				} else if _, ok := predefined[imp.Path]; ok {
					// Skip.
				} else {
					// Check if the path is a package path (path is already a valid path).
					if err := validPackagePath(n.Path); err != nil {
						if err == ErrNotCanonicalImportPath {
							return nil, nil, nil, fmt.Errorf("non-canonical import path %q (should be %q)", n.Path, cleanPath(n.Path))
						}
						return nil, nil, nil, fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
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
			for k, v := range deps {
				dependencies[k] = v
			}
		default:
			panic("scrigo: unexpected type from package loader")
		}

		if last == len(imports)-1 {
			imports = imports[:last]
		}

	}

	return main.Tree, dependencies, predefined, nil
}
