package parser

import (
	"scrigo/ast"
)

func checkScript(tree *ast.Tree, main *GoPackage) (_ *PackageInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(*Error); ok {
				err = rerr
			} else {
				panic(r)
			}
		}
	}()
	tc := newTypechecker(tree.Path, true)
	tc.universe = universe
	if main != nil {
		tc.scopes = append(tc.scopes, main.toTypeCheckerScope())
	}
	tc.checkNodesInNewScope(tree.Nodes)
	pkgInfo := &PackageInfo{}
	pkgInfo.IndirectVars = tc.upValues
	return pkgInfo, err
}
