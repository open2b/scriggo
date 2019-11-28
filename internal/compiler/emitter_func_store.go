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

type functionStore struct {
	emitter *emitter

	availableScriggoFuncs map[*ast.Package]map[string]*runtime.Function
	scriggoFuncIndexes    map[*runtime.Function]map[*runtime.Function]int8

	predefFuncIndexes map[*runtime.Function]map[reflect.Value]int8
}

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

func (fs *functionStore) predefFunc(fn ast.Expression, allowMethod bool) (int8, bool) {
	ti := fs.emitter.ti(fn)
	if (ti == nil) || (!ti.IsPredefined()) {
		return 0, false
	}
	if !allowMethod && ti.MethodType != NoMethod {
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
		switch e := fn.Expr.(type) {
		case *ast.Identifier:
			name = e.Name // TODO: why just name? Where is the dot?
			// default:
			// 	return 0, false
		}
	default:
		// return 0, false
	}

	{
		fn := ti.value.(reflect.Value)
		currFn := fs.emitter.fb.fn
		if fs.predefFuncIndexes[currFn] == nil {
			fs.predefFuncIndexes[currFn] = map[reflect.Value]int8{}
		}
		if index, ok := fs.predefFuncIndexes[currFn][fn]; ok {
			return index, true
		}
		f := newPredefinedFunction(ti.PredefPackageName, name, fn.Interface())
		index := int8(len(currFn.Predefined))
		currFn.Predefined = append(currFn.Predefined, f)
		if fs.predefFuncIndexes[currFn] == nil {
			fs.predefFuncIndexes[currFn] = map[reflect.Value]int8{}
		}
		fs.predefFuncIndexes[currFn][fn] = index
		return index, true
	}

}
