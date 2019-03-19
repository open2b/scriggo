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
	maxInt  = int(maxUint >> 1)
	minInt  = -maxInt - 1
	maxUint = ^uint(0)
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

// containsDuplicates returns true if ss contains at least one duplicate string.
func containsDuplicates(ss []string) bool {
	for _, a := range ss {
		count := 0
		for _, b := range ss {
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
func convert(ti *ast.TypeInfo, t2 reflect.Type) (interface{}, error) {

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
func convertImplicit(ti *ast.TypeInfo, t2 reflect.Type) (interface{}, error) {

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

// fieldByName returns the struct field with the given name and a boolean
// indicating if the field was found.
func fieldByName(t *ast.TypeInfo, name string) (*ast.TypeInfo, bool) {
	if t.Type.Kind() == reflect.Struct {
		field, ok := t.Type.FieldByName(name)
		if ok {
			return &ast.TypeInfo{Type: field.Type}, true
		}
	}
	if t.Type.Kind() == reflect.Ptr {
		field, ok := t.Type.Elem().FieldByName(name)
		if ok {
			return &ast.TypeInfo{Type: field.Type}, true
		}
	}
	return nil, false
}

// isAssignableTo reports whether x is assignable to type t.
// See https://golang.org/ref/spec#Assignability for details.
func isAssignableTo(x *ast.TypeInfo, t reflect.Type) bool {
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
	if x.Untyped() {
		_, err := representedBy(x, t)
		return err == nil
	}
	if t.Kind() == reflect.Interface && x.Type.Implements(t) {
		return true
	}
	// Checks if the type of x and t have identical underlying types and at
	// least one is not a defined type.
	return x.Type.AssignableTo(t)
}

// isComparison reports whether op is a comparison operator.
func isComparison(op ast.OperatorType) bool {
	return op >= ast.OperatorEqual && op <= ast.OperatorGreaterOrEqual
}

// isOrdered reports whether t is ordered.
func isOrdered(t *ast.TypeInfo) bool {
	k := t.Type.Kind()
	return numericKind[k] || k == reflect.String
}

// methodByName returns a function type that describe the method with that
// name and a boolean indicating if the method was found.
//
// Only for type classes, the returned function type has the method's
// receiver as first argument.
func methodByName(t *ast.TypeInfo, name string) (*ast.TypeInfo, bool) {
	if t.IsType() {
		if method, ok := t.Type.MethodByName(name); ok {
			return &ast.TypeInfo{Type: method.Type}, true
		}
		return nil, false
	}
	method := reflect.Zero(t.Type).MethodByName(name)
	if method.IsValid() {
		return &ast.TypeInfo{Type: method.Type()}, true
	}
	if t.Type.Kind() != reflect.Ptr {
		method = reflect.Zero(reflect.PtrTo(t.Type)).MethodByName(name)
		if method.IsValid() {
			return &ast.TypeInfo{Type: method.Type()}, true
		}
	}
	return nil, false
}

// representedBy returns a constant value represented as a value of type t2.
func representedBy(t1 *ast.TypeInfo, t2 reflect.Type) (interface{}, error) {

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

// stringInSlice indicates if ss contains s.
func sliceContainsString(ss []string, s string) bool {
	for _, ts := range ss {
		if s == ts {
			return true
		}
	}
	return false
}

// tBinaryOp executes a binary expression where the operands are typed
// constants and returns its result. Returns an error if the operation can not
// be executed.
func tBinaryOp(t1 *ast.TypeInfo, expr *ast.BinaryOperator, t2 *ast.TypeInfo) (*ast.TypeInfo, error) {

	if t1.Type != t2.Type {
		return nil, fmt.Errorf("invalid operation: %v (mismatched types %s and %s)", expr, t1, t2)
	}

	t := &ast.TypeInfo{Type: boolType, Properties: ast.PropertyIsConstant}

	switch v1 := t1.Value.(type) {
	case bool:
		v2 := t2.Value.(bool)
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = v1 == v2
		case ast.OperatorNotEqual:
			t.Value = v1 != v2
		case ast.OperatorAnd:
			t.Value = v1 && v2
		case ast.OperatorOr:
			t.Value = v1 || v2
		default:
			return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t1)
		}
	case string:
		v2 := t2.Value.(string)
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = v1 == v2
		case ast.OperatorNotEqual:
			t.Value = v1 != v2
		case ast.OperatorAddition:
			t.Value = v1 + v2
			t.Type = t1.Type
		default:
			return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t1)
		}
	case *big.Int:
		v2 := t2.Value.(*big.Int)
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = v1.Cmp(v2) == 0
		case ast.OperatorNotEqual:
			t.Value = v1.Cmp(v2) != 0
		case ast.OperatorLess:
			t.Value = v1.Cmp(v2) < 0
		case ast.OperatorLessOrEqual:
			t.Value = v1.Cmp(v2) <= 0
		case ast.OperatorGreater:
			t.Value = v1.Cmp(v2) > 0
		case ast.OperatorGreaterOrEqual:
			t.Value = v1.Cmp(v2) >= 0
		case ast.OperatorAddition:
			t.Value = big.NewInt(0).Add(v1, v2)
		case ast.OperatorSubtraction:
			t.Value = big.NewInt(0).Sub(v1, v2)
		case ast.OperatorMultiplication:
			t.Value = big.NewInt(0).Mul(v1, v2)
		case ast.OperatorDivision:
			t.Value = big.NewInt(0).Quo(v1, v2)
		case ast.OperatorModulo:
			t.Value = big.NewInt(0).Rem(v1, v2)
		}
		if v, ok := t.Value.(*big.Int); ok {
			min := integerRanges[t1.Type.Kind()-2].min
			max := integerRanges[t1.Type.Kind()-2].max
			if min == nil && v.Sign() < 0 || min != nil && v.Cmp(min) < 0 || v.Cmp(max) > 0 {
				return nil, fmt.Errorf("constant %v overflows %s", v, t1)
			}
			t.Type = t1.Type
		}
	case *big.Float:
		v2 := t2.Value.(*big.Float)
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = v1.Cmp(v2) == 0
		case ast.OperatorNotEqual:
			t.Value = v1.Cmp(v2) != 0
		case ast.OperatorLess:
			t.Value = v1.Cmp(v2) < 0
		case ast.OperatorLessOrEqual:
			t.Value = v1.Cmp(v2) <= 0
		case ast.OperatorGreater:
			t.Value = v1.Cmp(v2) > 0
		case ast.OperatorGreaterOrEqual:
			t.Value = v1.Cmp(v2) >= 0
		case ast.OperatorAddition:
			t.Value = big.NewFloat(0).Add(v1, v2)
		case ast.OperatorSubtraction:
			t.Value = big.NewFloat(0).Sub(v1, v2)
		case ast.OperatorMultiplication:
			t.Value = big.NewFloat(0).Mul(v1, v2)
		case ast.OperatorDivision:
			t.Value = big.NewFloat(0).Quo(v1, v2)
		case ast.OperatorModulo:
			return nil, fmt.Errorf("invalid operation: %v (operator %% not defined on %s)", expr, t1)
		}
		if v, ok := t.Value.(*big.Float); ok {
			var acc big.Accuracy
			if t1.Type.Kind() == reflect.Float64 {
				_, acc = v.Float64()
			} else {
				_, acc = v.Float32()
			}
			if acc != 0 {
				return nil, fmt.Errorf("constant %v overflows %s", v, t1)
			}
			t.Type = t1.Type
		}
	}

	return t, nil
}

