// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"scriggo/ast"
	"scriggo/runtime"
)

// A varStore holds informations about closure variables, predefined variables
// and package-level variables during the emission.
type varStore struct {

	// emitter is a reference to the current emitter.
	emitter *emitter

	// indirectVars indicates if a given identifier must declare an indirect
	// variable. This field is set during the creation of the varStore and is
	// read-only.
	indirectVars map[*ast.Identifier]bool

	predefVarRef map[*runtime.Function]map[*reflect.Value]int16

	// Holds all Scriggo-defined and pre-predefined global variables.
	globals []Global

	scriggoPackageVars map[*ast.Package]map[string]int16

	closureVars map[*runtime.Function]map[string]int16
}

// newVarStore returns a new *varStore.
func newVarStore(emitter *emitter, indirectVars map[*ast.Identifier]bool) *varStore {
	return &varStore{
		emitter:            emitter,
		predefVarRef:       map[*runtime.Function]map[*reflect.Value]int16{},
		indirectVars:       indirectVars,
		scriggoPackageVars: map[*ast.Package]map[string]int16{},
		closureVars:        map[*runtime.Function]map[string]int16{},
	}
}

// closureVar returns the index of the closure variable with the given name for
// the given function. If name is not a closure var then false is returned.
func (vs *varStore) closureVar(fn *runtime.Function, name string) (int16, bool) {
	index, ok := vs.closureVars[fn][name]
	return index, ok
}

// setClosureVar the index of the closure variable name for the given function.
func (vs *varStore) setClosureVar(fn *runtime.Function, name string, index int16) {
	if vs.closureVars[fn] == nil {
		vs.closureVars[fn] = map[string]int16{}
	}
	vs.closureVars[fn][name] = index
}

func (vs *varStore) registerScriggoPackageVar(pkg *ast.Package, name string, index int16) {
	if vs.scriggoPackageVars[pkg] == nil {
		vs.scriggoPackageVars[pkg] = map[string]int16{}
	}
	vs.scriggoPackageVars[pkg][name] = index
}

func (vs *varStore) createScriggoPackageVar(pkg *ast.Package, global Global) int16 {
	index := int16(len(vs.globals))
	vs.globals = append(vs.globals, global)
	if vs.scriggoPackageVars[pkg] == nil {
		vs.scriggoPackageVars[pkg] = map[string]int16{}
	}
	vs.scriggoPackageVars[pkg][global.Name] = index
	return index
}

func (vs *varStore) scriggoPackageVar(pkg *ast.Package, name string) (int16, bool) {
	index, ok := vs.scriggoPackageVars[pkg][name]
	return index, ok
}

// mustBeDeclaredAsIndirect reports whether v must be declared as indirect.
func (vs *varStore) mustBeDeclaredAsIndirect(v *ast.Identifier) bool {
	return vs.indirectVars[v]
}

// predefVarIndex returns the index of the predefined variable v. If v is not
// available in the global slice then it is added by this call.
func (vs *varStore) predefVarIndex(v *reflect.Value, pkg, name string) int16 {
	currFn := vs.emitter.fb.fn
	if index, ok := vs.predefVarRef[currFn][v]; ok {
		return index
	}
	index := int16(len(vs.globals))
	g := newGlobal(pkg, name, v.Type().Elem(), nil)
	if !v.IsNil() {
		g.Value = v.Interface()
	}
	if vs.predefVarRef[currFn] == nil {
		vs.predefVarRef[currFn] = map[*reflect.Value]int16{}
	}
	vs.globals = append(vs.globals, g)
	vs.predefVarRef[currFn][v] = index
	return index
}

func (vs *varStore) setPredefVarIndex(fn *runtime.Function, v *reflect.Value, index int16) {
	if vs.predefVarRef[fn] == nil {
		vs.predefVarRef[fn] = map[*reflect.Value]int16{}
	}
	vs.predefVarRef[fn][v] = index
}

// getGlobals returns the slice of all Globals collected during the emission.
func (vs *varStore) getGlobals() []Global {
	return vs.globals
}

func (vs *varStore) nonLocalVarIdentifier(v *ast.Identifier) (index int, ok bool) {
	ti := vs.emitter.ti(v)
	currFn := vs.emitter.fb.fn
	currPkg := vs.emitter.pkg
	// v is a predefined variable.
	if ti != nil && ti.IsPredefined() {
		index := vs.predefVarIndex(ti.value.(*reflect.Value), ti.PredefPackageName, v.Name)
		return int(index), true
	}
	if index, ok := vs.closureVar(currFn, v.Name); ok {
		return int(index), true
	}
	if index, ok := vs.scriggoPackageVar(currPkg, v.Name); ok {
		return int(index), true
	}
	return 0, false
}

func (vs *varStore) nonLocalVarSelector(v *ast.Selector) (index int, ok bool) {
	ti := vs.emitter.ti(v)
	currFn := vs.emitter.fb.fn
	currPkg := vs.emitter.pkg
	// v is a predefined variable.
	if ti != nil && ti.IsPredefined() {
		index := vs.predefVarIndex(ti.value.(*reflect.Value), ti.PredefPackageName, v.Ident)
		return int(index), true
	}
	name := v.Expr.(*ast.Identifier).Name + "." + v.Ident
	if index, ok := vs.closureVar(currFn, name); ok {
		return int(index), true
	}
	if index, ok := vs.scriggoPackageVar(currPkg, name); ok {
		return int(index), true
	}
	return 0, false
}
