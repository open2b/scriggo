// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"math"
	"reflect"

	"scriggo/ast"
	"scriggo/vm"
)

// An emitter emits instructions for the VM.
type emitter struct {
	fb             *functionBuilder
	indirectVars   map[*ast.Identifier]bool
	labels         map[*vm.Function]map[string]uint32
	pkg            *ast.Package
	typeInfos      map[ast.Node]*TypeInfo
	closureVarRefs map[*vm.Function]map[string]int // Index in the Function VarRefs field for each closure variable.
	options        EmitterOptions

	isTemplate   bool // Reports whether it's a template.
	templateRegs struct {
		gA, gB, gC, gD, gE int8 // Reserved general registers.
		iA                 int8 // Reserved int register.
	}

	// Scriggo functions.
	functions   map[*ast.Package]map[string]*vm.Function
	funcIndexes map[*vm.Function]map[*vm.Function]int8

	// Scriggo variables.
	varIndexes map[*ast.Package]map[string]int16

	// Predefined functions.
	predFunIndexes map[*vm.Function]map[reflect.Value]int8

	// Predefined variables.
	predVarIndexes map[*vm.Function]map[reflect.Value]int16

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

// ti returns the type info of node n.
func (em *emitter) ti(n ast.Node) *TypeInfo {
	if ti, ok := em.typeInfos[n]; ok {
		if ti.valueType != nil {
			ti.Type = ti.valueType
		}
		return ti
	}
	return nil
}

// newEmitter returns a new emitter with the given type infos, indirect
// variables and options.
func newEmitter(typeInfos map[ast.Node]*TypeInfo, indirectVars map[*ast.Identifier]bool, opts EmitterOptions) *emitter {
	return &emitter{
		funcIndexes:    map[*vm.Function]map[*vm.Function]int8{},
		functions:      map[*ast.Package]map[string]*vm.Function{},
		indirectVars:   indirectVars,
		labels:         make(map[*vm.Function]map[string]uint32),
		options:        opts,
		varIndexes:     map[*ast.Package]map[string]int16{},
		predFunIndexes: map[*vm.Function]map[reflect.Value]int8{},
		predVarIndexes: map[*vm.Function]map[reflect.Value]int16{},
		typeInfos:      typeInfos,
		closureVarRefs: map[*vm.Function]map[string]int{},
	}
}

// reserveTemplateRegisters reverses the register used for implement
// specific template functions.
func (em *emitter) reserveTemplateRegisters() {
	em.templateRegs.gA = em.fb.newRegister(reflect.Interface) // w io.Writer
	em.templateRegs.gB = em.fb.newRegister(reflect.Interface) // Write
	em.templateRegs.gC = em.fb.newRegister(reflect.Interface) // Render
	em.templateRegs.gD = em.fb.newRegister(reflect.Interface) // free.
	em.templateRegs.gE = em.fb.newRegister(reflect.Interface) // free.
	em.templateRegs.iA = em.fb.newRegister(reflect.Int)       // free.
	em.fb.emitGetVar(0, em.templateRegs.gA)
	em.fb.emitGetVar(1, em.templateRegs.gB)
	em.fb.emitGetVar(2, em.templateRegs.gC)
}

// emitPackage emits a package and returns the exported functions, the
// exported variables and the init functions.
func (em *emitter) emitPackage(pkg *ast.Package, extendingPage bool) (map[string]*vm.Function, map[string]int16, []*vm.Function) {

	if !extendingPage {
		em.pkg = pkg
		em.functions[em.pkg] = map[string]*vm.Function{}
	}
	if em.varIndexes[em.pkg] == nil {
		em.varIndexes[em.pkg] = map[string]int16{}
	}

	// TODO(Gianluca): if a package is imported more than once, its init
	// functions are called more than once: that is wrong.
	inits := []*vm.Function{} // List of all "init" functions in current package.

	// Emit the imports.
	for _, decl := range pkg.Declarations {
		if imp, ok := decl.(*ast.Import); ok {
			if imp.Tree == nil {
				// Nothing to do. Predefined variables, constants, types
				// and functions are added as information to the tree by
				// the type-checker.
			} else {
				backupPkg := em.pkg
				pkg := imp.Tree.Nodes[0].(*ast.Package)
				funcs, vars, pkgInits := em.emitPackage(pkg, false)
				em.pkg = backupPkg
				inits = append(inits, pkgInits...)
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
						em.functions[em.pkg][name] = fn
					} else {
						em.functions[em.pkg][importName+"."+name] = fn
					}
				}
				for name, v := range vars {
					if importName == "" {
						em.varIndexes[em.pkg][name] = v
					} else {
						em.varIndexes[em.pkg][importName+"."+name] = v
					}
				}
			}
		}
	}

	functions := map[string]*vm.Function{}

	initToBuild := len(inits) // Index of the next "init" function to build.
	if extendingPage {
		// If the page is extending another page, the function declarations
		// have already been added to the list of available functions, so they
		// can't be added twice.
	} else {
		// Store all function declarations in current package before building
		// their bodies: order of declaration doesn't matter at package level.
		for _, dec := range pkg.Declarations {
			if fun, ok := dec.(*ast.Func); ok {
				fn := newFunction("main", fun.Ident.Name, fun.Type.Reflect)
				if fun.Ident.Name == "init" {
					inits = append(inits, fn)
				} else {
					em.functions[em.pkg][fun.Ident.Name] = fn
					if isExported(fun.Ident.Name) {
						functions[fun.Ident.Name] = fn
					}
				}
			}
		}
	}

	vars := map[string]int16{}

	// Emit the package variables.
	var initVarsFn *vm.Function
	var initVarsFb *functionBuilder
	pkgVarRegs := map[string]int8{}
	for _, dec := range pkg.Declarations {
		if n, ok := dec.(*ast.Var); ok {
			// If the package has some variable declarations, a special "init"
			// function must be created to initialize them. "$initvars" is
			// used because is not a valid Go identifier, so there's no risk
			// of collision with Scriggo defined functions.
			backupFb := em.fb
			if initVarsFn == nil {
				initVarsFn = newFunction("main", "$initvars", reflect.FuncOf(nil, nil, false))
				em.functions[em.pkg]["$initvars"] = initVarsFn
				initVarsFb = newBuilder(initVarsFn)
				initVarsFb.emitSetAlloc(em.options.MemoryLimit)
				initVarsFb.enterScope()
			}
			em.fb = initVarsFb
			addresses := make([]address, len(n.Lhs))
			for i, v := range n.Lhs {
				if isBlankIdentifier(v) {
					addresses[i] = em.newAddress(addressBlank, reflect.Type(nil), 0, 0)
				} else {
					staticType := em.ti(v).Type
					varReg := -em.fb.newRegister(reflect.Interface)
					em.fb.bindVarReg(v.Name, varReg)
					addresses[i] = em.newAddress(addressIndirectDeclaration, staticType, varReg, 0)
					// Store the variable register. It will be used later to store
					// initialized value inside the proper global index during
					// the building of $initvars.
					pkgVarRegs[v.Name] = varReg
					em.globals = append(em.globals, Global{Pkg: "main", Name: v.Name, Type: staticType})
					em.varIndexes[em.pkg][v.Name] = int16(len(em.globals) - 1)
					vars[v.Name] = int16(len(em.globals) - 1)
				}
			}
			em.assign(addresses, n.Rhs)
			em.fb = backupFb
		}
	}

	// Emit the function declarations.
	for _, dec := range pkg.Declarations {
		if n, ok := dec.(*ast.Func); ok {
			var fn *vm.Function
			if n.Ident.Name == "init" {
				fn = inits[initToBuild]
				initToBuild++
			} else {
				fn = em.functions[em.pkg][n.Ident.Name]
			}
			em.fb = newBuilder(fn)
			em.fb.emitSetAlloc(em.options.MemoryLimit)
			em.fb.enterScope()
			// If it is the "main" function, variable initialization functions
			// must be called before everything else inside main's body.
			if n.Ident.Name == "main" {
				// First: initialize the package variables.
				if initVarsFn != nil {
					iv := em.functions[em.pkg]["$initvars"]
					index := em.fb.addFunction(iv)
					em.fb.emitCall(int8(index), vm.StackShift{}, 0)
				}
				// Second: call all init functions, in order.
				for _, initFunc := range inits {
					index := em.fb.addFunction(initFunc)
					em.fb.emitCall(int8(index), vm.StackShift{}, 0)
				}
			}
			em.prepareFunctionBodyParameters(n)
			em.EmitNodes(n.Body.Nodes)
			em.fb.end()
			em.fb.exitScope()
		}
	}

	if initVarsFn != nil {
		// Global variables have been locally defined inside the "$initvars"
		// function; their values must now be exported to be available
		// globally.
		backupFb := em.fb
		em.fb = initVarsFb
		for name, reg := range pkgVarRegs {
			index := em.varIndexes[em.pkg][name]
			em.fb.emitSetVar(false, reg, int(index))
		}
		em.fb = backupFb
		initVarsFb.exitScope()
		initVarsFb.emitReturn()
		initVarsFb.end()
	}

	// If this package is imported, initFuncs must contain initVarsFn, that is
	// processed as a common "init" function.
	if initVarsFn != nil {
		inits = append(inits, initVarsFn)
	}

	return functions, vars, inits

}

