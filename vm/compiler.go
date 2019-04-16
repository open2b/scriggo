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
	parser           *parser.Parser
	currentPkg       *Package
	typeinfo         map[ast.Node]*parser.TypeInfo
	fb               *FunctionBuilder // current function builder.
	importableGoPkgs map[string]*parser.GoPackage
}

func NewCompiler(r parser.Reader, packages map[string]*parser.GoPackage) *Compiler {
	c := &Compiler{
		importableGoPkgs: packages,
	}
	c.parser = parser.New(r, packages, true)
	return c
}

// goPackageToVMPackage converts a Parser's GoPackage to a VM's Package.
func goPackageToVMPackage(goPkg *parser.GoPackage) *Package {
	pkg := NewPackage(goPkg.Name)
	for ident, value := range goPkg.Declarations {
		_ = ident
		if t, ok := value.(reflect.Type); ok {
			// TODO: import type
			_ = t
			continue
		}
		if reflect.TypeOf(value).Kind() == reflect.Ptr {
			pkg.DefineVariable(ident, value)
			continue
		}
		if reflect.TypeOf(value).Kind() == reflect.Func {
			nativeFunc := NewNativeFunction(ident, value)
			index, ok := pkg.AddNativeFunction(nativeFunc)
			if !ok {
				panic("TODO: not implemented")
			}
			pkg.gofunctionsNames[ident] = int8(index)
			continue
		}
		// TODO: import constant
	}
	return pkg
}

// Compile compiles path and returns its package.
func (c *Compiler) Compile(path string) (*Package, error) {
	tree, err := c.parser.Parse(path, ast.ContextNone)
	if err != nil {
		return nil, err
	}
	tci := c.parser.TypeCheckInfos()
	c.typeinfo = tci["/test.go"].TypeInfo
	node := tree.Nodes[0].(*ast.Package)
	c.compilePackage(node)
	return c.currentPkg, nil
}

// compilePackage compiles the pkg package.
func (c *Compiler) compilePackage(pkg *ast.Package) {
	c.currentPkg = NewPackage(pkg.Name)
	for _, dec := range pkg.Declarations {
		switch n := dec.(type) {
		case *ast.Var:
			// TODO (Gianluca): this makes a new init function for every
			// variable, which is wrong. Putting initFn declaration
			// outside this switch is wrong too: init.-1 cannot be created
			// if there's no need.
			initFn, _ := c.currentPkg.NewFunction("init.-1", reflect.FuncOf(nil, nil, false))
			initBuilder := initFn.Builder()
			if len(n.Identifiers) == 1 && len(n.Values) == 1 {
				currentBuilder := c.fb
				c.fb = initBuilder
				reg := c.fb.NewRegister(reflect.Int)
				c.compileExpr(n.Values[0], reg)
				c.fb = currentBuilder
				name := "A"                 // TODO
				v := interface{}(int64(10)) // TODO
				c.currentPkg.DefineVariable(name, v)
			} else {
				panic("TODO: not implemented")
			}
		case *ast.Func:
			fn, index := c.currentPkg.NewFunction(n.Ident.Name, n.Type.Reflect)
			c.fb = fn.Builder()
			c.fb.EnterScope()
			// Binds function argument names to pre-allocated registers.
			shift := StackShift{}
			fillParametersTypes(n.Type.Result)
			for _, res := range n.Type.Result {
				resType := res.Type.(*ast.Value).Val.(reflect.Type)
				kind := resType.Kind()
				retReg := c.fb.NewRegister(kind)
				_ = retReg // TODO (Gianluca): add support for named return parameters. Binding retReg to the name of the paramter should be enough.
				shift[kindToVMIndex(kind)]++
			}
			fillParametersTypes(n.Type.Parameters)
			for _, par := range n.Type.Parameters {
				parType := par.Type.(*ast.Value).Val.(reflect.Type)
				kind := parType.Kind()
				argReg := c.fb.NewRegister(kind)
				c.fb.BindVarReg(par.Ident.Name, argReg)
			}
			c.currentPkg.functionsNames[n.Ident.Name] = index
			c.compileNodes(n.Body.Nodes)
			c.fb.End()
			c.fb.ExitScope()
		case *ast.Import:
			if n.Tree == nil { // Go package.
				parserGoPkg, ok := c.importableGoPkgs[n.Path]
				if !ok {
					panic(fmt.Errorf("bug: trying to import Go package %q, but it's not available (availables are: %v)!", n.Path, c.importableGoPkgs))
				}
				goPkg := goPackageToVMPackage(parserGoPkg)
				pkgIndex := c.currentPkg.Import(goPkg)
				c.currentPkg.packagesNames[parserGoPkg.Name] = pkgIndex
				c.currentPkg.isGoPkg[parserGoPkg.Name] = true
			}
		}
	}
}

