// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/runtime"
)

// A functionStore holds information about native and non-native functions
// during the emission.
type functionStore struct {

	// emitter is a reference to the current emitter.
	emitter *emitter

	// availableFunctions holds a list of non-native compiled functions that
	// are available to be called or evaluated as expressions. The functions
	// stored in availableFunctions are not added nor to Functions or
	// NativeFunctions if they are not used in the source code.
	availableFunctions map[*ast.Package]map[string]*runtime.Function

	// functionIndexes holds the indexes of the non-native functions that have
	// been added to Functions because they are referenced in the source code.
	functionIndexes map[*runtime.Function]map[*runtime.Function]int8

	// nativeFunctionIndexes holds the indexes of the native functions that
	// have been added to NativeFunctions because they are referenced in the
	// source code.
	nativeFunctionIndexes map[*runtime.Function]map[reflect.Value]int8
}

// newFunctionStore returns a new functionStore.
func newFunctionStore(emitter *emitter) *functionStore {
	return &functionStore{
		emitter:               emitter,
		availableFunctions:    map[*ast.Package]map[string]*runtime.Function{},
		functionIndexes:       map[*runtime.Function]map[*runtime.Function]int8{},
		nativeFunctionIndexes: map[*runtime.Function]map[reflect.Value]int8{},
	}
}

// makeAvailableFunction makes available the given non-native function with
// the given name in the pkg package, ensuring that such function can be later
// retrieved if it is referenced in the source code.
func (fs *functionStore) makeAvailableFunction(pkg *ast.Package, name string, fn *runtime.Function) {
	if fs.availableFunctions[pkg] == nil {
		fs.availableFunctions[pkg] = map[string]*runtime.Function{}
	}
	fs.availableFunctions[pkg][name] = fn
}

// availableFunction returns the non-native function with the given name
// available in the pkg package. If not available then false is returned.
func (fs *functionStore) availableFunction(pkg *ast.Package, name string) (*runtime.Function, bool) {
	fn, ok := fs.availableFunctions[pkg][name]
	return fn, ok
}

// functionIndex returns the index of the given non-native function inside the
// Functions slice of the current function. If fn is not present in such slice
// it is added by this call.
func (fs *functionStore) functionIndex(fn *runtime.Function) int8 {
	currFn := fs.emitter.fb.fn
	if fs.functionIndexes[currFn] == nil {
		fs.functionIndexes[currFn] = map[*runtime.Function]int8{}
	}
	if index, ok := fs.functionIndexes[currFn][fn]; ok {
		return index
	}
	index := int8(len(currFn.Functions))
	currFn.Functions = append(currFn.Functions, fn)
	fs.functionIndexes[currFn][fn] = index
	return index
}

// nativeFunction returns the index of the native function 'contained' in fn
// if there's one, else returns 0 and false.
func (fs *functionStore) nativeFunction(fn ast.Expression, allowMethod bool) (int8, bool) {
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

	// Add the function to the NativeFunctions slice, or get the index if
	// already present.
	fnRv := ti.value.(reflect.Value)
	currFn := fs.emitter.fb.fn
	if fs.nativeFunctionIndexes[currFn] == nil {
		fs.nativeFunctionIndexes[currFn] = map[reflect.Value]int8{}
	}
	if index, ok := fs.nativeFunctionIndexes[currFn][fnRv]; ok {
		return index, true
	}
	f := newNativeFunction(ti.NativePackageName, name, fnRv.Interface())
	index := int8(len(currFn.NativeFunctions))
	currFn.NativeFunctions = append(currFn.NativeFunctions, f)
	if fs.nativeFunctionIndexes[currFn] == nil {
		fs.nativeFunctionIndexes[currFn] = map[reflect.Value]int8{}
	}
	fs.nativeFunctionIndexes[currFn][fnRv] = index
	return index, true

}
