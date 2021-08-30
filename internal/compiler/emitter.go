// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"strings"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/compiler/types"
	"github.com/open2b/scriggo/internal/runtime"
)

// An emitter emits instructions for the VM.
type emitter struct {
	fnStore  *functionStore
	varStore *varStore

	// fb is the current function builder.
	fb *functionBuilder

	labels map[*runtime.Function]map[string]label

	// pkg is the package that is currently being emitted.
	pkg *ast.Package

	// typeInfos maps nodes to their type info.
	// Should be accessed using method 'ti'.
	typeInfos map[ast.Node]*typeInfo

	// formatTypes maps a format to its type.
	formatTypes map[ast.Format]reflect.Type

	// isTemplate reports whether the emitter is currently emitting a template.
	isTemplate bool

	// rangeLabels is a list of current active Ranges. First element is the
	// Range address, second refers to the first instruction outside Range's
	// body.
	rangeLabels [][2]label

	// breakable is true if emitting a "breakable" statement (except ForRange,
	// which implements his own "breaking" system).
	breakable bool

	// inForRange reports whether the emitter is currently emitting the body of
	// a ForRange node.
	inForRange bool

	// breakLabel, if not nil, is the label to which pre-stated "breaks" must
	// jump.
	breakLabel *label

	// inURL indicates if the emitter is currently inside an *ast.URL node.
	inURL bool

	// isURLSet reports whether the current URL, if inURL is true, is a set of URLs.
	isURLSet bool

	// types refers the types of the current compilation and it is used to
	// create and manipulate types and values, both predefined and defined only
	// by Scriggo.
	types *types.Types

	// alreadyEmittedFuncs reports if a given function has already been emitted
	// by storing the emitted function. This avoids emitting the same function
	// twice, that also ensures that init functions are called just once when
	// imported by two different packages.
	// TODO: consider moving this field to the funcStore.
	alreadyEmittedFuncs map[*ast.Func]*runtime.Function

	// alreadyInitializedVars maps the identifiers of package variable
	// declarations to their index (used in the SetVar/GetVar instructions).
	// This map has three purposes:
	//
	//  - speed up the compilation avoiding emitting a variable twice.
	//  - avoid emitting the same variable with two different indexes, that
	//    would result in an invalid behavior.
	//  - avoid initializing the same variable more than once, that would
	//    result in an invalid behavior.
	alreadyInitializedVars map[*ast.Identifier]int16

	// alreadyInitializedTemplatePkgs keeps track of the template packages for
	// which the initialization code has already been emitted.
	alreadyInitializedTemplatePkgs map[string]bool
}

// newEmitter returns a new emitter with the given type infos, format types,
// indirect variables and options.
func newEmitter(typeInfos map[ast.Node]*typeInfo, formatTypes map[ast.Format]reflect.Type, indirectVars map[*ast.Identifier]bool) *emitter {
	em := &emitter{
		labels:                         make(map[*runtime.Function]map[string]label),
		typeInfos:                      typeInfos,
		formatTypes:                    formatTypes,
		types:                          types.NewTypes(), // TODO: this is wrong: the instance should be taken from the type checker.
		alreadyEmittedFuncs:            map[*ast.Func]*runtime.Function{},
		alreadyInitializedVars:         map[*ast.Identifier]int16{},
		alreadyInitializedTemplatePkgs: map[string]bool{},
	}
	em.fnStore = newFunctionStore(em)
	em.varStore = newVarStore(em, indirectVars)
	return em
}

// ti returns the type info of node n.
func (em *emitter) ti(n ast.Node) *typeInfo {
	if ti, ok := em.typeInfos[n]; ok {
		if ti.valueType != nil {
			ti.Type = ti.valueType
		}
		return ti
	}
	return nil
}

// typ returns the reflect.Type associated to the given expression.
func (em *emitter) typ(expr ast.Expression) reflect.Type {
	return em.ti(expr).Type
}

