// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"

	"scrigo/internal/compiler/ast"
	"scrigo/native"
	"scrigo/vm"
)

// An Emitter emits instructions for the VM.
type Emitter struct {
	CurrentFunction  *vm.ScrigoFunction
	TypeInfo         map[ast.Node]*TypeInfo
	FB               *FunctionBuilder
	importableGoPkgs map[string]*native.GoPackage
	IndirectVars     map[*ast.Identifier]bool

	upvarsNames map[*vm.ScrigoFunction]map[string]int

	availableScrigoFunctions map[string]*vm.ScrigoFunction
	availableNativeFunctions map[string]*vm.NativeFunction

	assignedScrigoFunctions map[*vm.ScrigoFunction]map[*vm.ScrigoFunction]int8
	assignedNativeFunctions map[*vm.ScrigoFunction]map[*vm.NativeFunction]int8

	isNativePkg map[string]bool

	// rangeLabels is a list of current active Ranges. First element is the
	// Range address, second referrs to the first instruction outside Range's
	// body.
	rangeLabels [][2]uint32

	globals         []vm.Global      // holds all Scrigo and native global variables.
	globalNameIndex map[string]int16 // maps global variable names to their index inside globals.

	labels map[*vm.ScrigoFunction]map[string]uint32
}

// Main returns main function.
func (e *Emitter) Main() *vm.ScrigoFunction {
	return e.availableScrigoFunctions["main"]
}

// NewEmitter returns a new emitter reading sources from r.
// Native packages are made available for importing.
func NewEmitter(packages map[string]*native.GoPackage, typeInfos map[ast.Node]*TypeInfo, indirectVars map[*ast.Identifier]bool) *Emitter {
	c := &Emitter{

		importableGoPkgs: packages,

		upvarsNames: make(map[*vm.ScrigoFunction]map[string]int),

		availableScrigoFunctions: map[string]*vm.ScrigoFunction{},
		availableNativeFunctions: map[string]*vm.NativeFunction{},

		assignedScrigoFunctions: map[*vm.ScrigoFunction]map[*vm.ScrigoFunction]int8{},
		assignedNativeFunctions: map[*vm.ScrigoFunction]map[*vm.NativeFunction]int8{},

		isNativePkg: map[string]bool{},

		globalNameIndex: map[string]int16{},

		TypeInfo:     typeInfos,
		IndirectVars: indirectVars,

		labels: make(map[*vm.ScrigoFunction]map[string]uint32),
	}
	return c
}

// Global represents a global variable with a package, name, type (only for
// Scrigo globals) and value (only for native globals). Value, if present, must
// be a pointer to the variable value.
type Global struct {
	Pkg   string
	Name  string
	Type  reflect.Type
	Value interface{}
}

// emitPackage emits pkg.
func EmitPackage(pkg *ast.Package, packages map[string]*native.GoPackage, typeInfos map[ast.Node]*TypeInfo, indirectVars map[*ast.Identifier]bool) (*vm.ScrigoFunction, []vm.Global) {

	e := &Emitter{

		importableGoPkgs: packages,

		upvarsNames: make(map[*vm.ScrigoFunction]map[string]int),

		availableScrigoFunctions: map[string]*vm.ScrigoFunction{},
		availableNativeFunctions: map[string]*vm.NativeFunction{},

		assignedScrigoFunctions: map[*vm.ScrigoFunction]map[*vm.ScrigoFunction]int8{},
		assignedNativeFunctions: map[*vm.ScrigoFunction]map[*vm.NativeFunction]int8{},

		isNativePkg: map[string]bool{},

		globalNameIndex: map[string]int16{},

		TypeInfo:     typeInfos,
		IndirectVars: indirectVars,

		labels: make(map[*vm.ScrigoFunction]map[string]uint32),
	}

	// Emits imports.
	for _, decl := range pkg.Declarations {
		if imp, ok := decl.(*ast.Import); ok {
			e.emitImport(imp)
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
				e.availableScrigoFunctions[fun.Ident.Name] = fn
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
			backupFn := e.CurrentFunction
			backupFb := e.FB
			if initVarsFn == nil {
				initVarsFn = NewScrigoFunction("main", "$initvars", reflect.FuncOf(nil, nil, false))
				e.availableScrigoFunctions["$initvars"] = initVarsFn
				initVarsFb = NewBuilder(initVarsFn)
				initVarsFb.EnterScope()
			}
			e.CurrentFunction = initVarsFn
			e.FB = initVarsFb
			addresses := make([]Address, len(n.Identifiers))
			for i, v := range n.Identifiers {
				varType := e.TypeInfo[v].Type
				varReg := -e.FB.NewRegister(reflect.Interface)
				e.FB.BindVarReg(v.Name, varReg)
				addresses[i] = e.NewAddress(AddressIndirectDeclaration, varType, varReg, 0)
				packageVariablesRegisters[v.Name] = varReg
				e.globals = append(e.globals, vm.Global{Pkg: "main", Name: v.Name, Type: varType})
				e.globalNameIndex[v.Name] = int16(len(e.globals) - 1)
			}
			e.assign(addresses, n.Values)
			e.CurrentFunction = backupFn
			e.FB = backupFb
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
				fn = e.availableScrigoFunctions[n.Ident.Name]
			}
			e.CurrentFunction = fn
			e.FB = NewBuilder(fn)
			e.FB.EnterScope()

			// If function is "main", variables initialization function
			// must be called as first statement inside main.
			if n.Ident.Name == "main" {
				// First: initializes package variables.
				if initVarsFn != nil {
					iv := e.availableScrigoFunctions["$initvars"]
					index := e.FB.AddScrigoFunction(iv)
					e.FB.Call(int8(index), vm.StackShift{}, 0)
				}
				// Second: calls all init functions, in order.
				for _, initFunc := range initFuncs {
					index := e.FB.AddScrigoFunction(initFunc)
					e.FB.Call(int8(index), vm.StackShift{}, 0)
				}
			}
			e.prepareFunctionBodyParameters(n)
			AddExplicitReturn(n)
			e.EmitNodes(n.Body.Nodes)
			e.FB.End()
			e.FB.ExitScope()
		}
	}

	if initVarsFn != nil {
		// Global variables have been locally defined inside the "$initvars"
		// function; their values must now be exported to be available
		// globally.
		backupFn := e.CurrentFunction
		backupFb := e.FB
		e.CurrentFunction = initVarsFn
		e.FB = initVarsFb
		for name, reg := range packageVariablesRegisters {
			index := e.globalNameIndex[name]
			e.FB.SetVar(false, reg, int(index))
		}
		e.CurrentFunction = backupFn
		e.FB = backupFb
		initVarsFb.ExitScope()
		initVarsFb.Return()
	}

	// Assigns globals to main's Globals.
	main := e.availableScrigoFunctions["main"]
	main.Globals = e.globals
	// All functions share Globals.
	for _, f := range e.availableScrigoFunctions {
		f.Globals = main.Globals
	}

	return main, e.globals
}

