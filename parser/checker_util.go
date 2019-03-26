// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"

	"scrigo/ast"
)

var errTypeConversion = errors.New("failed type conversion")

const (
	maxInt   = int(maxUint >> 1)
	minInt   = -maxInt - 1
	maxUint  = ^uint(0)
	minInt64 = -1 << 63
)

var integerRanges = [...]struct{ min, max *big.Int }{
	{big.NewInt(int64(minInt)), big.NewInt(int64(maxInt))}, // int
	{big.NewInt(-1 << 7), big.NewInt(1<<7 - 1)},            // int8
	{big.NewInt(-1 << 15), big.NewInt(1<<15 - 1)},          // int16
	{big.NewInt(-1 << 31), big.NewInt(1<<31 - 1)},          // int32
	{big.NewInt(-1 << 63), big.NewInt(1<<63 - 1)},          // int64
	{nil, big.NewInt(0).SetUint64(uint64(maxUint))},        // uint
	{nil, big.NewInt(1<<8 - 1)},                            // uint8
	{nil, big.NewInt(1<<16 - 1)},                           // uint16
	{nil, big.NewInt(1<<32 - 1)},                           // uint32
	{nil, big.NewInt(0).SetUint64(1<<64 - 1)},              // uint64
}

var integerKind = [...]bool{
	reflect.Int:           true,
	reflect.Int8:          true,
	reflect.Int16:         true,
	reflect.Int32:         true,
	reflect.Int64:         true,
	reflect.Uint:          true,
	reflect.Uint8:         true,
	reflect.Uint16:        true,
	reflect.Uint32:        true,
	reflect.Uint64:        true,
	reflect.UnsafePointer: false,
}

var numericKind = [...]bool{
	reflect.Int:           true,
	reflect.Int8:          true,
	reflect.Int16:         true,
	reflect.Int32:         true,
	reflect.Int64:         true,
	reflect.Uint:          true,
	reflect.Uint8:         true,
	reflect.Uint16:        true,
	reflect.Uint32:        true,
	reflect.Uint64:        true,
	reflect.Float32:       true,
	reflect.Float64:       true,
	reflect.Complex64:     true,
	reflect.Complex128:    true,
	reflect.UnsafePointer: false,
}

var boolOperators = [15]bool{
	ast.OperatorEqual:    true,
	ast.OperatorNotEqual: true,
	ast.OperatorAnd:      true,
	ast.OperatorOr:       true,
}

var intOperators = [15]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessOrEqual:    true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterOrEqual: true,
	ast.OperatorAddition:       true,
	ast.OperatorSubtraction:    true,
	ast.OperatorMultiplication: true,
	ast.OperatorDivision:       true,
	ast.OperatorModulo:         true,
}

var floatOperators = [15]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessOrEqual:    true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterOrEqual: true,
	ast.OperatorAddition:       true,
	ast.OperatorSubtraction:    true,
	ast.OperatorMultiplication: true,
	ast.OperatorDivision:       true,
}

var stringOperators = [15]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessOrEqual:    true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterOrEqual: true,
	ast.OperatorAddition:       true,
}

var interfaceOperators = [15]bool{
	ast.OperatorEqual:    true,
	ast.OperatorNotEqual: true,
}

var operatorsOfKind = [...][15]bool{
	reflect.Bool:      boolOperators,
	reflect.Int:       intOperators,
	reflect.Int8:      intOperators,
	reflect.Int16:     intOperators,
	reflect.Int32:     intOperators,
	reflect.Int64:     intOperators,
	reflect.Uint:      intOperators,
	reflect.Uint8:     intOperators,
	reflect.Uint16:    intOperators,
	reflect.Uint32:    intOperators,
	reflect.Uint64:    intOperators,
	reflect.Float32:   floatOperators,
	reflect.Float64:   floatOperators,
	reflect.String:    stringOperators,
	reflect.Interface: interfaceOperators,
}

// containsDuplicates returns true if slice contains at least one duplicate
// string.
func containsDuplicates(slice []string) bool {
	for _, a := range slice {
		count := 0
		for _, b := range slice {
			if a == b {
				count++
			}
		}
		if count != 1 {
			return true
		}
	}
	return false
}

