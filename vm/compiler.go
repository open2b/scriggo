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

// A Compiler compiles sources generating VM's packages.
type Compiler struct {
	parser           *parser.Parser
	currentFunction  *ScrigoFunction
	typeinfo         map[ast.Node]*parser.TypeInfo
	fb               *FunctionBuilder // current function builder.
	importableGoPkgs map[string]*parser.GoPackage
	indirectVars     map[*ast.Identifier]bool

	availableScrigoFunctions map[string]*ScrigoFunction
	availableNativeFunctions map[string]*NativeFunction
	availableVariables       map[string]variable

	// TODO (Gianluca): find better names.
	// TODO (Gianluca): do these maps have to have a *ScrigoFunction key or
	// can they be related to currentFunction in some way?
	assignedIndexesOfScrigoFunctions map[*ScrigoFunction]map[*ScrigoFunction]int8
	assignedIndexesOfNativeFunctions map[*ScrigoFunction]map[*NativeFunction]int8
	assignedIndexesOfVariables       map[*ScrigoFunction]map[variable]uint8

	isNativePkg map[string]bool
}

// NewCompiler returns a new compiler reading sources from r.
// Native (Go) packages are made available for importing.
func NewCompiler(r parser.Reader, packages map[string]*parser.GoPackage) *Compiler {
	c := &Compiler{
		importableGoPkgs: packages,
		indirectVars:     map[*ast.Identifier]bool{},

		availableScrigoFunctions: map[string]*ScrigoFunction{},
		availableNativeFunctions: map[string]*NativeFunction{},
		availableVariables:       map[string]variable{},

		assignedIndexesOfScrigoFunctions: map[*ScrigoFunction]map[*ScrigoFunction]int8{},
		assignedIndexesOfNativeFunctions: map[*ScrigoFunction]map[*NativeFunction]int8{},
		assignedIndexesOfVariables:       map[*ScrigoFunction]map[variable]uint8{},

		isNativePkg: map[string]bool{},
	}
	c.parser = parser.New(r, packages, true)
	return c
}

func (c *Compiler) CompilePackage(path string) (*ScrigoFunction, error) {
	tree, err := c.parser.Parse(path, ast.ContextNone)
	if err != nil {
		return nil, err
	}
	tci := c.parser.TypeCheckInfos()
	c.typeinfo = tci[path].TypeInfo
	c.indirectVars = tci[path].IndirectVars
	node := tree.Nodes[0].(*ast.Package)
	c.compilePackage(node)
	fun := c.currentFunction
	return fun, nil
}

func (c *Compiler) CompileScript(path string) (*ScrigoFunction, error) {
	tree, err := c.parser.Parse(path, ast.ContextNone)
	if err != nil {
		return nil, err
	}
	tci := c.parser.TypeCheckInfos()
	c.typeinfo = tci["main"].TypeInfo
	c.indirectVars = tci["main"].IndirectVars
	main := NewScrigoFunction("main", "main", reflect.FuncOf(nil, nil, false))
	c.currentFunction = main
	c.fb = c.currentFunction.Builder()
	c.fb.EnterScope()
	addExplicitReturn(tree)
	c.compileNodes(tree.Nodes)
	c.fb.ExitScope()
	fun := c.currentFunction
	return fun, nil
}

// scrigoFunctionIndex returns fun's index inside current function, creating it
// if not exists.
func (c *Compiler) scrigoFunctionIndex(fun *ScrigoFunction) int8 {
	currFun := c.currentFunction
	i, ok := c.assignedIndexesOfScrigoFunctions[currFun][fun]
	if ok {
		return i
	}
	i = int8(len(currFun.scrigoFunctions))
	currFun.scrigoFunctions = append(currFun.scrigoFunctions, fun)
	if c.assignedIndexesOfScrigoFunctions[currFun] == nil {
		c.assignedIndexesOfScrigoFunctions[currFun] = make(map[*ScrigoFunction]int8)
	}
	c.assignedIndexesOfScrigoFunctions[currFun][fun] = i
	return i
}

// nativeFunctionIndex returns fun's index inside current function, creating it
// if not exists.
func (c *Compiler) nativeFunctionIndex(fun *NativeFunction) int8 {
	currFun := c.currentFunction
	i, ok := c.assignedIndexesOfNativeFunctions[currFun][fun]
	if ok {
		return i
	}
	i = int8(len(currFun.nativeFunctions))
	currFun.nativeFunctions = append(currFun.nativeFunctions, fun)
	if c.assignedIndexesOfNativeFunctions[currFun] == nil {
		c.assignedIndexesOfNativeFunctions[currFun] = make(map[*NativeFunction]int8)
	}
	c.assignedIndexesOfNativeFunctions[currFun][fun] = i
	return i
}

// variableIndex returns v's index inside current function, creating it if not
// exists.
func (c *Compiler) variableIndex(v variable) uint8 {
	currFun := c.currentFunction
	i, ok := c.assignedIndexesOfVariables[currFun][v]
	if ok {
		return i
	}
	i = uint8(len(currFun.variables))
	currFun.variables = append(currFun.variables, v)
	if c.assignedIndexesOfVariables[currFun] == nil {
		c.assignedIndexesOfVariables[currFun] = make(map[variable]uint8)
	}
	c.assignedIndexesOfVariables[currFun][v] = i
	return i
}

// changeRegister moves src content into dst, making a conversion if necessary.
func (c *Compiler) changeRegister(k bool, src, dst int8, srcType reflect.Type, dstType reflect.Type) {
	if srcType.Kind() != reflect.Interface && dstType.Kind() == reflect.Interface {
		if k {
			c.fb.EnterStack()
			tmpReg := c.fb.NewRegister(srcType.Kind())
			c.fb.Move(true, src, tmpReg, srcType.Kind(), srcType.Kind())
			c.fb.Convert(tmpReg, srcType, srcType, dst)
			c.fb.ExitStack()
		} else {
			c.fb.Convert(src, srcType, srcType, dst)
		}
	} else {
		c.fb.Move(k, src, dst, srcType.Kind(), dstType.Kind())
	}
}

