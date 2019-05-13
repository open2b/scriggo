// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"

	"scrigo/compiler/ast"
	"scrigo/vm"
)

// An Emitter emits instructions for the VM.
type Emitter struct {
	parser           *Parser
	currentFunction  *vm.ScrigoFunction
	typeinfo         map[ast.Node]*TypeInfo
	fb               *FunctionBuilder // current function builder.
	importableGoPkgs map[string]*GoPackage
	indirectVars     map[*ast.Identifier]bool

	availableScrigoFunctions map[string]*vm.ScrigoFunction
	availableNativeFunctions map[string]*vm.NativeFunction
	availableVariables       map[string]vm.Variable

	// TODO (Gianluca): find better names.
	// TODO (Gianluca): do these maps have to have a *vm.ScrigoFunction key or
	// can they be related to currentFunction in some way?
	assignedScrigoFunctions map[*vm.ScrigoFunction]map[*vm.ScrigoFunction]int8
	assignedNativeFunctions map[*vm.ScrigoFunction]map[*vm.NativeFunction]int8
	assignedVariables       map[*vm.ScrigoFunction]map[vm.Variable]uint8

	isNativePkg map[string]bool
}

// TODO(Gianluca): rename exported methods from "Compile" to "Emit".

// NewCompiler returns a new compiler reading sources from r.
// Native (Go) packages are made available for importing.
func NewCompiler(r Reader, packages map[string]*GoPackage) *Emitter {
	c := &Emitter{
		importableGoPkgs: packages,
		indirectVars:     map[*ast.Identifier]bool{},

		availableScrigoFunctions: map[string]*vm.ScrigoFunction{},
		availableNativeFunctions: map[string]*vm.NativeFunction{},
		availableVariables:       map[string]vm.Variable{},

		assignedScrigoFunctions: map[*vm.ScrigoFunction]map[*vm.ScrigoFunction]int8{},
		assignedNativeFunctions: map[*vm.ScrigoFunction]map[*vm.NativeFunction]int8{},
		assignedVariables:       map[*vm.ScrigoFunction]map[vm.Variable]uint8{},

		isNativePkg: map[string]bool{},
	}
	c.parser = New(r, packages, true)
	return c
}

func (c *Emitter) CompilePackage(path string) (*vm.ScrigoFunction, error) {
	tree, err := c.parser.Parse(path, ast.ContextNone)
	if err != nil {
		return nil, err
	}
	tci := c.parser.TypeCheckInfos()
	c.typeinfo = map[ast.Node]*TypeInfo{}
	for _, pkgInfo := range tci {
		for node, ti := range pkgInfo.TypeInfo {
			c.typeinfo[node] = ti
		}
	}
	// TODO(Gianluca): also add c.indirectVars of all packages, not just main.
	c.indirectVars = tci[path].IndirectVars
	node := tree.Nodes[0].(*ast.Package)
	c.emitPackage(node)
	fun := c.currentFunction
	return fun, nil
}

func (c *Emitter) CompileScript(path string) (*vm.ScrigoFunction, error) {
	tree, err := c.parser.Parse(path, ast.ContextNone)
	if err != nil {
		return nil, err
	}
	tci := c.parser.TypeCheckInfos()
	c.typeinfo = tci["main"].TypeInfo
	c.indirectVars = tci["main"].IndirectVars
	main := NewScrigoFunction("main", "main", reflect.FuncOf(nil, nil, false))
	c.currentFunction = main
	c.fb = NewBuilder(c.currentFunction)
	c.fb.EnterScope()
	addExplicitReturn(tree)
	c.emitNodes(tree.Nodes)
	c.fb.ExitScope()
	fun := c.currentFunction
	return fun, nil
}

// scrigoFunctionIndex returns fun's index inside current function, creating it
// if not exists.
func (c *Emitter) scrigoFunctionIndex(fun *vm.ScrigoFunction) int8 {
	currFun := c.currentFunction
	i, ok := c.assignedScrigoFunctions[currFun][fun]
	if ok {
		return i
	}
	i = int8(len(currFun.ScrigoFunctions))
	currFun.ScrigoFunctions = append(currFun.ScrigoFunctions, fun)
	if c.assignedScrigoFunctions[currFun] == nil {
		c.assignedScrigoFunctions[currFun] = make(map[*vm.ScrigoFunction]int8)
	}
	c.assignedScrigoFunctions[currFun][fun] = i
	return i
}

// nativeFunctionIndex returns fun's index inside current function, creating it
// if not exists.
func (c *Emitter) nativeFunctionIndex(fun *vm.NativeFunction) int8 {
	currFun := c.currentFunction
	i, ok := c.assignedNativeFunctions[currFun][fun]
	if ok {
		return i
	}
	i = int8(len(currFun.NativeFunctions))
	currFun.NativeFunctions = append(currFun.NativeFunctions, fun)
	if c.assignedNativeFunctions[currFun] == nil {
		c.assignedNativeFunctions[currFun] = make(map[*vm.NativeFunction]int8)
	}
	c.assignedNativeFunctions[currFun][fun] = i
	return i
}