// quickCompileExpr checks if expr is a value or a register, putting it into
// out. If it's neither of them, both isValue and isRegister are false and
// content of out is unspecified.
func (c *Compiler) quickCompileExpr(expr ast.Expression) (out int8, isValue, isRegister bool) {
	switch expr := expr.(type) {
	case *ast.Int: // TODO (Gianluca): must be removed, is here because of a type-checker's bug.
		i := int64(expr.Value.Int64())
		return int8(i), true, false
	case *ast.Identifier:
		v := c.fb.ScopeLookup(expr.Name)
		return v, false, true
	case *ast.Value:
		kind := c.typeinfo[expr].Type.Kind()
		switch kind {
		case reflect.Int:
			n := expr.Val.(int)
			if n < 0 || n > 127 {
				c := c.fb.MakeIntConstant(int64(n))
				return c, false, true
			} else {
				return int8(n), true, false
			}
		case reflect.String:
			sConst := c.fb.MakeStringConstant(expr.Val.(string))
			reg := c.fb.NewRegister(reflect.String)
			c.fb.Move(true, sConst, reg, reflect.String)
			return reg, false, true
		case reflect.Float64:
			// TODO (Gianluca): handle all kind of floats.
			v := int8(expr.Val.(float64))
			return v, true, false
		default:
			panic("TODO: not implemented")

		}
	case *ast.String: // TODO (Gianluca): must be removed, is here because of a type-checker's bug.
		sConst := c.fb.MakeStringConstant(expr.Text)
		reg := c.fb.NewRegister(reflect.String)
		c.fb.Move(true, sConst, reg, reflect.String)
		return reg, false, true
	}
	return 0, false, false
}

// compileCall compiles call, returning the list of registers (and their
// respective kind) within which return values are inserted.
func (c *Compiler) compileCall(call *ast.Call) (regs []int8, kinds []reflect.Kind) {
	stackShift := StackShift{
		int8(c.fb.numRegs[reflect.Int]),
		int8(c.fb.numRegs[reflect.Float64]),
		int8(c.fb.numRegs[reflect.String]),
		int8(c.fb.numRegs[reflect.Interface]),
	}
	switch fun := call.Func.(type) {
	case *ast.Identifier:
		ident := fun.Name
		i, ok := c.currentPkg.functionsNames[ident]
		if !ok {
			panic("bug")
		}
		f := c.currentPkg.scrigoFunctions[i]
		funcType := f.typ
		for i := 0; i < funcType.NumOut(); i++ {
			kind := funcType.Out(i).Kind()
			regs = append(regs, c.fb.NewRegister(kind))
			kinds = append(kinds, kind)
		}
		for i := 0; i < funcType.NumIn(); i++ {
			kind := funcType.In(i).Kind()
			reg := c.fb.NewRegister(kind)
			c.compileExpr(call.Args[i], reg)
		}
		c.fb.Call(CurrentPackage, i, stackShift)
	case *ast.Selector: // currently supports only Go calls.
		pkgName := fun.Expr.(*ast.Identifier).Name
		funcName := fun.Ident
		pkgIndex := int8(c.currentPkg.packagesNames[pkgName])
		// isNative := c.currentPkg.isGoPkg[pkgName]
		goPkg := c.currentPkg.packages[pkgIndex]
		funcIndex := int8(goPkg.gofunctionsNames[funcName])
		var funcType reflect.Type
		if goPkg.nativeFunctions[funcIndex].fast != nil {
			funcType = reflect.TypeOf(goPkg.nativeFunctions[funcIndex].fast)
		} else {
			funcType = goPkg.nativeFunctions[funcIndex].value.Type()
		}
		for i := 0; i < funcType.NumOut(); i++ {
			kind := funcType.Out(i).Kind()
			regs = append(regs, c.fb.NewRegister(kind))
			kinds = append(kinds, kind)
		}
		for i := 0; i < funcType.NumIn(); i++ {
			kind := funcType.In(i).Kind()
			reg := c.fb.NewRegister(kind)
			c.compileExpr(call.Args[i], reg)
		}
		c.fb.CallFunc(pkgIndex, funcIndex, NoVariadic, stackShift)
	default:
		panic("TODO: not implemented")
	}
	return regs, kinds
}