// convert converts a value explicitly. If the converted value is a constant,
// convert returns its value, otherwise returns nil.
//
// If the value can not be converted, returns an errTypeConversion type error,
// errConstantTruncated or errConstantOverflow.
func convert(ti *TypeInfo, t2 reflect.Type) (interface{}, error) {

	t := ti.Type
	v := ti.Value
	k2 := t2.Kind()

	if ti.Nil() {
		switch t2.Kind() {
		case reflect.Ptr, reflect.Func, reflect.Slice, reflect.Map, reflect.Chan, reflect.Interface:
			return nil, nil
		}
		return nil, errTypeConversion
	}

	if ti.IsConstant() && k2 != reflect.Interface {
		k1 := t.Kind()
		if k2 == reflect.String && reflect.Int <= k1 && k1 <= reflect.Uint64 {
			// As a special case, an integer constant can be explicitly
			// converted to a string type.
			switch v := v.(type) {
			case *big.Int:
				if v.IsInt64() {
					return string(v.Int64()), nil
				}
			}
			return "\uFFFD", nil
		} else if k2 == reflect.Slice && k1 == reflect.String {
			// As a special case, a string constant can be explicitly converted
			// to a slice of runes or bytes.
			if elem := t2.Elem(); elem == uint8Type || elem == int32Type {
				return nil, nil
			}
		}
		return representedBy(ti, t2)
	}

	if t.ConvertibleTo(t2) {
		return nil, nil
	}

	return nil, errTypeConversion
}

// convertImplicit converts implicitly a value. If the converted value is a
// constant, convert returns its value, otherwise returns nil.
//
// If the value can not be converted, returns an errTypeConversion type error,
// errConstantTruncated or errConstantOverflow.
func convertImplicit(ti *TypeInfo, t2 reflect.Type) (interface{}, error) {

	t := ti.Type
	k2 := t2.Kind()

	if ti.Nil() {
		switch t2.Kind() {
		case reflect.Ptr, reflect.Func, reflect.Slice, reflect.Map, reflect.Chan, reflect.Interface:
			return nil, nil
		}
		return nil, errTypeConversion
	}

	if ti.IsConstant() && k2 != reflect.Interface {
		return representedBy(ti, t2)
	}

	if t.ConvertibleTo(t2) {
		return nil, nil
	}

	return nil, errTypeConversion
}

// emptyMethodSet reports whether an interface has an empty method set.
func emptyMethodSet(iface reflect.Type) bool {
	return iface == emptyInterfaceType || boolType.Implements(iface)
}

// fieldByName returns the struct field with the given name and a boolean
// indicating if the field was found.
func fieldByName(t *TypeInfo, name string) (*TypeInfo, bool) {
	if t.Type.Kind() == reflect.Struct {
		field, ok := t.Type.FieldByName(name)
		if ok {
			return &TypeInfo{Type: field.Type}, true
		}
	}
	if t.Type.Kind() == reflect.Ptr {
		field, ok := t.Type.Elem().FieldByName(name)
		if ok {
			return &TypeInfo{Type: field.Type}, true
		}
	}
	return nil, false
}

// fillParametersTypes takes a list of parameters (function arguments or
// function return values) and "fills" their types. For instance, a function
// arguments signature "a, b int" becocmes "a int, b int".
func fillParametersTypes(params []*ast.Field) []*ast.Field {
	if len(params) == 0 {
		return nil
	}
	filled := make([]*ast.Field, len(params))
	typ := params[len(params)-1].Type
	for i := len(params) - 1; i >= 0; i-- {
		if params[i].Type != nil {
			typ = params[i].Type
		}
		filled[i] = ast.NewField(params[i].Ident, typ)
	}
	return filled
}

