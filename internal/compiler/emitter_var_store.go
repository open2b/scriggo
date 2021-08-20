// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/runtime"
)

// A varStore holds information about closure variables, predefined variables
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

	scriggoPackageVarRefs map[*ast.Package]map[string]int16

	closureVars map[*runtime.Function]map[string]int16
}

// newVarStore returns a new *varStore.
func newVarStore(emitter *emitter, indirectVars map[*ast.Identifier]bool) *varStore {
	return &varStore{
		emitter:               emitter,
		predefVarRef:          map[*runtime.Function]map[*reflect.Value]int16{},
		indirectVars:          indirectVars,
		scriggoPackageVarRefs: map[*ast.Package]map[string]int16{},
		closureVars:           map[*runtime.Function]map[string]int16{},
	}
}

// setClosureVar the index of the closure variable name for the given function.
func (vs *varStore) setClosureVar(fn *runtime.Function, name string, index int16) {
	if vs.closureVars[fn] == nil {
		vs.closureVars[fn] = map[string]int16{}
	}
	vs.closureVars[fn][name] = index
}

// bindScriggoPackageVar binds name to the package variable defined in Scriggo
// located at the given index, making it available for use in pkg.
func (vs *varStore) bindScriggoPackageVar(pkg *ast.Package, name string, index int16) {
	if vs.scriggoPackageVarRefs[pkg] == nil {
		vs.scriggoPackageVarRefs[pkg] = map[string]int16{}
	}
	vs.scriggoPackageVarRefs[pkg][name] = index
}

func (vs *varStore) createScriggoPackageVar(pkg *ast.Package, global Global) int16 {
	index := int16(len(vs.globals))
	vs.globals = append(vs.globals, global)
	if vs.scriggoPackageVarRefs[pkg] == nil {
		vs.scriggoPackageVarRefs[pkg] = map[string]int16{}
	}
	vs.scriggoPackageVarRefs[pkg][global.Name] = index
	return index
}

// mustBeDeclaredAsIndirect reports whether v must be declared as indirect.
func (vs *varStore) mustBeDeclaredAsIndirect(v *ast.Identifier) bool {
	return vs.indirectVars[v]
}

// predefVarIndex returns the index of the predefined variable v. If v is not
// available in the global slice then it is added by this call.
func (vs *varStore) predefVarIndex(v *reflect.Value, typ reflect.Type, pkg, name string) int16 {
	currFn := vs.emitter.fb.fn
	if index, ok := vs.predefVarRef[currFn][v]; ok {
		return index
	}
	index := int16(len(vs.globals))
	g := newGlobal(pkg, name, typ, reflect.Value{})
	if v.IsValid() {
		g.Value = *v
	}
	if vs.predefVarRef[currFn] == nil {
		vs.predefVarRef[currFn] = map[*reflect.Value]int16{}
	}
	vs.globals = append(vs.globals, g)
	vs.predefVarRef[currFn][v] = index
	return index
}

func (vs *varStore) setPredefVarRef(fn *runtime.Function, v *reflect.Value, index int16) {
	if vs.predefVarRef[fn] == nil {
		vs.predefVarRef[fn] = map[*reflect.Value]int16{}
	}
	vs.predefVarRef[fn][v] = index
}

// getGlobals returns the slice of all Globals collected during the emission.
func (vs *varStore) getGlobals() []Global {
	return vs.globals
}

func (vs *varStore) nonLocalVarIndex(v ast.Expression) (index int, ok bool) {

	var name string
	var fullName string

	switch v := v.(type) {
	case *ast.Identifier:
		name = v.Name
		fullName = v.Name
	case *ast.Selector:
		switch e := v.Expr.(type) {
		case *ast.Identifier:
			name = v.Ident
			fullName = e.Name + "." + v.Ident
		default:
			return 0, false
		}
	default:
		return 0, false
	}

	ti := vs.emitter.ti(v)
	currFn := vs.emitter.fb.fn
	currPkg := vs.emitter.pkg

	// v is a predefined variable.
	if ti != nil && ti.IsNative() {
		index := vs.predefVarIndex(ti.value.(*reflect.Value), ti.Type, ti.NativePackageName, name)
		return int(index), true
	}

	if index, ok := vs.closureVars[currFn][fullName]; ok {
		return int(index), true
	}
	if index, ok := vs.scriggoPackageVarRefs[currPkg][fullName]; ok {
		return int(index), true
	}
	return 0, false

}
