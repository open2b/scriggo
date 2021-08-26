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

// A functionStore holds information about functions defined in Scriggo and
// predefined functions during the emission.
type functionStore struct {

	// emitter is a reference to the current emitter.
	emitter *emitter

	// availableScriggoFuncs holds a list of Scriggo compiled functions that are
	// available to be called or evaluated as expressions. The functions stored
	// in availableScriggoFuncs are not added nor to Functions or Predefined if
	// they are not used in the Scriggo code.
	availableScriggoFuncs map[*ast.Package]map[string]*runtime.Function

	// scriggoFuncIndexes holds the indexes of the Scriggo functions that have
	// been added to Functions because they are referenced in the Scriggo code.
	scriggoFuncIndexes map[*runtime.Function]map[*runtime.Function]int8

	// predefFuncIndexes holds the indexes of the predefined functions that have
	// been added to Predefined because they are referenced in the Scriggo code.
	predefFuncIndexes map[*runtime.Function]map[reflect.Value]int8
}

// newFunctionStore returns a new functionStore.
func newFunctionStore(emitter *emitter) *functionStore {
	return &functionStore{
		emitter:               emitter,
		availableScriggoFuncs: map[*ast.Package]map[string]*runtime.Function{},
		scriggoFuncIndexes:    map[*runtime.Function]map[*runtime.Function]int8{},
		predefFuncIndexes:     map[*runtime.Function]map[reflect.Value]int8{},
	}
}

// makeAvailableScriggoFn makes available the given function with the given name
// in the pkg package, ensuring that such function can be later retrieved if it
// is referenced in the Scriggo compiled code.
func (fs *functionStore) makeAvailableScriggoFn(pkg *ast.Package, name string, fn *runtime.Function) {
	if fs.availableScriggoFuncs[pkg] == nil {
		fs.availableScriggoFuncs[pkg] = map[string]*runtime.Function{}
	}
	fs.availableScriggoFuncs[pkg][name] = fn
}

// availableScriggoFn returns the Scriggo function with the given name available
// in the pkg package. If not available then false is returned.
func (fs *functionStore) availableScriggoFn(pkg *ast.Package, name string) (*runtime.Function, bool) {
	fn, ok := fs.availableScriggoFuncs[pkg][name]
	return fn, ok
}

// scriggoFnIndex returns the index of the given Scriggo function inside the
// Functions slice of the current function. If fun is not present in such slice
// it is added by this call.
func (fs *functionStore) scriggoFnIndex(fn *runtime.Function) int8 {
	currFn := fs.emitter.fb.fn
	if fs.scriggoFuncIndexes[currFn] == nil {
		fs.scriggoFuncIndexes[currFn] = map[*runtime.Function]int8{}
	}
	if index, ok := fs.scriggoFuncIndexes[currFn][fn]; ok {
		return index
	}
	index := int8(len(currFn.Functions))
	currFn.Functions = append(currFn.Functions, fn)
	fs.scriggoFuncIndexes[currFn][fn] = index
	return index
}

// predefFunc returns the index of the predefined function 'contained' in fn if
// there's one, else returns 0 and false.
func (fs *functionStore) predefFunc(fn ast.Expression, allowMethod bool) (int8, bool) {
	ti := fs.emitter.ti(fn)
	if (ti == nil) || (!ti.IsNative()) {
		return 0, false
	}
	if !allowMethod && ti.MethodType != noMethod {
		return 0, false
	}
	if ti.Type.Kind() != reflect.Func {
		return 0, false
	}
	if ti.Addressable() {
		return 0, false
	}
	var name string
	switch fn := fn.(type) {
	case *ast.Identifier:
		name = fn.Name
	case *ast.Selector:
		switch fn.Expr.(type) {
		case *ast.Identifier:
			name = fn.Ident
		}
	default:
		// return 0, false // TODO
	}

	// Add the function to the Predefined slice, or get the index if already
	// present.
	fnRv := ti.value.(reflect.Value)
	currFn := fs.emitter.fb.fn
	if fs.predefFuncIndexes[currFn] == nil {
		fs.predefFuncIndexes[currFn] = map[reflect.Value]int8{}
	}
	if index, ok := fs.predefFuncIndexes[currFn][fnRv]; ok {
		return index, true
	}
	f := newNativeFunction(ti.NativePackageName, name, fnRv.Interface())
	index := int8(len(currFn.NativeFunctions))
	currFn.NativeFunctions = append(currFn.NativeFunctions, f)
	if fs.predefFuncIndexes[currFn] == nil {
		fs.predefFuncIndexes[currFn] = map[reflect.Value]int8{}
	}
	fs.predefFuncIndexes[currFn][fnRv] = index
	return index, true

}
