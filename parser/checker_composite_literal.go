// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"reflect"

	"scrigo/ast"
)

// maxIndex returns the maximum element index in the composite literal node.
func (tc *typechecker) maxIndex(node *ast.CompositeLiteral) (int, error) {
	maxIndex := 0
	currentIndex := -1
	for _, kv := range node.KeyValues {
		if kv.Key != nil {
			ti := tc.checkExpression(kv.Key)
			constant := ti.Constant
			if constant != nil {
				// TODO (Gianluca):
				// currentIndex := ti.Constant.Number
			}
		} else {
			currentIndex++
		}
		if currentIndex > maxIndex {
			maxIndex = currentIndex
		}
	}
	return maxIndex, nil
}

func (tc *typechecker) checkCompositeLiteral(node *ast.CompositeLiteral, explicitType reflect.Type) (*ast.TypeInfo, error) {

	maxIndex, err := tc.maxIndex(node)
	if err != nil {
		return nil, err
	}
	_ = maxIndex

	// TODO (Gianluca): use maxIndex as argument.
	ti := tc.typeof(node.Type, noEllipses)
	if !ti.IsType() {
		return nil, tc.errorf(node, "%s is not a type", node.Type)
	}

	switch ti.Type.Kind() {

	case reflect.Struct:

		explicitFields := false
		declType := 0
		for _, kv := range node.KeyValues {
			if kv.Key == nil {
				if declType == 1 {
					return nil, tc.errorf(node, "mixture of field:value and value initializers")
				}
				declType = -1
				continue
			} else {
				if declType == -1 {
					return nil, tc.errorf(node, "mixture of field:value and value initializers")
				}
				declType = 1
			}
		}
		explicitFields = declType == 1
		if explicitFields { // struct with explicit fields.
			for _, keyValue := range node.KeyValues {
				ident, ok := keyValue.Key.(*ast.Identifier)
				if !ok {
					return nil, tc.errorf(node, "invalid field name %s in struct initializer", keyValue.Key)
				}
				fieldTi, ok := ti.Type.FieldByName(ident.Name)
				if !ok {
					return nil, tc.errorf(node, "unknown field '%s' in struct literal of type %s", keyValue.Key, ti)
				}
				valueTi := tc.typeof(keyValue.Value, noEllipses)
				if !tc.isAssignableTo(valueTi, fieldTi.Type) {
					return nil, tc.errorf(node, "cannot use %v (type %v) as type %v in field value", keyValue.Value, valueTi.ShortString(), ti.Type.Key())
				}
			}
		} else { // struct with implicit fields.
			if len(node.KeyValues) == 0 {
				return ti, nil
			}
			if len(node.KeyValues) < ti.Type.NumField() {
				return nil, tc.errorf(node, "too few values in %s literal", node.Type)
			}
			if len(node.KeyValues) > ti.Type.NumField() {
				return nil, tc.errorf(node, "too many values in %s literal", node.Type)
			}
			for i, keyValue := range node.KeyValues {
				valueTi := tc.typeof(keyValue.Value, noEllipses)
				fieldTi := ti.Type.Field(i)
				if tc.isAssignableTo(valueTi, fieldTi.Type) {
					return nil, tc.errorf(node, "cannot use %v (type %s) as type %v in field value", keyValue.Value, valueTi.ShortString(), fieldTi.Type)
				}
			}
		}

	case reflect.Array:

		if maxIndex > ti.Type.Len()-1 {
			return nil, tc.errorf(node, "array index %d out of bounds [0:%d]", maxIndex, ti.Type.Len()-1)
		}
		for _, kv := range node.KeyValues {
			if kv.Key != nil {
				keyTi := tc.typeof(kv.Key, noEllipses)
				if keyTi.Constant == nil {
					panic(tc.errorf(node, "index must be non-negative integer constant"))
				}
			}
			var elemTi *ast.TypeInfo
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				elemTi, err = tc.checkCompositeLiteral(cl, ti.Type.Elem())
				if err != nil {
					return nil, err
				}
			} else {
				elemTi = tc.typeof(kv.Value, noEllipses)
			}
			if !tc.isAssignableTo(elemTi, ti.Type.Elem()) {
				return nil, tc.errorf(node, "cannot convert %v (type %v) to type %v", kv.Value, elemTi, ti.Type.Elem())
			}
		}

	case reflect.Slice:

		for _, kv := range node.KeyValues {
			if kv.Key != nil {
				keyTi := tc.typeof(kv.Key, noEllipses)
				if keyTi.Constant == nil {
					panic(tc.errorf(node, "index must be non-negative integer constant"))
				}
			}
			var elemTi *ast.TypeInfo
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				elemTi, err = tc.checkCompositeLiteral(cl, ti.Type.Elem())
				if err != nil {
					return nil, err
				}
			} else {
				elemTi = tc.typeof(kv.Value, noEllipses)
			}
			if !tc.isAssignableTo(elemTi, ti.Type.Elem()) {
				return nil, tc.errorf(node, "cannot convert %v (type %v) to type %v", kv.Value, elemTi, ti.Type.Elem())
			}
		}

	case reflect.Map:

		for _, kv := range node.KeyValues {
			var keyTi *ast.TypeInfo
			if compLit, ok := kv.Value.(*ast.CompositeLiteral); ok {
				keyTi, err = tc.checkCompositeLiteral(compLit, ti.Type.Key())
				if err != nil {
					return nil, err
				}
			} else {
				keyTi = tc.typeof(kv.Key, noEllipses)
				if err != nil {
					return nil, err
				}
			}
			if !tc.isAssignableTo(keyTi, ti.Type.Key()) {
				return nil, tc.errorf(node, "cannot use %s (type %v) as type %v in map key", kv.Key, keyTi.ShortString(), ti.Type.Key())
			}
			var valueTi *ast.TypeInfo
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				valueTi, err = tc.checkCompositeLiteral(cl, ti.Type.Elem())
				if err != nil {
					panic(err)
				}
			} else {
				valueTi = tc.typeof(kv.Value, noEllipses)
			}
			if !tc.isAssignableTo(valueTi, ti.Type.Elem()) {
				return nil, tc.errorf(node, "cannot use %s (type %v) as type %v in map value", kv.Value, valueTi, ti.Type.Elem())
			}
		}
	}

	return ti, nil
}