// prepareCallParameters prepares the parameters (out and in) for a function
// call. funcType is the reflect type of the function, args are the arguments
// and isPredefined reports whether it is a predefined function.
//
// It returns the registers for the returned values and their respective
// reflect types.
//
// While prepareCallParameters is called before calling the function,
// prepareFunctionBodyParameters is called before emitting its body.
func (em *emitter) prepareCallParameters(typ reflect.Type, args []ast.Expression, isPredefined bool, receiverAsArg bool) ([]int8, []reflect.Type) {
	numOut := typ.NumOut()
	numIn := typ.NumIn()
	regs := make([]int8, numOut)
	types := make([]reflect.Type, numOut)
	for i := 0; i < numOut; i++ {
		t := typ.Out(i)
		regs[i] = em.fb.newRegister(t.Kind())
		types[i] = t
	}
	if receiverAsArg {
		reg := em.fb.newRegister(em.ti(args[0]).Type.Kind())
		em.fb.enterStack()
		em.emitExpr(args[0], reg, em.ti(args[0]).Type)
		em.fb.exitStack()
		args = args[1:]
	}
	if typ.IsVariadic() {
		for i := 0; i < numIn-1; i++ {
			t := typ.In(i)
			reg := em.fb.newRegister(t.Kind())
			em.fb.enterScope()
			em.emitExpr(args[i], reg, t)
			em.fb.exitScope()
		}
		if varArgs := len(args) - (numIn - 1); varArgs > 0 {
			t := typ.In(numIn - 1).Elem()
			if isPredefined {
				for i := 0; i < varArgs; i++ {
					reg := em.fb.newRegister(t.Kind())
					em.fb.enterStack()
					em.emitExpr(args[i+numIn-1], reg, t)
					em.fb.exitStack()
				}
			} else {
				sliceReg := int8(numIn)
				em.fb.emitMakeSlice(true, true, typ.In(numIn-1), int8(varArgs), int8(varArgs), sliceReg)
				for i := 0; i < varArgs; i++ {
					tmpReg := em.fb.newRegister(t.Kind())
					em.fb.enterStack()
					em.emitExpr(args[i+numIn-1], tmpReg, t)
					em.fb.exitStack()
					indexReg := em.fb.newRegister(reflect.Int)
					em.fb.emitMove(true, int8(i), indexReg, reflect.Int)
					em.fb.emitSetSlice(false, sliceReg, tmpReg, indexReg, t.Kind())
				}
			}
		}
	} else { // No-variadic function.
		if numIn > 1 && len(args) == 1 { // f(g()), where f takes more than 1 argument.
			regs, types := em.emitCall(args[0].(*ast.Call))
			for i := range regs {
				dstType := typ.In(i)
				reg := em.fb.newRegister(dstType.Kind())
				em.changeRegister(false, regs[i], reg, types[i], dstType)
			}
		} else {
			for i := 0; i < numIn; i++ {
				t := typ.In(i)
				reg := em.fb.newRegister(t.Kind())
				em.fb.enterStack()
				em.emitExpr(args[i], reg, t)
				em.fb.exitStack()
			}
		}
	}
	return regs, types
}

// prepareFunctionBodyParameters prepares fun's parameters (in and out) before
// emitting its body.
//
// While prepareCallParameters is called before calling the function,
// prepareFunctionBodyParameters is called before emitting its body.
func (em *emitter) prepareFunctionBodyParameters(fn *ast.Func) {

	// Reserve space for the return parameters.
	fillParametersTypes(fn.Type.Result)
	for _, res := range fn.Type.Result {
		resType := em.ti(res.Type).Type
		kind := resType.Kind()
		retReg := em.fb.newRegister(kind)
		if res.Ident != nil {
			em.fb.bindVarReg(res.Ident.Name, retReg)
		}
	}
	// Bind the function argument names to pre-allocated registers.
	fillParametersTypes(fn.Type.Parameters)
	for i, par := range fn.Type.Parameters {
		parType := em.ti(par.Type).Type
		kind := parType.Kind()
		if fn.Type.IsVariadic && i == len(fn.Type.Parameters)-1 {
			kind = reflect.Slice
		}
		argReg := em.fb.newRegister(kind)
		if par.Ident != nil {
			em.fb.bindVarReg(par.Ident.Name, argReg)
		}
	}

	if em.isTemplate {
		em.reserveTemplateRegisters()
	}

	return
}

// emitCall emits instructions for a function call. It returns the registers
// and the reflect types of the returned values.
func (em *emitter) emitCall(call *ast.Call) ([]int8, []reflect.Type) {

	stackShift := vm.StackShift{
		int8(em.fb.numRegs[reflect.Int]),
		int8(em.fb.numRegs[reflect.Float64]),
		int8(em.fb.numRegs[reflect.String]),
		int8(em.fb.numRegs[reflect.Interface]),
	}

	ti := em.ti(call.Func)
	typ := ti.Type

	// Method call on a interface value.
	if ti.MethodType == MethodCallInterface {
		rcvrExpr := call.Func.(*ast.Selector).Expr
		rcvrType := em.ti(rcvrExpr).Type
		rcvr, k, ok := em.quickEmitExpr(rcvrExpr, rcvrType)
		if !ok || k {
			rcvr = em.fb.newRegister(rcvrType.Kind())
			em.emitExpr(rcvrExpr, rcvr, rcvrType)
		}
		// MethodValue reads receiver from general.
		if kindToType(rcvrType.Kind()) != vm.TypeGeneral {
			// TODO(Gianluca): put rcvr in general
			panic("not implemented")
		}
		method := em.fb.newRegister(reflect.Func)
		name := call.Func.(*ast.Selector).Ident
		em.fb.emitMethodValue(name, rcvr, method)
		call.Args = append([]ast.Expression{rcvrExpr}, call.Args...)
		regs, types := em.prepareCallParameters(typ, call.Args, true, true)
		em.fb.emitcCallIndirect(method, 0, stackShift)
		return regs, types
	}

	// Predefined function (identifiers, selectors etc...).
	if ti.IsPredefined() {
		if ti.MethodType == MethodCallConcrete {
			rcv := call.Func.(*ast.Selector).Expr // TODO(Gianluca): is this correct?
			call.Args = append([]ast.Expression{rcv}, call.Args...)
		}
		regs, types := em.prepareCallParameters(typ, call.Args, true, ti.MethodType == MethodCallConcrete)
		var name string
		switch f := call.Func.(type) {
		case *ast.Identifier:
			name = f.Name
		case *ast.Selector:
			name = f.Ident
		}
		index := em.predFuncIndex(ti.value.(reflect.Value), ti.PredefPackageName, name)
		if typ.IsVariadic() {
			numVar := len(call.Args) - (typ.NumIn() - 1)
			em.fb.emitCallPredefined(index, int8(numVar), stackShift)
		} else {
			em.fb.emitCallPredefined(index, vm.NoVariadic, stackShift)
		}
		return regs, types
	}

	// Scriggo-defined function (identifier).
	if ident, ok := call.Func.(*ast.Identifier); ok && !em.fb.isVariable(ident.Name) {
		if fn, ok := em.functions[em.pkg][ident.Name]; ok {
			regs, types := em.prepareCallParameters(fn.Type, call.Args, false, false)
			index := em.functionIndex(fn)
			em.fb.emitCall(index, stackShift, call.Pos().Line)
			return regs, types
		}
	}

	// Scriggo-defined function (selector).
	if selector, ok := call.Func.(*ast.Selector); ok {
		if ident, ok := selector.Expr.(*ast.Identifier); ok {
			if fun, ok := em.functions[em.pkg][ident.Name+"."+selector.Ident]; ok {
				regs, types := em.prepareCallParameters(fun.Type, call.Args, false, false)
				index := em.functionIndex(fun)
				em.fb.emitCall(index, stackShift, call.Pos().Line)
				return regs, types
			}
		}
	}

	// Indirect function.
	reg, k, ok := em.quickEmitExpr(call.Func, em.ti(call.Func).Type)
	if !ok || k {
		reg = em.fb.newRegister(reflect.Func)
		em.emitExpr(call.Func, reg, em.ti(call.Func).Type)
	}
	regs, types := em.prepareCallParameters(typ, call.Args, true, false)
	em.fb.emitcCallIndirect(reg, 0, stackShift)

	return regs, types
}

