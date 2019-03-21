// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"scrigo/ast"
)

const maxInt = int64(^uint(0) >> 1)
const minInt = -int64(^uint(0)>>1) - 1

var errIntegerDivideByZero = errors.New("integer divide by zero")

var reflectValueNil = reflect.ValueOf(new(interface{})).Elem()

// Package represents a package.
type Package struct {
	Name         string
	Declarations map[string]interface{}
}

// Bytes implements the mutable bytes values.
type Bytes = []byte

// eval evaluates expr in a single-value context and returns its value.
func (r *rendering) eval(expr ast.Expression) (value interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				panic(r)
			}
		}
	}()
	return r.evalExpression(expr), nil
}

// eval0 evaluates expr in a void context. It returns an error if the
// expression evaluates to a value.
func (r *rendering) eval0(expr ast.Expression) error {
	_, err := r.evalCallN(expr.(*ast.Call), 0)
	return err
}

// eval1 evaluates expr in a single-value context converting number constants to
// a number type.
func (r *rendering) eval1(expr ast.Expression) (interface{}, error) {
	v, err := r.eval(expr)
	if err == nil {
		if n, ok := v.(ConstantNumber); ok {
			v, err = n.ToTyped()
		}
	}
	return v, err
}

// eval2 evaluates expr1 and expr2 in a two-value context and returns its
// values. Number constants are converted to a number type.
func (r *rendering) eval2(expr1, expr2 ast.Expression) (v1, v2 interface{}, err error) {
	if expr2 == nil {
		switch e := expr1.(type) {
		case *ast.TypeAssertion:
			v, err := r.eval(e.Expr)
			if err != nil {
				return nil, false, err
			}
			typ, err := r.evalType(e.Type, noEllipses)
			if err != nil {
				return nil, false, err
			}
			ok, err := hasType(v, typ)
			if err != nil {
				return nil, nil, r.errorf(e.Expr, "%s", err)
			}
			if !ok {
				v = zeroOf(typ)
			}
			return v, ok, nil
		case *ast.Selector:
			return r.evalSelector2(e)
		case *ast.Index:
			return r.evalIndex2(e, 2)
		case *ast.Call:
			values, err := r.evalCallN(e, 2)
			if err != nil {
				return nil, nil, err
			}
			return values[0].Interface(), values[1].Interface(), nil
		case *ast.Identifier:
			for i := len(r.vars) - 1; i >= 0; i-- {
				if r.vars[i] != nil {
					if v, ok := r.vars[i][e.Name]; ok {
						switch v.(type) {
						default:
							return v, true, nil
						case reference:
							panic("referenced not implemented in (*rendering).eval2")
						}
					}
				}
			}
			return nil, false, nil
		}
		return nil, nil, nil
	} else {
		v1, err := r.eval(expr1)
		if err != nil {
			return nil, nil, err
		}
		if n, ok := v1.(ConstantNumber); ok {
			v1, err = n.ToTyped()
			if err != nil {
				return nil, nil, err
			}
		}
		v2, err := r.eval(expr2)
		if err != nil {
			return nil, nil, err
		}
		if n, ok := v2.(ConstantNumber); ok {
			v2, err = n.ToTyped()
			if err != nil {
				return nil, nil, err
			}
		}
		return v1, v2, nil
	}
}

// evalN evaluates expressions in a n-value context, with n > 0, and returns
// its values. Number constants are converted to a number type.
func (r *rendering) evalN(expressions []ast.Expression, n int) ([]interface{}, error) {
	if len(expressions) == 1 && n > 1 {
		if e, ok := expressions[0].(*ast.Call); ok {
			vals, err := r.evalCallN(e, n)
			if err != nil {
				return nil, err
			}
			values := make([]interface{}, len(vals))
			for i, value := range vals {
				values[i] = value.Interface()
			}
			return values, nil
		}
	} else if len(expressions) == n {
		values := make([]interface{}, len(expressions))
		for i, expr := range expressions {
			value, err := r.eval(expr)
			if err != nil {
				return nil, err
			}
			if n, ok := value.(ConstantNumber); ok {
				value, err = n.ToTyped()
				if err != nil {
					return nil, err
				}
			}
			values[i] = value
		}
		return values, nil
	}
	return nil, nil
}

// evalSliceType evaluates a slice type returning its value.
func (r *rendering) evalSliceType(node *ast.SliceType) reflect.Type {
	t, err := r.evalType(node, noEllipses)
	if err != nil {
		panic(err)
	}
	return t
}

// evalMapType evaluates a map type returning its value.
func (r *rendering) evalMapType(node *ast.MapType) reflect.Type {
	t, err := r.evalType(node, noEllipses)
	if err != nil {
		panic(err)
	}
	return t
}

// evalExpression evaluates an expression and returns its value.
// In the event of an error, calls panic with the error as parameter.
func (r *rendering) evalExpression(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.String:
		return e.Text
	case *ast.Rune:
		return newConstantRune(e.Value)
	case *ast.Int:
		return newConstantInt(&e.Value)
	case *ast.Float:
		return newConstantFloat(&e.Value)
	case *ast.Parenthesis:
		return r.evalExpression(e.Expr)
	case *ast.UnaryOperator:
		return r.evalUnaryOperator(e)
	case *ast.BinaryOperator:
		return r.evalBinaryOperator(e)
	case *ast.Identifier:
		return r.evalIdentifier(e)
	case *ast.MapType:
		return r.evalMapType(e)
	case *ast.SliceType:
		return r.evalSliceType(e)
	case *ast.CompositeLiteral:
		return r.evalCompositeLiteral(e, nil)
	case *ast.Func:
		return r.evalFunc(e)
	case *ast.Call:
		return r.evalCall(e)
	case *ast.Index:
		return r.evalIndex(e)
	case *ast.Slicing:
		return r.evalSlicing(e)
	case *ast.Selector:
		return r.evalSelector(e)
	case *ast.TypeAssertion:
		return r.evalTypeAssertion(e)
	default:
		panic(fmt.Errorf("unexpected node type %s", typeof(expr)))
	}
}

// refToCopy returns a reference to a copy of v (not to v itself).
func refToCopy(v interface{}) reflect.Value {
	rv := reflect.New(reflect.TypeOf(v))
	rv.Elem().Set(reflect.ValueOf(v))
	return rv
}

// referenceInScope substitutes the value of variable in the scope with a
// referenced one which provides a static address. If variable is already
// referenced, referenceInScope returns its value as is.
func (r *rendering) referenceInScope(variable string) (reflect.Value, error) {
	for i := len(r.vars) - 1; i >= 0; i-- {
		if r.vars[i] != nil {
			if v, ok := r.vars[i][variable]; ok {
				switch vv := v.(type) {
				case reference:
					return vv.rv.Addr(), nil
				default:
					rv := refToCopy(vv)
					r.vars[i][variable] = reference{rv.Elem()}
					return rv, nil
				}
			}
		}
	}
	return reflect.Value{}, fmt.Errorf("undefined: %s", variable)
}

