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

// An emitter emits instructions for the VM.
type emitter struct {
	addAllocInstructions bool
	fb                   *functionBuilder
	indirectVars         map[*ast.Identifier]bool
	labels               map[*vm.Function]map[string]uint32
	pkg                  *ast.Package
	typeInfos            map[ast.Node]*TypeInfo
	upvarsNames          map[*vm.Function]map[string]int

	// Scrigo functions.
	availableFunctions map[*ast.Package]map[string]*vm.Function
	assignedFunctions  map[*vm.Function]map[*vm.Function]int8

	// Scrigo variables.
	pkgVariables map[*ast.Package]map[string]int16

	// Predefined functions.
	predefFunIndexes map[*vm.Function]map[reflect.Value]int8

	// Predefined variables.
	predefVarIndexes map[*vm.Function]map[reflect.Value]int16

	// Holds all Scrigo-defined and pre-predefined global variables.
	globals []vm.Global

	// rangeLabels is a list of current active Ranges. First element is the
	// Range address, second refers to the first instruction outside Range's
	// body.
	rangeLabels [][2]uint32

	// breakable is true if emitting a "breakable" statement (except ForRange,
	// which implements his own "breaking" system).
	breakable bool

	// breakLabel, if not nil, is the label to which pre-stated "breaks" must
	// jump.
	breakLabel *uint32
}

// newEmitter returns a new emitter reading sources from r.
func newEmitter(typeInfos map[ast.Node]*TypeInfo, indirectVars map[*ast.Identifier]bool) *emitter {
	c := &emitter{
		assignedFunctions:  map[*vm.Function]map[*vm.Function]int8{},
		availableFunctions: map[*ast.Package]map[string]*vm.Function{},
		indirectVars:       indirectVars,
		labels:             make(map[*vm.Function]map[string]uint32),
		predefFunIndexes:   map[*vm.Function]map[reflect.Value]int8{},
		predefVarIndexes:   map[*vm.Function]map[reflect.Value]int16{},
		pkgVariables:       map[*ast.Package]map[string]int16{},
		typeInfos:          typeInfos,
		upvarsNames:        make(map[*vm.Function]map[string]int),
	}
	return c
}

func EmitSingle(tree *ast.Tree, typeInfos map[ast.Node]*TypeInfo, indirectVars map[*ast.Identifier]bool, alloc bool) (*vm.Function, []vm.Global) {
	e := newEmitter(typeInfos, indirectVars)
	e.addAllocInstructions = alloc
	e.fb = newBuilder(NewFunction("main", "main", reflect.FuncOf(nil, nil, false)))
	e.fb.SetAlloc(alloc)
	e.fb.EnterScope()
	e.EmitNodes(tree.Nodes)
	e.fb.ExitScope()
	e.fb.End()
	return e.fb.fn, e.globals
}

// Package is the result of a package emitting process.
type Package struct {
	Globals   []vm.Global
	Functions map[string]*vm.Function
	Main      *vm.Function
}

// EmitPackageMain emits package main, returning a Package.
func EmitPackageMain(pkgMain *ast.Package, typeInfos map[ast.Node]*TypeInfo, indirectVars map[*ast.Identifier]bool, alloc bool) *Package {
	e := newEmitter(typeInfos, indirectVars)
	e.addAllocInstructions = alloc
	funcs, _, _ := e.emitPackage(pkgMain)
	main := e.availableFunctions[pkgMain]["main"]
	pkg := &Package{
		Globals:   e.globals,
		Functions: funcs,
		Main:      main,
	}
	return pkg
}

// emitPackage emits package pkg. Returns a list of exported functions and
// exported variables.
func (e *emitter) emitPackage(pkg *ast.Package) (map[string]*vm.Function, map[string]int16, []*vm.Function) {
	e.pkg = pkg
	e.availableFunctions[e.pkg] = map[string]*vm.Function{}
	e.pkgVariables[e.pkg] = map[string]int16{}

	// TODO(Gianluca): if a package is imported more than once, its init
	// functions are called more than once: that is wrong.
	allInits := []*vm.Function{} // List of all "init" functions in current package.

	// Emits imports.
	for _, decl := range pkg.Declarations {
		if imp, ok := decl.(*ast.Import); ok {
			if imp.Tree == nil {
				// Nothing to do. Predefined variables, constants, types
				// and functions are added as informations to tree by
				// type-checker.
			} else {
				backupPkg := e.pkg
				pkg := imp.Tree.Nodes[0].(*ast.Package)
				funcs, vars, inits := e.emitPackage(pkg)
				e.pkg = backupPkg
				allInits = append(allInits, inits...)
				var importName string
				if imp.Ident == nil {
					importName = pkg.Name
				} else {
					switch imp.Ident.Name {
					case "_":
						panic("TODO(Gianluca): not implemented")
					case ".":
						importName = ""
					default:
						importName = imp.Ident.Name
					}
				}
				for name, fn := range funcs {
					if importName == "" {
						e.availableFunctions[e.pkg][name] = fn
					} else {
						e.availableFunctions[e.pkg][importName+"."+name] = fn
					}
				}
				for name, v := range vars {
					if importName == "" {
						e.pkgVariables[e.pkg][name] = v
					} else {
						e.pkgVariables[e.pkg][importName+"."+name] = v
					}
				}
			}
		}
	}

	exportedFunctions := map[string]*vm.Function{}

	// Stores all function declarations in current package before building
	// their bodies: order of declaration doesn't matter at package level.
	initToBuild := len(allInits) // Index of next "init" function to build.
	for _, dec := range pkg.Declarations {
		if fun, ok := dec.(*ast.Func); ok {
			fn := NewFunction("main", fun.Ident.Name, fun.Type.Reflect)
			if fun.Ident.Name == "init" {
				allInits = append(allInits, fn)
			} else {
				e.availableFunctions[e.pkg][fun.Ident.Name] = fn
				if isExported(fun.Ident.Name) {
					exportedFunctions[fun.Ident.Name] = fn
				}
			}
		}
	}

	exportedVars := map[string]int16{}

	// Emits package variables.
	var initVarsFn *vm.Function
	var initVarsFb *functionBuilder
	pkgVarRegs := map[string]int8{}
	for _, dec := range pkg.Declarations {
		if n, ok := dec.(*ast.Var); ok {
			// If package has some variable declarations, a special "init" function
			// must be created to initialize them. "$initvars" is used because is not
			// a valid Go identifier, so there's no risk of collision with Scrigo
			// defined functions.
			backupFb := e.fb
			if initVarsFn == nil {
				initVarsFn = NewFunction("main", "$initvars", reflect.FuncOf(nil, nil, false))
				e.availableFunctions[e.pkg]["$initvars"] = initVarsFn
				initVarsFb = newBuilder(initVarsFn)
				initVarsFb.SetAlloc(e.addAllocInstructions)
				initVarsFb.EnterScope()
			}
			e.fb = initVarsFb
			addresses := make([]address, len(n.Lhs))
			for i, v := range n.Lhs {
				staticType := e.typeInfos[v].Type
				varReg := -e.fb.NewRegister(reflect.Interface)
				e.fb.BindVarReg(v.Name, varReg)
				addresses[i] = e.newAddress(addressIndirectDeclaration, staticType, varReg, 0)
				// Variable register is stored: will be used later to
				// store initialized value inside proper global index
				// during building of $initvars.
				pkgVarRegs[v.Name] = varReg
				e.globals = append(e.globals, vm.Global{Pkg: "main", Name: v.Name, Type: staticType})
				e.pkgVariables[e.pkg][v.Name] = int16(len(e.globals) - 1)
				exportedVars[v.Name] = int16(len(e.globals) - 1)
			}
			e.assign(addresses, n.Rhs)
			e.fb = backupFb
		}
	}

	// Emits function declarations.
	for _, dec := range pkg.Declarations {
		if n, ok := dec.(*ast.Func); ok {
			var fn *vm.Function
			if n.Ident.Name == "init" {
				fn = allInits[initToBuild]
				initToBuild++
			} else {
				fn = e.availableFunctions[e.pkg][n.Ident.Name]
			}
			e.fb = newBuilder(fn)
			e.fb.SetAlloc(e.addAllocInstructions)
			e.fb.EnterScope()
			// If function is "main", variable initialization functions
			// must be called before everything else inside main's body.
			if n.Ident.Name == "main" {
				// First: initializes package variables.
				if initVarsFn != nil {
					iv := e.availableFunctions[e.pkg]["$initvars"]
					index := e.fb.AddFunction(iv)
					e.fb.Call(int8(index), vm.StackShift{}, 0)
				}
				// Second: calls all init functions, in order.
				for _, initFunc := range allInits {
					index := e.fb.AddFunction(initFunc)
					e.fb.Call(int8(index), vm.StackShift{}, 0)
				}
			}
			e.prepareFunctionBodyParameters(n)
			e.EmitNodes(n.Body.Nodes)
			e.fb.End()
			e.fb.ExitScope()
		}
	}

	if initVarsFn != nil {
		// Global variables have been locally defined inside the "$initvars"
		// function; their values must now be exported to be available
		// globally.
		backupFb := e.fb
		e.fb = initVarsFb
		for name, reg := range pkgVarRegs {
			index := e.pkgVariables[e.pkg][name]
			e.fb.SetVar(false, reg, int(index))
		}
		e.fb = backupFb
		initVarsFb.ExitScope()
		initVarsFb.Return()
		initVarsFb.End()
	}

	// All functions share Globals.
	for _, f := range e.availableFunctions[pkg] {
		f.Globals = e.globals
	}

	// If this package is imported, initFuncs must contain initVarsFn, that is
	// processed as a common "init" function.
	if initVarsFn != nil {
		allInits = append(allInits, initVarsFn)
	}

	return exportedFunctions, exportedVars, allInits

}

