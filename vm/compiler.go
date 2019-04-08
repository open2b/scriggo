// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"reflect"
	"scrigo/ast"
	"scrigo/parser"
)

type Compiler struct {
	parser   *parser.Parser
	pkg      *Package
	typeinfo map[ast.Node]*parser.TypeInfo
}

func NewCompiler(r parser.Reader, packages map[string]*parser.GoPackage) *Compiler {
	c := &Compiler{}
	c.parser = parser.New(r, packages, true)
	return c
}

func (c *Compiler) Compile(path string) (*Package, error) {
	tree, err := c.parser.Parse(path, ast.ContextNone)
	if err != nil {
		return nil, err
	}
	tci := c.parser.TypeCheckInfos()
	c.typeinfo = tci["/test.go"].TypeInfo
	node := tree.Nodes[0].(*ast.Package)
	pkg, err := c.compilePackage(node)
	if err != nil {
		return nil, err
	}
	return pkg, nil
}

func (c *Compiler) compilePackage(node *ast.Package) (*Package, error) {
	pkg := NewPackage(node.Name)
	for _, dec := range node.Declarations {
		if n, ok := dec.(*ast.Func); ok {
			err := c.compileFunction(pkg, n)
			if err != nil {
				return nil, err
			}
		}
	}
	return pkg, nil
}

// compileExpr compiles expression expr using fb and puts results into
// reg.
func (c *Compiler) compileExpr(expr ast.Expression, fb *FunctionBuilder, reg int8) {
	switch expr := expr.(type) {

	case *ast.BinaryOperator:
		c.compileExpr(expr.Expr1, fb, 1)
		c.compileExpr(expr.Expr2, fb, 2)
		switch expr.Operator() {
		case ast.OperatorAddition:
			panic("TODO: not implemented")
		}

	case *ast.CompositeLiteral:
		switch expr.Type.(*ast.Value).Val.(reflect.Type).Kind() {
		case reflect.Slice:
			panic("TODO: not implemented")
		case reflect.Array:
			panic("TODO: not implemented")
		case reflect.Struct:
			panic("TODO: not implemented")
		case reflect.Map:
			panic("TODO: not implemented")
		}

	case *ast.UnaryOperator:
		c.compileExpr(expr.Expr, fb, reg)
		kind := c.typeinfo[expr.Expr].Type.Kind()
		switch expr.Operator() {
		case ast.OperatorNot:
			panic("TODO: not implemented")
		case ast.OperatorSubtraction:
			// TODO (Gianluca): should be z = 0 - x (i.e. z = -x).
			fb.Sub(true, 0, reg, reg, kind)
		default:
			panic("TODO: not implemented")
		}

	case *ast.Value:
		kind := c.typeinfo[expr].Type.Kind()
		switch kind {
		case reflect.Int:
			c := fb.MakeIntConstant(int64(expr.Val.(int)))
			fb.Move(false, reg, c, kind)
		default:
			panic("TODO: not implemented")

		}
	}
}

func (c *Compiler) compileNodes(nodes []ast.Node, fb *FunctionBuilder) {
	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.Assignment:
			if len(node.Variables) == 1 && len(node.Values) == 1 {
				variableExpr := node.Variables[0]
				valueExpr := node.Values[0]
				if isBlankIdentifier(variableExpr) {
					continue
				}
				kind := c.typeinfo[valueExpr].Type.Kind()
				switch node.Type {
				case ast.AssignmentDeclaration:
					variableReg := fb.NewVar(variableExpr.(*ast.Identifier).Name, kind)
					c.compileExpr(valueExpr, fb, variableReg)
				case ast.AssignmentSimple:
					variableReg := fb.VariableRegister(variableExpr.(*ast.Identifier).Name)
					c.compileExpr(valueExpr, fb, variableReg)
				}
			} else {
				panic("TODO: not implemented")
			}

		case *ast.Call:
			fb.Call(0, StackShift{}) // TODO

		case *ast.If:
			fb.EnterScope()
			if node.Assignment != nil {
				c.compileNodes([]ast.Node{node.Assignment}, fb)
			}
			var k bool
			var x, y int8
			var o Condition
			var kind reflect.Kind
			switch cond := node.Condition.(type) {
			case *ast.BinaryOperator:
				kind = c.typeinfo[cond.Expr1].Type.Kind()
				x := fb.NewRegister(kind)
				y := fb.NewRegister(kind)
				c.compileExpr(cond.Expr1, fb, x)
				c.compileExpr(cond.Expr2, fb, y)
				switch cond.Operator() {
				case ast.OperatorEqual:
					o = ConditionEqual
				}
			default:
				x := fb.NewRegister(kind)
				c.compileExpr(cond, fb, x)
				o = ConditionEqual
				y = fb.MakeIntConstant(1)
			}
			fb.If(k, x, o, y, kind)
			elsLabel := fb.NewLabel()
			fb.Goto(elsLabel)
			c.compileNodes(node.Then.Nodes, fb)
			endIfLabel := fb.NewLabel()
			fb.Goto(endIfLabel)
			fb.SetLabelAddr(elsLabel)
			if node.Else != nil {
				switch els := node.Else.(type) {
				case *ast.If:
					c.compileNodes([]ast.Node{els}, fb)
				case *ast.Block:
					c.compileNodes(els.Nodes, fb)
				}
			}
			fb.SetLabelAddr(endIfLabel)
			fb.ExitScope()

		case *ast.For:
			fb.EnterScope()
			if node.Init != nil {
				c.compileNodes([]ast.Node{node.Init}, fb)
			}
			if node.Condition != nil {
				panic("TODO: not implemented")
			}
			forLabel := fb.NewLabel()
			fb.SetLabelAddr(forLabel)
			if node.Post != nil {
				c.compileNodes([]ast.Node{node.Init}, fb)
			}
			c.compileNodes(node.Body, fb)
			fb.Goto(forLabel)
			fb.ExitScope()

		}
	}
}

func (c *Compiler) compileFunction(pkg *Package, node *ast.Func) error {

	fn := pkg.NewFunction(node.Ident.Name, nil, nil, node.Type.IsVariadic)
	fb := fn.Builder()

	fb.EnterScope()
	c.compileNodes(node.Body.Nodes, fb)
	fb.End()
	fb.ExitScope()

	return nil
}

// isBlankIdentifier indicates if expr is an identifier representing the blank
// identifier "_".
func isBlankIdentifier(expr ast.Expression) bool {
	ident, ok := expr.(*ast.Identifier)
	return ok && ident.Name == "_"
}
