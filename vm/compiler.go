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
	c := &Compiler{importableGoPkgs: packages}
	c.parser = parser.New(r, packages, true)
	return c
}

// goPackageToVMPackage converts a Parser's GoPackage to a VM's Package.
func goPackageToVMPackage(goPkg *parser.GoPackage) *Package {
	pkg := NewPackage(goPkg.Name)
	for ident, value := range goPkg.Declarations {
		_ = ident
		if _, ok := value.(reflect.Type); ok {
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
			pkg.nativeFunctionsNames[ident] = int8(index)
			continue
		}
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

// compilePackage compiles pkg.
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
			fillParametersTypes(n.Type.Result)
			for _, res := range n.Type.Result {
				resType := res.Type.(*ast.Value).Val.(reflect.Type)
				kind := resType.Kind()
				retReg := c.fb.NewRegister(kind)
				_ = retReg // TODO (Gianluca): add support for named return parameters. Binding retReg to the name of the paramter should be enough.
			}
			fillParametersTypes(n.Type.Parameters)
			for _, par := range n.Type.Parameters {
				parType := par.Type.(*ast.Value).Val.(reflect.Type)
				kind := parType.Kind()
				argReg := c.fb.NewRegister(kind)
				c.fb.BindVarReg(par.Ident.Name, argReg)
			}
			c.currentPkg.scrigoFunctionsNames[n.Ident.Name] = index
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
		case reflect.Bool:
			v := int8(0)
			if expr.Val.(bool) {
				v = 1
			}
			return v, true, false
		case reflect.Int:
			n := expr.Val.(int)
			if n < 0 || n > 127 {
				c := c.fb.MakeIntConstant(int64(n))
				return c, false, true
			} else {
				return int8(n), true, false
			}
		case reflect.Int64:
			n := expr.Val.(int64)
			if n < 0 || n > 127 {
				c := c.fb.MakeIntConstant(int64(n))
				return c, false, true
			} else {
				return int8(n), true, false
			}
		case reflect.Float64:
			// TODO (Gianluca): handle all kind of floats.
			v := int8(expr.Val.(float64))
			return v, true, false
		case reflect.Slice, reflect.Map, reflect.Func:
			genConst := c.fb.MakeGeneralConstant(expr.Val)
			reg := c.fb.NewRegister(reflect.Interface)
			c.fb.Move(true, genConst, reg, reflect.Interface)
			return reg, false, true
		case reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
			reflect.Uintptr,
			reflect.Float32,
			reflect.Complex64,
			reflect.Complex128,
			reflect.Array,
			reflect.Chan,
			reflect.Interface,
			reflect.Ptr,
			reflect.Struct,
			reflect.UnsafePointer:
			panic(fmt.Sprintf("TODO: not implemented kind %q", kind))
		case reflect.String:
			sConst := c.fb.MakeStringConstant(expr.Val.(string))
			reg := c.fb.NewRegister(reflect.String)
			c.fb.Move(true, sConst, reg, reflect.String)
			return reg, false, true

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
	if ident, ok := call.Func.(*ast.Identifier); ok {
		if !c.fb.IsAVariable(ident.Name) {
			if i, isScrigoFunc := c.currentPkg.scrigoFunctionsNames[ident.Name]; isScrigoFunc {
				funcType := c.currentPkg.scrigoFunctions[i].typ
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
				c.fb.Call(CurrentPackage, i, stackShift, call.Pos().Line)
				return
			}
			if nativeFunc, isNativeFunc := c.currentPkg.nativeFunctionsNames[ident.Name]; isNativeFunc {
				_ = nativeFunc
				panic("TODO: calling native functions imported with '.' not implemented")
				return
			}
		}
	}
	if sel, ok := call.Func.(*ast.Selector); ok {
		if name, ok := sel.Expr.(*ast.Identifier); ok {
			pkgIndex, isPkg := c.currentPkg.packagesNames[name.Name]
			if isPkg {
				if isGoPkg := c.currentPkg.isGoPkg[name.Name]; isGoPkg {
					goPkg := c.currentPkg.packages[pkgIndex]
					i := goPkg.nativeFunctionsNames[sel.Ident]
					var funcType reflect.Type
					funcType = reflect.TypeOf(goPkg.nativeFunctions[i].fast)
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
					c.fb.CallFunc(int8(pkgIndex), i, NoVariadic, stackShift)
				} else {
					panic("TODO: calling scrigo functions from imported packages not implemented")
				}
			}
			return
		}
	}
	var funReg int8
	out, _, isRegister := c.quickCompileExpr(call.Func)
	if isRegister {
		funReg = out
	} else {
		funReg = c.fb.NewRegister(reflect.Func)
		c.compileExpr(call.Func, funReg)
	}
	funcType := c.typeinfo[call.Func].Type
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
	c.fb.CallIndirect(funReg, 0, stackShift)
	return regs, kinds
}

// compileExpr compiles expression expr and puts results into reg. Compiled
// result is discarded if reg is 0.
func (c *Compiler) compileExpr(expr ast.Expression, reg int8) {
	switch expr := expr.(type) {

	case *ast.BinaryOperator:
		if op := expr.Operator(); op == ast.OperatorAnd || op == ast.OperatorOr {
			cmp := int8(0)
			if op == ast.OperatorAnd {
				cmp = 1
			}
			c.compileExpr(expr.Expr1, reg)
			endIf := c.fb.NewLabel()
			c.fb.If(true, reg, ConditionEqual, cmp, reflect.Int)
			c.fb.Goto(endIf)
			c.compileExpr(expr.Expr2, reg)
			c.fb.SetLabelAddr(endIf)
			return
		}
		kind := c.typeinfo[expr.Expr1].Type.Kind()
		op1 := c.fb.NewRegister(kind)
		c.compileExpr(expr.Expr1, op1)
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
				op2 = c.fb.NewRegister(kind)
				c.compileExpr(expr.Expr2, op2)
			}
		}
		switch op := expr.Operator(); {
		case op == ast.OperatorAddition && kind == reflect.String:
			c.fb.Concat(op1, op2, reg)
		case op == ast.OperatorAddition:
			c.fb.Add(ky, op1, op2, reg, kind)
		case op == ast.OperatorSubtraction:
			c.fb.Sub(ky, op1, op2, reg, kind)
		case op == ast.OperatorMultiplication:
			c.fb.Mul(op1, op2, reg, kind)
		case op == ast.OperatorDivision:
			c.fb.Div(op1, op2, reg, kind)
		case op == ast.OperatorModulo:
			c.fb.Rem(op1, op2, reg, kind)
		case ast.OperatorEqual <= op && op <= ast.OperatorGreaterOrEqual:
			var cond Condition
			switch op {
			case ast.OperatorEqual:
				cond = ConditionEqual
			case ast.OperatorNotEqual:
				cond = ConditionNotEqual
			case ast.OperatorLess:
				cond = ConditionLess
			case ast.OperatorLessOrEqual:
				cond = ConditionLessOrEqual
			case ast.OperatorGreater:
				cond = ConditionGreater
			case ast.OperatorGreaterOrEqual:
				cond = ConditionGreaterOrEqual
			}
			c.fb.Move(true, 1, reg, kind)
			c.fb.If(ky, op1, cond, op2, kind)
			c.fb.Move(true, 0, reg, kind)
		}

	case *ast.Call:
		// Builtin call.
		ok := c.callBuiltin(expr, reg)
		if ok {
			return
		}
		// Conversion.
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
		// TODO (Gianluca): explicit key seems to be ignored when assigning
		// to slice.
		typ := expr.Type.(*ast.Value).Val.(reflect.Type)
		switch typ.Kind() {
		case reflect.Slice:
			size := int8(size(expr))
			c.fb.MakeSlice(true, true, typ, size, size, reg)
			var index int8 = -1
			for _, kv := range expr.KeyValues {
				if kv.Key != nil {
					index = int8(kv.Key.(*ast.Value).Val.(int))
				} else {
					index++
				}
				indexReg := c.fb.NewRegister(reflect.Int)
				c.fb.Move(true, index, indexReg, reflect.Int)
				var kvalue bool
				var value int8
				out, isValue, isRegister := c.quickCompileExpr(kv.Value)
				if isValue {
					value = out
					kvalue = true
				} else if isRegister {
					value = out
				} else {
					value = c.fb.NewRegister(typ.Elem().Kind())
					c.compileExpr(kv.Value, value)
				}
				c.fb.SetSlice(kvalue, reg, value, indexReg, typ.Elem().Kind())
			}
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

	case *ast.TypeAssertion:
		kind := c.typeinfo[expr.Expr].Type.Kind()
		out, _, isRegister := c.quickCompileExpr(expr.Expr)
		var exprReg int8
		if isRegister {
			exprReg = out
		} else {
			exprReg = c.fb.NewRegister(kind)
			c.compileExpr(expr.Expr, exprReg)
		}
		typ := expr.Type.(*ast.Value).Val.(reflect.Type)
		c.fb.Assert(exprReg, typ, reg)

	case *ast.Func:
		typ := c.typeinfo[expr].Type
		scrigoFunc := c.fb.Func(reg, typ)
		funcLitBuilder := scrigoFunc.Builder()
		currentFb := c.fb
		c.fb = funcLitBuilder
		c.fb.EnterScope()
		// Binds function argument names to pre-allocated registers.
		fillParametersTypes(expr.Type.Result)
		for _, res := range expr.Type.Result {
			resType := res.Type.(*ast.Value).Val.(reflect.Type)
			kind := resType.Kind()
			retReg := c.fb.NewRegister(kind)
			_ = retReg // TODO (Gianluca): add support for named return parameters. Binding retReg to the name of the paramter should be enough.
		}
		fillParametersTypes(expr.Type.Parameters)
		for _, par := range expr.Type.Parameters {
			parType := par.Type.(*ast.Value).Val.(reflect.Type)
			kind := parType.Kind()
			argReg := c.fb.NewRegister(kind)
			c.fb.BindVarReg(par.Ident.Name, argReg)
		}
		c.compileNodes(expr.Body.Nodes)
		c.fb.ExitScope()
		c.fb = currentFb

	case *ast.Selector:
		pkgName := expr.Expr.(*ast.Identifier).Name
		funcName := expr.Ident
		pkgIndex := int8(c.currentPkg.packagesNames[pkgName])
		goPkg := c.currentPkg.packages[pkgIndex]
		funcIndex := int8(goPkg.nativeFunctionsNames[funcName])
		c.fb.GetFunc(pkgIndex, funcIndex, reg)

	case *ast.UnaryOperator:
		c.compileExpr(expr.Expr, reg)
		switch expr.Operator() {
		case ast.OperatorNot:
			c.fb.SubInv(true, reg, int8(1), reg, reflect.Int)
		case ast.OperatorAmpersand:
			panic("TODO: not implemented")
		case ast.OperatorAddition:
			// Do nothing.
		case ast.OperatorSubtraction:
			kind := c.typeinfo[expr.Expr].Type.Kind()
			c.fb.SubInv(true, reg, 0, reg, kind)
		case ast.OperatorMultiplication:
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

	case *ast.Index:
		exprType := c.typeinfo[expr.Expr].Type
		var exprReg int8
		out, _, isRegister := c.quickCompileExpr(expr.Expr)
		if isRegister {
			exprReg = out
		} else {
			exprReg = c.fb.NewRegister(exprType.Kind())
		}
		out, isValue, isRegister := c.quickCompileExpr(expr.Index)
		ki := false
		var i int8
		if isValue {
			ki = true
			i = out
		} else if isRegister {
			i = out
		} else {
			i = c.fb.NewRegister(reflect.Int)
			c.compileExpr(expr.Index, i)
		}
		c.fb.Index(ki, exprReg, i, reg, exprType)

	case *ast.Slicing:
		panic("TODO: not implemented")

	case *ast.Rune, *ast.Float: // TODO (Gianluca): to review.
		panic("bug")

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
			// TODO (Gianluca): this is wrong. Consider
			//
			// 	_ = f1() + f2()
			//
			// both functions must be called.
			switch value.(type) {
			case *ast.Call:
				c.compileNodes([]ast.Node{value})
			}
			return
		}
		switch variable := variable.(type) {
		case *ast.Identifier:
			var varReg int8
			if isDecl {
				varReg = c.fb.NewRegister(kind)
				c.fb.BindVarReg(variable.Name, varReg)
			} else {
				varReg = c.fb.ScopeLookup(variable.Name)
			}
			out, isValue, isRegister := c.quickCompileExpr(value)
			if isValue {
				c.fb.Move(true, out, varReg, kind)
			} else if isRegister {
				c.fb.Move(false, out, varReg, kind)
			} else {
				tmpReg := c.fb.NewRegister(kind)
				c.compileExpr(value, tmpReg)
				c.fb.Move(false, tmpReg, varReg, kind)
			}
		case *ast.Index:
			switch c.typeinfo[variable.Expr].Type.Kind() {
			case reflect.Slice:
				var slice int8
				out, _, isRegister := c.quickCompileExpr(variable.Expr)
				if isRegister {
					slice = out
				} else {
					slice = c.fb.NewRegister(reflect.Interface)
					c.compileExpr(variable.Expr, slice)
				}
				var kvalue bool
				var valueReg int8
				out, isValue, isRegister := c.quickCompileExpr(value)
				if isValue {
					valueReg = out
					kvalue = true
				} else if isRegister {
					valueReg = out
				} else {
					valueReg = c.fb.NewRegister(kind)
					c.compileExpr(value, valueReg)
				}
				var index int8
				out, _, isRegister = c.quickCompileExpr(variable.Index)
				if isRegister {
					index = out
				} else {
					index = c.fb.NewRegister(reflect.Int)
					c.compileExpr(variable.Index, index)
				}
				c.fb.SetSlice(kvalue, slice, valueReg, index, kind)
			default:
				panic("TODO: not implemented")
			}
		default:
			panic("TODO: not implemented")
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
		panic("bug")
	}
}

// TODO (Gianluca): a builtin can be shadowed, but the compiler can't know it.
// Typechecker should flag *ast.Call nodes with a boolean indicating if it's a
// builtin.
func (c *Compiler) callBuiltin(call *ast.Call, reg int8) (ok bool) {
	if ident, ok := call.Func.(*ast.Identifier); ok {
		var i instruction
		switch ident.Name {
		case "append":
			panic("TODO: not implemented")
		case "cap":
			panic("TODO: not implemented")
		case "close":
			panic("TODO: not implemented")
		case "complex":
			panic("TODO: not implemented")
		case "copy":
			panic("TODO: not implemented")
		case "delete":
			mapExpr := call.Args[0]
			keyExpr := call.Args[1]
			mapType := c.typeinfo[mapExpr].Type
			keyType := c.typeinfo[keyExpr].Type
			var mapp, key int8
			out, _, isRegister := c.quickCompileExpr(mapExpr)
			if isRegister {
				mapp = out
			} else {
				mapp = c.fb.NewRegister(mapType.Kind())
				c.compileExpr(mapExpr, mapp)
			}
			out, _, isRegister = c.quickCompileExpr(keyExpr)
			if isRegister {
				key = out
			} else {
				key = c.fb.NewRegister(keyType.Kind())
				c.compileExpr(keyExpr, key)
			}
			c.fb.Delete(mapp, key)
		case "imag":
			panic("TODO: not implemented")
		case "html": // TODO (Gianluca): to review.
			panic("TODO: not implemented")
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
			case reflect.Slice:
				lenExpr := call.Args[1]
				capExpr := call.Args[2]
				var len, cap int8
				var kLen, kCap bool
				out, isValue, isRegister := c.quickCompileExpr(lenExpr)
				if isValue {
					len = out
					kLen = true
				} else if isRegister {
					len = out
				} else {
					len = c.fb.NewRegister(reflect.Int)
					c.compileExpr(lenExpr, len)
				}
				out, isValue, isRegister = c.quickCompileExpr(capExpr)
				if isValue {
					cap = out
					kCap = true
				} else if isRegister {
					cap = out
				} else {
					cap = c.fb.NewRegister(reflect.Int)
					c.compileExpr(capExpr, cap)
				}
				c.fb.MakeSlice(kLen, kCap, typ, len, cap, reg)
			case reflect.Chan:
				panic("TODO: not implemented")
			default:
				panic("bug")
			}
		case "new":
			panic("TODO: not implemented")
			// typ := c.typeinfo[call.Args[0]].Type
			// t := c.currFb.Type(typ)
			// i = instruction{op: opNew, b: t, c: }
		case "panic":
			// TODO (Gianluca): pass argument to panic.
			c.fb.Panic(0, call.Pos().Line)
		case "print":
			// TODO (Gianluca): move argument to general
			arg := c.fb.NewRegister(reflect.Int)
			c.compileExpr(call.Args[0], arg)
			i = instruction{op: opPrint, a: arg}
		case "println":
			panic("TODO: not implemented")
		case "real":
			panic("TODO: not implemented")
		case "recover":
			panic("TODO: not implemented")
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
			case len(node.Variables) == 1 && len(node.Values) == 1 && (node.Type == ast.AssignmentAddition || node.Type == ast.AssignmentSubtraction):
				ident := node.Variables[0].(*ast.Identifier)
				varReg := c.fb.ScopeLookup(ident.Name)
				kind := c.typeinfo[ident].Type.Kind()
				out, isValue, isRegister := c.quickCompileExpr(node.Values[0])
				var y int8
				var ky bool
				if isValue {
					y = out
					ky = true
				} else if isRegister {
					y = out
				} else {
					c.compileExpr(node.Values[0], y)
				}
				if node.Type == ast.AssignmentAddition {
					c.fb.Add(ky, varReg, y, varReg, kind)
				} else {
					c.fb.Sub(ky, varReg, y, varReg, kind)
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
				default:
					panic("bug")
				}
			default:
				panic("bug")
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
					c.fb.TailCall(CurrentPackage, CurrentFunction, node.Pos().Line)
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

		case *ast.TypeSwitch:
			c.compileTypeSwitch(node)

		case *ast.Var:
			if len(node.Identifiers) == len(node.Values) {
				for i := range node.Identifiers {
					c.compileVarsGetValue([]ast.Expression{node.Identifiers[i]}, node.Values[i], true)
				}
			} else {
				expr := make([]ast.Expression, len(node.Identifiers))
				for i := range node.Identifiers {
					expr[i] = node.Identifiers[i]
				}
				c.compileVarsGetValue(expr, node.Values[0], true)
			}

		case ast.Expression:
			// TODO (Gianluca): use 0 (which is no longer a valid
			// register) and handle it as a special case in compileExpr.
			c.compileExpr(node, 0)

		}
	}
}