// prepareCallParameters prepares parameters (out and in) for a function call of
// type funcType and arguments args. Returns the list of return registers and
// their respective type.
//
// Note that this functions is different than prepareFunctionBodyParameters;
// while the former is used before emitting a function's body, the latter is
// used before calling it.
func (e *emitter) prepareCallParameters(funcType reflect.Type, args []ast.Expression, isPredefined bool) ([]int8, []reflect.Type) {
	numOut := funcType.NumOut()
	numIn := funcType.NumIn()
	regs := make([]int8, numOut)
	types := make([]reflect.Type, numOut)
	for i := 0; i < numOut; i++ {
		typ := funcType.Out(i)
		regs[i] = e.fb.NewRegister(typ.Kind())
		types[i] = typ
	}
	if funcType.IsVariadic() {
		for i := 0; i < numIn-1; i++ {
			typ := funcType.In(i)
			reg := e.fb.NewRegister(typ.Kind())
			e.fb.EnterScope()
			e.emitExpr(args[i], reg, typ)
			e.fb.ExitScope()
		}
		if varArgs := len(args) - (numIn - 1); varArgs > 0 {
			typ := funcType.In(numIn - 1).Elem()
			if isPredefined {
				for i := 0; i < varArgs; i++ {
					reg := e.fb.NewRegister(typ.Kind())
					e.fb.EnterStack()
					e.emitExpr(args[i+numIn-1], reg, typ)
					e.fb.ExitStack()
				}
			} else {
				sliceReg := int8(numIn)
				e.fb.MakeSlice(true, true, funcType.In(numIn-1), int8(varArgs), int8(varArgs), sliceReg)
				for i := 0; i < varArgs; i++ {
					tmpReg := e.fb.NewRegister(typ.Kind())
					e.fb.EnterStack()
					e.emitExpr(args[i+numIn-1], tmpReg, typ)
					e.fb.ExitStack()
					indexReg := e.fb.NewRegister(reflect.Int)
					e.fb.Move(true, int8(i), indexReg, reflect.Int)
					e.fb.SetSlice(false, sliceReg, tmpReg, indexReg, typ.Kind())
				}
			}
		}
	} else { // No-variadic function.
		if numIn > 1 && len(args) == 1 { // f(g()), where f takes more than 1 argument.
			regs, types := e.emitCall(args[0].(*ast.Call))
			for i := range regs {
				dstType := funcType.In(i)
				reg := e.fb.NewRegister(dstType.Kind())
				e.changeRegister(false, regs[i], reg, types[i], dstType)
			}
		} else {
			for i := 0; i < numIn; i++ {
				typ := funcType.In(i)
				reg := e.fb.NewRegister(typ.Kind())
				e.fb.EnterStack()
				e.emitExpr(args[i], reg, typ)
				e.fb.ExitStack()
			}
		}
	}
	return regs, types
}

// prepareFunctionBodyParameters prepares fun's parameters (out and int) before
// emitting its body.
//
// Note that this functions is different than prepareCallParameters; while the
// former is used before calling a function, the latter is used before emitting
// it's body.
func (e *emitter) prepareFunctionBodyParameters(fun *ast.Func) {
	// Reserves space for return parameters.
	fillParametersTypes(fun.Type.Result)
	for _, res := range fun.Type.Result {
		resType := res.Type.(*ast.Value).Val.(reflect.Type)
		kind := resType.Kind()
		retReg := e.fb.NewRegister(kind)
		if res.Ident != nil {
			e.fb.BindVarReg(res.Ident.Name, retReg)
		}
	}
	// Binds function argument names to pre-allocated registers.
	fillParametersTypes(fun.Type.Parameters)
	for _, par := range fun.Type.Parameters {
		parType := par.Type.(*ast.Value).Val.(reflect.Type)
		kind := parType.Kind()
		argReg := e.fb.NewRegister(kind)
		if par.Ident != nil {
			e.fb.BindVarReg(par.Ident.Name, argReg)
		}
	}
}