// evalUnaryOperator evaluates a unary operator and returns its value. On error
// it calls panic with the error as argument.
func (r *rendering) evalUnaryOperator(node *ast.UnaryOperator) interface{} {
	if node.Operator() == ast.OperatorAmpersand {
		switch expr := node.Expr.(type) {
		case *ast.Identifier: // &a
			ref, err := r.referenceInScope(expr.Name)
			if err != nil {
				panic(r.errorf(node, err.Error()))
			}
			return ref.Interface()
		case *ast.CompositeLiteral: // &T{...}
			return refToCopy(r.evalCompositeLiteral(expr, nil)).Interface()
		case *ast.UnaryOperator: // &*a
			v, err := r.eval(expr.Expr)
			if err != nil {
				return err
			}
			return v
		case *ast.Selector: // &a.b
			ident, ok := expr.Expr.(*ast.Identifier)
			if ok {
				rv, err := r.referenceInScope(ident.Name)
				if err != nil {
					panic(r.errorf(node, err.Error()))
				}
				switch rv.Elem().Kind() {
				case reflect.Struct:
					return rv.Elem().FieldByName(expr.Ident).Addr().Interface()
				}
			}
		case *ast.Index: // &...[i]
			switch e := expr.Expr.(type) {
			case *ast.CompositeLiteral: // &T{...}[i]
				compositeLiteralValue := r.evalCompositeLiteral(e, nil)
				compositeLiteralReflectValue := reflect.ValueOf(compositeLiteralValue)
				switch compositeLiteralReflectValue.Kind() {
				case reflect.Slice: // &[]T1{...}[i]
					i, err := r.sliceIndex(expr.Index)
					if err != nil {
						panic(err)
					}
					elem := compositeLiteralReflectValue.Index(i)
					return refToCopy(elem.Interface()).Interface()
				}
			case *ast.Identifier: // &a[i]
				rv, err := r.referenceInScope(e.Name)
				if err != nil {
					panic(r.errorf(node, err.Error()))
				}
				switch rv.Elem().Kind() {
				case reflect.Slice, reflect.Array: // &a[i] where "a" is a slice or an array
					i, err := r.sliceIndex(expr.Index)
					if err != nil {
						panic(err)
					}
					return rv.Elem().Index(i).Addr().Interface()
				}
			}
		}
		return nil
	}
	var expr = r.evalExpression(node.Expr)
	switch node.Op {
	case ast.OperatorMultiplication:
		if reflect.TypeOf(expr).Kind() != reflect.Ptr {
			// TODO (Gianluca): this error message doesn't match with Go's one.
			panic(r.errorf(node, "invalid indirect of %s (type %s)", node.Expr, typeof(expr)))
		}
		return reflect.ValueOf(expr).Elem().Interface()
	case ast.OperatorNot:
		switch b := expr.(type) {
		case bool:
			return !b
		}
		panic(r.errorf(node, "invalid operation: ! %s", typeof(expr)))
	case ast.OperatorAddition:
		switch expr.(type) {
		case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8, float64, float32, CustomNumber:
			return expr
		}
		panic(r.errorf(node, "invalid operation: + %s", typeof(expr)))
	case ast.OperatorSubtraction:
		switch n := expr.(type) {
		case int:
			return -n
		case int64:
			return -n
		case int32:
			return -n
		case int16:
			return -n
		case int8:
			return -n
		case uint:
			return -n
		case uint64:
			return -n
		case uint32:
			return -n
		case uint16:
			return -n
		case uint8:
			return -n
		case float64:
			return -n
		case float32:
			return -n
		case ConstantNumber:
			return n.Neg()
		case CustomNumber:
			return n.New().Neg(n)
		}
		panic(r.errorf(node, "invalid operation: - %s", typeof(expr)))
	}
	panic(fmt.Sprintf("unknown unary operator %s", node.Op))
}

// evalBinaryOperator evaluates a binary operator and returns its value.
// On error it calls panic with the error as argument.
func (r *rendering) evalBinaryOperator(op *ast.BinaryOperator) interface{} {
	expr1 := r.evalExpression(op.Expr1)
	e, err := r.evalBinary(expr1, op.Op, op.Expr1, op.Expr2)
	if err != nil {
		if err == errIntegerDivideByZero {
			panic(r.errorf(op, err.Error()))
		}
		panic(r.errorf(op, "invalid operation: %s (%s)", op, err))
	}
	return e
}

// evalBinaryInt64 returns the result of the binary expression e1 op e2.
func evalBinaryInt64(e1 int64, op ast.OperatorType, e2 int64) (interface{}, error) {
	switch op {
	case ast.OperatorEqual:
		return e1 == e2, nil
	case ast.OperatorNotEqual:
		return e1 != e2, nil
	case ast.OperatorLess:
		return e1 < e2, nil
	case ast.OperatorLessOrEqual:
		return e1 <= e2, nil
	case ast.OperatorGreater:
		return e1 > e2, nil
	case ast.OperatorGreaterOrEqual:
		return e1 >= e2, nil
	case ast.OperatorAddition:
		return e1 + e2, nil
	case ast.OperatorSubtraction:
		return e1 - e2, nil
	case ast.OperatorMultiplication:
		return e1 * e2, nil
	case ast.OperatorDivision:
		if e2 == 0 {
			return nil, errIntegerDivideByZero
		}
		return e1 / e2, nil
	case ast.OperatorModulo:
		if e2 == 0 {
			return nil, errIntegerDivideByZero
		}
		return e1 % e2, nil
	}
	return nil, fmt.Errorf("error")
}

// evalBinaryUint64 returns the result of the binary expression e1 op e2.
func evalBinaryUint64(e1 uint64, op ast.OperatorType, e2 uint64) (interface{}, error) {
	switch op {
	case ast.OperatorEqual:
		return e1 == e2, nil
	case ast.OperatorNotEqual:
		return e1 != e2, nil
	case ast.OperatorLess:
		return e1 < e2, nil
	case ast.OperatorLessOrEqual:
		return e1 <= e2, nil
	case ast.OperatorGreater:
		return e1 > e2, nil
	case ast.OperatorGreaterOrEqual:
		return e1 >= e2, nil
	case ast.OperatorAddition:
		return e1 + e2, nil
	case ast.OperatorSubtraction:
		return e1 - e2, nil
	case ast.OperatorMultiplication:
		return e1 * e2, nil
	case ast.OperatorDivision:
		if e2 == 0 {
			return nil, errIntegerDivideByZero
		}
		return e1 / e2, nil
	case ast.OperatorModulo:
		if e2 == 0 {
			return nil, errIntegerDivideByZero
		}
		return e1 % e2, nil
	}
	return nil, fmt.Errorf("error")
}

// evalBinaryFloat64 returns the result of the binary expression e1 op e2.
func evalBinaryFloat64(e1 float64, op ast.OperatorType, e2 float64) (interface{}, error) {
	switch op {
	case ast.OperatorEqual:
		return e1 == e2, nil
	case ast.OperatorNotEqual:
		return e1 != e2, nil
	case ast.OperatorLess:
		return e1 < e2, nil
	case ast.OperatorLessOrEqual:
		return e1 <= e2, nil
	case ast.OperatorGreater:
		return e1 > e2, nil
	case ast.OperatorGreaterOrEqual:
		return e1 >= e2, nil
	case ast.OperatorAddition:
		return e1 + e2, nil
	case ast.OperatorSubtraction:
		return e1 - e2, nil
	case ast.OperatorMultiplication:
		return e1 * e2, nil
	case ast.OperatorDivision:
		return e1 / e2, nil
	default:
		return false, fmt.Errorf("operator %s not defined on string", op)
	}
}

