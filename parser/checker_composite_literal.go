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
			value := kv.Value
			// TODO (Gianluca):	evaluate value, populating TypeInfo
			constant := value.TypeInfo().Constant
			if constant != nil {
				// TODO get constant value
				// currentIndex = constantValue
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
	typ := tc.typeof(node.Type, noEllipses)
	if !typ.IsType() {
		return nil, tc.errorf(node, "composite literal type is not a type") // TODO (Gianluca): this needs a review.
	}

	switch typ.Type.Kind() {

	case reflect.Struct:

		var explicitFields bool
		{
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
		}

		if explicitFields { // struct with explicit fields.
			for _, keyValue := range node.KeyValues {
				ident, ok := keyValue.Key.(*ast.Identifier)
				if !ok {
					return nil, tc.errorf(node, "invalid field name %s in struct initializer", keyValue.Key)
				}
				fieldCt, ok := typ.Type.FieldByName(ident.Name)
				if !ok {
					return nil, tc.errorf(node, "unknown field '%s' in struct literal of type %s", keyValue.Key, typ)
				}
				valueCt := tc.typeof(keyValue.Value, noEllipses)
				// TODO (Gianluca): review!
				_ = fieldCt
				_ = valueCt
				// err = valueCt.MustAssignableTo(fieldCt.ReflectType())
				// if err != nil {
				// 	return nil, tc.errorf(node, "cannot use %v (type %v) as type %v in field value", keyValue.Value, keyValue.Value, typ.Key())
				// }
			}
		} else { // struct with implicit fields.
			if len(node.KeyValues) != typ.Type.NumField() {
				panic("too many or not enough elements to struct composite literal")
			}
			for i, keyValue := range node.KeyValues {
				valueCt := tc.typeof(keyValue.Value, noEllipses)
				// TODO (Gianluca): review!
				_ = i
				_ = valueCt
				// err = valueCt.MustAssignableTo(typ.ReflectType().Field(i).Type)
				// if err != nil {
				// 	return nil, tc.errorf(node, "cannot use %v (type %v) as type %v in field value", keyValue.Value, valueCt.Type(), typ.ReflectType().Field(i).Type)
				// }
			}
		}

	case reflect.Array:

		var ok bool
		maxIndex := 0
		currentIndex := 0
		for _, keyValue := range node.KeyValues {
			// TODO (Gianluca): review!
			_ = ok
			_ = maxIndex
			_ = currentIndex
			_ = keyValue
			// if keyValue.Key != nil {
			// 	keyType := tc.typeof(keyValue.Key, noEllipses)
			// 	if keyType.Constant != nil {
			// 		return nil, tc.errorf(node, "index must be non-negative integer constant")
			// 	}
			// 	currentIndex, ok = keyType.Value().(int) // TODO (Gianluca): review!
			// 	if !ok {
			// 		return nil, tc.errorf(node, "index must be non-negative integer constant")
			// 	}
			// } else {
			// 	currentIndex++
			// }
			// if currentIndex > maxIndex {
			// 	maxIndex = currentIndex
			// }
			// if maxIndex > typ.ReflectType().Len()-1 {
			// 	return nil, tc.errorf(node, "array index %d out of bounds [0:%d]", maxIndex, typ.ReflectType().Len()-1)
			// }
			// var valueType *checked
			// if compLit, ok := keyValue.Value.(*ast.CompositeLiteral); ok {
			// 	valueType, err = imp.checkCompositeLiteral(compLit, typ.Elem())
			// 	if err != nil {
			// 		panic(err)
			// 	}
			// } else {
			// 	valueType, err = imp.typeof(keyValue.Value)
			// 	if err != nil {
			// 		panic(err)
			// 	}
			// }
			// if err != nil {
			// 	panic(err)
			// }
			// err = valueType.MustAssignableTo(typ.Elem().ReflectType())
			// if err != nil {
			// 	return nil, imp.errorf(node, "cannot convert \"%v\" (type %v) to type %v", keyValue, valueType, typ.Elem())
			// }
		}

	case reflect.Slice:

		// TODO (Gianluca): add support for slice composite literal with
		// explicit keys.

		for _, kv := range node.KeyValues {
			keyTyp := tc.typeof(kv.Key, noEllipses)
			var valueTyp *ast.TypeInfo
			if compLit, ok := kv.Value.(*ast.CompositeLiteral); ok {
				valueTyp, err = tc.checkCompositeLiteral(compLit, keyTyp.Type.Key())
				if err != nil {
					return nil, err
				}
			} else {
				valueTyp = tc.typeof(kv.Value, noEllipses)
			}
			// TODO (Gianluca): review!
			_ = valueTyp
			// err = keyTyp.MustAssignableTo(typ.Type.Elem().ReflectType())
			// if err != nil {
			// 	return nil, tc.errorf(node, "cannot convert %v (type %v) to type %v", kv.Value, valueTyp, typ.Key())
			// }
		}

	case reflect.Map:

		for _, keyValue := range node.KeyValues {
			var keyTyp *ast.TypeInfo
			if compLit, ok := keyValue.Value.(*ast.CompositeLiteral); ok {
				keyTyp, err = tc.checkCompositeLiteral(compLit, typ.Type.Key())
				if err != nil {
					return nil, err
				}
			} else {
				keyTyp = tc.typeof(keyValue.Key, noEllipses)
				if err != nil {
					return nil, err
				}
			}
			// TODO (Gianluca): review!
			// err = keyTyp.MustAssignableTo(typ.Key().ReflectType())
			// if err != nil {
			// 	return nil, tc.errorf(node, "cannot use %v (type %v) as type %v in map key", keyValue.Value, keyTyp.Type(), typ.Key())
			// }
			var valueTyp *ast.TypeInfo
			if compLit, ok := keyValue.Value.(*ast.CompositeLiteral); ok {
				valueTyp, err = tc.checkCompositeLiteral(compLit, typ.Type.Elem())
				if err != nil {
					panic(err)
				}
			} else {
				valueTyp = tc.typeof(keyValue.Value, noEllipses)
			}
			// TODO (Gianluca): review!
			_ = keyTyp
			_ = valueTyp
			// err = valueTyp.MustAssignableTo(typ.Elem().ReflectType())
			// if err != nil {
			// 	return nil, tc.errorf(node, "cannot convert %v (type %v) to type %v", keyValue.Value, keyTyp, typ.Key())
			// }
		}
	}

	return typ, nil
}