// compileExpr compiles expression expr and puts results into reg. Compiled
// result is discarded if reg is 0.
func (c *Compiler) compileExpr(expr ast.Expression, reg int8) {
	switch expr := expr.(type) {

	case *ast.BinaryOperator:
		kind := c.typeinfo[expr.Expr1].Type.Kind()
		var op2 int8
		var ky bool
		{
			out, isValue, isRegister := c.quickCompileExpr(expr.Expr2)
			if isValue {
				op2 = out
				ky = true
			} else if isRegister {
				op2 = out
			} else {
				op2 = int8(c.fb.numRegs[kind])
				c.fb.allocRegister(kind, op2)
				c.compileExpr(expr.Expr2, op2)
			}
		}
		c.compileExpr(expr.Expr1, reg)
		switch expr.Operator() {
		case ast.OperatorAddition:
			if kind == reflect.String {
				c.fb.Concat(reg, op2, reg)
			} else {
				c.fb.Add(ky, reg, op2, reg, kind)
			}
		case ast.OperatorSubtraction:
			c.fb.Sub(ky, reg, op2, reg, kind)
		case ast.OperatorMultiplication:
			c.fb.Mul(reg, op2, reg, kind)
		case ast.OperatorDivision:
			c.fb.Div(reg, op2, reg, kind)
		case ast.OperatorModulo:
			c.fb.Rem(reg, op2, reg, kind)
		default:
			panic("TODO: not implemented")
		}

	case *ast.Call:
		ok := c.callBuiltin(expr, reg)
		if ok {
			return
		}
		if val, ok := expr.Func.(*ast.Value); ok {
			if typ, ok := val.Val.(reflect.Type); ok {
				kind := c.typeinfo[expr.Args[0]].Type.Kind()
				arg := c.fb.NewRegister(kind)
				c.compileExpr(expr.Args[0], arg)
				c.fb.Convert(arg, typ, reg)
				return
			}
		}
		regs, kinds := c.compileCall(expr)
		if reg != 0 {
			c.fb.Move(false, regs[0], reg, kinds[0])
		}

	case *ast.CompositeLiteral:
		typ := expr.Type.(*ast.Value).Val.(reflect.Type)
		switch typ.Kind() {
		case reflect.Slice:
			c.fb.Slice(typ, 0, 0, reg)
		case reflect.Array:
			panic("TODO: not implemented")
		case reflect.Struct:
			panic("TODO: not implemented")
		case reflect.Map:
			// TODO (Gianluca): handle maps with bigger size.
			size := c.fb.MakeIntConstant(int64(len(expr.KeyValues)))
			regType := c.fb.Type(typ)
			c.fb.MakeMap(regType, true, size, reg)
			if size > 0 {
				panic("TODO: not implemented")
			}
		}

	case *ast.Func:
		currentFunc := c.fb
		fn, _ := c.currentPkg.NewFunction("", reflect.FuncOf(nil, nil, expr.Type.IsVariadic))
		c.fb = fn.Builder()
		c.fb.EnterScope()
		c.compileNodes(expr.Body.Nodes)
		c.fb.End()
		c.fb.ExitScope()
		c.fb = currentFunc
		c.fb.Func(0, reflect.FuncOf(nil, nil, expr.Type.IsVariadic))

	case *ast.Selector:
		pkgName := expr.Expr.(*ast.Identifier).Name
		funcName := expr.Ident
		pkgIndex := int8(c.currentPkg.packagesNames[pkgName])
		goPkg := c.currentPkg.packages[pkgIndex]
		funcIndex := int8(goPkg.gofunctionsNames[funcName])
		c.fb.GetFunc(pkgIndex, funcIndex, reg)

	case *ast.UnaryOperator:
		c.compileExpr(expr.Expr, reg)
		// kind := c.typeinfo[expr.Expr].Type.Kind()
		switch expr.Operator() {
		case ast.OperatorNot:
			panic("TODO: not implemented")
		case ast.OperatorSubtraction:
			// TODO (Gianluca): should be z = 0 - x (i.e. z = -x).
			// c.fb.Sub(true, 0, reg, reg, kind)
			panic("TODO: not implemented")
		default:
			panic("TODO: not implemented")
		}

	case *ast.Value, *ast.Int, *ast.Identifier, *ast.String: // TODO (Gianluca): remove Int and String
		kind := c.typeinfo[expr].Type.Kind()
		out, isValue, isRegister := c.quickCompileExpr(expr)
		if isValue {
			c.fb.Move(true, out, reg, kind)
		} else if isRegister {
			c.fb.Move(false, out, reg, kind)
		} else {
			panic("bug")
		}

	default:
		panic(fmt.Sprintf("compileExpr currently does not support %T nodes", expr))

	}

}

