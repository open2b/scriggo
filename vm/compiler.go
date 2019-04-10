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
		switch n := dec.(type) {
		case *ast.Func:
			fn := pkg.NewFunction(n.Ident.Name, nil, nil, n.Type.IsVariadic)
			fb := fn.Builder()
			fb.EnterScope()
			c.compileNodes(n.Body.Nodes, fb)
			fb.End()
			fb.ExitScope()
		case *ast.Var:
			panic("TODO: not implemented")
		}
	}
	return pkg, nil
}

// quickCompile checks if expr is a value or a register, putting it into out. If
// it's neither of them, both isValue and isRegister are false and content of
// out is unspecified.
func (c *Compiler) quickCompile(expr ast.Expression, fb *FunctionBuilder) (out int8, isValue, isRegister bool) {
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
func (c *Compiler) compileExpr(expr ast.Expression, fb *FunctionBuilder, reg int8) {
	switch expr := expr.(type) {

	case *ast.BinaryOperator:
		kind := c.typeinfo[expr.Expr1].Type.Kind()
		var op2 int8
		var ky bool
		{
			out, isValue, isRegister := c.quickCompile(expr.Expr2, fb)
			if isValue {
				op2 = out
				ky = true
			} else if isRegister {
				op2 = out
			} else {
				op2 = int8(fb.numRegs[kind])
				fb.allocRegister(kind, op2)
				c.compileExpr(expr.Expr2, fb, op2)
			}
		}
		c.compileExpr(expr.Expr1, fb, reg)
		switch expr.Operator() {
		case ast.OperatorAddition:
			fb.Add(ky, reg, op2, reg, kind)
		case ast.OperatorSubtraction:
			fb.Sub(ky, reg, op2, reg, kind)
		case ast.OperatorMultiplication:
			fb.Mul(reg, op2, reg, kind)
		case ast.OperatorDivision:
			fb.Div(reg, op2, reg, kind)
		case ast.OperatorModulo:
			fb.Rem(reg, op2, reg, kind)
		default:
			panic("TODO: not implemented")
		}

	case *ast.Call:
		ok := c.callBuiltin(expr, fb)
		if !ok {
			fb.Call(0, 0, StackShift{}) // TODO
		}

	case *ast.CompositeLiteral:
		switch expr.Type.(*ast.Value).Val.(reflect.Type).Kind() {
		case reflect.Slice:
			typ := expr.Type.(*ast.Value).Val.(reflect.Type)
			fb.MakeSlice(typ, 0, 0, reg)
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
		out, isValue, isRegister := c.quickCompile(expr, fb)
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

// compileValueToVar assign value to variable.
func (c *Compiler) compileValueToVar(value, variable ast.Expression, fb *FunctionBuilder, isDecl bool) {
	kind := c.typeinfo[value].Type.Kind()
	if isBlankIdentifier(variable) {
		switch value.(type) {
		case *ast.Call:
			c.compileNodes([]ast.Node{value}, fb)
		}
		return
	}
	var varReg int8
	if isDecl {
		varReg = fb.NewVar(variable.(*ast.Identifier).Name, kind)
	} else {
		varReg = fb.VariableRegister(variable.(*ast.Identifier).Name)
	}
	out, isValue, isRegister := c.quickCompile(value, fb)
	if isValue {
		fb.Move(true, out, varReg, kind)
	} else if isRegister {
		fb.Move(false, out, varReg, kind)
	} else {
		c.compileExpr(value, fb, varReg)
	}
}

// TODO (Gianluca): a builtin can be shadowed, but the compiler can't know it.
// Typechecker should flag *ast.Call nodes with a boolean indicating if it's a
// builtin.
func (c *Compiler) callBuiltin(call *ast.Call, fb *FunctionBuilder) (ok bool) {
	if ident, ok := call.Func.(*ast.Identifier); ok {
		var i instruction
		switch ident.Name {
		case "len":
			typ := c.typeinfo[call.Args[0]].Type
			kind := typ.Kind()
			var a, b int8
			out, _, isRegister := c.quickCompile(call.Args[0], fb)
			if isRegister {
				b = out
			} else {
				reg := int8(fb.numRegs[kind])
				fb.allocRegister(kind, reg)
				c.compileExpr(call.Args[0], fb, reg)
				b = reg
			}
			switch typ {
			case reflect.TypeOf(""): // TODO (Gianluca): or should check for kind string?
				a = 0
			default:
				a = 1
			case reflect.TypeOf([]byte{}):
				a = 2
			}
			i = instruction{op: opLen, a: a, b: b}
		default:
			return false
		}
		fb.fn.body = append(fb.fn.body, i)
		return true
	}
	return false
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
					c.compileValueToVar(node.Values[0], node.Variables[0], fb, node.Type == ast.AssignmentDeclaration)
				default:
					panic("TODO: not implemented")
				}
			} else if len(node.Variables) == len(node.Values) {
				for i := range node.Variables {
					c.compileValueToVar(node.Values[i], node.Variables[i], fb, node.Type == ast.AssignmentDeclaration)
				}
			} else {
				panic("TODO: not implemented")
			}

		case *ast.Block:
			fb.EnterScope()
			c.compileNodes(node.Nodes, fb)
			fb.ExitScope()

		case *ast.Call:
			ok := c.callBuiltin(node, fb)
			if !ok {
				fb.Call(0, 0, StackShift{}) // TODO
			}

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

		case *ast.Switch:
			fb.EnterScope()
			if node.Init != nil {
				c.compileNodes([]ast.Node{node.Init}, fb)
			}
			kind := c.typeinfo[node.Expr].Type.Kind()
			expr := fb.NewRegister(kind)
			c.compileExpr(node.Expr, fb, expr)
			caseLabels := make([]uint32, 0, len(node.Cases))
			for _, cas := range node.Cases {
				caseLab := fb.NewLabel()
				caseLabels = append(caseLabels, caseLab)
				for _, cond := range cas.Expressions {
					var ky bool
					var y int8
					out, isValue, isRegister := c.quickCompile(cond, fb)
					if isValue {
						ky = true
						y = out
					} else if isRegister {
						y = out
					} else {
						c.compileExpr(cond, fb, y)
					}
					fb.If(ky, expr, ConditionEqual, y, kind)
					nextLabel := fb.NewLabel()
					fb.Goto(nextLabel)
					fb.Goto(caseLab)
					fb.SetLabelAddr(nextLabel)
				}
			}
			endSwitch := fb.NewLabel()
			for i, cas := range node.Cases {

				fb.SetLabelAddr(caseLabels[i])
				c.compileNodes(cas.Body, fb)
				fb.Goto(endSwitch)
			}
			fb.SetLabelAddr(endSwitch)
			fb.ExitScope()

		case *ast.Var:
			for i := range node.Identifiers {
				c.compileValueToVar(node.Values[i], node.Identifiers[i], fb, true)
			}

		}
	}
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
//
// 	ConditionEqual             Condition = iota // x == y
// 	ConditionNotEqual                           // x != y
// 	ConditionLess                               // x <  y
// 	ConditionLessOrEqual                        // x <= y
// 	ConditionGreater                            // x >  y
// 	ConditionGreaterOrEqual                     // x >= y
// 	ConditionEqualLen                           // len(x) == y
// 	ConditionNotEqualLen                        // len(x) != y
// 	ConditionLessLen                            // len(x) <  y
// 	ConditionLessOrEqualLen                     // len(x) <= y
// 	ConditionGreaterLen                         // len(x) >  y
// 	ConditionGreaterOrEqualLen                  // len(x) >= y
// 	ConditionNil                                // x == nil
// 	ConditionNotNil                             // x != nil
//
func (c *Compiler) compileCondition(expr ast.Expression, fb *FunctionBuilder) (x, y int8, kind reflect.Kind, o Condition, yk bool) {
	switch cond := expr.(type) {
	case *ast.BinaryOperator:
		kind = c.typeinfo[cond.Expr1].Type.Kind()
		var out int8
		var isValue, isRegister bool
		out, _, isRegister = c.quickCompile(cond.Expr1, fb)
		if isRegister {
			x = out
		} else {
			x = fb.NewRegister(kind)
			c.compileExpr(cond.Expr1, fb, x)
		}
		if isNil(cond.Expr2) {
			switch cond.Operator() {
			case ast.OperatorEqual:
				o = ConditionNil
			case ast.OperatorNotEqual:
				o = ConditionNotNil
			}
		} else {
			out, isValue, isRegister = c.quickCompile(cond.Expr2, fb)
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
		}

	default:
		x := fb.NewRegister(kind)
		c.compileExpr(cond, fb, x)
		o = ConditionEqual
		y = fb.MakeIntConstant(1)
	}
	return x, y, kind, o, yk
}

// isNil indicates if expr is the nil identifier.
func isNil(expr ast.Expression) bool {
	ident, ok := expr.(*ast.Identifier)
	if !ok {
		return false
	}
	return ident.Name == "nil"
}
