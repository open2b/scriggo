// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"

	"scrigo/internal/compiler/ast"
	"scrigo/vm"
)

// An Emitter emits instructions for the VM.
type Emitter struct {
	CurrentFunction  *vm.ScrigoFunction
	TypeInfo         map[ast.Node]*TypeInfo
	FB               *FunctionBuilder // current function builder.
	importableGoPkgs map[string]*GoPackage
	IndirectVars     map[*ast.Identifier]bool

	upvarsNames map[*vm.ScrigoFunction]map[string]int

	availableScrigoFunctions map[string]*vm.ScrigoFunction
	availableNativeFunctions map[string]*vm.NativeFunction
	availableVariables       map[string]vm.Global

	// TODO (Gianluca): find better names.
	// TODO (Gianluca): do these maps have to have a *vm.ScrigoFunction key or
	// can they be related to CurrentFunction in some way?
	assignedScrigoFunctions map[*vm.ScrigoFunction]map[*vm.ScrigoFunction]int8
	assignedNativeFunctions map[*vm.ScrigoFunction]map[*vm.NativeFunction]int8
	assignedVariables       map[*vm.ScrigoFunction]map[vm.Global]uint8

	isNativePkg map[string]bool

	// rangeLabels is a list of current active Ranges. First element is the
	// Range address, second referrs to the first instruction outside Range's
	// body.
	rangeLabels [][2]uint32

	// globals holds all global variables.
	globals []reflect.Type

	// globalsIndexes maps global variable names to their index inside globals.
	globalsIndexes map[string]int16
}

// Main returns main function.
func (c *Emitter) Main() *vm.ScrigoFunction {
	return c.availableScrigoFunctions["main"]
}

// TODO(Gianluca): rename exported methods from "Compile" to "Emit".

// NewEmitter returns a new compiler reading sources from r.
// Native (Go) packages are made available for importing.
func NewEmitter(tree *ast.Tree, packages map[string]*GoPackage, typeInfos map[ast.Node]*TypeInfo, indirectVars map[*ast.Identifier]bool) *Emitter {
	c := &Emitter{

		importableGoPkgs: packages,

		upvarsNames: make(map[*vm.ScrigoFunction]map[string]int),

		availableScrigoFunctions: map[string]*vm.ScrigoFunction{},
		availableNativeFunctions: map[string]*vm.NativeFunction{},
		availableVariables:       map[string]vm.Global{},

		assignedScrigoFunctions: map[*vm.ScrigoFunction]map[*vm.ScrigoFunction]int8{},
		assignedNativeFunctions: map[*vm.ScrigoFunction]map[*vm.NativeFunction]int8{},
		assignedVariables:       map[*vm.ScrigoFunction]map[vm.Global]uint8{},

		isNativePkg: map[string]bool{},

		globalsIndexes: map[string]int16{},

		TypeInfo:     typeInfos,
		IndirectVars: indirectVars,
	}
	return c
}