// compileVarsGetValue assign value to variables. If variables contains more
// than one variable, value must be a function call, a map indexing operation or
// a type assertion. These last two cases involve that variables contains 2
// elements.
func (c *Compiler) compileVarsGetValue(variables []ast.Expression, value ast.Expression, isDecl bool) {
	if len(variables) == 1 {
		variable := variables[0]
		kind := c.typeinfo[value].Type.Kind()
		if isBlankIdentifier(variable) {
			switch value.(type) {
			case *ast.Call:
				c.compileNodes([]ast.Node{value})
			}
			return
		}
		var varReg int8
		if isDecl {
			varReg = c.fb.NewRegister(kind)
			c.fb.BindVarReg(variable.(*ast.Identifier).Name, varReg)
		} else {
			varReg = c.fb.ScopeLookup(variable.(*ast.Identifier).Name)
		}
		out, isValue, isRegister := c.quickCompileExpr(value)
		if isValue {
			c.fb.Move(true, out, varReg, kind)
		} else if isRegister {
			c.fb.Move(false, out, varReg, kind)
		} else {
			c.compileExpr(value, varReg)
		}
		return
	}
	// len(variables) > 1
	// TODO (Gianluca): handle "_".
	switch value := value.(type) {
	case *ast.Call:
		varRegs := []int8{}
		for i := 0; i < len(variables); i++ {
			variable := variables[i]
			kind := c.typeinfo[variable].Type.Kind()
			var varReg int8
			if isDecl {
				varReg = c.fb.NewRegister(kind)
				c.fb.BindVarReg(variable.(*ast.Identifier).Name, varReg)
			} else {
				varReg = c.fb.ScopeLookup(variable.(*ast.Identifier).Name)
			}
			varRegs = append(varRegs, varReg)
		}
		retRegs, retKinds := c.compileCall(value)
		for i := range retRegs {
			c.fb.Move(false, retRegs[i], varRegs[i], retKinds[i])
		}
	case *ast.TypeAssertion:
		var dst int8
		if isDecl {
			kind := c.typeinfo[variables[0]].Type.Kind()
			dst = c.fb.NewRegister(kind)
			c.fb.BindVarReg(variables[0].(*ast.Identifier).Name, dst)
		} else {
			dst = c.fb.ScopeLookup(variables[0].(*ast.Identifier).Name)
		}
		var okReg int8
		if isDecl {
			okReg = c.fb.NewRegister(reflect.Int)
			c.fb.BindVarReg(variables[1].(*ast.Identifier).Name, okReg)
		} else {
			okReg = c.fb.ScopeLookup(variables[1].(*ast.Identifier).Name)
		}
		typ := value.Type.(*ast.Value).Val.(reflect.Type)
		kind := c.typeinfo[value.Expr].Type.Kind()
		expr := c.fb.NewRegister(kind)
		c.compileExpr(value.Expr, expr)
		c.fb.Assert(expr, typ, dst)
		c.fb.IfOk()
		c.fb.Move(true, 0, okReg, reflect.Int)
		c.fb.Move(true, 1, okReg, reflect.Int)
	default:
		panic("TODO: not implemented")
	}
}