// evalBinary returns the result of the binary expression expr1 op expr2.
func (r *rendering) evalBinary(expr1 interface{}, op ast.OperatorType, op1, op2 ast.Expression) (interface{}, error) {

	var err error
	var expr2 interface{}

	if _, ok := expr1.(bool); !ok {
		expr2, err = r.eval(op2)
		if err != nil {
			return nil, err
		}
	}

SWITCH:
	switch e1 := expr1.(type) {
	case string:
		var e2 string
		switch b := expr2.(type) {
		case string:
			e2 = b
		case HTML:
			e2 = string(b)
		default:
			break SWITCH
		}
		switch op {
		case ast.OperatorEqual:
			return e1 == e2, nil
		case ast.OperatorNotEqual:
			return e1 != e2, nil
		case ast.OperatorLess:
			return e1 < e2, nil
		case ast.OperatorLessOrEqual:
			return e1 <= e2, nil
		case ast.OperatorGreater:
			return e1 > e2, nil
		case ast.OperatorGreaterOrEqual:
			return e1 >= e2, nil
		case ast.OperatorAddition:
			return e1 + e2, nil
		default:
			return false, fmt.Errorf("operator %s not defined on string", op)
		}
	case HTML:
		var e2 HTML
		switch b := expr2.(type) {
		case HTML:
			e2 = b
		case string:
			e2 = HTML(b)
		default:
			break SWITCH
		}
		switch op {
		case ast.OperatorEqual:
			return e1 == e2, nil
		case ast.OperatorNotEqual:
			return e1 != e2, nil
		case ast.OperatorLess:
			return e1 < e2, nil
		case ast.OperatorLessOrEqual:
			return e1 <= e2, nil
		case ast.OperatorGreater:
			return e1 > e2, nil
		case ast.OperatorGreaterOrEqual:
			return e1 >= e2, nil
		case ast.OperatorAddition:
			return e1 + e2, nil
		default:
			return false, fmt.Errorf("operator %s not defined on string", op)
		}
	case int:
		var e2 int
		switch y := expr2.(type) {
		case int:
			e2 = y
		case ConstantNumber:
			e2, err = y.ToInt()
			if err != nil {
				return nil, err
			}
		default:
			break SWITCH
		}
		x, err := evalBinaryInt64(int64(e1), op, int64(e2))
		if v, ok := x.(int64); ok {
			return int(v), nil
		}
		return x, err
	case int64:
		var e2 int64
		switch y := expr2.(type) {
		case int64:
			e2 = y
		case ConstantNumber:
			e2, err = y.ToInt64()
			if err != nil {
				return nil, err
			}
		default:
			break SWITCH
		}
		return evalBinaryInt64(e1, op, e2)
	case int32:
		var e2 int32
		switch y := expr2.(type) {
		case int32:
			e2 = y
		case ConstantNumber:
			e2, err = y.ToInt32()
			if err != nil {
				return nil, err
			}
		default:
			break SWITCH
		}
		x, err := evalBinaryInt64(int64(e1), op, int64(e2))
		if v, ok := x.(int64); ok {
			return int32(v), nil
		}
		return x, err
	case int16:
		var e2 int16
		switch y := expr2.(type) {
		case int16:
			e2 = y
		case ConstantNumber:
			e2, err = y.ToInt16()
			if err != nil {
				return nil, err
			}
		default:
			break SWITCH
		}
		x, err := evalBinaryInt64(int64(e1), op, int64(e2))
		if v, ok := x.(int16); ok {
			return int(v), nil
		}
		return x, err
	case int8:
		var e2 int8
		switch y := expr2.(type) {
		case int8:
			e2 = y
		case ConstantNumber:
			e2, err = y.ToInt8()
			if err != nil {
				return nil, err
			}
		default:
			break SWITCH
		}
		x, err := evalBinaryInt64(int64(e1), op, int64(e2))
		if v, ok := x.(int8); ok {
			return int(v), nil
		}
		return x, err
	case uint:
		var e2 uint
		switch y := expr2.(type) {
		case uint:
			e2 = y
		case ConstantNumber:
			e2, err = y.ToUint()
			if err != nil {
				return nil, err
			}
		default:
			break SWITCH
		}
		x, err := evalBinaryUint64(uint64(e1), op, uint64(e2))
		if v, ok := x.(uint64); ok {
			return uint(v), nil
		}
		return x, err
	case uint64:
		var e2 uint64
		switch y := expr2.(type) {
		case uint64:
			e2 = y
		case ConstantNumber:
			e2, err = y.ToUint64()
			if err != nil {
				return nil, err
			}
		default:
			break SWITCH
		}
		return evalBinaryUint64(e1, op, e2)
	case uint32:
		var e2 uint32
		switch y := expr2.(type) {
		case uint32:
			e2 = y
		case ConstantNumber:
			e2, err = y.ToUint32()
			if err != nil {
				return nil, err
			}
		default:
			break SWITCH
		}
		x, err := evalBinaryUint64(uint64(e1), op, uint64(e2))
		if v, ok := x.(uint64); ok {
			return uint32(v), nil
		}
		return x, err
	case uint16:
		var e2 uint16
		switch y := expr2.(type) {
		case uint16:
			e2 = y
		case ConstantNumber:
			e2, err = y.ToUint16()
			if err != nil {
				return nil, err
			}
		default:
			break SWITCH
		}
		x, err := evalBinaryUint64(uint64(e1), op, uint64(e2))
		if v, ok := x.(uint64); ok {
			return uint16(v), nil
		}
		return x, err
	case uint8:
		var e2 uint8
		switch y := expr2.(type) {
		case uint8:
			e2 = y
		case ConstantNumber:
			e2, err = y.ToUint8()
			if err != nil {
				return nil, err
			}
		default:
			break SWITCH
		}
		x, err := evalBinaryUint64(uint64(e1), op, uint64(e2))
		if v, ok := x.(uint64); ok {
			return uint8(v), nil
		}
		return x, err
	case float64:
		var e2 float64
		switch y := expr2.(type) {
		case float64:
			e2 = y
		case ConstantNumber:
			e2 = y.ToFloat64()
		default:
			break SWITCH
		}
		return evalBinaryFloat64(e1, op, e2)
	case float32:
		var e2 float32
		switch y := expr2.(type) {
		case float32:
			e2 = y
		case ConstantNumber:
			e2 = y.ToFloat32()
		default:
			break SWITCH
		}
		x, err := evalBinaryFloat64(float64(e1), op, float64(e2))
		if v, ok := x.(float64); ok {
			return float32(v), nil
		}
		return x, err
	case ConstantNumber:
		switch y := expr2.(type) {
		case int:
			x, err := e1.ToInt()
			if err != nil {
				return nil, err
			}
			z, err := evalBinaryInt64(int64(x), op, int64(y))
			if v, ok := z.(int64); ok {
				return int(v), nil
			}
			return z, err
		case int64:
			x, err := e1.ToInt64()
			if err != nil {
				return nil, err
			}
			return evalBinaryInt64(x, op, y)
		case int32:
			x, err := e1.ToInt32()
			if err != nil {
				return nil, err
			}
			z, err := evalBinaryInt64(int64(x), op, int64(y))
			if v, ok := z.(int64); ok {
				return int32(v), nil
			}
			return z, err
		case int16:
			x, err := e1.ToInt16()
			if err != nil {
				return nil, err
			}
			z, err := evalBinaryInt64(int64(x), op, int64(y))
			if v, ok := z.(int64); ok {
				return int16(v), nil
			}
			return z, err
		case int8:
			x, err := e1.ToInt8()
			if err != nil {
				return nil, err
			}
			z, err := evalBinaryInt64(int64(x), op, int64(y))
			if v, ok := z.(int64); ok {
				return int8(v), nil
			}
			return z, err
		case uint:
			x, err := e1.ToUint()
			if err != nil {
				return nil, err
			}
			z, err := evalBinaryUint64(uint64(x), op, uint64(y))
			if v, ok := z.(uint64); ok {
				return uint(v), nil
			}
			return z, err
		case uint64:
			x, err := e1.ToUint64()
			if err != nil {
				return nil, err
			}
			return evalBinaryUint64(uint64(x), op, uint64(y))
		case uint32:
			x, err := e1.ToUint32()
			if err != nil {
				return nil, err
			}
			z, err := evalBinaryUint64(uint64(x), op, uint64(y))
			if v, ok := z.(uint64); ok {
				return uint32(v), nil
			}
			return z, err
		case uint16:
			x, err := e1.ToUint16()
			if err != nil {
				return nil, err
			}
			z, err := evalBinaryUint64(uint64(x), op, uint64(y))
			if v, ok := z.(uint64); ok {
				return uint16(v), nil
			}
			return z, err
		case uint8:
			x, err := e1.ToUint8()
			if err != nil {
				return nil, err
			}
			z, err := evalBinaryUint64(uint64(x), op, uint64(y))
			if v, ok := z.(uint64); ok {
				return uint8(v), nil
			}
			return z, err
		case float64:
			x := e1.ToFloat64()
			return evalBinaryFloat64(x, op, y)
		case float32:
			x := e1.ToFloat32()
			z, err := evalBinaryFloat64(float64(x), op, float64(y))
			if v, ok := z.(float64); ok {
				return float32(v), nil
			}
			return z, err
		case ConstantNumber:
			return e1.BinaryOp(op, y)
		case CustomNumber:
			e1, err := y.New().Convert(e1)
			if err != nil {
				return nil, err
			}
			return e1.BinaryOp(e1, op, y), nil
		}
	case CustomNumber:
		var e2 CustomNumber
		switch y := expr2.(type) {
		case CustomNumber:
			e2 = y
		case ConstantNumber:
			e2, err = e1.New().Convert(e2)
			if err != nil {
				return nil, err
			}
		default:
			break SWITCH
		}
		if reflect.TypeOf(e1) == reflect.TypeOf(e2) {
			switch op {
			case ast.OperatorEqual:
				return !e1.IsNaN() && !e2.IsNaN() && e1.Cmp(e2) == 0, nil
			case ast.OperatorNotEqual:
				return e1.IsNaN() || e2.IsNaN() || e1.Cmp(e2) != 0, nil
			case ast.OperatorLess:
				return !e1.IsNaN() && !e2.IsNaN() && e1.Cmp(e2) < 0, nil
			case ast.OperatorLessOrEqual:
				return !e1.IsNaN() && !e2.IsNaN() && e1.Cmp(e2) <= 0, nil
			case ast.OperatorGreater:
				return !e1.IsNaN() && !e2.IsNaN() && e1.Cmp(e2) > 0, nil
			case ast.OperatorGreaterOrEqual:
				return !e1.IsNaN() && !e2.IsNaN() && e1.Cmp(e2) >= 0, nil
			case ast.OperatorAddition:
				return e1.New().BinaryOp(e1, ast.OperatorAddition, e2), nil
			case ast.OperatorSubtraction:
				return e1.New().BinaryOp(e1, ast.OperatorSubtraction, e2), nil
			case ast.OperatorMultiplication:
				return e1.New().BinaryOp(e1, ast.OperatorMultiplication, e2), nil
			case ast.OperatorDivision:
				return e1.New().BinaryOp(e1, ast.OperatorDivision, e2), nil
			case ast.OperatorModulo:
				return e1.New().BinaryOp(e1, ast.OperatorModulo, e2), nil
			}
		}
	case bool:
		switch op {
		case ast.OperatorAnd:
			if !e1 {
				return false, nil
			}
		case ast.OperatorOr:
			if e1 {
				return true, nil
			}
		}
		expr2, err = r.eval(op2)
		if err != nil {
			return nil, err
		}
		if e2, ok := expr2.(bool); ok {
			switch op {
			case ast.OperatorEqual:
				return e1 == e2, nil
			case ast.OperatorNotEqual:
				return e1 != e2, nil
			case ast.OperatorAnd:
				return e2, nil
			case ast.OperatorOr:
				return e2, nil
			}
		}
	}

	uNil1 := expr1 == nil && r.isBuiltin("nil", op1)
	uNil2 := expr2 == nil && r.isBuiltin("nil", op2)
	if uNil1 && uNil2 {
		return false, fmt.Errorf("operator %s not defined on nil", op)
	}

	switch op {
	case ast.OperatorEqual:
		if uNil2 {
			return isNil(expr1), nil
		}
		if uNil1 {
			return isNil(expr2), nil
		}
		return false, nil
	case ast.OperatorNotEqual:
		if uNil2 {
			return !isNil(expr1), nil
		}
		if uNil1 {
			return !isNil(expr2), nil
		}
		return true, nil
	}

	// TODO(marco): manage the other cases.

	return nil, fmt.Errorf("mismatched types %s and %s", typeof(expr1), typeof(expr2))
}