// compilePackage compiles pkg.
func (c *Compiler) compilePackage(pkg *ast.Package) {

	haveVariables := false
	for _, dec := range pkg.Declarations {
		_, ok := dec.(*ast.Var)
		if ok {
			haveVariables = true
			break
		}
	}

	var initVars *ScrigoFunction
	if haveVariables {
		initVars = NewScrigoFunction("main", "init.vars", reflect.FuncOf(nil, nil, false))
	}

	for _, dec := range pkg.Declarations {
		switch n := dec.(type) {
		case *ast.Var:
			if len(n.Identifiers) == 1 && len(n.Values) == 1 {
				backupFunction := c.currentFunction
				c.currentFunction = initVars
				c.fb = c.currentFunction.Builder()
				value := n.Values[0]
				typ := c.typeinfo[n.Identifiers[0]].Type
				kind := typ.Kind()
				reg := c.fb.NewRegister(kind)
				c.compileExpr(value, reg, typ)
				v := NewVariable("main", n.Identifiers[0].Name, nil)
				c.availableVariables[n.Identifiers[0].Name] = v
				index := c.variableIndex(v)
				c.fb.SetVar(reg, index)
				c.currentFunction = backupFunction
				c.fb = c.currentFunction.Builder()
			} else {
				panic("TODO(Gianluca): not implemented")
			}
			// TODO (Gianluca): this makes a new init function for every
			// variable, which is wrong. Putting initFn declaration
			// outside this switch is wrong too: init.-1 cannot be created
			// if there's no need.
			// initFn, _ := c.currentPkg.NewFunction("init.-1", reflect.FuncOf(nil, nil, false))
			// initBuilder := initFn.Builder()
			// if len(n.Identifiers) == 1 && len(n.Values) == 1 {
			// 	currentBuilder := c.fb
			// 	c.fb = initBuilder
			// 	reg := c.fb.NewRegister(reflect.Int)
			// 	c.compileExpr(n.Values[0], reg, reflect.Int)
			// 	c.fb = currentBuilder
			// 	name := "A"                 // TODO
			// 	v := interface{}(int64(10)) // TODO
			// 	c.currentPkg.DefineVariable(name, v)
			// } else {
			// 	panic("TODO: not implemented")
			// }
		case *ast.Func:
			fn := NewScrigoFunction("main", n.Ident.Name, n.Type.Reflect)
			c.availableScrigoFunctions[n.Ident.Name] = fn
			c.currentFunction = fn
			c.fb = fn.Builder()
			c.fb.EnterScope()
			c.prepareFunctionBodyParameters(n)
			addExplicitReturn(n)
			c.compileNodes(n.Body.Nodes)
			c.fb.End()
			c.fb.ExitScope()
		case *ast.Import:
			if n.Tree == nil { // Go package.
				var importPkgName string
				parserGoPkg := c.importableGoPkgs[n.Path]
				if n.Ident == nil {
					importPkgName = parserGoPkg.Name
				} else {
					switch n.Ident.Name {
					case "_":
						panic("TODO(Gianluca): not implemented")
					case ".":
						importPkgName = ""
					default:
						importPkgName = n.Ident.Name
					}
				}
				for ident, value := range parserGoPkg.Declarations {
					_ = ident
					if _, ok := value.(reflect.Type); ok {
						continue
					}
					if reflect.TypeOf(value).Kind() == reflect.Ptr {
						// pkg.DefineVariable(ident, value)
						// continue
						v := NewVariable(parserGoPkg.Name, ident, value)
						if importPkgName == "" {
							c.availableVariables[ident] = v
						} else {
							c.availableVariables[importPkgName+"."+ident] = v
						}
					}
					if reflect.TypeOf(value).Kind() == reflect.Func {
						nativeFunc := NewNativeFunction(parserGoPkg.Name, ident, value)
						// index, ok := pkg.AddNativeFunction(nativeFunc)
						// if !ok {
						// 	panic("TODO: not implemented")
						// }
						// pkg.nativeFunctionsNames[ident] = int8(index)
						// continue
						if importPkgName == "" {
							c.availableNativeFunctions[ident] = nativeFunc
						} else {
							c.availableNativeFunctions[importPkgName+"."+ident] = nativeFunc
						}
					}
				}
				c.isNativePkg[importPkgName] = true
			} else {
				panic("TODO(Gianluca): not implemented")
			}
		}
	}

	if haveVariables {
		b := initVars.Builder()
		b.Return()
	}
}

// prepareCallParameters prepares parameters (out and in) for a function call of
// type funcType and arguments args. Returns the list of return registers and
// their respective type.
func (c *Compiler) prepareCallParameters(funcType reflect.Type, args []ast.Expression, isNative bool) ([]int8, []reflect.Type) {
	numOut := funcType.NumOut()
	numIn := funcType.NumIn()
	regs := make([]int8, numOut)
	types := make([]reflect.Type, numOut)
	for i := 0; i < numOut; i++ {
		typ := funcType.Out(i)
		regs[i] = c.fb.NewRegister(typ.Kind())
		types[i] = typ
	}
	if funcType.IsVariadic() {
		for i := 0; i < numIn-1; i++ {
			typ := funcType.In(i)
			reg := c.fb.NewRegister(typ.Kind())
			c.compileExpr(args[i], reg, typ)
		}
		if varArgs := len(args) - (numIn - 1); varArgs > 0 {
			typ := funcType.In(numIn - 1).Elem()
			if isNative {
				for i := 0; i < varArgs; i++ {
					reg := c.fb.NewRegister(typ.Kind())
					c.compileExpr(args[i], reg, typ)
				}
			} else {
				sliceReg := int8(numIn)
				c.fb.MakeSlice(true, true, funcType.In(numIn-1), int8(varArgs), int8(varArgs), sliceReg)
				for i := 0; i < varArgs; i++ {
					tmpReg := c.fb.NewRegister(typ.Kind())
					c.compileExpr(args[i+numIn-1], tmpReg, typ)
					indexReg := c.fb.NewRegister(reflect.Int)
					c.fb.Move(true, int8(i), indexReg, reflect.Int, reflect.Int)
					c.fb.SetSlice(false, sliceReg, tmpReg, indexReg, typ.Kind())
				}
			}
		}
	} else { // No-variadic function.
		if numIn > 1 && len(args) == 1 { // f(g()), where f takes more than 1 argument.
			regs, types := c.compileCall(args[0].(*ast.Call))
			for i := range regs {
				dstKind := funcType.In(i).Kind()
				reg := c.fb.NewRegister(dstKind)
				c.fb.Move(false, regs[i], reg, types[i].Kind(), dstKind)
			}
		} else {
			for i := 0; i < numIn; i++ {
				typ := funcType.In(i)
				reg := c.fb.NewRegister(typ.Kind())
				c.compileExpr(args[i], reg, typ)
			}
		}
	}
	return regs, types
}

// prepareFunctionBodyParameters prepares fun's parameters (out and int) before
// compiling its body.
func (c *Compiler) prepareFunctionBodyParameters(fun *ast.Func) {
	// Reserves space for return parameters.
	fillParametersTypes(fun.Type.Result)
	for _, res := range fun.Type.Result {
		resType := res.Type.(*ast.Value).Val.(reflect.Type)
		kind := resType.Kind()
		retReg := c.fb.NewRegister(kind)
		if res.Ident != nil {
			c.fb.BindVarReg(res.Ident.Name, retReg)
		}
	}
	// Binds function argument names to pre-allocated registers.
	fillParametersTypes(fun.Type.Parameters)
	for _, par := range fun.Type.Parameters {
		parType := par.Type.(*ast.Value).Val.(reflect.Type)
		kind := parType.Kind()
		argReg := c.fb.NewRegister(kind)
		if par.Ident != nil {
			c.fb.BindVarReg(par.Ident.Name, argReg)
		}
	}
}

