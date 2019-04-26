// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"reflect"

	"scrigo/ast"
)

// kindToType returns VM's type of k.
func kindToType(k reflect.Kind) Type {
	switch k {
	case reflect.Int,
		reflect.Bool,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Int8,
		reflect.Uint,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uint8,
		reflect.Uintptr:
		return TypeInt
	case reflect.Float64, reflect.Float32:
		return TypeFloat
	case reflect.Invalid:
		panic("unexpected")
	case reflect.String:
		return TypeString
	case reflect.Complex64,
		reflect.Complex128,
		reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Ptr,
		reflect.Slice,
		reflect.Struct:
		return TypeIface
	case reflect.UnsafePointer:
		panic("TODO: not implemented")
	default:
		panic("bug")
	}
}

// isNil indicates if expr is the nil identifier.
func isNil(expr ast.Expression) bool {
	ident, ok := expr.(*ast.Identifier)
	if !ok {
		return false
	}
	return ident.Name == "nil"
}

// isBlankIdentifier indicates if expr is an identifier representing the blank
// identifier "_".
func isBlankIdentifier(expr ast.Expression) bool {
	ident, ok := expr.(*ast.Identifier)
	return ok && ident.Name == "_"
}

// fillParametersTypes takes a list of parameters (function arguments or
// function return values) and "fills" their types. For instance, a function
// arguments signature "a, b int" becomes "a int, b int".
func fillParametersTypes(params []*ast.Field) {
	if len(params) == 0 {
		return
	}
	typ := params[len(params)-1].Type
	for i := len(params) - 1; i >= 0; i-- {
		if params[i].Type != nil {
			typ = params[i].Type
		}
		params[i].Type = typ
	}
}

// addExplicitReturn adds an explicit return statement as last statement to fun
// if it is implicit.
func addExplicitReturn(fun *ast.Func) {
	var pos *ast.Position
	if len(fun.Body.Nodes) == 0 {
		pos = fun.Pos()
	} else {
		last := fun.Body.Nodes[len(fun.Body.Nodes)-1]
		if _, ok := last.(*ast.Return); ok {
			pos = last.Pos()
		}
	}
	if pos != nil {
		ret := ast.NewReturn(pos, nil)
		fun.Body.Nodes = append(fun.Body.Nodes, ret)
	}
}

// size returns node's size.
func size(node *ast.CompositeLiteral) int {
	size := 0
	for _, kv := range node.KeyValues {
		if kv.Key != nil {
			key := kv.Key.(*ast.Value).Val.(int)
			if key > size {
				size = key
			}
		}
		size++
	}
	return size
}
