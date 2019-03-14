package parser

import (
	"errors"
	"fmt"
	"reflect"
	"scrigo/ast"
	"strings"
)

// TODO (Gianluca): find a better name.
type GoPackage struct {
	Name         string
	Declarations map[string]interface{}
}

// TODO (Gianluca): find a better name.
type tcPackage struct {
	Name         string
	Declarations map[string]*ast.TypeInfo
}

// notChecked represents the type info of a not type-checked package
// declaration.
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

	if len(node.Nodes) == 0 {
		return nil, nil, errors.New("expected 'package', found EOF")
	}
	packageNode, ok := node.Nodes[0].(*ast.Package)
	if !ok {
		t := fmt.Sprintf("%T", node.Nodes[0])
		t = strings.ToLower(t[len("*ast."):])
		return nil, nil, fmt.Errorf("expected 'package', found '%s'", t)
	}

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
			importedPkg := &tcPackage{}
			if n.Tree == nil {
				// Go package.
				goPkg, ok := imports[n.Path]
				if !ok {
					return nil, nil, tc.errorf(n, "cannot find package %q", n.Path)
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
			} else {
				// Scrigo package.
				var err error
				_, importedPkg, err = checkPackage(n.Tree, nil)
				if err != nil {
					return nil, nil, err
				}
			}
			if n.Ident == nil {
				tc.fileBlock[importedPkg.Name] = &ast.TypeInfo{Value: importedPkg, Properties: ast.PropertyIsPackage}
			} else {
				switch n.Ident.Name {
				case "_":
				case ".":
					for ident, ti := range importedPkg.Declarations {
						tc.fileBlock[ident] = ti
					}
				default:
					tc.fileBlock[n.Ident.Name] = &ast.TypeInfo{Value: importedPkg, Properties: ast.PropertyIsPackage}
				}
			}
		case *ast.Const:
			for i := range n.Identifiers {
				tc.declarations = append(tc.declarations, &Declaration{Ident: n.Identifiers[i].Name, Value: n.Values[i], Type: n.Type, DeclarationType: DeclarationConstant})
				tc.packageBlock[n.Identifiers[i].Name] = notChecked
			}
		case *ast.Var:
			for i := range n.Identifiers {
				tc.declarations = append(tc.declarations, &Declaration{Ident: n.Identifiers[i].Name, Value: n.Values[i], Type: n.Type, DeclarationType: DeclarationVariable}) // TODO (Gianluca): add support for var a, b, c = f()
				tc.packageBlock[n.Identifiers[i].Name] = notChecked
			}
		case *ast.Func:
			tc.declarations = append(tc.declarations, &Declaration{Ident: n.Ident.Name, Value: n.Body, Type: n.Type, DeclarationType: DeclarationFunction})
			tc.packageBlock[n.Ident.Name] = notChecked
		}
	}

	// Constants.
	for _, c := range tc.declarations {
		if c.DeclarationType == DeclarationConstant {
			tc.currentIdent = c.Ident
			tc.currentlyEvaluating = []string{c.Ident}
			tc.temporaryEvaluated = make(map[string]*ast.TypeInfo)
			ti := tc.checkExpression(c.Value.(ast.Expression))
			if !ti.IsConstant() {
				return nil, nil, tc.errorf(c.Value, "const initializer %v is not a constant", c.Value)
			}
			if c.Type != nil {
				typ := tc.checkType(c.Type, noEllipses)
				if !tc.isAssignableTo(ti, typ.Type) {
					return nil, nil, tc.errorf(c.Value, "cannot convert %v (type %s) to type %v", c.Value, ti.String(), typ.Type)
				}
			}
			tc.packageBlock[c.Ident] = ti
		}
	}

	// Functions.
	for _, v := range tc.declarations {
		if v.DeclarationType == DeclarationFunction {
			tc.currentIdent = v.Ident
			tc.currentlyEvaluating = []string{v.Ident}
			tc.temporaryEvaluated = make(map[string]*ast.TypeInfo)
			tc.checkNodes(v.Value.(*ast.Block).Nodes)
			tc.packageBlock[v.Ident] = &ast.TypeInfo{Type: tc.typeof(v.Type, noEllipses).Type}
			tc.initOrder = append(tc.initOrder, v.Ident)
		}
	}

	// Variables.
	for unresolvedDeps := true; unresolvedDeps; {
		unresolvedDeps = false
		for _, v := range tc.declarations {
			if v.DeclarationType == DeclarationVariable {
				tc.currentIdent = v.Ident
				tc.currentlyEvaluating = []string{v.Ident}
				tc.temporaryEvaluated = make(map[string]*ast.TypeInfo)
				ti := tc.checkExpression(v.Value.(ast.Expression))
				if v.Type != nil {
					typ := tc.checkType(v.Type, noEllipses)
					if !tc.isAssignableTo(ti, typ.Type) {
						return nil, nil, tc.errorf(v.Value, "cannot convert %v (type %s) to type %v", v.Value, ti.String(), typ.Type)
					}
				}
				tc.packageBlock[v.Ident] = &ast.TypeInfo{Type: ti.Type, Properties: ast.PropertyAddressable}
				if !tc.tryAddingToInitOrder(v.Ident) {
					unresolvedDeps = true
				}
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
		if tc.getDecl(v).DeclarationType == DeclarationVariable {
			tree.VariableOrdering = append(tree.VariableOrdering, v)
		}
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