// emitSelector emits selector in register reg.
func (em *emitter) emitSelector(expr *ast.Selector, reg int8, dstType reflect.Type) {

	ti := em.ti(expr)

	// Method value on concrete and interface values.
	if ti.MethodType == MethodValueConcrete || ti.MethodType == MethodValueInterface {
		rcvrExpr := expr.Expr
		rcvrType := em.ti(rcvrExpr).Type
		rcvr, k, ok := em.quickEmitExpr(rcvrExpr, rcvrType)
		if !ok || k {
			rcvr = em.fb.newRegister(rcvrType.Kind())
			em.emitExpr(rcvrExpr, rcvr, rcvrType)
		}
		// MethodValue reads receiver from general.
		if kindToType(rcvrType.Kind()) != vm.TypeGeneral {
			oldRcvr := rcvr
			rcvr = em.fb.newRegister(reflect.Interface)
			em.fb.emitTypify(false, rcvrType, oldRcvr, rcvr)
		}
		if kindToType(dstType.Kind()) == vm.TypeGeneral {
			em.fb.emitMethodValue(expr.Ident, rcvr, reg)
		} else {
			panic("not implemented")
		}
		return
	}

	if ti.IsPredefined() {
		// Predefined function.
		if ti.Type.Kind() == reflect.Func {
			index := em.predFuncIndex(ti.value.(reflect.Value), ti.PredefPackageName, expr.Ident)
			em.fb.emitGetFunc(true, index, reg)
			return
		}
		// Predefined variable.
		index := em.predVarIndex(ti.value.(reflect.Value), ti.PredefPackageName, expr.Ident)
		em.fb.emitGetVar(int(index), reg)
		return
	}

	// Scriggo-defined package variables.
	if index, ok := em.varIndexes[em.pkg][expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
		if reg == 0 {
			return
		}
		if sameRegType(ti.Type.Kind(), dstType.Kind()) {
			em.fb.emitGetVar(int(index), reg)
			return
		}
		tmpReg := em.fb.newRegister(ti.Type.Kind())
		em.fb.emitGetVar(int(index), tmpReg)
		em.changeRegister(false, tmpReg, reg, ti.Type, dstType)
		return
	}

	// Scriggo-defined package functions.
	if sf, ok := em.functions[em.pkg][expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
		if reg == 0 {
			return
		}
		index := em.functionIndex(sf)
		em.fb.emitGetFunc(false, index, reg)
		return
	}

	// Struct field.
	exprType := em.ti(expr.Expr).Type
	exprReg, k, ok := em.quickEmitExpr(expr.Expr, exprType)
	if !ok || k {
		exprReg = em.fb.newRegister(exprType.Kind())
		em.emitExpr(expr.Expr, exprReg, exprType)
	}
	field, _ := exprType.FieldByName(expr.Ident)
	index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
	fieldType := em.ti(expr).Type
	if sameRegType(fieldType.Kind(), dstType.Kind()) {
		em.fb.emitField(exprReg, index, reg)
		return
	}
	tmpReg := em.fb.newRegister(fieldType.Kind())
	em.fb.emitField(exprReg, index, tmpReg)
	em.changeRegister(false, tmpReg, reg, fieldType, dstType)

	return
}