// TODO (Gianluca): a builtin can be shadowed, but the compiler can't know it.
// Typechecker should flag *ast.Call nodes with a boolean indicating if it's a
// builtin.
func (c *Compiler) callBuiltin(call *ast.Call, reg int8) (ok bool) {
	if ident, ok := call.Func.(*ast.Identifier); ok {
		var i instruction
		switch ident.Name {
		case "len":
			typ := c.typeinfo[call.Args[0]].Type
			kind := typ.Kind()
			var a, b int8
			out, _, isRegister := c.quickCompileExpr(call.Args[0])
			if isRegister {
				b = out
			} else {
				arg := c.fb.NewRegister(kind)
				c.compileExpr(call.Args[0], arg)
				b = arg
			}
			switch typ {
			case reflect.TypeOf(""): // TODO (Gianluca): or should check for kind string?
				a = 0
			default:
				a = 1
			case reflect.TypeOf([]byte{}):
				a = 2
			}
			i = instruction{op: opLen, a: a, b: b, c: reg}
		// case "new":
		// 	typ := c.typeinfo[call.Args[0]].Type
		// 	t := c.currFb.Type(typ)
		// 	i = instruction{op: opNew, b: t, c: }
		case "print":
			// TODO (Gianluca): move argument to general
			arg := c.fb.NewRegister(reflect.Int)
			c.compileExpr(call.Args[0], arg)
			i = instruction{op: opPrint, a: arg}
		case "make":
			typ := call.Args[0].(*ast.Value).Val.(reflect.Type)
			regType := c.fb.Type(typ)
			switch typ.Kind() {
			case reflect.Map:
				var size int8
				var kSize bool
				out, isValue, isRegister := c.quickCompileExpr(call.Args[1])
				if isValue {
					kSize = true
					size = out
				} else if isRegister {
					size = out
				} else {
					size = c.fb.NewRegister(reflect.Int)
					c.compileExpr(call.Args[1], size)
				}
				c.fb.MakeMap(regType, kSize, size, reg)
			default:
				panic("TODO: not implemented")
			}
		default:
			return false
		}
		c.fb.fn.body = append(c.fb.fn.body, i)
		return true
	}
	return false
}