// isAssignableTo reports whether x is assignable to type t.
// See https://golang.org/ref/spec#Assignability for details.
func isAssignableTo(x *TypeInfo, t reflect.Type) bool {
	if x.Type == t {
		return true
	}
	if x.Nil() {
		switch t.Kind() {
		case reflect.Ptr, reflect.Func, reflect.Slice, reflect.Map, reflect.Chan, reflect.Interface:
			return true
		}
		return false
	}
	k := t.Kind()
	if x.Untyped() {
		if k == reflect.Interface {
			return emptyMethodSet(t)
		}
		_, err := representedBy(x, t)
		return err == nil
	}
	if k == reflect.Interface && x.Type.Implements(t) {
		return true
	}
	// Checks if the type of x and t have identical underlying types and at
	// least one is not a defined type.
	return x.Type.AssignableTo(t)
}

// isBlankIdentifier indicates if expr is an identifier representing the blank
// identifier "_".
func isBlankIdentifier(expr ast.Expression) bool {
	ident, ok := expr.(*ast.Identifier)
	return ok && ident.Name == "_"
}

// isComparison reports whether op is a comparison operator.
func isComparison(op ast.OperatorType) bool {
	return op >= ast.OperatorEqual && op <= ast.OperatorGreaterOrEqual
}

// isOrdered reports whether t is ordered.
func isOrdered(t *TypeInfo) bool {
	k := t.Type.Kind()
	return numericKind[k] || k == reflect.String
}

// methodByName returns a function type that describe the method with that
// name and a boolean indicating if the method was found.
//
// Only for type classes, the returned function type has the method's
// receiver as first argument.
func methodByName(t *TypeInfo, name string) (*TypeInfo, bool) {
	if t.IsType() {
		if method, ok := t.Type.MethodByName(name); ok {
			return &TypeInfo{Type: method.Type}, true
		}
		return nil, false
	}
	// If t represents and interface, Show.MethodByName cannot be called on
	// it's zero value (it would panic); so, in case of interfaces, the
	// Type.MethodByName method is used.
	if t.Type.Kind() == reflect.Interface {
		method, ok := t.Type.MethodByName(name)
		if ok {
			return &TypeInfo{Type: method.Type}, true
		}
		if t.Type.Kind() != reflect.Ptr {
			method, ok := reflect.PtrTo(t.Type).MethodByName(name)
			if ok {
				return &TypeInfo{Type: method.Type}, true
			}
		}
		return nil, false
	}
	method := reflect.Zero(t.Type).MethodByName(name)
	if method.IsValid() {
		return &TypeInfo{Type: method.Type()}, true
	}
	if t.Type.Kind() != reflect.Ptr {
		method = reflect.Zero(reflect.PtrTo(t.Type)).MethodByName(name)
		if method.IsValid() {
			return &TypeInfo{Type: method.Type()}, true
		}
	}
	return nil, false
}