// isNil indicates if v is nil or the value is nil.
func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Ptr, reflect.UnsafePointer, reflect.Slice:
		return rv.IsNil()
	}
	return false
}

// evalBytes evaluates a bytes expression and returns its value.
func (r *rendering) evalBytes(node *ast.CompositeLiteral) interface{} {
	if node.KeyValues == nil {
		return make(Bytes, 0)
	}
	indexValue, maxIndex := r.indexValueMap(node.KeyValues)
	elements := make(Bytes, maxIndex+1)
	for i, expr := range indexValue {
		var err error
		v := r.evalExpression(expr)
		switch n := v.(type) {
		case byte:
			elements[i] = n
		case ConstantNumber:
			elements[i], err = n.ToUint8()
		default:
			err = fmt.Errorf("cannot use %s (type %s) as type byte", expr, typeof(v))
		}
		if err != nil {
			panic(r.errorf(node, "%s in bytes literal", err))
		}
	}
	return elements
}

// evalSelector evaluates a selector expression and returns its value.
func (r *rendering) evalSelector(node *ast.Selector) interface{} {
	v, ok, err := r.evalSelector2(node)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic(r.errorf(node, "field %q does not exist", node.Ident))
	}
	return v
}

// evalSelector2 evaluates a selector in a two-values context. If its
// identifier exists it returns the value and true, otherwise it returns nil
// and false.
// TODO (Gianluca): structKeys is currently used for non struct types too.
func (r *rendering) evalSelector2(node *ast.Selector) (interface{}, bool, error) {
	if ident, ok := node.Expr.(*ast.Identifier); ok {
		v, ok := r.variable(ident.Name)
		if ok {
			if reflect.TypeOf(v).Kind() == reflect.Struct {
				rv, err := r.referenceInScope(ident.Name)
				if err != nil {
					panic(err) // TODO (Gianluca): to review.
				}
				keys := structKeys(rv)
				if keys == nil {
					return nil, false, r.errorf(node, "%s undefined (type %s has no field or method %s)", node, rv.Type(), node.Ident)
				}
				sk, ok := keys[node.Ident]
				if !ok {
					return nil, false, nil
				}
				return sk.value(rv), true, nil
			}
		}
	}
	value, err := r.eval(node.Expr)
	if err != nil {
		return nil, false, err
	}
	if p, ok := value.(*Package); ok {
		v, ok := p.Declarations[node.Ident]
		if !ok {
			if fc, _ := utf8.DecodeRuneInString(node.Ident); !unicode.Is(unicode.Lu, fc) {
				return nil, false, r.errorf(node, "cannot refer to unexported name %s", node)
			}
			return nil, false, r.errorf(node, "undefined: %s", node)
		}
		if _, ok := v.(reflect.Type); ok {
			return v, true, nil
		}
		if reflect.TypeOf(v).Kind() == reflect.Ptr {
			return reflect.ValueOf(v).Elem().Interface(), true, nil
		}
		if c, ok := v.(pkgConstant); ok {
			// TODO (Gianluca): currently ignores types.
			switch v := c.value.(type) {
			// TODO (Gianluca): add missing cases.
			case int:
				return newConstantInt(big.NewInt(int64(v))), true, nil
			case float64:
				return newConstantFloat(big.NewFloat(v)), true, nil
			default:
				return v, true, nil
			}
		}
		return v, true, nil
	}
	rv := reflect.ValueOf(value)
	keys := make(map[string]structKey)
	if rv.Kind() == reflect.Struct || (rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.Struct) {
		keys = structKeys(rv)
	} else {
		n := rv.NumMethod()
		for i := 0; i < n; i++ {
			met := rv.Type().Method(i)
			keys[met.Name] = structKey{index: i, isMethod: true}
		}
	}
	if keys == nil {
		return nil, false, r.errorf(node, "%s undefined (type %s has no field or method %s)", node, typeof(value), node.Ident)
	}
	sk, ok := keys[node.Ident]
	if !ok {
		return nil, false, nil
	}
	return sk.value(rv), true, nil
}

