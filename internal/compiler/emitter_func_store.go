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
	emitter               *emitter
	availableScriggoFuncs map[*ast.Package]map[string]*runtime.Function
	indexes               map[*runtime.Function]map[*runtime.Function]int8
	predefFuncIndex       map[*runtime.Function]map[reflect.Value]int8
}

func newFunctionStore(emitter *emitter) *functionStore {
	return &functionStore{
		emitter:               emitter,
		availableScriggoFuncs: map[*ast.Package]map[string]*runtime.Function{},
		indexes:               map[*runtime.Function]map[*runtime.Function]int8{},
		predefFuncIndex:       map[*runtime.Function]map[reflect.Value]int8{},
	}
}

func (fs *functionStore) declareScriggoFn(pkg *ast.Package, name string, fn *runtime.Function) {
	if fs.availableScriggoFuncs[pkg] == nil {
		fs.availableScriggoFuncs[pkg] = map[string]*runtime.Function{}
	}
	fs.availableScriggoFuncs[pkg][name] = fn
}

func (fs *functionStore) isScriggoFn(pkg *ast.Package, name string) bool {
	_, ok := fs.availableScriggoFuncs[pkg][name]
	return ok
}

func (fs *functionStore) getScriggoFn(pkg *ast.Package, name string) *runtime.Function {
	fn, ok := fs.availableScriggoFuncs[pkg][name]
	if !ok {
		// TODO
	}
	return fn
}

func (fs *functionStore) scriggoFnIndex(fun *runtime.Function) int8 {
	currFn := fs.emitter.fb.fn
	if fs.indexes[currFn] == nil {
		fs.indexes[currFn] = map[*runtime.Function]int8{}
	}
	if index, ok := fs.indexes[currFn][fun]; ok {
		return index
	}
	index := int8(len(currFn.Functions))
	currFn.Functions = append(currFn.Functions, fun)
	fs.indexes[currFn][fun] = index
	return index
}

func (fs *functionStore) predefFnIndex(fn reflect.Value, pkg, name string) int8 {
	currFn := fs.emitter.fb.fn
	if fs.predefFuncIndex[currFn] == nil {
		fs.predefFuncIndex[currFn] = map[reflect.Value]int8{}
	}
	if index, ok := fs.predefFuncIndex[currFn][fn]; ok {
		return index
	}
	f := newPredefinedFunction(pkg, name, fn.Interface())
	index := int8(len(currFn.Predefined))
	currFn.Predefined = append(currFn.Predefined, f)
	if fs.predefFuncIndex[currFn] == nil {
		fs.predefFuncIndex[currFn] = map[reflect.Value]int8{}
	}
	fs.predefFuncIndex[currFn][fn] = index
	return index
}
