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
					varr := -em.fb.newRegister(reflect.Interface)
					em.fb.bindVarReg(v.Name, varr)
					addresses[i] = em.newAddress(addressIndirectDeclaration, staticType, varr, 0)
					// Store the variable register. It will be used later to store
					// initialized value inside the proper global index during
					// the building of $initvars.
					pkgVarRegs[v.Name] = varr
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
			em.emitNodes(n.Body.Nodes)
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
		em.emitExprR(args[0], em.ti(args[0]).Type, reg)
		em.fb.exitStack()
		args = args[1:]
	}
	if typ.IsVariadic() {
		for i := 0; i < numIn-1; i++ {
			t := typ.In(i)
			reg := em.fb.newRegister(t.Kind())
			em.fb.enterStack()
			em.emitExprR(args[i], t, reg)
			em.fb.exitStack()
		}
		if varArgs := len(args) - (numIn - 1); varArgs > 0 {
			t := typ.In(numIn - 1).Elem()
			if isPredefined {
				for i := 0; i < varArgs; i++ {
					reg := em.fb.newRegister(t.Kind())
					em.fb.enterStack()
					em.emitExprR(args[i+numIn-1], t, reg)
					em.fb.exitStack()
				}
			} else {
				slice := em.fb.newRegister(reflect.Slice)
				em.fb.emitMakeSlice(true, true, typ.In(numIn-1), int8(varArgs), int8(varArgs), slice)
				for i := 0; i < varArgs; i++ {
					tmp := em.fb.newRegister(t.Kind())
					em.fb.enterStack()
					em.emitExprR(args[i+numIn-1], t, tmp)
					em.fb.exitStack()
					index := em.fb.newRegister(reflect.Int)
					em.fb.emitMove(true, int8(i), index, reflect.Int)
					em.fb.emitSetSlice(false, slice, tmp, index)
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
				em.emitExprR(args[i], t, reg)
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
		ret := em.fb.newRegister(kind)
		if res.Ident != nil {
			em.fb.bindVarReg(res.Ident.Name, ret)
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
		arg := em.fb.newRegister(kind)
		if par.Ident != nil {
			em.fb.bindVarReg(par.Ident.Name, arg)
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

	stackShift := em.fb.currentStackShift()

	ti := em.ti(call.Func)
	typ := ti.Type

	// Method call on a interface value.
	if ti.MethodType == MethodCallInterface {
		rcvrExpr := call.Func.(*ast.Selector).Expr
		rcvrType := em.ti(rcvrExpr).Type
		rcvr := em.emitExpr(rcvrExpr, rcvrType)
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
		em.fb.emitCallIndirect(method, 0, stackShift)
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
	reg := em.emitExpr(call.Func, em.ti(call.Func).Type)
	regs, types := em.prepareCallParameters(typ, call.Args, true, false)
	em.fb.emitCallIndirect(reg, 0, stackShift)

	return regs, types
}

// emitSelector emits selector in register reg.
func (em *emitter) emitSelector(expr *ast.Selector, reg int8, dstType reflect.Type) {

	ti := em.ti(expr)

	// Method value on concrete and interface values.
	if ti.MethodType == MethodValueConcrete || ti.MethodType == MethodValueInterface {
		rcvrExpr := expr.Expr
		rcvrType := em.ti(rcvrExpr).Type
		rcvr := em.emitExpr(rcvrExpr, rcvrType)
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
	if ident, ok := expr.Expr.(*ast.Identifier); ok {
		if index, ok := em.varIndexes[em.pkg][ident.Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			if sameRegType(ti.Type.Kind(), dstType.Kind()) {
				em.fb.emitGetVar(int(index), reg)
				return
			}
			tmp := em.fb.newRegister(ti.Type.Kind())
			em.fb.emitGetVar(int(index), tmp)
			em.changeRegister(false, tmp, reg, ti.Type, dstType)
			return
		}

		// Scriggo-defined package functions.
		if sf, ok := em.functions[em.pkg][ident.Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := em.functionIndex(sf)
			em.fb.emitGetFunc(false, index, reg)
			return
		}
	}

	// Struct field.
	exprType := em.ti(expr.Expr).Type
	exprReg := em.emitExpr(expr.Expr, exprType)
	field, _ := exprType.FieldByName(expr.Ident)
	index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
	fieldType := em.ti(expr).Type
	if sameRegType(fieldType.Kind(), dstType.Kind()) {
		em.fb.emitField(exprReg, index, reg)
		return
	}
	tmp := em.fb.newRegister(fieldType.Kind())
	em.fb.emitField(exprReg, index, tmp)
	em.changeRegister(false, tmp, reg, fieldType, dstType)

	return
}

// emitBuiltin emits instructions for a builtin call, writing the result, if
// necessary, into the register reg.
func (em *emitter) emitBuiltin(call *ast.Call, reg int8, dstType reflect.Type) {
	args := call.Args
	switch call.Func.(*ast.Identifier).Name {
	case "append":
		sliceType := em.ti(args[0]).Type
		slice := em.emitExpr(args[0], sliceType)
		if call.IsVariadic {
			tmp := em.fb.newRegister(sliceType.Kind())
			em.fb.emitMove(false, slice, tmp, sliceType.Kind())
			arg := em.emitExpr(args[1], em.ti(args[1]).Type)
			em.fb.emitAppendSlice(arg, tmp)
			em.changeRegister(false, tmp, reg, sliceType, dstType)
			return
		}
		// TODO(Gianluca): moving to a different register is not always
		// necessary. For instance, in case of `s = append(s, t)` moving can be
		// avoided. The problem is that now is too late to check for left-hand
		// symbol which receives the return value of the appending.
		tmp := em.fb.newRegister(sliceType.Kind())
		em.changeRegister(false, slice, tmp, sliceType, sliceType)
		for i := range args {
			if i == 0 {
				continue
			}
			arg := em.emitExpr(args[i], em.ti(args[i]).Type)
			// TODO(Gianluca): in case of append(s, e1, e2, e3) use the length
			// parameter of Append.
			em.fb.emitAppend(arg, 1, tmp)
		}
		em.changeRegister(false, tmp, reg, sliceType, dstType)
	case "cap":
		typ := em.ti(args[0]).Type
		s := em.emitExpr(args[0], typ)
		if sameRegType(intType.Kind(), dstType.Kind()) {
			em.fb.emitCap(s, reg)
			return
		}
		tmp := em.fb.newRegister(intType.Kind())
		em.fb.emitCap(s, tmp)
		em.changeRegister(false, tmp, reg, intType, dstType)
	case "close":
		chann := em.emitExpr(args[0], em.ti(args[0]).Type)
		em.fb.emitClose(chann)
	case "complex":
		panic("TODO: not implemented")
	case "copy":
		dst := em.emitExpr(args[0], em.ti(args[0]).Type)
		src := em.emitExpr(args[1], em.ti(args[1]).Type)
		if reg == 0 {
			em.fb.emitCopy(dst, src, 0)
			return
		}
		if sameRegType(reflect.Int, dstType.Kind()) {
			em.fb.emitCopy(dst, src, reg)
			return
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(reflect.Int)
		em.fb.emitCopy(dst, src, tmp)
		em.changeRegister(false, tmp, reg, intType, dstType)
		em.fb.exitStack()
	case "delete":
		mapp := em.emitExpr(args[0], emptyInterfaceType)
		key := em.emitExpr(args[1], emptyInterfaceType)
		em.fb.emitDelete(mapp, key)
	case "imag":
		panic("TODO: not implemented")
	case "len":
		typ := em.ti(args[0]).Type
		s := em.emitExpr(args[0], typ)
		if sameRegType(reflect.Int, dstType.Kind()) {
			em.fb.emitLen(s, reg, typ)
			return
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(reflect.Int)
		em.fb.emitLen(s, tmp, typ)
		em.changeRegister(false, tmp, reg, intType, dstType)
		em.fb.exitStack()
	case "make":
		typ := em.ti(args[0]).Type
		switch typ.Kind() {
		case reflect.Map:
			if len(args) == 1 {
				em.fb.emitMakeMap(typ, true, 0, reg)
			} else {
				size, kSize := em.emitExprK(args[1], intType)
				em.fb.emitMakeMap(typ, kSize, size, reg)
			}
		case reflect.Slice:
			lenExpr := args[1]
			lenn, kLen := em.emitExprK(lenExpr, intType)
			var kCap bool
			var capp int8
			if len(args) == 3 {
				capArg := args[2]
				capp, kCap = em.emitExprK(capArg, intType)
			} else {
				kCap = kLen
				capp = lenn
			}
			em.fb.emitMakeSlice(kLen, kCap, typ, lenn, capp, reg)
		case reflect.Chan:
			chanType := em.ti(args[0]).Type
			var kCapacity bool
			var capacity int8
			if len(args) == 1 {
				capacity = 0
				kCapacity = true
			} else {
				capacity, kCapacity = em.emitExprK(args[1], intType)
			}
			em.fb.emitMakeChan(chanType, kCapacity, capacity, reg)
		default:
			panic("bug")
		}
	case "new":
		newType := em.ti(args[0]).Type
		em.fb.emitNew(newType, reg)
	case "panic":
		arg := em.emitExpr(args[0], emptyInterfaceType)
		em.fb.emitPanic(arg, call.Pos().Line)
	case "print":
		for _, argExpr := range args {
			arg := em.emitExpr(argExpr, emptyInterfaceType)
			em.fb.emitPrint(arg)
		}
	case "println":
		last := len(args) - 1
		for i, argExpr := range args {
			arg := em.emitExpr(argExpr, emptyInterfaceType)
			em.fb.emitPrint(arg)
			var str int8
			if i < last {
				str = em.fb.makeStringConstant(" ")
			} else {
				str = em.fb.makeStringConstant("\n")
			}
			sep := em.fb.newRegister(reflect.Interface)
			em.changeRegister(true, str, sep, stringType, emptyInterfaceType)
			em.fb.emitPrint(sep)
		}
	case "real":
		panic("TODO: not implemented")
	case "recover":
		em.fb.emitRecover(reg)
	default:
		panic("unknown builtin") // TODO(Gianluca): remove.
	}
}

// emitNodes emits instructions for nodes.
func (em *emitter) emitNodes(nodes []ast.Node) {

	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.Assignment:
			em.emitAssignmentNode(node)

		case *ast.Block:
			em.fb.enterScope()
			em.emitNodes(node.Nodes)
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
			fun := em.fb.newRegister(reflect.Func)
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
			em.emitExprR(fnNode, em.ti(fnNode).Type, fun)
			offset := em.fb.currentStackShift()
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
				em.fb.emitDefer(fun, vm.NoVariadic, offset, argsShift)
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
				em.emitNodes([]ast.Node{node.Init})
			}
			if node.Condition != nil {
				forLabel := em.fb.newLabel()
				em.fb.setLabelAddr(forLabel)
				em.emitCondition(node.Condition)
				endForLabel := em.fb.newLabel()
				em.fb.emitGoto(endForLabel)
				em.emitNodes(node.Body)
				if node.Post != nil {
					em.emitNodes([]ast.Node{node.Post})
				}
				em.fb.emitGoto(forLabel)
				em.fb.setLabelAddr(endForLabel)
			} else {
				forLabel := em.fb.newLabel()
				em.fb.setLabelAddr(forLabel)
				em.emitNodes(node.Body)
				if node.Post != nil {
					em.emitNodes([]ast.Node{node.Post})
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
			elem := int8(0)
			if len(vars) == 2 && !isBlankIdentifier(vars[1]) {
				typ := em.ti(vars[1]).Type
				name := vars[1].(*ast.Identifier).Name
				if node.Assignment.Type == ast.AssignmentDeclaration {
					elem = em.fb.newRegister(typ.Kind())
					em.fb.bindVarReg(name, elem)
				} else {
					elem = em.fb.scopeLookup(name)
				}
			}
			expr := node.Assignment.Rhs[0]
			exprType := em.ti(expr).Type
			exprReg, kExpr := em.emitExprK(expr, exprType)
			if exprType.Kind() != reflect.String && kExpr {
				kExpr = false
				exprReg = em.emitExpr(expr, exprType)
			}
			rangeLabel := em.fb.newLabel()
			em.fb.setLabelAddr(rangeLabel)
			endRange := em.fb.newLabel()
			em.rangeLabels = append(em.rangeLabels, [2]uint32{rangeLabel, endRange})
			em.fb.emitRange(kExpr, exprReg, indexReg, elem, exprType.Kind())
			em.fb.emitGoto(endRange)
			em.fb.enterScope()
			em.emitNodes(node.Body)
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
				em.emitNodes([]ast.Node{node.Assignment})
			}
			em.emitCondition(node.Condition)
			if node.Else == nil { // TODO (Gianluca): can "then" and "else" be unified in some way?
				endIfLabel := em.fb.newLabel()
				em.fb.emitGoto(endIfLabel)
				em.emitNodes(node.Then.Nodes)
				em.fb.setLabelAddr(endIfLabel)
			} else {
				elseLabel := em.fb.newLabel()
				em.fb.emitGoto(elseLabel)
				em.emitNodes(node.Then.Nodes)
				endIfLabel := em.fb.newLabel()
				em.fb.emitGoto(endIfLabel)
				em.fb.setLabelAddr(elseLabel)
				if node.Else != nil {
					switch els := node.Else.(type) {
					case *ast.If:
						em.emitNodes([]ast.Node{els})
					case *ast.Block:
						em.emitNodes(els.Nodes)
					}
				}
				em.fb.setLabelAddr(endIfLabel)
			}
			em.fb.exitScope()

		case *ast.Include:
			em.emitNodes(node.Tree.Nodes)

		case *ast.Label:
			if _, found := em.labels[em.fb.fn][node.Name.Name]; !found {
				if em.labels[em.fb.fn] == nil {
					em.labels[em.fb.fn] = make(map[string]uint32)
				}
				em.labels[em.fb.fn][node.Name.Name] = em.fb.newLabel()
			}
			em.fb.setLabelAddr(em.labels[em.fb.fn][node.Name.Name])
			if node.Statement != nil {
				em.emitNodes([]ast.Node{node.Statement})
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
				em.emitExprR(v, typ, reg)
			}
			em.fb.emitReturn()

		case *ast.Send:
			chann := em.emitExpr(node.Channel, em.ti(node.Channel).Type)
			value := em.emitExpr(node.Value, em.ti(node.Value).Type)
			em.fb.emitSend(chann, value)

		case *ast.Show:
			// render([implicit *vm.Env,] gD io.Writer, gE interface{}, iA ast.Context)
			em.emitExprR(node.Expr, emptyInterfaceType, em.templateRegs.gE)
			em.fb.emitMove(true, int8(node.Context), em.templateRegs.iA, reflect.Int)
			em.fb.emitMove(false, em.templateRegs.gA, em.templateRegs.gD, reflect.Interface)
			em.fb.emitCallIndirect(em.templateRegs.gC, 0, vm.StackShift{em.templateRegs.iA - 1, 0, 0, em.templateRegs.gC})

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
			em.fb.emitCallIndirect(em.templateRegs.gB, 0, vm.StackShift{em.templateRegs.iA - 1, 0, 0, em.templateRegs.gC})

		case *ast.TypeDeclaration:
			// Nothing to do.

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
					varr := -em.fb.newRegister(reflect.Interface)
					em.fb.bindVarReg(v.Name, varr)
					addresses[i] = em.newAddress(addressIndirectDeclaration, staticType, varr, 0)
				} else {
					varr := em.fb.newRegister(staticType.Kind())
					em.fb.bindVarReg(v.Name, varr)
					addresses[i] = em.newAddress(addressRegister, staticType, varr, 0)
				}
			}
			em.assign(addresses, node.Rhs)

		case *ast.Comment:

		case ast.Expression:
			em.emitExprR(node, reflect.Type(nil), 0)

		default:
			panic(fmt.Sprintf("node %T not supported", node)) // TODO(Gianluca): remove.

		}

	}

}

// emitExpr emits expr into a register of a given type. emitExpr tries to not
// create a new register, but to use an existing one. The register used for
// emission is returned.
// TODO(Gianluca): add an option/another method to force the creation of an new
// register? Is necessary?
func (em *emitter) emitExpr(expr ast.Expression, dstType reflect.Type) int8 {
	reg, _ := em._emitExpr(expr, dstType, 0, false, false)
	return reg
}

// emitExprK emits expr into a register of a given type. The boolean return
// parameter reports whether the returned int8 is a constant or not.
func (em *emitter) emitExprK(expr ast.Expression, dstType reflect.Type) (int8, bool) {
	return em._emitExpr(expr, dstType, 0, false, true)
}

// emitExprR emits expr into register reg with the given type.
func (em *emitter) emitExprR(expr ast.Expression, dstType reflect.Type, reg int8) {
	_, _ = em._emitExpr(expr, dstType, reg, true, false)
}

// _emitExpr emits expression expr.
//
// If a register is given and putInReg is true, then such register is used for
// emission; otherwise _emitExpr chooses the output register, returning it to
// the caller.
//
// If allowK is true, then the returned register can be an immediante value and
// the boolean return parameters is true.
//
// _emitExpr is an internal support method, and should be called by emitExpr,
// emitExprK and emitExprR exclusively.
//
func (em *emitter) _emitExpr(expr ast.Expression, dstType reflect.Type, reg int8, useGivenReg bool, allowK bool) (int8, bool) {

	if allowK && !useGivenReg {
		ti := em.ti(expr)
		if ti.value != nil && !ti.IsPredefined() {
			switch v := ti.value.(type) {
			case int64:
				if kindToType(dstType.Kind()) == vm.TypeInt {
					if -127 < v && v < 126 {
						return int8(v), true
					}
				}
			case float64:
				if kindToType(dstType.Kind()) == vm.TypeFloat {
					if math.Floor(v) == v && -127 < v && v < 126 {
						return int8(v), true
					}
				}
			}
		}
		if expr, ok := expr.(*ast.Identifier); ok && em.fb.isVariable(expr.Name) {
			return em.fb.scopeLookup(expr.Name), false
		}
	}

	if !useGivenReg {
		if expr, ok := expr.(*ast.Identifier); ok && em.fb.isVariable(expr.Name) {
			return em.fb.scopeLookup(expr.Name), false
		}
		reg = em.fb.newRegister(dstType.Kind())
	}

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
			return reg, false
		}
		switch v := ti.value.(type) {
		case int64:
			c := em.fb.makeIntConstant(v)
			if sameRegType(typ.Kind(), dstType.Kind()) {
				em.fb.emitLoadNumber(vm.TypeInt, c, reg)
				em.changeRegister(false, reg, reg, typ, dstType)
				return reg, false
			}
			tmp := em.fb.newRegister(typ.Kind())
			em.fb.emitLoadNumber(vm.TypeInt, c, tmp)
			em.changeRegister(false, tmp, reg, typ, dstType)
			return reg, false
		case float64:
			c := em.fb.makeFloatConstant(v)
			if sameRegType(typ.Kind(), dstType.Kind()) {
				em.fb.emitLoadNumber(vm.TypeFloat, c, reg)
				em.changeRegister(false, reg, reg, typ, dstType)
				return reg, false
			}
			tmp := em.fb.newRegister(typ.Kind())
			em.fb.emitLoadNumber(vm.TypeFloat, c, tmp)
			em.changeRegister(false, tmp, reg, typ, dstType)
			return reg, false
		case string:
			c := em.fb.makeStringConstant(v)
			em.changeRegister(true, c, reg, typ, dstType)
			return reg, false
		}
		v := reflect.ValueOf(em.ti(expr).value)
		switch v.Kind() {
		case reflect.Interface:
			panic("not implemented") // TODO(Gianluca).
		case reflect.Slice,
			reflect.Complex64,
			reflect.Complex128,
			reflect.Array,
			reflect.Chan,
			reflect.Func,
			reflect.Map,
			reflect.Ptr,
			reflect.Struct:
			c := em.fb.makeGeneralConstant(v.Interface())
			em.changeRegister(true, c, reg, typ, dstType)
		case reflect.UnsafePointer:
			panic("not implemented") // TODO(Gianluca).
		default:
			panic(fmt.Errorf("unsupported value type %T (expr: %s)", em.ti(expr).value, expr))
		}
		return reg, false
	}
	switch expr := expr.(type) {

	case *ast.BinaryOperator:

		// Binary operations on complex numbers.
		if exprType := em.ti(expr).Type; exprType.Kind() == reflect.Complex64 || exprType.Kind() == reflect.Complex128 {
			stackShift := em.fb.currentStackShift()
			em.fb.enterScope()
			index := em.fb.complexOperationIndex(expr.Operator(), false)
			ret := em.fb.newRegister(reflect.Complex128)
			c1 := em.fb.newRegister(reflect.Complex128)
			c2 := em.fb.newRegister(reflect.Complex128)
			em.fb.enterScope()
			em.emitExprR(expr.Expr1, exprType, c1)
			em.fb.exitScope()
			em.fb.enterScope()
			em.emitExprR(expr.Expr2, exprType, c2)
			em.fb.exitScope()
			em.fb.emitCallPredefined(index, 0, stackShift)
			em.changeRegister(false, ret, reg, exprType, dstType)
			em.fb.exitScope()
			return reg, false
		}

		// Binary && and ||.
		if op := expr.Operator(); op == ast.OperatorAndAnd || op == ast.OperatorOrOr {
			cmp := int8(0)
			if op == ast.OperatorAndAnd {
				cmp = 1
			}
			if sameRegType(dstType.Kind(), reflect.Bool) {
				em.emitExprR(expr.Expr1, dstType, reg)
				endIf := em.fb.newLabel()
				em.fb.emitIf(true, reg, vm.ConditionEqual, cmp, reflect.Int)
				em.fb.emitGoto(endIf)
				em.emitExprR(expr.Expr2, dstType, reg)
				em.fb.setLabelAddr(endIf)
				return reg, false
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(reflect.Bool)
			em.emitExprR(expr.Expr1, boolType, tmp)
			endIf := em.fb.newLabel()
			em.fb.emitIf(true, tmp, vm.ConditionEqual, cmp, reflect.Int)
			em.fb.emitGoto(endIf)
			em.emitExprR(expr.Expr2, boolType, tmp)
			em.fb.setLabelAddr(endIf)
			em.changeRegister(false, tmp, reg, boolType, dstType)
			em.fb.exitStack()
			return reg, false
		}
		// ==, !=, <, <=, >=, >, &&, ||, +, -, *, /, %, ^, &^, <<, >>.
		exprType := em.ti(expr).Type
		t1 := em.ti(expr.Expr1).Type
		t2 := em.ti(expr.Expr2).Type
		v1 := em.emitExpr(expr.Expr1, t1)
		v2, k := em.emitExprK(expr.Expr2, t2)
		if reg == 0 {
			return reg, false
		}
		// String concatenation.
		if expr.Operator() == ast.OperatorAddition && t1.Kind() == reflect.String {
			if k {
				v2 = em.emitExpr(expr.Expr2, t2)
			}
			if sameRegType(exprType.Kind(), dstType.Kind()) {
				em.fb.emitConcat(v1, v2, reg)
				return reg, false
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(exprType.Kind())
			em.fb.emitConcat(v1, v2, tmp)
			em.changeRegister(false, tmp, reg, exprType, dstType)
			em.fb.exitStack()
			return reg, false
		}
		switch expr.Operator() {
		case ast.OperatorAddition, ast.OperatorSubtraction, ast.OperatorMultiplication, ast.OperatorDivision,
			ast.OperatorModulo, ast.OperatorAnd, ast.OperatorOr, ast.OperatorXor, ast.OperatorAndNot,
			ast.OperatorLeftShift, ast.OperatorRightShift:
			emitFn := map[ast.OperatorType]func(bool, int8, int8, int8, reflect.Kind){
				ast.OperatorAddition:       em.fb.emitAdd,
				ast.OperatorSubtraction:    em.fb.emitSub,
				ast.OperatorMultiplication: em.fb.emitMul,
				ast.OperatorDivision:       em.fb.emitDiv,
				ast.OperatorModulo:         em.fb.emitRem,
				ast.OperatorAnd:            em.fb.emitAnd,
				ast.OperatorOr:             em.fb.emitOr,
				ast.OperatorXor:            em.fb.emitXor,
				ast.OperatorAndNot:         em.fb.emitAndNot,
				ast.OperatorLeftShift:      em.fb.emitLeftShift,
				ast.OperatorRightShift:     em.fb.emitRightShift,
			}[expr.Operator()]
			if sameRegType(exprType.Kind(), dstType.Kind()) {
				emitFn(k, v1, v2, reg, exprType.Kind())
				return reg, false
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(exprType.Kind())
			emitFn(k, v1, v2, tmp, exprType.Kind())
			em.changeRegister(false, tmp, reg, exprType, dstType)
			em.fb.exitStack()
			return reg, false
		case ast.OperatorEqual, ast.OperatorNotEqual, ast.OperatorLess, ast.OperatorLessOrEqual,
			ast.OperatorGreaterOrEqual, ast.OperatorGreater:
			cond := map[ast.OperatorType]vm.Condition{
				ast.OperatorEqual:          vm.ConditionEqual,
				ast.OperatorNotEqual:       vm.ConditionNotEqual,
				ast.OperatorLess:           vm.ConditionLess,
				ast.OperatorLessOrEqual:    vm.ConditionLessOrEqual,
				ast.OperatorGreater:        vm.ConditionGreater,
				ast.OperatorGreaterOrEqual: vm.ConditionGreaterOrEqual,
			}[expr.Operator()]
			if sameRegType(exprType.Kind(), dstType.Kind()) {
				em.fb.emitMove(true, 1, reg, reflect.Bool)
				em.fb.emitIf(k, v1, cond, v2, t1.Kind())
				em.fb.emitMove(true, 0, reg, reflect.Bool)
				return reg, false
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(exprType.Kind())
			em.fb.emitMove(true, 1, tmp, reflect.Bool)
			em.fb.emitIf(k, v1, cond, v2, t1.Kind())
			em.fb.emitMove(true, 0, tmp, reflect.Bool)
			em.changeRegister(false, tmp, reg, exprType, dstType)
			em.fb.exitStack()
		}

	case *ast.Call:

		// ShowMacro which must be ignored (cannot be resolved).
		if em.ti(expr.Func) == showMacroIgnoredTi {
			return reg, false
		}

		// Builtin call.
		if em.ti(expr.Func).IsBuiltin() {
			em.emitBuiltin(expr, reg, dstType)
			return reg, false
		}

		// Conversion.
		if em.ti(expr.Func).IsType() {
			convertType := em.ti(expr.Func).Type
			// A conversion cannot have side-effects.
			if reg == 0 {
				return reg, false
			}
			typ := em.ti(expr.Args[0]).Type
			arg := em.emitExpr(expr.Args[0], typ)
			if sameRegType(convertType.Kind(), dstType.Kind()) {
				em.changeRegister(false, arg, reg, typ, convertType)
				return reg, false
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(convertType.Kind())
			em.changeRegister(false, arg, tmp, typ, convertType)
			em.changeRegister(false, tmp, reg, convertType, dstType)
			em.fb.exitStack()
			return reg, false
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
					em.emitExprR(kv.Value, typ, 0)
				}
				return reg, false
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
				elem, k := em.emitExprK(kv.Value, typ.Elem())
				if reg != 0 {
					em.fb.emitSetSlice(k, reg, elem, indexReg)
				}
				em.fb.exitStack()
			}
		case reflect.Struct:
			if reg == 0 {
				for _, kv := range expr.KeyValues {
					typ := em.ti(kv.Value).Type
					em.emitExprR(kv.Value, typ, 0)
				}
				return reg, false
			}
			em.fb.enterStack()
			tmpTyp := reflect.PtrTo(typ)
			tmp := -em.fb.newRegister(tmpTyp.Kind())
			em.fb.emitNew(typ, -tmp)
			if len(expr.KeyValues) > 0 {
				for _, kv := range expr.KeyValues {
					name := kv.Key.(*ast.Identifier).Name
					field, _ := typ.FieldByName(name)
					valueType := em.ti(kv.Value).Type
					var value int8
					if sameRegType(field.Type.Kind(), valueType.Kind()) {
						value = em.emitExpr(kv.Value, valueType)
					} else {
						panic("TODO: not implemented") // TODO(Gianluca): to implement.
					}
					// TODO(Gianluca): use field "k" of SetField.
					fieldConstIndex := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
					em.fb.emitSetField(false, tmp, fieldConstIndex, value)
				}
			}
			em.changeRegister(false, tmp, reg, tmpTyp, dstType)
			em.fb.exitStack()

		case reflect.Map:
			if reg == 0 {
				for _, kv := range expr.KeyValues {
					typ := em.ti(kv.Value).Type
					em.emitExprR(kv.Value, typ, 0)
				}
				return reg, false
			}
			size := len(expr.KeyValues)
			if 0 <= size && size < 126 {
				em.fb.emitMakeMap(typ, true, int8(size), reg)
			} else {
				sizeReg := em.fb.makeIntConstant(int64(size))
				em.fb.emitMakeMap(typ, false, sizeReg, reg)
			}
			for _, kv := range expr.KeyValues {
				key := em.fb.newRegister(typ.Key().Kind())
				em.fb.enterStack()
				em.emitExprR(kv.Key, typ.Key(), key)
				value, k := em.emitExprK(kv.Value, typ.Elem())
				em.fb.exitStack()
				em.fb.emitSetMap(k, reg, value, key, typ)
			}
		}

	case *ast.TypeAssertion:

		exprType := em.ti(expr.Expr).Type
		exprReg := em.emitExpr(expr.Expr, exprType)
		assertType := em.ti(expr.Type).Type
		if sameRegType(assertType.Kind(), dstType.Kind()) {
			em.fb.emitAssert(exprReg, assertType, reg)
			em.fb.emitNop()
			return reg, false
		}
		em.fb.enterScope()
		tmp := em.fb.newRegister(assertType.Kind())
		em.fb.emitAssert(exprReg, assertType, tmp)
		em.fb.emitNop()
		em.changeRegister(false, tmp, reg, assertType, dstType)
		em.fb.exitScope()

	case *ast.Selector:

		em.emitSelector(expr, reg, dstType)

	case *ast.UnaryOperator:

		// Unary operation (negation) on a complex number.
		if exprType := em.ti(expr).Type; exprType.Kind() == reflect.Complex64 || exprType.Kind() == reflect.Complex128 {
			if expr.Operator() != ast.OperatorSubtraction {
				panic("bug: expected operator subtraction")
			}
			stackShift := em.fb.currentStackShift()
			em.fb.enterScope()
			index := em.fb.complexOperationIndex(ast.OperatorSubtraction, true)
			ret := em.fb.newRegister(reflect.Complex128)
			arg := em.fb.newRegister(reflect.Complex128)
			em.fb.enterScope()
			em.emitExprR(expr.Expr, exprType, arg)
			em.fb.exitScope()
			em.fb.emitCallPredefined(index, 0, stackShift)
			em.changeRegister(false, ret, reg, exprType, dstType)
			em.fb.exitScope()
			return reg, false
		}

		exprType := em.ti(expr.Expr).Type
		typ := em.ti(expr).Type
		switch expr.Operator() {
		case ast.OperatorNot:
			if reg == 0 {
				em.emitExprR(expr.Expr, exprType, 0)
				return reg, false
			}
			if sameRegType(typ.Kind(), dstType.Kind()) {
				em.emitExprR(expr.Expr, exprType, reg)
				em.fb.emitSubInv(true, reg, int8(1), reg, reflect.Int)
				return reg, false
			}
			em.fb.enterScope()
			exprReg := em.emitExpr(expr.Expr, exprType)
			em.fb.emitSubInv(true, exprReg, int8(1), exprReg, reflect.Int)
			em.changeRegister(false, exprReg, reg, exprType, dstType)
			em.fb.exitScope()
		case ast.OperatorMultiplication:
			if reg == 0 {
				em.emitExprR(expr.Expr, exprType, 0)
				return reg, false
			}
			if sameRegType(typ.Kind(), dstType.Kind()) {
				exprReg := em.emitExpr(expr.Expr, exprType)
				em.changeRegister(false, -exprReg, reg, exprType.Elem(), dstType)
				return reg, false
			}
			exprReg := em.emitExpr(expr.Expr, exprType)
			tmp := em.fb.newRegister(exprType.Elem().Kind())
			em.changeRegister(false, -exprReg, tmp, exprType.Elem(), exprType.Elem())
			em.changeRegister(false, tmp, reg, exprType.Elem(), dstType)
		case ast.OperatorAnd:
			switch expr := expr.Expr.(type) {
			case *ast.Identifier:
				if em.fb.isVariable(expr.Name) {
					varr := em.fb.scopeLookup(expr.Name)
					em.fb.emitNew(reflect.PtrTo(typ), reg)
					em.fb.emitMove(false, -varr, reg, dstType.Kind())
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
			if reg == 0 {
				em.emitExprR(expr.Expr, dstType, 0)
				return reg, false
			}
			exprReg := em.emitExpr(expr.Expr, dstType)
			if sameRegType(exprType.Kind(), dstType.Kind()) {
				em.fb.emitSubInv(true, exprReg, 0, reg, dstType.Kind())
				return reg, false
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(exprType.Kind())
			em.fb.emitSubInv(true, exprReg, 0, tmp, exprType.Kind())
			em.changeRegister(false, tmp, reg, exprType, dstType)
			em.fb.exitStack()

		case ast.OperatorReceive:
			if sameRegType(typ.Kind(), dstType.Kind()) {
				chann := em.emitExpr(expr.Expr, exprType)
				em.fb.emitReceive(chann, 0, reg)
				return reg, false
			}
			panic("TODO: not implemented") // TODO(Gianluca): to implement.
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
			em.emitNodes(expr.Body.Nodes)
			em.fb.end()
			em.fb.exitScope()
			em.fb = fb
			return reg, false
		}

		// Script function definition.
		if expr.Ident != nil && !em.isTemplate {
			varr := em.fb.newRegister(reflect.Func)
			em.fb.bindVarReg(expr.Ident.Name, varr)
			ident := expr.Ident
			expr.Ident = nil // avoids recursive calls.
			funcType := em.ti(expr).Type
			if em.isTemplate {
				addr := em.newAddress(addressRegister, funcType, varr, 0)
				em.assign([]address{addr}, []ast.Expression{expr})
			}
			expr.Ident = ident
			return reg, false
		}

		if reg == 0 {
			return reg, false
		}

		fn := em.fb.emitFunc(reg, em.ti(expr).Type)
		em.setClosureRefs(fn, expr.Upvars)

		funcLitBuilder := newBuilder(fn)
		funcLitBuilder.emitSetAlloc(em.options.MemoryLimit)
		currFB := em.fb
		em.fb = funcLitBuilder

		em.fb.enterScope()
		em.prepareFunctionBodyParameters(expr)
		em.emitNodes(expr.Body.Nodes)
		em.fb.exitScope()
		em.fb.end()
		em.fb = currFB

	case *ast.Identifier:

		// An identifier evaluation cannot have side effects.
		if reg == 0 {
			return reg, false
		}

		typ := em.ti(expr).Type

		if em.fb.isVariable(expr.Name) {
			ident := em.fb.scopeLookup(expr.Name)
			em.changeRegister(false, ident, reg, typ, dstType)
			return reg, false
		}

		// Identifier represents a function.
		if fun, ok := em.functions[em.pkg][expr.Name]; ok {
			em.fb.emitGetFunc(false, em.functionIndex(fun), reg)
			return reg, false
		}

		// Clojure variable.
		if index, ok := em.closureVarRefs[em.fb.fn][expr.Name]; ok {
			if sameRegType(typ.Kind(), dstType.Kind()) {
				em.fb.emitGetVar(index, reg)
				return reg, false
			}
			tmp := em.fb.newRegister(typ.Kind())
			em.fb.emitGetVar(index, tmp)
			em.changeRegister(false, tmp, reg, typ, dstType)
			return reg, false
		}

		// Scriggo variable.
		if index, ok := em.varIndexes[em.pkg][expr.Name]; ok {
			if sameRegType(typ.Kind(), dstType.Kind()) {
				em.fb.emitGetVar(int(index), reg)
				return reg, false
			}
			tmp := em.fb.newRegister(typ.Kind())
			em.fb.emitGetVar(int(index), tmp)
			em.changeRegister(false, tmp, reg, typ, dstType)
			return reg, false
		}

		// Predefined variable.
		if ti := em.ti(expr); ti.IsPredefined() && ti.Type.Kind() != reflect.Func {
			index := em.predVarIndex(ti.value.(reflect.Value), ti.PredefPackageName, expr.Name)
			if sameRegType(ti.Type.Kind(), dstType.Kind()) {
				em.fb.emitGetVar(int(index), reg)
				return reg, false
			}
			tmp := em.fb.newRegister(ti.Type.Kind())
			em.fb.emitGetVar(int(index), tmp)
			em.changeRegister(false, tmp, reg, ti.Type, dstType)
			return reg, false
		}

		panic(fmt.Errorf("bug: none of the previous conditions matched identifier %v", expr))

	case *ast.Index:

		exprType := em.ti(expr.Expr).Type
		exprReg := em.emitExpr(expr.Expr, exprType)
		var indexType reflect.Type
		if exprType.Kind() == reflect.Map {
			indexType = exprType.Key()
		} else {
			indexType = intType
		}
		index, kindex := em.emitExprK(expr.Index, indexType)
		var elemType reflect.Type
		if exprType.Kind() == reflect.String {
			elemType = uint8Type
		} else {
			elemType = exprType.Elem()
		}
		if sameRegType(elemType.Kind(), dstType.Kind()) {
			em.fb.emitIndex(kindex, exprReg, index, reg, exprType)
			return reg, false
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(elemType.Kind())
		em.fb.emitIndex(kindex, exprReg, index, tmp, exprType)
		em.changeRegister(false, tmp, reg, elemType, dstType)
		em.fb.exitStack()

	case *ast.Slicing:

		src := em.emitExpr(expr.Expr, em.ti(expr.Expr).Type)
		var low, high, max int8 = 0, -1, -1
		var kLow, kHigh, kMax = true, true, true
		// emit low
		if expr.Low != nil {
			typ := em.ti(expr.Low).Type
			low, kLow = em.emitExprK(expr.Low, typ)
		}
		// emit high
		if expr.High != nil {
			typ := em.ti(expr.High).Type
			high, kHigh = em.emitExprK(expr.High, typ)
		}
		// emit max
		if expr.Max != nil {
			typ := em.ti(expr.Max).Type
			max, kMax = em.emitExprK(expr.Max, typ)
		}
		em.fb.emitSlice(kLow, kHigh, kMax, src, reg, low, high, max)

	default:

		panic(fmt.Sprintf("emitExpr currently does not support %T nodes (expr: %s)", expr, expr))

	}

	return reg, false
}

// emitTypeSwitch emits instructions for a type switch node.
func (em *emitter) emitTypeSwitch(node *ast.TypeSwitch) {
	// TODO (Gianluca): a type-checker bug does not replace type switch type with proper value.

	em.fb.enterScope()

	if node.Init != nil {
		em.emitNodes([]ast.Node{node.Init})
	}

	typeAssertion := node.Assignment.Rhs[0].(*ast.TypeAssertion)
	expr := em.emitExpr(typeAssertion.Expr, em.ti(typeAssertion.Expr).Type)

	if len(node.Assignment.Lhs) == 1 {
		n := ast.NewAssignment(
			node.Assignment.Pos(),
			[]ast.Expression{node.Assignment.Lhs[0]},
			node.Assignment.Type,
			[]ast.Expression{typeAssertion.Expr},
		)
		em.emitNodes([]ast.Node{n})
	}

	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := em.fb.newLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = em.fb.newLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			if em.ti(caseExpr).Nil() {
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
		em.emitNodes(cas.Body)
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
		em.emitNodes([]ast.Node{node.Init})
	}

	var expr int8
	var typ reflect.Type

	if node.Expr == nil {
		typ = reflect.TypeOf(false)
		expr = em.fb.newRegister(typ.Kind())
		em.fb.emitMove(true, 1, expr, typ.Kind())
	} else {
		typ = em.ti(node.Expr).Type
		expr = em.emitExpr(node.Expr, typ)
	}

	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := em.fb.newLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = em.fb.newLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			y, ky := em.emitExprK(caseExpr, typ)
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
		em.emitNodes(cas.Body)
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
		v1 := em.emitExpr(cond, ti.Type)
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
		if em.ti(cond.Expr1).Nil() != em.ti(cond.Expr2).Nil() {
			expr := cond.Expr1
			if em.ti(cond.Expr1).Nil() {
				expr = cond.Expr2
			}
			typ := em.ti(expr).Type
			v := em.emitExpr(expr, typ)
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
				v1 := em.emitExpr(lenArg, em.ti(lenArg).Type)
				typ := em.ti(expr).Type
				v2, k2 := em.emitExprK(expr, typ)
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
				v1 := em.emitExpr(cond.Expr1, t1)
				v2, k2 := em.emitExprK(cond.Expr2, t2)
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

		v1 := em.emitExpr(cond, em.ti(cond).Type)
		k2 := em.fb.makeIntConstant(1)
		v2 := em.fb.newRegister(reflect.Bool)
		em.fb.emitLoadNumber(vm.TypeInt, k2, v2)
		em.fb.emitIf(false, v1, vm.ConditionEqual, v2, reflect.Bool)

	}

}
