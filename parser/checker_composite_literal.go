// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"math/big"
	"reflect"

	"scrigo/ast"
)

// maxIndex returns the maximum element index in the composite literal node.
func (tc *typechecker) maxIndex(node *ast.CompositeLiteral) int {
	switch node.Type.(type) {
	case *ast.ArrayType, *ast.SliceType:
	default:
		return noEllipses
	}
	maxIndex := noEllipses
	currentIndex := -1
	for _, kv := range node.KeyValues {
		if kv.Key != nil {
			ti := tc.checkExpression(kv.Key)
			if ti.Value == nil {
				panic(tc.errorf(node, "index must be non-negative integer constant"))
			}
			index, err := tc.convert(ti, intType, false)
			if err != nil {
				panic(tc.errorf(node, err.Error()))
			}
			n := int(index.(*big.Int).Int64())
			if n < 0 {
				panic(tc.errorf(node, "index must be non-negative integer constant"))
			}
			currentIndex = n
		} else {
			currentIndex++
		}
		if currentIndex > maxIndex {
			maxIndex = currentIndex
		}
	}
	return maxIndex
}

// checkDuplicatedKeys checks if node contains duplicates keys.
func (tc *typechecker) checkDuplicatedKeys(node *ast.CompositeLiteral, kind reflect.Kind) {
	found := []interface{}{}
	for _, kv := range node.KeyValues {
		if kv.Key == nil {
			continue
		}
		var value interface{}
		if kind == reflect.Struct {
			ident, ok := kv.Key.(*ast.Identifier)
			if !ok {
				panic(tc.errorf(kv.Key, "invalid field name composite literal in struct initializer"))
			}
			value = ident.Name
		} else {
			ti := tc.checkExpression(kv.Key)
			if ti.Value == nil {
				continue
			}
			value = ti.Value
		}
		for _, f := range found {
			areEqual := false
			switch v1 := value.(type) {
			case *big.Int:
				v2 := f.(*big.Int)
				areEqual = v1.Cmp(v2) == 0
			case *big.Float:
				v2 := f.(*big.Float)
				areEqual = v1.Cmp(v2) == 0
			default:
				areEqual = f == value
			}
			if areEqual {
				switch kind {
				case reflect.Struct:
					panic(tc.errorf(node, "duplicate field name in struct literal: %s", kv.Key))
				case reflect.Array, reflect.Slice:
					panic(tc.errorf(node, "duplicate index in array literal: %s", kv.Key))
				case reflect.Map:
					panic(tc.errorf(node, "duplicate key in map literal: %s", kv.Key))
				}
			}
		}
		found = append(found, value)
	}
}