// variableIndex returns v's index inside current function, creating it if not
// exists.
func (c *Emitter) variableIndex(v vm.Variable) uint8 {
	currFun := c.currentFunction
	i, ok := c.assignedVariables[currFun][v]
	if ok {
		return i
	}
	i = uint8(len(currFun.Variables))
	currFun.Variables = append(currFun.Variables, v)
	if c.assignedVariables[currFun] == nil {
		c.assignedVariables[currFun] = make(map[vm.Variable]uint8)
	}
	c.assignedVariables[currFun][v] = i
	return i
}

// changeRegister moves src content into dst, making a conversion if necessary.
func (c *Emitter) changeRegister(k bool, src, dst int8, srcType reflect.Type, dstType reflect.Type) {
	if kindToType(srcType.Kind()) != vm.TypeGeneral && dstType.Kind() == reflect.Interface {
		if k {
			c.fb.EnterStack()
			tmpReg := c.fb.NewRegister(srcType.Kind())
			c.fb.Move(true, src, tmpReg, srcType.Kind())
			c.fb.Convert(tmpReg, srcType, dst, srcType.Kind())
			c.fb.ExitStack()
		} else {
			c.fb.Convert(src, srcType, dst, srcType.Kind())
		}
	} else {
		c.fb.Move(k, src, dst, srcType.Kind())
	}
}

// emitPackage emits pkg.
func (c *Emitter) emitPackage(pkg *ast.Package) {

	haveVariables := false
	for _, dec := range pkg.Declarations {
		_, ok := dec.(*ast.Var)
		if ok {
			haveVariables = true
			break
		}
	}

	var initVars *vm.ScrigoFunction
	if haveVariables {
		initVars = NewScrigoFunction("main", "init.vars", reflect.FuncOf(nil, nil, false))
	}

	for _, dec := range pkg.Declarations {
		if fun, ok := dec.(*ast.Func); ok {
			fn := NewScrigoFunction("main", fun.Ident.Name, fun.Type.Reflect)
			c.availableScrigoFunctions[fun.Ident.Name] = fn
		}
	}

	for _, dec := range pkg.Declarations {
		switch n := dec.(type) {
		case *ast.Var:
			if len(n.Identifiers) == 1 && len(n.Values) == 1 {
				backupFunction := c.currentFunction
				c.currentFunction = initVars
				c.fb = NewBuilder(c.currentFunction)
				value := n.Values[0]
				typ := c.typeinfo[n.Identifiers[0]].Type
				kind := typ.Kind()
				reg := c.fb.NewRegister(kind)
				c.emitExpr(value, reg, typ)
				v := NewVariable("main", n.Identifiers[0].Name, nil)
				c.availableVariables[n.Identifiers[0].Name] = v
				index := c.variableIndex(v)
				c.fb.SetVar(reg, index)
				c.currentFunction = backupFunction
				c.fb = NewBuilder(c.currentFunction)
			} else {
				panic("TODO(Gianluca): not implemented")
			}
			// TODO (Gianluca): this makes a new init function for every
			// variable, which is wrong. Putting initFn declaration
			// outside this switch is wrong too: init.-1 cannot be created
			// if there's no need.
			// initFn, _ := c.currentPkg.NewFunction("init.-1", reflect.FuncOf(nil, nil, false))
			// initBuilder := NewBuilder(initFn)
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
			fn := c.availableScrigoFunctions[n.Ident.Name]
			c.currentFunction = fn
			c.fb = NewBuilder(fn)
			c.fb.EnterScope()
			c.prepareFunctionBodyParameters(n)
			addExplicitReturn(n)
			c.emitNodes(n.Body.Nodes)
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
				c.emitPackage(n.Tree.Nodes[0].(*ast.Package))
			}
		}
	}

	if haveVariables {
		b := NewBuilder(initVars)
		b.Return()
	}
}

