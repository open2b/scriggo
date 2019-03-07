// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TODO (Gianluca): this files needs a revision.

package parser

import (
	"fmt"
	"reflect"
	"scrigo/ast"
)

type identifierKind int

const (
	packageIdentifier identifierKind = iota
	constIdentifier
	varIdentifier
)

// Package represents a package.
// TODO (Gianluca): to review.
type Package struct {
	Name         string
	Declarations map[string]interface{}
}

// TODO (Gianluca): to review.
type declaration struct {
	name  string
	kind  identifierKind
	pkg   *Package
	typ   reflect.Type
	ident *ast.Identifier
	value interface{}
	node  ast.Node
	expr  ast.Expression
	deps  []*ast.Identifier
}

// TODO (Gianluca): to review.
type packageInfo struct {
	checktypes map[string]*ast.TypeInfo
}

func importPackage(path string, node *ast.Package, imports map[string]*Package) (*packageInfo, error) {

	tc := typechecker{
		path:         path,
		imports:      imports,
		fileBlock:    typeCheckerScope{},
		packageBlock: typeCheckerScope{},
		scopes:       []typeCheckerScope{},
		// TODO (Gianluca): to review.
		// 	pkg:          &Package{},
	}

	var constants []*declaration
	var variables []*declaration
	var functions []*declaration

	packageBlockIdentifiers := make([]*declaration, 0, len(node.Declarations))

	for _, n := range node.Declarations {

		switch n := n.(type) {

		case *ast.Import:
			var pkg *packageInfo
			var name string
			if n.Tree == nil {
				// TODO (Gianluca): to review.
				_, name = pkg, name
				// var ok bool
				// pkg, ok = imp.imports[n.Path]
				// if !ok {
				// 	panic(fmt.Sprintf("scrigo: package with path %q does not exist", n.Path))
				// }
			} else {
				node := n.Tree.Nodes[0].(*ast.Package)
				name = node.Name
				var err error
				pkg, err = importPackage(n.Path, node, imports)
				if err != nil {
					return nil, err
				}
			}
			ident := n.Ident
			if ident == nil {
				// TODO (Gianluca): to review.
				// imp.fileBlock[pkg.Name] = &declaration{kind: packageIdentifier}
			} else if ident.Name == "." {
				// TODO (Gianluca): to review.
				// for name := range pkg.Declarations {
				// 	if prev, ok := imp.fileBlock[name]; ok {
				// 		return nil, imp.errorf(n.Ident,
				// 			"%s redeclared as imported package name\n\tprevious declaration at %s",
				// 			name, prev.node.Pos())
				// 	}
				// 	imp.fileBlock[name] = &declaration{kind: packageIdentifier}
				// }
			} else if ident.Name != "_" {
				// TODO (Gianluca): to review.
				// if prev, ok := imp.fileBlock[ident.Name]; ok {
				// 	return nil, imp.errorf(n.Ident, "%s redeclared as imported package name\n\tprevious declaration at %s",
				// 		ident.Name, prev.node.Pos())
				// }
				// imp.fileBlock[ident.Name] = &declaration{kind: packageIdentifier, ident: ident, node: n}
			}

		case *ast.Const:
			for i, ident := range n.Identifiers {
				// TODO (Gianluca): to review.
				// if prev, ok := imp.packageBlock[ident.Name]; ok {
				// 	return nil, imp.errorf(ident, "%s redeclared in this block\n\tprevious declaration at %s",
				// 		ident, prev.ident.Pos())
				// }
				var expr = n.Values[i]
				var deps = identifiersInExpression(expr)
				// TODO (Gianluca): to review.
				g := &declaration{name: ident.Name, ident: ident, node: expr, deps: deps}
				// imp.packageBlock[ident.Name] = g
				constants = append(constants, g)
				packageBlockIdentifiers = append(packageBlockIdentifiers, g)
			}

		case *ast.Var:
			var expr ast.Expression
			var deps []*ast.Identifier
			single := len(n.Values) == 1
			if single {
				expr = n.Values[0]
				deps = identifiersInExpression(expr)
			}
			for i, ident := range n.Identifiers {
				// TODO (Gianluca): to review.
				// if prev, ok := imp.packageBlock[ident.Name]; ok {
				// 	return nil, imp.errorf(ident, "%s redeclared in this block\n\tprevious declaration at %s",
				// 		ident, prev.ident.Pos())
				// }
				// if prev, ok := imp.fileBlock[ident.Name]; ok {
				// 	// TODO(marco): check only packages and not global variables in other packages.
				// 	return nil, imp.errorf(ident, "%s redeclared in this block\n\tprevious declaration at %s",
				// 		ident, prev.ident.Pos())
				// }
				if !single {
					expr = n.Values[i]
					deps = identifiersInExpression(expr)
				}
				g := &declaration{ident: ident, node: expr, deps: deps}
				// TODO (Gianluca): to review.
				// imp.packageBlock[ident.Name] = g
				variables = append(variables, g)
				packageBlockIdentifiers = append(packageBlockIdentifiers, g)
			}

		case *ast.Func:
			// TODO (Gianluca): to review.
			// if prev, ok := imp.packageBlock[n.Ident.Name]; ok {
			// 	return nil, imp.errorf(n.Ident, "%s redeclared in this block\n\tprevious declaration at %s",
			// 		n.Ident, prev.ident.Pos())
			// }
			for _, param := range n.Type.Parameters {
				if param.Type != nil {
					// typ, err := imp.typeCheck(param.Type)
					// if err != nil {
					// 	return nil, err
					// }
					// TODO
				}
			}
			g := &declaration{ident: n.Ident, node: n}
			// TODO (Gianluca): to review.
			// imp.packageBlock[n.Ident.Name] = g
			functions = append(functions, g)

		}

	}

	// Initializes the constants.
	for _, c := range constants {
		err := tc.initConstant(c)
		if err != nil {
			return nil, err
		}
	}

	// Initializes the variables.
	for _, v := range variables {
		err := tc.initVariable(v)
		if err != nil {
			return nil, err
		}
	}

	// Checks for loops.
	for _, a := range packageBlockIdentifiers {
		fmt.Printf("declaration %s\n", a.ident.Name)
		for _, ref := range a.deps {
			b := tc.packageBlock[ref.Name]
			if b == nil {
				return nil, tc.errorf(ref, "undefined: %s", ref.Name)
			}
			// Checks "a := b".
			// TODO (Gianluca): to review.
			// loop := imp.checkPath(b, a)
			// if loop != nil {
			// 	var pathStr string
			// 	for _, g := range loop {
			// 		pathStr += fmt.Sprintf("\n\t%s:%s %s = %s", imp.path, g.node.Pos(), g.ident.Name, g.node)
			// 	}
			// 	g := loop[len(loop)-1]
			// 	return nil, imp.errorf(g.ident, "typechecking loop involving %s = %s%s", g.ident.Name, g.node, pathStr)
			// }
		}
	}

	return nil, nil
}