// evalTypeAssertion evaluates a type assertion.
func (r *rendering) evalTypeAssertion(node *ast.TypeAssertion) interface{} {
	val := r.evalExpression(node.Expr)
	if node.Type == nil { // .(type) assertion.
		return val
	}
	typ, err := r.evalType(node.Type, noEllipses)
	if err != nil {
		panic(err)
	}
	has, err := hasType(val, typ)
	if err != nil {
		panic(r.errorf(node.Expr, "%s", err))
	}
	if !has {
		panic(r.errorf(node.Type, "%s is %s, not %s", node.Expr, reflect.TypeOf(val), typ))
	}
	return val
}

// indexValueMap takes a sparse list of expression-indexed values and returns a
// map with evaluated indexes as keys and the corresponding expressions as
// values. It also returns the maximum index found.
func (r *rendering) indexValueMap(indexValues []ast.KeyValue) (map[int]ast.Expression, int) {
	indexValue := make(map[int]ast.Expression, len(indexValues))
	maxIndex := 0
	prevIndex := -1
	var i int
	var err error
	for _, kv := range indexValues {
		if kv.Key == nil {
			i = prevIndex + 1
		} else {
			i, err = r.sliceIndex(kv.Key)
			if err != nil {
				panic(err)
			}
		}
		if i > maxIndex {
			maxIndex = i
		}
		indexValue[i] = kv.Value
		prevIndex = i
	}
	return indexValue, maxIndex
}

// evalCompositeLiteral evaluates and returns the value of the composite literal
// node. It panics on error.
//
// TODO (Gianluca): use more efficient reflect functions to create slices and
// maps when possibile.
//
func (r *rendering) evalCompositeLiteral(node *ast.CompositeLiteral, outerType reflect.Type) interface{} {
	var err error
	var indexValue map[int]ast.Expression
	var maxIndex int
	var typ reflect.Type
	if node.Type != nil { // Composite literal with explicit type.
		length := noEllipses
		if _, ok := node.Type.(*ast.ArrayType); ok {
			// Composite literal has an array type, so the maximum index must be
			// determined to create ellipses and to check size matching.
			indexValue, maxIndex = r.indexValueMap(node.KeyValues)
			length = maxIndex + 1
		}
		typ, err = r.evalType(node.Type, length)
		if err != nil {
			panic(err)
		}
	} else { // Composite literal with implicit type.
		typ = outerType
	}
	// Generic Go types.
	switch typ.Kind() {
	case reflect.Array: // [n]T{...}
		if node.KeyValues == nil {
			return reflect.New(typ).Elem().Interface()
		}
		array := reflect.New(typ).Elem()
		elemType := array.Type().Elem()
		indexValue, _ := r.indexValueMap(node.KeyValues)
		for i, v := range indexValue {
			var refValue reflect.Value
			var evalValue interface{}
			if cl, ok := v.(*ast.CompositeLiteral); ok {
				evalValue = r.evalCompositeLiteral(cl, typ.Elem())
			} else {
				evalValue = r.evalExpression(v)
			}
			if cn, ok := evalValue.(ConstantNumber); ok {
				typed, err := cn.ToType(elemType)
				if err != nil {
					panic(err)
				}
				refValue = reflect.ValueOf(typed)
			} else {
				refValue = reflect.ValueOf(evalValue)
			}
			array.Index(i).Set(refValue)
		}
		return array.Interface()
	case reflect.Slice: // []T{...}
		if typ.Elem() == reflect.TypeOf(byte(0)) {
			return r.evalBytes(node)
		}
		if node.KeyValues == nil {
			return reflect.MakeSlice(typ, 0, 0).Interface()
		}
		indexValue, maxIndex = r.indexValueMap(node.KeyValues)
		slice := reflect.MakeSlice(typ, maxIndex+1, maxIndex+1)
		for i, v := range indexValue {
			var evalValue interface{}
			if cl, ok := v.(*ast.CompositeLiteral); ok {
				evalValue = r.evalCompositeLiteral(cl, typ.Elem())
			} else {
				evalValue = r.evalExpression(v)
			}
			if cn, ok := evalValue.(ConstantNumber); ok {
				var err error
				evalValue, err = cn.ToType(typ.Elem())
				if err != nil {
					panic(err)
				}
			}
			slice.Index(i).Set(reflect.ValueOf(evalValue))
		}
		return slice.Interface()
	case reflect.Map: // map[T1]T2{...}
		if node.KeyValues == nil {
			return reflect.MakeMap(typ).Interface()
		}
		mapValue := reflect.MakeMap(typ)
		for _, kv := range node.KeyValues {
			key, err := r.mapIndex(kv.Key, typ.Key())
			if err != nil {
				panic(err)
			}
			var evalValue interface{}
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				evalValue = r.evalCompositeLiteral(cl, typ.Elem())
			} else {
				evalValue = r.evalExpression(kv.Value)
			}
			var refValue reflect.Value
			if cn, ok := evalValue.(ConstantNumber); ok {
				typed, err := cn.ToType(typ.Elem())
				if err != nil {
					panic(err)
				}
				refValue = reflect.ValueOf(typed)
			} else {
				refValue = reflect.ValueOf(evalValue)
			}
			mapValue.SetMapIndex(reflect.ValueOf(key), refValue)
		}
		return mapValue.Interface()
	case reflect.Ptr: // *T{...}
		cl := r.evalCompositeLiteral(node, typ.Elem())
		return refToCopy(cl).Interface()
	case reflect.Struct: // T{...}
		if node.KeyValues == nil {
			return reflect.New(typ).Elem().Interface()
		}
		s := reflect.New(typ).Elem()
		// Checks if struct composite literal contains explicit fields or not.
		var explicitFields bool
		if len(node.KeyValues) > 0 {
			explicitFields = node.KeyValues[0].Key != nil
		}
		if explicitFields {
			for _, fv := range node.KeyValues {
				val := r.evalExpression(fv.Value)
				var refExpr reflect.Value
				var field reflect.StructField
				var ident *ast.Identifier
				if cn, okcn := val.(ConstantNumber); okcn {
					field, _ = typ.FieldByName(fv.Key.(*ast.Identifier).Name)
					typed, err := cn.ToType(field.Type)
					if err != nil {
						panic(err)
					}
					refExpr = reflect.ValueOf(typed)
				} else {
					refExpr = reflect.ValueOf(val)
				}
				s.FieldByName(ident.Name).Set(refExpr)
			}
		} else {
			for i, kv := range node.KeyValues {
				val := r.evalExpression(kv.Value)
				var refExpr reflect.Value
				if cn, ok := val.(ConstantNumber); ok {
					typed, err := cn.ToType(typ.Field(i).Type)
					if err != nil {
						panic(err)
					}
					refExpr = reflect.ValueOf(typed)
				} else {
					refExpr = reflect.ValueOf(val)
				}
				s.Field(i).Set(refExpr)
			}
		}
		return s.Interface()
	}
	return nil
}

