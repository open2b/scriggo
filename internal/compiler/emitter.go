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
	"scriggo/internal/compiler/types"
	"scriggo/runtime"
)

// An emitter emits instructions for the VM.
type emitter struct {
	fnStore *functionStore

	// Index in the Function VarRefs field for each predefined variable.
	// TODO(Gianluca): this is the new way of accessing predefined vars.
	// Incrementally integrate into Scriggo, then remove the other (unused)
	// fields.
	// predefinedVarRefs map[*runtime.Function]map[*reflect.Value]int

	// fb is the current function builder.
	fb *functionBuilder

	varStore *varStore

	labels map[*runtime.Function]map[string]label

	// pkg is the package that is currently being emitted.
	pkg *ast.Package

	// typeInfos maps nodes to their type info.
	// Should be accessed using method 'ti'.
	typeInfos map[ast.Node]*TypeInfo

	// Index in the Function VarRefs field for each closure variable.
	closureVars map[*runtime.Function]map[string]int

	options EmitterOptions

	// isTemplate reports whether the emitter is currently emitting a template.
	isTemplate bool

	// rangeLabels is a list of current active Ranges. First element is the
	// Range address, second refers to the first instruction outside Range's
	// body.
	rangeLabels [][2]label

	// breakable is true if emitting a "breakable" statement (except ForRange,
	// which implements his own "breaking" system).
	breakable bool

	// breakLabel, if not nil, is the label to which pre-stated "breaks" must
	// jump.
	breakLabel *label

	// inURL indicates if the emitter is currently inside an *ast.URL node.
	inURL bool

	// types refers the types of the current compilation and it is used to
	// create and manipulate types and values, both predefined and defined only
	// by Scriggo.
	types *types.Types
}

// newEmitter returns a new emitter with the given type infos, indirect
// variables and options.
func newEmitter(typeInfos map[ast.Node]*TypeInfo, indirectVars map[*ast.Identifier]bool, opts EmitterOptions) *emitter {
	em := &emitter{
		labels:      make(map[*runtime.Function]map[string]label),
		options:     opts,
		typeInfos:   typeInfos,
		closureVars: map[*runtime.Function]map[string]int{},
		types:       types.NewTypes(), // TODO: this is wrong: the instance should be taken from the type checker.
	}
	em.fnStore = newFunctionStore(em)
	em.varStore = newVarStore(em, indirectVars)
	return em
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

// reserveTemplateRegisters reverses the register used for implement
// specific template functions.
func (em *emitter) reserveTemplateRegisters() {
	// Sync with:
	//
	// - case *ast.Show of emitter.emitNodes
	// - case *ast.Text of emitter.emitNodes
	// - EmitTemplate
	// - emitter.setClosureRefs
	//
	em.fb.templateRegs.gA = em.fb.newRegister(reflect.Interface) // w io.Writer
	em.fb.templateRegs.gB = em.fb.newRegister(reflect.Interface) // Write
	em.fb.templateRegs.gC = em.fb.newRegister(reflect.Interface) // Render
	em.fb.templateRegs.gD = em.fb.newRegister(reflect.Interface) // free.
	em.fb.templateRegs.gE = em.fb.newRegister(reflect.Interface) // free.
	em.fb.templateRegs.gF = em.fb.newRegister(reflect.Interface) // urlWriter
	em.fb.templateRegs.iA = em.fb.newRegister(reflect.Int)       // free.
	em.fb.emitGetVar(0, em.fb.templateRegs.gA, reflect.Interface)
	em.fb.emitGetVar(1, em.fb.templateRegs.gB, reflect.Interface)
	em.fb.emitGetVar(2, em.fb.templateRegs.gC, reflect.Interface)
	em.fb.emitGetVar(3, em.fb.templateRegs.gF, reflect.Interface)
}

// emitPackage emits a package and returns the exported functions, the
// exported variables and the init functions.
// extendingPage reports whether emitPackage is going to emit a package that
// extends another page.
func (em *emitter) emitPackage(pkg *ast.Package, extendingPage bool, path string) (map[string]*runtime.Function, map[string]int16, []*runtime.Function) {

	if !extendingPage {
		em.pkg = pkg
	}

	// https://github.com/open2b/scriggo/issues/476
	inits := []*runtime.Function{} // List of all "init" functions in current package.

	// Emit the imports.
	for _, decl := range pkg.Declarations {
		if node, ok := decl.(*ast.Import); ok {
			// If importing a predefined package, the emitter doesn't have to do
			// anything. Predefined variables, constants, types and functions
			// are added as information to the tree by the type-checker.
			if node.Tree != nil {
				backupPkg := em.pkg
				pkg := node.Tree.Nodes[0].(*ast.Package)
				funcs, vars, pkgInits := em.emitPackage(pkg, false, node.Tree.Path)
				em.pkg = backupPkg
				inits = append(inits, pkgInits...)
				var importName string
				if node.Ident == nil {
					importName = pkg.Name
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
						em.fnStore.declareScriggoFn(em.pkg, name, fn)
					} else {
						em.fnStore.declareScriggoFn(em.pkg, importName+"."+name, fn)
					}
				}
				for name, v := range vars {
					if importName == "" {
						em.varStore.addScriggoPackageVar(em.pkg, name, v)
					} else {
						em.varStore.addScriggoPackageVar(em.pkg, importName+"."+name, v)
					}
				}
			}
		}
	}

	// Package level functions.
	functions := map[string]*runtime.Function{}

	// initToBuild is the index of the next "init" function to build.
	initToBuild := len(inits)

	if extendingPage {
		// The function declarations have already been added to the list of
		// available functions, so they can't be added twice.
	} else {
		// Store all function declarations in current package before building
		// their bodies: order of declaration doesn't matter at package level.
		for _, dec := range pkg.Declarations {
			if fun, ok := dec.(*ast.Func); ok {
				fn := newFunction("main", fun.Ident.Name, fun.Type.Reflect)
				if fun.Ident.Name == "init" {
					inits = append(inits, fn)
					continue
				}
				em.fnStore.declareScriggoFn(em.pkg, fun.Ident.Name, fn)
				if isExported(fun.Ident.Name) {
					functions[fun.Ident.Name] = fn
				}
			}
		}
	}

	// Package level variables.
	vars := map[string]int16{}

	// Emit the package variables.
	var initVarsFn *runtime.Function
	var initVarsFb *functionBuilder
	for _, dec := range pkg.Declarations {
		if n, ok := dec.(*ast.Var); ok {
			// If the package has some variable declarations, a special "init"
			// function must be created to initialize them. "$initvars" is
			// used because is not a valid Go identifier, so there's no risk
			// of collision with Scriggo defined functions.
			backupFb := em.fb
			if initVarsFn == nil {
				initVarsFn = newFunction("main", "$initvars", reflect.FuncOf(nil, nil, false))
				em.fnStore.declareScriggoFn(em.pkg, "$initvars", initVarsFn)
				initVarsFb = newBuilder(initVarsFn, path)
				initVarsFb.emitSetAlloc(em.options.MemoryLimit)
				initVarsFb.enterScope()
			}
			em.fb = initVarsFb
			addresses := make([]address, len(n.Lhs))
			pkgVarRegs := map[string]int8{}
			pkgVarTypes := map[string]reflect.Type{}
			for i, v := range n.Lhs {
				if isBlankIdentifier(v) {
					addresses[i] = em.addressBlankIdent(v.Pos())
					continue
				}
				varType := em.ti(v).Type
				varr := em.fb.newRegister(varType.Kind())
				em.fb.bindVarReg(v.Name, varr)
				addresses[i] = em.addressLocalVar(varr, varType, v.Pos(), 0)
				// Store the variable register. It will be used later to store
				// initialized value inside the proper global index.
				pkgVarRegs[v.Name] = varr
				pkgVarTypes[v.Name] = varType
				index := em.varStore.addGlobal(newGlobal("main", v.Name, varType, nil))
				em.varStore.addScriggoPackageVar(em.pkg, v.Name, index)
				vars[v.Name] = index
			}
			em.assignValuesToAddresses(addresses, n.Rhs)
			for name, reg := range pkgVarRegs {
				index, _ := em.varStore.emitter.varStore.scriggoPackageVarIndex(em.pkg, name)
				em.fb.emitSetVar(false, reg, int(index), pkgVarTypes[name].Kind())
			}
			em.fb = backupFb
		}
	}

	// Emit function declarations.
	for _, dec := range pkg.Declarations {
		if n, ok := dec.(*ast.Func); ok {
			var fn *runtime.Function
			if isBlankIdentifier(n.Ident) {
				// Do not emit this function declaration; it has already been
				// type checked, so there's no need to enter into its body
				// again.
				continue
			}
			if n.Ident.Name == "init" {
				fn = inits[initToBuild]
				initToBuild++
			} else {
				fn = em.fnStore.getScriggoFn(em.pkg, n.Ident.Name)
			}
			em.fb = newBuilder(fn, path)
			em.fb.emitSetAlloc(em.options.MemoryLimit)
			em.fb.enterScope()
			// If this is the main function, functions that initialize variables
			// must be called before executing every other statement of the main
			// function.
			if n.Ident.Name == "main" {
				// First: initialize the package variables.
				if initVarsFn != nil {
					iv := em.fnStore.getScriggoFn(em.pkg, "$initvars")
					index := em.fb.addFunction(iv)
					em.fb.emitCall(int8(index), runtime.StackShift{}, nil)
				}
				// Second: call all init functions, in order.
				for _, initFunc := range inits {
					index := em.fb.addFunction(initFunc)
					em.fb.emitCall(int8(index), runtime.StackShift{}, nil)
				}
			}
			em.prepareFunctionBodyParameters(n)
			em.emitNodes(n.Body.Nodes)
			em.fb.end()
			em.fb.exitScope()
		}
	}

	if initVarsFn != nil {
		initVarsFb.exitScope()
		initVarsFb.emitReturn()
		initVarsFb.end()
	}

	// If this package is imported, initFuncs must contain initVarsFn, that is
	// processed as a generic "init" function.
	if initVarsFn != nil {
		inits = append(inits, initVarsFn)
	}

	return functions, vars, inits

}