// compileCall compiles call, returning the list of registers (and their
// respective type) within which return values are inserted.
func (c *Compiler) compileCall(call *ast.Call) ([]int8, []reflect.Type) {
	stackShift := StackShift{
		int8(c.fb.numRegs[reflect.Int]),
		int8(c.fb.numRegs[reflect.Float64]),
		int8(c.fb.numRegs[reflect.String]),
		int8(c.fb.numRegs[reflect.Interface]),
	}
	if ident, ok := call.Func.(*ast.Identifier); ok {
		if !c.fb.IsVariable(ident.Name) {
			if fun, isScrigoFunc := c.availableScrigoFunctions[ident.Name]; isScrigoFunc {
				regs, types := c.prepareCallParameters(fun.typ, call.Args, false)
				index := c.scrigoFunctionIndex(fun)
				c.fb.Call(index, stackShift, call.Pos().Line)
				return regs, types
			}
			if _, isNativeFunc := c.availableNativeFunctions[ident.Name]; isNativeFunc {
				fun := c.availableNativeFunctions[ident.Name]
				funcType := reflect.TypeOf(fun.fast)
				regs, types := c.prepareCallParameters(funcType, call.Args, true)
				index := c.nativeFunctionIndex(fun)
				if funcType.IsVariadic() {
					numVar := len(call.Args) - (funcType.NumIn() - 1)
					c.fb.CallNative(index, int8(numVar), stackShift)
				} else {
					c.fb.CallNative(index, NoVariadic, stackShift)
				}
				return regs, types
			}
		}
	}
	if sel, ok := call.Func.(*ast.Selector); ok {
		if name, ok := sel.Expr.(*ast.Identifier); ok {
			if isGoPkg := c.isNativePkg[name.Name]; isGoPkg {
				fun := c.availableNativeFunctions[name.Name+"."+sel.Ident]
				funcType := reflect.TypeOf(fun.fast)
				regs, types := c.prepareCallParameters(funcType, call.Args, true)
				index := c.nativeFunctionIndex(fun)
				if funcType.IsVariadic() {
					numVar := len(call.Args) - (funcType.NumIn() - 1)
					c.fb.CallNative(index, int8(numVar), stackShift)
				} else {
					c.fb.CallNative(index, NoVariadic, stackShift)
				}
				return regs, types
			} else {
				panic("TODO(Gianluca): not implemented")
			}
		}
	}
	funReg, _, isRegister := c.quickCompileExpr(call.Func, c.typeinfo[call.Func].Type)
	if !isRegister {
		funReg = c.fb.NewRegister(reflect.Func)
		c.compileExpr(call.Func, funReg, c.typeinfo[call.Func].Type)
	}
	funcType := c.typeinfo[call.Func].Type
	regs, types := c.prepareCallParameters(funcType, call.Args, true)
	c.fb.CallIndirect(funReg, 0, stackShift)
	return regs, types
}