// representedBy returns a constant value represented as a value of type t2.
func representedBy(t1 *TypeInfo, t2 reflect.Type) (interface{}, error) {

	if t1.Untyped() && t2 == emptyInterfaceType {
		if !t1.IsConstant() {
			return t1.Value, nil
		}
		t2 = t1.Type
	}

	v := t1.Value
	k := t2.Kind()

	switch v := v.(type) {
	case bool:
		if k == reflect.Bool {
			return v, nil
		}
	case string:
		if k == reflect.String {
			return v, nil
		}
	case *big.Int:
		switch {
		case reflect.Int <= k && k <= reflect.Uint64:
			if t1.Untyped() || k != t1.Type.Kind() {
				min := integerRanges[k-2].min
				max := integerRanges[k-2].max
				if min == nil && v.Sign() < 0 || min != nil && v.Cmp(min) < 0 || v.Cmp(max) > 0 {
					return nil, fmt.Errorf("constant %v overflows %s", v, t2)
				}
			}
			return v, nil
		case k == reflect.Float64:
			n := (&big.Float{}).SetInt(v)
			if t1.Untyped() && !v.IsInt64() && !v.IsUint64() {
				if _, acc := n.Float64(); acc != big.Exact {
					return nil, fmt.Errorf("constant %v overflows %s", v, t2)
				}
			}
			return n, nil
		case k == reflect.Float32:
			n := (&big.Float{}).SetInt(v).SetPrec(24)
			if t1.Untyped() && !v.IsInt64() && !v.IsUint64() {
				if _, acc := n.Float32(); acc != big.Exact {
					return nil, fmt.Errorf("constant %v overflows %s", v, t2)
				}
			}
			return n, nil
		}
	case *big.Float:
		switch {
		case reflect.Int <= k && k <= reflect.Uint64:
			if n, acc := v.Int(nil); acc == big.Exact {
				min := integerRanges[k-2].min
				max := integerRanges[k-2].max
				if (min != nil || v.Sign() >= 0) && (min == nil || n.Cmp(min) >= 0) && n.Cmp(max) <= 0 {
					return n, nil
				}
			}
			return nil, fmt.Errorf("constant %v truncated to integer", v)
		case k == reflect.Float64:
			if t1.Untyped() {
				n, _ := v.Float64()
				if math.IsInf(n, 0) {
					return nil, fmt.Errorf("constant %v overflows %s", v, t2)
				}
				v = big.NewFloat(n)
			}
			return v, nil
		case k == reflect.Float32:
			n, _ := v.Float32()
			if math.IsInf(float64(n), 0) {
				return nil, fmt.Errorf("constant %v overflows %s", v, t2)
			}
			return big.NewFloat(float64(n)), nil
		}
	case *big.Rat:
		switch {
		case reflect.Int <= k && k <= reflect.Uint64:
			if v.IsInt() {
				n := v.Num()
				min := integerRanges[k-2].min
				max := integerRanges[k-2].max
				if (min != nil || v.Sign() >= 0) && (min == nil || n.Cmp(min) >= 0) && n.Cmp(max) <= 0 {
					return n, nil
				}
			}
			return nil, fmt.Errorf("constant %v truncated to integer", v)
		case k == reflect.Float64:
			n, _ := v.Float64()
			if math.IsInf(n, 0) {
				return nil, fmt.Errorf("constant %v overflows %s", v, t2)
			}
			return big.NewFloat(n), nil
		case k == reflect.Float32:
			n, _ := v.Float32()
			if math.IsInf(float64(n), 0) {
				return nil, fmt.Errorf("constant %v overflows %s", v, t2)
			}
			return big.NewFloat(float64(n)), nil
		}
	}

	return nil, fmt.Errorf("cannot convert %v (type %s) to type %s", v, t1, t2)
}

// stringInSlice indicates if slice contains s.
func sliceContainsString(slice []string, s string) bool {
	for _, ts := range slice {
		if s == ts {
			return true
		}
	}
	return false
}