// compileNodes compiles nodes.
func (c *Compiler) compileNodes(nodes []ast.Node) {
	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.Assignment:
			// TODO (Gianluca): clean up.
			switch {
			case len(node.Variables) == 1 &&
				len(node.Values) == 1 &&
				(node.Type == ast.AssignmentAddition || node.Type == ast.AssignmentSubtraction):
				switch node.Type {
				case ast.AssignmentAddition:
					panic("TODO: not implemented")
					// TODO (Gianluca): this is wrong:
					// name := node.Variables[0].(*ast.Identifier).Name
					// reg := c.fb.ScopeLookup(name)
					// kind := c.typeinfo[node.Variables[0]].Type.Kind()
					// c.fb.Add(true, reg, 1, reg, kind)
				case ast.AssignmentSubtraction:
					panic("TODO: not implemented")
					// TODO (Gianluca): this is wrong:
					// name := node.Variables[0].(*ast.Identifier).Name
					// reg := c.fb.ScopeLookup(name)
					// kind := c.typeinfo[node.Variables[0]].Type.Kind()
					// c.fb.Add(true, reg, -1, reg, kind)
				}
			case len(node.Variables) == len(node.Values):
				for i := range node.Variables {
					c.compileVarsGetValue([]ast.Expression{node.Variables[i]}, node.Values[i], node.Type == ast.AssignmentDeclaration)
				}
			case len(node.Variables) > 1 && len(node.Values) == 1:
				c.compileVarsGetValue(node.Variables, node.Values[0], node.Type == ast.AssignmentDeclaration)
			case len(node.Variables) == 1 && len(node.Values) == 0:
				switch node.Type {
				case ast.AssignmentIncrement:
					name := node.Variables[0].(*ast.Identifier).Name
					reg := c.fb.ScopeLookup(name)
					kind := c.typeinfo[node.Variables[0]].Type.Kind()
					c.fb.Add(true, reg, 1, reg, kind)
				case ast.AssignmentDecrement:
					panic("TODO: not implemented")
				}
			default:
				panic("TODO: not implemented")
			}

		case *ast.Block:
			c.fb.EnterScope()
			c.compileNodes(node.Nodes)
			c.fb.ExitScope()

		case *ast.If:
			c.fb.EnterScope()
			if node.Assignment != nil {
				c.compileNodes([]ast.Node{node.Assignment})
			}
			x, y, kind, o, ky := c.compileCondition(node.Condition)
			c.fb.If(ky, x, o, y, kind)
			if node.Else == nil { // TODO (Gianluca): can "then" and "else" be unified in some way?
				endIfLabel := c.fb.NewLabel()
				c.fb.Goto(endIfLabel)
				c.compileNodes(node.Then.Nodes)
				c.fb.SetLabelAddr(endIfLabel)
			} else {
				elseLabel := c.fb.NewLabel()
				c.fb.Goto(elseLabel)
				c.compileNodes(node.Then.Nodes)
				endIfLabel := c.fb.NewLabel()
				c.fb.Goto(endIfLabel)
				c.fb.SetLabelAddr(elseLabel)
				if node.Else != nil {
					switch els := node.Else.(type) {
					case *ast.If:
						c.compileNodes([]ast.Node{els})
					case *ast.Block:
						c.compileNodes(els.Nodes)
					}
				}
				c.fb.SetLabelAddr(endIfLabel)
			}
			c.fb.ExitScope()

		case *ast.For:
			c.fb.EnterScope()
			if node.Init != nil {
				c.compileNodes([]ast.Node{node.Init})
			}
			if node.Condition != nil {
				forLabel := c.fb.NewLabel()
				c.fb.SetLabelAddr(forLabel)
				x, y, kind, o, ky := c.compileCondition(node.Condition)
				c.fb.If(ky, x, o, y, kind)
				endForLabel := c.fb.NewLabel()
				c.fb.Goto(endForLabel)
				if node.Post != nil {
					c.compileNodes([]ast.Node{node.Post})
				}
				c.compileNodes(node.Body)
				c.fb.Goto(forLabel)
				c.fb.SetLabelAddr(endForLabel)
			} else {
				forLabel := c.fb.NewLabel()
				c.fb.SetLabelAddr(forLabel)
				if node.Post != nil {
					c.compileNodes([]ast.Node{node.Post})
				}
				c.compileNodes(node.Body)
				c.fb.Goto(forLabel)
			}
			c.fb.ExitScope()

		case *ast.ForRange:
			panic("TODO: not implemented")
			// c.fb.EnterScope()
			// expr := c.fb.NewRegister(reflect.String)
			// kind := c.typeinfo[node.Assignment.Values[0]].Type.Kind()
			// c.compileExpr(node.Assignment.Values[0], expr)
			// c.fb.ForRange(expr, kind)
			// c.compileNodes(node.Body)
			// c.fb.ExitScope()

		case *ast.Return:
			if len(node.Values) == 1 {
				if _, isCall := node.Values[0].(*ast.Call); isCall {
					// TODO (Gianluca): must assign new values.
					// TODO (Gianluca): use the appropiate function, cause not necessarily is CurrentPackage, CurrentFunction.
					c.fb.TailCall(CurrentPackage, CurrentFunction)
					continue
				}
			}
			for i, v := range node.Values {
				kind := c.typeinfo[v].Type.Kind()
				reg := int8(i + 1)
				c.fb.allocRegister(kind, reg)
				c.compileExpr(v, reg)
			}
			c.fb.Return()

		case *ast.Switch:
			c.compileSwitch(node)

		case *ast.Var:
			for i := range node.Identifiers {
				c.compileVarsGetValue([]ast.Expression{node.Identifiers[i]}, node.Values[i], true)
			}

		case ast.Expression:
			// TODO (Gianluca): use 0 (which is no longer a valid
			// register) and handle it as a special case in compileExpr.
			c.compileExpr(node, 0)

		}
	}
}