const noEllipses = -1

// evalType evaluates an expression as type. If expr is an array type, length
// specifies the number of elements.
func (r *rendering) evalType(expr ast.Expression, length int) (typ reflect.Type, err error) {
	var value interface{}
	switch e := expr.(type) {
	case *ast.Identifier:
		for i := len(r.vars) - 1; i >= 0; i-- {
			if r.vars[i] != nil {
				if i == 0 && e.Name == "len" {
					return nil, r.errorf(e, "use of builtin len not in function call")
				}
				if v, ok := r.vars[i][e.Name]; ok {
					if typ, ok = v.(reflect.Type); ok && i <= 1 {
						return typ, nil
					}
					return nil, r.errorf(e, "%s is not a type", expr)
				}
			}
		}
		return nil, r.errorf(expr, "undefined: %s", expr)
	case *ast.Selector:
		if ident, ok := e.Expr.(*ast.Identifier); ok {
			v2 := r.evalIdentifier(ident)
			if pkg, ok := v2.(Package); ok {
				value, ok = pkg.Declarations[e.Ident]
				if !ok {
					if fc, _ := utf8.DecodeRuneInString(e.Ident); !unicode.Is(unicode.Lu, fc) {
						return nil, r.errorf(expr, "cannot refer to unexported name %s", expr)
					}
					return nil, r.errorf(expr, "undefined: %s", expr)
				}
				typ, ok = value.(reflect.Type)
				if !ok {
					return nil, r.errorf(expr, "%s is not a type", expr)
				}
				return typ, nil
			}
		}
	case *ast.ArrayType:
		elemType, err := r.evalType(e.ElementType, noEllipses)
		if err != nil {
			panic(err)
		}
		var declLength int
		if e.Len == nil {
			if length == noEllipses {
				return nil, r.errorf(expr, "use of [...] array outside of array literal")
			}
			return reflect.ArrayOf(length, elemType), nil
		}
		l, err := r.eval(e.Len)
		if err != nil {
			return nil, err
		}
		switch l := l.(type) {
		case ConstantNumber:
			n, err := l.ToType(intType)
			if err != nil {
				return nil, err
			}
			declLength = n.(int)
			if declLength < 0 {
				return nil, r.errorf(expr, "array bound must be non-negative")
			}
		default:
			return nil, r.errorf(expr, "non-constant array bound %s", l)
		}
		if declLength < length {
			return nil, r.errorf(expr, "array index %d out of bounds [0:%d]", declLength, declLength)
		}
		return reflect.ArrayOf(declLength, elemType), nil
	case *ast.SliceType:
		elemType, err := r.evalType(e.ElementType, noEllipses)
		if err != nil {
			panic(err)
		}
		return reflect.SliceOf(elemType), nil
	case *ast.MapType:
		var keyType, valueType reflect.Type
		keyType, err = r.evalType(e.KeyType, noEllipses)
		if err != nil {
			panic(err)
		}
		valueType, err = r.evalType(e.ValueType, noEllipses)
		if err != nil {
			panic(err)
		}
		defer func() {
			if rec := recover(); rec != nil {
				err = r.errorf(expr, "invalid map key type %s", keyType)
			}
		}()
		return reflect.MapOf(keyType, valueType), nil
	case *ast.UnaryOperator:
		if e.Operator() != ast.OperatorMultiplication {
			panic(fmt.Errorf("%T is not a type", expr))
		}
		elemTyp, err := r.evalType(e.Expr, noEllipses)
		if err != nil {
			panic(err)
		}
		return reflect.PtrTo(elemTyp), nil
	default:
		panic(fmt.Errorf("%T is not a type", expr))
	}
	return nil, r.errorf(expr, "%s is not a type", expr)
}

var errNotAssignable = errors.New("not assignable")

// asAssignableTo returns v with type typ.
//
// If v is a constant number and it is not representable by a value of typ,
// returns errors errConstantTruncated or errConstantOverflow. If v is not
// assignable to typ returns errNotAssignable error.
func asAssignableTo(v interface{}, typ reflect.Type) (interface{}, error) {
	switch v := v.(type) {
	case nil:
		switch typ.Kind() {
		case reflect.Ptr, reflect.Func, reflect.Slice, reflect.Map, reflect.Chan, reflect.Interface:
		default:
			return nil, errNotAssignable
		}
		return nil, nil
	case ConstantNumber:
		return v.ToType(typ)
	case HTML:
		if typ != htmlType && typ != stringType {
			return nil, errNotAssignable
		}
		return string(v), nil
	}
	if !reflect.TypeOf(v).AssignableTo(typ) {
		return nil, errNotAssignable
	}
	return v, nil
}

