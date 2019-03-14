package parser

import (
	"reflect"
	"scrigo/ast"
)

type GoPackage struct {
	Name         string
	Declarations map[string]interface{}
}

type tcPackage struct {
	Name         string
	Declarations map[string]*ast.TypeInfo
}

var notChecked = &ast.TypeInfo{}

// TODO (Gianluca): CheckPackage should have 'src' (type []byte) instead of
// 'node' as argument, and its name should be something like: 'ParsePackage'.
func CheckPackage(node *ast.Tree, imports map[string]*GoPackage) (*ast.Tree, error) {
	tree, _, err := checkPackage(node, imports)
	return tree, err
}

func checkPackage(node *ast.Tree, imports map[string]*GoPackage) (tree *ast.Tree, pkg *tcPackage, err error) {

	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(*Error); ok {
				err = rerr
			} else {
				panic(r)
			}
		}
	}()

	packageNode := node.Nodes[0].(*ast.Package) // TODO (Gianluca): to review.

	tc := typechecker{
		scopes:       []typeCheckerScope{universe},
		packageBlock: make(typeCheckerScope, len(packageNode.Declarations)),
		varDeps:      make(map[string][]string, 3), // TODO (Gianluca): to review.
	}
	pkg = &tcPackage{
		Name:         packageNode.Name,
		Declarations: make(map[string]*ast.TypeInfo, len(packageNode.Declarations)),
	}

	for _, n := range packageNode.Declarations {
		switch n := n.(type) {
		case *ast.Import:
			if n.Tree == nil {
				// Go package.
				importedPkg := tcPackage{}
				goPkg, ok := imports[n.Path]
				if !ok {
					panic(tc.errorf(n, "cannot find package %q", n.Path))
				}
				importedPkg.Declarations = make(map[string]*ast.TypeInfo, len(goPkg.Declarations))
				for ident, value := range goPkg.Declarations {
					ti := &ast.TypeInfo{}
					switch value {
					case reflect.TypeOf(value).Kind() == reflect.Ptr:
						ti = &ast.TypeInfo{Type: reflect.TypeOf(value).Elem(), Properties: ast.PropertyAddressable}
					case reflect.TypeOf(value) == reflect.TypeOf(reflect.Type(nil)):
						ti = &ast.TypeInfo{Type: value.(reflect.Type), Properties: ast.PropertyIsType}
					case reflect.TypeOf(value).Kind() == reflect.Func:
						// Being not addressable, a global function can be
						// differentiated from a global function literal.
						ti = &ast.TypeInfo{Type: reflect.TypeOf(value)}
					default:
						// TODO (Gianluca): handle 'constants' properly.
						ti = &ast.TypeInfo{Value: value}
					}
					importedPkg.Declarations[ident] = ti
				}
				importedPkg.Name = goPkg.Name
				tc.fileBlock[importedPkg.Name] = &ast.TypeInfo{Value: importedPkg, Properties: ast.PropertyIsPackage}
			} else {
				// Scrigo package.
				_, importedPkg, err := checkPackage(n.Tree, nil)
				if err != nil {
					return nil, &tcPackage{}, err
				}
				tc.fileBlock[importedPkg.Name] = &ast.TypeInfo{Value: importedPkg, Properties: ast.PropertyIsPackage}
			}
		case *ast.Const:
			for i := range n.Identifiers {
				tc.declarations.Constants = append(tc.declarations.Constants, &Declaration{Ident: n.Identifiers[i].Name, Expr: n.Values[i], Type: n.Type})
				tc.packageBlock[n.Identifiers[i].Name] = notChecked
			}
		case *ast.Var:
			for i := range n.Identifiers {
				tc.declarations.Variables = append(tc.declarations.Variables, &Declaration{Variable: n, Ident: n.Identifiers[i].Name, Expr: n.Values[i], Type: n.Type}) // TODO (Gianluca): add support for var a, b, c = f()
				tc.packageBlock[n.Identifiers[i].Name] = notChecked
			}
		case *ast.Func:
			tc.declarations.Functions = append(tc.declarations.Functions, &Declaration{Ident: n.Ident.Name, Body: n.Body, Type: n.Type})
			tc.packageBlock[n.Ident.Name] = notChecked
		}
	}

	// Constants.
	for i := 0; i < len(tc.declarations.Constants); i++ { // TODO (Gianluca): n. of iterations can be reduced.
		for _, c := range tc.declarations.Constants {
			ti := tc.checkExpression(c.Expr)
			if !ti.IsConstant() {
				panic(tc.errorf(c.Expr, "const initializer %v is not a constant", c.Expr))
			}
			if c.Type != nil {
				typ := tc.checkType(c.Type, noEllipses)
				if !tc.isAssignableTo(ti, typ.Type) {
					panic(tc.errorf(c.Expr, "cannot convert %v (type %s) to type %v", c.Expr, ti.String(), typ.Type))
				}
			}
			tc.packageBlock[c.Ident] = ti
		}
	}

	// Functions.
	for _, v := range tc.declarations.Functions {
		tc.currentIdent = v.Ident
		tc.currentlyEvaluating = []string{v.Ident}
		tc.temporaryEvaluated = make(map[string]*ast.TypeInfo)
		tc.checkNodes(v.Body.Nodes)
		tc.packageBlock[v.Ident] = &ast.TypeInfo{Type: tc.typeof(v.Type, noEllipses).Type}
		tc.initOrder = append(tc.initOrder, v.Ident)
	}

	// Variables.
	for unresolvedDeps := true; unresolvedDeps; {
		unresolvedDeps = false
		for _, v := range tc.declarations.Variables {
			tc.currentIdent = v.Ident
			tc.currentlyEvaluating = []string{v.Ident}
			tc.temporaryEvaluated = make(map[string]*ast.TypeInfo)
			ti := tc.checkExpression(v.Expr)
			tc.packageBlock[v.Ident] = &ast.TypeInfo{Type: ti.Type, Properties: ast.PropertyAddressable}
			if !tc.tryAddingToInitOrder(v.Ident) {
				unresolvedDeps = true
			}
		}
	}

	pkg.Declarations = make(map[string]*ast.TypeInfo)
	for ident, ti := range tc.packageBlock {
		pkg.Declarations[ident] = ti
	}

	tree = node
	tree.VariableOrdering = nil
	for _, v := range tc.initOrder {
		if tc.getDecl(v).Body != nil { // v is a function.
			continue
		}
		tree.VariableOrdering = append(tree.VariableOrdering, v)
	}

	return tree, pkg, nil
}

// stringInSlice indicates if ss contains s.
func sliceContainsString(ss []string, s string) bool {
	for _, ts := range ss {
		if s == ts {
			return true
		}
	}
	return false
}

// containsDuplicates returns true if ss contains at least one duplicate string.
func containsDuplicates(ss []string) bool {
	for _, a := range ss {
		count := 0
		for _, b := range ss {
			if a == b {
				count++
			}
		}
		if count != 1 {
			return true
		}
	}
	return false
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