// emitPackage emits pkg.
func (c *Emitter) EmitPackage(pkg *ast.Package) {

	// Emits imports.
	for _, decl := range pkg.Declarations {
		if imp, ok := decl.(*ast.Import); ok {
			c.emitImport(imp)
		}
	}

	// Stores all function declarations in current package before building
	// their bodies: order of declaration doesn't matter at package level.
	initFuncs := []*vm.ScrigoFunction{} // List of all "init" functions in current package.
	initToBuild := 0                    // Index of next "init" function to build.
	for _, dec := range pkg.Declarations {
		if fun, ok := dec.(*ast.Func); ok {
			fn := NewScrigoFunction("main", fun.Ident.Name, fun.Type.Reflect)
			if fun.Ident.Name == "init" {
				initFuncs = append(initFuncs, fn)
			} else {
				c.availableScrigoFunctions[fun.Ident.Name] = fn
			}
		}
	}

	// Emits package variables.
	var initVarsFn *vm.ScrigoFunction
	var initVarsFb *FunctionBuilder
	packageVariablesRegisters := map[string]int8{}
	for _, dec := range pkg.Declarations {
		if n, ok := dec.(*ast.Var); ok {
			// If package has some variable declarations, a special "init" function
			// must be created to initialize them. "$initvars" is used because is not
			// a valid Go identifier, so there's no risk of collision with Scrigo
			// defined functions.
			backupFn := c.CurrentFunction
			backupFb := c.FB
			if initVarsFn == nil {
				initVarsFn = NewScrigoFunction("main", "$initvars", reflect.FuncOf(nil, nil, false))
				c.availableScrigoFunctions["$initvars"] = initVarsFn
				initVarsFb = NewBuilder(initVarsFn)
				initVarsFb.EnterScope()
			}
			c.CurrentFunction = initVarsFn
			c.FB = initVarsFb
			addresses := make([]Address, len(n.Identifiers))
			for i, v := range n.Identifiers {
				varType := c.TypeInfo[v].Type
				varReg := -c.FB.NewRegister(reflect.Interface)
				c.FB.BindVarReg(v.Name, varReg)
				addresses[i] = c.NewAddress(AddressIndirectDeclaration, varType, varReg, 0)
				packageVariablesRegisters[v.Name] = varReg
				c.globals = append(c.globals, varType)
				c.globalsIndexes[v.Name] = int16(len(c.globals) - 1)
			}
			c.assign(addresses, n.Values)
			c.CurrentFunction = backupFn
			c.FB = backupFb
		}
	}

	// Emits function declarations.
	for _, dec := range pkg.Declarations {
		if n, ok := dec.(*ast.Func); ok {
			var fn *vm.ScrigoFunction
			if n.Ident.Name == "init" {
				fn = initFuncs[initToBuild]
				initToBuild++
			} else {
				fn = c.availableScrigoFunctions[n.Ident.Name]
			}
			c.CurrentFunction = fn
			c.FB = NewBuilder(fn)
			c.FB.EnterScope()

			// If function is "main", variables initialization function
			// must be called as first statement inside main.
			if n.Ident.Name == "main" {
				// First: initializes package variables.
				if initVarsFn != nil {
					iv := c.availableScrigoFunctions["$initvars"]
					index := c.FB.AddScrigoFunction(iv)
					c.FB.Call(int8(index), vm.StackShift{}, 0)
				}
				// Second: calls all init functions, in order.
				for _, initFunc := range initFuncs {
					index := c.FB.AddScrigoFunction(initFunc)
					c.FB.Call(int8(index), vm.StackShift{}, 0)
				}
			}
			c.prepareFunctionBodyParameters(n)
			AddExplicitReturn(n)
			c.EmitNodes(n.Body.Nodes)
			c.FB.End()
			c.FB.ExitScope()
		}
	}

	if initVarsFn != nil {
		// Global variables have been locally defined inside the "$initvars"
		// function; their values must now be exported to be available
		// globally.
		backupFn := c.CurrentFunction
		backupFb := c.FB
		c.CurrentFunction = initVarsFn
		c.FB = initVarsFb
		for name, reg := range packageVariablesRegisters {
			index := c.globalsIndexes[name]
			c.FB.SetVar(false, reg, int(index))
		}
		c.CurrentFunction = backupFn
		c.FB = backupFb
		initVarsFb.ExitScope()
		initVarsFb.Return()
	}

	// Assigns globals to main's Globals.
	main := c.availableScrigoFunctions["main"]
	for name, index := range c.globalsIndexes {
		global := c.globals[index]
		main.Globals = append(main.Globals, vm.Global{Pkg: "main", Name: name, Type: global})
	}
	// All functions share Globals.
	for _, f := range c.availableScrigoFunctions {
		f.Globals = main.Globals
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
		regs[i] = c.FB.NewRegister(typ.Kind())
		types[i] = typ
	}
	if funcType.IsVariadic() {
		for i := 0; i < numIn-1; i++ {
			typ := funcType.In(i)
			reg := c.FB.NewRegister(typ.Kind())
			c.FB.EnterScope()
			c.emitExpr(args[i], reg, typ)
			c.FB.ExitScope()
		}
		if varArgs := len(args) - (numIn - 1); varArgs > 0 {
			typ := funcType.In(numIn - 1).Elem()
			if isNative {
				for i := 0; i < varArgs; i++ {
					reg := c.FB.NewRegister(typ.Kind())
					c.FB.EnterStack()
					c.emitExpr(args[i+numIn-1], reg, typ)
					c.FB.ExitStack()
				}
			} else {
				sliceReg := int8(numIn)
				c.FB.MakeSlice(true, true, funcType.In(numIn-1), int8(varArgs), int8(varArgs), sliceReg)
				for i := 0; i < varArgs; i++ {
					tmpReg := c.FB.NewRegister(typ.Kind())
					c.FB.EnterStack()
					c.emitExpr(args[i+numIn-1], tmpReg, typ)
					c.FB.ExitStack()
					indexReg := c.FB.NewRegister(reflect.Int)
					c.FB.Move(true, int8(i), indexReg, reflect.Int)
					c.FB.SetSlice(false, sliceReg, tmpReg, indexReg, typ.Kind())
				}
			}
		}
	} else { // No-variadic function.
		if numIn > 1 && len(args) == 1 { // f(g()), where f takes more than 1 argument.
			regs, types := c.emitCall(args[0].(*ast.Call))
			for i := range regs {
				dstType := funcType.In(i)
				reg := c.FB.NewRegister(dstType.Kind())
				c.changeRegister(false, regs[i], reg, types[i], dstType)
			}
		} else {
			for i := 0; i < numIn; i++ {
				typ := funcType.In(i)
				reg := c.FB.NewRegister(typ.Kind())
				c.FB.EnterStack()
				c.emitExpr(args[i], reg, typ)
				c.FB.ExitStack()
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
		retReg := c.FB.NewRegister(kind)
		if res.Ident != nil {
			c.FB.BindVarReg(res.Ident.Name, retReg)
		}
	}
	// Binds function argument names to pre-allocated registers.
	fillParametersTypes(fun.Type.Parameters)
	for _, par := range fun.Type.Parameters {
		parType := par.Type.(*ast.Value).Val.(reflect.Type)
		kind := parType.Kind()
		argReg := c.FB.NewRegister(kind)
		if par.Ident != nil {
			c.FB.BindVarReg(par.Ident.Name, argReg)
		}
	}
}

// emitCall emits instruction for a call, returning the list of registers (and
// their respective type) within which return values are inserted.
func (c *Emitter) emitCall(call *ast.Call) ([]int8, []reflect.Type) {
	stackShift := vm.StackShift{
		int8(c.FB.numRegs[reflect.Int]),
		int8(c.FB.numRegs[reflect.Float64]),
		int8(c.FB.numRegs[reflect.String]),
		int8(c.FB.numRegs[reflect.Interface]),
	}
	if ident, ok := call.Func.(*ast.Identifier); ok {
		if !c.FB.IsVariable(ident.Name) {
			if fun, isScrigoFunc := c.availableScrigoFunctions[ident.Name]; isScrigoFunc {
				regs, types := c.prepareCallParameters(fun.Type, call.Args, false)
				index := c.scrigoFunctionIndex(fun)
				c.FB.Call(index, stackShift, call.Pos().Line)
				return regs, types
			}
			if _, isNativeFunc := c.availableNativeFunctions[ident.Name]; isNativeFunc {
				fun := c.availableNativeFunctions[ident.Name]
				funcType := reflect.TypeOf(fun.Func)
				regs, types := c.prepareCallParameters(funcType, call.Args, true)
				index := c.nativeFunctionIndex(fun)
				if funcType.IsVariadic() {
					numVar := len(call.Args) - (funcType.NumIn() - 1)
					c.FB.CallNative(index, int8(numVar), stackShift)
				} else {
					c.FB.CallNative(index, vm.NoVariadic, stackShift)
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
					c.FB.CallNative(index, int8(numVar), stackShift)
				} else {
					c.FB.CallNative(index, vm.NoVariadic, stackShift)
				}
				return regs, types
			} else {
				panic("TODO(Gianluca): not implemented")
			}
		}
	}
	funReg, _, isRegister := c.quickEmitExpr(call.Func, c.TypeInfo[call.Func].Type)
	if !isRegister {
		funReg = c.FB.NewRegister(reflect.Func)
		c.emitExpr(call.Func, funReg, c.TypeInfo[call.Func].Type)
	}
	funcType := c.TypeInfo[call.Func].Type
	regs, types := c.prepareCallParameters(funcType, call.Args, true)
	c.FB.CallIndirect(funReg, 0, stackShift)
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
			c.FB.EnterStack()
			c.emitExpr(expr.Expr1, reg, dstType)
			endIf := c.FB.NewLabel()
			c.FB.If(true, reg, vm.ConditionEqual, cmp, reflect.Int)
			c.FB.Goto(endIf)
			c.emitExpr(expr.Expr2, reg, dstType)
			c.FB.SetLabelAddr(endIf)
			c.FB.ExitStack()
			return
		}

		c.FB.EnterStack()

		xType := c.TypeInfo[expr.Expr1].Type
		x := c.FB.NewRegister(xType.Kind())
		c.emitExpr(expr.Expr1, x, xType)

		y, ky, isRegister := c.quickEmitExpr(expr.Expr2, xType)
		if !ky && !isRegister {
			y = c.FB.NewRegister(xType.Kind())
			c.emitExpr(expr.Expr2, y, xType)
		}

		res := c.FB.NewRegister(xType.Kind())

		switch op := expr.Operator(); {
		case op == ast.OperatorAddition && xType.Kind() == reflect.String && reg != 0:
			if ky {
				y = c.FB.NewRegister(reflect.String)
				c.emitExpr(expr.Expr2, y, xType)
			}
			c.FB.Concat(x, y, reg)
		case op == ast.OperatorAddition && reg != 0:
			c.FB.Add(ky, x, y, res, xType.Kind())
			c.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorSubtraction && reg != 0:
			c.FB.Sub(ky, x, y, res, xType.Kind())
			c.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorMultiplication && reg != 0:
			c.FB.Mul(ky, x, y, res, xType.Kind())
			c.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorDivision && reg != 0:
			c.FB.Div(ky, x, y, res, xType.Kind())
			c.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorDivision && reg == 0:
			dummyReg := c.FB.NewRegister(xType.Kind())
			c.FB.Div(ky, x, y, dummyReg, xType.Kind()) // produces division by zero.
		case op == ast.OperatorModulo && reg != 0:
			c.FB.Rem(ky, x, y, res, xType.Kind())
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
				c.FB.Move(true, 1, reg, reflect.Bool)
				c.FB.If(ky, x, cond, y, xType.Kind())
				c.FB.Move(true, 0, reg, reflect.Bool)
			}
		case op == ast.OperatorOr,
			op == ast.OperatorAnd,
			op == ast.OperatorXor,
			op == ast.OperatorAndNot,
			op == ast.OperatorLeftShift,
			op == ast.OperatorRightShift:
			if reg != 0 {
				c.FB.BinaryBitOperation(op, ky, x, y, reg, xType.Kind())
				if kindToType(xType.Kind()) != kindToType(dstType.Kind()) {
					c.changeRegister(ky, reg, reg, xType, dstType)
				}
			}
		default:
			panic(fmt.Errorf("TODO: not implemented operator %s", expr.Operator()))
		}

		c.FB.ExitStack()

	case *ast.Call:
		// Builtin call.
		c.FB.EnterStack()
		if c.TypeInfo[expr.Func].IsBuiltin() {
			c.emitBuiltin(expr, reg, dstType)
			c.FB.ExitStack()
			return
		}
		// Conversion.
		if val, ok := expr.Func.(*ast.Value); ok {
			if convertType, ok := val.Val.(reflect.Type); ok {
				typ := c.TypeInfo[expr.Args[0]].Type
				arg := c.FB.NewRegister(typ.Kind())
				c.emitExpr(expr.Args[0], arg, typ)
				c.FB.Convert(arg, convertType, reg, typ.Kind())
				c.FB.ExitStack()
				return
			}
		}
		regs, types := c.emitCall(expr)
		if reg != 0 {
			c.changeRegister(false, regs[0], reg, types[0], dstType)
		}
		c.FB.ExitStack()

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
				c.FB.MakeSlice(true, true, typ, size, size, reg)
			}
			var index int8 = -1
			for _, kv := range expr.KeyValues {
				if kv.Key != nil {
					index = int8(kv.Key.(*ast.Value).Val.(int))
				} else {
					index++
				}
				c.FB.EnterStack()
				indexReg := c.FB.NewRegister(reflect.Int)
				c.FB.Move(true, index, indexReg, reflect.Int)
				value, kvalue, isRegister := c.quickEmitExpr(kv.Value, typ.Elem())
				if !kvalue && !isRegister {
					value = c.FB.NewRegister(typ.Elem().Kind())
					c.emitExpr(kv.Value, value, typ.Elem())
				}
				if reg != 0 {
					c.FB.SetSlice(kvalue, reg, value, indexReg, typ.Elem().Kind())
				}
				c.FB.ExitStack()
			}
		case reflect.Struct:
			panic("TODO: not implemented")
		case reflect.Map:
			// TODO (Gianluca): handle maps with bigger size.
			size := len(expr.KeyValues)
			sizeReg := c.FB.MakeIntConstant(int64(size))
			regType := c.FB.Type(typ)
			c.FB.MakeMap(regType, true, sizeReg, reg)
			for _, kv := range expr.KeyValues {
				keyReg := c.FB.NewRegister(typ.Key().Kind())
				valueReg := c.FB.NewRegister(typ.Elem().Kind())
				c.FB.EnterStack()
				c.emitExpr(kv.Key, keyReg, typ.Key())
				kValue := false // TODO(Gianluca).
				c.emitExpr(kv.Value, valueReg, typ.Elem())
				c.FB.ExitStack()
				c.FB.SetMap(kValue, reg, valueReg, keyReg)
			}
		}

	case *ast.TypeAssertion:
		typ := c.TypeInfo[expr.Expr].Type
		exprReg, _, isRegister := c.quickEmitExpr(expr.Expr, typ)
		if !isRegister {
			exprReg = c.FB.NewRegister(typ.Kind())
			c.emitExpr(expr.Expr, exprReg, typ)
		}
		assertType := expr.Type.(*ast.Value).Val.(reflect.Type)
		c.FB.Assert(exprReg, assertType, reg)
		c.FB.Nop()

	case *ast.Selector:
		if v, ok := c.availableVariables[expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := c.variableIndex(v)
			c.FB.GetVar(int(index), reg) // TODO (Gianluca): to review.
			return
		}
		if nf, ok := c.availableNativeFunctions[expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := c.nativeFunctionIndex(nf)
			c.FB.GetFunc(true, index, reg)
			return
		}
		if sf, ok := c.availableScrigoFunctions[expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := c.scrigoFunctionIndex(sf)
			c.FB.GetFunc(false, index, reg)
			return
		}
		exprType := c.TypeInfo[expr.Expr].Type
		exprReg := c.FB.NewRegister(exprType.Kind())
		c.emitExpr(expr.Expr, exprReg, exprType)
		field := int8(0) // TODO(Gianluca).
		c.FB.Selector(exprReg, field, reg)

	case *ast.UnaryOperator:
		typ := c.TypeInfo[expr.Expr].Type
		var tmpReg int8
		if reg != 0 {
			tmpReg = c.FB.NewRegister(typ.Kind())
		}
		c.emitExpr(expr.Expr, tmpReg, typ)
		if reg == 0 {
			return
		}
		switch expr.Operator() {
		case ast.OperatorNot:
			c.FB.SubInv(true, tmpReg, int8(1), tmpReg, reflect.Int)
			if reg != 0 {
				c.changeRegister(false, tmpReg, reg, typ, dstType)
			}
		case ast.OperatorMultiplication:
			c.changeRegister(false, -tmpReg, reg, typ.Elem(), dstType)
		case ast.OperatorAnd:
			switch expr := expr.Expr.(type) {
			case *ast.Identifier:
				if c.FB.IsVariable(expr.Name) {
					varReg := c.FB.ScopeLookup(expr.Name)
					c.FB.New(reflect.PtrTo(typ), reg)
					c.FB.Move(false, -varReg, reg, dstType.Kind())
				} else {
					panic("TODO(Gianluca): not implemented")
				}
			case *ast.UnaryOperator:
				if expr.Operator() != ast.OperatorMultiplication {
					panic("bug") // TODO(Gianluca): to review.
				}
				panic("TODO(Gianluca): not implemented")
			case *ast.Index:
				panic("TODO(Gianluca): not implemented")
			case *ast.Selector:
				panic("TODO(Gianluca): not implemented")
			case *ast.CompositeLiteral:
				panic("TODO(Gianluca): not implemented")
			default:
				panic("TODO(Gianluca): not implemented")
			}
		case ast.OperatorAddition:
			// Do nothing.
		case ast.OperatorSubtraction:
			c.FB.SubInv(true, tmpReg, 0, tmpReg, dstType.Kind())
			if reg != 0 {
				c.changeRegister(false, tmpReg, reg, typ, dstType)
			}
		case ast.OperatorReceive:
			c.FB.Receive(tmpReg, 0, reg)
		default:
			panic(fmt.Errorf("TODO: not implemented operator %s", expr.Operator()))
		}

	case *ast.Func:
		if reg == 0 {
			return
		}

		fn := c.FB.Func(reg, c.TypeInfo[expr].Type)
		c.setClosureRefs(fn, expr.Upvars)

		funcLitBuilder := NewBuilder(fn)
		currFb := c.FB
		currFn := c.CurrentFunction
		c.FB = funcLitBuilder
		c.CurrentFunction = fn

		c.FB.EnterScope()
		c.prepareFunctionBodyParameters(expr)
		AddExplicitReturn(expr)
		c.EmitNodes(expr.Body.Nodes)
		c.FB.ExitScope()
		c.FB = currFb
		c.CurrentFunction = currFn

	case *ast.Identifier:
		if reg == 0 {
			return
		}
		typ := c.TypeInfo[expr].Type
		out, isValue, isRegister := c.quickEmitExpr(expr, typ)
		if isValue {
			c.changeRegister(true, out, reg, typ, dstType)
		} else if isRegister {
			c.changeRegister(false, out, reg, typ, dstType)
		} else {
			if fun, isScrigoFunc := c.availableScrigoFunctions[expr.Name]; isScrigoFunc {
				index := c.scrigoFunctionIndex(fun)
				c.FB.GetFunc(false, index, reg)
			} else if index, ok := c.upvarsNames[c.CurrentFunction][expr.Name]; ok {
				// TODO(Gianluca): this is an experimental handling of
				// emitting an expression into a register of a different
				// type. If this is correct, apply this solution to all
				// other expression emitting cases or generalize in some
				// way.
				if kindToType(typ.Kind()) == kindToType(dstType.Kind()) {
					c.FB.GetVar(index, reg)
				} else {
					tmpReg := c.FB.NewRegister(typ.Kind())
					c.FB.GetVar(index, tmpReg)
					c.changeRegister(false, tmpReg, reg, typ, dstType)
				}
			} else if index, ok := c.globalsIndexes[expr.Name]; ok {
				if kindToType(typ.Kind()) == kindToType(dstType.Kind()) {
					c.FB.GetVar(int(index), reg)
				} else {
					tmpReg := c.FB.NewRegister(typ.Kind())
					c.FB.GetVar(int(index), tmpReg)
					c.changeRegister(false, tmpReg, reg, typ, dstType)
				}
			} else {
				panic("bug")
			}
		}

	case *ast.Value:
		if reg == 0 {
			return
		}
		typ := c.TypeInfo[expr].Type
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
				constant := c.FB.MakeIntConstant(int64(v))
				c.FB.LoadNumber(vm.TypeInt, constant, reg)
				c.changeRegister(false, reg, reg, typ, dstType)
			case int8:
				constant := c.FB.MakeIntConstant(int64(v))
				c.FB.LoadNumber(vm.TypeInt, constant, reg)
				c.changeRegister(false, reg, reg, typ, dstType)
			case int16:
				constant := c.FB.MakeIntConstant(int64(v))
				c.FB.LoadNumber(vm.TypeInt, constant, reg)
				c.changeRegister(false, reg, reg, typ, dstType)
			case int32:
				constant := c.FB.MakeIntConstant(int64(v))
				c.FB.LoadNumber(vm.TypeInt, constant, reg)
				c.changeRegister(false, reg, reg, typ, dstType)
			case int64:
				constant := c.FB.MakeIntConstant(int64(v))
				c.FB.LoadNumber(vm.TypeInt, constant, reg)
				c.changeRegister(false, reg, reg, typ, dstType)
			case uint:
				constant := c.FB.MakeIntConstant(int64(v))
				c.FB.LoadNumber(vm.TypeInt, constant, reg)
				c.changeRegister(false, reg, reg, typ, dstType)
			case uint8:
				constant := c.FB.MakeIntConstant(int64(v))
				c.FB.LoadNumber(vm.TypeInt, constant, reg)
				c.changeRegister(false, reg, reg, typ, dstType)
			case uint16:
				constant := c.FB.MakeIntConstant(int64(v))
				c.FB.LoadNumber(vm.TypeInt, constant, reg)
				c.changeRegister(false, reg, reg, typ, dstType)
			case uint32:
				constant := c.FB.MakeIntConstant(int64(v))
				c.FB.LoadNumber(vm.TypeInt, constant, reg)
				c.changeRegister(false, reg, reg, typ, dstType)
			case uint64:
				constant := c.FB.MakeIntConstant(int64(v))
				c.FB.LoadNumber(vm.TypeInt, constant, reg)
				c.changeRegister(false, reg, reg, typ, dstType)
			case string:
				constant := c.FB.MakeStringConstant(v)
				c.changeRegister(true, constant, reg, typ, dstType)
			case float32:
				constant := c.FB.MakeFloatConstant(float64(v))
				c.FB.LoadNumber(vm.TypeFloat, constant, reg)
			case float64:
				constant := c.FB.MakeFloatConstant(v)
				c.FB.LoadNumber(vm.TypeFloat, constant, reg)
				c.changeRegister(false, reg, reg, typ, dstType)
			default:
				constant := c.FB.MakeGeneralConstant(v)
				c.changeRegister(true, constant, reg, typ, dstType)
			}
		}

	case *ast.Index:
		exprType := c.TypeInfo[expr.Expr].Type
		var exprReg int8
		out, _, isRegister := c.quickEmitExpr(expr.Expr, exprType)
		if isRegister {
			exprReg = out
		} else {
			exprReg = c.FB.NewRegister(exprType.Kind())
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
			i = c.FB.NewRegister(reflect.Int)
			c.emitExpr(expr.Index, i, dstType)
		}
		c.FB.Index(ki, exprReg, i, reg, exprType)

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
	if kindToType(expectedType.Kind()) != kindToType(c.TypeInfo[expr].Type.Kind()) {
		return 0, false, false
	}
	switch expr := expr.(type) {
	case *ast.Identifier:
		if c.FB.IsVariable(expr.Name) {
			return c.FB.ScopeLookup(expr.Name), false, true
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
		typ := c.TypeInfo[call.Args[0]].Type
		s := c.FB.NewRegister(typ.Kind())
		c.emitExpr(call.Args[0], s, typ)
		c.FB.Cap(s, reg)
		c.changeRegister(false, reg, reg, intType, dstType)
	case "close":
		panic("TODO: not implemented")
	case "complex":
		panic("TODO: not implemented")
	case "copy":
		dst, _, isRegister := c.quickEmitExpr(call.Args[0], c.TypeInfo[call.Args[0]].Type)
		if !isRegister {
			dst = c.FB.NewRegister(reflect.Slice)
			c.emitExpr(call.Args[0], dst, c.TypeInfo[call.Args[0]].Type)
		}
		src, _, isRegister := c.quickEmitExpr(call.Args[1], c.TypeInfo[call.Args[1]].Type)
		if !isRegister {
			src = c.FB.NewRegister(reflect.Slice)
			c.emitExpr(call.Args[0], src, c.TypeInfo[call.Args[0]].Type)
		}
		c.FB.Copy(dst, src, reg)
		if reg != 0 {
			c.changeRegister(false, reg, reg, intType, dstType)
		}
	case "delete":
		mapExpr := call.Args[0]
		keyExpr := call.Args[1]
		mapp := c.FB.NewRegister(reflect.Interface)
		c.emitExpr(mapExpr, mapp, emptyInterfaceType)
		key := c.FB.NewRegister(reflect.Interface)
		c.emitExpr(keyExpr, key, emptyInterfaceType)
		c.FB.Delete(mapp, key)
	case "imag":
		panic("TODO: not implemented")
	case "html": // TODO (Gianluca): to review.
		panic("TODO: not implemented")
	case "len":
		typ := c.TypeInfo[call.Args[0]].Type
		s := c.FB.NewRegister(typ.Kind())
		c.emitExpr(call.Args[0], s, typ)
		c.FB.Len(s, reg, typ)
		c.changeRegister(false, reg, reg, intType, dstType)
	case "make":
		typ := call.Args[0].(*ast.Value).Val.(reflect.Type)
		regType := c.FB.Type(typ)
		switch typ.Kind() {
		case reflect.Map:
			if len(call.Args) == 1 {
				c.FB.MakeMap(regType, true, 0, reg)
			} else {
				size, kSize, isRegister := c.quickEmitExpr(call.Args[1], intType)
				if !kSize && !isRegister {
					size = c.FB.NewRegister(reflect.Int)
					c.emitExpr(call.Args[1], size, c.TypeInfo[call.Args[1]].Type)
				}
				c.FB.MakeMap(regType, kSize, size, reg)
			}
		case reflect.Slice:
			lenExpr := call.Args[1]
			capExpr := call.Args[2]
			len, kLen, isRegister := c.quickEmitExpr(lenExpr, intType)
			if !kLen && !isRegister {
				len = c.FB.NewRegister(reflect.Int)
				c.emitExpr(lenExpr, len, c.TypeInfo[lenExpr].Type)
			}
			cap, kCap, isRegister := c.quickEmitExpr(capExpr, intType)
			if !kCap && !isRegister {
				cap = c.FB.NewRegister(reflect.Int)
				c.emitExpr(capExpr, cap, c.TypeInfo[capExpr].Type)
			}
			c.FB.MakeSlice(kLen, kCap, typ, len, cap, reg)
		case reflect.Chan:
			chanType := c.TypeInfo[call.Args[0]].Type
			chanTypeIndex := c.FB.AddType(chanType)
			var kCapacity bool
			var capacity int8
			if len(call.Args) == 1 {
				capacity = 0
				kCapacity = true
			} else {
				var isRegister bool
				capacity, kCapacity, isRegister = c.quickEmitExpr(call.Args[1], intType)
				if !kCapacity && !isRegister {
					capacity = c.FB.NewRegister(reflect.Int)
					c.emitExpr(call.Args[1], capacity, intType)
				}
			}
			c.FB.MakeChan(int8(chanTypeIndex), kCapacity, capacity, reg)
		default:
			panic("bug")
		}
	case "new":
		newType := call.Args[0].(*ast.Value).Val.(reflect.Type)
		c.FB.New(newType, reg)
	case "panic":
		arg := call.Args[0]
		reg, _, isRegister := c.quickEmitExpr(arg, emptyInterfaceType)
		if !isRegister {
			reg = c.FB.NewRegister(reflect.Interface)
			c.emitExpr(arg, reg, emptyInterfaceType)
		}
		c.FB.Panic(reg, call.Pos().Line)
	case "print":
		for i := range call.Args {
			arg := c.FB.NewRegister(reflect.Interface)
			c.emitExpr(call.Args[i], arg, emptyInterfaceType)
			c.FB.Print(arg)
		}
	case "println":
		panic("TODO: not implemented")
	case "real":
		panic("TODO: not implemented")
	case "recover":
		c.FB.Recover(reg)
	default:
		panic("unkown builtin") // TODO(Gianluca): remove.
	}
}