// compileExpr compiles expression expr and puts results into reg.
// Result is discarded if reg is 0.
func (c *Compiler) compileExpr(expr ast.Expression, reg int8, dstType reflect.Type) {
	// TODO (Gianluca): review all "kind" arguments in every compileExpr call.
	// TODO (Gianluca): use "tmpReg" instead "reg" and move evaluated value to reg only if reg != 0.
	switch expr := expr.(type) {

	case *ast.BinaryOperator:

		// Binary && and ||.
		if op := expr.Operator(); op == ast.OperatorAndAnd || op == ast.OperatorOrOr {
			cmp := int8(0)
			if op == ast.OperatorAndAnd {
				cmp = 1
			}
			c.fb.EnterStack()
			c.compileExpr(expr.Expr1, reg, dstType)
			endIf := c.fb.NewLabel()
			c.fb.If(true, reg, ConditionEqual, cmp, reflect.Int)
			c.fb.Goto(endIf)
			c.compileExpr(expr.Expr2, reg, dstType)
			c.fb.SetLabelAddr(endIf)
			c.fb.ExitStack()
			return
		}

		c.fb.EnterStack()

		xType := c.typeinfo[expr.Expr1].Type
		x := c.fb.NewRegister(xType.Kind())
		c.compileExpr(expr.Expr1, x, xType)

		y, ky, isRegister := c.quickCompileExpr(expr.Expr2, xType)
		if !ky && !isRegister {
			y = c.fb.NewRegister(xType.Kind())
			c.compileExpr(expr.Expr2, y, xType)
		}

		res := c.fb.NewRegister(xType.Kind())

		switch op := expr.Operator(); {
		case op == ast.OperatorAddition && xType.Kind() == reflect.String && reg != 0:
			if ky {
				y = c.fb.NewRegister(reflect.String)
				c.compileExpr(expr.Expr2, y, xType)
			}
			c.fb.Concat(x, y, reg)
		case op == ast.OperatorAddition && reg != 0:
			c.fb.Add(ky, x, y, res, xType.Kind())
			c.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorSubtraction && reg != 0:
			c.fb.Sub(ky, x, y, res, xType.Kind())
			c.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorMultiplication && reg != 0:
			c.fb.Mul(ky, x, y, res, xType.Kind())
			c.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorDivision && reg != 0:
			c.fb.Div(ky, x, y, res, xType.Kind())
			c.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorDivision && reg == 0:
			dummyReg := c.fb.NewRegister(xType.Kind())
			c.fb.Div(ky, x, y, dummyReg, xType.Kind()) // produces division by zero.
		case op == ast.OperatorModulo && reg != 0:
			c.fb.Rem(ky, x, y, res, xType.Kind())
			c.changeRegister(false, res, reg, xType, dstType)
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
			if reg != 0 {
				c.fb.Move(true, 1, reg, xType.Kind(), dstType.Kind())
				c.fb.If(ky, x, cond, y, xType.Kind())
				c.fb.Move(true, 0, reg, xType.Kind(), dstType.Kind())
			}
		case op == ast.OperatorOr,
			op == ast.OperatorAnd,
			op == ast.OperatorXor,
			op == ast.OperatorAndNot:
			if reg != 0 {
				c.fb.BinaryBitOperation(op, ky, x, y, reg)
				if kindToType(xType.Kind()) != kindToType(dstType.Kind()) {
					c.changeRegister(ky, reg, reg, xType, dstType)
				}
			}
		default:
			panic(fmt.Errorf("TODO: not implemented operator %s", expr.Operator()))
		}

		c.fb.ExitStack()

	case *ast.Call:
		// Builtin call.
		c.fb.EnterStack()
		if c.typeinfo[expr.Func].IsBuiltin() {
			c.compileBuiltin(expr, reg, dstType)
			c.fb.ExitStack()
			return
		}
		// Conversion.
		if val, ok := expr.Func.(*ast.Value); ok {
			if convertType, ok := val.Val.(reflect.Type); ok {
				typ := c.typeinfo[expr.Args[0]].Type
				arg := c.fb.NewRegister(typ.Kind())
				c.compileExpr(expr.Args[0], arg, typ)
				c.fb.Convert(arg, typ, convertType, reg)
				c.fb.ExitStack()
				return
			}
		}
		regs, types := c.compileCall(expr)
		if reg != 0 {
			c.fb.Move(false, regs[0], reg, types[0].Kind(), dstType.Kind())
		}
		c.fb.ExitStack()

	case *ast.CompositeLiteral:
		typ := expr.Type.(*ast.Value).Val.(reflect.Type)
		switch typ.Kind() {
		case reflect.Slice, reflect.Array:
			size := int8(compositeLiteralLen(expr))
			// TODO(Gianluca): incorrect when reg is 0: slice is not
			// created, but values must be evaluated anyway.
			if reg != 0 {
				if typ.Kind() == reflect.Array {
					typ = reflect.SliceOf(typ.Elem())
				}
				c.fb.MakeSlice(true, true, typ, size, size, reg)
			}
			var index int8 = -1
			for _, kv := range expr.KeyValues {
				if kv.Key != nil {
					index = int8(kv.Key.(*ast.Value).Val.(int))
				} else {
					index++
				}
				c.fb.EnterStack()
				indexReg := c.fb.NewRegister(reflect.Int)
				c.fb.Move(true, index, indexReg, reflect.Int, reflect.Int)
				value, kvalue, isRegister := c.quickCompileExpr(kv.Value, typ.Elem())
				if !kvalue && !isRegister {
					value = c.fb.NewRegister(typ.Elem().Kind())
					c.compileExpr(kv.Value, value, typ.Elem())
				}
				if reg != 0 {
					c.fb.SetSlice(kvalue, reg, value, indexReg, typ.Elem().Kind())
				}
				c.fb.ExitStack()
			}
		case reflect.Struct:
			panic("TODO: not implemented")
		case reflect.Map:
			// TODO (Gianluca): handle maps with bigger size.
			size := len(expr.KeyValues)
			sizeReg := c.fb.MakeIntConstant(int64(size))
			regType := c.fb.Type(typ)
			c.fb.MakeMap(regType, true, sizeReg, reg)
			for _, kv := range expr.KeyValues {
				keyReg := c.fb.NewRegister(typ.Key().Kind())
				valueReg := c.fb.NewRegister(typ.Elem().Kind())
				c.fb.EnterStack()
				c.compileExpr(kv.Key, keyReg, typ.Key())
				kValue := false // TODO(Gianluca).
				c.compileExpr(kv.Value, valueReg, typ.Elem())
				c.fb.ExitStack()
				c.fb.SetMap(kValue, reg, valueReg, keyReg)
			}
		}

	case *ast.TypeAssertion:
		typ := c.typeinfo[expr.Expr].Type
		exprReg, _, isRegister := c.quickCompileExpr(expr.Expr, typ)
		if !isRegister {
			exprReg = c.fb.NewRegister(typ.Kind())
			c.compileExpr(expr.Expr, exprReg, typ)
		}
		assertType := expr.Type.(*ast.Value).Val.(reflect.Type)
		c.fb.Assert(exprReg, assertType, reg)
		c.fb.Nop()

	case *ast.Selector:
		if v, ok := c.availableVariables[expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := c.variableIndex(v)
			c.fb.GetVar(index, reg) // TODO (Gianluca): to review.
			return
		}
		if nf, ok := c.availableNativeFunctions[expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := c.nativeFunctionIndex(nf)
			c.fb.GetFunc(true, index, reg)
			return
		}
		if sf, ok := c.availableScrigoFunctions[expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := c.scrigoFunctionIndex(sf)
			c.fb.GetFunc(false, index, reg)
			return
		}
		exprType := c.typeinfo[expr.Expr].Type
		exprReg := c.fb.NewRegister(exprType.Kind())
		c.compileExpr(expr.Expr, exprReg, exprType)
		field := int8(0) // TODO(Gianluca).
		c.fb.Selector(exprReg, field, reg)

	case *ast.UnaryOperator:
		typ := c.typeinfo[expr.Expr].Type
		var tmpReg int8
		if reg != 0 {
			tmpReg = c.fb.NewRegister(typ.Kind())
		}
		c.compileExpr(expr.Expr, tmpReg, typ)
		if reg == 0 {
			return
		}
		switch expr.Operator() {
		case ast.OperatorNot:
			c.fb.SubInv(true, tmpReg, int8(1), tmpReg, reflect.Int)
			if reg != 0 {
				c.fb.Move(false, tmpReg, reg, typ.Kind(), dstType.Kind())
			}
		case ast.OperatorAnd:
			panic("TODO: not implemented")
		case ast.OperatorAddition:
			// Do nothing.
		case ast.OperatorSubtraction:
			c.fb.SubInv(true, tmpReg, 0, tmpReg, dstType.Kind())
			if reg != 0 {
				c.fb.Move(false, tmpReg, reg, typ.Kind(), dstType.Kind())
			}
		case ast.OperatorMultiplication:
			panic("TODO: not implemented")
		case ast.OperatorReceive:
			c.fb.Receive(tmpReg, 0, reg)
		default:
			panic(fmt.Errorf("TODO: not implemented operator %s", expr.Operator()))
		}

	case *ast.Func:
		if reg == 0 {
			return
		}
		typ := c.typeinfo[expr].Type
		out, isValue, isRegister := c.quickCompileExpr(expr, typ)
		if isValue {
			c.fb.Move(true, out, reg, typ.Kind(), dstType.Kind())
		} else if isRegister {
			c.fb.Move(false, out, reg, typ.Kind(), dstType.Kind())
		} else {
			panic("bug")
		}

	case *ast.Identifier:
		if reg == 0 {
			return
		}
		typ := c.typeinfo[expr].Type
		out, isValue, isRegister := c.quickCompileExpr(expr, typ)
		if isValue {
			c.fb.Move(true, out, reg, typ.Kind(), dstType.Kind())
		} else if isRegister {
			c.changeRegister(false, out, reg, typ, dstType)
		} else {
			if fun, isScrigoFunc := c.availableScrigoFunctions[expr.Name]; isScrigoFunc {
				index := c.scrigoFunctionIndex(fun)
				c.fb.GetFunc(false, index, reg)
			} else {
				panic("bug")
			}
		}

	case *ast.Value:
		if reg == 0 {
			return
		}
		typ := c.typeinfo[expr].Type
		out, isValue, isRegister := c.quickCompileExpr(expr, typ)
		if isValue {
			c.changeRegister(true, out, reg, typ, dstType)
		} else if isRegister {
			c.changeRegister(false, out, reg, typ, dstType)
		} else {
			// TODO(Gianluca): this switch only handles predeclared types.
			// Add support for defined types.
			switch v := expr.Val.(type) {
			case complex64, complex128:
				panic("TODO(Gianluca): not implemented")
			case uintptr:
				panic("TODO(Gianluca): not implemented")
			case int:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(TypeInt, constant, reg)
			case int8:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(TypeInt, constant, reg)
			case int16:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(TypeInt, constant, reg)
			case int32:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(TypeInt, constant, reg)
			case int64:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(TypeInt, constant, reg)
			case uint:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(TypeInt, constant, reg)
			case uint8:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(TypeInt, constant, reg)
			case uint16:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(TypeInt, constant, reg)
			case uint32:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(TypeInt, constant, reg)
			case uint64:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(TypeInt, constant, reg)
			case string:
				constant := c.fb.MakeStringConstant(v)
				c.changeRegister(true, constant, reg, typ, dstType)
			case float32:
				constant := c.fb.MakeFloatConstant(float64(v))
				c.fb.LoadNumber(TypeFloat, constant, reg)
			case float64:
				constant := c.fb.MakeFloatConstant(v)
				c.fb.LoadNumber(TypeFloat, constant, reg)
			default:
				constant := c.fb.MakeGeneralConstant(v)
				c.changeRegister(true, constant, reg, typ, dstType)
			}
		}

	case *ast.Int: // TODO(Gianluca): obsolete: use *ast.Value instead.
		if reg == 0 {
			return
		}
		intType := c.typeinfo[expr].Type
		i, ki, isRegister := c.quickCompileExpr(expr, intType)
		if ki {
			c.changeRegister(true, i, reg, intType, dstType)
		} else if isRegister {
			c.changeRegister(false, i, reg, intType, dstType)
		} else {
			panic("TODO(Gianluca): not implemented")
		}

	case *ast.String: // TODO(Gianluca): obsolete: use *ast.Value instead.
		if reg == 0 {
			return
		}
		stringType := c.typeinfo[expr].Type
		i, ki, isRegister := c.quickCompileExpr(expr, stringType)
		if ki {
			c.changeRegister(true, i, reg, stringType, dstType)
		} else if isRegister {
			c.changeRegister(false, i, reg, stringType, dstType)
		} else {
			constant := c.fb.MakeStringConstant(expr.Text)
			c.changeRegister(true, constant, reg, stringType, dstType)
		}

	case *ast.Index:
		exprType := c.typeinfo[expr.Expr].Type
		var exprReg int8
		out, _, isRegister := c.quickCompileExpr(expr.Expr, exprType)
		if isRegister {
			exprReg = out
		} else {
			exprReg = c.fb.NewRegister(exprType.Kind())
		}
		out, isValue, isRegister := c.quickCompileExpr(expr.Index, intType)
		ki := false
		var i int8
		if isValue {
			ki = true
			i = out
		} else if isRegister {
			i = out
		} else {
			i = c.fb.NewRegister(reflect.Int)
			c.compileExpr(expr.Index, i, dstType)
		}
		c.fb.Index(ki, exprReg, i, reg, exprType)

	case *ast.Slicing:
		panic("TODO: not implemented")

	default:
		panic(fmt.Sprintf("compileExpr currently does not support %T nodes", expr))

	}

}

