// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"io"
	"io/ioutil"
	"reflect"

	"scrigo/internal/compiler"
	"scrigo/internal/compiler/ast"
	"scrigo/native"
	"scrigo/vm"
)

// Global represents a global variable with a package, name, type (only for
// Scrigo globals) and value (only for native globals). Value, if present,
// must be a pointer to the variable value.
type Global struct {
	Name string
	Type reflect.Type
}

type Script struct {
	Fn      *vm.ScrigoFunction
	globals []Global
}

func Compile(src io.Reader, main *native.GoPackage) (*Script, error) {

	// Parsing.
	buf, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	tree, err := compiler.ParseSource(buf, ast.ContextNone)
	if err != nil {
		return nil, err
	}

	// Type checking.
	pkgInfo, err := typecheck(tree, main)
	if err != nil {
		return nil, err
	}
	tci := map[string]*compiler.PackageInfo{"main": pkgInfo}

	// Emitting.
	// TODO(Gianluca): pass "main" to emitter.
	// main contains user defined variables.
	emitter := compiler.NewEmitter(nil, tci["main"].TypeInfo, tci["main"].IndirectVars)
	fn := compiler.NewScrigoFunction("main", "main", reflect.FuncOf(nil, nil, false))
	emitter.CurrentFunction = fn
	emitter.FB = compiler.NewBuilder(emitter.CurrentFunction)
	emitter.FB.EnterScope()
	compiler.AddExplicitReturn(tree)
	emitter.EmitNodes(tree.Nodes)
	emitter.FB.ExitScope()

	return &Script{Fn: emitter.CurrentFunction}, nil
}

func Execute(script *Script, vars map[string]interface{}) error {
	vmm := vm.New()
	if n := len(script.globals); n > 0 {
		globals := make([]interface{}, n)
		for i, global := range script.globals {
			if value, ok := vars[global.Name]; ok {
				if v, ok := value.(reflect.Value); ok {
					globals[i] = v.Addr().Interface()
				} else {
					rv := reflect.New(global.Type).Elem()
					rv.Set(reflect.ValueOf(v))
					globals[i] = rv.Interface()
				}
			} else {
				globals[i] = reflect.New(global.Type).Interface()
			}
		}
		vmm.SetGlobals(globals)
	}
	_, err := vmm.Run(script.Fn)
	return err
}

func typecheck(tree *ast.Tree, main *native.GoPackage) (_ *compiler.PackageInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(*compiler.Error); ok {
				err = rerr
			} else {
				panic(r)
			}
		}
	}()
	tc := compiler.NewTypechecker(tree.Path, true)
	tc.Universe = compiler.Universe
	if main != nil {
		tc.Scopes = append(tc.Scopes, compiler.ToTypeCheckerScope(main))
	}
	tc.CheckNodesInNewScope(tree.Nodes)
	pkgInfo := &compiler.PackageInfo{}
	pkgInfo.IndirectVars = tc.IndirectVars
	pkgInfo.TypeInfo = tc.TypeInfo
	return pkgInfo, err
}