// tBinaryOp executes a binary expression where the operands are typed
// constants and returns its result. Returns an error if the operation can not
// be executed.
func tBinaryOp(t1 *TypeInfo, expr *ast.BinaryOperator, t2 *TypeInfo) (*TypeInfo, error) {

	if t1.Type != t2.Type {
		return nil, fmt.Errorf("invalid operation: %v (mismatched types %s and %s)", expr, t1, t2)
	}

	t, err := binaryOp(t1, expr, t2)
	if err != nil {
		return nil, err
	}
	if t.Type == nil {
		t.Type = t1.Type
	}

	v := t.Value
	k := t1.Type.Kind()

	t.Value = nil

	switch v := v.(type) {
	case bool, string:
		t.Value = v
	case int64:
		var overflow bool
		switch k {
		case reflect.Int:
			overflow = v != int64(int(v))
		case reflect.Int32:
			overflow = v != int64(int32(v))
		case reflect.Int16:
			overflow = v != int64(int16(v))
		case reflect.Int8:
			overflow = v != int64(int8(v))
		case reflect.Uint:
			overflow = v != int64(uint(v))
		case reflect.Uint64:
			overflow = v != int64(uint64(v))
		case reflect.Uint32:
			overflow = v != int64(uint32(v))
		case reflect.Uint16:
			overflow = v != int64(uint16(v))
		case reflect.Uint8:
			overflow = v != int64(uint8(v))
		}
		if !overflow {
			t.Value = v
		}
	case *big.Int:
		min := integerRanges[k-2].min
		max := integerRanges[k-2].max
		if (min != nil || v.Sign() >= 0) && (min == nil || v.Cmp(min) >= 0) && v.Cmp(max) <= 0 {
			t.Value = v
		}
	case float64:
		if k == reflect.Float64 || (k == reflect.Float32 && v != float64(float32(v))) {
			t.Value = v
		}
	case *big.Float:
		switch {
		case reflect.Int <= k && k <= reflect.Uint64:
			if n, acc := v.Int(nil); acc == big.Exact {
				min := integerRanges[k-2].min
				max := integerRanges[k-2].max
				if (min != nil || v.Sign() >= 0) && (min == nil || n.Cmp(min) >= 0) && n.Cmp(max) <= 0 {
					t.Value = n
				}
			}
		case k == reflect.Float64:
			if n, _ := v.Float64(); !math.IsInf(n, 0) {
				t.Value = n
			}
		case k == reflect.Float32:
			if n, _ := v.Float32(); !math.IsInf(float64(n), 0) {
				t.Value = n
			}
		}
	case *big.Rat:
		switch {
		case reflect.Int <= k && k <= reflect.Uint64:
			if v.IsInt() {
				n := v.Num()
				min := integerRanges[k-2].min
				max := integerRanges[k-2].max
				if (min != nil || v.Sign() >= 0) && (min == nil || n.Cmp(min) >= 0) && n.Cmp(max) <= 0 {
					t.Value = n
				}
			}
		case k == reflect.Float64:
			if n, _ := v.Float64(); !math.IsInf(n, 0) {
				t.Value = n
			}
		case k == reflect.Float32:
			if n, _ := v.Float32(); !math.IsInf(float64(n), 0) {
				t.Value = n
			}
		}
	}

	if t.Value == nil {
		return nil, fmt.Errorf("constant %v overflows %s", v, t1)
	}

	return t, nil
}

// toSameType returns v1 and v2 with the same types.
func toSameType(v1, v2 interface{}) (interface{}, interface{}) {
	switch n1 := v1.(type) {
	case int64:
		switch n2 := v2.(type) {
		case *big.Int:
			v1 = (&big.Int{}).SetInt64(n1)
		case float64:
			if int64(float64(n1)) == n1 {
				n2 = float64(n2)
			} else {
				v1 = (&big.Float{}).SetInt64(n1)
				v2 = (&big.Float{}).SetFloat64(n2)
			}
		case *big.Float:
			v1 = (&big.Float{}).SetInt64(n1)
		case *big.Rat:
			v1 = (&big.Rat{}).SetInt64(n1)
		}
	case *big.Int:
		switch n2 := v2.(type) {
		case int64:
			v2 = (&big.Int{}).SetInt64(n2)
		case float64:
			v1 = (&big.Float{}).SetInt(n1)
			v2 = (&big.Float{}).SetFloat64(n2)
		case *big.Float:
			v1 = (&big.Float{}).SetInt(n1)
		case *big.Rat:
			v1 = (&big.Rat{}).SetInt(n1)
		}
	case float64:
		switch n2 := v2.(type) {
		case int64:
			if int64(float64(n2)) == n2 {
				v2 = float64(n2)
			} else {
				v1 = (&big.Float{}).SetFloat64(n1)
				v2 = (&big.Float{}).SetInt64(n2)
			}
		case *big.Int:
			v1 = (&big.Float{}).SetFloat64(n1)
			v2 = (&big.Float{}).SetInt(n2)
		case *big.Float:
			v1 = (&big.Float{}).SetFloat64(n1)
		case *big.Rat:
			v1 = (&big.Rat{}).SetFloat64(n1)
		}
	case *big.Float:
		switch n2 := v2.(type) {
		case int64:
			v2 = (&big.Float{}).SetInt64(n2)
		case float64:
			v2 = (&big.Float{}).SetFloat64(n2)
		case *big.Int:
			v2 = (&big.Float{}).SetInt(n2)
		case *big.Rat:
			num := (&big.Float{}).SetInt(n2.Num())
			den := (&big.Float{}).SetInt(n2.Denom())
			v2 = num.Quo(num, den)
		}
	case *big.Rat:
		switch n2 := v2.(type) {
		case int64:
			v2 = (&big.Rat{}).SetInt64(n2)
		case float64:
			v2 = (&big.Rat{}).SetFloat64(n2)
		case *big.Int:
			v2 = (&big.Rat{}).SetInt(n2)
		case *big.Float:
			num := (&big.Float{}).SetInt(n1.Num())
			den := (&big.Float{}).SetInt(n1.Denom())
			v1 = num.Quo(num, den)
		}
	}
	return v1, v2
}