// EmitNodes emits instructions for nodes.
func (c *Emitter) EmitNodes(nodes []ast.Node) {
	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.Assignment:
			c.emitAssignmentNode(node)

		case *ast.Block:
			c.FB.EnterScope()
			c.EmitNodes(node.Nodes)
			c.FB.ExitScope()

		case *ast.Break:
			if node.Label != nil {
				panic("TODO(Gianluca): not implemented")
			}
			c.FB.Break(c.rangeLabels[len(c.rangeLabels)-1][0])
			c.FB.Goto(c.rangeLabels[len(c.rangeLabels)-1][1])

		case *ast.Continue:
			if node.Label != nil {
				panic("TODO(Gianluca): not implemented")
			}
			c.FB.Continue(c.rangeLabels[len(c.rangeLabels)-1][0])

		case *ast.Defer, *ast.Go:
			if def, ok := node.(*ast.Defer); ok {
				if c.TypeInfo[def.Call.Func].IsBuiltin() {
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
			funReg := c.FB.NewRegister(reflect.Func)
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
			funType := c.TypeInfo[funNode].Type
			c.emitExpr(funNode, funReg, c.TypeInfo[funNode].Type)
			offset := vm.StackShift{
				int8(c.FB.numRegs[reflect.Int]),
				int8(c.FB.numRegs[reflect.Float64]),
				int8(c.FB.numRegs[reflect.String]),
				int8(c.FB.numRegs[reflect.Interface]),
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
				c.FB.Defer(funReg, vm.NoVariadic, offset, argsShift)
			case *ast.Go:
				// TODO(Gianluca):
				c.FB.Go()
			}

		case *ast.If:
			c.FB.EnterScope()
			if node.Assignment != nil {
				c.EmitNodes([]ast.Node{node.Assignment})
			}
			c.emitCondition(node.Condition)
			if node.Else == nil { // TODO (Gianluca): can "then" and "else" be unified in some way?
				endIfLabel := c.FB.NewLabel()
				c.FB.Goto(endIfLabel)
				c.EmitNodes(node.Then.Nodes)
				c.FB.SetLabelAddr(endIfLabel)
			} else {
				elseLabel := c.FB.NewLabel()
				c.FB.Goto(elseLabel)
				c.EmitNodes(node.Then.Nodes)
				endIfLabel := c.FB.NewLabel()
				c.FB.Goto(endIfLabel)
				c.FB.SetLabelAddr(elseLabel)
				if node.Else != nil {
					switch els := node.Else.(type) {
					case *ast.If:
						c.EmitNodes([]ast.Node{els})
					case *ast.Block:
						c.EmitNodes(els.Nodes)
					}
				}
				c.FB.SetLabelAddr(endIfLabel)
			}
			c.FB.ExitScope()

		case *ast.For:
			c.FB.EnterScope()
			if node.Init != nil {
				c.EmitNodes([]ast.Node{node.Init})
			}
			if node.Condition != nil {
				forLabel := c.FB.NewLabel()
				c.FB.SetLabelAddr(forLabel)
				c.emitCondition(node.Condition)
				endForLabel := c.FB.NewLabel()
				c.FB.Goto(endForLabel)
				c.EmitNodes(node.Body)
				if node.Post != nil {
					c.EmitNodes([]ast.Node{node.Post})
				}
				c.FB.Goto(forLabel)
				c.FB.SetLabelAddr(endForLabel)
			} else {
				forLabel := c.FB.NewLabel()
				c.FB.SetLabelAddr(forLabel)
				c.EmitNodes(node.Body)
				if node.Post != nil {
					c.EmitNodes([]ast.Node{node.Post})
				}
				c.FB.Goto(forLabel)
			}
			c.FB.ExitScope()

		case *ast.ForRange:
			c.FB.EnterScope()
			vars := node.Assignment.Variables
			indexReg := int8(0)
			if len(vars) >= 1 && !isBlankIdentifier(vars[0]) {
				name := vars[0].(*ast.Identifier).Name
				if node.Assignment.Type == ast.AssignmentDeclaration {
					indexReg = c.FB.NewRegister(reflect.Int)
					c.FB.BindVarReg(name, indexReg)
				} else {
					indexReg = c.FB.ScopeLookup(name)
				}
			}
			elemReg := int8(0)
			if len(vars) == 2 && !isBlankIdentifier(vars[1]) {
				typ := c.TypeInfo[vars[1]].Type
				name := vars[1].(*ast.Identifier).Name
				if node.Assignment.Type == ast.AssignmentDeclaration {
					elemReg = c.FB.NewRegister(typ.Kind())
					c.FB.BindVarReg(name, elemReg)
				} else {
					elemReg = c.FB.ScopeLookup(name)
				}
			}
			expr := node.Assignment.Values[0]
			exprType := c.TypeInfo[expr].Type
			exprReg, kExpr, isRegister := c.quickEmitExpr(expr, exprType)
			if (!kExpr && !isRegister) || exprType.Kind() != reflect.String {
				kExpr = false
				exprReg = c.FB.NewRegister(exprType.Kind())
				c.emitExpr(expr, exprReg, exprType)
			}
			rangeLabel := c.FB.NewLabel()
			c.FB.SetLabelAddr(rangeLabel)
			endRange := c.FB.NewLabel()
			c.rangeLabels = append(c.rangeLabels, [2]uint32{rangeLabel, endRange})
			c.FB.Range(kExpr, exprReg, indexReg, elemReg, exprType.Kind())
			c.FB.Goto(endRange)
			c.FB.EnterScope()
			c.EmitNodes(node.Body)
			c.FB.Continue(rangeLabel)
			c.FB.SetLabelAddr(endRange)
			c.rangeLabels = c.rangeLabels[:len(c.rangeLabels)-1]
			c.FB.ExitScope()
			c.FB.ExitScope()

		case *ast.Return:
			// TODO(Gianluca): complete implementation of tail call optimization.
			// if len(node.Values) == 1 {
			// 	if call, ok := node.Values[0].(*ast.Call); ok {
			// 		tmpRegs := make([]int8, len(call.Args))
			// 		paramPosition := make([]int8, len(call.Args))
			// 		tmpTypes := make([]reflect.Type, len(call.Args))
			// 		shift := vm.StackShift{}
			// 		for i := range call.Args {
			// 			tmpTypes[i] = c.TypeInfo[call.Args[i]].Type
			// 			t := int(kindToType(tmpTypes[i].Kind()))
			// 			tmpRegs[i] = c.FB.NewRegister(tmpTypes[i].Kind())
			// 			shift[t]++
			// 			c.compileExpr(call.Args[i], tmpRegs[i], tmpTypes[i])
			// 			paramPosition[i] = shift[t]
			// 		}
			// 		for i := range call.Args {
			// 			c.changeRegister(false, tmpRegs[i], paramPosition[i], tmpTypes[i], c.TypeInfo[call.Func].Type.In(i))
			// 		}
			// 		c.FB.TailCall(vm.CurrentFunction, node.Pos().Line)
			// 		continue
			// 	}
			// }
			offset := [4]int8{}
			for i, v := range node.Values {
				typ := c.CurrentFunction.Type.Out(i)
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
			c.FB.Return()

		case *ast.Send:
			ch := c.FB.NewRegister(reflect.Chan)
			c.emitExpr(node.Channel, ch, c.TypeInfo[node.Channel].Type)
			elemType := c.TypeInfo[node.Value].Type
			v := c.FB.NewRegister(elemType.Kind())
			c.emitExpr(node.Value, v, elemType)
			c.FB.Send(ch, v)

		case *ast.Switch:
			c.emitSwitch(node)

		case *ast.TypeSwitch:
			c.emitTypeSwitch(node)

		case *ast.Var:
			addresses := make([]Address, len(node.Identifiers))
			for i, v := range node.Identifiers {
				varType := c.TypeInfo[v].Type
				if c.IndirectVars[v] {
					varReg := -c.FB.NewRegister(reflect.Interface)
					c.FB.BindVarReg(v.Name, varReg)
					addresses[i] = c.NewAddress(AddressIndirectDeclaration, varType, varReg, 0)
				} else {
					varReg := c.FB.NewRegister(varType.Kind())
					c.FB.BindVarReg(v.Name, varReg)
					addresses[i] = c.NewAddress(AddressRegister, varType, varReg, 0)
				}
			}
			c.assign(addresses, node.Values)

		case ast.Expression:
			// TODO (Gianluca): use 0 (which is no longer a valid
			// register) and handle it as a special case in emitExpr.
			c.emitExpr(node, 0, reflect.Type(nil))

		default:
			panic(fmt.Sprintf("node %T not supported", node)) // TODO(Gianluca): remove.

		}
	}
}

// emitTypeSwitch emits instructions for a type switch node.
func (c *Emitter) emitTypeSwitch(node *ast.TypeSwitch) {
	// TODO (Gianluca): a type-checker bug does not replace type switch type with proper value.

	c.FB.EnterScope()

	if node.Init != nil {
		c.EmitNodes([]ast.Node{node.Init})
	}

	typAss := node.Assignment.Values[0].(*ast.TypeAssertion)
	typ := c.TypeInfo[typAss.Expr].Type
	expr := c.FB.NewRegister(typ.Kind())
	c.emitExpr(typAss.Expr, expr, typ)

	// typ := node.Assignment.Values[0].(*ast.Value).Val.(reflect.Type)
	// typReg := c.FB.Type(typ)
	// if variab := node.Assignment.Globals[0]; !isBlankast.Identifier(variab) {
	// 	c.compileVarsGetValue([]ast.Expression{variab}, node.Assignment.Values[0], node.Assignment.Type == ast.AssignmentDeclaration)
	// }

	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := c.FB.NewLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = c.FB.NewLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			caseType := caseExpr.(*ast.Value).Val.(reflect.Type)
			c.FB.Assert(expr, caseType, 0)
			next := c.FB.NewLabel()
			c.FB.Goto(next)
			c.FB.Goto(bodyLabels[i])
			c.FB.SetLabelAddr(next)
		}
	}

	if hasDefault {
		defaultLabel = c.FB.NewLabel()
		c.FB.Goto(defaultLabel)
	} else {
		c.FB.Goto(endSwitchLabel)
	}

	for i, cas := range node.Cases {
		if cas.Expressions == nil {
			c.FB.SetLabelAddr(defaultLabel)
		}
		c.FB.SetLabelAddr(bodyLabels[i])
		c.FB.EnterScope()
		c.EmitNodes(cas.Body)
		c.FB.ExitScope()
		c.FB.Goto(endSwitchLabel)
	}

	c.FB.SetLabelAddr(endSwitchLabel)
	c.FB.ExitScope()
}