// emitExpr emits the instructions that evaluate the expression expr and put
// the result into the register reg. If reg is zero, instructions are emitted
// anyway but the result is discarded.
func (em *emitter) emitExpr(expr ast.Expression, reg int8, dstType reflect.Type) {

	// If the instructions that emit expr put result in a register type
	// different than the register type of dstType, use an intermediate
	// temporary register. Consider that this is not always necessary to check
	// this: for example if expr is a function, dstType must be a function or an
	// interface (this is guaranteed by the type checker) and in the current
	// implementation of the VM functions and interfaces use the same register
	// type.

	if ti := em.ti(expr); ti != nil && ti.value != nil && !ti.IsPredefined() {
		typ := ti.Type
		if reg == 0 {
			return
		}
		switch v := ti.value.(type) {
		case int64:
			c := em.fb.makeIntConstant(v)
			em.fb.emitLoadNumber(vm.TypeInt, c, reg)
			em.changeRegister(false, reg, reg, typ, dstType)
			return
		case float64:
			c := em.fb.makeFloatConstant(v)
			em.fb.emitLoadNumber(vm.TypeFloat, c, reg)
			em.changeRegister(false, reg, reg, typ, dstType)
			return
		case string:
			c := em.fb.makeStringConstant(v)
			em.changeRegister(true, c, reg, typ, dstType)
			return
		}
		v := reflect.ValueOf(em.ti(expr).value)
		switch v.Kind() {
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
			c := em.fb.makeGeneralConstant(v.Interface())
			em.changeRegister(true, c, reg, typ, dstType)
		case reflect.UnsafePointer:
			panic("not implemented") // TODO(Gianluca).
		default:
			panic(fmt.Errorf("unsupported value type %T (expr: %s)", em.ti(expr).value, expr))
		}
		return
	}

	switch expr := expr.(type) {

	case *ast.BinaryOperator:

		// Binary && and ||.
		if op := expr.Operator(); op == ast.OperatorAndAnd || op == ast.OperatorOrOr {
			cmp := int8(0)
			if op == ast.OperatorAndAnd {
				cmp = 1
			}
			em.fb.enterStack()
			if sameRegType(dstType.Kind(), reflect.Bool) {
				em.emitExpr(expr.Expr1, reg, dstType)
				endIf := em.fb.newLabel()
				em.fb.emitIf(true, reg, vm.ConditionEqual, cmp, reflect.Int)
				em.fb.emitGoto(endIf)
				em.emitExpr(expr.Expr2, reg, dstType)
				em.fb.setLabelAddr(endIf)
			} else {
				tmpReg := em.fb.newRegister(reflect.Bool)
				em.emitExpr(expr.Expr1, tmpReg, boolType)
				endIf := em.fb.newLabel()
				em.fb.emitIf(true, tmpReg, vm.ConditionEqual, cmp, reflect.Int)
				em.fb.emitGoto(endIf)
				em.emitExpr(expr.Expr2, tmpReg, boolType)
				em.fb.setLabelAddr(endIf)
				em.changeRegister(false, tmpReg, reg, boolType, dstType)
			}
			em.fb.exitStack()
			return
		}

		em.fb.enterStack()

		t1 := em.ti(expr.Expr1).Type
		v1 := em.fb.newRegister(t1.Kind())
		em.emitExpr(expr.Expr1, v1, t1)

		v2, k, ok := em.quickEmitExpr(expr.Expr2, t1)
		if !ok {
			v2 = em.fb.newRegister(t1.Kind())
			em.emitExpr(expr.Expr2, v2, t1)
		}

		res := em.fb.newRegister(t1.Kind())

		switch op := expr.Operator(); {
		case op == ast.OperatorAddition && t1.Kind() == reflect.String && reg != 0:
			if k {
				v2 = em.fb.newRegister(reflect.String)
				em.emitExpr(expr.Expr2, v2, t1)
			}
			em.fb.emitConcat(v1, v2, reg)
		case op == ast.OperatorAddition && reg != 0:
			em.fb.emitAdd(k, v1, v2, res, t1.Kind())
			em.changeRegister(false, res, reg, t1, dstType)
		case op == ast.OperatorSubtraction && reg != 0:
			em.fb.emitSub(k, v1, v2, res, t1.Kind())
			em.changeRegister(false, res, reg, t1, dstType)
		case op == ast.OperatorMultiplication && reg != 0:
			em.fb.emitMul(k, v1, v2, res, t1.Kind())
			em.changeRegister(false, res, reg, t1, dstType)
		case op == ast.OperatorDivision && reg != 0:
			em.fb.emitDiv(k, v1, v2, res, t1.Kind())
			em.changeRegister(false, res, reg, t1, dstType)
		case op == ast.OperatorDivision && reg == 0:
			dummyReg := em.fb.newRegister(t1.Kind())
			em.fb.emitDiv(k, v1, v2, dummyReg, t1.Kind()) // produce division by zero.
		case op == ast.OperatorModulo && reg != 0:
			em.fb.emitRem(k, v1, v2, res, t1.Kind())
			em.changeRegister(false, res, reg, t1, dstType)
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
				em.fb.emitMove(true, 1, reg, reflect.Bool)
				em.fb.emitIf(k, v1, cond, v2, t1.Kind())
				em.fb.emitMove(true, 0, reg, reflect.Bool)
			}
		case op == ast.OperatorAnd,
			op == ast.OperatorOr,
			op == ast.OperatorXor,
			op == ast.OperatorAndNot,
			op == ast.OperatorLeftShift,
			op == ast.OperatorRightShift:
			if reg != 0 {
				switch op {
				case ast.OperatorAnd:
					em.fb.emitAnd(k, v1, v2, reg, t1.Kind())
				case ast.OperatorOr:
					em.fb.Or(k, v1, v2, reg, t1.Kind())
				case ast.OperatorXor:
					em.fb.emitXor(k, v1, v2, reg, t1.Kind())
				case ast.OperatorAndNot:
					em.fb.emitAndNot(k, v1, v2, reg, t1.Kind())
				case ast.OperatorLeftShift:
					em.fb.emitLeftShift(k, v1, v2, reg, t1.Kind())
				case ast.OperatorRightShift:
					em.fb.emitRightShift(k, v1, v2, reg, t1.Kind())
				}
				if kindToType(t1.Kind()) != kindToType(dstType.Kind()) {
					em.changeRegister(k, reg, reg, t1, dstType)
				}
			}
		default:
			panic(fmt.Errorf("TODO: not implemented operator %s", expr.Operator()))
		}

		em.fb.exitStack()

	case *ast.Call:

		// ShowMacro which must be ignored (cannot be resolved).
		if em.ti(expr.Func) == showMacroIgnoredTi {
			return
		}

		// Builtin call.
		if em.ti(expr.Func).IsBuiltin() {
			em.emitBuiltin(expr, reg, dstType)
			return
		}

		// Conversion.
		if em.ti(expr.Func).IsType() {
			convertType := em.ti(expr.Func).Type
			// A conversion cannot have side-effects.
			if reg == 0 {
				return
			}
			typ := em.ti(expr.Args[0]).Type
			arg := em.fb.newRegister(typ.Kind())
			em.emitExpr(expr.Args[0], arg, typ)
			if sameRegType(convertType.Kind(), dstType.Kind()) {
				em.changeRegister(false, arg, reg, typ, convertType)
				return
			}
			em.fb.enterStack()
			tmpReg := em.fb.newRegister(convertType.Kind())
			em.changeRegister(false, arg, tmpReg, typ, convertType)
			em.changeRegister(false, tmpReg, reg, convertType, dstType)
			em.fb.exitStack()
			return
		}

		// Function call.
		em.fb.enterStack()
		regs, types := em.emitCall(expr)
		if reg != 0 {
			em.changeRegister(false, regs[0], reg, types[0], dstType)
		}
		em.fb.exitStack()

	case *ast.CompositeLiteral:

		typ := em.ti(expr.Type).Type
		switch typ.Kind() {
		case reflect.Slice, reflect.Array:
			if reg == 0 {
				for _, kv := range expr.KeyValues {
					typ := em.ti(kv.Value).Type
					em.emitExpr(kv.Value, 0, typ)
				}
				return
			}
			length := int8(em.compositeLiteralLen(expr)) // TODO(Gianluca): length is int
			if typ.Kind() == reflect.Array {
				typ = reflect.SliceOf(typ.Elem())
			}
			em.fb.emitMakeSlice(true, true, typ, length, length, reg)
			var index int8 = -1
			for _, kv := range expr.KeyValues {
				if kv.Key != nil {
					index = int8(em.ti(kv.Key).Constant.int64())
				} else {
					index++
				}
				em.fb.enterStack()
				indexReg := em.fb.newRegister(reflect.Int)
				em.fb.emitMove(true, index, indexReg, reflect.Int)
				elem, k, ok := em.quickEmitExpr(kv.Value, typ.Elem())
				if !ok {
					elem = em.fb.newRegister(typ.Elem().Kind())
					em.emitExpr(kv.Value, elem, typ.Elem())
				}
				if reg != 0 {
					em.fb.emitSetSlice(k, reg, elem, indexReg, typ.Elem().Kind())
				}
				em.fb.exitStack()
			}
		case reflect.Struct:
			if reg == 0 {
				for _, kv := range expr.KeyValues {
					typ := em.ti(kv.Value).Type
					em.emitExpr(kv.Value, 0, typ)
				}
				return
			}
			em.fb.enterStack()
			tmpTyp := reflect.PtrTo(typ)
			tmpReg := -em.fb.newRegister(tmpTyp.Kind())
			em.fb.emitNew(typ, -tmpReg)
			if len(expr.KeyValues) > 0 {
				for _, kv := range expr.KeyValues {
					name := kv.Key.(*ast.Identifier).Name
					field, _ := typ.FieldByName(name)
					valueType := em.ti(kv.Value).Type
					var valueReg int8
					if sameRegType(field.Type.Kind(), valueType.Kind()) {
						valueReg = em.fb.newRegister(field.Type.Kind())
						em.emitExpr(kv.Value, valueReg, valueType)
					} else {
						panic("TODO: not implemented") // TODO(Gianluca): to implement.
					}
					// TODO(Gianluca): use field "k" of SetField.
					fieldConstIndex := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
					em.fb.emitSetField(false, tmpReg, fieldConstIndex, valueReg)
				}
			}
			em.changeRegister(false, tmpReg, reg, tmpTyp, dstType)
			em.fb.exitStack()

		case reflect.Map:
			if reg == 0 {
				for _, kv := range expr.KeyValues {
					typ := em.ti(kv.Value).Type
					em.emitExpr(kv.Value, 0, typ)
				}
				return
			}
			// TODO (Gianluca): handle maps with bigger size.
			size := len(expr.KeyValues)
			sizeReg := em.fb.makeIntConstant(int64(size))
			em.fb.emitMakeMap(typ, true, sizeReg, reg)
			for _, kv := range expr.KeyValues {
				keyReg := em.fb.newRegister(typ.Key().Kind())
				valueReg := em.fb.newRegister(typ.Elem().Kind())
				em.fb.enterStack()
				em.emitExpr(kv.Key, keyReg, typ.Key())
				k := false // TODO(Gianluca).
				em.emitExpr(kv.Value, valueReg, typ.Elem())
				em.fb.exitStack()
				em.fb.emitSetMap(k, reg, valueReg, keyReg, typ)
			}
		}

	case *ast.TypeAssertion:

		typ := em.ti(expr.Expr).Type
		exprReg, k, ok := em.quickEmitExpr(expr.Expr, typ)
		if !ok || k {
			exprReg = em.fb.newRegister(typ.Kind())
			em.emitExpr(expr.Expr, exprReg, typ)
		}
		assertType := em.ti(expr.Type).Type
		if sameRegType(assertType.Kind(), dstType.Kind()) {
			em.fb.emitAssert(exprReg, assertType, reg)
			em.fb.emitNop()
		} else {
			tmpReg := em.fb.newRegister(assertType.Kind())
			em.fb.emitAssert(exprReg, assertType, tmpReg)
			em.fb.emitNop()
			em.changeRegister(false, tmpReg, reg, assertType, dstType)
		}

	case *ast.Selector:

		em.emitSelector(expr, reg, dstType)

	case *ast.UnaryOperator:

		typ := em.ti(expr.Expr).Type
		var tmpReg int8
		if reg != 0 {
			tmpReg = em.fb.newRegister(typ.Kind())
		}
		em.emitExpr(expr.Expr, tmpReg, typ)
		if reg == 0 {
			return
		}
		switch expr.Operator() {
		case ast.OperatorNot:
			em.fb.emitSubInv(true, tmpReg, int8(1), tmpReg, reflect.Int)
			if reg != 0 {
				em.changeRegister(false, tmpReg, reg, typ, dstType)
			}
		case ast.OperatorMultiplication:
			em.changeRegister(false, -tmpReg, reg, typ.Elem(), dstType)
		case ast.OperatorAnd:
			switch expr := expr.Expr.(type) {
			case *ast.Identifier:
				if em.fb.isVariable(expr.Name) {
					varReg := em.fb.scopeLookup(expr.Name)
					em.fb.emitNew(reflect.PtrTo(typ), reg)
					em.fb.emitMove(false, -varReg, reg, dstType.Kind())
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
			em.fb.emitSubInv(true, tmpReg, 0, tmpReg, dstType.Kind())
			if reg != 0 {
				em.changeRegister(false, tmpReg, reg, typ, dstType)
			}
		case ast.OperatorReceive:
			em.fb.emitReceive(tmpReg, 0, reg)
		default:
			panic(fmt.Errorf("TODO: not implemented operator %s", expr.Operator()))
		}

	case *ast.Func:

		// Template macro definition.
		if expr.Ident != nil && em.isTemplate {
			macroFn := newFunction("", expr.Ident.Name, expr.Type.Reflect)
			if em.functions[em.pkg] == nil {
				em.functions[em.pkg] = map[string]*vm.Function{}
			}
			em.functions[em.pkg][expr.Ident.Name] = macroFn
			fb := em.fb
			em.setClosureRefs(macroFn, expr.Upvars)
			em.fb = newBuilder(macroFn)
			em.fb.emitSetAlloc(em.options.MemoryLimit)
			em.fb.enterScope()
			em.prepareFunctionBodyParameters(expr)
			em.EmitNodes(expr.Body.Nodes)
			em.fb.end()
			em.fb.exitScope()
			em.fb = fb
			return
		}

		// Script function definition.
		if expr.Ident != nil && !em.isTemplate {
			varReg := em.fb.newRegister(reflect.Func)
			em.fb.bindVarReg(expr.Ident.Name, varReg)
			ident := expr.Ident
			expr.Ident = nil // avoids recursive calls.
			funcType := em.ti(expr).Type
			if em.isTemplate {
				addr := em.newAddress(addressRegister, funcType, varReg, 0)
				em.assign([]address{addr}, []ast.Expression{expr})
			}
			expr.Ident = ident
			return
		}

		if reg == 0 {
			return
		}

		fn := em.fb.emitFunc(reg, em.ti(expr).Type)
		em.setClosureRefs(fn, expr.Upvars)

		funcLitBuilder := newBuilder(fn)
		funcLitBuilder.emitSetAlloc(em.options.MemoryLimit)
		currFB := em.fb
		em.fb = funcLitBuilder

		em.fb.enterScope()
		em.prepareFunctionBodyParameters(expr)
		em.EmitNodes(expr.Body.Nodes)
		em.fb.exitScope()
		em.fb.end()
		em.fb = currFB

	case *ast.Identifier:

		// An identifier evaluation cannot have side effects.
		if reg == 0 {
			return
		}

		typ := em.ti(expr).Type

		// Identifier can be quick-emitted.
		if out, k, ok := em.quickEmitExpr(expr, typ); ok {
			em.changeRegister(k, out, reg, typ, dstType)
			return
		}

		// Identifier represents a function.
		if fun, ok := em.functions[em.pkg][expr.Name]; ok {
			em.fb.emitGetFunc(false, em.functionIndex(fun), reg)
			return
		}

		// Clojure variable.
		if index, ok := em.closureVarRefs[em.fb.fn][expr.Name]; ok {
			if sameRegType(typ.Kind(), dstType.Kind()) {
				em.fb.emitGetVar(index, reg)
				return
			}
			tmpReg := em.fb.newRegister(typ.Kind())
			em.fb.emitGetVar(index, tmpReg)
			em.changeRegister(false, tmpReg, reg, typ, dstType)
			return
		}

		// Scriggo variable.
		if index, ok := em.varIndexes[em.pkg][expr.Name]; ok {
			if sameRegType(typ.Kind(), dstType.Kind()) {
				em.fb.emitGetVar(int(index), reg)
				return
			}
			tmpReg := em.fb.newRegister(typ.Kind())
			em.fb.emitGetVar(int(index), tmpReg)
			em.changeRegister(false, tmpReg, reg, typ, dstType)
			return
		}

		// Predefined variable.
		if ti := em.ti(expr); ti.IsPredefined() && ti.Type.Kind() != reflect.Func {
			index := em.predVarIndex(ti.value.(reflect.Value), ti.PredefPackageName, expr.Name)
			if sameRegType(ti.Type.Kind(), dstType.Kind()) {
				em.fb.emitGetVar(int(index), reg)
				return
			}
			tmpReg := em.fb.newRegister(ti.Type.Kind())
			em.fb.emitGetVar(int(index), tmpReg)
			em.changeRegister(false, tmpReg, reg, ti.Type, dstType)
			return
		}

		panic(fmt.Errorf("bug: none of the previous conditions matched identifier %v", expr))

	case *ast.Index:

		exprType := em.ti(expr.Expr).Type
		indexType := em.ti(expr.Index).Type
		var exprReg int8
		if out, k, ok := em.quickEmitExpr(expr.Expr, exprType); ok && !k {
			exprReg = out
		} else {
			exprReg = em.fb.newRegister(exprType.Kind())
		}
		var i int8
		out, ki, ok := em.quickEmitExpr(expr.Index, indexType)
		if ok {
			i = out
		} else {
			i = em.fb.newRegister(indexType.Kind())
			em.emitExpr(expr.Index, i, indexType)
		}
		if sameRegType(exprType.Elem().Kind(), dstType.Kind()) {
			em.fb.emitIndex(ki, exprReg, i, reg, exprType)
			return
		}
		tmp := em.fb.newRegister(exprType.Elem().Kind())
		em.fb.emitIndex(ki, exprReg, i, tmp, exprType)
		em.changeRegister(false, tmp, reg, exprType.Elem(), dstType)

	case *ast.Slicing:

		exprType := em.ti(expr.Expr).Type
		var src int8
		if out, k, ok := em.quickEmitExpr(expr.Expr, exprType); ok && !k {
			src = out
		} else {
			src = em.fb.newRegister(exprType.Kind())
		}
		var ok bool
		var low, high, max int8 = 0, -1, -1
		var kLow, kHigh, kMax = true, true, true
		// emit low
		if expr.Low != nil {
			typ := em.ti(expr.Low).Type
			low, kLow, ok = em.quickEmitExpr(expr.Low, typ)
			if !ok {
				low = em.fb.newRegister(typ.Kind())
				em.emitExpr(expr.Low, low, typ)
			}
		}
		// emit high
		if expr.High != nil {
			typ := em.ti(expr.High).Type
			high, kHigh, ok = em.quickEmitExpr(expr.High, typ)
			if !ok {
				high = em.fb.newRegister(typ.Kind())
				em.emitExpr(expr.High, high, typ)
			}
		}
		// emit max
		if expr.Max != nil {
			typ := em.ti(expr.Max).Type
			max, kMax, ok = em.quickEmitExpr(expr.Max, typ)
			if !ok {
				max = em.fb.newRegister(typ.Kind())
				em.emitExpr(expr.Max, max, typ)
			}
		}
		em.fb.emitSlice(kLow, kHigh, kMax, src, reg, low, high, max)

	default:

		panic(fmt.Sprintf("emitExpr currently does not support %T nodes (expr: %s)", expr, expr))

	}

	return
}

// quickEmitExpr try to evaluate expr as a constant or a register without
// emitting code, in this case ok is true otherwise is false.
//
// If expr is a constant, out is the constant and k is true.
// if expr is a register, out is the register and k is false.
func (em *emitter) quickEmitExpr(expr ast.Expression, typ reflect.Type) (out int8, k, ok bool) {

	// TODO (Gianluca): quickEmitExpr must evaluate only expression which does
	// not need extra registers for evaluation.

	ti := em.ti(expr)

	// Src kind and dst kind are different, so a Move/Conversion is required.
	if kindToType(typ.Kind()) != kindToType(ti.Type.Kind()) {
		return 0, false, false
	}

	if ti.value != nil && !ti.IsPredefined() {

		switch v := ti.value.(type) {
		case int64:
			if kindToType(typ.Kind()) != vm.TypeInt {
				return 0, false, false
			}
			if -127 < v && v < 126 {
				return int8(v), true, true
			}
		case float64:
			if kindToType(typ.Kind()) != vm.TypeFloat {
				return 0, false, false
			}
			if math.Floor(v) == v && -127 < v && v < 126 {
				return int8(v), true, true
			}
		}
		return 0, false, false
	}

	if expr, ok := expr.(*ast.Identifier); ok && em.fb.isVariable(expr.Name) {
		return em.fb.scopeLookup(expr.Name), false, true
	}

	return 0, false, false
}

// emitBuiltin emits instructions for a builtin call, writing the result, if
// necessary, into the register reg.
func (em *emitter) emitBuiltin(call *ast.Call, reg int8, dstType reflect.Type) {
	switch call.Func.(*ast.Identifier).Name {
	case "append":
		sliceType := em.ti(call.Args[0]).Type
		sliceReg := em.fb.newRegister(sliceType.Kind())
		em.emitExpr(call.Args[0], sliceReg, sliceType)
		tmpSliceReg := em.fb.newRegister(sliceType.Kind())
		// TODO(Gianluca): moving to a different register is not always
		// necessary. For instance, in case of `s = append(s, t)` moving can
		// be avoided.
		// TODO(Gianluca): in case of append(s, e1, e2, e3) use the length
		// parameter of Append.
		em.fb.emitMove(false, sliceReg, tmpSliceReg, sliceType.Kind())
		if call.IsVariadic {
			argType := em.ti(call.Args[1]).Type
			argReg := em.fb.newRegister(argType.Kind())
			em.emitExpr(call.Args[1], argReg, sliceType)
			em.fb.emitAppendSlice(argReg, tmpSliceReg)
			em.changeRegister(false, tmpSliceReg, reg, sliceType, dstType)
		} else {
			for i := range call.Args {
				if i == 0 {
					continue
				}
				argType := em.ti(call.Args[i]).Type
				argReg := em.fb.newRegister(argType.Kind())
				em.emitExpr(call.Args[i], argReg, sliceType.Elem())
				em.fb.emitAppend(argReg, 1, tmpSliceReg)
			}
			em.changeRegister(false, tmpSliceReg, reg, sliceType, dstType)
		}
	case "cap":
		typ := em.ti(call.Args[0]).Type
		s := em.fb.newRegister(typ.Kind())
		em.emitExpr(call.Args[0], s, typ)
		if sameRegType(intType.Kind(), dstType.Kind()) {
			em.fb.emitCap(s, reg)
			return
		}
		tmp := em.fb.newRegister(intType.Kind())
		em.fb.emitCap(s, tmp)
		em.changeRegister(false, tmp, reg, intType, dstType)
	case "close":
		chanType := em.ti(call.Args[0]).Type
		chanReg := em.fb.newRegister(chanType.Kind())
		em.emitExpr(call.Args[0], chanReg, chanType)
		em.fb.emitClose(chanReg)
	case "complex":
		panic("TODO: not implemented")
	case "copy":
		dst, k, ok := em.quickEmitExpr(call.Args[0], em.ti(call.Args[0]).Type)
		if !ok || k {
			dst = em.fb.newRegister(reflect.Slice)
			em.emitExpr(call.Args[0], dst, em.ti(call.Args[0]).Type)
		}
		src, k, ok := em.quickEmitExpr(call.Args[1], em.ti(call.Args[1]).Type)
		if !ok || k {
			src = em.fb.newRegister(reflect.Slice)
			em.emitExpr(call.Args[0], src, em.ti(call.Args[0]).Type)
		}
		em.fb.emitCopy(dst, src, reg)
		if reg != 0 {
			em.changeRegister(false, reg, reg, intType, dstType)
		}
	case "delete":
		mapp := em.fb.newRegister(reflect.Interface)
		em.emitExpr(call.Args[0], mapp, emptyInterfaceType)
		key := em.fb.newRegister(reflect.Interface)
		em.emitExpr(call.Args[1], key, emptyInterfaceType)
		em.fb.emitDelete(mapp, key)
	case "imag":
		panic("TODO: not implemented")
	case "len":
		typ := em.ti(call.Args[0]).Type
		s := em.fb.newRegister(typ.Kind())
		em.emitExpr(call.Args[0], s, typ)
		if sameRegType(intType.Kind(), dstType.Kind()) {
			em.fb.emitLen(s, reg, typ)
			return
		}
		tmp := em.fb.newRegister(intType.Kind())
		em.fb.emitLen(s, tmp, typ)
		em.changeRegister(false, tmp, reg, intType, dstType)
	case "make":
		typ := em.ti(call.Args[0]).Type
		switch typ.Kind() {
		case reflect.Map:
			if len(call.Args) == 1 {
				em.fb.emitMakeMap(typ, true, 0, reg)
			} else {
				size, kSize, ok := em.quickEmitExpr(call.Args[1], intType)
				if !ok {
					size = em.fb.newRegister(reflect.Int)
					em.emitExpr(call.Args[1], size, em.ti(call.Args[1]).Type)
				}
				em.fb.emitMakeMap(typ, kSize, size, reg)
			}
		case reflect.Slice:
			lenExpr := call.Args[1]
			lenReg, kLen, ok := em.quickEmitExpr(lenExpr, intType)
			if !ok {
				lenReg = em.fb.newRegister(reflect.Int)
				em.emitExpr(lenExpr, lenReg, em.ti(lenExpr).Type)
			}
			var kCap bool
			var capReg int8
			if len(call.Args) == 3 {
				capExpr := call.Args[2]
				var ok bool
				capReg, kCap, ok = em.quickEmitExpr(capExpr, intType)
				if !ok {
					capReg = em.fb.newRegister(reflect.Int)
					em.emitExpr(capExpr, capReg, em.ti(capExpr).Type)
				}
			} else {
				kCap = kLen
				capReg = lenReg
			}
			em.fb.emitMakeSlice(kLen, kCap, typ, lenReg, capReg, reg)
		case reflect.Chan:
			chanType := em.ti(call.Args[0]).Type
			var kCapacity bool
			var capacity int8
			if len(call.Args) == 1 {
				capacity = 0
				kCapacity = true
			} else {
				var ok bool
				capacity, kCapacity, ok = em.quickEmitExpr(call.Args[1], intType)
				if !ok {
					capacity = em.fb.newRegister(reflect.Int)
					em.emitExpr(call.Args[1], capacity, intType)
				}
			}
			em.fb.emitMakeChan(chanType, kCapacity, capacity, reg)
		default:
			panic("bug")
		}
	case "new":
		newType := em.ti(call.Args[0]).Type
		em.fb.emitNew(newType, reg)
	case "panic":
		msg := call.Args[0]
		reg, k, ok := em.quickEmitExpr(msg, emptyInterfaceType)
		if !ok || k {
			reg = em.fb.newRegister(reflect.Interface)
			em.emitExpr(msg, reg, emptyInterfaceType)
		}
		em.fb.emitPanic(reg, call.Pos().Line)
	case "print":
		for _, arg := range call.Args {
			reg := em.fb.newRegister(reflect.Interface)
			em.emitExpr(arg, reg, emptyInterfaceType)
			em.fb.emitPrint(reg)
		}
	case "println":
		last := len(call.Args) - 1
		for i, arg := range call.Args {
			reg := em.fb.newRegister(reflect.Interface)
			em.emitExpr(arg, reg, emptyInterfaceType)
			em.fb.emitPrint(reg)
			if i < last {
				str := em.fb.makeStringConstant(" ")
				sep := em.fb.newRegister(reflect.Interface)
				em.changeRegister(true, str, sep, stringType, emptyInterfaceType)
				em.fb.emitPrint(sep)
			}
		}
	case "real":
		panic("TODO: not implemented")
	case "recover":
		em.fb.emitRecover(reg)
	default:
		panic("unknown builtin") // TODO(Gianluca): remove.
	}
}

// EmitNodes emits instructions for nodes.
func (em *emitter) EmitNodes(nodes []ast.Node) {

	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.Assignment:
			em.emitAssignmentNode(node)

		case *ast.Block:
			em.fb.enterScope()
			em.EmitNodes(node.Nodes)
			em.fb.exitScope()

		case *ast.Break:
			if em.breakable {
				if em.breakLabel == nil {
					label := em.fb.newLabel()
					em.breakLabel = &label
				}
				em.fb.emitGoto(*em.breakLabel)
			} else {
				if node.Label != nil {
					panic("TODO(Gianluca): not implemented")
				}
				em.fb.emitBreak(em.rangeLabels[len(em.rangeLabels)-1][0])
				em.fb.emitGoto(em.rangeLabels[len(em.rangeLabels)-1][1])
			}

		case *ast.Const:
			// Nothing to do.

		case *ast.Continue:
			if node.Label != nil {
				panic("TODO(Gianluca): not implemented")
			}
			em.fb.emitContinue(em.rangeLabels[len(em.rangeLabels)-1][0])

		case *ast.Defer, *ast.Go:
			if def, ok := node.(*ast.Defer); ok {
				if em.ti(def.Call.Func).IsBuiltin() {
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
			fnReg := em.fb.newRegister(reflect.Func)
			var fnNode ast.Expression
			var args []ast.Expression
			switch node := node.(type) {
			case *ast.Defer:
				fnNode = node.Call.Func
				args = node.Call.Args
			case *ast.Go:
				fnNode = node.Call.Func
				args = node.Call.Args
			}
			funType := em.ti(fnNode).Type
			em.emitExpr(fnNode, fnReg, em.ti(fnNode).Type)
			offset := vm.StackShift{
				int8(em.fb.numRegs[reflect.Int]),
				int8(em.fb.numRegs[reflect.Float64]),
				int8(em.fb.numRegs[reflect.String]),
				int8(em.fb.numRegs[reflect.Interface]),
			}
			// TODO(Gianluca): currently supports only deferring or
			// starting goroutines of not predefined functions.
			isPredefined := false
			em.prepareCallParameters(funType, args, isPredefined, false)
			// TODO(Gianluca): currently supports only deferring functions
			// and starting goroutines with no arguments and no return
			// parameters.
			argsShift := vm.StackShift{}
			switch node.(type) {
			case *ast.Defer:
				em.fb.emitDefer(fnReg, vm.NoVariadic, offset, argsShift)
			case *ast.Go:
				// TODO(Gianluca):
				em.fb.emitGo()
			}

		case *ast.Import:
			if em.isTemplate {
				if node.Ident != nil && node.Ident.Name == "_" {
					// Nothing to do: template pages cannot have
					// collateral effects.
				} else {
					backupBuilder := em.fb
					backupPkg := em.pkg
					functions, vars, inits := em.emitPackage(node.Tree.Nodes[0].(*ast.Package), false)
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
					if em.functions[backupPkg] == nil {
						em.functions[backupPkg] = map[string]*vm.Function{}
					}
					for name, fn := range functions {
						if importName == "" {
							em.functions[backupPkg][name] = fn
						} else {
							em.functions[backupPkg][importName+"."+name] = fn
						}
					}
					if em.varIndexes[backupPkg] == nil {
						em.varIndexes[backupPkg] = map[string]int16{}
					}
					for name, v := range vars {
						if importName == "" {
							em.varIndexes[backupPkg][name] = v
						} else {
							em.varIndexes[backupPkg][importName+"."+name] = v
						}
					}
					if len(inits) > 0 {
						panic("have inits!") // TODO(Gianluca): review.
					}
					em.fb = backupBuilder
					em.pkg = backupPkg
				}
			}

		case *ast.For:
			currentBreakable := em.breakable
			currentBreakLabel := em.breakLabel
			em.breakable = true
			em.breakLabel = nil
			em.fb.enterScope()
			if node.Init != nil {
				em.EmitNodes([]ast.Node{node.Init})
			}
			if node.Condition != nil {
				forLabel := em.fb.newLabel()
				em.fb.setLabelAddr(forLabel)
				em.emitCondition(node.Condition)
				endForLabel := em.fb.newLabel()
				em.fb.emitGoto(endForLabel)
				em.EmitNodes(node.Body)
				if node.Post != nil {
					em.EmitNodes([]ast.Node{node.Post})
				}
				em.fb.emitGoto(forLabel)
				em.fb.setLabelAddr(endForLabel)
			} else {
				forLabel := em.fb.newLabel()
				em.fb.setLabelAddr(forLabel)
				em.EmitNodes(node.Body)
				if node.Post != nil {
					em.EmitNodes([]ast.Node{node.Post})
				}
				em.fb.emitGoto(forLabel)
			}
			em.fb.exitScope()
			if em.breakLabel != nil {
				em.fb.setLabelAddr(*em.breakLabel)
			}
			em.breakable = currentBreakable
			em.breakLabel = currentBreakLabel

		case *ast.ForRange:
			em.fb.enterScope()
			vars := node.Assignment.Lhs
			indexReg := int8(0)
			if len(vars) >= 1 && !isBlankIdentifier(vars[0]) {
				name := vars[0].(*ast.Identifier).Name
				if node.Assignment.Type == ast.AssignmentDeclaration {
					indexReg = em.fb.newRegister(reflect.Int)
					em.fb.bindVarReg(name, indexReg)
				} else {
					indexReg = em.fb.scopeLookup(name)
				}
			}
			elemReg := int8(0)
			if len(vars) == 2 && !isBlankIdentifier(vars[1]) {
				typ := em.ti(vars[1]).Type
				name := vars[1].(*ast.Identifier).Name
				if node.Assignment.Type == ast.AssignmentDeclaration {
					elemReg = em.fb.newRegister(typ.Kind())
					em.fb.bindVarReg(name, elemReg)
				} else {
					elemReg = em.fb.scopeLookup(name)
				}
			}
			expr := node.Assignment.Rhs[0]
			exprType := em.ti(expr).Type
			exprReg, kExpr, ok := em.quickEmitExpr(expr, exprType)
			if !ok || exprType.Kind() != reflect.String {
				kExpr = false
				exprReg = em.fb.newRegister(exprType.Kind())
				em.emitExpr(expr, exprReg, exprType)
			}
			rangeLabel := em.fb.newLabel()
			em.fb.setLabelAddr(rangeLabel)
			endRange := em.fb.newLabel()
			em.rangeLabels = append(em.rangeLabels, [2]uint32{rangeLabel, endRange})
			em.fb.emitRange(kExpr, exprReg, indexReg, elemReg, exprType.Kind())
			em.fb.emitGoto(endRange)
			em.fb.enterScope()
			em.EmitNodes(node.Body)
			em.fb.emitContinue(rangeLabel)
			em.fb.setLabelAddr(endRange)
			em.rangeLabels = em.rangeLabels[:len(em.rangeLabels)-1]
			em.fb.exitScope()
			em.fb.exitScope()

		case *ast.Goto:
			if label, ok := em.labels[em.fb.fn][node.Label.Name]; ok {
				em.fb.emitGoto(label)
			} else {
				if em.labels[em.fb.fn] == nil {
					em.labels[em.fb.fn] = make(map[string]uint32)
				}
				label = em.fb.newLabel()
				em.fb.emitGoto(label)
				em.labels[em.fb.fn][node.Label.Name] = label
			}

		case *ast.If:
			em.fb.enterScope()
			if node.Assignment != nil {
				em.EmitNodes([]ast.Node{node.Assignment})
			}
			em.emitCondition(node.Condition)
			if node.Else == nil { // TODO (Gianluca): can "then" and "else" be unified in some way?
				endIfLabel := em.fb.newLabel()
				em.fb.emitGoto(endIfLabel)
				em.EmitNodes(node.Then.Nodes)
				em.fb.setLabelAddr(endIfLabel)
			} else {
				elseLabel := em.fb.newLabel()
				em.fb.emitGoto(elseLabel)
				em.EmitNodes(node.Then.Nodes)
				endIfLabel := em.fb.newLabel()
				em.fb.emitGoto(endIfLabel)
				em.fb.setLabelAddr(elseLabel)
				if node.Else != nil {
					switch els := node.Else.(type) {
					case *ast.If:
						em.EmitNodes([]ast.Node{els})
					case *ast.Block:
						em.EmitNodes(els.Nodes)
					}
				}
				em.fb.setLabelAddr(endIfLabel)
			}
			em.fb.exitScope()

		case *ast.Include:
			em.EmitNodes(node.Tree.Nodes)

		case *ast.Label:
			if _, found := em.labels[em.fb.fn][node.Name.Name]; !found {
				if em.labels[em.fb.fn] == nil {
					em.labels[em.fb.fn] = make(map[string]uint32)
				}
				em.labels[em.fb.fn][node.Name.Name] = em.fb.newLabel()
			}
			em.fb.setLabelAddr(em.labels[em.fb.fn][node.Name.Name])
			if node.Statement != nil {
				em.EmitNodes([]ast.Node{node.Statement})
			}

		case *ast.Return:
			// TODO(Gianluca): complete implementation of tail call optimization.
			// if len(node.Rhs) == 1 {
			// 	if call, ok := node.Rhs[0].(*ast.Call); ok {
			// 		tmpRegs := make([]int8, len(call.Args))
			// 		paramPosition := make([]int8, len(call.Args))
			// 		tmpTypes := make([]reflect.Type, len(call.Args))
			// 		shift := vm.StackShift{}
			// 		for i := range call.Args {
			// 			tmpTypes[i] = em.TypeInfo[call.Args[i]].Type
			// 			t := int(kindToType(tmpTypes[i].Kind()))
			// 			tmpRegs[i] = em.FB.newRegister(tmpTypes[i].Kind())
			// 			shift[t]++
			// 			c.compileExpr(call.Args[i], tmpRegs[i], tmpTypes[i])
			// 			paramPosition[i] = shift[t]
			// 		}
			// 		for i := range call.Args {
			// 			em.changeRegister(false, tmpRegs[i], paramPosition[i], tmpTypes[i], em.TypeInfo[call.Func].Type.In(i))
			// 		}
			// 		em.FB.TailCall(vm.CurrentFunction, node.Pos().Line)
			// 		continue
			// 	}
			// }
			offset := [4]int8{}
			for i, v := range node.Values {
				typ := em.fb.fn.Type.Out(i)
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
				em.emitExpr(v, reg, typ)
			}
			em.fb.emitReturn()

		case *ast.Send:
			ch := em.fb.newRegister(reflect.Chan)
			em.emitExpr(node.Channel, ch, em.ti(node.Channel).Type)
			valueType := em.ti(node.Value).Type
			v := em.fb.newRegister(valueType.Kind())
			em.emitExpr(node.Value, v, valueType)
			em.fb.emitSend(ch, v)

		case *ast.Show:
			// render([implicit *vm.Env,] gD io.Writer, gE interface{}, iA ast.Context)
			em.emitExpr(node.Expr, em.templateRegs.gE, emptyInterfaceType)
			em.fb.emitMove(true, int8(node.Context), em.templateRegs.iA, reflect.Int)
			em.fb.emitMove(false, em.templateRegs.gA, em.templateRegs.gD, reflect.Interface)
			em.fb.emitcCallIndirect(em.templateRegs.gC, 0, vm.StackShift{em.templateRegs.iA - 1, 0, 0, em.templateRegs.gC})

		case *ast.Switch:
			currentBreakable := em.breakable
			currentBreakLabel := em.breakLabel
			em.breakable = true
			em.breakLabel = nil
			em.emitSwitch(node)
			if em.breakLabel != nil {
				em.fb.setLabelAddr(*em.breakLabel)
			}
			em.breakable = currentBreakable
			em.breakLabel = currentBreakLabel

		case *ast.Text:
			// Write(gE []byte) (iA int, gD error)
			index := len(em.fb.fn.Data)
			em.fb.fn.Data = append(em.fb.fn.Data, node.Text) // TODO(Gianluca): cut text.
			em.fb.emitLoadData(int16(index), em.templateRegs.gE)
			em.fb.emitcCallIndirect(em.templateRegs.gB, 0, vm.StackShift{em.templateRegs.iA - 1, 0, 0, em.templateRegs.gC})

		case *ast.TypeSwitch:
			currentBreakable := em.breakable
			currentBreakLabel := em.breakLabel
			em.breakable = true
			em.breakLabel = nil
			em.emitTypeSwitch(node)
			if em.breakLabel != nil {
				em.fb.setLabelAddr(*em.breakLabel)
			}
			em.breakable = currentBreakable
			em.breakLabel = currentBreakLabel

		case *ast.Var:
			addresses := make([]address, len(node.Lhs))
			for i, v := range node.Lhs {
				staticType := em.ti(v).Type
				if em.indirectVars[v] {
					varReg := -em.fb.newRegister(reflect.Interface)
					em.fb.bindVarReg(v.Name, varReg)
					addresses[i] = em.newAddress(addressIndirectDeclaration, staticType, varReg, 0)
				} else {
					varReg := em.fb.newRegister(staticType.Kind())
					em.fb.bindVarReg(v.Name, varReg)
					addresses[i] = em.newAddress(addressRegister, staticType, varReg, 0)
				}
			}
			em.assign(addresses, node.Rhs)

		case *ast.Comment:

		case ast.Expression:
			em.emitExpr(node, 0, reflect.Type(nil))

		default:
			panic(fmt.Sprintf("node %T not supported", node)) // TODO(Gianluca): remove.

		}

	}

}

// emitTypeSwitch emits instructions for a type switch node.
func (em *emitter) emitTypeSwitch(node *ast.TypeSwitch) {
	// TODO (Gianluca): a type-checker bug does not replace type switch type with proper value.

	em.fb.enterScope()

	if node.Init != nil {
		em.EmitNodes([]ast.Node{node.Init})
	}

	typAssertion := node.Assignment.Rhs[0].(*ast.TypeAssertion)
	typ := em.ti(typAssertion.Expr).Type
	expr := em.fb.newRegister(typ.Kind())
	em.emitExpr(typAssertion.Expr, expr, typ)

	if len(node.Assignment.Lhs) == 1 {
		n := ast.NewAssignment(
			node.Assignment.Pos(),
			[]ast.Expression{node.Assignment.Lhs[0]},
			node.Assignment.Type,
			[]ast.Expression{typAssertion.Expr},
		)
		em.EmitNodes([]ast.Node{n})
	}

	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := em.fb.newLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = em.fb.newLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			if isNil(caseExpr) {
				panic("TODO(Gianluca): not implemented")
			}
			caseType := em.ti(caseExpr).Type
			em.fb.emitAssert(expr, caseType, 0)
			next := em.fb.newLabel()
			em.fb.emitGoto(next)
			em.fb.emitGoto(bodyLabels[i])
			em.fb.setLabelAddr(next)
		}
	}

	if hasDefault {
		defaultLabel = em.fb.newLabel()
		em.fb.emitGoto(defaultLabel)
	} else {
		em.fb.emitGoto(endSwitchLabel)
	}

	for i, cas := range node.Cases {
		if cas.Expressions == nil {
			em.fb.setLabelAddr(defaultLabel)
		}
		em.fb.setLabelAddr(bodyLabels[i])
		em.fb.enterScope()
		em.EmitNodes(cas.Body)
		em.fb.exitScope()
		em.fb.emitGoto(endSwitchLabel)
	}

	em.fb.setLabelAddr(endSwitchLabel)
	em.fb.exitScope()

	return
}