// checkPath returns the path from b to a. Returns nil if no path exists.
func (tc *typechecker) checkPath(b, a *declaration) []*declaration {

	if b.deps == nil {
		return nil
	}

	var path []*declaration
	var globals = []*declaration{b}

	for i := 0; i >= 0; i = len(globals) - 1 {
		g := globals[i]
		if j := len(path) - 1; j < 0 || g != path[j] {
			fmt.Printf("add %s to path\n", g.ident.Name)
			path = append(path, g)
		} else {
			fmt.Printf("remove %s from path and packageBlock\n", g.ident.Name)
			path = path[:j]
			globals = globals[:i]
		}
		// TODO (Gianluca): to review.
		// for _, r := range g.deps {
		// 	ref := tc.packageBlock[r.Name]
		// 	if ref == a {
		// 		fmt.Printf("add %s to path\n", a.ident.Name)
		// 		return append(path, a)
		// 	}
		// 	if ref.deps != nil {
		// 		fmt.Printf("add %s to packageBlock\n", ref.ident.Name)
		// 		globals = append(globals, ref)
		// 	}
		// }
	}

	return nil
}

func identifiersInExpression(expr ast.Expression) []*ast.Identifier {

	var identifiers []*ast.Identifier
	expressions := []ast.Expression{expr}

	for i := 0; i >= 0; i = len(expressions) - 1 {
		e := expressions[i]
		expressions = expressions[:i]
		switch e := e.(type) {
		case *ast.Parentesis:
			expressions = append(expressions, e.Expr)
		case *ast.UnaryOperator:
			expressions = append(expressions, e.Expr)
		case *ast.BinaryOperator:
			expressions = append(expressions, e.Expr2, e.Expr1)
		case *ast.Identifier:
			for _, ident := range identifiers {
				if ident.Name == e.Name {
					e = nil
					break
				}
			}
			if e != nil {
				identifiers = append(identifiers, e)
				fmt.Printf("%q ", e.Name)
			}
		case *ast.CompositeLiteral:
			for _, kv := range e.KeyValues {
				if kv.Key != nil {
					expressions = append(expressions, kv.Key)
				}
				expressions = append(expressions, kv.Value)
			}
		case *ast.Call:
			for j := len(e.Args) - 1; j >= 0; j-- {
				expressions = append(expressions, e.Args[j])
			}
			expressions = append(expressions, e.Func)
		case *ast.Index:
			expressions = append(expressions, e.Index, e.Expr)
		case *ast.Slicing:
			expressions = append(expressions, e.Expr, e.High, e.Low)
		case *ast.Selector:
			expressions = append(expressions, e.Expr)
		}
	}

	return identifiers
}

