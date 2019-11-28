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

type varStore struct {
	emitter      *emitter
	indirectVars map[*ast.Identifier]bool
	predefVarRef map[*runtime.Function]map[*reflect.Value]int16
	// Holds all Scriggo-defined and pre-predefined global variables.
	globals            []Global
	scriggoPackageVars map[*ast.Package]map[string]int16
	closureVars        map[*runtime.Function]map[string]int
}

func newVarStore(emitter *emitter, indirectVars map[*ast.Identifier]bool) *varStore {
	return &varStore{
		emitter:            emitter,
		predefVarRef:       map[*runtime.Function]map[*reflect.Value]int16{},
		indirectVars:       indirectVars,
		scriggoPackageVars: map[*ast.Package]map[string]int16{},
		closureVars:        map[*runtime.Function]map[string]int{},
	}
}

func (vs *varStore) closureVar(fn *runtime.Function, name string) (int, bool) {
	// TODO: is 'reg' ok as a name?
	reg, ok := vs.closureVars[fn][name]
	return reg, ok
}

// TODO: is 'reg' ok as a name?
func (vs *varStore) setClosureVar(fn *runtime.Function, name string, reg int) {
	if vs.closureVars[fn] == nil {
		vs.closureVars[fn] = map[string]int{}
	}
	vs.closureVars[fn][name] = reg
}

func (vs *varStore) addScriggoPackageVar(pkg *ast.Package, name string, index int16) {
	if vs.scriggoPackageVars[pkg] == nil {
		vs.scriggoPackageVars[pkg] = map[string]int16{}
	}
	vs.scriggoPackageVars[pkg][name] = index
}

func (vs *varStore) scriggoPackageVarIndex(pkg *ast.Package, name string) (int16, bool) {
	index, ok := vs.scriggoPackageVars[pkg][name]
	return index, ok
}

func (vs *varStore) mustBeDeclaredAsIndirect(v *ast.Identifier) bool {
	return vs.indirectVars[v]
}

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

func (vs *varStore) isPredefVar(v *reflect.Value) (int16, bool) {
	currFn := vs.emitter.fb.fn
	index, ok := vs.predefVarRef[currFn][v]
	return index, ok
}

func (vs *varStore) addGlobal(g Global) int16 {
	index := int16(len(vs.globals))
	vs.globals = append(vs.globals, g)
	return index
}

func (vs *varStore) getGlobals() []Global {
	return vs.globals
}