// emitCall emits instruction for a call, returning the list of registers (and
// their respective type) within which return values are inserted.
func (e *emitter) emitCall(call *ast.Call) ([]int8, []reflect.Type) {

	stackShift := vm.StackShift{
		int8(e.fb.numRegs[reflect.Int]),
		int8(e.fb.numRegs[reflect.Float64]),
		int8(e.fb.numRegs[reflect.String]),
		int8(e.fb.numRegs[reflect.Interface]),
	}

	// Predefined function (identifiers, selectors etc...).
	funcTypeInfo := e.typeInfos[call.Func]
	funcType := funcTypeInfo.Type
	if funcTypeInfo.IsPredefined() {
		regs, types := e.prepareCallParameters(funcType, call.Args, true)
		var name string
		switch f := call.Func.(type) {
		case *ast.Identifier:
			name = f.Name
		case *ast.Selector:
			name = f.Ident
		}
		index := e.predefFuncIndex(funcTypeInfo.Value.(reflect.Value), funcTypeInfo.PredefPackageName, name)
		if funcType.IsVariadic() {
			numVar := len(call.Args) - (funcType.NumIn() - 1)
			e.fb.CallPredefined(index, int8(numVar), stackShift)
		} else {
			e.fb.CallPredefined(index, vm.NoVariadic, stackShift)
		}
		return regs, types
	}

	// Scrigo-defined function (identifier).
	if ident, ok := call.Func.(*ast.Identifier); ok && !e.fb.IsVariable(ident.Name) {
		if fun, ok := e.availableFunctions[e.pkg][ident.Name]; ok {
			regs, types := e.prepareCallParameters(fun.Type, call.Args, false)
			index := e.functionIndex(fun)
			e.fb.Call(index, stackShift, call.Pos().Line)
			return regs, types
		}
	}

	// Scrigo-defined function (selector).
	if selector, ok := call.Func.(*ast.Selector); ok {
		if ident, ok := selector.Expr.(*ast.Identifier); ok {
			if fun, ok := e.availableFunctions[e.pkg][selector.Ident+"."+ident.Name]; ok {
				regs, types := e.prepareCallParameters(fun.Type, call.Args, false)
				index := e.functionIndex(fun)
				e.fb.Call(index, stackShift, call.Pos().Line)
				return regs, types
			}
		}
	}

	// Indirect function.
	funReg, _, isRegister := e.quickEmitExpr(call.Func, e.typeInfos[call.Func].Type)
	if !isRegister {
		funReg = e.fb.NewRegister(reflect.Func)
		e.emitExpr(call.Func, funReg, e.typeInfos[call.Func].Type)
	}
	regs, types := e.prepareCallParameters(funcType, call.Args, true)
	e.fb.CallIndirect(funReg, 0, stackShift)
	return regs, types
}