// emitSwitch emits instructions for a switch node.
func (em *emitter) emitSwitch(node *ast.Switch) {

	em.fb.enterScope()

	if node.Init != nil {
		em.EmitNodes([]ast.Node{node.Init})
	}

	var expr int8
	var typ reflect.Type

	if node.Expr == nil {
		typ = reflect.TypeOf(false)
		expr = em.fb.newRegister(typ.Kind())
		em.fb.emitMove(true, 1, expr, typ.Kind())
	} else {
		typ = em.ti(node.Expr).Type
		expr = em.fb.newRegister(typ.Kind())
		em.emitExpr(node.Expr, expr, typ)
	}

	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := em.fb.newLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = em.fb.newLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			y, ky, ok := em.quickEmitExpr(caseExpr, typ)
			if !ok {
				y = em.fb.newRegister(typ.Kind())
				em.emitExpr(caseExpr, y, typ)
			}
			em.fb.emitIf(ky, expr, vm.ConditionNotEqual, y, typ.Kind()) // Condizione negata per poter concatenare gli if
			em.fb.emitGoto(bodyLabels[i])
		}
	}

	if hasDefault {
		defaultLabel = em.fb.newLabel()
		em.fb.emitGoto(defaultLabel)
	} else {
		em.fb.emitGoto(endSwitchLabel)
	}

	for i, cas := range node.Cases {
		if cas.Expressions == nil {
			em.fb.setLabelAddr(defaultLabel)
		}
		em.fb.setLabelAddr(bodyLabels[i])
		em.fb.enterScope()
		em.EmitNodes(cas.Body)
		if !cas.Fallthrough {
			em.fb.emitGoto(endSwitchLabel)
		}
		em.fb.exitScope()
	}

	em.fb.setLabelAddr(endSwitchLabel)

	em.fb.exitScope()

	return
}

