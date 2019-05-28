// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"

	"scrigo/internal/compiler/ast"
)

var errTypeConversion = errors.New("failed type conversion")

const constPrecision = 512

const (
	maxInt   = int(maxUint >> 1)
	minInt   = -maxInt - 1
	maxInt64 = 1<<63 - 1
	minInt64 = -1 << 63
	maxUint  = ^uint(0)
)

var boolOperators = [21]bool{
	ast.OperatorEqual:    true,
	ast.OperatorNotEqual: true,
	ast.OperatorAndAnd:   true,
	ast.OperatorOrOr:     true,
}

var intOperators = [21]bool{
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
	ast.OperatorAnd:            true,
	ast.OperatorOr:             true,
	ast.OperatorXor:            true,
	ast.OperatorAndNot:         true,
	ast.OperatorLeftShift:      true,
	ast.OperatorRightShift:     true,
}

var floatOperators = [21]bool{
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

var stringOperators = [21]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessOrEqual:    true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterOrEqual: true,
	ast.OperatorAddition:       true,
}

var interfaceOperators = [21]bool{
	ast.OperatorEqual:    true,
	ast.OperatorNotEqual: true,
}

var operatorsOfKind = [...][21]bool{
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
		case ast.OperatorAndAnd:
			t.Value = t1.Value.(bool) && t2.Value.(bool)
		case ast.OperatorOrOr:
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
					b1 := newInt().SetInt64(v1)
					b2 := newInt().SetInt64(v2)
					t.Value = b1.Add(b1, b2)
				} else {
					t.Value = v
				}
			case *big.Int:
				t.Value = newInt().Add(v1.(*big.Int), v2)
			case float64:
				if v1 == 0 {
					t.Value = v2
				} else if v2 == 0 {
					t.Value = v1
				} else {
					f1 := newFloat().SetFloat64(v1.(float64))
					f2 := newFloat().SetFloat64(v2)
					t.Value = f1.Add(f1, f2)
				}
			case *big.Float:
				t.Value = newFloat().Add(v1.(*big.Float), v2)
			case *big.Rat:
				t.Value = newRat().Add(v1.(*big.Rat), v2)
			}
		case ast.OperatorSubtraction:
			switch v2 := v2.(type) {
			case int64:
				v1 := v1.(int64)
				v := v1 - v2
				if (v < v1) != (v2 > 0) {
					b1 := newInt().SetInt64(v1)
					b2 := newInt().SetInt64(v2)
					t.Value = b1.Sub(b1, b2)
				} else {
					t.Value = v
				}
			case *big.Int:
				t.Value = newInt().Sub(v1.(*big.Int), v2)
			case float64:
				if v2 == 0 {
					t.Value = v1
				} else {
					f1 := newFloat().SetFloat64(v1.(float64))
					f2 := newFloat().SetFloat64(v2)
					t.Value = f1.Sub(f1, f2)
				}
			case *big.Float:
				t.Value = newFloat().Sub(v1.(*big.Float), v2)
			case *big.Rat:
				t.Value = newRat().Sub(v1.(*big.Rat), v2)
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
						b1 := newInt().SetInt64(v1)
						b2 := newInt().SetInt64(v2)
						t.Value = b1.Mul(b1, b2)
					} else {
						t.Value = v
					}
				}
			case *big.Int:
				t.Value = newInt().Mul(v1.(*big.Int), v2)
			case float64:
				if v1 == 0 || v2 == 0 {
					t.Value = float64(0)
				} else if v1 == 1 {
					t.Value = v2
				} else if v2 == 1 {
					t.Value = v1
				} else {
					f1 := newFloat().SetFloat64(v1.(float64))
					f2 := newFloat().SetFloat64(v2)
					t.Value = f1.Mul(f1, f2)
				}
			case *big.Float:
				t.Value = newFloat().Sub(v1.(*big.Float), v2)
			case *big.Rat:
				t.Value = newRat().Sub(v1.(*big.Rat), v2)
			}
		case ast.OperatorDivision:
			switch v2 := v2.(type) {
			case int64:
				if v2 == 0 {
					return nil, errDivisionByZero
				}
				v1 := v1.(int64)
				if v1%v2 == 0 && !(v1 == minInt64 && v2 == -1) {
					t.Value = v1 / v2
				} else {
					b1 := newInt().SetInt64(v1)
					b2 := newInt().SetInt64(v2)
					t.Value = newRat().SetFrac(b1, b2)
				}
			case *big.Int:
				if v2.Sign() == 0 {
					return nil, errDivisionByZero
				}
				t.Value = newRat().SetFrac(v1.(*big.Int), v2)
			case float64:
				if v2 == 0 {
					return nil, errDivisionByZero
				}
				if v2 == 1 {
					t.Value = v1
				} else {
					f1 := newFloat().SetFloat64(v1.(float64))
					f2 := newFloat().SetFloat64(v2)
					t.Value = newFloat().Quo(f1, f2)
				}
			case *big.Float:
				if v2.Sign() == 0 {
					return nil, errDivisionByZero
				}
				t.Value = newFloat().Quo(v1.(*big.Float), v2)
			case *big.Rat:
				if v2.Sign() == 0 {
					return nil, errDivisionByZero
				}
				t.Value = newRat().Quo(v1.(*big.Rat), v2)
			}
		case ast.OperatorModulo:
			k2 := t2.Type.Kind()
			if isFloatingPoint(k1) || isFloatingPoint(k2) {
				return nil, errors.New("illegal constant expression: floating-point % operation")
			}
			switch v2 := v2.(type) {
			case int64:
				if v2 == 0 {
					return nil, errDivisionByZero
				}
				t.Value = v1.(int64) % v2
			case *big.Int:
				if v2.Sign() == 0 {
					return nil, errDivisionByZero
				}
				t.Value = newRat().SetFrac(v1.(*big.Int), v2)
			}
		case ast.OperatorAnd:
			k2 := t2.Type.Kind()
			if isFloatingPoint(k1) || isFloatingPoint(k2) {
				return nil, errors.New("invalid operation: operator & not defined on untyped float")
			}
			switch v2 := v2.(type) {
			case int64:
				t.Value = v1.(int64) & v2
			case *big.Int:
				t.Value = newInt().And(v1.(*big.Int), v2)
			}
		case ast.OperatorOr:
			k2 := t2.Type.Kind()
			if isFloatingPoint(k1) || isFloatingPoint(k2) {
				return nil, errors.New("invalid operation: operator | not defined on untyped float")
			}
			switch v2 := v2.(type) {
			case int64:
				t.Value = v1.(int64) | v2
			case *big.Int:
				t.Value = newInt().Or(v1.(*big.Int), v2)
			}
		case ast.OperatorXor:
			k2 := t2.Type.Kind()
			if isFloatingPoint(k1) || isFloatingPoint(k2) {
				return nil, errors.New("invalid operation: operator ^ not defined on untyped float")
			}
			switch v2 := v2.(type) {
			case int64:
				t.Value = v1.(int64) ^ v2
			case *big.Int:
				t.Value = newInt().Xor(v1.(*big.Int), v2)
			}
		case ast.OperatorAndNot:
			k2 := t2.Type.Kind()
			if isFloatingPoint(k1) || isFloatingPoint(k2) {
				return nil, errors.New("invalid operation: operator &^ not defined on untyped float")
			}
			switch v2 := v2.(type) {
			case int64:
				t.Value = v1.(int64) &^ v2
			case *big.Int:
				t.Value = newInt().AndNot(v1.(*big.Int), v2)
			}
		}
	}

	return t, nil
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
		if k2 == reflect.String && isInteger(k1) {
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
	return
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

// isFloatingPoint reports whether a reflect kind is floating point.
func isFloatingPoint(k reflect.Kind) bool {
	return k == reflect.Float64 || k == reflect.Float32
}

// isInteger reports whether a reflect kind is integer.
func isInteger(k reflect.Kind) bool {
	return reflect.Int <= k && k <= reflect.Uint64
}

// isNumeric reports whether a reflect kind is numeric.
func isNumeric(k reflect.Kind) bool {
	return reflect.Int <= k && k <= reflect.Float64
}

// isOrdered reports whether t is ordered.
func isOrdered(t *TypeInfo) bool {
	k := t.Type.Kind()
	return isNumeric(k) || k == reflect.String
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

// newInt returns a new big.Int.
func newInt() *big.Int {
	return new(big.Int)
}

// newFloat returns a new big.Float with precision constPrecision.
func newFloat() *big.Float {
	return new(big.Float).SetPrec(constPrecision)
}

// newRat returns a new big.Rat.
func newRat() *big.Rat {
	return new(big.Rat)
}

// representedBy returns t1 ( a constant or an untyped boolean value )
// represented as a value of type t2. t2 can not be an interface.
func representedBy(t1 *TypeInfo, t2 reflect.Type) (interface{}, error) {

	v := t1.Value
	k := t2.Kind()

	switch {
	case k == reflect.Bool:
		if t1.Type.Kind() != reflect.Bool {
			return nil, fmt.Errorf("cannot convert %v (type %s) to type %s", v, t1, t2)
		}
		return v, nil
	case k == reflect.String:
		if t1.Type.Kind() != reflect.String {
			return nil, fmt.Errorf("cannot convert %v (type %s) to type %s", v, t1, t2)
		}
		return v, nil
	case reflect.Int <= k && k <= reflect.Int64:
		if t1.CanInt64() {
			switch k {
			case reflect.Int:
				n := t1.Int64()
				if int64(minInt) <= n && n <= int64(maxInt) {
					return n, nil
				}
			case reflect.Int8:
				n := t1.Int64()
				if -1<<7 <= n && n <= 1<<7-1 {
					return n, nil
				}
			case reflect.Int16:
				n := t1.Int64()
				if -1<<15 <= n && n <= 1<<15-1 {
					return n, nil
				}
			case reflect.Int32:
				n := t1.Int64()
				if -1<<31 <= n && n <= 1<<31-1 {
					return n, nil
				}
			case reflect.Int64:
				return t1.Int64(), nil
			}
		}
		return nil, fmt.Errorf("constant %v truncated to integer", v)
	case reflect.Uint <= k && k <= reflect.Uintptr:
		if t1.CanUint64() {
			switch k {
			case reflect.Uint:
				n := t1.Uint64()
				if n <= uint64(maxUint) {
					if n <= uint64(maxInt64) {
						return int64(n), nil
					}
					return (&big.Int{}).SetUint64(n), nil
				}
			case reflect.Uint8:
				n := t1.Uint64()
				if n <= 1<<8-1 {
					return int64(n), nil
				}
			case reflect.Uint16:
				n := t1.Uint64()
				if n <= 1<<16-1 {
					return int64(n), nil
				}
			case reflect.Uint32:
				n := t1.Uint64()
				if n <= 1<<32-1 {
					if n <= uint64(maxInt) {
						return int64(n), nil
					}
					return (&big.Int{}).SetUint64(n), nil
				}
			case reflect.Uint64:
				n := t1.Uint64()
				if n <= uint64(maxInt) {
					return int64(n), nil
				}
				return (&big.Int{}).SetUint64(n), nil
			}
		}
		return nil, fmt.Errorf("constant %v overflows %s", v, t2)
	case k == reflect.Float64:
		if t1.CanFloat64() {
			return t1.Float64(), nil
		}
		return nil, fmt.Errorf("constant %v overflows %s", v, t2)
	case k == reflect.Float32:
		if t1.CanFloat64() {
			if f := t1.Float64(); -math.MaxFloat32 <= f && f <= math.MaxFloat32 {
				return float64(float32(f)), nil
			}
		}
		return nil, fmt.Errorf("constant %v overflows %s", v, t2)
	}

	return nil, fmt.Errorf("cannot convert %v (type %s) to type %s", v, t1, t2)
}

// shiftOp executes a shift expression t1 << t2 or t1 >> t2 and returns its
// result. Returns an error if the operation can not be executed.
func shiftOp(t1 *TypeInfo, expr *ast.BinaryOperator, t2 *TypeInfo) (*TypeInfo, error) {
	if t2.Nil() {
		return nil, errors.New("cannot convert nil to type uint")
	}
	if t2.IsUntypedConstant() && !t2.CanInt64() || !t2.IsUntypedConstant() && !t2.IsUnsignedInteger() {
		return nil, fmt.Errorf("invalid operation: %s (shift count type %s, must be unsigned integer)", expr, t2.ShortString())
	}
	var s uint
	if t2.IsConstant() {
		v2 := t2.Int64()
		var err error
		if v2 < 0 {
			err = fmt.Errorf("invalid negative shift count: %d", v2)
		} else if v2 >= constPrecision {
			err = fmt.Errorf("shift count too large: %d", v2)
		}
		if err != nil {
			if !t2.IsUntypedConstant() {
				err = fmt.Errorf("invalid operation: %s (shift count type %s, must be unsigned integer)", expr, t2.ShortString())
			}
			return nil, err
		}
		s = uint(v2)
	}
	if !t1.IsInteger() {
		return nil, fmt.Errorf("invalid operation: %s (shift of type %s)", expr, t1)
	}
	var t *TypeInfo
	if t1.IsConstant() && t2.IsConstant() {
		t = &TypeInfo{Type: t1.Type, Properties: t1.Properties}
		switch v1 := t1.Value.(type) {
		case int64:
			if expr.Op == ast.OperatorLeftShift {
				t.Value = newInt().Lsh(newInt().SetInt64(v1), s)
			} else {
				t.Value = v1 >> s
			}
		case *big.Int:
			if expr.Op == ast.OperatorLeftShift {
				t.Value = newInt().Lsh(v1, s)
			} else {
				t.Value = newInt().Rsh(v1, s)
			}
		}
	} else {
		t = &TypeInfo{Type: t1.Type}
	}
	return t, nil
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
	v, err := representedBy(t, t.Type)
	if err != nil {
		return nil, fmt.Errorf("constant %v overflows %s", v, t1)
	}
	t.Value = v
	return t, nil
}

// typedValue returns a constant type info value represented with a given
// type.
func typedValue(ti *TypeInfo, t reflect.Type) interface{} {
	k := t.Kind()
	if k == reflect.Interface {
		t = ti.Type
		k = t.Kind()
	}
	if t.Name() == "" {
		switch k {
		case reflect.Bool:
			return ti.Value.(bool)
		case reflect.String:
			return ti.Value.(string)
		case reflect.Int:
			return int(ti.Int64())
		case reflect.Int8:
			return int8(ti.Int64())
		case reflect.Int16:
			return int16(ti.Int64())
		case reflect.Int32:
			return int32(ti.Int64())
		case reflect.Int64:
			return ti.Int64()
		case reflect.Uint:
			return uint(ti.Uint64())
		case reflect.Uint8:
			return uint8(ti.Uint64())
		case reflect.Uint16:
			return uint16(ti.Uint64())
		case reflect.Uint32:
			return uint32(ti.Uint64())
		case reflect.Uint64:
			return ti.Uint64()
		case reflect.Float32:
			return float32(ti.Float64())
		case reflect.Float64:
			return ti.Float64()
		default:
			panic(fmt.Sprintf("unexpected kind %q", k))
		}
	}
	nv := reflect.New(t).Elem()
	switch k {
	case reflect.Bool:
		nv.SetBool(ti.Value.(bool))
	case reflect.String:
		nv.SetString(ti.Value.(string))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		nv.SetInt(ti.Int64())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		nv.SetUint(ti.Uint64())
	case reflect.Float32, reflect.Float64:
		nv.SetFloat(ti.Float64())
	default:
		panic("unexpected kind")
	}
	return nv.Interface()
}

// toSameType returns v1 and v2 with the same types.
func toSameType(v1, v2 interface{}) (interface{}, interface{}) {
	switch n1 := v1.(type) {
	case int64:
		switch n2 := v2.(type) {
		case *big.Int:
			v1 = newInt().SetInt64(n1)
		case float64:
			if int64(float64(n1)) == n1 {
				v2 = float64(n2)
			} else {
				v1 = newFloat().SetInt64(n1)
				v2 = newFloat().SetFloat64(n2)
			}
		case *big.Float:
			v1 = newFloat().SetInt64(n1)
		case *big.Rat:
			v1 = newRat().SetInt64(n1)
		}
	case *big.Int:
		switch n2 := v2.(type) {
		case int64:
			v2 = newInt().SetInt64(n2)
		case float64:
			v1 = newFloat().SetInt(n1)
			v2 = newFloat().SetFloat64(n2)
		case *big.Float:
			v1 = newFloat().SetInt(n1)
		case *big.Rat:
			v1 = newRat().SetInt(n1)
		}
	case float64:
		switch n2 := v2.(type) {
		case int64:
			if int64(float64(n2)) == n2 {
				v2 = float64(n2)
			} else {
				v1 = newFloat().SetFloat64(n1)
				v2 = newFloat().SetInt64(n2)
			}
		case *big.Int:
			v1 = newFloat().SetFloat64(n1)
			v2 = newFloat().SetInt(n2)
		case *big.Float:
			v1 = newFloat().SetFloat64(n1)
		case *big.Rat:
			v1 = newRat().SetFloat64(n1)
		}
	case *big.Float:
		switch n2 := v2.(type) {
		case int64:
			v2 = newFloat().SetInt64(n2)
		case float64:
			v2 = newFloat().SetFloat64(n2)
		case *big.Int:
			v2 = newFloat().SetInt(n2)
		case *big.Rat:
			num := newFloat().SetInt(n2.Num())
			den := newFloat().SetInt(n2.Denom())
			v2 = num.Quo(num, den)
		}
	case *big.Rat:
		switch n2 := v2.(type) {
		case int64:
			v2 = newRat().SetInt64(n2)
		case float64:
			v2 = newRat().SetFloat64(n2)
		case *big.Int:
			v2 = newRat().SetInt(n2)
		case *big.Float:
			num := newFloat().SetInt(n1.Num())
			den := newFloat().SetInt(n1.Denom())
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

	if !(k1 == k2 || isNumeric(k1) && isNumeric(k2)) {
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

// unaryOp executes an unary expression and returns its result. Returns an
// error if the operation can not be executed.
func (tc *typechecker) unaryOp(t *TypeInfo, expr *ast.UnaryOperator) (*TypeInfo, error) {

	var k reflect.Kind
	if !t.Nil() {
		k = t.Type.Kind()
	}

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
		if t.Nil() || !isNumeric(k) {
			return nil, fmt.Errorf("invalid operation: + %s", t)
		}
		if t.IsConstant() {
			ti.Value = t.Value
		}
	case ast.OperatorSubtraction:
		if t.Nil() || !isNumeric(k) {
			return nil, fmt.Errorf("invalid operation: - %s", t)
		}
		if t.IsConstant() {
			switch v := t.Value.(type) {
			case int64:
				if v > 0 && reflect.Uint8 <= k && k <= reflect.Uintptr {
					return nil, fmt.Errorf("constant %v overflows %s", v, ti)
				}
				if v > minInt64 {
					ti.Value = -v
				} else {
					n := big.NewInt(v)
					ti.Value = n.Neg(n)
				}
				if !ti.Untyped() {
					v, err := representedBy(ti, ti.Type)
					if err != nil {
						return nil, fmt.Errorf("constant %v overflows %s", v, ti)
					}
					t.Value = v
				}
			case *big.Int:
				ti.Value = newInt().Neg(v)
				if !ti.Untyped() {
					v, err := representedBy(ti, ti.Type)
					if err != nil {
						return nil, fmt.Errorf("constant %v overflows %s", v, ti)
					}
					t.Value = v
				}
			case float64:
				ti.Value = -v
			case *big.Float:
				ti.Value = newFloat().Neg(v)
			case *big.Rat:
				ti.Value = newRat().Neg(v)
			}
		}
	case ast.OperatorMultiplication:
		if t.Nil() {
			return nil, fmt.Errorf("invalid indirect of nil")
		}
		if k != reflect.Ptr {
			return nil, fmt.Errorf("invalid indirect of %s (type %s)", expr.Expr, t)
		}
		ti.Type = t.Type.Elem()
		ti.Properties = ti.Properties | PropertyAddressable
	case ast.OperatorAnd:
		if _, ok := expr.Expr.(*ast.CompositeLiteral); !ok && !t.Addressable() {
			return nil, fmt.Errorf("cannot take the address of %s", expr.Expr)
		}
		ti.Type = reflect.PtrTo(t.Type)
		// When taking the address of a variable, such variable must be
		// marked as "indirect".
		if ident, ok := expr.Expr.(*ast.Identifier); ok {
		scopesLoop:
			for i := len(tc.Scopes) - 1; i >= 0; i-- {
				for n := range tc.Scopes[i] {
					if n == ident.Name {
						tc.IndirectVars[tc.Scopes[i][n].decl] = true
						break scopesLoop
					}
				}
			}
		}
	case ast.OperatorXor:
		panic("TODO: implements unary XOR")
	case ast.OperatorReceive:
		if t.Nil() {
			return nil, fmt.Errorf("use of untyped nil")
		}
		if k != reflect.Chan {
			return nil, fmt.Errorf("invalid operation: %s (receive from non-chan type %s)", expr.Expr, t.Type)
		}
		ti.Type = t.Type.Elem()
	}

	return ti, nil
}
