package compiler

import "scriggo/ast"

func (tc *typechecker) checkImportLocal(d *ast.Import, imports PackageLoader, pkgInfos map[string]*PackageInfo) error {
	importedPkg := &PackageInfo{}
	if d.Tree == nil {

	} else {

	}
	if tc.opts.SyntaxType == TemplateSyntax {
		if d.Ident != nil && d.Ident.Name == "_" {
			return nil
		}
		err := tc.templatePageToPackage(d.Tree, d.Tree.Path)
		if err != nil {
			return err
		}
		pkgInfos := map[string]*PackageInfo{}
		if d.Tree.Nodes[0].(*ast.Package).Name == "main" {
			return tc.programImportError(d)
		}
		err = checkPackage(d.Tree.Nodes[0].(*ast.Package), d.Path, nil, pkgInfos, tc.opts, tc.globalScope)
		if err != nil {
			return err
		}
		// TypeInfos of imported packages in templates are
		// "manually" added to the map of typeinfos of typechecker.
		for k, v := range pkgInfos[d.Path].TypeInfos {
			tc.typeInfos[k] = v
		}
		importedPkg := pkgInfos[d.Path]
		if d.Ident == nil {
			tc.unusedImports[importedPkg.Name] = nil
			for ident, ti := range importedPkg.Declarations {
				tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
				tc.filePackageBlock[ident] = scopeElement{t: ti}
			}
			return nil
		}
		switch d.Ident.Name {
		case "_":
		case ".":
			tc.unusedImports[importedPkg.Name] = nil
			for ident, ti := range importedPkg.Declarations {
				tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
				tc.filePackageBlock[ident] = scopeElement{t: ti}
			}
		default:
			tc.filePackageBlock[d.Ident.Name] = scopeElement{
				t: &TypeInfo{
					value:      importedPkg,
					Properties: PropertyIsPackage | PropertyHasValue,
				},
			}
			tc.unusedImports[d.Ident.Name] = nil
		}
		return nil
	}
	if tc.opts.SyntaxType == ScriptSyntax {
		pkg, err := tc.predefinedPkgs.Load(d.Path)
		if err != nil {
			return tc.errorf(d, "%s", err)
		}
		predefinedPkg := pkg.(predefinedPackage)
		if predefinedPkg.Name() == "main" {
			return tc.programImportError(d)
		}
		decls := predefinedPkg.DeclarationNames()
		importedPkg = &PackageInfo{}
		importedPkg.Declarations = make(map[string]*TypeInfo, len(decls))
		for n, d := range toTypeCheckerScope(predefinedPkg, 0, tc.opts) {
			importedPkg.Declarations[n] = d.t
		}
		importedPkg.Name = predefinedPkg.Name()
		if d.Ident == nil {
			tc.filePackageBlock[importedPkg.Name] = scopeElement{t: &TypeInfo{value: importedPkg, Properties: PropertyIsPackage | PropertyHasValue}}
			tc.unusedImports[importedPkg.Name] = nil
		} else {
			switch d.Ident.Name {
			case "_":
			case ".":
				tc.unusedImports[importedPkg.Name] = nil
				for ident, ti := range importedPkg.Declarations {
					tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
					tc.filePackageBlock[ident] = scopeElement{t: ti}
				}
			default:
				tc.filePackageBlock[d.Ident.Name] = scopeElement{t: &TypeInfo{value: importedPkg, Properties: PropertyIsPackage | PropertyHasValue}}
				tc.unusedImports[d.Ident.Name] = nil
			}
		}
		return nil
	}

	if tc.opts.SyntaxType == ProgramSyntax {
		if d.Ident == nil {
			tc.filePackageBlock[importedPkg.Name] = scopeElement{t: &TypeInfo{value: importedPkg, Properties: PropertyIsPackage | PropertyHasValue}}
			tc.unusedImports[importedPkg.Name] = nil
		} else {
			switch d.Ident.Name {
			case "_":
			case ".":
				tc.unusedImports[importedPkg.Name] = nil
				for ident, ti := range importedPkg.Declarations {
					tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
					tc.filePackageBlock[ident] = scopeElement{t: ti}
				}
			default:
				tc.filePackageBlock[d.Ident.Name] = scopeElement{
					t: &TypeInfo{
						value:      importedPkg,
						Properties: PropertyIsPackage | PropertyHasValue,
					},
				}
				tc.unusedImports[d.Ident.Name] = nil
			}
		}
	}

	return nil
}
