package parser

import (
	"errors"
	"fmt"
	"reflect"
	"scrigo/ast"
	"strings"
)

// TODO (Gianluca): find a better name.
// TODO (Gianluca): this identifier must be accessed from outside as
// "scrigo.Package" or something similar.
type GoPackage struct {
	Name         string
	Declarations map[string]interface{}
}

type PackageInfo struct {
	Name                 string
	Declarations         map[string]*TypeInfo
	VariableOrdering     []string                 // ordering of initialization of global variables.
	ConstantsExpressions map[ast.Node]interface{} // expressions of constants.
	UpValues             map[*ast.Identifier]bool
}

func (pi *PackageInfo) String() string {
	s := "{\n"
	s += "\tName:                 " + pi.Name + "\n"
	s += "\tDeclarations:\n"
	for i, d := range pi.Declarations {
		s += fmt.Sprintf("                              %s: %s\n", i, d)
	}
	s += "\tVariableOrdering:     " + "{" + strings.Join(pi.VariableOrdering, ",") + "}\n"
	// TODO (Gianluca): add constant expressions.
	s += "}\n"
	return s
}

// notChecked represents the type info of a not type-checked package
// declaration.
var notChecked = &TypeInfo{}

func checkPackage(tree *ast.Tree, imports map[string]*GoPackage) (pkgInfo *PackageInfo, err error) {

	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(*Error); ok {
				err = rerr
			} else {
				panic(r)
			}
		}
	}()

	if len(tree.Nodes) == 0 {
		return nil, errors.New("expected 'package', found EOF")
	}
	packageNode, ok := tree.Nodes[0].(*ast.Package)
	if !ok {
		t := fmt.Sprintf("%T", tree.Nodes[0])
		t = strings.ToLower(t[len("*ast."):])
		return nil, fmt.Errorf("expected 'package', found '%s'", t)
	}

	tc := newTypechecker()
	tc.universe = universe

	pkgInfo = &PackageInfo{
		Name:         packageNode.Name,
		Declarations: make(map[string]*TypeInfo, len(packageNode.Declarations)),
	}

	for _, n := range packageNode.Declarations {
		switch n := n.(type) {
		case *ast.Import:
			importedPkg := &PackageInfo{}
			if n.Tree == nil {
				// Go package.
				goPkg, ok := imports[n.Path]
				if !ok {
					return nil, tc.errorf(n, "cannot find package %q", n.Path)
				}
				importedPkg.Declarations = make(map[string]*TypeInfo, len(goPkg.Declarations))
				for ident, value := range goPkg.Declarations {
					ti := &TypeInfo{}
					switch {
					case reflect.TypeOf(value).Kind() == reflect.Ptr:
						ti = &TypeInfo{Type: reflect.TypeOf(value).Elem(), Properties: PropertyAddressable}
					case reflect.TypeOf(value) == reflect.TypeOf(reflect.Type(nil)):
						ti = &TypeInfo{Type: value.(reflect.Type), Properties: PropertyIsType}
					case reflect.TypeOf(value).Kind() == reflect.Func:
						// Being not addressable, a global function can be
						// differentiated from a global function literal.
						ti = &TypeInfo{Type: reflect.TypeOf(value)}
					default:
						// TODO (Gianluca): handle 'constants' properly.
						ti = &TypeInfo{Value: value, Properties: PropertyIsConstant}
					}
					importedPkg.Declarations[ident] = ti
				}
				importedPkg.Name = goPkg.Name
			} else {
				// Scrigo package.
				var err error
				importedPkg, err = checkPackage(n.Tree, nil)
				if err != nil {
					return nil, err
				}
			}
			if n.Ident == nil {
				tc.filePackageBlock[importedPkg.Name] = scopeElement{t: &TypeInfo{Value: importedPkg, Properties: PropertyIsPackage}}
				tc.unusedImports[importedPkg.Name] = nil
			} else {
				switch n.Ident.Name {
				case "_":
				case ".":
					tc.unusedImports[importedPkg.Name] = nil
					for ident, ti := range importedPkg.Declarations {
						tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
						tc.filePackageBlock[ident] = scopeElement{t: ti}
					}
				default:
					tc.filePackageBlock[n.Ident.Name] = scopeElement{t: &TypeInfo{Value: importedPkg, Properties: PropertyIsPackage}}
					tc.unusedImports[n.Ident.Name] = nil
				}
			}
		case *ast.Const:
			for i := range n.Identifiers {
				name := n.Identifiers[i].Name
				if _, ok := tc.filePackageBlock[name]; ok {
					panic(tc.errorf(n.Identifiers[i], "%s redeclared in this block", name))
				}
				tc.filePackageBlock[name] = scopeElement{t: notChecked}
				tc.declarations = append(tc.declarations, &Declaration{Node: n, Ident: name, Value: n.Values[i], Type: n.Type, DeclarationType: DeclarationConstant})
			}
		case *ast.Var:
			for i := range n.Identifiers {
				name := n.Identifiers[i].Name
				if _, ok := tc.filePackageBlock[name]; ok {
					panic(tc.errorf(n.Identifiers[i], "%s redeclared in this block", name))
				}
				tc.declarations = append(tc.declarations, &Declaration{Node: n, Ident: name, Value: n.Values[i], Type: n.Type, DeclarationType: DeclarationVariable}) // TODO (Gianluca): add support for var a, b, c = f()
				tc.filePackageBlock[name] = scopeElement{t: notChecked}
			}
		case *ast.Func:
			if n.Ident.Name == "init" {
				if len(n.Type.Parameters) > 0 || len(n.Type.Result) > 0 {
					panic(tc.errorf(n.Ident, "func init must have no arguments and no return values"))
				}
			} else {
				if _, ok := tc.filePackageBlock[n.Ident.Name]; ok {
					panic(tc.errorf(n.Ident, "%s redeclared in this block", n.Ident.Name))
				}
			}
			tc.declarations = append(tc.declarations, &Declaration{Node: n, Ident: n.Ident.Name, Value: n.Body, Type: n.Type, DeclarationType: DeclarationFunction})
			tc.filePackageBlock[n.Ident.Name] = scopeElement{t: notChecked}
		}
	}

	// Constants.
	for _, c := range tc.declarations {
		if c.DeclarationType == DeclarationConstant {
			tc.currentIdent = c.Ident
			tc.currentlyEvaluating = []string{c.Ident}
			tc.temporaryEvaluated = make(map[string]*TypeInfo)
			ti := tc.checkExpression(c.Value.(ast.Expression))
			if !ti.IsConstant() {
				return nil, tc.errorf(c.Value, "const initializer %v is not a constant", c.Value)
			}
			if c.Type != nil {
				typ := tc.checkType(c.Type, noEllipses)
				if !isAssignableTo(ti, typ.Type) {
					return nil, tc.errorf(c.Value, "cannot convert %v (type %s) to type %v", c.Value, ti.String(), typ.Type)
				}
			}
			tc.filePackageBlock[c.Ident] = scopeElement{t: ti}
		}
	}

	// Functions.
	for _, v := range tc.declarations {
		if v.DeclarationType == DeclarationFunction {
			tc.addScope()
			tc.ancestors = append(tc.ancestors, &ancestor{len(tc.scopes), v.Node})
			// Adds parameters to the function body scope.
			params := fillParametersTypes(v.Type.(*ast.FuncType).Parameters)
			isVariadic := v.Type.(*ast.FuncType).IsVariadic
			for i, param := range params {
				if param.Ident != nil {
					t := tc.checkType(param.Type, noEllipses)
					if isVariadic && i == len(params)-1 {
						tc.assignScope(param.Ident.Name, &TypeInfo{Type: reflect.SliceOf(t.Type), Properties: PropertyAddressable}, nil)
					} else {
						tc.assignScope(param.Ident.Name, &TypeInfo{Type: t.Type, Properties: PropertyAddressable}, nil)
					}
				}
			}
			// Adds named return values to the function body scope.
			for _, ret := range fillParametersTypes(v.Type.(*ast.FuncType).Result) {
				t := tc.checkType(ret.Type, noEllipses)
				if ret.Ident != nil {
					tc.assignScope(ret.Ident.Name, &TypeInfo{Type: t.Type, Properties: PropertyAddressable}, nil)
				}
			}
			tc.currentIdent = v.Ident
			tc.currentlyEvaluating = []string{v.Ident}
			tc.temporaryEvaluated = make(map[string]*TypeInfo)
			tc.filePackageBlock[v.Ident] = scopeElement{t: &TypeInfo{Type: tc.typeof(v.Type, noEllipses).Type}}
			tc.checkNodes(v.Value.(*ast.Block).Nodes)
			tc.initOrder = append(tc.initOrder, v.Ident)
			tc.ancestors = tc.ancestors[:len(tc.ancestors)-1]
			tc.removeCurrentScope()
		}
	}

	// Variables.
	for unresolvedDeps := true; unresolvedDeps; {
		unresolvedDeps = false
		for _, v := range tc.declarations {
			if v.DeclarationType == DeclarationVariable {
				tc.currentIdent = v.Ident
				tc.currentlyEvaluating = []string{v.Ident}
				tc.temporaryEvaluated = make(map[string]*TypeInfo)
				ti := tc.checkExpression(v.Value.(ast.Expression))
				if v.Type != nil {
					typ := tc.checkType(v.Type, noEllipses)
					if !isAssignableTo(ti, typ.Type) {
						return nil, tc.errorf(v.Value, "cannot convert %v (type %s) to type %v", v.Value, ti.String(), typ.Type)
					}
				}
				tc.filePackageBlock[v.Ident] = scopeElement{t: &TypeInfo{Type: ti.Type, Properties: PropertyAddressable}}
				if !tc.tryAddingToInitOrder(v.Ident) {
					unresolvedDeps = true
				}
			}
		}
	}
	tc.temporaryEvaluated = nil

	for pkg := range tc.unusedImports {
		// TODO (Gianluca): position is not correct.
		return nil, tc.errorf(new(ast.Position), "imported and not used: \"%s\"", pkg)
	}

	pkgInfo.Declarations = make(map[string]*TypeInfo)
	for ident, ti := range tc.filePackageBlock {
		pkgInfo.Declarations[ident] = ti.t
	}

	pkgInfo.UpValues = tc.upValues

	pkgInfo.VariableOrdering = nil
	for _, v := range tc.initOrder {
		if tc.getDecl(v).DeclarationType == DeclarationVariable {
			pkgInfo.VariableOrdering = append(pkgInfo.VariableOrdering, v)
		}
	}

	return pkgInfo, nil
}

// tryAddingToInitOrder tries to add name to the initialization order. Returns
// true if operation ended successfully.
func (tc *typechecker) tryAddingToInitOrder(name string) bool {
	for _, dep := range tc.varDeps[name] {
		if !sliceContainsString(tc.initOrder, dep) {
			return false
		}
	}
	if !sliceContainsString(tc.initOrder, name) {
		tc.initOrder = append(tc.initOrder, name)
	}
	return true
}