// importing represents the state of a type importing.
//
// TODO (Gianluca): remove?
//
// type importing struct {
// 	path         string
// 	imports      map[string]*Package
// 	fileBlock    map[string]*declaration
// 	packageBlock map[string]*declaration
// 	scopes       []checkerScope
// 	pkg          *Package
// }

func (tc *typechecker) initConstant(c *declaration) error {
	// TODO (Gianluca): to review.
	// fmt.Printf("constant %s\n", c.name)
	// declarations := tc.pkg.Declarations
	// if _, ok := declarations[c.name]; ok {
	// 	panic("constant definition loop!") // TODO (Gianluca): review!
	// }
	// declarations[c.name] = nil
	// value, err := tc.evalConstantExpr(c.expr)
	// if err != nil {
	// 	return err
	// }
	// declarations[c.name] = value
	return nil
}

func (tc *typechecker) initVariable(v *declaration) error {
	fmt.Printf("variable %s\n", v.name)
	// TODO (Gianluca): to review.
	// declarations := tc.pkg.Declarations
	// if _, ok := declarations[v.name]; ok {
	// 	panic("variable init loop!") // TODO (Gianluca): review!
	// }
	// declarations[v.name] = nil
	// value, err := tc.evalConstantExpr(v.expr)
	// if err != nil {
	// 	return err
	// }
	// declarations[v.name] = value
	return nil
}

// TODO (Gianluca): remove, declared to make this file build.
type pkgConstant struct{}