// hasType indicates if v has type typ.
func hasType(v interface{}, typ reflect.Type) (bool, error) {
	t := reflect.TypeOf(v)
	if typ.Kind() == reflect.Interface {
		return t.Implements(typ), nil
	}
	if t == constantNumberType {
		n, err := v.(ConstantNumber).ToTyped()
		if err != nil {
			return false, err
		}
		return reflect.TypeOf(n) == typ, nil

	}
	return t == typ || t == htmlType && typ == stringType, nil
}

// evalFunc evaluates a function literal expression in a single context.
func (r *rendering) evalFunc(node *ast.Func) interface{} {
	return function{r.path, node}
}

// evalIndex evaluates an index expression in a single context.
func (r *rendering) evalIndex(node *ast.Index) interface{} {
	v, _, err := r.evalIndex2(node, 1)
	if err != nil {
		panic(err)
	}
	return v
}

// evalIndex2 evaluates an index expression in a single ( n == 1 ) or
// two-values ( n == 2 ) context. If its index exists, returns the value and
// true, otherwise it returns nil and false.
func (r *rendering) evalIndex2(node *ast.Index, n int) (interface{}, bool, error) {
	v, err := r.eval(node.Expr)
	if err != nil {
		return nil, false, err
	}
	if v == nil {
		if r.isBuiltin("nil", node.Expr) {
			return nil, false, r.errorf(node.Expr, "use of untyped nil")
		}
		return nil, false, r.errorf(node, "index out of range")
	}
	checkSlice := func(length int) (int, error) {
		if n == 2 {
			return 0, r.errorf(node, "assignment mismatch: 2 variables but 1 values")
		}
		i, err := r.sliceIndex(node.Index)
		if err != nil {
			return 0, err
		}
		if i >= length {
			return 0, r.errorf(node, "index out of range")
		}
		return i, nil
	}
	switch vv := v.(type) {
	case string:
		if n == 2 {
			return nil, false, r.errorf(node, "assignment mismatch: 2 variables but 1 values")
		}
		i, err := r.sliceIndex(node.Index)
		if err != nil {
			return nil, false, err
		}
		if i >= len(vv) {
			return nil, false, r.errorf(node, "index out of range")
		}
		return vv[i], true, nil
	case HTML:
		if n == 2 {
			return nil, false, r.errorf(node, "assignment mismatch: 2 variables but 1 values")
		}
		i, err := r.sliceIndex(node.Index)
		if err != nil {
			return nil, false, err
		}
		if i >= len(vv) {
			return nil, false, r.errorf(node, "index out of range")
		}
		return vv[i], true, nil
	case map[interface{}]interface{}:
		k, err := r.mapIndex(node.Index, interfaceType)
		if err != nil {
			return nil, false, err
		}
		u, ok := vv[k]
		return u, ok, nil
	case map[string]interface{}:
		k, err := r.mapIndex(node.Index, stringType)
		if err != nil {
			return nil, false, err
		}
		if s, ok := k.(string); ok {
			if u, ok := vv[s]; ok {
				return u, true, nil
			}
		}
		return nil, false, nil
	case map[string]string:
		k, err := r.mapIndex(node.Index, stringType)
		if err != nil {
			return nil, false, err
		}
		if s, ok := k.(string); ok {
			if u, ok := vv[s]; ok {
				return u, true, nil
			}
		}
		return "", false, nil
	case map[string]HTML:
		k, err := r.mapIndex(node.Index, stringType)
		if err != nil {
			return nil, false, err
		}
		if s, ok := k.(string); ok {
			if u, ok := vv[s]; ok {
				return u, true, nil
			}
		}
		return HTML(""), false, nil
	case map[string]int:
		k, err := r.mapIndex(node.Index, stringType)
		if err != nil {
			return nil, false, err
		}
		if s, ok := k.(string); ok {
			if u, ok := vv[s]; ok {
				return u, true, nil
			}
		}
		return 0, false, nil
	case map[string]bool:
		k, err := r.mapIndex(node.Index, stringType)
		if err != nil {
			return nil, false, err
		}
		if s, ok := k.(string); ok {
			if u, ok := vv[s]; ok {
				return u, true, nil
			}
		}
		return false, false, nil
	case []interface{}:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	case []string:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	case []HTML:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	case []int:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	case []byte:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	case []bool:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	}
	var rv = reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr:
		if rv.Elem().Kind() != reflect.Array {
			return nil, false, r.errorf(node, "invalid operation: %s (type %s does not support indexing)", node, typeof(v))
		}
		i, err := checkSlice(rv.Elem().Len())
		if err != nil {
			return nil, false, err
		}
		return rv.Elem().Index(i).Interface(), true, nil
	case reflect.String:
		if n == 2 {
			return nil, false, r.errorf(node, "assignment mismatch: 2 variables but 1 values")
		}
		i, err := r.sliceIndex(node.Index)
		if err != nil {
			return nil, false, err
		}
		if i >= rv.Len() {
			return nil, false, r.errorf(node, "index out of range")
		}
		return rv.Index(i).Interface(), true, nil
	case reflect.Map:
		k, err := r.mapIndex(node.Index, rv.Type().Key())
		if err != nil {
			return nil, false, err
		}
		mk := reflectValueNil
		if k != nil {
			mk = reflect.ValueOf(k)
		}
		rt := rv.Type()
		if mk.Type().AssignableTo(rt.Key()) {
			if v := rv.MapIndex(mk); v.IsValid() {
				return v.Interface(), true, nil
			}
		}
		return reflect.Zero(rt.Key()).Interface(), false, nil
	case reflect.Slice:
		i, err := checkSlice(rv.Len())
		if err != nil {
			return nil, false, err
		}
		return rv.Index(i).Interface(), true, nil
	case reflect.Array:
		i, err := checkSlice(rv.Len())
		if err != nil {
			return nil, false, err
		}
		return rv.Index(i).Interface(), true, nil
	}
	return nil, false, r.errorf(node, "invalid operation: %s (type %s does not support indexing)", node, typeof(v))
}

// sliceIndex evaluates node as a slice index and returns the value.
func (r *rendering) sliceIndex(node ast.Expression) (i int, err error) {
	switch n := r.evalExpression(node).(type) {
	case int:
		i = n
	case ConstantNumber:
		i, err = n.ToInt()
		if err != nil {
			return 0, r.errorf(node, "%s", err)
		}
	default:
		return 0, r.errorf(node, "non-integer slice index %v", node)
	}
	if i < 0 {
		return 0, r.errorf(node, "invalid slice index %d (index must be non-negative)", i)
	}
	return
}