// prepareCallParameters prepares parameters (out and in) for a function call of
// type funcType and arguments args. Returns the list of return registers and
// their respective type.
func (c *Emitter) prepareCallParameters(funcType reflect.Type, args []ast.Expression, isNative bool) ([]int8, []reflect.Type) {
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
			c.emitExpr(args[i], reg, typ)
		}
		if varArgs := len(args) - (numIn - 1); varArgs > 0 {
			typ := funcType.In(numIn - 1).Elem()
			if isNative {
				for i := 0; i < varArgs; i++ {
					reg := c.fb.NewRegister(typ.Kind())
					c.emitExpr(args[i+numIn-1], reg, typ)
				}
			} else {
				sliceReg := int8(numIn)
				c.fb.MakeSlice(true, true, funcType.In(numIn-1), int8(varArgs), int8(varArgs), sliceReg)
				for i := 0; i < varArgs; i++ {
					tmpReg := c.fb.NewRegister(typ.Kind())
					c.emitExpr(args[i+numIn-1], tmpReg, typ)
					indexReg := c.fb.NewRegister(reflect.Int)
					c.fb.Move(true, int8(i), indexReg, reflect.Int)
					c.fb.SetSlice(false, sliceReg, tmpReg, indexReg, typ.Kind())
				}
			}
		}
	} else { // No-variadic function.
		if numIn > 1 && len(args) == 1 { // f(g()), where f takes more than 1 argument.
			regs, types := c.emitCall(args[0].(*ast.Call))
			for i := range regs {
				dstType := funcType.In(i)
				reg := c.fb.NewRegister(dstType.Kind())
				c.changeRegister(false, regs[i], reg, types[i], dstType)
			}
		} else {
			for i := 0; i < numIn; i++ {
				typ := funcType.In(i)
				reg := c.fb.NewRegister(typ.Kind())
				c.emitExpr(args[i], reg, typ)
			}
		}
	}
	return regs, types
}