func (tc *typechecker) checkCompositeLiteral(node *ast.CompositeLiteral, explicitType reflect.Type) *ast.TypeInfo {

	var err error

	maxIndex := tc.maxIndex(node)
	ti := tc.checkType(node.Type, maxIndex+1)

	switch ti.Type.Kind() {

	case reflect.Struct:

		explicitFields := false
		declType := 0
		for _, kv := range node.KeyValues {
			if kv.Key == nil {
				if declType == 1 {
					panic(tc.errorf(node, "mixture of field:value and value initializers"))
				}
				declType = -1
				continue
			} else {
				if declType == -1 {
					panic(tc.errorf(node, "mixture of field:value and value initializers"))
				}
				declType = 1
			}
		}
		explicitFields = declType == 1
		if explicitFields { // struct with explicit fields.
			tc.checkDuplicatedKeys(node, reflect.Struct)
			for _, keyValue := range node.KeyValues {
				ident, ok := keyValue.Key.(*ast.Identifier)
				if !ok {
					panic(tc.errorf(node, "invalid field name %s in struct initializer", keyValue.Key))
				}
				fieldTi, ok := ti.Type.FieldByName(ident.Name)
				if !ok {
					panic(tc.errorf(node, "unknown field '%s' in struct literal of type %s", keyValue.Key, ti))
				}
				valueTi := tc.typeof(keyValue.Value, noEllipses)
				if !tc.isAssignableTo(valueTi, fieldTi.Type) {
					panic(tc.errorf(node, "cannot use %v (type %v) as type %v in field value", keyValue.Value, valueTi.ShortString(), ti.Type.Key()))
				}
			}
		} else { // struct with implicit fields.
			if len(node.KeyValues) == 0 {
				// TODO (Gianluca): PropertyAddressable?
				return &ast.TypeInfo{Type: ti.Type, Properties: ast.PropertyAddressable}
			}
			if len(node.KeyValues) < ti.Type.NumField() {
				panic(tc.errorf(node, "too few values in %s literal", node.Type))
			}
			if len(node.KeyValues) > ti.Type.NumField() {
				panic(tc.errorf(node, "too many values in %s literal", node.Type))
			}
			for i, keyValue := range node.KeyValues {
				valueTi := tc.typeof(keyValue.Value, noEllipses)
				fieldTi := ti.Type.Field(i)
				if !tc.isAssignableTo(valueTi, fieldTi.Type) {
					panic(tc.errorf(node, "cannot use %v (type %s) as type %v in field value", keyValue.Value, valueTi.ShortString(), fieldTi.Type))
				}
			}
		}

	case reflect.Array:

		tc.checkDuplicatedKeys(node, reflect.Array)
		for _, kv := range node.KeyValues {
			if kv.Key != nil {
				keyTi := tc.typeof(kv.Key, noEllipses)
				if keyTi.Value == nil {
					panic(tc.errorf(node, "index must be non-negative integer constant"))
				}
			}
			var elemTi *ast.TypeInfo
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				elemTi = tc.checkCompositeLiteral(cl, ti.Type.Elem())
			} else {
				elemTi = tc.typeof(kv.Value, noEllipses)
			}
			if !tc.isAssignableTo(elemTi, ti.Type.Elem()) {
				panic(tc.errorf(node, "cannot convert %v (type %v) to type %v", kv.Value, elemTi, ti.Type.Elem()))
			}
		}

	case reflect.Slice:

		tc.checkDuplicatedKeys(node, reflect.Slice)
		for _, kv := range node.KeyValues {
			if kv.Key != nil {
				keyTi := tc.typeof(kv.Key, noEllipses)
				if keyTi.Value == nil {
					panic(tc.errorf(node, "index must be non-negative integer constant"))
				}
			}
			var elemTi *ast.TypeInfo
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				elemTi = tc.checkCompositeLiteral(cl, ti.Type.Elem())
			} else {
				elemTi = tc.typeof(kv.Value, noEllipses)
			}
			if !tc.isAssignableTo(elemTi, ti.Type.Elem()) {
				panic(tc.errorf(node, "cannot convert %v (type %v) to type %v", kv.Value, elemTi, ti.Type.Elem()))
			}
		}

	case reflect.Map:

		// TODO (Gianluca): checking is done by parser, so it's impossibile to
		// test this. Decomment when parsing duplicates check has been removed.
		//tc.checkDuplicates(node, reflect.Map)
		for _, kv := range node.KeyValues {
			var keyTi *ast.TypeInfo
			if compLit, ok := kv.Value.(*ast.CompositeLiteral); ok {
				keyTi = tc.checkCompositeLiteral(compLit, ti.Type.Key())
			} else {
				keyTi = tc.typeof(kv.Key, noEllipses)
				if err != nil {
					panic(err)
				}
			}
			if !tc.isAssignableTo(keyTi, ti.Type.Key()) {
				panic(tc.errorf(node, "cannot use %s (type %v) as type %v in map key", kv.Key, keyTi.ShortString(), ti.Type.Key()))
			}
			var valueTi *ast.TypeInfo
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				valueTi = tc.checkCompositeLiteral(cl, ti.Type.Elem())
			} else {
				valueTi = tc.typeof(kv.Value, noEllipses)
			}
			if !tc.isAssignableTo(valueTi, ti.Type.Elem()) {
				panic(tc.errorf(node, "cannot use %s (type %v) as type %v in map value", kv.Value, valueTi, ti.Type.Elem()))
			}
		}
	}

	// TODO (Gianluca): are composite literals addressable? In Go Specifications
	// composite literal are handled as "exceptions": « As an exception to the
	// addressability requirement, x may also be a (possibly parenthesized)
	// composite literal.». What do we mean with "addressable"?
	return &ast.TypeInfo{Type: ti.Type, Properties: ast.PropertyAddressable}
}