// mapIndex evaluates node as a map index of type typ and returns the value.
func (r *rendering) mapIndex(node ast.Expression, typ reflect.Type) (interface{}, error) {
	var key interface{}
	if cl, ok := node.(*ast.CompositeLiteral); ok {
		key = r.evalCompositeLiteral(cl, typ)
	} else {
		key = r.evalExpression(node)
	}
	mapKeyError := func(keyType reflect.Type) error {
		return r.errorf(node, "cannot use %s (type %s) as type %s in map key", node, keyType, typ)
	}
	switch k := key.(type) {
	case nil:
	case string, HTML, int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8, float64, float32, bool:
		keyType := reflect.TypeOf(key)
		if typ.Kind() == reflect.Interface {
			if !keyType.Implements(typ) {
				return nil, mapKeyError(keyType)
			}
		} else {
			if keyType != typ {
				return nil, mapKeyError(keyType)
			}
		}
	case ConstantNumber:
		var err error
		key, err = k.ToType(typ)
		if err != nil {
			return nil, r.errorf(node, "%s", err)
		}
	default:
		keyType := reflect.TypeOf(key)
		if !keyType.Comparable() {
			return nil, r.errorf(node, "hash of unhashable type %s", typeof(key))
		}
		if typ.Kind() == reflect.Interface {
			if !keyType.Implements(typ) {
				return nil, mapKeyError(keyType)
			}
		} else {
			if keyType != typ {
				return nil, mapKeyError(keyType)
			}
		}
	}
	return key, nil
}

// evalSlicing evaluates a slice expression and returns its value.
func (r *rendering) evalSlicing(node *ast.Slicing) interface{} {
	var err error
	var l, h int
	if node.Low != nil {
		switch n := r.evalExpression(node.Low).(type) {
		case int:
			l = n
		case ConstantNumber:
			l, err = n.ToInt()
			if err != nil {
				panic(r.errorf(node.Low, "%s", err))
			}
		default:
			panic(r.errorf(node.Low, "invalid slice index %s (type %s)", node.Low, typeof(n)))
		}
		if l < 0 {
			panic(r.errorf(node.Low, "invalid slice index %d (index must be non-negative)", node.Low))
		}
	}
	if node.High != nil {
		switch n := r.evalExpression(node.High).(type) {
		case int:
			h = n
		case ConstantNumber:
			h, err = n.ToInt()
			if err != nil {
				panic(r.errorf(node.High, "%s", err))
			}
		default:
			panic(r.errorf(node.High, "invalid slice index %s (type %s)", node.High, typeof(n)))
		}
		if h < 0 {
			panic(r.errorf(node.High, "invalid slice index %d (index must be non-negative)", node.High))
		}
		if l > h {
			panic(r.errorf(node.Low, "invalid slice index: %d > %d", l, h))
		}
	}
	var v = r.evalExpression(node.Expr)
	if v == nil {
		if r.isBuiltin("nil", node.Expr) {
			panic(r.errorf(node.Expr, "use of untyped nil"))
		}
		panic(r.errorf(node, "slice bounds out of range"))
	}
	h2 := func(length int) int {
		if node.High == nil {
			h = length
		} else if h > length {
			panic(r.errorf(node.High, "slice bounds out of range"))
		}
		return h
	}
	switch v := v.(type) {
	case string:
		return v[l:h2(len(v))]
	case HTML:
		return v[l:h2(len(v))]
	case []interface{}:
		return v[l:h2(len(v))]
	case []string:
		return v[l:h2(len(v))]
	case []HTML:
		return v[l:h2(len(v))]
	case []int:
		return v[l:h2(len(v))]
	case []byte:
		return v[l:h2(len(v))]
	case []bool:
		return v[l:h2(len(v))]
	}
	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.String:
		return rv.Slice(l, h2(rv.Len())).Interface()
	case reflect.Slice:
		return rv.Slice(l, h2(rv.Len())).Interface()
	}
	panic(r.errorf(node, "cannot slice %s (type %s)", node.Expr, typeof(v)))
}

// evalIdentifier evaluates an identifier expression and returns its value.
func (r *rendering) evalIdentifier(node *ast.Identifier) interface{} {
	for i := len(r.vars) - 1; i >= 0; i-- {
		if r.vars[i] != nil {
			if i == 0 && node.Name == "len" {
				panic(r.errorf(node, "use of builtin len not in function call"))
			}
			if v, ok := r.vars[i][node.Name]; ok {
				if vv, ok := v.(reference); ok {
					return vv.rv.Interface()
				}
				return v
			}
		}
	}
	panic(r.errorf(node, "undefined: %s", node.Name))
}

// isBuiltin indicates if expr is the builtin with the given name.
func (r *rendering) isBuiltin(name string, expr ast.Expression) bool {
	if n, ok := expr.(*ast.Identifier); ok {
		if n.Name != name {
			return false
		}
		for i := len(r.vars) - 1; i >= 0; i-- {
			if r.vars[i] != nil {
				if _, ok := r.vars[i][name]; ok {
					return i == 0
				}
			}
		}
	}
	return false
}

// zeroOf returns the zero value of the type typ.
func zeroOf(typ reflect.Type) interface{} {
	return reflect.Zero(typ)
}

// structs maintains the association between the field names of a struct,
// as they are called in the template, and the field index in the struct.
var structs = struct {
	keys map[reflect.Type]map[string]structKey
	sync.RWMutex
}{map[reflect.Type]map[string]structKey{}, sync.RWMutex{}}

// structKey represents the fields of a struct.
type structKey struct {
	isMethod bool
	index    int
}

func (sk structKey) value(st reflect.Value) interface{} {
	if sk.isMethod {
		return st.Method(sk.index).Interface()
	}
	st = reflect.Indirect(st)
	return st.Field(sk.index).Interface()
}

func structKeys(st reflect.Value) map[string]structKey {
	typ := st.Type()
	structs.RLock()
	keys, ok := structs.keys[typ]
	structs.RUnlock()
	if ok {
		return keys
	}
	ityp := typ
	kind := st.Kind()
	if kind == reflect.Ptr {
		st = st.Elem()
		kind = st.Kind()
		ityp = st.Type()
	}
	if kind != reflect.Struct {
		return nil
	}
	structs.Lock()
	keys, ok = structs.keys[typ]
	if ok {
		structs.Unlock()
		return keys
	}
	keys = map[string]structKey{}
	n := ityp.NumField()
	for i := 0; i < n; i++ {
		fieldType := ityp.Field(i)
		if fieldType.PkgPath != "" {
			continue
		}
		name := fieldType.Name
		if tag, ok := fieldType.Tag.Lookup("scrigo"); ok {
			name = parseVarTag(tag)
			if name == "" {
				structs.Unlock()
				panic(fmt.Errorf("scrigo: invalid tag of field %q", fieldType.Name))
			}
		}
		keys[name] = structKey{index: i}
	}
	n = typ.NumMethod()
	for i := 0; i < n; i++ {
		name := typ.Method(i).Name
		if _, ok := keys[name]; !ok {
			keys[name] = structKey{index: i, isMethod: true}
		}
	}
	structs.keys[typ] = keys
	structs.Unlock()
	return keys
}

// parseVarTag parses the tag of a struct field and returns its name.
func parseVarTag(tag string) string {
	sp := strings.SplitN(tag, ",", 2)
	if len(sp) == 0 {
		return ""
	}
	name := sp[0]
	if name == "" {
		return ""
	}
	for _, r := range name {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return ""
		}
	}
	return name
}