// compileSwitch compiles switch node.
func (c *Compiler) compileSwitch(node *ast.Switch) {

	kind := c.typeinfo[node.Expr].Type.Kind()
	expr := c.fb.NewRegister(kind)
	c.compileExpr(node.Expr, expr)
	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := c.fb.NewLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = c.fb.NewLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			var ky bool
			var y int8
			out, isValue, isRegister := c.quickCompileExpr(caseExpr)
			if isValue {
				ky = true
				y = out
			} else if isRegister {
				y = out
			} else {
				c.fb.allocRegister(kind, y)
				c.compileExpr(caseExpr, y)
			}
			c.fb.If(ky, expr, ConditionNotEqual, y, kind) // Condizione negata per poter concatenare gli if
			c.fb.Goto(bodyLabels[i])
		}
	}

	if hasDefault {
		defaultLabel = c.fb.NewLabel()
		c.fb.Goto(defaultLabel)
	} else {
		c.fb.Goto(endSwitchLabel)
	}

	for i, cas := range node.Cases {
		if cas.Expressions == nil {
			c.fb.SetLabelAddr(defaultLabel)
		}
		c.fb.SetLabelAddr(bodyLabels[i])
		c.compileNodes(cas.Body)
		if !cas.Fallthrough {
			c.fb.Goto(endSwitchLabel)
		}
	}

	c.fb.SetLabelAddr(endSwitchLabel)
}

// compileCondition compiles expr using c.currFb. Returns the two values of the
// condition (x and y), a kind, the condition ad a boolean ky which indicates
// whether y is a constant value.
func (c *Compiler) compileCondition(expr ast.Expression) (x, y int8, kind reflect.Kind, o Condition, yk bool) {
	// 	ConditionEqual               x == y
	// 	ConditionNotEqual            x != y
	// 	ConditionLess                x <  y
	// 	ConditionLessOrEqual         x <= y
	// 	ConditionGreater             x >  y
	// 	ConditionGreaterOrEqual      x >= y
	// 	ConditionEqualLen            len(x) == y
	// 	ConditionNotEqualLen         len(x) != y
	// 	ConditionLessLen             len(x) <  y
	// 	ConditionLessOrEqualLen      len(x) <= y
	// 	ConditionGreaterLen          len(x) >  y
	// 	ConditionGreaterOrEqualLen   len(x) >= y
	// 	ConditionNil                 x == nil
	// 	ConditionNotNil              x != nil
	switch cond := expr.(type) {
	case *ast.BinaryOperator:
		kind = c.typeinfo[cond.Expr1].Type.Kind()
		var out int8
		var isValue, isRegister bool
		out, _, isRegister = c.quickCompileExpr(cond.Expr1)
		if isRegister {
			x = out
		} else {
			x = c.fb.NewRegister(kind)
			c.compileExpr(cond.Expr1, x)
		}
		if isNil(cond.Expr2) {
			switch cond.Operator() {
			case ast.OperatorEqual:
				o = ConditionNil
			case ast.OperatorNotEqual:
				o = ConditionNotNil
			}
		} else {
			out, isValue, isRegister = c.quickCompileExpr(cond.Expr2)
			if isValue {
				y = out
				yk = true
			} else if isRegister {
				y = out
			} else {
				y = c.fb.NewRegister(kind)
				c.compileExpr(cond.Expr2, y)
			}
			switch cond.Operator() {
			case ast.OperatorEqual:
				o = ConditionEqual
			case ast.OperatorGreater:
				o = ConditionGreater
			case ast.OperatorGreaterOrEqual:
				o = ConditionGreaterOrEqual
			case ast.OperatorLess:
				o = ConditionLess
			case ast.OperatorLessOrEqual:
				o = ConditionLessOrEqual
			case ast.OperatorNotEqual:
				o = ConditionNotEqual
			}
		}

	default:
		x := c.fb.NewRegister(kind)
		c.compileExpr(cond, x)
		o = ConditionEqual
		y = c.fb.MakeIntConstant(1) // TODO.
	}
	return x, y, kind, o, yk
}