// toSameType returns v1 and v2 with the same types.
func toSameType(v1, v2 interface{}) (interface{}, interface{}) {
	switch n1 := v1.(type) {
	case *big.Int:
		switch v2.(type) {
		case *big.Float:
			v1 = (&big.Float{}).SetInt(n1)
		case *big.Rat:
			v1 = (&big.Rat{}).SetInt(n1)
		}
	case *big.Float:
		switch n2 := v2.(type) {
		case *big.Int:
			v2 = (&big.Float{}).SetInt(n2)
		case *big.Rat:
			num := (&big.Float{}).SetInt(n2.Num())
			den := (&big.Float{}).SetInt(n2.Denom())
			v2 = num.Quo(num, den)
		}
	case *big.Rat:
		switch n2 := v2.(type) {
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
func uBinaryOp(t1 *ast.TypeInfo, expr *ast.BinaryOperator, t2 *ast.TypeInfo) (*ast.TypeInfo, error) {

	k1 := t1.Type.Kind()
	k2 := t2.Type.Kind()

	if !(k1 == k2 || numericKind[k1] && numericKind[k2]) {
		return nil, fmt.Errorf("invalid operation: %v (mismatched types %s and %s)",
			expr, t1.ShortString(), t2.ShortString())
	}

	t := &ast.TypeInfo{Type: boolType, Properties: ast.PropertyUntyped | ast.PropertyIsConstant}

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
	case reflect.String:
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = t1.Value.(string) == t2.Value.(string)
		case ast.OperatorNotEqual:
			t.Value = t1.Value.(string) != t2.Value.(string)
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
			case *big.Int:
				cmp = v1.Cmp(v2.(*big.Int))
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
			return t, nil
		case ast.OperatorAddition:
			switch v2 := v2.(type) {
			case *big.Int:
				t.Value = (&big.Int{}).Add(v1.(*big.Int), v2)
			case *big.Float:
				t.Value = (&big.Float{}).Add(v1.(*big.Float), v2)
			case *big.Rat:
				t.Value = (&big.Rat{}).Add(v1.(*big.Rat), v2)
			}
		case ast.OperatorSubtraction:
			switch v2 := t2.Value.(type) {
			case *big.Int:
				t.Value = (&big.Int{}).Sub(v1.(*big.Int), v2)
			case *big.Float:
				t.Value = (&big.Float{}).Sub(v1.(*big.Float), v2)
			case *big.Rat:
				t.Value = (&big.Rat{}).Sub(v1.(*big.Rat), v2)
			}
		case ast.OperatorMultiplication:
			switch v2 := v2.(type) {
			case *big.Int:
				t.Value = (&big.Int{}).Sub(v1.(*big.Int), v2)
			case *big.Float:
				t.Value = (&big.Float{}).Sub(v1.(*big.Float), v2)
			case *big.Rat:
				t.Value = (&big.Rat{}).Sub(v1.(*big.Rat), v2)
			}
		case ast.OperatorDivision:
			switch v2 := t2.Value.(type) {
			case *big.Int:
				if v2.Sign() == 0 {
					return nil, errDivisionByZero
				}
				t.Value = (&big.Rat{}).SetFrac(v1.(*big.Int), v2)
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
			if k1 == reflect.Float64 || k2 == reflect.Float64 {
				return nil, errors.New("illegal constant expression: floating-point % operation")
			}
			vv2 := v2.(*big.Int)
			if vv2.Sign() == 0 {
				return nil, errDivisionByZero
			}
			t.Value = (&big.Rat{}).SetFrac(v1.(*big.Int), vv2)
		}
		t.Type = t1.Type
		if k2 > k1 {
			t.Type = t2.Type
		}
	}

	return t, nil
}

// unaryOp executes an unary expression and returns its result. Returns an
// error if the operation can not be executed.
func unaryOp(expr *ast.UnaryOperator) (*ast.TypeInfo, error) {

	t := expr.Expr.TypeInfo()
	k := t.Type.Kind()

	ti := &ast.TypeInfo{
		Type:       t.Type,
		Properties: t.Properties & (ast.PropertyUntyped | ast.PropertyIsConstant),
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
			return nil, fmt.Errorf("invalid indirect of %s (type %s)", expr, t)
		}
		ti.Type = t.Type.Elem()
		ti.Properties = ti.Properties | ast.PropertyAddressable
	case ast.OperatorAmpersand:
		if _, ok := expr.Expr.(*ast.CompositeLiteral); !ok && !t.Addressable() {
			return nil, fmt.Errorf("cannot take the address of %s", expr)
		}
		ti.Type = reflect.PtrTo(t.Type)
	}

	return ti, nil
}