// quickCompileExpr checks if expr is k (which means immediate for integers and
// floats and constant for strings and generals) or a register, putting it into
// out. If it's neither of them, both k and isRegister are false and content of
// out is unspecified.
func (c *Compiler) quickCompileExpr(expr ast.Expression, expectedType reflect.Type) (out int8, k, isRegister bool) {
	// TODO (Gianluca): quickCompileExpr must evaluate only expression which does
	// not need extra registers for evaluation.

	// Src kind and dst kind are different, so a Move/Conversion is required.
	if kindToType(expectedType.Kind()) != kindToType(c.typeinfo[expr].Type.Kind()) {
		return 0, false, false
	}
	switch expr := expr.(type) {

	// TODO(Gianluca): strings and int must be put inside an *ast.Value
	// node by typechecker.
	case *ast.String:
		return 0, false, false
	case *ast.Int:
		return int8(expr.Value.Int64()), true, false

	case *ast.Identifier:
		if c.fb.IsVariable(expr.Name) {
			return c.fb.ScopeLookup(expr.Name), false, true
		}
		return 0, false, false
	case *ast.Value:
		switch v := expr.Val.(type) {
		case int:
			if -127 < v && v < 126 {
				return int8(v), true, false
			}
		case bool:
			b := int8(0)
			if expr.Val.(bool) {
				b = 1
			}
			return b, true, false
		}

	case *ast.Func:
		typ := c.typeinfo[expr].Type
		reg := c.fb.NewRegister(reflect.Func)
		scrigoFunc := c.fb.Func(reg, typ)
		funcLitBuilder := scrigoFunc.Builder()
		currentFb := c.fb
		c.fb = funcLitBuilder
		c.fb.EnterScope()
		c.prepareFunctionBodyParameters(expr)
		addExplicitReturn(expr)
		c.compileNodes(expr.Body.Nodes)
		c.fb.ExitScope()
		c.fb = currentFb
		return reg, false, true
	}
	return 0, false, false
}

// compileSingleAssignment assigns value to variable.
func (c *Compiler) compileSingleAssignment(variable ast.Expression, value int8, valueType reflect.Type, kvalue bool, isDecl bool) {
	if isBlankIdentifier(variable) {
		return
	}
	variableType := c.typeinfo[variable].Type
	switch variable := variable.(type) {
	case *ast.Selector:
		if v, ok := c.availableVariables[variable.Expr.(*ast.Identifier).Name+"."+variable.Ident]; ok {
			if kvalue {
				// TODO(Gianluca): is this correct?
				c.fb.Move(true, value, value, valueType.Kind(), valueType.Kind())
			}
			index := c.variableIndex(v)
			c.fb.SetVar(value, index)
			return
		}
		panic("TODO: not implemented")
	case *ast.Identifier:
		var varReg int8
		if isDecl {
			varReg = c.fb.NewRegister(variableType.Kind())
			c.fb.BindVarReg(variable.Name, varReg)
		} else {
			varReg = c.fb.ScopeLookup(variable.Name)
		}
		c.fb.Move(kvalue, value, varReg, valueType.Kind(), variableType.Kind())
	case *ast.Index:
		switch exprType := c.typeinfo[variable.Expr].Type; exprType.Kind() {
		case reflect.Slice:
			var slice int8
			slice, _, isRegister := c.quickCompileExpr(variable.Expr, c.typeinfo[variable.Expr].Type)
			if !isRegister {
				slice = c.fb.NewRegister(reflect.Interface)
				c.compileExpr(variable.Expr, slice, variableType)
			}
			index, _, isRegister := c.quickCompileExpr(variable.Index, intType)
			if !isRegister {
				index = c.fb.NewRegister(reflect.Int)
				c.compileExpr(variable.Index, index, intType)
			}
			c.fb.SetSlice(kvalue, slice, value, index, valueType.Kind())
		default:
			panic("TODO: not implemented")
		}
	default:
		panic("TODO: not implemented")
	}
}

