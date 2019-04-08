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

// compileExpression compiles expression expr using fb and puts results into
// reg.
func (c *Compiler) compileExpression(expr ast.Expression, fb *FunctionBuilder, reg int8) {
	switch expr := expr.(type) {
	case *ast.Value:
		kind := c.typeinfo[expr].Type.Kind()
		switch kind {
		case reflect.Int:
			fb.MakeIntConstant(expr.Val.(int64))
		default:
			panic("TODO: not implemented")
		}
	case *ast.UnaryOperator:
		c.compileExpression(expr.Expr, fb, 1)
		switch expr.Operator() {
		case ast.OperatorNot:
			panic("TODO: not implemented")
		}
	case *ast.BinaryOperator:
		c.compileExpression(expr.Expr1, fb, 1)
		c.compileExpression(expr.Expr2, fb, 2)
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
	}
}

func (c *Compiler) compileNodes(nodes []ast.Node, fb *FunctionBuilder) error {
	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.If:
			if node.Assignment != nil {
				panic("TODO: not implemented")
			}
			var k bool
			var x, y int8
			var o Condition
			var kind reflect.Kind
			if binOp, ok := node.Condition.(*ast.BinaryOperator); ok {
				kind = c.typeinfo[binOp.Expr1].Type.Kind()
				expr1 := fb.NewRegister(kind)
				expr2 := fb.NewRegister(kind)
				c.compileExpression(binOp.Expr1, fb, expr1)
				c.compileExpression(binOp.Expr2, fb, expr2)
				switch binOp.Operator() {
				case ast.OperatorEqual:
					o = ConditionEqual
				}
			}
			fb.If(k, x, o, y, kind)
			elsLabel := fb.NewEmptyLabel()
			fb.Goto(elsLabel)
			c.compileNodes(node.Then.Nodes, fb)
			endIfLabel := fb.NewEmptyLabel()
			fb.Goto(endIfLabel)
			fb.UpdateLabelWithCurrentPos(elsLabel)
			if node.Else != nil {
				switch els := node.Else.(type) {
				case *ast.If:
					c.compileNodes([]ast.Node{els}, fb)
				case *ast.Block:
					c.compileNodes(els.Nodes, fb)
				}
			}
			fb.UpdateLabelWithCurrentPos(endIfLabel)

		case *ast.Assignment:
			if len(node.Variables) == 1 && len(node.Values) == 1 {
				variableExpr := node.Variables[0]
				valueExpr := node.Values[0]
				if isBlankIdentifier(variableExpr) {
					continue
				}
				kind := c.typeinfo[valueExpr].Type.Kind()
				if node.Type == ast.AssignmentDeclaration {
					variableReg := int8(fb.numRegs[kind])
					fb.allocRegister(kind, int8(variableReg))
					valueReg := int8(fb.numRegs[kind])
					fb.allocRegister(kind, valueReg)
					c.compileExpression(valueExpr, fb, valueReg)
					fb.Move(false, valueReg, variableReg, kind)
				} else if node.Type == ast.AssignmentSimple {
					panic("TODO: not implemented")
					// valueReg := int8(fb.numRegs[kind])
					// fb.allocRegister(kind, valueReg)
					// c.compileExpression(valueExpr, fb, valueReg)
					// fb.Move(false, valueReg, variableReg, kind)
				}
			}
		}
	}
	return nil
}

func (c *Compiler) compileFunction(pkg *Package, node *ast.Func) error {

	fn := pkg.NewFunction(node.Ident.Name, nil, nil, node.Type.IsVariadic)
	fb := fn.Builder()

	c.compileNodes(node.Body.Nodes, fb)

	// 	switch n := n.(type) {
	// 	case *ast.Var:
	// 		for i := range n.Identifiers {
	// 			value := n.Values[i]
	// 			kind := c.typeinfo[value].Type.Kind()
	// 			fb.Move(false, 5, 0, kind)
	// 		}
	// 	}

	return nil
}

// isBlankIdentifier indicates if expr is an identifier representing the blank
// identifier "_".
func isBlankIdentifier(expr ast.Expression) bool {
	ident, ok := expr.(*ast.Identifier)
	return ok && ident.Name == "_"
}
