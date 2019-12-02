package compiler

import "scriggo/ast"

func (tc *typechecker) checkImportLocal(node *ast.Import) error {
	switch tc.opts.SyntaxType {
	case ScriptSyntax:
		pkg, err := tc.predefinedPkgs.Load(node.Path)
		if err != nil {
			return tc.errorf(node, "%s", err)
		}
		predefinedPkg := pkg.(predefinedPackage)
		if predefinedPkg.Name() == "main" {
			return tc.programImportError(node)
		}
		decls := predefinedPkg.DeclarationNames()
		importedPkg := &PackageInfo{}
		importedPkg.Declarations = make(map[string]*TypeInfo, len(decls))
		for n, d := range toTypeCheckerScope(predefinedPkg, 0, tc.opts) {
			importedPkg.Declarations[n] = d.t
		}
		importedPkg.Name = predefinedPkg.Name()
		if node.Ident == nil {
			tc.filePackageBlock[importedPkg.Name] = scopeElement{t: &TypeInfo{value: importedPkg, Properties: PropertyIsPackage | PropertyHasValue}}
			tc.unusedImports[importedPkg.Name] = nil
		} else {
			switch node.Ident.Name {
			case "_":
			case ".":
				tc.unusedImports[importedPkg.Name] = nil
				for ident, ti := range importedPkg.Declarations {
					tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
					tc.filePackageBlock[ident] = scopeElement{t: ti}
				}
			default:
				tc.filePackageBlock[node.Ident.Name] = scopeElement{t: &TypeInfo{value: importedPkg, Properties: PropertyIsPackage | PropertyHasValue}}
				tc.unusedImports[node.Ident.Name] = nil
			}
		}
	case TemplateSyntax:
		if node.Ident != nil && node.Ident.Name == "_" {
			return nil
		}
		err := tc.templatePageToPackage(node.Tree, node.Tree.Path)
		if err != nil {
			return err
		}
		pkgInfos := map[string]*PackageInfo{}
		if node.Tree.Nodes[0].(*ast.Package).Name == "main" {
			return tc.programImportError(node)
		}
		err = checkPackage(node.Tree.Nodes[0].(*ast.Package), node.Path, nil, pkgInfos, tc.opts, tc.globalScope)
		if err != nil {
			return err
		}
		// TypeInfos of imported packages in templates are
		// "manually" added to the map of typeinfos of typechecker.
		for k, v := range pkgInfos[node.Path].TypeInfos {
			tc.typeInfos[k] = v
		}
		importedPkg := pkgInfos[node.Path]
		if node.Ident == nil {
			tc.unusedImports[importedPkg.Name] = nil
			for ident, ti := range importedPkg.Declarations {
				tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
				tc.filePackageBlock[ident] = scopeElement{t: ti}
			}
			return nil
		}
		switch node.Ident.Name {
		case "_":
		case ".":
			tc.unusedImports[importedPkg.Name] = nil
			for ident, ti := range importedPkg.Declarations {
				tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
				tc.filePackageBlock[ident] = scopeElement{t: ti}
			}
		default:
			tc.filePackageBlock[node.Ident.Name] = scopeElement{
				t: &TypeInfo{
					value:      importedPkg,
					Properties: PropertyIsPackage | PropertyHasValue,
				},
			}
			tc.unusedImports[node.Ident.Name] = nil
		}
	}
	return nil
}