// emitCondition emits the instructions for a condition. The last instruction
// emitted is always the "If" instruction.
func (em *emitter) emitCondition(cond ast.Expression) {

	if ti := em.ti(cond); ti != nil {
		v1, k1, ok := em.quickEmitExpr(cond, ti.Type)
		if !ok || k1 {
			v1 = em.fb.newRegister(ti.Type.Kind())
			em.emitExpr(cond, v1, ti.Type)
		}
		k2 := em.fb.makeIntConstant(1)
		v2 := em.fb.newRegister(reflect.Bool)
		em.fb.emitLoadNumber(vm.TypeInt, k2, v2)
		em.fb.emitIf(false, v1, vm.ConditionEqual, v2, reflect.Bool)
		return
	}

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
			typ := em.ti(expr).Type
			v, k, ok := em.quickEmitExpr(expr, typ)
			if !ok || k {
				v = em.fb.newRegister(typ.Kind())
				em.emitExpr(expr, v, typ)
			}
			condType := vm.ConditionNotNil
			if cond.Operator() == ast.OperatorEqual {
				condType = vm.ConditionNil
			}
			em.fb.emitIf(false, v, condType, 0, typ.Kind())
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
		if em.isLenBuiltinCall(cond.Expr1) != em.isLenBuiltinCall(cond.Expr2) {
			var lenArg, expr ast.Expression
			if em.isLenBuiltinCall(cond.Expr1) {
				lenArg = cond.Expr1.(*ast.Call).Args[0]
				expr = cond.Expr2
			} else {
				lenArg = cond.Expr2.(*ast.Call).Args[0]
				expr = cond.Expr1
			}
			if em.ti(lenArg).Type.Kind() == reflect.String { // len is optimized for strings only.
				lenArgType := em.ti(lenArg).Type
				v1, k1, ok := em.quickEmitExpr(lenArg, lenArgType)
				if !ok || k1 {
					v1 = em.fb.newRegister(lenArgType.Kind())
					em.emitExpr(lenArg, v1, lenArgType)
				}
				typ := em.ti(expr).Type
				v2, k2, ok := em.quickEmitExpr(expr, typ)
				if !ok {
					v2 = em.fb.newRegister(typ.Kind())
					em.emitExpr(expr, v2, typ)
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
				em.fb.emitIf(k2, v1, condType, v2, reflect.String)
				return
			}
		}

		// if v1 == v2
		// if v1 != v2
		// if v1 <  v2
		// if v1 <= v2
		// if v1 >  v2
		// if v1 >= v2
		t1 := em.ti(cond.Expr1).Type
		t2 := em.ti(cond.Expr2).Type
		if t1.Kind() == t2.Kind() {
			switch kind := t1.Kind(); kind {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
				reflect.Float32, reflect.Float64,
				reflect.String:
				v1, k1, ok := em.quickEmitExpr(cond.Expr1, t1)
				if !ok || k1 {
					v1 = em.fb.newRegister(t1.Kind())
					em.emitExpr(cond.Expr1, v1, t1)
				}
				v2, k2, ok := em.quickEmitExpr(cond.Expr2, t2)
				if !ok {
					v2 = em.fb.newRegister(t2.Kind())
					em.emitExpr(cond.Expr2, v2, t2)
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
				if reflect.Uint <= kind && kind <= reflect.Uintptr {
					// Equality and not equality checks are not optimized for uints.
					if condType == vm.ConditionEqual || condType == vm.ConditionNotEqual {
						kind = reflect.Int
					}
				}
				em.fb.emitIf(k2, v1, condType, v2, kind)
				return
			}
		}

	default:

		t := em.ti(cond).Type
		v1, k1, ok := em.quickEmitExpr(cond, t)
		if !ok || k1 {
			v1 = em.fb.newRegister(t.Kind())
			em.emitExpr(cond, v1, t)
		}
		k2 := em.fb.makeIntConstant(1)
		v2 := em.fb.newRegister(reflect.Bool)
		em.fb.emitLoadNumber(vm.TypeInt, k2, v2)
		em.fb.emitIf(false, v1, vm.ConditionEqual, v2, reflect.Bool)

	}

}
