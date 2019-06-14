// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"

	"scriggo/internal/compiler/ast"
	"scriggo/vm"
)

// An emitter emits instructions for the VM.
type emitter struct {
	fb           *functionBuilder
	indirectVars map[*ast.Identifier]bool
	labels       map[*vm.Function]map[string]uint32
	pkg          *ast.Package
	typeInfos    map[ast.Node]*TypeInfo
	upvarsNames  map[*vm.Function]map[string]int
	opts         Options

	isTemplate bool // Reports whether it's a template.
	template   struct {
		gA, gB, gC, gD, gE int8 // Reserved general registers.
		iA                 int8 // Reserved int register.
	}

	// Scriggo functions.
	availableFuncs map[*ast.Package]map[string]*vm.Function
	assignedFuncs  map[*vm.Function]map[*vm.Function]int8

	// Scriggo variables.
	pkgVariables map[*ast.Package]map[string]int16

	// Predefined functions.
	predefFunIndexes map[*vm.Function]map[reflect.Value]int8

	// Predefined variables.
	predefVarIndexes map[*vm.Function]map[reflect.Value]int16

	// Holds all Scriggo-defined and pre-predefined global variables.
	globals []Global

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

// newEmitter returns a new emitter with the given type infos and indirect
// variables.
func newEmitter(typeInfos map[ast.Node]*TypeInfo, indirectVars map[*ast.Identifier]bool, opts Options) *emitter {
	c := &emitter{
		assignedFuncs:    map[*vm.Function]map[*vm.Function]int8{},
		availableFuncs:   map[*ast.Package]map[string]*vm.Function{},
		indirectVars:     indirectVars,
		labels:           make(map[*vm.Function]map[string]uint32),
		opts:             opts,
		pkgVariables:     map[*ast.Package]map[string]int16{},
		predefFunIndexes: map[*vm.Function]map[reflect.Value]int8{},
		predefVarIndexes: map[*vm.Function]map[reflect.Value]int16{},
		typeInfos:        typeInfos,
		upvarsNames:      map[*vm.Function]map[string]int{},
	}
	return c
}

func (e *emitter) reserveTemplateRegisters() {
	e.template.gA = e.fb.NewRegister(reflect.Interface) // w io.Writer
	e.template.gB = e.fb.NewRegister(reflect.Interface) // Write
	e.template.gC = e.fb.NewRegister(reflect.Interface) // Render
	e.template.gD = e.fb.NewRegister(reflect.Interface) // free.
	e.template.gE = e.fb.NewRegister(reflect.Interface) // free.
	e.template.iA = e.fb.NewRegister(reflect.Int)       // free.
	e.fb.GetVar(0, e.template.gA)
	e.fb.GetVar(1, e.template.gB)
	e.fb.GetVar(2, e.template.gC)
}

// emitPackage emits package pkg returning exported function, exported
// variables and init functions.
func (e *emitter) emitPackage(pkg *ast.Package, isExtendingPage bool) (map[string]*vm.Function, map[string]int16, []*vm.Function) {
	if !isExtendingPage {
		e.pkg = pkg
		e.availableFuncs[e.pkg] = map[string]*vm.Function{}
	}
	if e.pkgVariables[e.pkg] == nil {
		e.pkgVariables[e.pkg] = map[string]int16{}
	}

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
				funcs, vars, inits := e.emitPackage(pkg, false)
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
						e.availableFuncs[e.pkg][name] = fn
					} else {
						e.availableFuncs[e.pkg][importName+"."+name] = fn
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

	initToBuild := len(allInits) // Index of next "init" function to build.
	if isExtendingPage {
		// If page is extending another page, function declarations have already
		// been added to the list of available functions, so they can't be added
		// twice.
	} else {
		// Stores all function declarations in current package before building
		// their bodies: order of declaration doesn't matter at package level.
		for _, dec := range pkg.Declarations {
			if fun, ok := dec.(*ast.Func); ok {
				fn := newFunction("main", fun.Ident.Name, fun.Type.Reflect)
				if fun.Ident.Name == "init" {
					allInits = append(allInits, fn)
				} else {
					e.availableFuncs[e.pkg][fun.Ident.Name] = fn
					if isExported(fun.Ident.Name) {
						exportedFunctions[fun.Ident.Name] = fn
					}
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
			// a valid Go identifier, so there's no risk of collision with Scriggo
			// defined functions.
			backupFb := e.fb
			if initVarsFn == nil {
				initVarsFn = newFunction("main", "$initvars", reflect.FuncOf(nil, nil, false))
				e.availableFuncs[e.pkg]["$initvars"] = initVarsFn
				initVarsFb = newBuilder(initVarsFn)
				initVarsFb.SetAlloc(e.opts.MemoryLimit)
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
				e.globals = append(e.globals, Global{Pkg: "main", Name: v.Name, Type: staticType})
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
				fn = e.availableFuncs[e.pkg][n.Ident.Name]
			}
			e.fb = newBuilder(fn)
			e.fb.SetAlloc(e.opts.MemoryLimit)
			e.fb.EnterScope()
			// If function is "main", variable initialization functions
			// must be called before everything else inside main's body.
			if n.Ident.Name == "main" {
				// First: initializes package variables.
				if initVarsFn != nil {
					iv := e.availableFuncs[e.pkg]["$initvars"]
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

	// If this package is imported, initFuncs must contain initVarsFn, that is
	// processed as a common "init" function.
	if initVarsFn != nil {
		allInits = append(allInits, initVarsFn)
	}

	return exportedFunctions, exportedVars, allInits

}

// prepareCallParameters prepares the parameters (out and in) for a function
// call. funcType is the reflect type of the function, args are the arguments
// and isPredefined reports whether it is a predefined function.
//
// It returns the registers for the returned values and their respective
// reflect types.
//
// While prepareCallParameters is called before calling the function,
// prepareFunctionBodyParameters is called before emitting the its body.
func (e *emitter) prepareCallParameters(funcType reflect.Type, args []ast.Expression, isPredefined bool, receiverAsArg bool) ([]int8, []reflect.Type) {
	numOut := funcType.NumOut()
	numIn := funcType.NumIn()
	regs := make([]int8, numOut)
	types := make([]reflect.Type, numOut)
	for i := 0; i < numOut; i++ {
		typ := funcType.Out(i)
		regs[i] = e.fb.NewRegister(typ.Kind())
		types[i] = typ
	}
	if receiverAsArg {
		reg := e.fb.NewRegister(e.typeInfos[args[0]].Type.Kind())
		e.fb.EnterStack()
		e.emitExpr(args[0], reg, e.typeInfos[args[0]].Type)
		e.fb.ExitStack()
		args = args[1:]
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

// prepareFunctionBodyParameters prepares fun's parameters (in and out) before
// emitting its body.
//
// While prepareCallParameters is called before calling the function,
// prepareFunctionBodyParameters is called before emitting the its body.
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

	if e.isTemplate {
		e.reserveTemplateRegisters()
	}
}

// emitCall emits instructions for a function call. It returns the registers
// and the reflect types of the returned values.
func (e *emitter) emitCall(call *ast.Call) ([]int8, []reflect.Type) {

	stackShift := vm.StackShift{
		int8(e.fb.numRegs[reflect.Int]),
		int8(e.fb.numRegs[reflect.Float64]),
		int8(e.fb.numRegs[reflect.String]),
		int8(e.fb.numRegs[reflect.Interface]),
	}

	funcTypeInfo := e.typeInfos[call.Func]
	funcType := funcTypeInfo.Type

	// Method call on a interface value.
	if funcTypeInfo.MethodType == MethodCallInterface {
		rcvrExpr := call.Func.(*ast.Selector).Expr
		rcvrType := e.typeInfos[rcvrExpr].Type
		rcvr, k, ok := e.quickEmitExpr(rcvrExpr, rcvrType)
		if !ok || k {
			rcvr = e.fb.NewRegister(rcvrType.Kind())
			e.emitExpr(rcvrExpr, rcvr, rcvrType)
		}
		// MethodValue reads receiver from general.
		if kindToType(rcvrType.Kind()) != vm.TypeGeneral {
			// TODO(Gianluca): put rcvr in general
			panic("not implemented")
		}
		method := e.fb.NewRegister(reflect.Func)
		name := call.Func.(*ast.Selector).Ident
		e.fb.MethodValue(name, rcvr, method)
		call.Args = append([]ast.Expression{rcvrExpr}, call.Args...)
		regs, types := e.prepareCallParameters(funcType, call.Args, true, true)
		e.fb.CallIndirect(method, 0, stackShift)
		return regs, types
	}

	// Predefined function (identifiers, selectors etc...).
	if funcTypeInfo.IsPredefined() {
		if funcTypeInfo.MethodType == MethodCallConcrete {
			rcv := call.Func.(*ast.Selector).Expr // TODO(Gianluca): is this correct?
			call.Args = append([]ast.Expression{rcv}, call.Args...)
		}
		regs, types := e.prepareCallParameters(funcType, call.Args, true, funcTypeInfo.MethodType == MethodCallConcrete)
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

	// Scriggo-defined function (identifier).
	if ident, ok := call.Func.(*ast.Identifier); ok && !e.fb.IsVariable(ident.Name) {
		if fun, ok := e.availableFuncs[e.pkg][ident.Name]; ok {
			regs, types := e.prepareCallParameters(fun.Type, call.Args, false, false)
			index := e.functionIndex(fun)
			e.fb.Call(index, stackShift, call.Pos().Line)
			return regs, types
		}
	}

	// Scriggo-defined function (selector).
	if selector, ok := call.Func.(*ast.Selector); ok {
		if ident, ok := selector.Expr.(*ast.Identifier); ok {
			if fun, ok := e.availableFuncs[e.pkg][ident.Name+"."+selector.Ident]; ok {
				regs, types := e.prepareCallParameters(fun.Type, call.Args, false, false)
				index := e.functionIndex(fun)
				e.fb.Call(index, stackShift, call.Pos().Line)
				return regs, types
			}
		}
	}

	// Indirect function.
	funReg, k, ok := e.quickEmitExpr(call.Func, e.typeInfos[call.Func].Type)
	if !ok || k {
		funReg = e.fb.NewRegister(reflect.Func)
		e.emitExpr(call.Func, funReg, e.typeInfos[call.Func].Type)
	}
	regs, types := e.prepareCallParameters(funcType, call.Args, true, false)
	e.fb.CallIndirect(funReg, 0, stackShift)
	return regs, types
}

// emitExpr emits the instructions that evaluate the expression expr and put
// the result into the register reg. If reg is zero, instructions are emitted
// anyway but the result is discarded.
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

		y, ky, ok := e.quickEmitExpr(expr.Expr2, xType)
		if !ok {
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
		if e.typeInfos[expr.Func] == showMacroIgnoredTi {
			return
		}
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
				e.changeRegister(false, arg, reg, typ, convertType)
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
				value, kvalue, ok := e.quickEmitExpr(kv.Value, typ.Elem())
				if !ok {
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
		exprReg, k, ok := e.quickEmitExpr(expr.Expr, typ)
		if !ok || k {
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

		// Scriggo-defined package variables.
		if index, ok := e.pkgVariables[e.pkg][expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			e.fb.GetVar(int(index), reg) // TODO (Gianluca): to review.
			return
		}

		// Scriggo-defined package functions.
		if sf, ok := e.availableFuncs[e.pkg][expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
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

		// Template macro definition.
		if expr.Ident != nil && e.isTemplate {
			macroFn := newFunction("", expr.Ident.Name, expr.Type.Reflect)
			if e.availableFuncs[e.pkg] == nil {
				e.availableFuncs[e.pkg] = map[string]*vm.Function{}
			}
			e.availableFuncs[e.pkg][expr.Ident.Name] = macroFn
			fb := e.fb
			e.setClosureRefs(macroFn, expr.Upvars)
			e.fb = newBuilder(macroFn)
			e.fb.SetAlloc(e.opts.MemoryLimit)
			e.fb.EnterScope()
			e.prepareFunctionBodyParameters(expr)
			e.EmitNodes(expr.Body.Nodes)
			e.fb.End()
			e.fb.ExitScope()
			e.fb = fb
			return
		}

		// Script function definition.
		if expr.Ident != nil && !e.isTemplate {
			varReg := e.fb.NewRegister(reflect.Func)
			e.fb.BindVarReg(expr.Ident.Name, varReg)
			ident := expr.Ident
			expr.Ident = nil // avoids recursive calls.
			funcType := e.typeInfos[expr].Type
			if e.isTemplate {
				addr := e.newAddress(addressRegister, funcType, varReg, 0)
				e.assign([]address{addr}, []ast.Expression{expr})
			}
			expr.Ident = ident
			return
		}

		if reg == 0 {
			return
		}

		fn := e.fb.Func(reg, e.typeInfos[expr].Type)
		e.setClosureRefs(fn, expr.Upvars)

		funcLitBuilder := newBuilder(fn)
		funcLitBuilder.SetAlloc(e.opts.MemoryLimit)
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
		if out, k, ok := e.quickEmitExpr(expr, typ); ok {
			e.changeRegister(k, out, reg, typ, dstType)
		} else {
			if fun, ok := e.availableFuncs[e.pkg][expr.Name]; ok {
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
		if out, k, ok := e.quickEmitExpr(expr, typ); ok {
			e.changeRegister(k, out, reg, typ, dstType)
		} else {
			v := reflect.ValueOf(expr.Val)
			switch v.Kind() {
			case reflect.Invalid:
				panic("not implemented") // TODO(Gianluca).
			case reflect.Bool:
				b := int64(0)
				if v.Bool() {
					b = 1
				}
				c := e.fb.MakeIntConstant(b)
				e.fb.LoadNumber(vm.TypeInt, c, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case reflect.Int,
				reflect.Int8,
				reflect.Int16,
				reflect.Int32,
				reflect.Int64:
				c := e.fb.MakeIntConstant(v.Int())
				e.fb.LoadNumber(vm.TypeInt, c, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case reflect.Uint,
				reflect.Uint8,
				reflect.Uint16,
				reflect.Uint32,
				reflect.Uint64:
				c := e.fb.MakeIntConstant(int64(v.Uint()))
				e.fb.LoadNumber(vm.TypeInt, c, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case reflect.Uintptr:
				panic("not implemented") // TODO(Gianluca).
			case reflect.Float32,
				reflect.Float64:
				c := e.fb.MakeFloatConstant(v.Float())
				e.fb.LoadNumber(vm.TypeFloat, c, reg)
				e.changeRegister(false, reg, reg, typ, dstType)
			case reflect.Complex64:
				panic("not implemented") // TODO(Gianluca).
			case reflect.Complex128:
				panic("not implemented") // TODO(Gianluca).
			case reflect.Interface:
				panic("not implemented") // TODO(Gianluca).
			case reflect.Slice,
				reflect.Map,
				reflect.Struct,
				reflect.Array,
				reflect.Chan,
				reflect.Ptr,
				reflect.Func:
				c := e.fb.MakeGeneralConstant(v.Interface())
				e.changeRegister(true, c, reg, typ, dstType)
			case reflect.String:
				c := e.fb.MakeStringConstant(v.String())
				e.changeRegister(true, c, reg, typ, dstType)
			case reflect.UnsafePointer:
				panic("not implemented") // TODO(Gianluca).
			}
		}

	case *ast.Index:
		exprType := e.typeInfos[expr.Expr].Type
		indexType := e.typeInfos[expr.Index].Type
		var exprReg int8
		if out, k, ok := e.quickEmitExpr(expr.Expr, exprType); ok && !k {
			exprReg = out
		} else {
			exprReg = e.fb.NewRegister(exprType.Kind())
		}
		var i int8
		out, ki, ok := e.quickEmitExpr(expr.Index, indexType)
		if ok {
			i = out
		} else {
			i = e.fb.NewRegister(indexType.Kind())
			e.emitExpr(expr.Index, i, indexType)
		}
		e.fb.Index(ki, exprReg, i, reg, exprType)

	case *ast.Slicing:
		exprType := e.typeInfos[expr.Expr].Type
		var src int8
		if out, k, ok := e.quickEmitExpr(expr.Expr, exprType); ok && !k {
			src = out
		} else {
			src = e.fb.NewRegister(exprType.Kind())
		}
		var ok bool
		var low, high, max int8 = 0, -1, -1
		var klow, khigh, kmax = true, true, true
		// emit low
		if expr.Low != nil {
			typ := e.typeInfos[expr.Low].Type
			low, klow, ok = e.quickEmitExpr(expr.Low, typ)
			if !ok {
				low = e.fb.NewRegister(typ.Kind())
				e.emitExpr(expr.Low, low, typ)
			}
		}
		// emit high
		if expr.High != nil {
			typ := e.typeInfos[expr.High].Type
			high, khigh, ok = e.quickEmitExpr(expr.High, typ)
			if !ok {
				high = e.fb.NewRegister(typ.Kind())
				e.emitExpr(expr.High, high, typ)
			}
		}
		// emit max
		if expr.Max != nil {
			typ := e.typeInfos[expr.Max].Type
			max, kmax, ok = e.quickEmitExpr(expr.Max, typ)
			if !ok {
				max = e.fb.NewRegister(typ.Kind())
				e.emitExpr(expr.Max, max, typ)
			}
		}
		e.fb.Slice(klow, khigh, kmax, src, reg, low, high, max)

	default:
		panic(fmt.Sprintf("emitExpr currently does not support %T nodes", expr))

	}

}

// quickEmitExpr try to evaluate expr as a constant or a register without
// emitting code, in this case ok is true otherwise is false.
//
// If expr is a constant, out is the constant and k is true.
// if expr is a register, out is the register and k is false.
func (e *emitter) quickEmitExpr(expr ast.Expression, expectedType reflect.Type) (out int8, k, ok bool) {
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
		v := reflect.ValueOf(expr.Val)
		switch v.Kind() {
		case reflect.Int:
			i := v.Int()
			if -127 < i && i < 126 {
				return int8(i), true, true
			}
		case reflect.Bool:
			b := int8(0)
			if v.Bool() {
				b = 1
			}
			return b, true, true
		case reflect.Float64:
			f := v.Float()
			if float64(int(f)) == f {
				if -127 < f && f < 126 {
					return int8(f), true, true
				}
			}
		}
	}
	return 0, false, false
}

// emitBuiltin emits instructions for a builtin call, writing the result, if
// necessary, into the register reg.
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
				e.emitExpr(call.Args[i], argReg, sliceType.Elem())
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
		dst, k, ok := e.quickEmitExpr(call.Args[0], e.typeInfos[call.Args[0]].Type)
		if !ok || k {
			dst = e.fb.NewRegister(reflect.Slice)
			e.emitExpr(call.Args[0], dst, e.typeInfos[call.Args[0]].Type)
		}
		src, k, ok := e.quickEmitExpr(call.Args[1], e.typeInfos[call.Args[1]].Type)
		if !ok || k {
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
				size, kSize, ok := e.quickEmitExpr(call.Args[1], intType)
				if !ok {
					size = e.fb.NewRegister(reflect.Int)
					e.emitExpr(call.Args[1], size, e.typeInfos[call.Args[1]].Type)
				}
				e.fb.MakeMap(typ, kSize, size, reg)
			}
		case reflect.Slice:
			lenExpr := call.Args[1]
			lenReg, kLen, ok := e.quickEmitExpr(lenExpr, intType)
			if !ok {
				lenReg = e.fb.NewRegister(reflect.Int)
				e.emitExpr(lenExpr, lenReg, e.typeInfos[lenExpr].Type)
			}
			var kCap bool
			var capReg int8
			if len(call.Args) == 3 {
				capExpr := call.Args[2]
				var ok bool
				capReg, kCap, ok = e.quickEmitExpr(capExpr, intType)
				if !ok {
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
				var ok bool
				capacity, kCapacity, ok = e.quickEmitExpr(call.Args[1], intType)
				if !ok {
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
		reg, k, ok := e.quickEmitExpr(arg, emptyInterfaceType)
		if !ok || k {
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
			e.prepareCallParameters(funType, args, isPredefined, false)
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
			if e.isTemplate {
				if node.Ident != nil && node.Ident.Name == "_" {
					// Nothing to do: template pages cannot have
					// collateral effects.
				} else {
					backupBuilder := e.fb
					funcs, vars, inits := e.emitPackage(node.Tree.Nodes[0].(*ast.Package), false)
					var importName string
					if node.Ident == nil {
						// Imports without identifiers are handled as 'import . "path"'.
						importName = ""
					} else {
						switch node.Ident.Name {
						case "_":
							panic("TODO(Gianluca): not implemented")
						case ".":
							importName = ""
						default:
							importName = node.Ident.Name
						}
					}
					for name, fn := range funcs {
						if importName == "" {
							e.availableFuncs[e.pkg][name] = fn
						} else {
							e.availableFuncs[e.pkg][importName+"."+name] = fn
						}
					}
					for name, v := range vars {
						if importName == "" {
							e.pkgVariables[e.pkg][name] = v
						} else {
							e.pkgVariables[e.pkg][importName+"."+name] = v
						}
					}
					if len(inits) > 0 {
						panic("have inits!") // TODO(Gianluca): review.
					}
					e.fb = backupBuilder
				}
			}

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
			exprReg, kExpr, ok := e.quickEmitExpr(expr, exprType)
			if !ok || exprType.Kind() != reflect.String {
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

		case *ast.Include:
			e.EmitNodes(node.Tree.Nodes)

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

		case *ast.Show:
			// render([implicit *vm.Env,] gD io.Writer, gE interface{}, iA ast.Context)
			e.emitExpr(node.Expr, e.template.gE, emptyInterfaceType)
			e.fb.Move(true, int8(node.Context), e.template.iA, reflect.Int)
			e.fb.Move(false, e.template.gA, e.template.gD, reflect.Interface)
			e.fb.CallIndirect(e.template.gC, 0, vm.StackShift{e.template.iA - 1, 0, 0, e.template.gC})

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

		case *ast.Text:
			// Write(gE []byte) (iA int, gD error)
			index := len(e.fb.fn.Data)
			e.fb.fn.Data = append(e.fb.fn.Data, node.Text) // TODO(Gianluca): cut text.
			e.fb.LoadData(int16(index), e.template.gE)
			e.fb.CallIndirect(e.template.gB, 0, vm.StackShift{e.template.iA - 1, 0, 0, e.template.gC})

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
			y, ky, ok := e.quickEmitExpr(caseExpr, typ)
			if !ok {
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

// emitCondition emits the instructions for a condition. The last instruction
// emitted is always the "If" instruction
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
			x, k, ok := e.quickEmitExpr(expr, exprType)
			if !ok || k {
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
				x, k, ok := e.quickEmitExpr(lenArg, lenArgType)
				if !ok || k {
					x = e.fb.NewRegister(lenArgType.Kind())
					e.emitExpr(lenArg, x, lenArgType)
				}
				exprType := e.typeInfos[expr].Type
				y, ky, ok := e.quickEmitExpr(expr, exprType)
				if !ok {
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
				x, k, ok := e.quickEmitExpr(cond.Expr1, expr1Type)
				if !ok || k {
					x = e.fb.NewRegister(expr1Type.Kind())
					e.emitExpr(cond.Expr1, x, expr1Type)
				}
				expr2Type := e.typeInfos[cond.Expr2].Type
				y, ky, ok := e.quickEmitExpr(cond.Expr2, expr2Type)
				if !ok {
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
		x, k, ok := e.quickEmitExpr(cond, condType)
		if !ok || k {
			x = e.fb.NewRegister(condType.Kind())
			e.emitExpr(cond, x, condType)
		}
		yConst := e.fb.MakeIntConstant(1)
		y := e.fb.NewRegister(reflect.Bool)
		e.fb.LoadNumber(vm.TypeInt, yConst, y)
		e.fb.If(false, x, vm.ConditionEqual, y, reflect.Bool)

	}

}