// compileTypeSwitch compiles type switch node.
// TODO (Gianluca): a type-checker bug does not replace type switch type with proper value.
func (c *Compiler) compileTypeSwitch(node *ast.TypeSwitch) {

	if node.Init != nil {
		c.compileNodes([]ast.Node{node.Init})
	}

	typAss := node.Assignment.Values[0].(*ast.TypeAssertion)
	kind := c.typeinfo[typAss.Expr].Type.Kind()
	expr := c.fb.NewRegister(kind)
	c.compileExpr(typAss.Expr, expr)

	// typ := node.Assignment.Values[0].(*ast.Value).Val.(reflect.Type)
	// typReg := c.fb.Type(typ)
	// if variab := node.Assignment.Variables[0]; !isBlankIdentifier(variab) {
	// 	c.compileVarsGetValue([]ast.Expression{variab}, node.Assignment.Values[0], node.Assignment.Type == ast.AssignmentDeclaration)
	// }

	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := c.fb.NewLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = c.fb.NewLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			caseType := caseExpr.(*ast.Value).Val.(reflect.Type)
			c.fb.Assert(expr, caseType, 0)
			c.fb.IfOk()
			next := c.fb.NewLabel()
			c.fb.Goto(next)
			c.fb.Goto(bodyLabels[i])
			c.fb.SetLabelAddr(next)
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
		c.fb.Goto(endSwitchLabel)
	}

	c.fb.SetLabelAddr(endSwitchLabel)
}

// compileSwitch compiles switch node.
func (c *Compiler) compileSwitch(node *ast.Switch) {

	if node.Init != nil {
		c.compileNodes([]ast.Node{node.Init})
	}

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

// compileCondition compiles expr using the current function builder. Returns
// the two values of the condition (x and y), a kind, the condition ad a boolean
// ky which indicates whether y is a constant value.
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