// compileMultipleAssignment assign value to variables. If variables contains more than
// one variable, value must be a function call, a map indexing operation or a
// type assertion. These last two cases involve that variables contains 2
// elements.
func (c *Compiler) compileMultipleAssignment(variables []ast.Expression, value ast.Expression, isDecl bool) {
	// TODO (Gianluca): in case of variable declaration, if quickCompileExpr returns
	// a register use it instead of creating a new one.
	if len(variables) <= 1 {
		panic("bug")
	}
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
		retRegs, retTypes := c.compileCall(value)
		for i := range retRegs {
			c.changeRegister(false, retRegs[i], varRegs[i], retTypes[i], c.typeinfo[variables[i]].Type)
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
		assertType := value.Type.(*ast.Value).Val.(reflect.Type)
		typ := c.typeinfo[value.Expr].Type
		expr := c.fb.NewRegister(typ.Kind())
		c.compileExpr(value.Expr, expr, typ)
		c.fb.Assert(expr, assertType, dst)
		c.fb.Move(true, 0, okReg, reflect.Int, reflect.Int)
		c.fb.Move(true, 1, okReg, reflect.Int, reflect.Int)
	case *ast.UnaryOperator:
		if value.Operator() != ast.OperatorReceive {
			panic(fmt.Errorf("type-checking bug: unexpected operator %s in assignment", value.Operator())) // TODO(Gianluca): remove.
		}
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
		ch := c.fb.NewRegister(reflect.Chan)
		c.compileExpr(value.Expr, ch, c.typeinfo[value.Expr].Type)
		c.fb.Receive(ch, okReg, dst)
	default:
		panic("bug")
	}
}

// compileBuiltin compiles a builtin, writing result into reg necessary.
func (c *Compiler) compileBuiltin(call *ast.Call, reg int8, dstType reflect.Type) {
	switch call.Func.(*ast.Identifier).Name {
	case "append":
		panic("TODO: not implemented")
	case "cap":
		panic("TODO: not implemented")
	case "close":
		panic("TODO: not implemented")
	case "complex":
		panic("TODO: not implemented")
	case "copy":
		dst, _, isRegister := c.quickCompileExpr(call.Args[0], c.typeinfo[call.Args[0]].Type)
		if !isRegister {
			dst = c.fb.NewRegister(reflect.Slice)
			c.compileExpr(call.Args[0], dst, c.typeinfo[call.Args[0]].Type)
		}
		src, _, isRegister := c.quickCompileExpr(call.Args[1], c.typeinfo[call.Args[1]].Type)
		if !isRegister {
			src = c.fb.NewRegister(reflect.Slice)
			c.compileExpr(call.Args[0], src, c.typeinfo[call.Args[0]].Type)
		}
		c.fb.Copy(dst, src, reg)
	case "delete":
		mapExpr := call.Args[0]
		keyExpr := call.Args[1]
		mapType := c.typeinfo[mapExpr].Type
		keyType := c.typeinfo[keyExpr].Type
		mapp, _, isRegister := c.quickCompileExpr(mapExpr, c.typeinfo[mapExpr].Type)
		if !isRegister {
			mapp = c.fb.NewRegister(mapType.Kind())
			c.compileExpr(mapExpr, mapp, c.typeinfo[mapExpr].Type)
		}
		key, _, isRegister := c.quickCompileExpr(keyExpr, keyType)
		if !isRegister {
			key = c.fb.NewRegister(keyType.Kind())
			c.compileExpr(keyExpr, key, c.typeinfo[keyExpr].Type)
		}
		c.fb.Delete(mapp, key)
	case "imag":
		panic("TODO: not implemented")
	case "html": // TODO (Gianluca): to review.
		panic("TODO: not implemented")
	case "len":
		typ := c.typeinfo[call.Args[0]].Type
		s := c.fb.NewRegister(typ.Kind())
		c.compileExpr(call.Args[0], s, typ)
		c.fb.Len(s, reg, typ)
	case "make":
		typ := call.Args[0].(*ast.Value).Val.(reflect.Type)
		regType := c.fb.Type(typ)
		switch typ.Kind() {
		case reflect.Map:
			size, kSize, isRegister := c.quickCompileExpr(call.Args[1], intType)
			if !kSize && !isRegister {
				size = c.fb.NewRegister(reflect.Int)
				c.compileExpr(call.Args[1], size, c.typeinfo[call.Args[1]].Type)
			}
			c.fb.MakeMap(regType, kSize, size, reg)
		case reflect.Slice:
			lenExpr := call.Args[1]
			capExpr := call.Args[2]
			len, kLen, isRegister := c.quickCompileExpr(lenExpr, intType)
			if !kLen && !isRegister {
				len = c.fb.NewRegister(reflect.Int)
				c.compileExpr(lenExpr, len, c.typeinfo[lenExpr].Type)
			}
			cap, kCap, isRegister := c.quickCompileExpr(capExpr, intType)
			if !kCap && !isRegister {
				cap = c.fb.NewRegister(reflect.Int)
				c.compileExpr(capExpr, cap, c.typeinfo[capExpr].Type)
			}
			c.fb.MakeSlice(kLen, kCap, typ, len, cap, reg)
		case reflect.Chan:
			chanType := c.typeinfo[call.Args[0]].Type
			chanTypeIndex := c.currentFunction.AddType(chanType)
			kCapacity := false
			var capacity int8
			if len(call.Args) == 1 {
				capacity = 0
				kCapacity = true
			} else {
				var isRegister bool
				capacity, kCapacity, isRegister = c.quickCompileExpr(call.Args[1], intType)
				if !kCapacity && !isRegister {
					capacity = c.fb.NewRegister(reflect.Int)
					c.compileExpr(call.Args[1], capacity, intType)
				}
			}
			c.fb.MakeChan(int8(chanTypeIndex), kCapacity, capacity, reg)
		default:
			panic("bug")
		}
	case "new":
		panic("TODO: not implemented")
		// typ := c.typeinfo[call.Args[0]].Type
		// t := c.currFb.Type(typ)
		// i = instruction{op: opNew, b: t, c: }
	case "panic":
		arg := call.Args[0]
		reg, _, isRegister := c.quickCompileExpr(arg, emptyInterfaceType)
		if !isRegister {
			reg = c.fb.NewRegister(reflect.Interface)
			c.compileExpr(arg, reg, emptyInterfaceType)
		}
		c.fb.Panic(reg, call.Pos().Line)
	case "print":
		for i := range call.Args {
			arg := c.fb.NewRegister(reflect.Interface)
			c.compileExpr(call.Args[i], arg, emptyInterfaceType)
			c.fb.Print(arg)
		}
	case "println":
		panic("TODO: not implemented")
	case "real":
		panic("TODO: not implemented")
	case "recover":
		c.fb.Recover(reg)
	default:
		panic("unkown builtin") // TODO(Gianluca): remove.
	}
}