// uBinaryOp executes a binary expression where the operands are untyped and
// returns its result. Returns an error if the operation can not be executed.
func uBinaryOp(t1 *TypeInfo, expr *ast.BinaryOperator, t2 *TypeInfo) (*TypeInfo, error) {

	k1 := t1.Type.Kind()
	k2 := t2.Type.Kind()

	if !(k1 == k2 || numericKind[k1] && numericKind[k2]) {
		return nil, fmt.Errorf("invalid operation: %v (mismatched types %s and %s)",
			expr, t1.ShortString(), t2.ShortString())
	}

	t, err := binaryOp(t1, expr, t2)
	if err != nil {
		return nil, err
	}

	if t.Type == nil {
		t.Type = t1.Type
		if k2 > k1 {
			t.Type = t2.Type
		}
	}
	t.Properties |= PropertyUntyped

	return t, nil
}

// binaryOp executes a binary expression where the operands are constants and
// returns its result. Returns an error if the operation can not be executed.
func binaryOp(t1 *TypeInfo, expr *ast.BinaryOperator, t2 *TypeInfo) (*TypeInfo, error) {

	k1 := t1.Type.Kind()

	t := &TypeInfo{Properties: PropertyIsConstant}

	switch k1 {
	case reflect.Bool:
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = t1.Value.(bool) == t2.Value.(bool)
		case ast.OperatorNotEqual:
			t.Value = t1.Value.(bool) != t2.Value.(bool)
		case ast.OperatorAnd:
			t.Value = t1.Value.(bool) && t2.Value.(bool)
		case ast.OperatorOr:
			t.Value = t1.Value.(bool) || t2.Value.(bool)
		default:
			return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t1.ShortString())
		}
		t.Type = boolType
	case reflect.String:
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = t1.Value.(string) == t2.Value.(string)
			t.Type = boolType
		case ast.OperatorNotEqual:
			t.Value = t1.Value.(string) != t2.Value.(string)
			t.Type = boolType
		case ast.OperatorAddition:
			t.Value = t1.Value.(string) + t2.Value.(string)
			t.Type = stringType
		default:
			return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t1.ShortString())
		}
	default:
		v1, v2 := toSameType(t1.Value, t2.Value)
		switch expr.Op {
		default:
			var cmp int
			switch v1 := v1.(type) {
			case int64:
				if v2 := v2.(int64); v1 < v2 {
					cmp = -1
				} else if v1 > v2 {
					cmp = 1
				}
			case *big.Int:
				cmp = v1.Cmp(v2.(*big.Int))
			case float64:
				if v2 := v2.(float64); v1 < v2 {
					cmp = -1
				} else if v1 > v2 {
					cmp = 1
				}
			case *big.Float:
				cmp = v1.Cmp(v2.(*big.Float))
			case *big.Rat:
				cmp = v1.Cmp(v2.(*big.Rat))
			}
			switch expr.Op {
			case ast.OperatorEqual:
				t.Value = cmp == 0
			case ast.OperatorNotEqual:
				t.Value = cmp != 0
			case ast.OperatorLess:
				t.Value = cmp < 0
			case ast.OperatorLessOrEqual:
				t.Value = cmp <= 0
			case ast.OperatorGreater:
				t.Value = cmp > 0
			case ast.OperatorGreaterOrEqual:
				t.Value = cmp >= 0
			default:
				return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on untyped number)", expr, expr.Op)
			}
			t.Type = boolType
		case ast.OperatorAddition:
			switch v2 := v2.(type) {
			case int64:
				v1 := v1.(int64)
				v := v1 + v2
				if (v < v1) != (v2 < 0) {
					b1 := (&big.Int{}).SetInt64(v1)
					b2 := (&big.Int{}).SetInt64(v2)
					t.Value = b1.Add(b1, b2)
				} else {
					t.Value = v
				}
			case *big.Int:
				t.Value = (&big.Int{}).Add(v1.(*big.Int), v2)
			case float64:
				if v1 == 0 {
					t.Value = v2
				} else if v2 == 0 {
					t.Value = v1
				} else {
					f1 := (&big.Float{}).SetFloat64(v1.(float64))
					f2 := (&big.Float{}).SetFloat64(v2)
					t.Value = f1.Add(f1, f2)
				}
			case *big.Float:
				t.Value = (&big.Float{}).Add(v1.(*big.Float), v2)
			case *big.Rat:
				t.Value = (&big.Rat{}).Add(v1.(*big.Rat), v2)
			}
		case ast.OperatorSubtraction:
			switch v2 := t2.Value.(type) {
			case int64:
				v1 := v1.(int64)
				v := v1 - v2
				if (v < v1) != (v2 > 0) {
					b1 := (&big.Int{}).SetInt64(v1)
					b2 := (&big.Int{}).SetInt64(v2)
					t.Value = b1.Sub(b1, b2)
				} else {
					t.Value = v
				}
			case *big.Int:
				t.Value = (&big.Int{}).Sub(v1.(*big.Int), v2)
			case float64:
				if v2 == 0 {
					t.Value = v1
				} else {
					f1 := (&big.Float{}).SetFloat64(v1.(float64))
					f2 := (&big.Float{}).SetFloat64(v2)
					t.Value = f1.Sub(f1, f2)
				}
			case *big.Float:
				t.Value = (&big.Float{}).Sub(v1.(*big.Float), v2)
			case *big.Rat:
				t.Value = (&big.Rat{}).Sub(v1.(*big.Rat), v2)
			}
		case ast.OperatorMultiplication:
			switch v2 := v2.(type) {
			case int64:
				if v1 == 0 || v2 == 0 {
					t.Value = int64(0)
				} else if v1 == 1 {
					t.Value = v2
				} else if v2 == 1 {
					t.Value = v1
				} else {
					v1 := v1.(int64)
					v := v1 * v2
					if (v < 0) != ((v1 < 0) != (v2 < 0)) || v/v2 != v1 {
						b1 := (&big.Int{}).SetInt64(v1)
						b2 := (&big.Int{}).SetInt64(v2)
						t.Value = b1.Mul(b1, b2)
					} else {
						t.Value = v
					}
				}
			case *big.Int:
				t.Value = (&big.Int{}).Sub(v1.(*big.Int), v2)
			case float64:
				if v1 == 0 || v2 == 0 {
					t.Value = float64(0)
				} else if v1 == 1 {
					t.Value = v2
				} else if v2 == 1 {
					t.Value = v1
				} else {
					f1 := (&big.Float{}).SetFloat64(v1.(float64))
					f2 := (&big.Float{}).SetFloat64(v2)
					t.Value = f1.Mul(f1, f2)
				}
			case *big.Float:
				t.Value = (&big.Float{}).Sub(v1.(*big.Float), v2)
			case *big.Rat:
				t.Value = (&big.Rat{}).Sub(v1.(*big.Rat), v2)
			}
		case ast.OperatorDivision:
			switch v2 := t2.Value.(type) {
			case int64:
				if v2 == 0 {
					return nil, errDivisionByZero
				}
				v1 := v1.(int64)
				if v1%v2 == 0 && !(v1 == minInt64 && v2 == -1) {
					t.Value = v1 / v2
				} else {
					b1 := (&big.Int{}).SetInt64(v1)
					b2 := (&big.Int{}).SetInt64(v2)
					t.Value = (&big.Rat{}).SetFrac(b1, b2)
				}
			case *big.Int:
				if v2.Sign() == 0 {
					return nil, errDivisionByZero
				}
				t.Value = (&big.Rat{}).SetFrac(v1.(*big.Int), v2)
			case float64:
				if v2 == 0 {
					return nil, errDivisionByZero
				}
				if v2 == 1 {
					t.Value = v1
				} else {
					f1 := (&big.Float{}).SetFloat64(v1.(float64))
					f2 := (&big.Float{}).SetFloat64(v2)
					t.Value = (&big.Float{}).Quo(f1, f2)
				}
			case *big.Float:
				if v2.Sign() == 0 {
					return nil, errDivisionByZero
				}
				t.Value = (&big.Float{}).Quo(v1.(*big.Float), v2)
			case *big.Rat:
				if v2.Sign() == 0 {
					return nil, errDivisionByZero
				}
				t.Value = (&big.Rat{}).Quo(v1.(*big.Rat), v2)
			}
		case ast.OperatorModulo:
			k2 := t2.Type.Kind()
			if k1 == reflect.Float64 || k1 == reflect.Float32 || k2 == reflect.Float64 || k2 == reflect.Float32 {
				return nil, errors.New("illegal constant expression: floating-point % operation")
			}
			switch v2 := t2.Value.(type) {
			case int64:
				if v2 == 0 {
					return nil, errDivisionByZero
				}
				t.Value = v1.(int64) % v2
			case *big.Int:
				if v2.Sign() == 0 {
					return nil, errDivisionByZero
				}
				t.Value = (&big.Rat{}).SetFrac(v1.(*big.Int), v2)
			}
		}
	}

	return t, nil
}