// prepareCallParameters prepares parameters (out and in) for a function call of
// type funcType and arguments args. Returns the list of return registers and
// their respective type.
func (e *Emitter) prepareCallParameters(funcType reflect.Type, args []ast.Expression, isNative bool) ([]int8, []reflect.Type) {
	numOut := funcType.NumOut()
	numIn := funcType.NumIn()
	regs := make([]int8, numOut)
	types := make([]reflect.Type, numOut)
	for i := 0; i < numOut; i++ {
		typ := funcType.Out(i)
		regs[i] = e.FB.NewRegister(typ.Kind())
		types[i] = typ
	}
	if funcType.IsVariadic() {
		for i := 0; i < numIn-1; i++ {
			typ := funcType.In(i)
			reg := e.FB.NewRegister(typ.Kind())
			e.FB.EnterScope()
			e.emitExpr(args[i], reg, typ)
			e.FB.ExitScope()
		}
		if varArgs := len(args) - (numIn - 1); varArgs > 0 {
			typ := funcType.In(numIn - 1).Elem()
			if isNative {
				for i := 0; i < varArgs; i++ {
					reg := e.FB.NewRegister(typ.Kind())
					e.FB.EnterStack()
					e.emitExpr(args[i+numIn-1], reg, typ)
					e.FB.ExitStack()
				}
			} else {
				sliceReg := int8(numIn)
				e.FB.MakeSlice(true, true, funcType.In(numIn-1), int8(varArgs), int8(varArgs), sliceReg)
				for i := 0; i < varArgs; i++ {
					tmpReg := e.FB.NewRegister(typ.Kind())
					e.FB.EnterStack()
					e.emitExpr(args[i+numIn-1], tmpReg, typ)
					e.FB.ExitStack()
					indexReg := e.FB.NewRegister(reflect.Int)
					e.FB.Move(true, int8(i), indexReg, reflect.Int)
					e.FB.SetSlice(false, sliceReg, tmpReg, indexReg, typ.Kind())
				}
			}
		}
	} else { // No-variadic function.
		if numIn > 1 && len(args) == 1 { // f(g()), where f takes more than 1 argument.
			regs, types := e.emitCall(args[0].(*ast.Call))
			for i := range regs {
				dstType := funcType.In(i)
				reg := e.FB.NewRegister(dstType.Kind())
				e.changeRegister(false, regs[i], reg, types[i], dstType)
			}
		} else {
			for i := 0; i < numIn; i++ {
				typ := funcType.In(i)
				reg := e.FB.NewRegister(typ.Kind())
				e.FB.EnterStack()
				e.emitExpr(args[i], reg, typ)
				e.FB.ExitStack()
			}
		}
	}
	return regs, types
}

// prepareFunctionBodyParameters prepares fun's parameters (out and int) before
// emitting its body.
func (e *Emitter) prepareFunctionBodyParameters(fun *ast.Func) {
	// Reserves space for return parameters.
	fillParametersTypes(fun.Type.Result)
	for _, res := range fun.Type.Result {
		resType := res.Type.(*ast.Value).Val.(reflect.Type)
		kind := resType.Kind()
		retReg := e.FB.NewRegister(kind)
		if res.Ident != nil {
			e.FB.BindVarReg(res.Ident.Name, retReg)
		}
	}
	// Binds function argument names to pre-allocated registers.
	fillParametersTypes(fun.Type.Parameters)
	for _, par := range fun.Type.Parameters {
		parType := par.Type.(*ast.Value).Val.(reflect.Type)
		kind := parType.Kind()
		argReg := e.FB.NewRegister(kind)
		if par.Ident != nil {
			e.FB.BindVarReg(par.Ident.Name, argReg)
		}
	}
}

// emitCall emits instruction for a call, returning the list of registers (and
// their respective type) within which return values are inserted.
func (e *Emitter) emitCall(call *ast.Call) ([]int8, []reflect.Type) {
	stackShift := vm.StackShift{
		int8(e.FB.numRegs[reflect.Int]),
		int8(e.FB.numRegs[reflect.Float64]),
		int8(e.FB.numRegs[reflect.String]),
		int8(e.FB.numRegs[reflect.Interface]),
	}
	if ident, ok := call.Func.(*ast.Identifier); ok {
		if !e.FB.IsVariable(ident.Name) {
			if fun, isScrigoFunc := e.availableScrigoFunctions[ident.Name]; isScrigoFunc {
				regs, types := e.prepareCallParameters(fun.Type, call.Args, false)
				index := e.scrigoFunctionIndex(fun)
				e.FB.Call(index, stackShift, call.Pos().Line)
				return regs, types
			}
			if _, isNativeFunc := e.availableNativeFunctions[ident.Name]; isNativeFunc {
				fun := e.availableNativeFunctions[ident.Name]
				funcType := reflect.TypeOf(fun.Func)
				regs, types := e.prepareCallParameters(funcType, call.Args, true)
				index := e.nativeFunctionIndex(fun)
				if funcType.IsVariadic() {
					numVar := len(call.Args) - (funcType.NumIn() - 1)
					e.FB.CallNative(index, int8(numVar), stackShift)
				} else {
					e.FB.CallNative(index, vm.NoVariadic, stackShift)
				}
				return regs, types
			}
		}
	}
	if sel, ok := call.Func.(*ast.Selector); ok {
		if name, ok := sel.Expr.(*ast.Identifier); ok {
			if isGoPkg := e.isNativePkg[name.Name]; isGoPkg {
				fun := e.availableNativeFunctions[name.Name+"."+sel.Ident]
				funcType := reflect.TypeOf(fun.Func)
				regs, types := e.prepareCallParameters(funcType, call.Args, true)
				index := e.nativeFunctionIndex(fun)
				if funcType.IsVariadic() {
					numVar := len(call.Args) - (funcType.NumIn() - 1)
					e.FB.CallNative(index, int8(numVar), stackShift)
				} else {
					e.FB.CallNative(index, vm.NoVariadic, stackShift)
				}
				return regs, types
			} else {
				panic("TODO(Gianluca): not implemented")
			}
		}
	}
	funReg, _, isRegister := e.quickEmitExpr(call.Func, e.TypeInfo[call.Func].Type)
	if !isRegister {
		funReg = e.FB.NewRegister(reflect.Func)
		e.emitExpr(call.Func, funReg, e.TypeInfo[call.Func].Type)
	}
	funcType := e.TypeInfo[call.Func].Type
	regs, types := e.prepareCallParameters(funcType, call.Args, true)
	e.FB.CallIndirect(funReg, 0, stackShift)
	return regs, types
}

