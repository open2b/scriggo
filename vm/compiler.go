// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"fmt"
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

// immediate checks if expr is a value or a register, putting it into out. If
// it's neither of them, both isValue and isRegister are false and content of
// out is unspecified.
func (c *Compiler) immediate(expr ast.Expression, fb *FunctionBuilder) (out int8, isValue, isRegister bool) {
	switch expr := expr.(type) {
	case *ast.Int: // TODO (Gianluca): must be removed, is here because of a type-checker's bug.
		i := int64(expr.Value.Int64())
		return int8(i), true, false
	case *ast.Identifier:
		v := fb.VariableRegister(expr.Name)
		return v, false, true
	case *ast.Value:
		kind := c.typeinfo[expr].Type.Kind()
		switch kind {
		case reflect.Int:
			n := expr.Val.(int)
			if n < 0 || n > 127 {
				c := fb.MakeIntConstant(int64(n))
				return c, false, true
			} else {
				return int8(n), true, false
			}
		case reflect.String:
			c := fb.MakeStringConstant(expr.Val.(string))
			return c, false, true
		default:
			panic("TODO: not implemented")

		}
	}
	return 0, false, false
}

// compileExpr compiles expression expr using fb and puts results into
// reg.
// TODO (Gianluca): optimize using "immediate".
func (c *Compiler) compileExpr(expr ast.Expression, fb *FunctionBuilder, reg int8) {
	switch expr := expr.(type) {

	case *ast.BinaryOperator:
		kind := c.typeinfo[expr.Expr1].Type.Kind()
		op2 := int8(fb.numRegs[kind])
		fb.allocRegister(kind, op2)
		c.compileExpr(expr.Expr1, fb, reg)
		c.compileExpr(expr.Expr2, fb, op2)
		switch expr.Operator() {
		case ast.OperatorAddition:
			fb.Add(false, reg, op2, reg, kind)
		case ast.OperatorSubtraction:
			fb.Sub(false, reg, op2, reg, kind)
		case ast.OperatorMultiplication:
			fb.Mul(reg, op2, reg, kind)
		case ast.OperatorDivision:
			panic("TODO: not implemented")
		case ast.OperatorModulo:
			panic("TODO: not implemented")
		default:
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

	case *ast.Value, *ast.Int, *ast.Identifier:
		kind := c.typeinfo[expr].Type.Kind()
		out, isValue, isRegister := c.immediate(expr, fb)
		if isValue {
			fb.Move(true, out, reg, kind)
		} else if isRegister {
			fb.Move(false, out, reg, kind)
		} else {
			panic("bug")
		}

	default:
		panic(fmt.Sprintf("compileExpr currently does not support %T nodes", expr))

	}

}

// compileSimpleAssignmentOrDeclaration compiles a simple assignment or a
// declaration.
// TODO (Gianluca): find a shorter/better name.
func (c *Compiler) compileSimpleAssignmentOrDeclaration(variableExpr, valueExpr ast.Expression, fb *FunctionBuilder, isDecl bool) {
	kind := c.typeinfo[valueExpr].Type.Kind()
	if isBlankIdentifier(variableExpr) {
		switch valueExpr.(type) {
		case *ast.Call:
			c.compileNodes([]ast.Node{valueExpr}, fb)
		}
	} else {
		var variableReg int8
		if isDecl {
			variableReg = fb.NewVar(variableExpr.(*ast.Identifier).Name, kind)
		} else {
			variableReg = fb.VariableRegister(variableExpr.(*ast.Identifier).Name)
		}
		out, isValue, isRegister := c.immediate(valueExpr, fb)
		if isValue {
			fb.Move(true, out, variableReg, kind)
		} else if isRegister {
			fb.Move(false, out, variableReg, kind)
		} else {
			c.compileExpr(valueExpr, fb, variableReg)
		}
	}
}

func (c *Compiler) compileNodes(nodes []ast.Node, fb *FunctionBuilder) {
	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.Assignment:
			if len(node.Variables) == 1 && len(node.Values) == 1 {
				switch node.Type {
				case ast.AssignmentIncrement:
					name := node.Variables[0].(*ast.Identifier).Name
					reg := fb.VariableRegister(name)
					kind := c.typeinfo[node.Variables[0]].Type.Kind()
					fb.Add(true, reg, 1, reg, kind)
				case ast.AssignmentDecrement:
					name := node.Variables[0].(*ast.Identifier).Name
					reg := fb.VariableRegister(name)
					kind := c.typeinfo[node.Variables[0]].Type.Kind()
					fb.Add(true, reg, -1, reg, kind)
				case ast.AssignmentDeclaration, ast.AssignmentSimple:
					c.compileSimpleAssignmentOrDeclaration(node.Variables[0], node.Values[0], fb, node.Type == ast.AssignmentDeclaration)
				default:
					panic("TODO: not implemented")
				}
			} else if len(node.Variables) == len(node.Values) {
				for i := range node.Variables {
					c.compileSimpleAssignmentOrDeclaration(node.Variables[i], node.Values[i], fb, node.Type == ast.AssignmentDeclaration)
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
			x, y, kind, o, ky := c.compileCondition(node.Condition, fb)
			fb.If(ky, x, o, y, kind)
			if node.Else == nil { // TODO (Gianluca): can "then" and "else" be unified in some way?
				endIfLabel := fb.NewLabel()
				fb.Goto(endIfLabel)
				c.compileNodes(node.Then.Nodes, fb)
				fb.SetLabelAddr(endIfLabel)
			} else {
				elseLabel := fb.NewLabel()
				fb.Goto(elseLabel)
				c.compileNodes(node.Then.Nodes, fb)
				endIfLabel := fb.NewLabel()
				fb.Goto(endIfLabel)
				fb.SetLabelAddr(elseLabel)
				if node.Else != nil {
					switch els := node.Else.(type) {
					case *ast.If:
						c.compileNodes([]ast.Node{els}, fb)
					case *ast.Block:
						c.compileNodes(els.Nodes, fb)
					}
				}
				fb.SetLabelAddr(endIfLabel)
			}
			fb.ExitScope()

		case *ast.For:
			fb.EnterScope()
			if node.Init != nil {
				c.compileNodes([]ast.Node{node.Init}, fb)
			}
			if node.Condition != nil {
				forLabel := fb.NewLabel()
				fb.SetLabelAddr(forLabel)
				x, y, kind, o, ky := c.compileCondition(node.Condition, fb)
				fb.If(ky, x, o, y, kind)
				endForLabel := fb.NewLabel()
				fb.Goto(endForLabel)
				if node.Post != nil {
					c.compileNodes([]ast.Node{node.Post}, fb)
				}
				c.compileNodes(node.Body, fb)
				fb.Goto(forLabel)
				fb.SetLabelAddr(endForLabel)
			} else {
				forLabel := fb.NewLabel()
				fb.SetLabelAddr(forLabel)
				if node.Post != nil {
					c.compileNodes([]ast.Node{node.Post}, fb)
				}
				c.compileNodes(node.Body, fb)
				fb.Goto(forLabel)
			}
			fb.ExitScope()

		case *ast.Return:
			fb.Return()

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

// compileCondition compiles expr using fb. Returns the two values of the
// condition (x and y), a kind, the condition ad a boolean ky which indicates
// whether y is a constant value.
func (c *Compiler) compileCondition(expr ast.Expression, fb *FunctionBuilder) (x, y int8, kind reflect.Kind, o Condition, yk bool) {
	// TODO (Gianluca): all Condition* must be generated
	switch cond := expr.(type) {
	case *ast.BinaryOperator:
		kind = c.typeinfo[cond.Expr1].Type.Kind()
		var out int8
		var isValue, isRegister bool
		out, _, isRegister = c.immediate(cond.Expr1, fb)
		if isRegister {
			x = out
		} else {
			x = fb.NewRegister(kind)
			c.compileExpr(cond.Expr1, fb, x)
		}
		out, isValue, isRegister = c.immediate(cond.Expr2, fb)
		if isValue {
			y = out
			yk = true
		} else if isRegister {
			y = out
		} else {
			y = fb.NewRegister(kind)
			c.compileExpr(cond.Expr2, fb, y)
		}
		switch cond.Operator() {
		case ast.OperatorEqual:
			o = ConditionEqual
		case ast.OperatorLess:
			o = ConditionLess
		case ast.OperatorLessOrEqual:
			o = ConditionLessOrEqual
		case ast.OperatorGreater:
			o = ConditionGreater
		case ast.OperatorGreaterOrEqual:
			o = ConditionGreaterOrEqual
		}
	default:
		x := fb.NewRegister(kind)
		c.compileExpr(cond, fb, x)
		o = ConditionEqual
		y = fb.MakeIntConstant(1)
	}
	return x, y, kind, o, yk
}