// compileNodes compiles nodes.
func (c *Compiler) compileNodes(nodes []ast.Node) {
	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.Assignment:
			switch {
			case len(node.Variables) == 1 && len(node.Values) == 1 && (node.Type == ast.AssignmentAddition || node.Type == ast.AssignmentSubtraction):
				ident := node.Variables[0].(*ast.Identifier)
				varReg := c.fb.ScopeLookup(ident.Name)
				typ := c.typeinfo[ident].Type
				y, ky, isRegister := c.quickCompileExpr(node.Values[0], typ)
				if !ky && !isRegister {
					c.compileExpr(node.Values[0], y, typ)
				}
				if node.Type == ast.AssignmentAddition {
					c.fb.Add(ky, varReg, y, varReg, typ.Kind())
				} else {
					c.fb.Sub(ky, varReg, y, varReg, typ.Kind())
				}
			case len(node.Variables) == len(node.Values):
				if mayHaveDepencencies(node.Variables, node.Values) {
					values := make([]int8, len(node.Values))
					valueTypes := make([]reflect.Type, len(node.Values))
					valueIsConst := make([]bool, len(node.Values)) // TODO(Gianluca): use me.
					for i := range node.Values {
						valueTypes[i] = c.typeinfo[node.Values[i]].Type
						values[i] = c.fb.NewRegister(valueTypes[i].Kind())
						c.compileExpr(node.Values[i], values[i], valueTypes[i])
					}
					for i := range node.Variables {
						c.compileSingleAssignment(node.Variables[i], values[i], valueTypes[i], valueIsConst[i], node.Type == ast.AssignmentDeclaration)
					}
				} else {
					for i := range node.Variables {
						valueType := c.typeinfo[node.Values[i]].Type
						value, kvalue, isRegister := c.quickCompileExpr(node.Values[i], valueType)
						if !kvalue && !isRegister {
							value = c.fb.NewRegister(valueType.Kind())
							c.compileExpr(node.Values[i], value, valueType)
						}
						c.compileSingleAssignment(node.Variables[i], value, valueType, kvalue, node.Type == ast.AssignmentDeclaration)
					}
				}
			case len(node.Variables) > 1 && len(node.Values) == 1:
				c.compileMultipleAssignment(node.Variables, node.Values[0], node.Type == ast.AssignmentDeclaration)
			case len(node.Variables) == 1 && len(node.Values) == 0:
				switch node.Type {
				case ast.AssignmentIncrement:
					name := node.Variables[0].(*ast.Identifier).Name
					reg := c.fb.ScopeLookup(name)
					kind := c.typeinfo[node.Variables[0]].Type.Kind()
					c.fb.Add(true, reg, 1, reg, kind)
				case ast.AssignmentDecrement:
					name := node.Variables[0].(*ast.Identifier).Name
					reg := c.fb.ScopeLookup(name)
					kind := c.typeinfo[node.Variables[0]].Type.Kind()
					c.fb.Sub(true, reg, 1, reg, kind)
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

		case *ast.Defer, *ast.Go:
			if def, ok := node.(*ast.Defer); ok {
				if c.typeinfo[def.Call.Func].IsBuiltin() {
					ident := def.Call.Func.(*ast.Identifier)
					if ident.Name == "recover" {
						continue
					} else {
						// TODO(Gianluca): builtins (except recover)
						// must be incapsulated inside a function
						// literal call when deferring (or starting
						// a goroutine?). For example
						//
						//	defer copy(dst, src)
						//
						// should be compiled into
						//
						// 	defer func() {
						// 		copy(dst, src)
						// 	}()
						//
						panic("TODO(Gianluca): not implemented")
					}
				}
			}
			funReg := c.fb.NewRegister(reflect.Func)
			var funNode ast.Expression
			var args []ast.Expression
			switch node := node.(type) {
			case *ast.Defer:
				funNode = node.Call.Func
				args = node.Call.Args
			case *ast.Go:
				funNode = node.Call.Func
				args = node.Call.Args
			}
			funType := c.typeinfo[funNode].Type
			c.compileExpr(funNode, funReg, c.typeinfo[funNode].Type)
			offset := StackShift{
				int8(c.fb.numRegs[reflect.Int]),
				int8(c.fb.numRegs[reflect.Float64]),
				int8(c.fb.numRegs[reflect.String]),
				int8(c.fb.numRegs[reflect.Interface]),
			}
			// TODO(Gianluca): currently supports only deferring or
			// starting goroutines of Scrigo defined functions.
			isNative := false
			c.prepareCallParameters(funType, args, isNative)
			// TODO(Gianluca): currently supports only deferring functions
			// and starting goroutines with no arguments and no return
			// parameters.
			argsShift := StackShift{}
			switch node.(type) {
			case *ast.Defer:
				c.fb.Defer(funReg, NoVariadic, offset, argsShift)
			case *ast.Go:
				// TODO(Gianluca):
				c.fb.Go()
			}

		case *ast.If:
			c.fb.EnterScope()
			if node.Assignment != nil {
				c.compileNodes([]ast.Node{node.Assignment})
			}
			c.compileCondition(node.Condition)
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
				c.compileCondition(node.Condition)
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
			// c.fb.Range(expr, kind)
			// c.compileNodes(node.Body)
			// c.fb.ExitScope()

		case *ast.Return:
			// TODO(Gianluca): complete implementation of tail call optimization.
			// if len(node.Values) == 1 {
			// 	if call, ok := node.Values[0].(*ast.Call); ok {
			// 		tmpRegs := make([]int8, len(call.Args))
			// 		paramPosition := make([]int8, len(call.Args))
			// 		tmpTypes := make([]reflect.Type, len(call.Args))
			// 		shift := StackShift{}
			// 		for i := range call.Args {
			// 			tmpTypes[i] = c.typeinfo[call.Args[i]].Type
			// 			t := int(kindToType(tmpTypes[i].Kind()))
			// 			tmpRegs[i] = c.fb.NewRegister(tmpTypes[i].Kind())
			// 			shift[t]++
			// 			c.compileExpr(call.Args[i], tmpRegs[i], tmpTypes[i])
			// 			paramPosition[i] = shift[t]
			// 		}
			// 		for i := range call.Args {
			// 			c.changeRegister(false, tmpRegs[i], paramPosition[i], tmpTypes[i], c.typeinfo[call.Func].Type.In(i))
			// 		}
			// 		c.fb.TailCall(CurrentFunction, node.Pos().Line)
			// 		continue
			// 	}
			// }
			for i, v := range node.Values {
				typ := c.typeinfo[v].Type
				reg := int8(i + 1)
				c.fb.allocRegister(typ.Kind(), reg)
				c.compileExpr(v, reg, typ) // TODO (Gianluca): must be return parameter kind.
			}
			c.fb.Return()

		case *ast.Send:
			ch := c.fb.NewRegister(reflect.Chan)
			c.compileExpr(node.Channel, ch, c.typeinfo[node.Channel].Type)
			elemType := c.typeinfo[node.Value].Type
			v := c.fb.NewRegister(elemType.Kind())
			c.compileExpr(node.Value, v, elemType)
			c.fb.Send(ch, v)

		case *ast.Switch:
			c.compileSwitch(node)

		case *ast.TypeSwitch:
			c.compileTypeSwitch(node)

		case *ast.Var:
			if len(node.Identifiers) == len(node.Values) {
				for i := range node.Identifiers {
					valueType := c.typeinfo[node.Values[i]].Type
					value, kvalue, isRegister := c.quickCompileExpr(node.Values[i], valueType)
					if !kvalue && !isRegister {
						value = c.fb.NewRegister(valueType.Kind())
						c.compileExpr(node.Values[i], value, valueType)
					}
					c.compileSingleAssignment(node.Identifiers[i], value, valueType, kvalue, true)
				}
			} else {
				expr := make([]ast.Expression, len(node.Identifiers))
				for i := range node.Identifiers {
					expr[i] = node.Identifiers[i]
				}
				c.compileMultipleAssignment(expr, node.Values[0], true)
			}

		case ast.Expression:
			// TODO (Gianluca): use 0 (which is no longer a valid
			// register) and handle it as a special case in compileExpr.
			c.compileExpr(node, 0, reflect.Type(nil))

		}
	}
}