// unaryOp executes an unary expression and returns its result. Returns an
// error if the operation can not be executed.
func unaryOp(t *TypeInfo, expr *ast.UnaryOperator) (*TypeInfo, error) {

	k := t.Type.Kind()

	ti := &TypeInfo{
		Type:       t.Type,
		Properties: t.Properties & (PropertyUntyped | PropertyIsConstant),
	}

	switch expr.Op {
	case ast.OperatorNot:
		if t.Nil() || k != reflect.Bool {
			return nil, fmt.Errorf("invalid operation: ! %s", t)
		}
		if t.IsConstant() {
			ti.Value = !t.Value.(bool)
		}
	case ast.OperatorAddition:
		if t.Nil() || !numericKind[k] {
			return nil, fmt.Errorf("invalid operation: + %s", t)
		}
		if t.IsConstant() {
			ti.Value = t.Value
		}
	case ast.OperatorSubtraction:
		if t.Nil() || !numericKind[k] {
			return nil, fmt.Errorf("invalid operation: - %s", t)
		}
		if t.IsConstant() {
			if t.Untyped() {
				switch v := t.Value.(type) {
				case *big.Int:
					ti.Value = (&big.Int{}).Neg(v)
				case *big.Float:
					v = (&big.Float{}).Neg(v)
				case *big.Rat:
					v = (&big.Rat{}).Neg(v)
				}
			} else {
				switch v := t.Value.(type) {
				case *big.Int:
					v = (&big.Int{}).Neg(v)
					min := integerRanges[k-2].min
					max := integerRanges[k-2].max
					if min == nil && v.Sign() < 0 || min != nil && v.Cmp(min) < 0 || v.Cmp(max) > 0 {
						return nil, fmt.Errorf("constant %v overflows %s", v, t)
					}
					ti.Value = v
				case *big.Float:
					v = (&big.Float{}).Neg(v)
					var acc big.Accuracy
					if k == reflect.Float64 {
						_, acc = v.Float64()
					} else {
						_, acc = v.Float32()
					}
					if acc != 0 {
						return nil, fmt.Errorf("constant %v overflows %s", v, t)
					}
					ti.Value = v
				}
			}
		}
	case ast.OperatorMultiplication:
		if t.Type.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("invalid indirect of %s (type %s)", expr.Expr, t)
		}
		ti.Type = t.Type.Elem()
		ti.Properties = ti.Properties | PropertyAddressable
	case ast.OperatorAmpersand:
		if _, ok := expr.Expr.(*ast.CompositeLiteral); !ok && !t.Addressable() {
			return nil, fmt.Errorf("cannot take the address of %s", expr.Expr)
		}
		ti.Type = reflect.PtrTo(t.Type)
	}

	return ti, nil
}
