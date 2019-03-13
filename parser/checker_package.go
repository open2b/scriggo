package parser

import (
	"scrigo/ast"
)

type Package struct {
	Name                     string
	Declarations             map[string]*ast.TypeInfo
	VariablesEvaluationOrder []string
}

var currentlyEvaluating = &ast.TypeInfo{}
var globalNotChecked = &ast.TypeInfo{}

func CheckPackage(node *ast.Package) (pkg Package, err error) {
	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(*Error); ok {
				err = rerr
			} else {
				panic(r)
			}
		}
	}()
	pkg = Package{
		Name:         node.Name,
		Declarations: make(map[string]*ast.TypeInfo, len(node.Declarations)),
	}
	tc := typechecker{
		scopes:       []typeCheckerScope{universe},
		packageBlock: make(typeCheckerScope, len(node.Declarations)),
	}
	for _, n := range node.Declarations {
		switch n := n.(type) {
		case *ast.Import:
			if n.Tree == nil {
				// TODO (Gianluca): (Go Package)
			} else {
				// TODO (Gianluca): (Scrigo Package)
				_, err := CheckPackage(n.Tree.Nodes[0].(*ast.Package))
				if err != nil {
					return pkg, err
				}
			}
		case *ast.Const:
			for i := range n.Identifiers {
				tc.declarations.Constants = append(tc.declarations.Constants, &Declaration{Ident: n.Identifiers[i].Name, Expr: n.Values[i], Type: n.Type})
				tc.packageBlock[n.Identifiers[i].Name] = globalNotChecked
			}
		case *ast.Var:
			for i := range n.Identifiers {
				tc.declarations.Variables = append(tc.declarations.Variables, &Declaration{Ident: n.Identifiers[i].Name, Expr: n.Values[i], Type: n.Type}) // TODO (Gianluca): add support for var a, b, c = f()
				tc.packageBlock[n.Identifiers[i].Name] = globalNotChecked
			}
		case *ast.Func:
			tc.declarations.Functions = append(tc.declarations.Functions, &Declaration{Ident: n.Ident.Name, Body: n.Body, Type: n.Type})
			tc.packageBlock[n.Ident.Name] = globalNotChecked
		}
	}
	for i := 0; i < len(tc.declarations.Constants); i++ { // TODO (Gianluca): n. of iterations can be reduced.
		for _, c := range tc.declarations.Constants {
			tc.packageBlock[c.Ident] = currentlyEvaluating
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
	uncheckedPaths := true
	for uncheckedPaths {
		uncheckedPaths = false
		for _, v := range tc.declarations.Variables {
			tc.packageBlock[v.Ident] = currentlyEvaluating
			tc.globalDependencies = nil
			ti := tc.checkExpression(v.Expr)
			tc.packageBlock[v.Ident] = &ast.TypeInfo{Type: ti.Type, Properties: ast.PropertyAddressable}
			allDepsSatisfied := true
			for _, dep := range tc.globalDependencies {
				if !sliceContainsString(tc.initOrder, dep) {
					allDepsSatisfied = false
					uncheckedPaths = true
					break
				}
			}
			if allDepsSatisfied && !sliceContainsString(tc.initOrder, v.Ident) {
				tc.initOrder = append(tc.initOrder, v.Ident)
			}
		}
		for _, f := range tc.declarations.Functions {
			// TODO (Gianluca): add function to package block?
			tc.globalDependencies = nil
			tc.checkNodes(f.Body.Nodes)
			allDepsSatisfied := true
			for _, dep := range tc.globalDependencies {
				if !sliceContainsString(tc.initOrder, dep) {
					allDepsSatisfied = false
					uncheckedPaths = true
					break
				}
			}
			if allDepsSatisfied && !sliceContainsString(tc.initOrder, f.Ident) {
				tc.initOrder = append(tc.initOrder, f.Ident)
			}
		}
	}
	for ident, ti := range tc.packageBlock {
		pkg.Declarations[ident] = ti
	}
	pkg.VariablesEvaluationOrder = tc.initOrder
	return pkg, nil
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