// emitPackage emits a package and returns the exported functions, the exported
// variables and the init functions.
// extendingFile reports whether emitPackage is going to emit a package that
// extends another file.
func (em *emitter) emitPackage(pkg *ast.Package, extendingFile bool, path string) (map[string]*runtime.Function, map[string]int16, []*runtime.Function) {

	if !extendingFile {
		em.pkg = pkg
	}

	// List of all "init" functions in current package.
	inits := []*runtime.Function{}

	// Emit the imports.
	for _, decl := range pkg.Declarations {
		if node, ok := decl.(*ast.Import); ok {
			pkgInits := em.emitImport(node, false)
			// Do not add duplicated init functions.
			for _, pkgInit := range pkgInits {
				add := true
				for _, ini := range inits {
					if ini == pkgInit {
						add = false
						break
					}
				}
				if add {
					inits = append(inits, pkgInits...)
				}
			}
		}
	}

	// Package level functions.
	functions := map[string]*runtime.Function{}

	// initToBuild is the index of the next "init" function to build.
	initToBuild := len(inits)

	if extendingFile {
		// The function declarations have already been added to the list of
		// available functions, so they can't be added twice.
	} else {
		// Store all function declarations in current package before building
		// their bodies: order of declaration doesn't matter at package level.
		for _, dec := range pkg.Declarations {
			if fun, ok := dec.(*ast.Func); ok {
				var fn *runtime.Function
				if emFn, ok := em.alreadyEmittedFuncs[fun]; ok {
					fn = emFn
				} else {
					if fun.Type.Macro {
						fn = newMacro("main", fun.Ident.Name, fun.Type.Reflect, fun.Format, path, fun.Pos())
					} else {
						fn = newFunction("main", fun.Ident.Name, fun.Type.Reflect, path, fun.Pos())
					}
				}
				if fun.Ident.Name == "init" {
					inits = append(inits, fn)
					continue
				}
				em.fnStore.makeAvailableScriggoFn(em.pkg, fun.Ident.Name, fn)
				isDummyMacroForRender := strings.HasPrefix(fun.Ident.Name, `"`) && strings.HasSuffix(fun.Ident.Name, `"`)
				if isExported(fun.Ident.Name) || isDummyMacroForRender {
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
				initVarsFn = newFunction("main", "$initvars", reflect.FuncOf(nil, nil, false), path, &ast.Position{})
				em.fnStore.makeAvailableScriggoFn(em.pkg, "$initvars", initVarsFn)
				initVarsFb = newBuilder(initVarsFn, path)
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
				// This variable has already been emitted and initialized; just
				// add it to the 'vars' slice, everything else has already been
				// done.
				if index, ok := em.alreadyInitializedVars[v]; ok {
					vars[v.Name] = index
					continue
				}
				varType := em.typ(v)
				varr := em.fb.newRegister(varType.Kind())
				addresses[i] = em.addressLocalVar(varr, varType, v.Pos(), 0)
				// Store the variable register. It will be used later to store
				// initialized value inside the proper global index.
				pkgVarRegs[v.Name] = varr
				pkgVarTypes[v.Name] = varType
				index := em.varStore.createScriggoPackageVar(em.pkg, newGlobal(pkg.Name, v.Name, varType, reflect.Value{}))
				em.alreadyInitializedVars[v] = index
				vars[v.Name] = index
			}
			em.assignValuesToAddresses(addresses, n.Rhs)
			for name, reg := range pkgVarRegs {
				index := vars[name]
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
			if _, ok := em.alreadyEmittedFuncs[n]; ok {
				// Function has already been emitted, nothing to do.
				continue
			}
			if n.Ident.Name == "init" {
				fn = inits[initToBuild]
				initToBuild++
			} else {
				fn, _ = em.fnStore.availableScriggoFn(em.pkg, n.Ident.Name)
			}
			em.fb = newBuilder(fn, path)
			em.fb.enterScope()
			// If this is the main function, functions that initialize variables
			// must be called before executing every other statement of the main
			// function.
			if n.Ident.Name == "main" {
				// First: initialize the package variables.
				if initVarsFn != nil {
					iv, _ := em.fnStore.availableScriggoFn(em.pkg, "$initvars")
					index := em.fb.addFunction(iv) // TODO: check addFunction
					em.fb.emitCallFunc(index, runtime.StackShift{}, nil)
				}
				// Second: call all init functions, in order.
				for _, initFunc := range inits {
					index := em.fb.addFunction(initFunc)
					em.fb.emitCallFunc(index, runtime.StackShift{}, nil)
				}
			}
			em.prepareFunctionBodyParameters(n)
			em.emitNodes(n.Body.Nodes)
			em.fb.end()
			em.fb.exitScope()
			em.alreadyEmittedFuncs[n] = fn
		}
	}

	if initVarsFn != nil {
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
//
// Returns the index (and the type) of the registers that will hold the function
// return parameters.
//
// Note that while prepareCallParameters is called before calling the function,
// prepareFunctionBodyParameters is called before emitting its body.
func (em *emitter) prepareCallParameters(fType reflect.Type, fArgs []ast.Expression, opts callOptions) ([]int8, []reflect.Type) {

	fNumOut := fType.NumOut()
	fNumIn := fType.NumIn()
	fOutRegs := make([]int8, fNumOut)
	fOutTypes := make([]reflect.Type, fNumOut)

	// Reserve space for the output parameters.
	for i := 0; i < fNumOut; i++ {
		t := fType.Out(i)
		fOutRegs[i] = em.fb.newRegister(t.Kind())
		fOutTypes[i] = t
	}

	// Emit the method receiver, if necessary.
	if opts.receiverAsArg {
		reg := em.fb.newRegister(em.typ(fArgs[0]).Kind())
		em.fb.enterStack()
		em.emitExprR(fArgs[0], em.typ(fArgs[0]), reg)
		em.fb.exitStack()
		fArgs = fArgs[1:]
	}

	// Variadic function calls.
	if fType.IsVariadic() {

		// f(g(..)) where 'f' is variadic.
		if em.isSpecialCall(fArgs) {
			g := fArgs[0].(*ast.Call)
			gOutCount, _ := em.numOut(g) // count of output parameters of 'g(..)'.
			nonVarArgsCount := fNumIn - 1
			varArgsCount := gOutCount - (fNumIn - 1)
			// Reserve space for non variadic parameters.
			var nonVarParamRegs []int8
			for i := 0; i < nonVarArgsCount; i++ {
				reg := em.fb.newRegister(fType.In(i).Kind())
				nonVarParamRegs = append(nonVarParamRegs, reg)
			}
			// Reserve space for variadic parameters.
			var varParamRegs []int8
			sliceType := fType.In(fNumIn - 1)
			if opts.predefined {
				// When calling a predefined variadic function, the variadic
				// parameters must be emitted each one in its register,
				// separately.
				for i := 0; i < varArgsCount; i++ {
					reg := em.fb.newRegister(sliceType.Elem().Kind())
					varParamRegs = append(varParamRegs, reg)
				}
			} else {
				// When calling a non-predefined variadic function, the
				// variadic parameters must be emitted inside a slice.
				varParamRegs = []int8{em.fb.newRegister(reflect.Slice)}
			}
			em.fb.enterStack()
			gOutRegs, gOutTypes := em.emitCallNode(g, false, false, runtime.ReturnString)
			// Move the non-variadic parameters to the space reserved.
			for i := 0; i < nonVarArgsCount; i++ {
				dstType := fType.In(i)
				reg := nonVarParamRegs[i]
				em.changeRegister(false, gOutRegs[i], reg, gOutTypes[i], dstType)
			}
			// Handle variadic parameters.
			if opts.predefined {
				// Just move parameters to their corresponding register.
				for i := range varParamRegs {
					em.changeRegister(false, gOutRegs[nonVarArgsCount+i], varParamRegs[i], gOutTypes[nonVarArgsCount+i], sliceType.Elem())
				}
			} else {
				slice := varParamRegs[0]
				if varArgsCount == 0 {
					// The slice must be nil, not empty.
					c := em.fb.makeGeneralValue(reflect.Zero(sliceType))
					em.changeRegister(true, c, slice, sliceType, sliceType)
				} else {
					pos := fArgs[0].Pos()
					em.fb.emitMakeSlice(true, true, sliceType, int8(varArgsCount), int8(varArgsCount), slice, pos)
					for i := nonVarArgsCount; i < len(gOutRegs); i++ {
						gArgReg := gOutRegs[i]
						gArgType := gOutTypes[i]
						index := em.fb.newRegister(reflect.Int)
						em.changeRegister(true, int8(i-nonVarArgsCount), index, intType, intType)
						if canEmitDirectly(gArgType.Kind(), sliceType.Elem().Kind()) {
							em.fb.emitSetSlice(false, slice, gArgReg, index, pos, sliceType.Elem().Kind())
						} else {
							tmp := em.fb.newRegister(sliceType.Elem().Kind())
							em.changeRegister(false, gArgReg, tmp, gArgType, sliceType.Elem())
							em.fb.emitSetSlice(false, slice, tmp, index, pos, sliceType.Elem().Kind())
						}
					}
				}
			}
			em.fb.exitStack()
			return fOutRegs, fOutTypes
		}

		// Emit the non-variadic arguments.
		for i := 0; i < fNumIn-1; i++ {
			t := fType.In(i)
			reg := em.fb.newRegister(t.Kind())
			em.fb.enterStack()
			em.emitExprR(fArgs[i], t, reg)
			em.fb.exitStack()
		}

		// Emit the slice argument in case of 'f(x, y, z, slice...)'.
		if opts.callHasDots {
			slice := fArgs[len(fArgs)-1]
			sliceType := fType.In(fType.NumIn() - 1)
			reg := em.fb.newRegister(sliceType.Kind())
			em.fb.enterStack()
			em.emitExprR(slice, sliceType, reg)
			em.fb.exitStack()
			return fOutRegs, fOutTypes
		}

		// Emit the variadic arguments.
		varArgsCount := len(fArgs) - (fNumIn - 1)
		t := fType.In(fNumIn - 1).Elem()
		if opts.predefined {
			for i := 0; i < varArgsCount; i++ {
				reg := em.fb.newRegister(t.Kind())
				em.fb.enterStack()
				em.emitExprR(fArgs[i+fNumIn-1], t, reg)
				em.fb.exitStack()
			}
		} else {
			slice := em.fb.newRegister(reflect.Slice)
			em.fb.emitMakeSlice(true, true, fType.In(fNumIn-1), int8(varArgsCount), int8(varArgsCount), slice, nil) // TODO: fix pos.
			for i := 0; i < varArgsCount; i++ {
				tmp := em.fb.newRegister(t.Kind())
				em.fb.enterStack()
				em.emitExprR(fArgs[i+fNumIn-1], t, tmp)
				em.fb.exitStack()
				index := em.fb.newRegister(reflect.Int)
				em.fb.emitMove(true, int8(i), index, reflect.Int)
				pos := fArgs[len(fArgs)-1].Pos()
				em.fb.emitSetSlice(false, slice, tmp, index, pos, fType.In(fNumIn-1).Elem().Kind())
			}
		}

		return fOutRegs, fOutTypes
	}

	// Non-variadic function call.

	// f(g(..)), where f takes more than one parameter.
	if fNumIn > 1 && len(fArgs) == 1 {
		gOutRegs, gOutTypes := em.emitCallNode(fArgs[0].(*ast.Call), false, false, runtime.ReturnString)
		for i := range gOutRegs {
			dstType := fType.In(i)
			reg := em.fb.newRegister(dstType.Kind())
			em.changeRegister(false, gOutRegs[i], reg, gOutTypes[i], dstType)
		}
		return fOutRegs, fOutTypes
	}

	// Emit the arguments in a non-variadic non-special call.
	for i := 0; i < fNumIn; i++ {
		t := fType.In(i)
		reg := em.fb.newRegister(t.Kind())
		em.fb.enterStack()
		em.emitExprR(fArgs[i], t, reg)
		em.fb.exitStack()
	}
	return fOutRegs, fOutTypes

}

// prepareFunctionBodyParameters prepares fun's parameters (in and out) before
// emitting its body.
//
// While prepareCallParameters is called before calling the function,
// prepareFunctionBodyParameters is called before emitting its body.
//
// prepareFunctionBodyParameters also sets the FinalRegs field of the current
// function, if necessary.
func (em *emitter) prepareFunctionBodyParameters(fn *ast.Func) {

	// Reserve space for the return parameters and eventually bind them.
	for _, out := range fn.Type.Result {
		kind := em.typ(out.Type).Kind()
		reg := em.fb.newRegister(kind)
		if out.Ident != nil && !isBlankIdentifier(out.Ident) {
			em.fb.bindVarReg(out.Ident.Name, reg)
		}
	}

	// Reserve space for the input parameters and eventually bind them.
	for i, inParam := range fn.Type.Parameters {
		kind := em.typ(inParam.Type).Kind()
		if fn.Type.IsVariadic && i == len(fn.Type.Parameters)-1 {
			kind = reflect.Slice
		}
		if inParam.Ident == nil || isBlankIdentifier(inParam.Ident) {
			// Just reserve space for this parameter.
			_ = em.fb.newRegister(kind)
		} else {
			// Here it is not necessary to check if the input parameter should
			// use an indirect register or not, because in both cases the space
			// for such register must be reserved.
			//
			// Indirect input parameters are handled below.
			arg := em.fb.newRegister(kind)
			em.fb.bindVarReg(inParam.Ident.Name, arg)
		}
	}

	// Replace indirect return named parameters with indirect registers and
	// inform the finalizer to copy the values when the function returns.
	for _, out := range fn.Type.Result {
		if out.Ident != nil && em.varStore.mustBeDeclaredAsIndirect(out.Ident) {
			dst := em.fb.scopeLookup(out.Ident.Name)
			reg := em.fb.newIndirectRegister()
			em.fb.emitNew(em.typ(out.Type), -reg)
			em.fb.bindVarReg(out.Ident.Name, reg)
			em.fb.fn.FinalRegs = append(em.fb.fn.FinalRegs, [2]int8{-reg, dst})
		}
	}

	// Rebind input parameters that should be declared as indirect.
	for _, param := range fn.Type.Parameters {
		if em.varStore.mustBeDeclaredAsIndirect(param.Ident) {
			// reg is used only to read input parameters; after copying values
			// into the indirect register it is not used anymore.
			// In this way, the caller of fn should not care if the input
			// parameter should be indirect or not.
			reg := em.fb.scopeLookup(param.Ident.Name)
			indirect := em.fb.newIndirectRegister()
			typ := em.typ(param.Type)
			em.fb.emitNew(typ, -indirect)
			em.changeRegister(false, reg, indirect, typ, typ)
			em.fb.bindVarReg(param.Ident.Name, indirect)
		}

	}

}

// emitCallNode emits instructions for a function call node. It returns the
// registers and the reflect types of the returned values.
// goStmt indicates if the call node belongs to a 'go statement', while
// deferStmt reports whether it must be deferred.
func (em *emitter) emitCallNode(call *ast.Call, goStmt bool, deferStmt bool, toFormat ast.Format) ([]int8, []reflect.Type) {

	funTi := em.ti(call.Func)

	// Method call on a interface value.
	if funTi.MethodType == methodCallInterface {
		rcvrExpr := call.Func.(*ast.Selector).Expr
		rcvrType := em.typ(rcvrExpr)
		rcvr := em.emitExpr(rcvrExpr, rcvrType)
		// MethodValue reads receiver from general.
		if kindToType(rcvrType.Kind()) != generalRegister {
			// TODO(Gianluca): put rcvr in general
			panic("BUG: not implemented") // remove.
		}
		method := em.fb.newRegister(reflect.Func)
		name := call.Func.(*ast.Selector).Ident
		s := em.fb.makeStringValue(name)
		em.fb.emitMethodValue(s, rcvr, method, call.Func.Pos())
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
		em.fb.emitCallIndirect(method, 0, stackShift, call.Pos(), funTi.Type, toFormat)
		return regs, types
	}

	// Predefined function (identifiers, selectors etc...).
	// Calls of predefined functions stored in builtin variables are handled as
	// common "indirect" calls.
	if funTi.IsNative() && !funTi.Addressable() {
		if funTi.MethodType == methodCallConcrete {
			rcv := call.Func.(*ast.Selector).Expr // TODO(Gianluca): is this correct?
			call.Args = append([]ast.Expression{rcv}, call.Args...)
		}
		stackShift := em.fb.currentStackShift()
		opts := callOptions{
			predefined:    true,
			receiverAsArg: funTi.MethodType == methodCallConcrete,
			callHasDots:   call.IsVariadic,
		}
		regs, types := em.prepareCallParameters(funTi.Type, call.Args, opts)
		index, _ := em.fnStore.predefFunc(call.Func, true)
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
		em.fb.emitCallNative(index, int8(numVar), stackShift, call.Pos())
		return regs, types
	}

	// Scriggo-defined function (identifier).
	if ident, ok := call.Func.(*ast.Identifier); ok && !em.fb.declaredInFunc(ident.Name) {
		if fn, ok := em.fnStore.availableScriggoFn(em.pkg, ident.Name); ok {
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
			if fn.Macro {
				em.fb.emitCallMacro(index, stackShift, call.Pos(), toFormat)
			} else {
				em.fb.emitCallFunc(index, stackShift, call.Pos())
			}
			return regs, types
		}
	}

	// Scriggo-defined function (selector).
	if selector, ok := call.Func.(*ast.Selector); ok {
		if ident, ok := selector.Expr.(*ast.Identifier); ok {
			if fun, ok := em.fnStore.availableScriggoFn(em.pkg, ident.Name+"."+selector.Ident); ok {
				stackShift := em.fb.currentStackShift()
				regs, types := em.prepareCallParameters(fun.Type, call.Args, callOptions{callHasDots: call.IsVariadic})
				index := em.fnStore.scriggoFnIndex(fun)
				if goStmt {
					em.fb.emitGo()
				}
				if deferStmt {
					panic("BUG: not implemented") // remove.
				}
				if fun.Macro {
					em.fb.emitCallMacro(index, stackShift, call.Pos(), toFormat)
				} else {
					em.fb.emitCallFunc(index, stackShift, call.Pos())
				}
				return regs, types
			}
		}
	}

	// Indirect function.
	reg := em.emitExpr(call.Func, em.typ(call.Func))
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
	em.fb.emitCallIndirect(reg, int8(runtime.NoVariadicArgs), stackShift, call.Pos(), funTi.Type, toFormat)

	return regs, types
}

// emitBuiltin emits instructions for a builtin call, writing the result, if
// necessary, into the register reg.
func (em *emitter) emitBuiltin(call *ast.Call, reg int8, dstType reflect.Type) {
	args := call.Args
	switch call.Func.(*ast.Identifier).Name {
	case "append":
		// Change the tree if necessary.
		if call.IR.AppendArg1 != nil {
			call.Args[1] = call.IR.AppendArg1
		}
		sliceType := em.typ(args[0])
		slice := em.emitExpr(args[0], sliceType)
		if call.IsVariadic {
			tmp := em.fb.newRegister(sliceType.Kind())
			em.fb.emitMove(false, slice, tmp, sliceType.Kind())
			arg := em.emitExpr(args[1], em.typ(args[1]))
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
		s := em.emitExpr(args[0], em.typ(args[0]))
		if canEmitDirectly(intType.Kind(), dstType.Kind()) {
			em.fb.emitCap(s, reg)
			return
		}
		tmp := em.fb.newRegister(intType.Kind())
		em.fb.emitCap(s, tmp)
		em.changeRegister(false, tmp, reg, intType, dstType)
	case "close":
		chann := em.emitExpr(args[0], em.typ(args[0]))
		em.fb.emitClose(chann, call.Pos())
	case "complex":
		floatType := em.typ(args[0])
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
		dst := em.emitExpr(args[0], em.typ(args[0]))
		src := em.emitExpr(args[1], em.typ(args[1]))
		em.fb.enterStack()
		// If src is a string, replace the 'src' register with a slice of byte
		// that contains such string.
		if stringType := em.typ(args[1]); stringType.Kind() == reflect.String {
			byteSlice := em.fb.newRegister(reflect.Slice)
			em.changeRegister(false, src, byteSlice, stringType, em.typ(args[0]))
			src = byteSlice
		}
		if reg == 0 {
			em.fb.emitCopy(dst, src, 0)
			return
		}
		if canEmitDirectly(reflect.Int, dstType.Kind()) {
			em.fb.emitCopy(dst, src, reg)
			em.fb.exitStack()
			return
		}
		tmp := em.fb.newRegister(reflect.Int)
		em.fb.emitCopy(dst, src, tmp)
		em.changeRegister(false, tmp, reg, intType, dstType)
		em.fb.exitStack()
	case "delete":
		mapp := em.emitExpr(args[0], em.typ(args[0]))
		key := em.emitExpr(args[1], emptyInterfaceType)
		em.fb.emitDelete(mapp, key)
	case "len":
		typ := em.typ(args[0])
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
		typ := em.typ(args[0])
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
			var kCapacity bool
			var capacity int8
			if len(args) == 1 {
				capacity = 0
				kCapacity = true
			} else {
				capacity, kCapacity = em.emitExprK(args[1], intType)
			}
			chanType := em.typ(args[0])
			em.fb.emitMakeChan(chanType, kCapacity, capacity, reg, call.Pos())
		default:
			panic("bug")
		}
	case "new":
		em.fb.emitNew(em.typ(args[0]), reg)
	case "panic":
		arg := em.emitExpr(args[0], emptyInterfaceType)
		em.fb.emitPanic(arg, nil, call.Pos())
	case "print":
		if em.isSpecialCall(args) {
			em.fb.enterStack()
			argRegs, argTypes := em.emitCallNode(args[0].(*ast.Call), false, false, runtime.ReturnString)
			for i := range argRegs {
				if canEmitDirectly(argTypes[i].Kind(), reflect.Interface) {
					em.fb.emitPrint(argRegs[i])
				} else {
					em.fb.enterStack()
					tmp := em.fb.newRegister(reflect.Interface)
					em.changeRegister(false, argRegs[i], tmp, argTypes[i], emptyInterfaceType)
					em.fb.emitPrint(tmp)
					em.fb.exitStack()
				}
			}
			em.fb.exitStack()
		} else {
			for _, argExpr := range args {
				em.fb.enterStack()
				arg := em.emitExpr(argExpr, emptyInterfaceType)
				em.fb.emitPrint(arg)
				em.fb.exitStack()
			}
		}
	case "println":
		if em.isSpecialCall(args) {
			em.fb.enterStack()
			argRegs, argTypes := em.emitCallNode(args[0].(*ast.Call), false, false, runtime.ReturnString)
			for i := range argRegs {
				if i > 0 {
					str := em.fb.makeStringValue(" ")
					sep := em.fb.newRegister(reflect.Interface)
					em.changeRegister(true, str, sep, stringType, emptyInterfaceType)
					em.fb.emitPrint(sep)
				}
				if canEmitDirectly(argTypes[i].Kind(), reflect.Interface) {
					em.fb.emitPrint(argRegs[i])
				} else {
					em.fb.enterStack()
					tmp := em.fb.newRegister(reflect.Interface)
					em.changeRegister(false, argRegs[i], tmp, argTypes[i], emptyInterfaceType)
					em.fb.emitPrint(tmp)
					em.fb.exitStack()
				}
			}
			em.fb.exitStack()
		} else {
			for i, argExpr := range args {
				if i > 0 {
					em.fb.enterStack()
					str := em.fb.makeStringValue(" ")
					sep := em.fb.newRegister(reflect.Interface)
					em.changeRegister(true, str, sep, stringType, emptyInterfaceType)
					em.fb.emitPrint(sep)
					em.fb.exitStack()
				}
				em.fb.enterStack()
				arg := em.emitExpr(argExpr, emptyInterfaceType)
				em.fb.emitPrint(arg)
				em.fb.exitStack()
			}
		}
		em.fb.enterStack()
		str := em.fb.makeStringValue("\n")
		sep := em.fb.newRegister(reflect.Interface)
		em.changeRegister(true, str, sep, stringType, emptyInterfaceType)
		em.fb.emitPrint(sep)
		em.fb.exitStack()
	case "real", "imag":
		complexType := em.typ(args[0])
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

// invertedOperatorType returns the inverted operator type of op.
func invertedOperatorType(op ast.OperatorType) ast.OperatorType {
	switch op {
	case ast.OperatorEqual:
		return ast.OperatorEqual
	case ast.OperatorNotEqual:
		return ast.OperatorNotEqual
	case ast.OperatorLess:
		return ast.OperatorGreater
	case ast.OperatorLessEqual:
		return ast.OperatorGreaterEqual
	case ast.OperatorGreater:
		return ast.OperatorLess
	case ast.OperatorGreaterEqual:
		return ast.OperatorLessEqual
	}
	panic("unexpected operator")
}

// emitCondition emits the instructions for a condition. The last instruction
// emitted is always the "If" instruction.
func (em *emitter) emitCondition(cond ast.Expression) {

	// Emit code for a boolean constant condition.
	if ti := em.ti(cond); ti != nil && ti.HasValue() && !ti.IsNative() {
		// The condition of the 'if' instruction of VM is a binary operation,
		// so the boolean constant expression 'x' is emitted as 'x == true'.
		var c int8 = 0
		if ti.value.(int64) == 1 {
			c = 1
		}
		r := em.fb.newRegister(reflect.Int)
		em.fb.emitMove(true, c, r, reflect.Int)
		em.fb.emitIf(false, r, runtime.ConditionNotZero, 0, reflect.Int, cond.Pos())
		return
	}

	switch cond := cond.(type) {

	case *ast.BinaryOperator:

		if cond.Op == ast.OperatorEqual || cond.Op == ast.OperatorNotEqual {

			// Emit code for comparison with 0.
			//
			//   if x == 0
			//   if 0 == x
			//   if x != 0
			//   if 0 != x
			//
			if expr := em.comparisonWithZeroInteger(cond); expr != nil {
				typ := em.typ(expr)
				x := em.emitExpr(expr, typ)
				condition := runtime.ConditionZero
				if cond.Operator() == ast.OperatorNotEqual {
					condition = runtime.ConditionNotZero
				}
				em.fb.emitIf(false, x, condition, 0, typ.Kind(), expr.Pos())
				return
			}

			// Emit code for comparison with nil.
			//
			//   if x == nil
			//   if x != nil
			//   if nil == x
			//   if nil != x
			//
			if em.ti(cond.Expr1).Nil() != em.ti(cond.Expr2).Nil() {
				expr := cond.Expr1
				if em.ti(cond.Expr1).Nil() {
					expr = cond.Expr2
				}
				typ := em.typ(expr)
				x := em.emitExpr(expr, typ)
				var condition runtime.Condition
				if typ.Kind() == reflect.Interface {
					condition = runtime.ConditionInterfaceNil
					if cond.Operator() == ast.OperatorNotEqual {
						condition = runtime.ConditionInterfaceNotNil
					}
				} else {
					condition = runtime.ConditionNil
					if cond.Operator() == ast.OperatorNotEqual {
						condition = runtime.ConditionNotNil
					}
				}
				em.fb.emitIf(false, x, condition, 0, typ.Kind(), cond.Pos())
				return
			}

		}

		if ast.OperatorEqual <= cond.Op && cond.Op <= ast.OperatorGreaterEqual {

			// Emit code for comparing the length of a string to a value.
			//
			//   if len(x) == y
			//   if len(x) != y
			//   if len(x) <  y
			//   if len(x) <= y
			//   if len(x) >  y
			//   if len(x) >= y
			//   if x == len(y)
			//   if x != len(y)
			//   if x <  len(y)
			//   if x <= len(y)
			//   if x >  len(y)
			//   if x >= len(y)
			//
			name1 := em.builtinCallName(cond.Expr1)
			name2 := em.builtinCallName(cond.Expr2)
			if (name1 == "len") != (name2 == "len") {
				op := cond.Operator()
				var lenArg, expr ast.Expression
				if name1 == "len" {
					lenArg = cond.Expr1.(*ast.Call).Args[0]
					expr = cond.Expr2
				} else {
					lenArg = cond.Expr2.(*ast.Call).Args[0]
					expr = cond.Expr1
					op = invertedOperatorType(op)
				}
				if em.typ(lenArg).Kind() == reflect.String { // len is optimized for strings only.
					x := em.emitExpr(lenArg, em.typ(lenArg))
					typ := em.typ(expr)
					y, ky := em.emitExprK(expr, typ)
					var condition runtime.Condition
					switch op {
					case ast.OperatorEqual:
						condition = runtime.ConditionLenEqual
					case ast.OperatorNotEqual:
						condition = runtime.ConditionLenNotEqual
					case ast.OperatorLess:
						condition = runtime.ConditionLenLess
					case ast.OperatorLessEqual:
						condition = runtime.ConditionLenLessEqual
					case ast.OperatorGreater:
						condition = runtime.ConditionLenGreater
					case ast.OperatorGreaterEqual:
						condition = runtime.ConditionLenGreaterEqual
					default:
						panic("unexpected operator")
					}
					em.fb.emitIf(ky, x, condition, y, reflect.String, cond.Pos())
					return
				}
			}

			// Emit code to compare two values.
			//
			//   if x == y
			//   if x != y
			//   if x <  y
			//   if x <= y
			//   if x >  y
			//   if x >= y
			//
			t1 := em.typ(cond.Expr1)
			t2 := em.typ(cond.Expr2)
			x := em.emitExpr(cond.Expr1, t1)
			y, ky := em.emitExprK(cond.Expr2, t2)
			em.emitComparison(cond.Operator(), ky, x, y, t1, t2, cond.Pos())
			return

		}

		// Emit code for a contains expression.
		//
		//  x contains y
		//
		if cond.Op == ast.OperatorContains || cond.Op == ast.OperatorNotContains {
			not := cond.Op == ast.OperatorNotContains
			t1 := em.typ(cond.Expr1)
			t2 := em.typ(cond.Expr2)
			x := em.emitExpr(cond.Expr1, t1)
			if t2 == nil {
				em.emitContains(not, false, x, 0, t1, nil, cond.Pos())
			} else {
				y, ky := em.emitExprK(cond.Expr2, t2)
				em.emitContains(not, ky, x, y, t1, t2, cond.Pos())
			}
			return
		}

	case *ast.UnaryOperator:

		// Emit code for the negation of a value.
		//
		//   if !x
		//
		if cond.Operator() == ast.OperatorNot {
			c := em.emitExpr(cond.Expr, em.typ(cond.Expr))
			em.fb.emitIf(false, c, runtime.ConditionZero, 0, reflect.Bool, cond.Pos())
			return
		}

	}

	// Emit code for all other conditions.
	x := em.emitExpr(cond, em.typ(cond))
	em.fb.emitIf(false, x, runtime.ConditionNotZero, 0, reflect.Bool, cond.Pos())

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
	em.fb.emitCallNative(index, 0, stackShift, expr1.Pos())
	em.changeRegister(false, ret, reg, exprType, dstType)
	em.fb.exitScope()
}