// emitExpr emits instruction such that expr value is put into reg. If reg is
// zero, instructions are emitted anyway but result is discarded.
func (e *emitter) emitExpr(expr ast.Expression, reg int8, dstType reflect.Type) {
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
			e.fb.EnterStack()
			e.emitExpr(expr.Expr1, reg, dstType)
			endIf := e.fb.NewLabel()
			e.fb.If(true, reg, vm.ConditionEqual, cmp, reflect.Int)
			e.fb.Goto(endIf)
			e.emitExpr(expr.Expr2, reg, dstType)
			e.fb.SetLabelAddr(endIf)
			e.fb.ExitStack()
			return
		}

		e.fb.EnterStack()

		xType := e.typeInfos[expr.Expr1].Type
		x := e.fb.NewRegister(xType.Kind())
		e.emitExpr(expr.Expr1, x, xType)

		y, ky, isRegister := e.quickEmitExpr(expr.Expr2, xType)
		if !ky && !isRegister {
			y = e.fb.NewRegister(xType.Kind())
			e.emitExpr(expr.Expr2, y, xType)
		}

		res := e.fb.NewRegister(xType.Kind())

		switch op := expr.Operator(); {
		case op == ast.OperatorAddition && xType.Kind() == reflect.String && reg != 0:
			if ky {
				y = e.fb.NewRegister(reflect.String)
				e.emitExpr(expr.Expr2, y, xType)
			}
			e.fb.Concat(x, y, reg)
		case op == ast.OperatorAddition && reg != 0:
			e.fb.Add(ky, x, y, res, xType.Kind())
			e.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorSubtraction && reg != 0:
			e.fb.Sub(ky, x, y, res, xType.Kind())
			e.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorMultiplication && reg != 0:
			e.fb.Mul(ky, x, y, res, xType.Kind())
			e.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorDivision && reg != 0:
			e.fb.Div(ky, x, y, res, xType.Kind())
			e.changeRegister(false, res, reg, xType, dstType)
		case op == ast.OperatorDivision && reg == 0:
			dummyReg := e.fb.NewRegister(xType.Kind())
			e.fb.Div(ky, x, y, dummyReg, xType.Kind()) // produces division by zero.
		case op == ast.OperatorModulo && reg != 0:
			e.fb.Rem(ky, x, y, res, xType.Kind())
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
				e.fb.Move(true, 1, reg, reflect.Bool)
				e.fb.If(ky, x, cond, y, xType.Kind())
				e.fb.Move(true, 0, reg, reflect.Bool)
			}
		case op == ast.OperatorOr,
			op == ast.OperatorAnd,
			op == ast.OperatorXor,
			op == ast.OperatorAndNot,
			op == ast.OperatorLeftShift,
			op == ast.OperatorRightShift:
			if reg != 0 {
				e.fb.BinaryBitOperation(op, ky, x, y, reg, xType.Kind())
				if kindToType(xType.Kind()) != kindToType(dstType.Kind()) {
					e.changeRegister(ky, reg, reg, xType, dstType)
				}
			}
		default:
			panic(fmt.Errorf("TODO: not implemented operator %s", expr.Operator()))
		}

		e.fb.ExitStack()

	case *ast.Call:
		e.fb.EnterStack()
		// Builtin call.
		if e.typeInfos[expr.Func].IsBuiltin() {
			e.emitBuiltin(expr, reg, dstType)
			e.fb.ExitStack()
			return
		}
		// Conversion.
		if val, ok := expr.Func.(*ast.Value); ok {
			if convertType, ok := val.Val.(reflect.Type); ok {
				typ := e.typeInfos[expr.Args[0]].Type
				arg := e.fb.NewRegister(typ.Kind())
				e.emitExpr(expr.Args[0], arg, typ)
				e.fb.Convert(arg, convertType, reg, typ.Kind())
				e.fb.ExitStack()
				return
			}
		}
		// Function call.
		regs, types := e.emitCall(expr)
		if reg != 0 {
			e.changeRegister(false, regs[0], reg, types[0], dstType)
		}
		e.fb.ExitStack()

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
				e.fb.MakeSlice(true, true, typ, size, size, reg)
			}
			var index int8 = -1
			for _, kv := range expr.KeyValues {
				if kv.Key != nil {
					index = int8(kv.Key.(*ast.Value).Val.(int))
				} else {
					index++
				}
				e.fb.EnterStack()
				indexReg := e.fb.NewRegister(reflect.Int)
				e.fb.Move(true, index, indexReg, reflect.Int)
				value, kvalue, isRegister := e.quickEmitExpr(kv.Value, typ.Elem())
				if !kvalue && !isRegister {
					value = e.fb.NewRegister(typ.Elem().Kind())
					e.emitExpr(kv.Value, value, typ.Elem())
				}
				if reg != 0 {
					e.fb.SetSlice(kvalue, reg, value, indexReg, typ.Elem().Kind())
				}
				e.fb.ExitStack()
			}
		case reflect.Struct:
			panic("TODO: not implemented")
		case reflect.Map:
			// TODO (Gianluca): handle maps with bigger size.
			size := len(expr.KeyValues)
			sizeReg := e.fb.MakeIntConstant(int64(size))
			e.fb.MakeMap(typ, true, sizeReg, reg)
			for _, kv := range expr.KeyValues {
				keyReg := e.fb.NewRegister(typ.Key().Kind())
				valueReg := e.fb.NewRegister(typ.Elem().Kind())
				e.fb.EnterStack()
				e.emitExpr(kv.Key, keyReg, typ.Key())
				kValue := false // TODO(Gianluca).
				e.emitExpr(kv.Value, valueReg, typ.Elem())
				e.fb.ExitStack()
				e.fb.SetMap(kValue, reg, valueReg, keyReg, typ)
			}
		}

	case *ast.TypeAssertion:
		typ := e.typeInfos[expr.Expr].Type
		exprReg, _, isRegister := e.quickEmitExpr(expr.Expr, typ)
		if !isRegister {
			exprReg = e.fb.NewRegister(typ.Kind())
			e.emitExpr(expr.Expr, exprReg, typ)
		}
		assertType := expr.Type.(*ast.Value).Val.(reflect.Type)
		e.fb.Assert(exprReg, assertType, reg)
		e.fb.Nop()

	case *ast.Selector:

		if ti := e.typeInfos[expr]; ti.IsPredefined() {

			// Predefined function.
			if ti.Type.Kind() == reflect.Func {
				index := e.predefFuncIndex(ti.Value.(reflect.Value), ti.PredefPackageName, expr.Ident)
				e.fb.GetFunc(true, index, reg)
				return
			}

			// Predefined variable.
			index := e.predefVarIndex(ti.Value.(reflect.Value), ti.PredefPackageName, expr.Ident)
			e.fb.GetVar(int(index), reg)
			return
		}

		// Scrigo-defined package variables.
		if index, ok := e.pkgVariables[e.pkg][expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			e.fb.GetVar(int(index), reg) // TODO (Gianluca): to review.
			return
		}

		// Scrigo-defined package functions.
		if sf, ok := e.availableFunctions[e.pkg][expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := e.functionIndex(sf)
			e.fb.GetFunc(false, index, reg)
			return
		}

		// Selector is emitted a general expression.
		exprType := e.typeInfos[expr.Expr].Type
		exprReg := e.fb.NewRegister(exprType.Kind())
		e.emitExpr(expr.Expr, exprReg, exprType)
		field := int8(0) // TODO(Gianluca).
		e.fb.Selector(exprReg, field, reg)

	case *ast.UnaryOperator:
		typ := e.typeInfos[expr.Expr].Type
		var tmpReg int8
		if reg != 0 {
			tmpReg = e.fb.NewRegister(typ.Kind())
		}
		e.emitExpr(expr.Expr, tmpReg, typ)
		if reg == 0 {
			return
		}
		switch expr.Operator() {
		case ast.OperatorNot:
			e.fb.SubInv(true, tmpReg, int8(1), tmpReg, reflect.Int)
			if reg != 0 {
				e.changeRegister(false, tmpReg, reg, typ, dstType)
			}
		case ast.OperatorMultiplication:
			e.changeRegister(false, -tmpReg, reg, typ.Elem(), dstType)
		case ast.OperatorAnd:
			switch expr := expr.Expr.(type) {
			case *ast.Identifier:
				if e.fb.IsVariable(expr.Name) {
					varReg := e.fb.ScopeLookup(expr.Name)
					e.fb.New(reflect.PtrTo(typ), reg)
					e.fb.Move(false, -varReg, reg, dstType.Kind())
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
			e.fb.SubInv(true, tmpReg, 0, tmpReg, dstType.Kind())
			if reg != 0 {
				e.changeRegister(false, tmpReg, reg, typ, dstType)
			}
		case ast.OperatorReceive:
			e.fb.Receive(tmpReg, 0, reg)
		default:
			panic(fmt.Errorf("TODO: not implemented operator %s", expr.Operator()))
		}

	case *ast.Func:

		// Supports scripts function declarations.
		if expr.Ident != nil {
			varReg := e.fb.NewRegister(reflect.Func)
			e.fb.BindVarReg(expr.Ident.Name, varReg)
			expr.Ident = nil // avoids recursive calls.
			funcType := e.typeInfos[expr].Type
			addr := e.newAddress(addressRegister, funcType, varReg, 0)
			e.assign([]address{addr}, []ast.Expression{expr})
			return
		}

		if reg == 0 {
			return
		}

		fn := e.fb.Func(reg, e.typeInfos[expr].Type)
		e.setClosureRefs(fn, expr.Upvars)

		funcLitBuilder := newBuilder(fn)
		funcLitBuilder.SetAlloc(e.addAllocInstructions)
		currFb := e.fb
		e.fb = funcLitBuilder

		e.fb.EnterScope()
		e.prepareFunctionBodyParameters(expr)
		e.EmitNodes(expr.Body.Nodes)
		e.fb.ExitScope()
		e.fb.End()
		e.fb = currFb

	case *ast.Identifier:
		// TODO(Gianluca): review this case.
		if reg == 0 {
			return
		}
		typ := e.typeInfos[expr].Type
		out, isValue, isRegister := e.quickEmitExpr(expr, typ)
		if isValue {
			e.changeRegister(true, out, reg, typ, dstType)
		} else if isRegister {
			e.changeRegister(false, out, reg, typ, dstType)
		} else {
			if fun, ok := e.availableFunctions[e.pkg][expr.Name]; ok {
				index := e.functionIndex(fun)
				e.fb.GetFunc(false, index, reg)
			} else if index, ok := e.upvarsNames[e.fb.fn][expr.Name]; ok {
				// TODO(Gianluca): this is an experimental handling of
				// emitting an expression into a register of a different
				// type. If this is correct, apply this solution to all
				// other expression emitting cases or generalize in some
				// way.
				if kindToType(typ.Kind()) == kindToType(dstType.Kind()) {
					e.fb.GetVar(index, reg)
				} else {
					tmpReg := e.fb.NewRegister(typ.Kind())
					e.fb.GetVar(index, tmpReg)
					e.changeRegister(false, tmpReg, reg, typ, dstType)
				}
			} else if index, ok := e.pkgVariables[e.pkg][expr.Name]; ok {
				if kindToType(typ.Kind()) == kindToType(dstType.Kind()) {
					e.fb.GetVar(int(index), reg)
				} else {
					tmpReg := e.fb.NewRegister(typ.Kind())
					e.fb.GetVar(int(index), tmpReg)
					e.changeRegister(false, tmpReg, reg, typ, dstType)
				}
			} else {
				// Predefined variable.
				if ti := e.typeInfos[expr]; ti.IsPredefined() && ti.Type.Kind() != reflect.Func {
					index := e.predefVarIndex(ti.Value.(reflect.Value), ti.PredefPackageName, expr.Name)
					if kindToType(ti.Type.Kind()) == kindToType(dstType.Kind()) {
						e.fb.GetVar(int(index), reg)
					} else {
						tmpReg := e.fb.NewRegister(ti.Type.Kind())
						e.fb.GetVar(int(index), tmpReg)
						e.changeRegister(false, tmpReg, reg, ti.Type, dstType)
					}
				} else {
					panic("bug")
				}
			}
		}

	case *ast.Value:
		if reg == 0 {
			return
		}
		typ := e.typeInfos[expr].Type
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
				constant := e.fb.MakeIntConstant(int64(v))
				e.fb.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case int8:
				constant := e.fb.MakeIntConstant(int64(v))
				e.fb.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case int16:
				constant := e.fb.MakeIntConstant(int64(v))
				e.fb.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case int32:
				constant := e.fb.MakeIntConstant(int64(v))
				e.fb.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case int64:
				constant := e.fb.MakeIntConstant(int64(v))
				e.fb.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case uint:
				constant := e.fb.MakeIntConstant(int64(v))
				e.fb.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case uint8:
				constant := e.fb.MakeIntConstant(int64(v))
				e.fb.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case uint16:
				constant := e.fb.MakeIntConstant(int64(v))
				e.fb.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case uint32:
				constant := e.fb.MakeIntConstant(int64(v))
				e.fb.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case uint64:
				constant := e.fb.MakeIntConstant(int64(v))
				e.fb.LoadNumber(vm.TypeInt, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case string:
				constant := e.fb.MakeStringConstant(v)
				e.changeRegister(true, constant, reg, typ, dstType)
			case float32:
				constant := e.fb.MakeFloatConstant(float64(v))
				e.fb.LoadNumber(vm.TypeFloat, constant, reg)
			case float64:
				constant := e.fb.MakeFloatConstant(v)
				e.fb.LoadNumber(vm.TypeFloat, constant, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			default:
				constant := e.fb.MakeGeneralConstant(v)
				e.changeRegister(true, constant, reg, typ, dstType)
			}
		}

	case *ast.Index:
		exprType := e.typeInfos[expr.Expr].Type
		var exprReg int8
		out, _, isRegister := e.quickEmitExpr(expr.Expr, exprType)
		if isRegister {
			exprReg = out
		} else {
			exprReg = e.fb.NewRegister(exprType.Kind())
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
			i = e.fb.NewRegister(reflect.Int)
			e.emitExpr(expr.Index, i, dstType)
		}
		e.fb.Index(ki, exprReg, i, reg, exprType)

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
func (e *emitter) quickEmitExpr(expr ast.Expression, expectedType reflect.Type) (out int8, k, isRegister bool) {
	// TODO (Gianluca): quickEmitExpr must evaluate only expression which does
	// not need extra registers for evaluation.

	// Src kind and dst kind are different, so a Move/Conversion is required.
	if kindToType(expectedType.Kind()) != kindToType(e.typeInfos[expr].Type.Kind()) {
		return 0, false, false
	}
	switch expr := expr.(type) {
	case *ast.Identifier:
		if e.fb.IsVariable(expr.Name) {
			return e.fb.ScopeLookup(expr.Name), false, true
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
func (e *emitter) emitBuiltin(call *ast.Call, reg int8, dstType reflect.Type) {
	switch call.Func.(*ast.Identifier).Name {
	case "append":
		sliceType := e.typeInfos[call.Args[0]].Type
		sliceReg := e.fb.NewRegister(sliceType.Kind())
		e.emitExpr(call.Args[0], sliceReg, sliceType)
		tmpSliceReg := e.fb.NewRegister(sliceType.Kind())
		// TODO(Gianluca): moving to a different register is not always
		// necessary. For instance, in case of `s = append(s, t)` moving can
		// be avoided.
		// TODO(Gianluca): in case of append(s, e1, e2, e3) use the length
		// parameter of Append.
		e.fb.Move(false, sliceReg, tmpSliceReg, sliceType.Kind())
		if call.IsVariadic {
			argType := e.typeInfos[call.Args[1]].Type
			argReg := e.fb.NewRegister(argType.Kind())
			e.emitExpr(call.Args[1], argReg, sliceType)
			e.fb.AppendSlice(argReg, tmpSliceReg)
			e.changeRegister(false, tmpSliceReg, reg, sliceType, dstType)
		} else {
			for i := range call.Args {
				if i == 0 {
					continue
				}
				argType := e.typeInfos[call.Args[i]].Type
				argReg := e.fb.NewRegister(argType.Kind())
				e.emitExpr(call.Args[i], argReg, sliceType)
				e.fb.Append(argReg, 1, tmpSliceReg)
			}
			e.changeRegister(false, tmpSliceReg, reg, sliceType, dstType)
		}
	case "cap":
		typ := e.typeInfos[call.Args[0]].Type
		s := e.fb.NewRegister(typ.Kind())
		e.emitExpr(call.Args[0], s, typ)
		e.fb.Cap(s, reg)
		e.changeRegister(false, reg, reg, intType, dstType)
	case "close":
		chanType := e.typeInfos[call.Args[0]].Type
		chanReg := e.fb.NewRegister(chanType.Kind())
		e.emitExpr(call.Args[0], chanReg, chanType)
		e.fb.Close(chanReg)
	case "complex":
		panic("TODO: not implemented")
	case "copy":
		dst, _, isRegister := e.quickEmitExpr(call.Args[0], e.typeInfos[call.Args[0]].Type)
		if !isRegister {
			dst = e.fb.NewRegister(reflect.Slice)
			e.emitExpr(call.Args[0], dst, e.typeInfos[call.Args[0]].Type)
		}
		src, _, isRegister := e.quickEmitExpr(call.Args[1], e.typeInfos[call.Args[1]].Type)
		if !isRegister {
			src = e.fb.NewRegister(reflect.Slice)
			e.emitExpr(call.Args[0], src, e.typeInfos[call.Args[0]].Type)
		}
		e.fb.Copy(dst, src, reg)
		if reg != 0 {
			e.changeRegister(false, reg, reg, intType, dstType)
		}
	case "delete":
		mapExpr := call.Args[0]
		keyExpr := call.Args[1]
		mapp := e.fb.NewRegister(reflect.Interface)
		e.emitExpr(mapExpr, mapp, emptyInterfaceType)
		key := e.fb.NewRegister(reflect.Interface)
		e.emitExpr(keyExpr, key, emptyInterfaceType)
		e.fb.Delete(mapp, key)
	case "imag":
		panic("TODO: not implemented")
	case "html": // TODO (Gianluca): to review.
		panic("TODO: not implemented")
	case "len":
		typ := e.typeInfos[call.Args[0]].Type
		s := e.fb.NewRegister(typ.Kind())
		e.emitExpr(call.Args[0], s, typ)
		e.fb.Len(s, reg, typ)
		e.changeRegister(false, reg, reg, intType, dstType)
	case "make":
		typ := call.Args[0].(*ast.Value).Val.(reflect.Type)
		switch typ.Kind() {
		case reflect.Map:
			if len(call.Args) == 1 {
				e.fb.MakeMap(typ, true, 0, reg)
			} else {
				size, kSize, isRegister := e.quickEmitExpr(call.Args[1], intType)
				if !kSize && !isRegister {
					size = e.fb.NewRegister(reflect.Int)
					e.emitExpr(call.Args[1], size, e.typeInfos[call.Args[1]].Type)
				}
				e.fb.MakeMap(typ, kSize, size, reg)
			}
		case reflect.Slice:
			lenExpr := call.Args[1]
			lenReg, kLen, isRegister := e.quickEmitExpr(lenExpr, intType)
			if !kLen && !isRegister {
				lenReg = e.fb.NewRegister(reflect.Int)
				e.emitExpr(lenExpr, lenReg, e.typeInfos[lenExpr].Type)
			}
			var kCap bool
			var capReg int8
			if len(call.Args) == 3 {
				capExpr := call.Args[2]
				var isRegister bool
				capReg, kCap, isRegister = e.quickEmitExpr(capExpr, intType)
				if !kCap && !isRegister {
					capReg = e.fb.NewRegister(reflect.Int)
					e.emitExpr(capExpr, capReg, e.typeInfos[capExpr].Type)
				}
			} else {
				kCap = kLen
				capReg = lenReg
			}
			e.fb.MakeSlice(kLen, kCap, typ, lenReg, capReg, reg)
		case reflect.Chan:
			chanType := e.typeInfos[call.Args[0]].Type
			var kCapacity bool
			var capacity int8
			if len(call.Args) == 1 {
				capacity = 0
				kCapacity = true
			} else {
				var isRegister bool
				capacity, kCapacity, isRegister = e.quickEmitExpr(call.Args[1], intType)
				if !kCapacity && !isRegister {
					capacity = e.fb.NewRegister(reflect.Int)
					e.emitExpr(call.Args[1], capacity, intType)
				}
			}
			e.fb.MakeChan(chanType, kCapacity, capacity, reg)
		default:
			panic("bug")
		}
	case "new":
		newType := call.Args[0].(*ast.Value).Val.(reflect.Type)
		e.fb.New(newType, reg)
	case "panic":
		arg := call.Args[0]
		reg, _, isRegister := e.quickEmitExpr(arg, emptyInterfaceType)
		if !isRegister {
			reg = e.fb.NewRegister(reflect.Interface)
			e.emitExpr(arg, reg, emptyInterfaceType)
		}
		e.fb.Panic(reg, call.Pos().Line)
	case "print":
		for i := range call.Args {
			arg := e.fb.NewRegister(reflect.Interface)
			e.emitExpr(call.Args[i], arg, emptyInterfaceType)
			e.fb.Print(arg)
		}
	case "println":
		for i := range call.Args {
			arg := e.fb.NewRegister(reflect.Interface)
			e.emitExpr(call.Args[i], arg, emptyInterfaceType)
			e.fb.Print(arg)
			if i < len(call.Args)-1 {
				str := e.fb.MakeStringConstant(" ")
				sep := e.fb.NewRegister(reflect.Interface)
				e.changeRegister(true, str, sep, stringType, emptyInterfaceType)
				e.fb.Print(sep)
			}
		}
	case "real":
		panic("TODO: not implemented")
	case "recover":
		e.fb.Recover(reg)
	default:
		panic("unkown builtin") // TODO(Gianluca): remove.
	}
}

// EmitNodes emits instructions for nodes.
func (e *emitter) EmitNodes(nodes []ast.Node) {
	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.Assignment:
			e.emitAssignmentNode(node)

		case *ast.Block:
			e.fb.EnterScope()
			e.EmitNodes(node.Nodes)
			e.fb.ExitScope()

		case *ast.Break:
			if e.breakable {
				if e.breakLabel == nil {
					label := e.fb.NewLabel()
					e.breakLabel = &label
				}
				e.fb.Goto(*e.breakLabel)
			} else {
				if node.Label != nil {
					panic("TODO(Gianluca): not implemented")
				}
				e.fb.Break(e.rangeLabels[len(e.rangeLabels)-1][0])
				e.fb.Goto(e.rangeLabels[len(e.rangeLabels)-1][1])
			}

		case *ast.Const:
			// Nothing to do.

		case *ast.Continue:
			if node.Label != nil {
				panic("TODO(Gianluca): not implemented")
			}
			e.fb.Continue(e.rangeLabels[len(e.rangeLabels)-1][0])

		case *ast.Defer, *ast.Go:
			if def, ok := node.(*ast.Defer); ok {
				if e.typeInfos[def.Call.Func].IsBuiltin() {
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
			funReg := e.fb.NewRegister(reflect.Func)
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
			funType := e.typeInfos[funNode].Type
			e.emitExpr(funNode, funReg, e.typeInfos[funNode].Type)
			offset := vm.StackShift{
				int8(e.fb.numRegs[reflect.Int]),
				int8(e.fb.numRegs[reflect.Float64]),
				int8(e.fb.numRegs[reflect.String]),
				int8(e.fb.numRegs[reflect.Interface]),
			}
			// TODO(Gianluca): currently supports only deferring or
			// starting goroutines of not predefined functions.
			isPredefined := false
			e.prepareCallParameters(funType, args, isPredefined)
			// TODO(Gianluca): currently supports only deferring functions
			// and starting goroutines with no arguments and no return
			// parameters.
			argsShift := vm.StackShift{}
			switch node.(type) {
			case *ast.Defer:
				e.fb.Defer(funReg, vm.NoVariadic, offset, argsShift)
			case *ast.Go:
				// TODO(Gianluca):
				e.fb.Go()
			}

		case *ast.Import:
			// Nothing to do.

		case *ast.For:
			currentBreakable := e.breakable
			currentBreakLabel := e.breakLabel
			e.breakable = true
			e.breakLabel = nil
			e.fb.EnterScope()
			if node.Init != nil {
				e.EmitNodes([]ast.Node{node.Init})
			}
			if node.Condition != nil {
				forLabel := e.fb.NewLabel()
				e.fb.SetLabelAddr(forLabel)
				e.emitCondition(node.Condition)
				endForLabel := e.fb.NewLabel()
				e.fb.Goto(endForLabel)
				e.EmitNodes(node.Body)
				if node.Post != nil {
					e.EmitNodes([]ast.Node{node.Post})
				}
				e.fb.Goto(forLabel)
				e.fb.SetLabelAddr(endForLabel)
			} else {
				forLabel := e.fb.NewLabel()
				e.fb.SetLabelAddr(forLabel)
				e.EmitNodes(node.Body)
				if node.Post != nil {
					e.EmitNodes([]ast.Node{node.Post})
				}
				e.fb.Goto(forLabel)
			}
			e.fb.ExitScope()
			if e.breakLabel != nil {
				e.fb.SetLabelAddr(*e.breakLabel)
			}
			e.breakable = currentBreakable
			e.breakLabel = currentBreakLabel

		case *ast.ForRange:
			e.fb.EnterScope()
			vars := node.Assignment.Variables
			indexReg := int8(0)
			if len(vars) >= 1 && !isBlankIdentifier(vars[0]) {
				name := vars[0].(*ast.Identifier).Name
				if node.Assignment.Type == ast.AssignmentDeclaration {
					indexReg = e.fb.NewRegister(reflect.Int)
					e.fb.BindVarReg(name, indexReg)
				} else {
					indexReg = e.fb.ScopeLookup(name)
				}
			}
			elemReg := int8(0)
			if len(vars) == 2 && !isBlankIdentifier(vars[1]) {
				typ := e.typeInfos[vars[1]].Type
				name := vars[1].(*ast.Identifier).Name
				if node.Assignment.Type == ast.AssignmentDeclaration {
					elemReg = e.fb.NewRegister(typ.Kind())
					e.fb.BindVarReg(name, elemReg)
				} else {
					elemReg = e.fb.ScopeLookup(name)
				}
			}
			expr := node.Assignment.Values[0]
			exprType := e.typeInfos[expr].Type
			exprReg, kExpr, isRegister := e.quickEmitExpr(expr, exprType)
			if (!kExpr && !isRegister) || exprType.Kind() != reflect.String {
				kExpr = false
				exprReg = e.fb.NewRegister(exprType.Kind())
				e.emitExpr(expr, exprReg, exprType)
			}
			rangeLabel := e.fb.NewLabel()
			e.fb.SetLabelAddr(rangeLabel)
			endRange := e.fb.NewLabel()
			e.rangeLabels = append(e.rangeLabels, [2]uint32{rangeLabel, endRange})
			e.fb.Range(kExpr, exprReg, indexReg, elemReg, exprType.Kind())
			e.fb.Goto(endRange)
			e.fb.EnterScope()
			e.EmitNodes(node.Body)
			e.fb.Continue(rangeLabel)
			e.fb.SetLabelAddr(endRange)
			e.rangeLabels = e.rangeLabels[:len(e.rangeLabels)-1]
			e.fb.ExitScope()
			e.fb.ExitScope()

		case *ast.Goto:
			if label, ok := e.labels[e.fb.fn][node.Label.Name]; ok {
				e.fb.Goto(label)
			} else {
				if e.labels[e.fb.fn] == nil {
					e.labels[e.fb.fn] = make(map[string]uint32)
				}
				label = e.fb.NewLabel()
				e.fb.Goto(label)
				e.labels[e.fb.fn][node.Label.Name] = label
			}

		case *ast.If:
			e.fb.EnterScope()
			if node.Assignment != nil {
				e.EmitNodes([]ast.Node{node.Assignment})
			}
			e.emitCondition(node.Condition)
			if node.Else == nil { // TODO (Gianluca): can "then" and "else" be unified in some way?
				endIfLabel := e.fb.NewLabel()
				e.fb.Goto(endIfLabel)
				e.EmitNodes(node.Then.Nodes)
				e.fb.SetLabelAddr(endIfLabel)
			} else {
				elseLabel := e.fb.NewLabel()
				e.fb.Goto(elseLabel)
				e.EmitNodes(node.Then.Nodes)
				endIfLabel := e.fb.NewLabel()
				e.fb.Goto(endIfLabel)
				e.fb.SetLabelAddr(elseLabel)
				if node.Else != nil {
					switch els := node.Else.(type) {
					case *ast.If:
						e.EmitNodes([]ast.Node{els})
					case *ast.Block:
						e.EmitNodes(els.Nodes)
					}
				}
				e.fb.SetLabelAddr(endIfLabel)
			}
			e.fb.ExitScope()

		case *ast.Label:
			if _, found := e.labels[e.fb.fn][node.Name.Name]; !found {
				if e.labels[e.fb.fn] == nil {
					e.labels[e.fb.fn] = make(map[string]uint32)
				}
				e.labels[e.fb.fn][node.Name.Name] = e.fb.NewLabel()
			}
			e.fb.SetLabelAddr(e.labels[e.fb.fn][node.Name.Name])
			if node.Statement != nil {
				e.EmitNodes([]ast.Node{node.Statement})
			}

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
				typ := e.fb.fn.Type.Out(i)
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
			e.fb.Return()

		case *ast.Send:
			ch := e.fb.NewRegister(reflect.Chan)
			e.emitExpr(node.Channel, ch, e.typeInfos[node.Channel].Type)
			elemType := e.typeInfos[node.Value].Type
			v := e.fb.NewRegister(elemType.Kind())
			e.emitExpr(node.Value, v, elemType)
			e.fb.Send(ch, v)

		case *ast.Switch:
			currentBreakable := e.breakable
			currentBreakLabel := e.breakLabel
			e.breakable = true
			e.breakLabel = nil
			e.emitSwitch(node)
			if e.breakLabel != nil {
				e.fb.SetLabelAddr(*e.breakLabel)
			}
			e.breakable = currentBreakable
			e.breakLabel = currentBreakLabel

		case *ast.TypeSwitch:
			currentBreakable := e.breakable
			currentBreakLabel := e.breakLabel
			e.breakable = true
			e.breakLabel = nil
			e.emitTypeSwitch(node)
			if e.breakLabel != nil {
				e.fb.SetLabelAddr(*e.breakLabel)
			}
			e.breakable = currentBreakable
			e.breakLabel = currentBreakLabel

		case *ast.Var:
			addresses := make([]address, len(node.Lhs))
			for i, v := range node.Lhs {
				staticType := e.typeInfos[v].Type
				if e.indirectVars[v] {
					varReg := -e.fb.NewRegister(reflect.Interface)
					e.fb.BindVarReg(v.Name, varReg)
					addresses[i] = e.newAddress(addressIndirectDeclaration, staticType, varReg, 0)
				} else {
					varReg := e.fb.NewRegister(staticType.Kind())
					e.fb.BindVarReg(v.Name, varReg)
					addresses[i] = e.newAddress(addressRegister, staticType, varReg, 0)
				}
			}
			e.assign(addresses, node.Rhs)

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
func (e *emitter) emitTypeSwitch(node *ast.TypeSwitch) {
	// TODO (Gianluca): a type-checker bug does not replace type switch type with proper value.

	e.fb.EnterScope()

	if node.Init != nil {
		e.EmitNodes([]ast.Node{node.Init})
	}

	typAss := node.Assignment.Values[0].(*ast.TypeAssertion)
	typ := e.typeInfos[typAss.Expr].Type
	expr := e.fb.NewRegister(typ.Kind())
	e.emitExpr(typAss.Expr, expr, typ)

	if len(node.Assignment.Variables) == 1 {
		n := ast.NewAssignment(
			node.Assignment.Pos(),
			[]ast.Expression{node.Assignment.Variables[0]},
			node.Assignment.Type,
			[]ast.Expression{typAss.Expr},
		)
		e.EmitNodes([]ast.Node{n})
	}

	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := e.fb.NewLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = e.fb.NewLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			if isNil(caseExpr) {
				panic("TODO(Gianluca): not implemented")
			}
			caseType := caseExpr.(*ast.Value).Val.(reflect.Type)
			e.fb.Assert(expr, caseType, 0)
			next := e.fb.NewLabel()
			e.fb.Goto(next)
			e.fb.Goto(bodyLabels[i])
			e.fb.SetLabelAddr(next)
		}
	}

	if hasDefault {
		defaultLabel = e.fb.NewLabel()
		e.fb.Goto(defaultLabel)
	} else {
		e.fb.Goto(endSwitchLabel)
	}

	for i, cas := range node.Cases {
		if cas.Expressions == nil {
			e.fb.SetLabelAddr(defaultLabel)
		}
		e.fb.SetLabelAddr(bodyLabels[i])
		e.fb.EnterScope()
		e.EmitNodes(cas.Body)
		e.fb.ExitScope()
		e.fb.Goto(endSwitchLabel)
	}

	e.fb.SetLabelAddr(endSwitchLabel)
	e.fb.ExitScope()
}

// emitSwitch emits instructions for a switch node.
func (e *emitter) emitSwitch(node *ast.Switch) {

	e.fb.EnterScope()

	if node.Init != nil {
		e.EmitNodes([]ast.Node{node.Init})
	}

	var expr int8
	var typ reflect.Type

	if node.Expr == nil {
		typ = reflect.TypeOf(false)
		expr = e.fb.NewRegister(typ.Kind())
		e.fb.Move(true, 1, expr, typ.Kind())
	} else {
		typ = e.typeInfos[node.Expr].Type
		expr = e.fb.NewRegister(typ.Kind())
		e.emitExpr(node.Expr, expr, typ)
	}

	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := e.fb.NewLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = e.fb.NewLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			y, ky, isRegister := e.quickEmitExpr(caseExpr, typ)
			if !ky && !isRegister {
				y = e.fb.NewRegister(typ.Kind())
				e.emitExpr(caseExpr, y, typ)
			}
			e.fb.If(ky, expr, vm.ConditionNotEqual, y, typ.Kind()) // Condizione negata per poter concatenare gli if
			e.fb.Goto(bodyLabels[i])
		}
	}

	if hasDefault {
		defaultLabel = e.fb.NewLabel()
		e.fb.Goto(defaultLabel)
	} else {
		e.fb.Goto(endSwitchLabel)
	}

	for i, cas := range node.Cases {
		if cas.Expressions == nil {
			e.fb.SetLabelAddr(defaultLabel)
		}
		e.fb.SetLabelAddr(bodyLabels[i])
		e.fb.EnterScope()
		e.EmitNodes(cas.Body)
		if !cas.Fallthrough {
			e.fb.Goto(endSwitchLabel)
		}
		e.fb.ExitScope()
	}

	e.fb.SetLabelAddr(endSwitchLabel)

	e.fb.ExitScope()
}

// emitCondition emits instructions for a condition. Last instruction added by
// this method is always "If".
func (e *emitter) emitCondition(cond ast.Expression) {

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
			exprType := e.typeInfos[expr].Type
			x, _, isRegister := e.quickEmitExpr(expr, exprType)
			if !isRegister {
				x = e.fb.NewRegister(exprType.Kind())
				e.emitExpr(expr, x, exprType)
			}
			condType := vm.ConditionNotNil
			if cond.Operator() == ast.OperatorEqual {
				condType = vm.ConditionNil
			}
			e.fb.If(false, x, condType, 0, exprType.Kind())
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
			if e.typeInfos[lenArg].Type.Kind() == reflect.String { // len is optimized for strings only.
				lenArgType := e.typeInfos[lenArg].Type
				x, _, isRegister := e.quickEmitExpr(lenArg, lenArgType)
				if !isRegister {
					x = e.fb.NewRegister(lenArgType.Kind())
					e.emitExpr(lenArg, x, lenArgType)
				}
				exprType := e.typeInfos[expr].Type
				y, ky, isRegister := e.quickEmitExpr(expr, exprType)
				if !ky && !isRegister {
					y = e.fb.NewRegister(exprType.Kind())
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
				e.fb.If(ky, x, condType, y, reflect.String)
				return
			}
		}

		// if v1 == v2
		// if v1 != v2
		// if v1 <  v2
		// if v1 <= v2
		// if v1 >  v2
		// if v1 >= v2
		expr1Type := e.typeInfos[cond.Expr1].Type
		expr2Type := e.typeInfos[cond.Expr2].Type
		if expr1Type.Kind() == expr2Type.Kind() {
			switch kind := expr1Type.Kind(); kind {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64,
				reflect.String:
				expr1Type := e.typeInfos[cond.Expr1].Type
				x, _, isRegister := e.quickEmitExpr(cond.Expr1, expr1Type)
				if !isRegister {
					x = e.fb.NewRegister(expr1Type.Kind())
					e.emitExpr(cond.Expr1, x, expr1Type)
				}
				expr2Type := e.typeInfos[cond.Expr2].Type
				y, ky, isRegister := e.quickEmitExpr(cond.Expr2, expr2Type)
				if !ky && !isRegister {
					y = e.fb.NewRegister(expr2Type.Kind())
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
				e.fb.If(ky, x, condType, y, kind)
				return
			}
		}

	default:

		condType := e.typeInfos[cond].Type
		x, _, isRegister := e.quickEmitExpr(cond, condType)
		if !isRegister {
			x = e.fb.NewRegister(condType.Kind())
			e.emitExpr(cond, x, condType)
		}
		yConst := e.fb.MakeIntConstant(1)
		y := e.fb.NewRegister(reflect.Bool)
		e.fb.LoadNumber(vm.TypeInt, yConst, y)
		e.fb.If(false, x, vm.ConditionEqual, y, reflect.Bool)

	}

}