// prepareFunctionBodyParameters prepares fun's parameters (out and int) before
// emitting its body.
func (c *Emitter) prepareFunctionBodyParameters(fun *ast.Func) {
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

// emitCall emits instruction for a call, returning the list of registers (and
// their respective type) within which return values is inserted.
func (c *Emitter) emitCall(call *ast.Call) ([]int8, []reflect.Type) {
	stackShift := vm.StackShift{
		int8(c.fb.numRegs[reflect.Int]),
		int8(c.fb.numRegs[reflect.Float64]),
		int8(c.fb.numRegs[reflect.String]),
		int8(c.fb.numRegs[reflect.Interface]),
	}
	if ident, ok := call.Func.(*ast.Identifier); ok {
		if !c.fb.IsVariable(ident.Name) {
			if fun, isScrigoFunc := c.availableScrigoFunctions[ident.Name]; isScrigoFunc {
				regs, types := c.prepareCallParameters(fun.Type, call.Args, false)
				index := c.scrigoFunctionIndex(fun)
				c.fb.Call(index, stackShift, call.Pos().Line)
				return regs, types
			}
			if _, isNativeFunc := c.availableNativeFunctions[ident.Name]; isNativeFunc {
				fun := c.availableNativeFunctions[ident.Name]
				funcType := reflect.TypeOf(fun.Func)
				regs, types := c.prepareCallParameters(funcType, call.Args, true)
				index := c.nativeFunctionIndex(fun)
				if funcType.IsVariadic() {
					numVar := len(call.Args) - (funcType.NumIn() - 1)
					c.fb.CallNative(index, int8(numVar), stackShift)
				} else {
					c.fb.CallNative(index, vm.NoVariadic, stackShift)
				}
				return regs, types
			}
		}
	}
	if sel, ok := call.Func.(*ast.Selector); ok {
		if name, ok := sel.Expr.(*ast.Identifier); ok {
			if isGoPkg := c.isNativePkg[name.Name]; isGoPkg {
				fun := c.availableNativeFunctions[name.Name+"."+sel.Ident]
				funcType := reflect.TypeOf(fun.Func)
				regs, types := c.prepareCallParameters(funcType, call.Args, true)
				index := c.nativeFunctionIndex(fun)
				if funcType.IsVariadic() {
					numVar := len(call.Args) - (funcType.NumIn() - 1)
					c.fb.CallNative(index, int8(numVar), stackShift)
				} else {
					c.fb.CallNative(index, vm.NoVariadic, stackShift)
				}
				return regs, types
			} else {
				panic("TODO(Gianluca): not implemented")
			}
		}
	}
	funReg, _, isRegister := c.quickEmitExpr(call.Func, c.typeinfo[call.Func].Type)
	if !isRegister {
		funReg = c.fb.NewRegister(reflect.Func)
		c.emitExpr(call.Func, funReg, c.typeinfo[call.Func].Type)
	}
	funcType := c.typeinfo[call.Func].Type
	regs, types := c.prepareCallParameters(funcType, call.Args, true)
	c.fb.CallIndirect(funReg, 0, stackShift)
	return regs, types
}

// emitExpr emits instruction such that expr value is put into reg. If reg is
// zero instructions are emitted anyway but result is discarded.
func (c *Emitter) emitExpr(expr ast.Expression, reg int8, dstType reflect.Type) {
	// TODO (Gianluca): review all "kind" arguments in every emitExpr call.
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
			c.emitExpr(expr.Expr1, reg, dstType)
			endIf := c.fb.NewLabel()
			c.fb.If(true, reg, vm.ConditionEqual, cmp, reflect.Int)
			c.fb.Goto(endIf)
			c.emitExpr(expr.Expr2, reg, dstType)
			c.fb.SetLabelAddr(endIf)
			c.fb.ExitStack()
			return
		}

		c.fb.EnterStack()

		xType := c.typeinfo[expr.Expr1].Type
		x := c.fb.NewRegister(xType.Kind())
		c.emitExpr(expr.Expr1, x, xType)

		y, ky, isRegister := c.quickEmitExpr(expr.Expr2, xType)
		if !ky && !isRegister {
			y = c.fb.NewRegister(xType.Kind())
			c.emitExpr(expr.Expr2, y, xType)
		}

		res := c.fb.NewRegister(xType.Kind())

		switch op := expr.Operator(); {
		case op == ast.OperatorAddition && xType.Kind() == reflect.String && reg != 0:
			if ky {
				y = c.fb.NewRegister(reflect.String)
				c.emitExpr(expr.Expr2, y, xType)
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
			var cond vm.Condition
			switch op {
			case ast.OperatorEqual:
				cond = vm.ConditionEqual
			case ast.OperatorNotEqual:
				cond = vm.ConditionNotEqual
			case ast.OperatorLess:
				cond = vm.ConditionLess
			case ast.OperatorLessOrEqual:
				cond = vm.ConditionLessOrEqual
			case ast.OperatorGreater:
				cond = vm.ConditionGreater
			case ast.OperatorGreaterOrEqual:
				cond = vm.ConditionGreaterOrEqual
			}
			if reg != 0 {
				c.fb.Move(true, 1, reg, reflect.Bool)
				c.fb.If(ky, x, cond, y, xType.Kind())
				c.fb.Move(true, 0, reg, reflect.Bool)
			}
		case op == ast.OperatorOr,
			op == ast.OperatorAnd,
			op == ast.OperatorXor,
			op == ast.OperatorAndNot,
			op == ast.OperatorLeftShift,
			op == ast.OperatorRightShift:
			if reg != 0 {
				c.fb.BinaryBitOperation(op, ky, x, y, reg, xType.Kind())
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
			c.emitBuiltin(expr, reg, dstType)
			c.fb.ExitStack()
			return
		}
		// Conversion.
		if val, ok := expr.Func.(*ast.Value); ok {
			if convertType, ok := val.Val.(reflect.Type); ok {
				typ := c.typeinfo[expr.Args[0]].Type
				arg := c.fb.NewRegister(typ.Kind())
				c.emitExpr(expr.Args[0], arg, typ)
				c.fb.Convert(arg, convertType, reg, typ.Kind())
				c.fb.ExitStack()
				return
			}
		}
		regs, types := c.emitCall(expr)
		if reg != 0 {
			c.changeRegister(false, regs[0], reg, types[0], dstType)
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
				c.fb.Move(true, index, indexReg, reflect.Int)
				value, kvalue, isRegister := c.quickEmitExpr(kv.Value, typ.Elem())
				if !kvalue && !isRegister {
					value = c.fb.NewRegister(typ.Elem().Kind())
					c.emitExpr(kv.Value, value, typ.Elem())
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
				c.emitExpr(kv.Key, keyReg, typ.Key())
				kValue := false // TODO(Gianluca).
				c.emitExpr(kv.Value, valueReg, typ.Elem())
				c.fb.ExitStack()
				c.fb.SetMap(kValue, reg, valueReg, keyReg)
			}
		}

	case *ast.TypeAssertion:
		typ := c.typeinfo[expr.Expr].Type
		exprReg, _, isRegister := c.quickEmitExpr(expr.Expr, typ)
		if !isRegister {
			exprReg = c.fb.NewRegister(typ.Kind())
			c.emitExpr(expr.Expr, exprReg, typ)
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
		c.emitExpr(expr.Expr, exprReg, exprType)
		field := int8(0) // TODO(Gianluca).
		c.fb.Selector(exprReg, field, reg)

	case *ast.UnaryOperator:
		typ := c.typeinfo[expr.Expr].Type
		var tmpReg int8
		if reg != 0 {
			tmpReg = c.fb.NewRegister(typ.Kind())
		}
		c.emitExpr(expr.Expr, tmpReg, typ)
		if reg == 0 {
			return
		}
		switch expr.Operator() {
		case ast.OperatorNot:
			c.fb.SubInv(true, tmpReg, int8(1), tmpReg, reflect.Int)
			if reg != 0 {
				c.changeRegister(false, tmpReg, reg, typ, dstType)
			}
		case ast.OperatorAnd, ast.OperatorMultiplication:
			switch expr := expr.Expr.(type) {
			case *ast.Identifier:
				if c.fb.IsVariable(expr.Name) {
					varReg := c.fb.ScopeLookup(expr.Name)
					c.fb.New(reflect.PtrTo(typ), reg)
					c.fb.Move(false, -varReg, reg, dstType.Kind())
				} else {
					panic("TODO(Gianluca): not implemented")
				}
			default:
				panic("TODO(Gianluca): not implemented")
			}
		case ast.OperatorAddition:
			// Do nothing.
		case ast.OperatorSubtraction:
			c.fb.SubInv(true, tmpReg, 0, tmpReg, dstType.Kind())
			if reg != 0 {
				c.changeRegister(false, tmpReg, reg, typ, dstType)
			}
		case ast.OperatorReceive:
			c.fb.Receive(tmpReg, 0, reg)
		default:
			panic(fmt.Errorf("TODO: not implemented operator %s", expr.Operator()))
		}

	case *ast.Func:
		if reg == 0 {
			return
		}
		fn := c.fb.Func(reg, c.typeinfo[expr].Type)
		funcLitBuilder := NewBuilder(fn)
		currFb := c.fb
		currFn := c.currentFunction
		c.fb = funcLitBuilder
		c.currentFunction = fn
		c.fb.EnterScope()
		c.prepareFunctionBodyParameters(expr)
		addExplicitReturn(expr)
		c.emitNodes(expr.Body.Nodes)
		c.fb.ExitScope()
		c.fb = currFb
		c.currentFunction = currFn

	case *ast.Identifier:
		if reg == 0 {
			return
		}
		typ := c.typeinfo[expr].Type
		out, isValue, isRegister := c.quickEmitExpr(expr, typ)
		if isValue {
			c.changeRegister(true, out, reg, typ, dstType)
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
		out, isValue, isRegister := c.quickEmitExpr(expr, typ)
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
				c.fb.LoadNumber(vm.TypeInt, constant, reg)
			case int8:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(vm.TypeInt, constant, reg)
			case int16:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(vm.TypeInt, constant, reg)
			case int32:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(vm.TypeInt, constant, reg)
			case int64:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(vm.TypeInt, constant, reg)
			case uint:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(vm.TypeInt, constant, reg)
			case uint8:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(vm.TypeInt, constant, reg)
			case uint16:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(vm.TypeInt, constant, reg)
			case uint32:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(vm.TypeInt, constant, reg)
			case uint64:
				constant := c.fb.MakeIntConstant(int64(v))
				c.fb.LoadNumber(vm.TypeInt, constant, reg)
			case string:
				constant := c.fb.MakeStringConstant(v)
				c.changeRegister(true, constant, reg, typ, dstType)
			case float32:
				constant := c.fb.MakeFloatConstant(float64(v))
				c.fb.LoadNumber(vm.TypeFloat, constant, reg)
			case float64:
				constant := c.fb.MakeFloatConstant(v)
				c.fb.LoadNumber(vm.TypeFloat, constant, reg)
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
		i, ki, isRegister := c.quickEmitExpr(expr, intType)
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
		i, ki, isRegister := c.quickEmitExpr(expr, stringType)
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
		out, _, isRegister := c.quickEmitExpr(expr.Expr, exprType)
		if isRegister {
			exprReg = out
		} else {
			exprReg = c.fb.NewRegister(exprType.Kind())
		}
		out, isValue, isRegister := c.quickEmitExpr(expr.Index, intType)
		ki := false
		var i int8
		if isValue {
			ki = true
			i = out
		} else if isRegister {
			i = out
		} else {
			i = c.fb.NewRegister(reflect.Int)
			c.emitExpr(expr.Index, i, dstType)
		}
		c.fb.Index(ki, exprReg, i, reg, exprType)

	case *ast.Slicing:
		panic("TODO: not implemented")

	default:
		panic(fmt.Sprintf("emitExpr currently does not support %T nodes", expr))

	}

}

// quickEmitExpr checks if expr is k (which means immediate for integers and
// floats and constant for strings and generals) or a register, putting it into
// out. If it's neither of them, both k and isRegister are false and content of
// out is unspecified.
func (c *Emitter) quickEmitExpr(expr ast.Expression, expectedType reflect.Type) (out int8, k, isRegister bool) {
	// TODO (Gianluca): quickEmitExpr must evaluate only expression which does
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
		case float64:
			if float64(int(v)) == v {
				if -127 < v && v < 126 {
					return int8(v), true, false
				}
			}
		}
	}
	return 0, false, false
}

// emitBuiltin emits instructions for a builtin call, writing result into reg if
// necessary.
func (c *Emitter) emitBuiltin(call *ast.Call, reg int8, dstType reflect.Type) {
	switch call.Func.(*ast.Identifier).Name {
	case "append":
		panic("TODO: not implemented")
	case "cap":
		typ := c.typeinfo[call.Args[0]].Type
		s := c.fb.NewRegister(typ.Kind())
		c.emitExpr(call.Args[0], s, typ)
		c.fb.Cap(s, reg)
	case "close":
		panic("TODO: not implemented")
	case "complex":
		panic("TODO: not implemented")
	case "copy":
		dst, _, isRegister := c.quickEmitExpr(call.Args[0], c.typeinfo[call.Args[0]].Type)
		if !isRegister {
			dst = c.fb.NewRegister(reflect.Slice)
			c.emitExpr(call.Args[0], dst, c.typeinfo[call.Args[0]].Type)
		}
		src, _, isRegister := c.quickEmitExpr(call.Args[1], c.typeinfo[call.Args[1]].Type)
		if !isRegister {
			src = c.fb.NewRegister(reflect.Slice)
			c.emitExpr(call.Args[0], src, c.typeinfo[call.Args[0]].Type)
		}
		c.fb.Copy(dst, src, reg)
	case "delete":
		mapExpr := call.Args[0]
		keyExpr := call.Args[1]
		mapType := c.typeinfo[mapExpr].Type
		keyType := c.typeinfo[keyExpr].Type
		mapp, _, isRegister := c.quickEmitExpr(mapExpr, c.typeinfo[mapExpr].Type)
		if !isRegister {
			mapp = c.fb.NewRegister(mapType.Kind())
			c.emitExpr(mapExpr, mapp, c.typeinfo[mapExpr].Type)
		}
		key, _, isRegister := c.quickEmitExpr(keyExpr, keyType)
		if !isRegister {
			key = c.fb.NewRegister(keyType.Kind())
			c.emitExpr(keyExpr, key, c.typeinfo[keyExpr].Type)
		}
		c.fb.Delete(mapp, key)
	case "imag":
		panic("TODO: not implemented")
	case "html": // TODO (Gianluca): to review.
		panic("TODO: not implemented")
	case "len":
		typ := c.typeinfo[call.Args[0]].Type
		s := c.fb.NewRegister(typ.Kind())
		c.emitExpr(call.Args[0], s, typ)
		c.fb.Len(s, reg, typ)
	case "make":
		typ := call.Args[0].(*ast.Value).Val.(reflect.Type)
		regType := c.fb.Type(typ)
		switch typ.Kind() {
		case reflect.Map:
			size, kSize, isRegister := c.quickEmitExpr(call.Args[1], intType)
			if !kSize && !isRegister {
				size = c.fb.NewRegister(reflect.Int)
				c.emitExpr(call.Args[1], size, c.typeinfo[call.Args[1]].Type)
			}
			c.fb.MakeMap(regType, kSize, size, reg)
		case reflect.Slice:
			lenExpr := call.Args[1]
			capExpr := call.Args[2]
			len, kLen, isRegister := c.quickEmitExpr(lenExpr, intType)
			if !kLen && !isRegister {
				len = c.fb.NewRegister(reflect.Int)
				c.emitExpr(lenExpr, len, c.typeinfo[lenExpr].Type)
			}
			cap, kCap, isRegister := c.quickEmitExpr(capExpr, intType)
			if !kCap && !isRegister {
				cap = c.fb.NewRegister(reflect.Int)
				c.emitExpr(capExpr, cap, c.typeinfo[capExpr].Type)
			}
			c.fb.MakeSlice(kLen, kCap, typ, len, cap, reg)
		case reflect.Chan:
			chanType := c.typeinfo[call.Args[0]].Type
			chanTypeIndex := c.fb.AddType(chanType)
			var kCapacity bool
			var capacity int8
			if len(call.Args) == 1 {
				capacity = 0
				kCapacity = true
			} else {
				var isRegister bool
				capacity, kCapacity, isRegister = c.quickEmitExpr(call.Args[1], intType)
				if !kCapacity && !isRegister {
					capacity = c.fb.NewRegister(reflect.Int)
					c.emitExpr(call.Args[1], capacity, intType)
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
		// i = vm.Instruction{Op: vm.OpNew, B: t, C: }
	case "panic":
		arg := call.Args[0]
		reg, _, isRegister := c.quickEmitExpr(arg, emptyInterfaceType)
		if !isRegister {
			reg = c.fb.NewRegister(reflect.Interface)
			c.emitExpr(arg, reg, emptyInterfaceType)
		}
		c.fb.Panic(reg, call.Pos().Line)
	case "print":
		for i := range call.Args {
			arg := c.fb.NewRegister(reflect.Interface)
			c.emitExpr(call.Args[i], arg, emptyInterfaceType)
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

// emitNodes emits instructions for nodes.
func (c *Emitter) emitNodes(nodes []ast.Node) {
	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.Assignment:
			c.emitAssignmentNode(node)

		case *ast.Block:
			c.fb.EnterScope()
			c.emitNodes(node.Nodes)
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
			c.emitExpr(funNode, funReg, c.typeinfo[funNode].Type)
			offset := vm.StackShift{
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
			argsShift := vm.StackShift{}
			switch node.(type) {
			case *ast.Defer:
				c.fb.Defer(funReg, vm.NoVariadic, offset, argsShift)
			case *ast.Go:
				// TODO(Gianluca):
				c.fb.Go()
			}

		case *ast.If:
			c.fb.EnterScope()
			if node.Assignment != nil {
				c.emitNodes([]ast.Node{node.Assignment})
			}
			c.emitCondition(node.Condition)
			if node.Else == nil { // TODO (Gianluca): can "then" and "else" be unified in some way?
				endIfLabel := c.fb.NewLabel()
				c.fb.Goto(endIfLabel)
				c.emitNodes(node.Then.Nodes)
				c.fb.SetLabelAddr(endIfLabel)
			} else {
				elseLabel := c.fb.NewLabel()
				c.fb.Goto(elseLabel)
				c.emitNodes(node.Then.Nodes)
				endIfLabel := c.fb.NewLabel()
				c.fb.Goto(endIfLabel)
				c.fb.SetLabelAddr(elseLabel)
				if node.Else != nil {
					switch els := node.Else.(type) {
					case *ast.If:
						c.emitNodes([]ast.Node{els})
					case *ast.Block:
						c.emitNodes(els.Nodes)
					}
				}
				c.fb.SetLabelAddr(endIfLabel)
			}
			c.fb.ExitScope()

		case *ast.For:
			c.fb.EnterScope()
			if node.Init != nil {
				c.emitNodes([]ast.Node{node.Init})
			}
			if node.Condition != nil {
				forLabel := c.fb.NewLabel()
				c.fb.SetLabelAddr(forLabel)
				c.emitCondition(node.Condition)
				endForLabel := c.fb.NewLabel()
				c.fb.Goto(endForLabel)
				c.emitNodes(node.Body)
				if node.Post != nil {
					c.emitNodes([]ast.Node{node.Post})
				}
				c.fb.Goto(forLabel)
				c.fb.SetLabelAddr(endForLabel)
			} else {
				forLabel := c.fb.NewLabel()
				c.fb.SetLabelAddr(forLabel)
				c.emitNodes(node.Body)
				if node.Post != nil {
					c.emitNodes([]ast.Node{node.Post})
				}
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
			// 		shift := vm.StackShift{}
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
			// 		c.fb.TailCall(vm.CurrentFunction, node.Pos().Line)
			// 		continue
			// 	}
			// }
			offset := [4]int8{}
			for i, v := range node.Values {
				typ := c.currentFunction.Type.Out(i)
				var reg int8
				switch kindToType(typ.Kind()) {
				case vm.TypeInt:
					offset[0]++
					reg = offset[0]
				case vm.TypeFloat:
					offset[1]++
					reg = offset[1]
				case vm.TypeString:
					offset[2]++
					reg = offset[2]
				case vm.TypeGeneral:
					offset[3]++
					reg = offset[3]
				}
				c.emitExpr(v, reg, typ)
			}
			c.fb.Return()

		case *ast.Send:
			ch := c.fb.NewRegister(reflect.Chan)
			c.emitExpr(node.Channel, ch, c.typeinfo[node.Channel].Type)
			elemType := c.typeinfo[node.Value].Type
			v := c.fb.NewRegister(elemType.Kind())
			c.emitExpr(node.Value, v, elemType)
			c.fb.Send(ch, v)

		case *ast.Switch:
			c.emitSwitch(node)

		case *ast.TypeSwitch:
			c.emitTypeSwitch(node)

		case *ast.Var:
			addresses := make([]Address, len(node.Identifiers))
			for i, v := range node.Identifiers {
				varType := c.typeinfo[v].Type
				if c.indirectVars[v] {
					varReg := -c.fb.NewRegister(reflect.Interface)
					c.fb.BindVarReg(v.Name, varReg)
					addresses[i] = c.NewAddress(AddressIndirectDeclaration, varType, varReg, 0)
				} else {
					varReg := c.fb.NewRegister(varType.Kind())
					c.fb.BindVarReg(v.Name, varReg)
					addresses[i] = c.NewAddress(AddressRegister, varType, varReg, 0)
				}

			}
			c.assign(addresses, node.Values)

		case ast.Expression:
			// TODO (Gianluca): use 0 (which is no longer a valid
			// register) and handle it as a special case in emitExpr.
			c.emitExpr(node, 0, reflect.Type(nil))

		}
	}
}

// emitTypeSwitch emits instructions for a type switch node.
func (c *Emitter) emitTypeSwitch(node *ast.TypeSwitch) {
	// TODO (Gianluca): a type-checker bug does not replace type switch type with proper value.

	c.fb.EnterScope()

	if node.Init != nil {
		c.emitNodes([]ast.Node{node.Init})
	}

	typAss := node.Assignment.Values[0].(*ast.TypeAssertion)
	typ := c.typeinfo[typAss.Expr].Type
	expr := c.fb.NewRegister(typ.Kind())
	c.emitExpr(typAss.Expr, expr, typ)

	// typ := node.Assignment.Values[0].(*ast.Value).Val.(reflect.Type)
	// typReg := c.fb.Type(typ)
	// if variab := node.Assignment.Variables[0]; !isBlankast.Identifier(variab) {
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
		c.fb.EnterScope()
		c.emitNodes(cas.Body)
		c.fb.ExitScope()
		c.fb.Goto(endSwitchLabel)
	}

	c.fb.SetLabelAddr(endSwitchLabel)
	c.fb.ExitScope()
}

// emitSwitch emits instructions for a switch node.
func (c *Emitter) emitSwitch(node *ast.Switch) {

	c.fb.EnterScope()

	if node.Init != nil {
		c.emitNodes([]ast.Node{node.Init})
	}

	typ := c.typeinfo[node.Expr].Type
	expr := c.fb.NewRegister(typ.Kind())
	c.emitExpr(node.Expr, expr, typ)
	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := c.fb.NewLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = c.fb.NewLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			y, ky, isRegister := c.quickEmitExpr(caseExpr, typ)
			if !ky && !isRegister {
				y = c.fb.NewRegister(typ.Kind())
				c.emitExpr(caseExpr, y, typ)
			}
			c.fb.If(ky, expr, vm.ConditionNotEqual, y, typ.Kind()) // Condizione negata per poter concatenare gli if
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
		c.fb.EnterScope()
		c.emitNodes(cas.Body)
		if !cas.Fallthrough {
			c.fb.Goto(endSwitchLabel)
		}
		c.fb.ExitScope()
	}

	c.fb.SetLabelAddr(endSwitchLabel)

	c.fb.ExitScope()
}

// emitCondition emits instructions for a condition. Last instruction added by
// this method is always "If".
func (c *Emitter) emitCondition(cond ast.Expression) {

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
			x, _, isRegister := c.quickEmitExpr(expr, exprType)
			if !isRegister {
				x = c.fb.NewRegister(exprType.Kind())
				c.emitExpr(expr, x, exprType)
			}
			condType := vm.ConditionNotNil
			if cond.Operator() == ast.OperatorEqual {
				condType = vm.ConditionNil
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
				x, _, isRegister := c.quickEmitExpr(lenArg, lenArgType)
				if !isRegister {
					x = c.fb.NewRegister(lenArgType.Kind())
					c.emitExpr(lenArg, x, lenArgType)
				}
				exprType := c.typeinfo[expr].Type
				y, ky, isRegister := c.quickEmitExpr(expr, exprType)
				if !ky && !isRegister {
					y = c.fb.NewRegister(exprType.Kind())
					c.emitExpr(expr, y, exprType)
				}
				var condType vm.Condition
				switch cond.Operator() {
				case ast.OperatorEqual:
					condType = vm.ConditionEqualLen
				case ast.OperatorNotEqual:
					condType = vm.ConditionNotEqualLen
				case ast.OperatorLess:
					condType = vm.ConditionLessLen
				case ast.OperatorLessOrEqual:
					condType = vm.ConditionLessOrEqualLen
				case ast.OperatorGreater:
					condType = vm.ConditionGreaterLen
				case ast.OperatorGreaterOrEqual:
					condType = vm.ConditionGreaterOrEqualLen
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
				x, _, isRegister := c.quickEmitExpr(cond.Expr1, expr1Type)
				if !isRegister {
					x = c.fb.NewRegister(expr1Type.Kind())
					c.emitExpr(cond.Expr1, x, expr1Type)
				}
				expr2Type := c.typeinfo[cond.Expr2].Type
				y, ky, isRegister := c.quickEmitExpr(cond.Expr2, expr2Type)
				if !ky && !isRegister {
					y = c.fb.NewRegister(expr2Type.Kind())
					c.emitExpr(cond.Expr2, y, expr2Type)
				}
				var condType vm.Condition
				switch cond.Operator() {
				case ast.OperatorEqual:
					condType = vm.ConditionEqual
				case ast.OperatorNotEqual:
					condType = vm.ConditionNotEqual
				case ast.OperatorLess:
					condType = vm.ConditionLess
				case ast.OperatorLessOrEqual:
					condType = vm.ConditionLessOrEqual
				case ast.OperatorGreater:
					condType = vm.ConditionGreater
				case ast.OperatorGreaterOrEqual:
					condType = vm.ConditionGreaterOrEqual
				}
				if reflect.Uint <= kind && kind <= reflect.Uint64 {
					// Equality and not equality checks are not
					// optimized for uints.
					if condType == vm.ConditionEqual || condType == vm.ConditionNotEqual {
						kind = reflect.Int
					}
				}
				c.fb.If(ky, x, condType, y, kind)
				return
			}
		}

	default:

		condType := c.typeinfo[cond].Type
		x, _, isRegister := c.quickEmitExpr(cond, condType)
		if !isRegister {
			x = c.fb.NewRegister(condType.Kind())
			c.emitExpr(cond, x, condType)
		}
		yConst := c.fb.MakeIntConstant(1)
		y := c.fb.NewRegister(reflect.Bool)
		c.fb.LoadNumber(vm.TypeInt, yConst, y)
		c.fb.If(false, x, vm.ConditionEqual, y, reflect.Bool)

	}

}