func (tc *typechecker) evalConstantExpr(expr ast.Expression) (pkgConstant, error) {

	// TODO (Gianluca): to review.

	// switch expr := expr.(type) {

	// case *ast.String:
	// 	return UntypedValue(expr.Text, nil), nil

	// case *ast.Rune:
	// 	return UntypedValue(newConstantRune(expr.Value), nil), nil

	// case *ast.Int:
	// 	return UntypedValue(newConstantInt(&expr.Value), nil), nil

	// case *ast.Float:
	// 	return UntypedValue(newConstantFloat(&expr.Value), nil), nil

	// case *ast.Parentesis:
	// 	return imp.evalConstantExpr(expr.Expr)

	// case *ast.UnaryOperator:
	// 	e, err := imp.evalConstantExpr(expr.Expr)
	// 	if err != nil {
	// 		return pkgConstant{}, err
	// 	}
	// 	switch expr.Op {
	// 	case ast.OperatorNot:
	// 		b, ok := e.value.(bool)
	// 		if !ok {
	// 			return pkgConstant{}, imp.errorf(expr, "invalid operation: ! %s", typeof(e))
	// 		}
	// 		e.value = !b
	// 	case ast.OperatorAddition:
	// 		_, ok := e.value.(ConstantNumber)
	// 		if !ok {
	// 			return pkgConstant{}, imp.errorf(expr, "invalid operation: + %s", typeof(e))
	// 		}
	// 	case ast.OperatorSubtraction:
	// 		n, ok := e.value.(ConstantNumber)
	// 		if !ok {
	// 			return pkgConstant{}, imp.errorf(expr, "invalid operation: + %s", typeof(e))
	// 		}
	// 		e.value = n.Neg()
	// 	case ast.OperatorAmpersand:
	// 		return pkgConstant{}, imp.errorf(expr, "cannot take the address of %v", e)
	// 	case ast.OperatorMultiplication:
	// 		return pkgConstant{}, imp.errorf(expr, "invalid indirect of %v (type %s)", e, typeof(e))
	// 	default:
	// 		return pkgConstant{}, fmt.Errorf("unknown unary operator %s", expr.Op)
	// 	}
	// 	return e, nil

	// case *ast.BinaryOperator:
	// 	expr1, err := imp.evalConstantExpr(expr.Expr1)
	// 	if err != nil {
	// 		return pkgConstant{}, err
	// 	}
	// 	expr2, err := imp.evalConstantExpr(expr.Expr2)
	// 	if err != nil {
	// 		return pkgConstant{}, err
	// 	}
	// 	if n1, ok := expr1.value.(ConstantNumber); ok {
	// 		if n2, ok := expr2.value.(ConstantNumber); ok {
	// 			n, err := n1.BinaryOp(expr.Op, n2)
	// 			if err != nil {
	// 				return pkgConstant{}, imp.errorf(expr, "%s", err)
	// 			}
	// 			return UntypedValue(n, nil), nil
	// 		}
	// 	}
	// 	if s1, ok := expr1.value.(string); ok {
	// 		if s2, ok := expr2.value.(string); ok {
	// 			if expr.Op != ast.OperatorAddition {
	// 				return pkgConstant{}, imp.errorf(expr,
	// 					"invalid operation: %v - %v (operator %s not defined on string)",
	// 					expr1, expr2, expr.Op)
	// 			}
	// 			return UntypedValue(s1+s2, nil), nil
	// 		}
	// 	}
	// 	return pkgConstant{}, imp.errorf(expr, "invalid operation: %v + %v (mismatched types %s and %s)",
	// 		expr1, expr2, typeof(expr1), typeof(expr2))

	// case *ast.Identifier:
	// 	decl, ok := imp.packageBlock[expr.Name]
	// 	if !ok {
	// 		decl, ok = imp.fileBlock[expr.Name]
	// 	}
	// 	if !ok {
	// 		return pkgConstant{}, fmt.Errorf("undefined %s", expr.Name)
	// 	}
	// 	if _, ok := decl.node.(*ast.Const); ok {
	// 		err := imp.initConstant(decl)
	// 		if err != nil {
	// 			return pkgConstant{}, err
	// 		}
	// 		return imp.pkg.Declarations[decl.name].(pkgConstant), nil
	// 	}

	// case *ast.Selector:
	// 	if ident, ok := expr.Expr.(*ast.Identifier); ok {
	// 		if p, ok := imp.fileBlock[ident.Name]; ok && p.kind == packageIdentifier {
	// 			v, ok := p.pkg.Declarations[expr.Ident]
	// 			if !ok {
	// 				panic(fmt.Sprintf("undefined %s", expr))
	// 			}
	// 			switch v := v.(type) {
	// 			case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8, string, bool:
	// 				return UntypedValue(v, nil), nil
	// 			case pkgConstant:
	// 				return v, nil
	// 			}
	// 		}
	// 	}

	// }

	return pkgConstant{}, tc.errorf(expr, "const initializer %s is not a constant", expr)
}