// callOptions holds information about a function call.
type callOptions struct {
	predefined    bool
	receiverAsArg bool
	callHasDots   bool
}

// prepareCallParameters prepares the input and the output parameters for a
// function call.

// Returns the index (and their respective type) of the registers that will hold
// the function return parameters.
//
// Note that while prepareCallParameters is called before calling the function,
// prepareFunctionBodyParameters is called before emitting its body.
func (em *emitter) prepareCallParameters(fnTyp reflect.Type, args []ast.Expression, opts callOptions) ([]int8, []reflect.Type) {
	numOut := fnTyp.NumOut()
	numIn := fnTyp.NumIn()
	regs := make([]int8, numOut)
	types := make([]reflect.Type, numOut)
	for i := 0; i < numOut; i++ {
		t := fnTyp.Out(i)
		regs[i] = em.fb.newRegister(t.Kind())
		types[i] = t
	}
	if opts.receiverAsArg {
		reg := em.fb.newRegister(em.ti(args[0]).Type.Kind())
		em.fb.enterStack()
		em.emitExprR(args[0], em.ti(args[0]).Type, reg)
		em.fb.exitStack()
		args = args[1:]
	}
	if fnTyp.IsVariadic() {
		// f(g()) where f is variadic.
		if fnTyp.NumIn() == 1 && len(args) == 1 {
			if g, ok := args[0].(*ast.Call); ok {
				if numOut, ok := em.numOut(g); ok && numOut > 1 {
					if opts.predefined {
						argRegs, argTypes := em.emitCallNode(g, false, false)
						for i := range argRegs {
							dstType := fnTyp.In(0).Elem()
							reg := em.fb.newRegister(dstType.Kind())
							em.changeRegister(false, argRegs[i], reg, argTypes[i], dstType)
						}
						return regs, types
					}
					// f(g()) where g returns more than 1 argument, f is variadic and not predefined.
					slice := em.fb.newRegister(reflect.Slice)
					em.fb.enterStack()
					pos := args[0].Pos()
					em.fb.emitMakeSlice(true, true, fnTyp.In(numIn-1), int8(numOut), int8(numOut), slice, pos)
					argRegs, _ := em.emitCallNode(g, false, false)
					for i := range argRegs {
						index := em.fb.newRegister(reflect.Int)
						em.changeRegister(true, int8(i), index, intType, intType)
						em.fb.emitSetSlice(false, slice, argRegs[i], index, pos, fnTyp.In(numIn-1).Elem().Kind())
					}
					em.fb.exitStack()
					return []int8{slice}, []reflect.Type{fnTyp.In(numIn - 1)}
				}
			}
		}
		for i := 0; i < numIn-1; i++ {
			t := fnTyp.In(i)
			reg := em.fb.newRegister(t.Kind())
			em.fb.enterStack()
			em.emitExprR(args[i], t, reg)
			em.fb.exitStack()
		}
		if opts.callHasDots {
			sliceArg := args[len(args)-1]
			sliceArgType := fnTyp.In(fnTyp.NumIn() - 1)
			reg := em.fb.newRegister(sliceArgType.Kind())
			em.fb.enterStack()
			em.emitExprR(sliceArg, sliceArgType, reg)
			em.fb.exitStack()
			return regs, types
		}
		if varArgs := len(args) - (numIn - 1); varArgs == 0 {
			slice := em.fb.newRegister(reflect.Slice)
			em.fb.emitMakeSlice(true, true, fnTyp.In(numIn-1), 0, 0, slice, nil) // TODO: fix pos.
			return regs, types
		}
		if varArgs := len(args) - (numIn - 1); varArgs > 0 {
			t := fnTyp.In(numIn - 1).Elem()
			if opts.predefined {
				for i := 0; i < varArgs; i++ {
					reg := em.fb.newRegister(t.Kind())
					em.fb.enterStack()
					em.emitExprR(args[i+numIn-1], t, reg)
					em.fb.exitStack()
				}
			} else {
				slice := em.fb.newRegister(reflect.Slice)
				em.fb.emitMakeSlice(true, true, fnTyp.In(numIn-1), int8(varArgs), int8(varArgs), slice, nil) // TODO: fix pos.
				for i := 0; i < varArgs; i++ {
					tmp := em.fb.newRegister(t.Kind())
					em.fb.enterStack()
					em.emitExprR(args[i+numIn-1], t, tmp)
					em.fb.exitStack()
					index := em.fb.newRegister(reflect.Int)
					em.fb.emitMove(true, int8(i), index, reflect.Int, false)
					pos := args[len(args)-1].Pos()
					em.fb.emitSetSlice(false, slice, tmp, index, pos, fnTyp.In(numIn-1).Elem().Kind())
				}
			}
		}
	} else { // No-variadic function.
		if numIn > 1 && len(args) == 1 { // f(g()), where f takes more than 1 argument.
			regs, types := em.emitCallNode(args[0].(*ast.Call), false, false)
			for i := range regs {
				dstType := fnTyp.In(i)
				reg := em.fb.newRegister(dstType.Kind())
				em.changeRegister(false, regs[i], reg, types[i], dstType)
			}
		} else {
			for i := 0; i < numIn; i++ {
				t := fnTyp.In(i)
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

// emitCallNode emits instructions for a function call node. It returns the
// registers and the reflect types of the returned values.
// goStmt indicates if the call node belongs to a 'go statement', while
// deferStmt reports whether it must be deferred.
func (em *emitter) emitCallNode(call *ast.Call, goStmt bool, deferStmt bool) ([]int8, []reflect.Type) {

	funTi := em.ti(call.Func)

	// Method call on a interface value.
	if funTi.MethodType == MethodCallInterface {
		rcvrExpr := call.Func.(*ast.Selector).Expr
		rcvrType := em.ti(rcvrExpr).Type
		rcvr := em.emitExpr(rcvrExpr, rcvrType)
		// MethodValue reads receiver from general.
		if kindToType(rcvrType.Kind()) != generalRegister {
			// TODO(Gianluca): put rcvr in general
			panic("BUG: not implemented") // remove.
		}
		method := em.fb.newRegister(reflect.Func)
		name := call.Func.(*ast.Selector).Ident
		em.fb.emitMethodValue(name, rcvr, method)
		call.Args = append([]ast.Expression{rcvrExpr}, call.Args...)
		stackShift := em.fb.currentStackShift()
		opts := callOptions{
			predefined:    true,
			receiverAsArg: true,
			callHasDots:   call.IsVariadic,
		}
		regs, types := em.prepareCallParameters(funTi.Type, call.Args, opts)
		// TODO(Gianluca): handle variadic method calls.
		if goStmt {
			em.fb.emitGo()
		}
		if deferStmt {
			panic("BUG: not implemented") // remove.
		}
		em.fb.emitCallIndirect(method, 0, stackShift, call.Pos(), funTi.Type)
		return regs, types
	}

	// Predefined function (identifiers, selectors etc...).
	// Calls of predefined functions stored in builtin variables are handled as
	// common "indirect" calls.
	if funTi.IsPredefined() && !funTi.Addressable() {
		if funTi.MethodType == MethodCallConcrete {
			rcv := call.Func.(*ast.Selector).Expr // TODO(Gianluca): is this correct?
			call.Args = append([]ast.Expression{rcv}, call.Args...)
		}
		stackShift := em.fb.currentStackShift()
		opts := callOptions{
			predefined:    true,
			receiverAsArg: funTi.MethodType == MethodCallConcrete,
			callHasDots:   call.IsVariadic,
		}
		regs, types := em.prepareCallParameters(funTi.Type, call.Args, opts)
		var name string
		switch f := call.Func.(type) {
		case *ast.Identifier:
			name = f.Name
		case *ast.Selector:
			name = f.Ident
		}
		index := em.fnStore.predefFnIndex(funTi.value.(reflect.Value), funTi.PredefPackageName, name)
		if goStmt {
			em.fb.emitGo()
		}
		numVar := runtime.NoVariadicArgs
		if funTi.Type.IsVariadic() && !call.IsVariadic {
			numArgs := len(call.Args)
			if len(call.Args) == 1 {
				if callArg, ok := call.Args[0].(*ast.Call); ok {
					if numOut, ok := em.numOut(callArg); ok {
						numArgs = numOut
					}
				}
			}
			numVar = numArgs - (funTi.Type.NumIn() - 1)
		}
		if deferStmt {
			args := em.fb.currentStackShift()
			reg := em.fb.newRegister(reflect.Func)
			em.fb.emitLoadFunc(true, index, reg)
			em.fb.emitDefer(reg, int8(numVar), stackShift, args, funTi.Type)
			return regs, types
		}
		em.fb.emitCallPredefined(index, int8(numVar), stackShift, call.Pos())
		return regs, types
	}

	// Scriggo-defined function (identifier).
	if ident, ok := call.Func.(*ast.Identifier); ok && !em.fb.isVariable(ident.Name) {
		if em.fnStore.isScriggoFn(em.pkg, ident.Name) {
			fn := em.fnStore.getScriggoFn(em.pkg, ident.Name)
			stackShift := em.fb.currentStackShift()
			regs, types := em.prepareCallParameters(fn.Type, call.Args, callOptions{callHasDots: call.IsVariadic})
			index := em.fnStore.scriggoFnIndex(fn)
			if goStmt {
				em.fb.emitGo()
			}
			if deferStmt {
				args := stackDifference(em.fb.currentStackShift(), stackShift)
				reg := em.fb.newRegister(reflect.Func)
				em.fb.emitLoadFunc(false, index, reg)
				// TODO(Gianluca): review vm.NoVariadicArgs.
				em.fb.emitDefer(reg, runtime.NoVariadicArgs, stackShift, args, fn.Type)
				return regs, types
			}
			em.fb.emitCall(index, stackShift, call.Pos())
			return regs, types
		}
	}

	// Scriggo-defined function (selector).
	if selector, ok := call.Func.(*ast.Selector); ok {
		if ident, ok := selector.Expr.(*ast.Identifier); ok {
			if em.fnStore.isScriggoFn(em.pkg, ident.Name+"."+selector.Ident) {
				fun := em.fnStore.getScriggoFn(em.pkg, ident.Name+"."+selector.Ident)
				stackShift := em.fb.currentStackShift()
				regs, types := em.prepareCallParameters(fun.Type, call.Args, callOptions{callHasDots: call.IsVariadic})
				index := em.fnStore.scriggoFnIndex(fun)
				if goStmt {
					em.fb.emitGo()
				}
				if deferStmt {
					panic("BUG: not implemented") // remove.
				}
				em.fb.emitCall(index, stackShift, call.Pos())
				return regs, types
			}
		}
	}

	// Indirect function.
	reg := em.emitExpr(call.Func, em.ti(call.Func).Type)
	stackShift := em.fb.currentStackShift()
	opts := callOptions{predefined: false, callHasDots: call.IsVariadic}
	regs, types := em.prepareCallParameters(funTi.Type, call.Args, opts)
	// CallIndirect is always emitted with 'NoVariadicArgs' because the emitter
	// cannot distinguish between Scriggo defined functions (that require a
	// []Type) and predefined function (that require Type1, Type2 ...). For this
	// reason the arguments of an indirect call are emitted as if always calling
	// a Scriggo defined function.
	if goStmt {
		em.fb.emitGo()
	}
	if deferStmt {
		args := stackDifference(em.fb.currentStackShift(), stackShift)
		em.fb.emitDefer(reg, int8(runtime.NoVariadicArgs), stackShift, args, funTi.Type)
		return regs, types
	}
	em.fb.emitCallIndirect(reg, int8(runtime.NoVariadicArgs), stackShift, call.Pos(), funTi.Type)

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
		if kindToType(rcvrType.Kind()) != generalRegister {
			oldRcvr := rcvr
			rcvr = em.fb.newRegister(reflect.Interface)
			em.fb.emitTypify(false, rcvrType, oldRcvr, rcvr)
		}
		if kindToType(dstType.Kind()) == generalRegister {
			em.fb.emitMethodValue(expr.Ident, rcvr, reg)
		} else {
			panic("not implemented")
		}
		return
	}

	// Scriggo-defined package variables.
	if ident, ok := expr.Expr.(*ast.Identifier); ok {

		if index, ok := em.nonLocalVarIndex(expr); ok {
			if reg == 0 {
				return
			}
			if canEmitDirectly(ti.Type.Kind(), dstType.Kind()) {
				em.fb.emitGetVar(int(index), reg, dstType.Kind())
				return
			}
			tmp := em.fb.newRegister(ti.Type.Kind())
			em.fb.emitGetVar(int(index), tmp, ti.Type.Kind())
			em.changeRegister(false, tmp, reg, ti.Type, dstType)
			return
		}

		// Scriggo-defined package functions.
		if em.fnStore.isScriggoFn(em.pkg, ident.Name+"."+expr.Ident) {
			sf := em.fnStore.getScriggoFn(em.pkg, ident.Name+"."+expr.Ident)
			if reg == 0 {
				return
			}
			index := em.fnStore.scriggoFnIndex(sf)
			em.fb.emitLoadFunc(false, index, reg)
			em.changeRegister(false, reg, reg, em.ti(expr).Type, dstType)
			return
		}
	}

	// Struct field.
	exprType := em.ti(expr.Expr).Type
	exprReg := em.emitExpr(expr.Expr, exprType)
	field, _ := exprType.FieldByName(expr.Ident)
	index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
	fieldType := em.ti(expr).Type
	if canEmitDirectly(fieldType.Kind(), dstType.Kind()) {
		em.fb.emitField(exprReg, index, reg, dstType.Kind(), true)
		return
	}
	// TODO: add enter/exit stack method calls.
	tmp := em.fb.newRegister(fieldType.Kind())
	em.fb.emitField(exprReg, index, tmp, fieldType.Kind(), true)
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
			em.fb.emitMove(false, slice, tmp, sliceType.Kind(), false)
			arg := em.emitExpr(args[1], em.ti(args[1]).Type)
			em.fb.emitAppendSlice(arg, tmp, call.Pos())
			em.changeRegister(false, tmp, reg, sliceType, dstType)
			return
		}
		// TODO(Gianluca): moving to a different register is not always
		// necessary. For instance, in case of `s = append(s, t)` moving can be
		// avoided. The problem is that now is too late to check for left-hand
		// symbol which receives the return value of the appending.
		em.fb.enterStack()
		tmp := em.fb.newRegister(sliceType.Kind())
		em.changeRegister(false, slice, tmp, sliceType, sliceType)
		elems := []int8{}
		for _, argExpr := range args[1:] {
			elem := em.fb.newRegister(sliceType.Elem().Kind())
			em.fb.enterStack()
			em.emitExprR(argExpr, sliceType.Elem(), elem)
			em.fb.exitStack()
			elems = append(elems, elem)
		}
		// TODO(Gianluca): if len(appendArgs) > 255 split in blocks
		if len(elems) > 0 {
			em.fb.emitAppend(elems[0], elems[0]+int8(len(elems)), tmp, sliceType.Elem().Kind())
		}
		em.changeRegister(false, tmp, reg, sliceType, dstType)
		em.fb.exitStack()
	case "cap":
		typ := em.ti(args[0]).Type
		s := em.emitExpr(args[0], typ)
		if canEmitDirectly(intType.Kind(), dstType.Kind()) {
			em.fb.emitCap(s, reg)
			return
		}
		tmp := em.fb.newRegister(intType.Kind())
		em.fb.emitCap(s, tmp)
		em.changeRegister(false, tmp, reg, intType, dstType)
	case "close":
		chann := em.emitExpr(args[0], em.ti(args[0]).Type)
		em.fb.emitClose(chann, call.Pos())
	case "complex":
		floatType := em.ti(args[0]).Type
		r := em.emitExpr(args[0], floatType)
		i := em.emitExpr(args[1], floatType)
		complexType := complex128Type
		if floatType.Kind() == reflect.Float32 {
			complexType = complex64Type
		}
		if canEmitDirectly(complexType.Kind(), dstType.Kind()) {
			em.fb.emitComplex(r, i, reg, dstType.Kind())
			return
		}
		tmp := em.fb.newRegister(complexType.Kind())
		em.fb.emitComplex(r, i, tmp, complexType.Kind())
		em.changeRegister(false, tmp, reg, complexType, dstType)
	case "copy":
		dst := em.emitExpr(args[0], em.ti(args[0]).Type)
		src := em.emitExpr(args[1], em.ti(args[1]).Type)
		if reg == 0 {
			em.fb.emitCopy(dst, src, 0)
			return
		}
		if canEmitDirectly(reflect.Int, dstType.Kind()) {
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
	case "len":
		typ := em.ti(args[0]).Type
		s := em.emitExpr(args[0], typ)
		if canEmitDirectly(reflect.Int, dstType.Kind()) {
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
			em.fb.emitMakeSlice(kLen, kCap, typ, lenn, capp, reg, call.Pos())
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
			em.fb.emitMakeChan(chanType, kCapacity, capacity, reg, call.Pos())
		default:
			panic("bug")
		}
	case "new":
		newType := em.ti(args[0]).Type
		em.fb.emitNew(newType, reg)
	case "panic":
		arg := em.emitExpr(args[0], emptyInterfaceType)
		em.fb.emitPanic(arg, nil, call.Pos())
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
			if i < last {
				str := em.fb.makeStringConstant(" ")
				sep := em.fb.newRegister(reflect.Interface)
				em.changeRegister(true, str, sep, stringType, emptyInterfaceType)
				em.fb.emitPrint(sep)
			}
		}
		str := em.fb.makeStringConstant("\n")
		sep := em.fb.newRegister(reflect.Interface)
		em.changeRegister(true, str, sep, stringType, emptyInterfaceType)
		em.fb.emitPrint(sep)
	case "real", "imag":
		complexType := em.ti(args[0]).Type
		complex, k := em.emitExprK(args[0], complexType)
		floatType := float64Type
		if complexType.Kind() == reflect.Complex64 {
			floatType = float32Type
		}
		if canEmitDirectly(floatType.Kind(), dstType.Kind()) {
			if call.Func.(*ast.Identifier).Name == "real" {
				em.fb.emitRealImag(k, complex, reg, 0)
			} else {
				em.fb.emitRealImag(k, complex, 0, reg)
			}
			return
		}
		tmp := em.fb.newRegister(floatType.Kind())
		if call.Func.(*ast.Identifier).Name == "real" {
			em.fb.emitRealImag(k, complex, tmp, 0)
		} else {
			em.fb.emitRealImag(k, complex, 0, tmp)
		}
		em.changeRegister(false, tmp, reg, floatType, dstType)
	case "recover":
		em.fb.emitRecover(reg, false)
	default:
		panic("BUG: unknown builtin") // remove.
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

	ti := em.ti(expr)

	// No need to use the given register: check if expr can be emitted without
	// allocating a new one.
	if !useGivenReg {
		// Check if expr can be emitted as immediate.
		if allowK && ti.HasValue() && !ti.IsPredefined() {
			switch v := ti.value.(type) {
			case int64:
				if canEmitDirectly(reflect.Int, dstType.Kind()) {
					if -127 < v && v < 126 {
						return int8(v), true
					}
				}
			case float64:
				if canEmitDirectly(reflect.Float64, dstType.Kind()) {
					if math.Floor(v) == v && -127 < v && v < 126 {
						return int8(v), true
					}
				}
			}
		}
		// Expr cannot be emitted as immediate: check if it's possible to emit
		// it without allocating a new register.
		if expr, ok := expr.(*ast.Identifier); ok && em.fb.isVariable(expr.Name) {
			if canEmitDirectly(ti.Type.Kind(), dstType.Kind()) {
				return em.fb.scopeLookup(expr.Name), false
			}
		}

		// None of the conditions above applied: a new register must be
		// allocated, and the emission must proceed.
		reg = em.fb.newRegister(dstType.Kind())
	}

	// If the instructions that emit expr put result in a register type
	// different than the register type of dstType, use an intermediate
	// temporary register. Consider that this is not always necessary to check
	// this: for example if expr is a function, dstType must be a function or an
	// interface (this is guaranteed by the type checker) and in the current
	// implementation of the VM functions and interfaces use the same register
	// type.
	//
	// Not that this does not imply that method 'changeRegister' doesn't have to
	// be called: in case when internal representation is different than the
	// external one (functions and bools), the calls to 'changeRegister' may
	// emit a Typify instruction to ensure that values are correctly converted.

	if ti != nil && ti.HasValue() && !ti.IsPredefined() {
		typ := ti.Type
		if reg == 0 {
			return reg, false
		}
		// Handle nil values.
		if ti.value == nil {
			c := em.fb.makeGeneralConstant(nil)
			em.changeRegister(true, c, reg, typ, dstType)
			return reg, false
		}
		switch v := ti.value.(type) {
		case int64:
			c := em.fb.makeIntConstant(v)
			if canEmitDirectly(typ.Kind(), dstType.Kind()) {
				em.fb.emitLoadNumber(intRegister, c, reg)
				em.changeRegister(false, reg, reg, typ, dstType)
				return reg, false
			}
			tmp := em.fb.newRegister(typ.Kind())
			em.fb.emitLoadNumber(intRegister, c, tmp)
			em.changeRegister(false, tmp, reg, typ, dstType)
			return reg, false
		case float64:
			var c int8
			if typ.Kind() == reflect.Float32 {
				c = em.fb.makeFloatConstant(float64(float32(v)))
			} else {
				c = em.fb.makeFloatConstant(v)
			}
			if canEmitDirectly(typ.Kind(), dstType.Kind()) {
				em.fb.emitLoadNumber(floatRegister, c, reg)
				em.changeRegister(false, reg, reg, typ, dstType)
				return reg, false
			}
			tmp := em.fb.newRegister(typ.Kind())
			em.fb.emitLoadNumber(floatRegister, c, tmp)
			em.changeRegister(false, tmp, reg, typ, dstType)
			return reg, false
		case string:
			c := em.fb.makeStringConstant(v)
			em.changeRegister(true, c, reg, typ, dstType)
			return reg, false
		}
		v := reflect.ValueOf(ti.value)
		switch v.Kind() {
		case reflect.Interface:
			panic("BUG: not implemented") // remove.
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
			panic("BUG: not implemented") // remove.
		default:
			panic(fmt.Errorf("unsupported value type %T (expr: %s)", ti.value, expr))
		}
		return reg, false
	}

	// Predefined values.
	if ti != nil && ti.IsPredefined() && ti.MethodType == NoMethod {

		// Predefined functions.
		if ti.Type.Kind() == reflect.Func && !ti.Addressable() {
			name := ""
			switch expr := expr.(type) {
			case *ast.Identifier:
				name = expr.Name
			case *ast.Selector:
				name = expr.Ident
			}
			index := em.fnStore.predefFnIndex(ti.value.(reflect.Value), ti.PredefPackageName, name)
			em.fb.emitLoadFunc(true, index, reg)
			em.changeRegister(false, reg, reg, ti.Type, dstType)
			return reg, false
		}

		// Predefined variable.
		if ident, ok := expr.(*ast.Identifier); ok {
			index := em.varStore.predefVarIndex(ti.value.(*reflect.Value), ti.PredefPackageName, ident.Name)
			if canEmitDirectly(ti.Type.Kind(), dstType.Kind()) {
				em.fb.emitGetVar(int(index), reg, dstType.Kind())
				return reg, false
			}
			tmp := em.fb.newRegister(ti.Type.Kind())
			em.fb.emitGetVar(int(index), tmp, ti.Type.Kind())
			em.changeRegister(false, tmp, reg, ti.Type, dstType)
			return reg, false
		}

	}

	switch expr := expr.(type) {

	case *ast.BinaryOperator:

		// Binary operations on complex numbers.
		if exprType := ti.Type; exprType.Kind() == reflect.Complex64 || exprType.Kind() == reflect.Complex128 {
			em.emitComplexOperation(exprType, expr.Expr1, expr.Operator(), expr.Expr2, reg, dstType)
			return reg, false
		}

		// Binary && and ||.
		if op := expr.Operator(); op == ast.OperatorAndAnd || op == ast.OperatorOrOr {
			cmp := int8(0)
			if op == ast.OperatorAndAnd {
				cmp = 1
			}
			if canEmitDirectly(dstType.Kind(), reflect.Bool) {
				em.emitExprR(expr.Expr1, dstType, reg)
				endIf := em.fb.newLabel()
				em.fb.emitIf(true, reg, runtime.ConditionEqual, cmp, reflect.Int, expr.Pos())
				em.fb.emitGoto(endIf)
				em.emitExprR(expr.Expr2, dstType, reg)
				em.fb.setLabelAddr(endIf)
				return reg, false
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(reflect.Bool)
			em.emitExprR(expr.Expr1, boolType, tmp)
			endIf := em.fb.newLabel()
			em.fb.emitIf(true, tmp, runtime.ConditionEqual, cmp, reflect.Int, expr.Pos())
			em.fb.emitGoto(endIf)
			em.emitExprR(expr.Expr2, boolType, tmp)
			em.fb.setLabelAddr(endIf)
			em.changeRegister(false, tmp, reg, boolType, dstType)
			em.fb.exitStack()
			return reg, false
		}

		// Equality (or not-equality) checking with the predeclared identifier 'nil'.
		if em.ti(expr.Expr1).Nil() || em.ti(expr.Expr2).Nil() {
			em.changeRegister(true, 1, reg, boolType, dstType)
			em.emitCondition(expr)
			em.changeRegister(true, 0, reg, boolType, dstType)
			return reg, false
		}

		// ==, !=, <, <=, >=, >, &&, ||, +, -, *, /, %, ^, &^, <<, >>.
		exprType := ti.Type
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
			if canEmitDirectly(exprType.Kind(), dstType.Kind()) {
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
			var emitFn func(bool, int8, int8, int8, reflect.Kind)
			switch expr.Operator() {
			case ast.OperatorAddition:
				emitFn = em.fb.emitAdd
			case ast.OperatorSubtraction:
				emitFn = em.fb.emitSub
			case ast.OperatorMultiplication:
				emitFn = em.fb.emitMul
			case ast.OperatorDivision:
				emitFn = func(k bool, x, y, z int8, kind reflect.Kind) { em.fb.emitDiv(k, x, y, z, kind, expr.Pos()) }
			case ast.OperatorModulo:
				emitFn = func(k bool, x, y, z int8, kind reflect.Kind) { em.fb.emitRem(k, x, y, z, kind, expr.Pos()) }
			case ast.OperatorAnd:
				emitFn = em.fb.emitAnd
			case ast.OperatorOr:
				emitFn = em.fb.emitOr
			case ast.OperatorXor:
				emitFn = em.fb.emitXor
			case ast.OperatorAndNot:
				emitFn = em.fb.emitAndNot
			case ast.OperatorLeftShift:
				emitFn = em.fb.emitLeftShift
			case ast.OperatorRightShift:
				emitFn = em.fb.emitRightShift
			}
			if canEmitDirectly(exprType.Kind(), dstType.Kind()) {
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
			var cond runtime.Condition
			if kind := t1.Kind(); reflect.Uint <= kind && kind <= reflect.Uintptr {
				cond = map[ast.OperatorType]runtime.Condition{
					ast.OperatorEqual:          runtime.ConditionEqual,    // same as signed integers
					ast.OperatorNotEqual:       runtime.ConditionNotEqual, // same as signed integers
					ast.OperatorLess:           runtime.ConditionLessU,
					ast.OperatorLessOrEqual:    runtime.ConditionLessOrEqualU,
					ast.OperatorGreater:        runtime.ConditionGreaterU,
					ast.OperatorGreaterOrEqual: runtime.ConditionGreaterOrEqualU,
				}[expr.Operator()]
			} else {
				cond = map[ast.OperatorType]runtime.Condition{
					ast.OperatorEqual:          runtime.ConditionEqual,
					ast.OperatorNotEqual:       runtime.ConditionNotEqual,
					ast.OperatorLess:           runtime.ConditionLess,
					ast.OperatorLessOrEqual:    runtime.ConditionLessOrEqual,
					ast.OperatorGreater:        runtime.ConditionGreater,
					ast.OperatorGreaterOrEqual: runtime.ConditionGreaterOrEqual,
				}[expr.Operator()]
			}
			pos := expr.Pos()
			if canEmitDirectly(exprType.Kind(), dstType.Kind()) {
				em.fb.emitMove(true, 1, reg, reflect.Bool, false)
				em.fb.emitIf(k, v1, cond, v2, t1.Kind(), pos)
				em.fb.emitMove(true, 0, reg, reflect.Bool, false)
				return reg, false
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(exprType.Kind())
			em.fb.emitMove(true, 1, tmp, reflect.Bool, false)
			em.fb.emitIf(k, v1, cond, v2, t1.Kind(), pos)
			em.fb.emitMove(true, 0, tmp, reflect.Bool, false)
			em.changeRegister(false, tmp, reg, exprType, dstType)
			em.fb.exitStack()
		}

	case *ast.Call:

		// ShowMacro which must be ignored (cannot be resolved).
		if em.ti(expr.Func) == showMacroIgnoredTi {
			return reg, false
		}

		// Predeclared built-in function call.
		if em.ti(expr.Func).IsBuiltinFunction() {
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
			if canEmitDirectly(convertType.Kind(), dstType.Kind()) {
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
		regs, types := em.emitCallNode(expr, false, false)
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
			if typ.Kind() == reflect.Slice {
				em.fb.emitMakeSlice(true, true, typ, length, length, reg, expr.Pos())
			} else {
				arrayZero := em.fb.makeGeneralConstant(em.types.New(typ).Elem().Interface())
				em.changeRegister(true, arrayZero, reg, typ, typ)
			}
			var index int8 = -1
			for _, kv := range expr.KeyValues {
				if kv.Key != nil {
					index = int8(em.ti(kv.Key).Constant.int64())
				} else {
					index++
				}
				em.fb.enterStack()
				indexReg := em.fb.newRegister(reflect.Int)
				em.fb.emitMove(true, index, indexReg, reflect.Int, true)
				elem, k := em.emitExprK(kv.Value, typ.Elem())
				if reg != 0 {
					em.fb.emitSetSlice(k, reg, elem, indexReg, expr.Pos(), typ.Elem().Kind())
				}
				em.fb.exitStack()
			}
			em.changeRegister(false, reg, reg, em.ti(expr.Type).Type, dstType)
		case reflect.Struct:
			// Struct should no be created, but its values must be emitted.
			if reg == 0 {
				for _, kv := range expr.KeyValues {
					em.emitExprR(kv.Value, em.ti(kv.Value).Type, 0)
				}
				return reg, false
			}
			// TODO: the types instance should be the same of the type checker!
			structZero := em.fb.makeGeneralConstant(em.types.New(typ).Elem().Interface())
			// When there are no values in the composite literal, optimize the
			// creation of the struct.
			if len(expr.KeyValues) == 0 {
				em.changeRegister(true, structZero, reg, typ, dstType)
				return reg, false
			}
			// Assign key-value pairs to the struct fields.
			em.fb.enterStack()
			var structt int8
			if canEmitDirectly(typ.Kind(), dstType.Kind()) {
				structt = em.fb.newRegister(reflect.Struct)
			} else {
				structt = reg
			}
			em.changeRegister(true, structZero, structt, typ, typ)
			for _, kv := range expr.KeyValues {
				name := kv.Key.(*ast.Identifier).Name
				field, _ := typ.FieldByName(name)
				valueType := em.ti(kv.Value).Type
				if canEmitDirectly(field.Type.Kind(), valueType.Kind()) {
					value, k := em.emitExprK(kv.Value, valueType)
					index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
					em.fb.emitSetField(k, structt, index, value, field.Type.Kind())
				} else {
					em.fb.enterStack()
					tmp := em.emitExpr(kv.Value, valueType)
					value := em.fb.newRegister(field.Type.Kind())
					em.changeRegister(false, tmp, value, valueType, field.Type)
					em.fb.exitStack()
					index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
					em.fb.emitSetField(false, structt, index, value, field.Type.Kind())
				}
				// TODO(Gianluca): use field "k" of SetField.
			}
			em.changeRegister(false, structt, reg, typ, dstType)
			em.fb.exitStack()

		case reflect.Map:
			if reg == 0 {
				for _, kv := range expr.KeyValues {
					typ := em.ti(kv.Value).Type
					em.emitExprR(kv.Value, typ, 0)
				}
				return reg, false
			}
			tmp := em.fb.newRegister(reflect.Map)
			size := len(expr.KeyValues)
			if 0 <= size && size < 126 {
				em.fb.emitMakeMap(typ, true, int8(size), tmp)
			} else {
				sizeReg := em.fb.makeIntConstant(int64(size))
				em.fb.emitMakeMap(typ, false, sizeReg, tmp)
			}
			for _, kv := range expr.KeyValues {
				key := em.fb.newRegister(typ.Key().Kind())
				em.fb.enterStack()
				em.emitExprR(kv.Key, typ.Key(), key)
				value, k := em.emitExprK(kv.Value, typ.Elem())
				em.fb.exitStack()
				em.fb.emitSetMap(k, tmp, value, key, typ, expr.Pos())
			}
			em.changeRegister(false, tmp, reg, typ, dstType)
		}

	case *ast.TypeAssertion:

		exprType := em.ti(expr.Expr).Type
		exprReg := em.emitExpr(expr.Expr, exprType)
		assertType := em.ti(expr.Type).Type
		pos := expr.Pos()
		if canEmitDirectly(assertType.Kind(), dstType.Kind()) {
			em.fb.emitAssert(exprReg, assertType, reg)
			em.fb.emitPanic(0, exprType, pos)
			return reg, false
		}
		em.fb.enterScope()
		tmp := em.fb.newRegister(assertType.Kind())
		em.fb.emitAssert(exprReg, assertType, tmp)
		em.fb.emitPanic(0, exprType, pos)
		em.changeRegister(false, tmp, reg, assertType, dstType)
		em.fb.exitScope()

	case *ast.Selector:

		em.emitSelector(expr, reg, dstType)

	case *ast.UnaryOperator:

		// Receive operation on channel.
		//
		//	v     = <- ch
		//  v, ok = <- ch
		//          <- ch
		if expr.Operator() == ast.OperatorReceive {
			chanType := em.ti(expr.Expr).Type
			valueType := ti.Type
			chann := em.emitExpr(expr.Expr, chanType)
			if reg == 0 {
				em.fb.emitReceive(chann, 0, 0)
				return reg, false
			}
			if canEmitDirectly(valueType.Kind(), dstType.Kind()) {
				em.fb.emitReceive(chann, 0, reg)
				return reg, false
			}
			tmp := em.fb.newRegister(valueType.Kind())
			em.fb.emitReceive(chann, 0, tmp)
			em.changeRegister(false, tmp, reg, valueType, dstType)
			return reg, false
		}

		// Unary operation (negation) on a complex number.
		if exprType := ti.Type; exprType.Kind() == reflect.Complex64 || exprType.Kind() == reflect.Complex128 {
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
			em.fb.emitCallPredefined(index, 0, stackShift, expr.Pos())
			em.changeRegister(false, ret, reg, exprType, dstType)
			em.fb.exitScope()
			return reg, false
		}

		// Emit a generic unary operator.
		em.emitUnaryOperator(expr, reg, dstType)

		return reg, false

	case *ast.Func:

		// Template macro definition.
		if expr.Ident != nil && em.isTemplate {
			macroFn := newFunction("", expr.Ident.Name, expr.Type.Reflect)
			em.fnStore.declareScriggoFn(em.pkg, expr.Ident.Name, macroFn)
			fb := em.fb
			em.setClosureRefs(macroFn, expr.Upvars)
			em.fb = newBuilder(macroFn, em.fb.getPath())
			em.fb.emitSetAlloc(em.options.MemoryLimit)
			em.fb.enterScope()
			em.prepareFunctionBodyParameters(expr)
			em.emitNodes(expr.Body.Nodes)
			em.fb.end()
			em.fb.exitScope()
			em.fb = fb
			return reg, false
		}

		if reg == 0 {
			return reg, false
		}

		var tmp int8
		if canEmitDirectly(reflect.Func, dstType.Kind()) {
			tmp = reg
		} else {
			tmp = em.fb.newRegister(reflect.Func)
		}

		fn := em.fb.emitFunc(tmp, ti.Type)
		em.setClosureRefs(fn, expr.Upvars)

		funcLitBuilder := newBuilder(fn, em.fb.getPath())
		funcLitBuilder.emitSetAlloc(em.options.MemoryLimit)
		currFB := em.fb
		em.fb = funcLitBuilder

		em.fb.enterScope()
		em.prepareFunctionBodyParameters(expr)
		em.emitNodes(expr.Body.Nodes)
		em.fb.exitScope()
		em.fb.end()
		em.fb = currFB

		em.changeRegister(false, tmp, reg, ti.Type, dstType)

	case *ast.Identifier:

		// An identifier evaluation cannot have side effects.
		if reg == 0 {
			return reg, false
		}

		typ := ti.Type

		if em.fb.isVariable(expr.Name) {
			ident := em.fb.scopeLookup(expr.Name)
			em.changeRegister(false, ident, reg, typ, dstType)
			return reg, false
		}

		// Identifier represents a function.
		if em.fnStore.isScriggoFn(em.pkg, expr.Name) {
			fun := em.fnStore.getScriggoFn(em.pkg, expr.Name)
			em.fb.emitLoadFunc(false, em.fnStore.scriggoFnIndex(fun), reg)
			em.changeRegister(false, reg, reg, ti.Type, dstType)
			return reg, false
		}

		// Scriggo variables and closure variables.
		if index, ok := em.nonLocalVarIndex(expr); ok {
			if canEmitDirectly(typ.Kind(), dstType.Kind()) {
				em.fb.emitGetVar(index, reg, dstType.Kind())
				return reg, false
			}
			tmp := em.fb.newRegister(typ.Kind())
			em.fb.emitGetVar(index, tmp, typ.Kind())
			em.changeRegister(false, tmp, reg, typ, dstType)
			return reg, false
		}

		panic(fmt.Errorf("BUG: none of the previous conditions matched identifier %v", expr)) // remove.

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
		pos := expr.Pos()
		if canEmitDirectly(elemType.Kind(), dstType.Kind()) {
			em.fb.emitIndex(kindex, exprReg, index, reg, exprType, pos, true)
			return reg, false
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(elemType.Kind())
		em.fb.emitIndex(kindex, exprReg, index, tmp, exprType, pos, true)
		em.changeRegister(false, tmp, reg, elemType, dstType)
		em.fb.exitStack()

	case *ast.Slicing:

		exprType := em.ti(expr.Expr).Type
		src := em.emitExpr(expr.Expr, exprType)
		var low, high int8 = 0, -1
		var kLow, kHigh = true, true
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
		pos := expr.Pos()
		if exprType.Kind() == reflect.String {
			em.fb.emitStringSlice(kLow, kHigh, src, reg, low, high, pos)
		} else {
			// emit max
			var max int8 = -1
			var kMax = true
			if expr.Max != nil {
				typ := em.ti(expr.Max).Type
				max, kMax = em.emitExprK(expr.Max, typ)
			}
			em.fb.emitSlice(kLow, kHigh, kMax, src, reg, low, high, max, pos)
		}

	default:

		panic(fmt.Sprintf("emitExpr currently does not support %T nodes (expr: %s)", expr, expr))

	}

	return reg, false
}

// emitCondition emits the instructions for a condition. The last instruction
// emitted is always the "If" instruction.
func (em *emitter) emitCondition(cond ast.Expression) {

	// cond is a boolean constant. Given that the 'if' instruction requires a
	// binary operation as condition, any boolean constant expressions 'b' is
	// converted to 'b == true'.
	if ti := em.ti(cond); ti != nil && ti.HasValue() {
		if ti.Type.Kind() != reflect.Bool {
			panic("BUG: expected a boolean constant") // remove.
		}
		v1 := em.emitExpr(cond, ti.Type)
		k2 := em.fb.makeIntConstant(1) // true
		v2 := em.fb.newRegister(reflect.Bool)
		em.fb.emitLoadNumber(intRegister, k2, v2)
		em.fb.emitIf(false, v1, runtime.ConditionEqual, v2, reflect.Bool, cond.Pos()) // v1 == true
		return
	}

	// if v   == nil
	// if v   != nil
	// if nil == v
	// if nil != v
	if cond, ok := cond.(*ast.BinaryOperator); ok {
		if em.ti(cond.Expr1).Nil() != em.ti(cond.Expr2).Nil() {
			expr := cond.Expr1
			if em.ti(cond.Expr1).Nil() {
				expr = cond.Expr2
			}
			typ := em.ti(expr).Type
			v := em.emitExpr(expr, typ)
			condType := runtime.ConditionNotNil
			if cond.Operator() == ast.OperatorEqual {
				condType = runtime.ConditionNil
			}
			if em.ti(expr).Type.Kind() == reflect.Interface {
				if condType == runtime.ConditionNil {
					condType = runtime.ConditionInterfaceNil
				} else {
					condType = runtime.ConditionInterfaceNotNil
				}
			}
			em.fb.emitIf(false, v, condType, 0, typ.Kind(), cond.Pos())
			return
		}
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
	if cond, ok := cond.(*ast.BinaryOperator); ok {
		name1, name2 := em.builtinCallName(cond.Expr1), em.builtinCallName(cond.Expr2)
		if (name1 == "len") != (name2 == "len") {
			var lenArg, expr ast.Expression
			if name1 == "len" {
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
				condType := map[ast.OperatorType]runtime.Condition{
					ast.OperatorEqual:          runtime.ConditionEqualLen,
					ast.OperatorNotEqual:       runtime.ConditionNotEqualLen,
					ast.OperatorLess:           runtime.ConditionLessLen,
					ast.OperatorLessOrEqual:    runtime.ConditionLessOrEqualLen,
					ast.OperatorGreater:        runtime.ConditionGreaterLen,
					ast.OperatorGreaterOrEqual: runtime.ConditionGreaterOrEqualLen,
				}[cond.Operator()]
				em.fb.emitIf(k2, v1, condType, v2, reflect.String, cond.Pos())
				return
			}
		}
	}

	// Binary operations that involves specific kinds of values that are
	// optimized in the VM.
	//
	// if v1 == v2
	// if v1 != v2
	// if v1 <  v2
	// if v1 <= v2
	// if v1 >  v2
	// if v1 >= v2
	if cond, ok := cond.(*ast.BinaryOperator); ok {
		t1 := em.ti(cond.Expr1).Type
		t2 := em.ti(cond.Expr2).Type
		if t1.Kind() == t2.Kind() {
			if kind := t1.Kind(); reflect.Int <= kind && kind <= reflect.Float64 {
				v1 := em.emitExpr(cond.Expr1, t1)
				v2, k2 := em.emitExprK(cond.Expr2, t2)
				var condType runtime.Condition
				if k := t1.Kind(); reflect.Uint <= k && k <= reflect.Uintptr {
					condType = map[ast.OperatorType]runtime.Condition{
						ast.OperatorEqual:          runtime.ConditionEqual,    // same as for signed integers
						ast.OperatorNotEqual:       runtime.ConditionNotEqual, // same as for signed integers
						ast.OperatorLess:           runtime.ConditionLessU,
						ast.OperatorLessOrEqual:    runtime.ConditionLessOrEqualU,
						ast.OperatorGreater:        runtime.ConditionGreaterU,
						ast.OperatorGreaterOrEqual: runtime.ConditionGreaterOrEqualU,
					}[cond.Operator()]
				} else {
					condType = map[ast.OperatorType]runtime.Condition{
						ast.OperatorEqual:          runtime.ConditionEqual,
						ast.OperatorNotEqual:       runtime.ConditionNotEqual,
						ast.OperatorLess:           runtime.ConditionLess,
						ast.OperatorLessOrEqual:    runtime.ConditionLessOrEqual,
						ast.OperatorGreater:        runtime.ConditionGreater,
						ast.OperatorGreaterOrEqual: runtime.ConditionGreaterOrEqual,
					}[cond.Operator()]
				}
				em.fb.emitIf(k2, v1, condType, v2, kind, cond.Pos())
				return
			}
		}
	}

	// // Any other binary condition is evaluated and compared to 'true'. For
	// // example 'if a == b || c == d' becomes 'if (a == b || c == d) == 1'.
	v1 := em.emitExpr(cond, em.ti(cond).Type)
	k2 := em.fb.makeIntConstant(1)
	v2 := em.fb.newRegister(reflect.Bool)
	em.fb.emitLoadNumber(intRegister, k2, v2)
	em.fb.emitIf(false, v1, runtime.ConditionEqual, v2, reflect.Bool, cond.Pos())
	return

}

func (em *emitter) emitUnaryOperator(unOp *ast.UnaryOperator, reg int8, dstType reflect.Type) {

	operand := unOp.Expr
	operandType := em.ti(operand).Type
	unOpType := em.ti(unOp).Type

	switch unOp.Operator() {

	// !operand
	case ast.OperatorNot:
		if reg == 0 {
			em.emitExprR(operand, operandType, 0)
			return
		}
		if canEmitDirectly(unOpType.Kind(), dstType.Kind()) {
			em.emitExprR(operand, operandType, reg)
			em.fb.emitSubInv(true, reg, int8(1), reg, reflect.Int)
			return
		}
		em.fb.enterScope()
		tmp := em.emitExpr(operand, operandType)
		em.fb.emitSubInv(true, tmp, int8(1), tmp, reflect.Int)
		em.changeRegister(false, tmp, reg, operandType, dstType)
		em.fb.exitScope()

	// *operand
	case ast.OperatorMultiplication:
		if reg == 0 {
			em.emitExprR(operand, operandType, 0)
			return
		}
		if canEmitDirectly(unOpType.Kind(), dstType.Kind()) {
			exprReg := em.emitExpr(operand, operandType)
			em.changeRegister(false, -exprReg, reg, operandType.Elem(), dstType)
			return
		}
		exprReg := em.emitExpr(operand, operandType)
		tmp := em.fb.newRegister(operandType.Elem().Kind())
		em.changeRegister(false, -exprReg, tmp, operandType.Elem(), operandType.Elem())
		em.changeRegister(false, tmp, reg, operandType.Elem(), dstType)

	// &operand
	case ast.OperatorAnd:
		switch operand := operand.(type) {

		// &a
		case *ast.Identifier:
			if em.fb.isVariable(operand.Name) {
				varr := em.fb.scopeLookup(operand.Name)
				em.fb.emitNew(em.types.PtrTo(unOpType), reg)
				em.fb.emitMove(false, -varr, reg, dstType.Kind(), false)
				return
			}
			// Closure variable address and Scriggo variables.
			if index, ok := em.nonLocalVarIndex(operand); ok {
				if canEmitDirectly(operandType.Kind(), dstType.Kind()) {
					em.fb.emitGetVarAddr(index, reg)
					return
				}
				tmp := em.fb.newRegister(operandType.Kind())
				em.fb.emitGetVarAddr(index, tmp)
				em.changeRegister(false, tmp, reg, operandType, dstType)
				return
			}

			panic("BUG: not implemented") // remove.

		// &*a
		case *ast.UnaryOperator:
			em.emitExprR(operand.Expr, dstType, reg)

		// &v[i]
		// (where v is a slice or an addressable array)
		case *ast.Index:
			expr := em.emitExpr(operand.Expr, em.ti(operand.Expr).Type)
			index := em.emitExpr(operand.Index, intType)
			pos := operand.Expr.Pos()
			if canEmitDirectly(unOpType.Kind(), dstType.Kind()) {
				em.fb.emitAddr(expr, index, reg, pos)
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(unOpType.Kind())
			em.fb.emitAddr(expr, index, tmp, pos)
			em.changeRegister(false, tmp, reg, unOpType, dstType)
			em.fb.exitStack()

		// &s.Field
		case *ast.Selector:
			operandExprType := em.ti(operand.Expr).Type
			expr := em.emitExpr(operand.Expr, operandExprType)
			field, _ := operandExprType.FieldByName(operand.Ident)
			index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
			pos := operand.Expr.Pos()
			if canEmitDirectly(reflect.PtrTo(field.Type).Kind(), dstType.Kind()) {
				em.fb.emitAddr(expr, index, reg, pos)
				return
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(reflect.Ptr)
			em.fb.emitAddr(expr, index, tmp, pos)
			em.changeRegister(false, tmp, reg, em.types.PtrTo(field.Type), dstType)
			em.fb.exitStack()

		// &T{..}
		case *ast.CompositeLiteral:
			tmp := em.fb.newRegister(reflect.Ptr)
			em.fb.emitNew(operandType, tmp)
			em.emitExprR(operand, operandType, -tmp)
			em.changeRegister(false, tmp, reg, unOpType, dstType)

		default:
			panic("TODO(Gianluca): not implemented")
		}

	// +operand
	case ast.OperatorAddition:
		// Nothing to do.

	// -operand
	case ast.OperatorSubtraction:
		if reg == 0 {
			em.emitExprR(operand, dstType, 0)
			return
		}
		if canEmitDirectly(operandType.Kind(), dstType.Kind()) {
			em.emitExprR(operand, dstType, reg)
			em.fb.emitSubInv(true, reg, 0, reg, dstType.Kind())
			return
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(operandType.Kind())
		em.emitExprR(operand, operandType, tmp)
		em.fb.emitSubInv(true, tmp, 0, tmp, operandType.Kind())
		em.changeRegister(false, tmp, reg, operandType, dstType)
		em.fb.exitStack()

	default:
		panic(fmt.Errorf("TODO: not implemented operator %s", unOp.Operator()))
	}

	return

}

// emitComplexOperation emits the operation on the given complex numbers putting
// the result into the given register.
func (em *emitter) emitComplexOperation(exprType reflect.Type, expr1 ast.Expression, op ast.OperatorType, expr2 ast.Expression, reg int8, dstType reflect.Type) {
	stackShift := em.fb.currentStackShift()
	em.fb.enterScope()
	index := em.fb.complexOperationIndex(op, false)
	ret := em.fb.newRegister(reflect.Complex128)
	c1 := em.fb.newRegister(reflect.Complex128)
	c2 := em.fb.newRegister(reflect.Complex128)
	em.fb.enterScope()
	em.emitExprR(expr1, exprType, c1)
	em.fb.exitScope()
	em.fb.enterScope()
	em.emitExprR(expr2, exprType, c2)
	em.fb.exitScope()
	em.fb.emitCallPredefined(index, 0, stackShift, expr1.Pos())
	em.changeRegister(false, ret, reg, exprType, dstType)
	em.fb.exitScope()
}