// emitSwitch emits instructions for a switch node.
func (c *Emitter) emitSwitch(node *ast.Switch) {

	c.FB.EnterScope()

	if node.Init != nil {
		c.EmitNodes([]ast.Node{node.Init})
	}

	typ := c.TypeInfo[node.Expr].Type
	expr := c.FB.NewRegister(typ.Kind())
	c.emitExpr(node.Expr, expr, typ)
	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := c.FB.NewLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = c.FB.NewLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			y, ky, isRegister := c.quickEmitExpr(caseExpr, typ)
			if !ky && !isRegister {
				y = c.FB.NewRegister(typ.Kind())
				c.emitExpr(caseExpr, y, typ)
			}
			c.FB.If(ky, expr, vm.ConditionNotEqual, y, typ.Kind()) // Condizione negata per poter concatenare gli if
			c.FB.Goto(bodyLabels[i])
		}
	}

	if hasDefault {
		defaultLabel = c.FB.NewLabel()
		c.FB.Goto(defaultLabel)
	} else {
		c.FB.Goto(endSwitchLabel)
	}

	for i, cas := range node.Cases {
		if cas.Expressions == nil {
			c.FB.SetLabelAddr(defaultLabel)
		}
		c.FB.SetLabelAddr(bodyLabels[i])
		c.FB.EnterScope()
		c.EmitNodes(cas.Body)
		if !cas.Fallthrough {
			c.FB.Goto(endSwitchLabel)
		}
		c.FB.ExitScope()
	}

	c.FB.SetLabelAddr(endSwitchLabel)

	c.FB.ExitScope()
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
			exprType := c.TypeInfo[expr].Type
			x, _, isRegister := c.quickEmitExpr(expr, exprType)
			if !isRegister {
				x = c.FB.NewRegister(exprType.Kind())
				c.emitExpr(expr, x, exprType)
			}
			condType := vm.ConditionNotNil
			if cond.Operator() == ast.OperatorEqual {
				condType = vm.ConditionNil
			}
			c.FB.If(false, x, condType, 0, exprType.Kind())
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
			if c.TypeInfo[lenArg].Type.Kind() == reflect.String { // len is optimized for strings only.
				lenArgType := c.TypeInfo[lenArg].Type
				x, _, isRegister := c.quickEmitExpr(lenArg, lenArgType)
				if !isRegister {
					x = c.FB.NewRegister(lenArgType.Kind())
					c.emitExpr(lenArg, x, lenArgType)
				}
				exprType := c.TypeInfo[expr].Type
				y, ky, isRegister := c.quickEmitExpr(expr, exprType)
				if !ky && !isRegister {
					y = c.FB.NewRegister(exprType.Kind())
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
				c.FB.If(ky, x, condType, y, reflect.String)
				return
			}
		}

		// if v1 == v2
		// if v1 != v2
		// if v1 <  v2
		// if v1 <= v2
		// if v1 >  v2
		// if v1 >= v2
		expr1Type := c.TypeInfo[cond.Expr1].Type
		expr2Type := c.TypeInfo[cond.Expr2].Type
		if expr1Type.Kind() == expr2Type.Kind() {
			switch kind := expr1Type.Kind(); kind {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64,
				reflect.String:
				expr1Type := c.TypeInfo[cond.Expr1].Type
				x, _, isRegister := c.quickEmitExpr(cond.Expr1, expr1Type)
				if !isRegister {
					x = c.FB.NewRegister(expr1Type.Kind())
					c.emitExpr(cond.Expr1, x, expr1Type)
				}
				expr2Type := c.TypeInfo[cond.Expr2].Type
				y, ky, isRegister := c.quickEmitExpr(cond.Expr2, expr2Type)
				if !ky && !isRegister {
					y = c.FB.NewRegister(expr2Type.Kind())
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
				c.FB.If(ky, x, condType, y, kind)
				return
			}
		}

	default:

		condType := c.TypeInfo[cond].Type
		x, _, isRegister := c.quickEmitExpr(cond, condType)
		if !isRegister {
			x = c.FB.NewRegister(condType.Kind())
			c.emitExpr(cond, x, condType)
		}
		yConst := c.FB.MakeIntConstant(1)
		y := c.FB.NewRegister(reflect.Bool)
		c.FB.LoadNumber(vm.TypeInt, yConst, y)
		c.FB.If(false, x, vm.ConditionEqual, y, reflect.Bool)

	}

}