// compileTypeSwitch compiles type switch node.
func (c *Compiler) compileTypeSwitch(node *ast.TypeSwitch) {
	// TODO (Gianluca): a type-checker bug does not replace type switch type with proper value.

	if node.Init != nil {
		c.compileNodes([]ast.Node{node.Init})
	}

	typAss := node.Assignment.Values[0].(*ast.TypeAssertion)
	typ := c.typeinfo[typAss.Expr].Type
	expr := c.fb.NewRegister(typ.Kind())
	c.compileExpr(typAss.Expr, expr, typ)

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

	typ := c.typeinfo[node.Expr].Type
	expr := c.fb.NewRegister(typ.Kind())
	c.compileExpr(node.Expr, expr, typ)
	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := c.fb.NewLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = c.fb.NewLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			y, ky, isRegister := c.quickCompileExpr(caseExpr, typ)
			if !ky && !isRegister {
				// TODO(Gianluca): compiler should not call allocRegister..
				c.fb.allocRegister(typ.Kind(), y)
				c.compileExpr(caseExpr, y, typ)
			}
			c.fb.If(ky, expr, ConditionNotEqual, y, typ.Kind()) // Condizione negata per poter concatenare gli if
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

// compileCondition compiles a condition. Last instruction added by this method
// is always "If".
func (c *Compiler) compileCondition(cond ast.Expression) {

	switch cond := cond.(type) {

	case *ast.BinaryOperator:

		// if v   == nil
		// if v   != nil
		// if nil == v
		// if nil != v
		if isNil(cond.Expr1) != isNil(cond.Expr2) {
			expr := cond.Expr1
			if isNil(cond.Expr1) {
				expr = cond.Expr2
			}
			exprType := c.typeinfo[expr].Type
			x, _, isRegister := c.quickCompileExpr(expr, exprType)
			if !isRegister {
				x = c.fb.NewRegister(exprType.Kind())
				c.compileExpr(expr, x, exprType)
			}
			condType := ConditionNotNil
			if cond.Operator() == ast.OperatorEqual {
				condType = ConditionNil
			}
			c.fb.If(false, x, condType, 0, exprType.Kind())
			return
		}

		// if len("str") == v
		// if len("str") != v
		// if len("str") <  v
		// if len("str") <= v
		// if len("str") >  v
		// if len("str") >= v
		// if v == len("str")
		// if v != len("str")
		// if v <  len("str")
		// if v <= len("str")
		// if v >  len("str")
		// if v >= len("str")
		if c.isLenBuiltinCall(cond.Expr1) != c.isLenBuiltinCall(cond.Expr2) {
			var lenArg, expr ast.Expression
			if c.isLenBuiltinCall(cond.Expr1) {
				lenArg = cond.Expr1.(*ast.Call).Args[0]
				expr = cond.Expr2
			} else {
				lenArg = cond.Expr2.(*ast.Call).Args[0]
				expr = cond.Expr1
			}
			if c.typeinfo[lenArg].Type.Kind() == reflect.String { // len is optimized for strings only.
				lenArgType := c.typeinfo[lenArg].Type
				x, _, isRegister := c.quickCompileExpr(lenArg, lenArgType)
				if !isRegister {
					x = c.fb.NewRegister(lenArgType.Kind())
					c.compileExpr(lenArg, x, lenArgType)
				}
				exprType := c.typeinfo[expr].Type
				y, ky, isRegister := c.quickCompileExpr(expr, exprType)
				if !ky && !isRegister {
					y = c.fb.NewRegister(exprType.Kind())
					c.compileExpr(expr, y, exprType)
				}
				var condType Condition
				switch cond.Operator() {
				case ast.OperatorEqual:
					condType = ConditionEqualLen
				case ast.OperatorNotEqual:
					condType = ConditionNotEqualLen
				case ast.OperatorLess:
					condType = ConditionLessLen
				case ast.OperatorLessOrEqual:
					condType = ConditionLessOrEqualLen
				case ast.OperatorGreater:
					condType = ConditionGreaterLen
				case ast.OperatorGreaterOrEqual:
					condType = ConditionGreaterOrEqualLen
				}
				c.fb.If(ky, x, condType, y, reflect.String)
				return
			}
		}

		// if v1 == v2
		// if v1 != v2
		// if v1 <  v2
		// if v1 <= v2
		// if v1 >  v2
		// if v1 >= v2
		expr1Type := c.typeinfo[cond.Expr1].Type
		expr2Type := c.typeinfo[cond.Expr2].Type
		if expr1Type.Kind() == expr2Type.Kind() {
			switch kind := expr1Type.Kind(); kind {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64,
				reflect.String:
				expr1Type := c.typeinfo[cond.Expr1].Type
				x, _, isRegister := c.quickCompileExpr(cond.Expr1, expr1Type)
				if !isRegister {
					x = c.fb.NewRegister(expr1Type.Kind())
					c.compileExpr(cond.Expr1, x, expr1Type)
				}
				expr2Type := c.typeinfo[cond.Expr2].Type
				y, ky, isRegister := c.quickCompileExpr(cond.Expr2, expr2Type)
				if !ky && !isRegister {
					y = c.fb.NewRegister(expr2Type.Kind())
					c.compileExpr(cond.Expr2, y, expr2Type)
				}
				var condType Condition
				switch cond.Operator() {
				case ast.OperatorEqual:
					condType = ConditionEqual
				case ast.OperatorNotEqual:
					condType = ConditionNotEqual
				case ast.OperatorLess:
					condType = ConditionLess
				case ast.OperatorLessOrEqual:
					condType = ConditionLessOrEqual
				case ast.OperatorGreater:
					condType = ConditionGreater
				case ast.OperatorGreaterOrEqual:
					condType = ConditionGreaterOrEqual
				}
				if reflect.Uint <= kind && kind <= reflect.Uint64 {
					// Equality and not equality checks are not
					// optimized for uints.
					if condType == ConditionEqual || condType == ConditionNotEqual {
						kind = reflect.Int
					}
				}
				c.fb.If(ky, x, condType, y, kind)
				return
			}
		}

	default:

		condType := c.typeinfo[cond].Type
		x, _, isRegister := c.quickCompileExpr(cond, condType)
		if !isRegister {
			x = c.fb.NewRegister(condType.Kind())
			c.compileExpr(cond, x, condType)
		}
		yConst := c.fb.MakeIntConstant(1)
		y := c.fb.NewRegister(reflect.Bool)
		c.fb.LoadNumber(TypeInt, yConst, y)
		c.fb.If(false, x, ConditionEqual, y, reflect.Bool)

	}

}
