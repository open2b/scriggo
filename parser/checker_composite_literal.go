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

	// TODO (Gianluca): use MaxIndex as argument.
	ti := tc.typeof(node.Type, noEllipses)
	if !ti.IsType() {
		return nil, tc.errorf(node, "composite literal type is not a type") // TODO (Gianluca): this needs a review.
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
					return nil, tc.errorf(node, "cannot use %v (type %v) as type %v in field value", keyValue.Value, valueTi.Type, ti.Type.Key())
				}
			}
		} else { // struct with implicit fields.
			if len(node.KeyValues) != ti.Type.NumField() {
				panic("too many or not enough elements to struct composite literal")
			}
			for i, keyValue := range node.KeyValues {
				valueTi := tc.typeof(keyValue.Value, noEllipses)
				fieldTi := ti.Type.Field(i)
				if tc.isAssignableTo(valueTi, fieldTi.Type) {
					return nil, tc.errorf(node, "cannot use %v (type %v) as type %v in field value", keyValue.Value, valueTi.Type, fieldTi.Type)
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

		for _, keyValue := range node.KeyValues {
			var keyTyp *ast.TypeInfo
			if compLit, ok := keyValue.Value.(*ast.CompositeLiteral); ok {
				keyTyp, err = tc.checkCompositeLiteral(compLit, ti.Type.Key())
				if err != nil {
					return nil, err
				}
			} else {
				keyTyp = tc.typeof(keyValue.Key, noEllipses)
				if err != nil {
					return nil, err
				}
			}
			if !tc.isAssignableTo(keyTyp, ti.Type.Key()) {
				return nil, tc.errorf(node, "cannot use %v (type %v) as type %v in map key", keyValue.Value, keyTyp.Type, ti.Type.Key())
			}
			var valueTyp *ast.TypeInfo
			if compLit, ok := keyValue.Value.(*ast.CompositeLiteral); ok {
				valueTyp, err = tc.checkCompositeLiteral(compLit, ti.Type.Elem())
				if err != nil {
					panic(err)
				}
			} else {
				valueTyp = tc.typeof(keyValue.Value, noEllipses)
			}
			if !tc.isAssignableTo(valueTyp, ti.Type.Elem()) {
				return nil, tc.errorf(node, "cannot convert %v (type %v) to type %v", keyValue.Value, valueTyp, ti.Type.Elem())
			}
		}
	}

	return ti, nil
}