// emitExpr emits instruction such that expr value is put into reg. If reg is
// zero instructions are emitted anyway but result is discarded.
func (e *Emitter) emitExpr(expr ast.Expression, reg int8, dstType reflect.Type) {
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
			e.FB.EnterStack()
			e.emitExpr(expr.Expr1, reg, dstType)
			endIf := e.FB.NewLabel()
			e.FB.If(true, reg, vm.ConditionEqual, cmp, reflect.Int)
			e.FB.Goto(endIf)
			e.emitExpr(expr.Expr2, reg, dstType)
			e.FB.SetLabelAddr(endIf)
			e.FB.ExitStack()
			return
		}

		e.FB.EnterStack()

		xType := e.TypeInfo[expr.Expr1].Type
		x := e.FB.NewRegister(xType.Kind())
		e.emitExpr(expr.Expr1, x, xType)

		y, ky, isRegister := e.quickEmitExpr(expr.Expr2, xType)
		if !ky && !isRegister {
			y = e.FB.NewRegister(xType.Kind())
			e.emitExpr(expr.Expr2, y, xType)
		}

		res := e.FB.NewRegister(xType.Kind())

		switch op := expr.Operator(); {
		case op == ast.OperatorAddition && xType.Kind() == reflect.String && reg != 0:
			if ky {
				y = e.FB.NewRegister(reflect.String)
				e.emitExpr(expr.Expr2, y, xType)
			}
			e.FB.Concat(x, y, reg)
		case op == ast.OperatorAddition && reg != 0:
			e.FB.Add(ky, x, y, res, xType.Kind())
			e.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorSubtraction && reg != 0:
			e.FB.Sub(ky, x, y, res, xType.Kind())
			e.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorMultiplication && reg != 0:
			e.FB.Mul(ky, x, y, res, xType.Kind())
			e.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorDivision && reg != 0:
			e.FB.Div(ky, x, y, res, xType.Kind())
			e.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorDivision && reg == 0:
			dummyReg := e.FB.NewRegister(xType.Kind())
			e.FB.Div(ky, x, y, dummyReg, xType.Kind()) // produces division by zero.
		case op == ast.OperatorModulo && reg != 0:
			e.FB.Rem(ky, x, y, res, xType.Kind())
			e.changeRegister(false, res, reg, xType, dstType)
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
				e.FB.Move(true, 1, reg, reflect.Bool)
				e.FB.If(ky, x, cond, y, xType.Kind())
				e.FB.Move(true, 0, reg, reflect.Bool)
			}
		case op == ast.OperatorOr,
			op == ast.OperatorAnd,
			op == ast.OperatorXor,
			op == ast.OperatorAndNot,
			op == ast.OperatorLeftShift,
			op == ast.OperatorRightShift:
			if reg != 0 {
				e.FB.BinaryBitOperation(op, ky, x, y, reg, xType.Kind())
				if kindToType(xType.Kind()) != kindToType(dstType.Kind()) {
					e.changeRegister(ky, reg, reg, xType, dstType)
				}
			}
		default:
			panic(fmt.Errorf("TODO: not implemented operator %s", expr.Operator()))
		}

		e.FB.ExitStack()

	case *ast.Call:
		// Builtin call.
		e.FB.EnterStack()
		if e.TypeInfo[expr.Func].IsBuiltin() {
			e.emitBuiltin(expr, reg, dstType)
			e.FB.ExitStack()
			return
		}
		// Conversion.
		if val, ok := expr.Func.(*ast.Value); ok {
			if convertType, ok := val.Val.(reflect.Type); ok {
				typ := e.TypeInfo[expr.Args[0]].Type
				arg := e.FB.NewRegister(typ.Kind())
				e.emitExpr(expr.Args[0], arg, typ)
				e.FB.Convert(arg, convertType, reg, typ.Kind())
				e.FB.ExitStack()
				return
			}
		}
		regs, types := e.emitCall(expr)
		if reg != 0 {
			e.changeRegister(false, regs[0], reg, types[0], dstType)
		}
		e.FB.ExitStack()

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
				e.FB.MakeSlice(true, true, typ, size, size, reg)
			}
			var index int8 = -1
			for _, kv := range expr.KeyValues {
				if kv.Key != nil {
					index = int8(kv.Key.(*ast.Value).Val.(int))
				} else {
					index++
				}
				e.FB.EnterStack()
				indexReg := e.FB.NewRegister(reflect.Int)
				e.FB.Move(true, index, indexReg, reflect.Int)
				value, kvalue, isRegister := e.quickEmitExpr(kv.Value, typ.Elem())
				if !kvalue && !isRegister {
					value = e.FB.NewRegister(typ.Elem().Kind())
					e.emitExpr(kv.Value, value, typ.Elem())
				}
				if reg != 0 {
					e.FB.SetSlice(kvalue, reg, value, indexReg, typ.Elem().Kind())
				}
				e.FB.ExitStack()
			}
		case reflect.Struct:
			panic("TODO: not implemented")
		case reflect.Map:
			// TODO (Gianluca): handle maps with bigger size.
			size := len(expr.KeyValues)
			sizeReg := e.FB.MakeIntConstant(int64(size))
			regType := e.FB.Type(typ)
			e.FB.MakeMap(regType, true, sizeReg, reg)
			for _, kv := range expr.KeyValues {
				keyReg := e.FB.NewRegister(typ.Key().Kind())
				valueReg := e.FB.NewRegister(typ.Elem().Kind())
				e.FB.EnterStack()
				e.emitExpr(kv.Key, keyReg, typ.Key())
				kValue := false // TODO(Gianluca).
				e.emitExpr(kv.Value, valueReg, typ.Elem())
				e.FB.ExitStack()
				e.FB.SetMap(kValue, reg, valueReg, keyReg)
			}
		}

	case *ast.TypeAssertion:
		typ := e.TypeInfo[expr.Expr].Type
		exprReg, _, isRegister := e.quickEmitExpr(expr.Expr, typ)
		if !isRegister {
			exprReg = e.FB.NewRegister(typ.Kind())
			e.emitExpr(expr.Expr, exprReg, typ)
		}
		assertType := expr.Type.(*ast.Value).Val.(reflect.Type)
		e.FB.Assert(exprReg, assertType, reg)
		e.FB.Nop()

	case *ast.Selector:
		if index, ok := e.globalNameIndex[expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			e.FB.GetVar(int(index), reg) // TODO (Gianluca): to review.
			return
		}
		if nf, ok := e.availableNativeFunctions[expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := e.nativeFunctionIndex(nf)
			e.FB.GetFunc(true, index, reg)
			return
		}
		if sf, ok := e.availableScrigoFunctions[expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := e.scrigoFunctionIndex(sf)
			e.FB.GetFunc(false, index, reg)
			return
		}
		exprType := e.TypeInfo[expr.Expr].Type
		exprReg := e.FB.NewRegister(exprType.Kind())
		e.emitExpr(expr.Expr, exprReg, exprType)
		field := int8(0) // TODO(Gianluca).
		e.FB.Selector(exprReg, field, reg)

	case *ast.UnaryOperator:
		typ := e.TypeInfo[expr.Expr].Type
		var tmpReg int8
		if reg != 0 {
			tmpReg = e.FB.NewRegister(typ.Kind())
		}
		e.emitExpr(expr.Expr, tmpReg, typ)
		if reg == 0 {
			return
		}
		switch expr.Operator() {
		case ast.OperatorNot:
			e.FB.SubInv(true, tmpReg, int8(1), tmpReg, reflect.Int)
			if reg != 0 {
				e.changeRegister(false, tmpReg, reg, typ, dstType)
			}
		case ast.OperatorMultiplication:
			e.changeRegister(false, -tmpReg, reg, typ.Elem(), dstType)
		case ast.OperatorAnd:
			switch expr := expr.Expr.(type) {
			case *ast.Identifier:
				if e.FB.IsVariable(expr.Name) {
					varReg := e.FB.ScopeLookup(expr.Name)
					e.FB.New(reflect.PtrTo(typ), reg)
					e.FB.Move(false, -varReg, reg, dstType.Kind())
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
			e.FB.SubInv(true, tmpReg, 0, tmpReg, dstType.Kind())
			if reg != 0 {
				e.changeRegister(false, tmpReg, reg, typ, dstType)
			}
		case ast.OperatorReceive:
			e.FB.Receive(tmpReg, 0, reg)
		default:
			panic(fmt.Errorf("TODO: not implemented operator %s", expr.Operator()))
		}

	case *ast.Func:
		if reg == 0 {
			return
		}

		fn := e.FB.Func(reg, e.TypeInfo[expr].Type)
		e.setClosureRefs(fn, expr.Upvars)

		funcLitBuilder := NewBuilder(fn)
		currFb := e.FB
		currFn := e.CurrentFunction
		e.FB = funcLitBuilder
		e.CurrentFunction = fn

		e.FB.EnterScope()
		e.prepareFunctionBodyParameters(expr)
		AddExplicitReturn(expr)
		e.EmitNodes(expr.Body.Nodes)
		e.FB.ExitScope()
		e.FB = currFb
		e.CurrentFunction = currFn

	case *ast.Identifier:
		if reg == 0 {
			return
		}
		typ := e.TypeInfo[expr].Type
		out, isValue, isRegister := e.quickEmitExpr(expr, typ)
		if isValue {
			e.changeRegister(true, out, reg, typ, dstType)
		} else if isRegister {
			e.changeRegister(false, out, reg, typ, dstType)
		} else {
			if fun, isScrigoFunc := e.availableScrigoFunctions[expr.Name]; isScrigoFunc {
				index := e.scrigoFunctionIndex(fun)
				e.FB.GetFunc(false, index, reg)
			} else if index, ok := e.upvarsNames[e.CurrentFunction][expr.Name]; ok {
				// TODO(Gianluca): this is an experimental handling of
				// emitting an expression into a register of a different
				// type. If this is correct, apply this solution to all
				// other expression emitting cases or generalize in some
				// way.
				if kindToType(typ.Kind()) == kindToType(dstType.Kind()) {
					e.FB.GetVar(index, reg)
				} else {
					tmpReg := e.FB.NewRegister(typ.Kind())
					e.FB.GetVar(index, tmpReg)
					e.changeRegister(false, tmpReg, reg, typ, dstType)
				}
			} else if index, ok := e.globalNameIndex[expr.Name]; ok {
				if kindToType(typ.Kind()) == kindToType(dstType.Kind()) {
					e.FB.GetVar(int(index), reg)
				} else {
					tmpReg := e.FB.NewRegister(typ.Kind())
					e.FB.GetVar(int(index), tmpReg)
					e.changeRegister(false, tmpReg, reg, typ, dstType)
				}
			} else {
				panic("bug")
			}
		}

	case *ast.Value:
		if reg == 0 {
			return
		}
		typ := e.TypeInfo[expr].Type
		out, isValue, isRegister := e.quickEmitExpr(expr, typ)
		if isValue {
			e.changeRegister(true, out, reg, typ, dstType)
		} else if isRegister {
			e.changeRegister(false, out, reg, typ, dstType)
		} else {
			// TODO(Gianluca): this switch only handles predeclared types.
			// Add support for defined types.
			switch v := expr.Val.(type) {
			case complex64, complex128:
				panic("TODO(Gianluca): not implemented")
			case uintptr:
				panic("TODO(Gianluca): not implemented")
			case int:
				constant := e.FB.MakeIntConstant(int64(v))
				e.FB.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case int8:
				constant := e.FB.MakeIntConstant(int64(v))
				e.FB.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case int16:
				constant := e.FB.MakeIntConstant(int64(v))
				e.FB.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case int32:
				constant := e.FB.MakeIntConstant(int64(v))
				e.FB.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case int64:
				constant := e.FB.MakeIntConstant(int64(v))
				e.FB.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case uint:
				constant := e.FB.MakeIntConstant(int64(v))
				e.FB.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case uint8:
				constant := e.FB.MakeIntConstant(int64(v))
				e.FB.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case uint16:
				constant := e.FB.MakeIntConstant(int64(v))
				e.FB.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case uint32:
				constant := e.FB.MakeIntConstant(int64(v))
				e.FB.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case uint64:
				constant := e.FB.MakeIntConstant(int64(v))
				e.FB.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case string:
				constant := e.FB.MakeStringConstant(v)
				e.changeRegister(true, constant, reg, typ, dstType)
			case float32:
				constant := e.FB.MakeFloatConstant(float64(v))
				e.FB.LoadNumber(vm.TypeFloat, constant, reg)
			case float64:
				constant := e.FB.MakeFloatConstant(v)
				e.FB.LoadNumber(vm.TypeFloat, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			default:
				constant := e.FB.MakeGeneralConstant(v)
				e.changeRegister(true, constant, reg, typ, dstType)
			}
		}

	case *ast.Index:
		exprType := e.TypeInfo[expr.Expr].Type
		var exprReg int8
		out, _, isRegister := e.quickEmitExpr(expr.Expr, exprType)
		if isRegister {
			exprReg = out
		} else {
			exprReg = e.FB.NewRegister(exprType.Kind())
		}
		out, isValue, isRegister := e.quickEmitExpr(expr.Index, intType)
		ki := false
		var i int8
		if isValue {
			ki = true
			i = out
		} else if isRegister {
			i = out
		} else {
			i = e.FB.NewRegister(reflect.Int)
			e.emitExpr(expr.Index, i, dstType)
		}
		e.FB.Index(ki, exprReg, i, reg, exprType)

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
func (e *Emitter) quickEmitExpr(expr ast.Expression, expectedType reflect.Type) (out int8, k, isRegister bool) {
	// TODO (Gianluca): quickEmitExpr must evaluate only expression which does
	// not need extra registers for evaluation.

	// Src kind and dst kind are different, so a Move/Conversion is required.
	if kindToType(expectedType.Kind()) != kindToType(e.TypeInfo[expr].Type.Kind()) {
		return 0, false, false
	}
	switch expr := expr.(type) {
	case *ast.Identifier:
		if e.FB.IsVariable(expr.Name) {
			return e.FB.ScopeLookup(expr.Name), false, true
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
func (e *Emitter) emitBuiltin(call *ast.Call, reg int8, dstType reflect.Type) {
	switch call.Func.(*ast.Identifier).Name {
	case "append":
		panic("TODO: not implemented")
	case "cap":
		typ := e.TypeInfo[call.Args[0]].Type
		s := e.FB.NewRegister(typ.Kind())
		e.emitExpr(call.Args[0], s, typ)
		e.FB.Cap(s, reg)
		e.changeRegister(false, reg, reg, intType, dstType)
	case "close":
		chanType := e.TypeInfo[call.Args[0]].Type
		chanReg := e.FB.NewRegister(chanType.Kind())
		e.emitExpr(call.Args[0], chanReg, chanType)
		e.FB.Close(chanReg)
	case "complex":
		panic("TODO: not implemented")
	case "copy":
		dst, _, isRegister := e.quickEmitExpr(call.Args[0], e.TypeInfo[call.Args[0]].Type)
		if !isRegister {
			dst = e.FB.NewRegister(reflect.Slice)
			e.emitExpr(call.Args[0], dst, e.TypeInfo[call.Args[0]].Type)
		}
		src, _, isRegister := e.quickEmitExpr(call.Args[1], e.TypeInfo[call.Args[1]].Type)
		if !isRegister {
			src = e.FB.NewRegister(reflect.Slice)
			e.emitExpr(call.Args[0], src, e.TypeInfo[call.Args[0]].Type)
		}
		e.FB.Copy(dst, src, reg)
		if reg != 0 {
			e.changeRegister(false, reg, reg, intType, dstType)
		}
	case "delete":
		mapExpr := call.Args[0]
		keyExpr := call.Args[1]
		mapp := e.FB.NewRegister(reflect.Interface)
		e.emitExpr(mapExpr, mapp, emptyInterfaceType)
		key := e.FB.NewRegister(reflect.Interface)
		e.emitExpr(keyExpr, key, emptyInterfaceType)
		e.FB.Delete(mapp, key)
	case "imag":
		panic("TODO: not implemented")
	case "html": // TODO (Gianluca): to review.
		panic("TODO: not implemented")
	case "len":
		typ := e.TypeInfo[call.Args[0]].Type
		s := e.FB.NewRegister(typ.Kind())
		e.emitExpr(call.Args[0], s, typ)
		e.FB.Len(s, reg, typ)
		e.changeRegister(false, reg, reg, intType, dstType)
	case "make":
		typ := call.Args[0].(*ast.Value).Val.(reflect.Type)
		regType := e.FB.Type(typ)
		switch typ.Kind() {
		case reflect.Map:
			if len(call.Args) == 1 {
				e.FB.MakeMap(regType, true, 0, reg)
			} else {
				size, kSize, isRegister := e.quickEmitExpr(call.Args[1], intType)
				if !kSize && !isRegister {
					size = e.FB.NewRegister(reflect.Int)
					e.emitExpr(call.Args[1], size, e.TypeInfo[call.Args[1]].Type)
				}
				e.FB.MakeMap(regType, kSize, size, reg)
			}
		case reflect.Slice:
			lenExpr := call.Args[1]
			lenReg, kLen, isRegister := e.quickEmitExpr(lenExpr, intType)
			if !kLen && !isRegister {
				lenReg = e.FB.NewRegister(reflect.Int)
				e.emitExpr(lenExpr, lenReg, e.TypeInfo[lenExpr].Type)
			}
			var kCap bool
			var capReg int8
			if len(call.Args) == 3 {
				capExpr := call.Args[2]
				var isRegister bool
				capReg, kCap, isRegister = e.quickEmitExpr(capExpr, intType)
				if !kCap && !isRegister {
					capReg = e.FB.NewRegister(reflect.Int)
					e.emitExpr(capExpr, capReg, e.TypeInfo[capExpr].Type)
				}
			} else {
				kCap = kLen
				capReg = lenReg
			}
			e.FB.MakeSlice(kLen, kCap, typ, lenReg, capReg, reg)
		case reflect.Chan:
			chanType := e.TypeInfo[call.Args[0]].Type
			chanTypeIndex := e.FB.AddType(chanType)
			var kCapacity bool
			var capacity int8
			if len(call.Args) == 1 {
				capacity = 0
				kCapacity = true
			} else {
				var isRegister bool
				capacity, kCapacity, isRegister = e.quickEmitExpr(call.Args[1], intType)
				if !kCapacity && !isRegister {
					capacity = e.FB.NewRegister(reflect.Int)
					e.emitExpr(call.Args[1], capacity, intType)
				}
			}
			e.FB.MakeChan(int8(chanTypeIndex), kCapacity, capacity, reg)
		default:
			panic("bug")
		}
	case "new":
		newType := call.Args[0].(*ast.Value).Val.(reflect.Type)
		e.FB.New(newType, reg)
	case "panic":
		arg := call.Args[0]
		reg, _, isRegister := e.quickEmitExpr(arg, emptyInterfaceType)
		if !isRegister {
			reg = e.FB.NewRegister(reflect.Interface)
			e.emitExpr(arg, reg, emptyInterfaceType)
		}
		e.FB.Panic(reg, call.Pos().Line)
	case "print":
		for i := range call.Args {
			arg := e.FB.NewRegister(reflect.Interface)
			e.emitExpr(call.Args[i], arg, emptyInterfaceType)
			e.FB.Print(arg)
		}
	case "println":
		panic("TODO: not implemented")
	case "real":
		panic("TODO: not implemented")
	case "recover":
		e.FB.Recover(reg)
	default:
		panic("unkown builtin") // TODO(Gianluca): remove.
	}
}

// EmitNodes emits instructions for nodes.
func (e *Emitter) EmitNodes(nodes []ast.Node) {
	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.Assignment:
			e.emitAssignmentNode(node)

		case *ast.Block:
			e.FB.EnterScope()
			e.EmitNodes(node.Nodes)
			e.FB.ExitScope()

		case *ast.Break:
			if node.Label != nil {
				panic("TODO(Gianluca): not implemented")
			}
			e.FB.Break(e.rangeLabels[len(e.rangeLabels)-1][0])
			e.FB.Goto(e.rangeLabels[len(e.rangeLabels)-1][1])

		case *ast.Continue:
			if node.Label != nil {
				panic("TODO(Gianluca): not implemented")
			}
			e.FB.Continue(e.rangeLabels[len(e.rangeLabels)-1][0])

		case *ast.Defer, *ast.Go:
			if def, ok := node.(*ast.Defer); ok {
				if e.TypeInfo[def.Call.Func].IsBuiltin() {
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
			funReg := e.FB.NewRegister(reflect.Func)
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
			funType := e.TypeInfo[funNode].Type
			e.emitExpr(funNode, funReg, e.TypeInfo[funNode].Type)
			offset := vm.StackShift{
				int8(e.FB.numRegs[reflect.Int]),
				int8(e.FB.numRegs[reflect.Float64]),
				int8(e.FB.numRegs[reflect.String]),
				int8(e.FB.numRegs[reflect.Interface]),
			}
			// TODO(Gianluca): currently supports only deferring or
			// starting goroutines of Scrigo defined functions.
			isNative := false
			e.prepareCallParameters(funType, args, isNative)
			// TODO(Gianluca): currently supports only deferring functions
			// and starting goroutines with no arguments and no return
			// parameters.
			argsShift := vm.StackShift{}
			switch node.(type) {
			case *ast.Defer:
				e.FB.Defer(funReg, vm.NoVariadic, offset, argsShift)
			case *ast.Go:
				// TODO(Gianluca):
				e.FB.Go()
			}

		case *ast.Goto:
			if label, ok := e.labels[e.CurrentFunction][node.Label.Name]; ok {
				e.FB.Goto(label)
			} else {
				if e.labels[e.CurrentFunction] == nil {
					e.labels[e.CurrentFunction] = make(map[string]uint32)
				}
				label = e.FB.NewLabel()
				e.FB.Goto(label)
				e.labels[e.CurrentFunction][node.Label.Name] = label
			}

		case *ast.Label:
			if _, found := e.labels[e.CurrentFunction][node.Name.Name]; !found {
				if e.labels[e.CurrentFunction] == nil {
					e.labels[e.CurrentFunction] = make(map[string]uint32)
				}
				e.labels[e.CurrentFunction][node.Name.Name] = e.FB.NewLabel()
			}
			e.FB.SetLabelAddr(e.labels[e.CurrentFunction][node.Name.Name])
			if node.Statement != nil {
				e.EmitNodes([]ast.Node{node.Statement})
			}

		case *ast.If:
			e.FB.EnterScope()
			if node.Assignment != nil {
				e.EmitNodes([]ast.Node{node.Assignment})
			}
			e.emitCondition(node.Condition)
			if node.Else == nil { // TODO (Gianluca): can "then" and "else" be unified in some way?
				endIfLabel := e.FB.NewLabel()
				e.FB.Goto(endIfLabel)
				e.EmitNodes(node.Then.Nodes)
				e.FB.SetLabelAddr(endIfLabel)
			} else {
				elseLabel := e.FB.NewLabel()
				e.FB.Goto(elseLabel)
				e.EmitNodes(node.Then.Nodes)
				endIfLabel := e.FB.NewLabel()
				e.FB.Goto(endIfLabel)
				e.FB.SetLabelAddr(elseLabel)
				if node.Else != nil {
					switch els := node.Else.(type) {
					case *ast.If:
						e.EmitNodes([]ast.Node{els})
					case *ast.Block:
						e.EmitNodes(els.Nodes)
					}
				}
				e.FB.SetLabelAddr(endIfLabel)
			}
			e.FB.ExitScope()

		case *ast.For:
			e.FB.EnterScope()
			if node.Init != nil {
				e.EmitNodes([]ast.Node{node.Init})
			}
			if node.Condition != nil {
				forLabel := e.FB.NewLabel()
				e.FB.SetLabelAddr(forLabel)
				e.emitCondition(node.Condition)
				endForLabel := e.FB.NewLabel()
				e.FB.Goto(endForLabel)
				e.EmitNodes(node.Body)
				if node.Post != nil {
					e.EmitNodes([]ast.Node{node.Post})
				}
				e.FB.Goto(forLabel)
				e.FB.SetLabelAddr(endForLabel)
			} else {
				forLabel := e.FB.NewLabel()
				e.FB.SetLabelAddr(forLabel)
				e.EmitNodes(node.Body)
				if node.Post != nil {
					e.EmitNodes([]ast.Node{node.Post})
				}
				e.FB.Goto(forLabel)
			}
			e.FB.ExitScope()

		case *ast.ForRange:
			e.FB.EnterScope()
			vars := node.Assignment.Variables
			indexReg := int8(0)
			if len(vars) >= 1 && !isBlankIdentifier(vars[0]) {
				name := vars[0].(*ast.Identifier).Name
				if node.Assignment.Type == ast.AssignmentDeclaration {
					indexReg = e.FB.NewRegister(reflect.Int)
					e.FB.BindVarReg(name, indexReg)
				} else {
					indexReg = e.FB.ScopeLookup(name)
				}
			}
			elemReg := int8(0)
			if len(vars) == 2 && !isBlankIdentifier(vars[1]) {
				typ := e.TypeInfo[vars[1]].Type
				name := vars[1].(*ast.Identifier).Name
				if node.Assignment.Type == ast.AssignmentDeclaration {
					elemReg = e.FB.NewRegister(typ.Kind())
					e.FB.BindVarReg(name, elemReg)
				} else {
					elemReg = e.FB.ScopeLookup(name)
				}
			}
			expr := node.Assignment.Values[0]
			exprType := e.TypeInfo[expr].Type
			exprReg, kExpr, isRegister := e.quickEmitExpr(expr, exprType)
			if (!kExpr && !isRegister) || exprType.Kind() != reflect.String {
				kExpr = false
				exprReg = e.FB.NewRegister(exprType.Kind())
				e.emitExpr(expr, exprReg, exprType)
			}
			rangeLabel := e.FB.NewLabel()
			e.FB.SetLabelAddr(rangeLabel)
			endRange := e.FB.NewLabel()
			e.rangeLabels = append(e.rangeLabels, [2]uint32{rangeLabel, endRange})
			e.FB.Range(kExpr, exprReg, indexReg, elemReg, exprType.Kind())
			e.FB.Goto(endRange)
			e.FB.EnterScope()
			e.EmitNodes(node.Body)
			e.FB.Continue(rangeLabel)
			e.FB.SetLabelAddr(endRange)
			e.rangeLabels = e.rangeLabels[:len(e.rangeLabels)-1]
			e.FB.ExitScope()
			e.FB.ExitScope()

		case *ast.Return:
			// TODO(Gianluca): complete implementation of tail call optimization.
			// if len(node.Values) == 1 {
			// 	if call, ok := node.Values[0].(*ast.Call); ok {
			// 		tmpRegs := make([]int8, len(call.Args))
			// 		paramPosition := make([]int8, len(call.Args))
			// 		tmpTypes := make([]reflect.Type, len(call.Args))
			// 		shift := vm.StackShift{}
			// 		for i := range call.Args {
			// 			tmpTypes[i] = e.TypeInfo[call.Args[i]].Type
			// 			t := int(kindToType(tmpTypes[i].Kind()))
			// 			tmpRegs[i] = e.FB.NewRegister(tmpTypes[i].Kind())
			// 			shift[t]++
			// 			c.compileExpr(call.Args[i], tmpRegs[i], tmpTypes[i])
			// 			paramPosition[i] = shift[t]
			// 		}
			// 		for i := range call.Args {
			// 			e.changeRegister(false, tmpRegs[i], paramPosition[i], tmpTypes[i], e.TypeInfo[call.Func].Type.In(i))
			// 		}
			// 		e.FB.TailCall(vm.CurrentFunction, node.Pos().Line)
			// 		continue
			// 	}
			// }
			offset := [4]int8{}
			for i, v := range node.Values {
				typ := e.CurrentFunction.Type.Out(i)
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
				e.emitExpr(v, reg, typ)
			}
			e.FB.Return()

		case *ast.Send:
			ch := e.FB.NewRegister(reflect.Chan)
			e.emitExpr(node.Channel, ch, e.TypeInfo[node.Channel].Type)
			elemType := e.TypeInfo[node.Value].Type
			v := e.FB.NewRegister(elemType.Kind())
			e.emitExpr(node.Value, v, elemType)
			e.FB.Send(ch, v)

		case *ast.Switch:
			e.emitSwitch(node)

		case *ast.TypeSwitch:
			e.emitTypeSwitch(node)

		case *ast.Var:
			addresses := make([]Address, len(node.Identifiers))
			for i, v := range node.Identifiers {
				varType := e.TypeInfo[v].Type
				if e.IndirectVars[v] {
					varReg := -e.FB.NewRegister(reflect.Interface)
					e.FB.BindVarReg(v.Name, varReg)
					addresses[i] = e.NewAddress(AddressIndirectDeclaration, varType, varReg, 0)
				} else {
					varReg := e.FB.NewRegister(varType.Kind())
					e.FB.BindVarReg(v.Name, varReg)
					addresses[i] = e.NewAddress(AddressRegister, varType, varReg, 0)
				}
			}
			e.assign(addresses, node.Values)

		case ast.Expression:
			// TODO (Gianluca): use 0 (which is no longer a valid
			// register) and handle it as a special case in emitExpr.
			e.emitExpr(node, 0, reflect.Type(nil))

		default:
			panic(fmt.Sprintf("node %T not supported", node)) // TODO(Gianluca): remove.

		}
	}
}

// emitTypeSwitch emits instructions for a type switch node.
func (e *Emitter) emitTypeSwitch(node *ast.TypeSwitch) {
	// TODO (Gianluca): a type-checker bug does not replace type switch type with proper value.

	e.FB.EnterScope()

	if node.Init != nil {
		e.EmitNodes([]ast.Node{node.Init})
	}

	typAss := node.Assignment.Values[0].(*ast.TypeAssertion)
	typ := e.TypeInfo[typAss.Expr].Type
	expr := e.FB.NewRegister(typ.Kind())
	e.emitExpr(typAss.Expr, expr, typ)

	// typ := node.Assignment.Values[0].(*ast.Value).Val.(reflect.Type)
	// typReg := e.FB.Type(typ)
	// if variab := node.Assignment.Globals[0]; !isBlankast.Identifier(variab) {
	// 	c.compileVarsGetValue([]ast.Expression{variab}, node.Assignment.Values[0], node.Assignment.Type == ast.AssignmentDeclaration)
	// }

	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := e.FB.NewLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = e.FB.NewLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			if isNil(caseExpr) {
				panic("TODO(Gianluca): not implemented")
			}
			caseType := caseExpr.(*ast.Value).Val.(reflect.Type)
			e.FB.Assert(expr, caseType, 0)
			next := e.FB.NewLabel()
			e.FB.Goto(next)
			e.FB.Goto(bodyLabels[i])
			e.FB.SetLabelAddr(next)
		}
	}

	if hasDefault {
		defaultLabel = e.FB.NewLabel()
		e.FB.Goto(defaultLabel)
	} else {
		e.FB.Goto(endSwitchLabel)
	}

	for i, cas := range node.Cases {
		if cas.Expressions == nil {
			e.FB.SetLabelAddr(defaultLabel)
		}
		e.FB.SetLabelAddr(bodyLabels[i])
		e.FB.EnterScope()
		e.EmitNodes(cas.Body)
		e.FB.ExitScope()
		e.FB.Goto(endSwitchLabel)
	}

	e.FB.SetLabelAddr(endSwitchLabel)
	e.FB.ExitScope()
}

// emitSwitch emits instructions for a switch node.
func (e *Emitter) emitSwitch(node *ast.Switch) {

	e.FB.EnterScope()

	if node.Init != nil {
		e.EmitNodes([]ast.Node{node.Init})
	}

	var expr int8
	var typ reflect.Type

	if node.Expr == nil {
		typ = reflect.TypeOf(false)
		expr = e.FB.NewRegister(typ.Kind())
		e.FB.Move(true, 1, expr, typ.Kind())
	} else {
		typ = e.TypeInfo[node.Expr].Type
		expr = e.FB.NewRegister(typ.Kind())
		e.emitExpr(node.Expr, expr, typ)
	}

	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := e.FB.NewLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = e.FB.NewLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			y, ky, isRegister := e.quickEmitExpr(caseExpr, typ)
			if !ky && !isRegister {
				y = e.FB.NewRegister(typ.Kind())
				e.emitExpr(caseExpr, y, typ)
			}
			e.FB.If(ky, expr, vm.ConditionNotEqual, y, typ.Kind()) // Condizione negata per poter concatenare gli if
			e.FB.Goto(bodyLabels[i])
		}
	}

	if hasDefault {
		defaultLabel = e.FB.NewLabel()
		e.FB.Goto(defaultLabel)
	} else {
		e.FB.Goto(endSwitchLabel)
	}

	for i, cas := range node.Cases {
		if cas.Expressions == nil {
			e.FB.SetLabelAddr(defaultLabel)
		}
		e.FB.SetLabelAddr(bodyLabels[i])
		e.FB.EnterScope()
		e.EmitNodes(cas.Body)
		if !cas.Fallthrough {
			e.FB.Goto(endSwitchLabel)
		}
		e.FB.ExitScope()
	}

	e.FB.SetLabelAddr(endSwitchLabel)

	e.FB.ExitScope()
}

// emitCondition emits instructions for a condition. Last instruction added by
// this method is always "If".
func (e *Emitter) emitCondition(cond ast.Expression) {

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
			exprType := e.TypeInfo[expr].Type
			x, _, isRegister := e.quickEmitExpr(expr, exprType)
			if !isRegister {
				x = e.FB.NewRegister(exprType.Kind())
				e.emitExpr(expr, x, exprType)
			}
			condType := vm.ConditionNotNil
			if cond.Operator() == ast.OperatorEqual {
				condType = vm.ConditionNil
			}
			e.FB.If(false, x, condType, 0, exprType.Kind())
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
		if e.isLenBuiltinCall(cond.Expr1) != e.isLenBuiltinCall(cond.Expr2) {
			var lenArg, expr ast.Expression
			if e.isLenBuiltinCall(cond.Expr1) {
				lenArg = cond.Expr1.(*ast.Call).Args[0]
				expr = cond.Expr2
			} else {
				lenArg = cond.Expr2.(*ast.Call).Args[0]
				expr = cond.Expr1
			}
			if e.TypeInfo[lenArg].Type.Kind() == reflect.String { // len is optimized for strings only.
				lenArgType := e.TypeInfo[lenArg].Type
				x, _, isRegister := e.quickEmitExpr(lenArg, lenArgType)
				if !isRegister {
					x = e.FB.NewRegister(lenArgType.Kind())
					e.emitExpr(lenArg, x, lenArgType)
				}
				exprType := e.TypeInfo[expr].Type
				y, ky, isRegister := e.quickEmitExpr(expr, exprType)
				if !ky && !isRegister {
					y = e.FB.NewRegister(exprType.Kind())
					e.emitExpr(expr, y, exprType)
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
				e.FB.If(ky, x, condType, y, reflect.String)
				return
			}
		}

		// if v1 == v2
		// if v1 != v2
		// if v1 <  v2
		// if v1 <= v2
		// if v1 >  v2
		// if v1 >= v2
		expr1Type := e.TypeInfo[cond.Expr1].Type
		expr2Type := e.TypeInfo[cond.Expr2].Type
		if expr1Type.Kind() == expr2Type.Kind() {
			switch kind := expr1Type.Kind(); kind {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64,
				reflect.String:
				expr1Type := e.TypeInfo[cond.Expr1].Type
				x, _, isRegister := e.quickEmitExpr(cond.Expr1, expr1Type)
				if !isRegister {
					x = e.FB.NewRegister(expr1Type.Kind())
					e.emitExpr(cond.Expr1, x, expr1Type)
				}
				expr2Type := e.TypeInfo[cond.Expr2].Type
				y, ky, isRegister := e.quickEmitExpr(cond.Expr2, expr2Type)
				if !ky && !isRegister {
					y = e.FB.NewRegister(expr2Type.Kind())
					e.emitExpr(cond.Expr2, y, expr2Type)
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
				e.FB.If(ky, x, condType, y, kind)
				return
			}
		}

	default:

		condType := e.TypeInfo[cond].Type
		x, _, isRegister := e.quickEmitExpr(cond, condType)
		if !isRegister {
			x = e.FB.NewRegister(condType.Kind())
			e.emitExpr(cond, x, condType)
		}
		yConst := e.FB.MakeIntConstant(1)
		y := e.FB.NewRegister(reflect.Bool)
		e.FB.LoadNumber(vm.TypeInt, yConst, y)
		e.FB.If(false, x, vm.ConditionEqual, y, reflect.Bool)

	}

}
