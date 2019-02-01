// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"errors"
	"fmt"
	"go/constant"
	"go/token"
	"math/big"
	"reflect"
	"strconv"

	"open2b/template/ast"
)

var emptyInterfaceType = reflect.TypeOf(new(interface{})).Elem()

var errDivisionByZero = errors.New("division by zero")
var errFloatModulo = errors.New("illegal constant expression: floating-point % operation")

type errConstantConversion struct {
	num ConstantNumber
	typ string
}

func (e errConstantConversion) Error() string {
	return fmt.Sprintf("cannot use %s (type %s) as type %s", e.num.val.String(), e.num.typ, e.typ)
}

type errConstantTruncated struct {
	num ConstantNumber
}

func (e errConstantTruncated) Error() string {
	return fmt.Sprintf("constant %s truncated to integer", e.num.val.String())
}

type errConstantOverflow struct {
	num ConstantNumber
	typ string
}

func (e errConstantOverflow) Error() string {
	return fmt.Sprintf("constant %s overflows %s", e.num.val.String(), e.typ)
}

type ConstantNumberType int

const (
	TypeInt ConstantNumberType = iota
	TypeRune
	TypeFloat64
)

func (t ConstantNumberType) String() string {
	switch t {
	case TypeInt:
		return "int"
	case TypeRune:
		return "rune"
	case TypeFloat64:
		return "float64"
	}
	panic("unknown type")
}

type ConstantNumber struct {
	val constant.Value
	typ ConstantNumberType
}

func newConstantInt(n *big.Int) ConstantNumber {
	return ConstantNumber{constant.MakeFromLiteral(n.String(), token.INT, 0), TypeInt}
}

func newConstantRune(r rune) ConstantNumber {
	return ConstantNumber{constant.MakeInt64(int64(r)), TypeRune}
}

func newConstantFloat(n *big.Float) ConstantNumber {
	return ConstantNumber{constant.MakeFromLiteral(n.Text('f', -1), token.FLOAT, 0), TypeFloat64}
}

func (n ConstantNumber) truncatedOrOverflow(typ string) error {
	if n.val.Kind() == constant.Float {
		return errConstantTruncated{num: n}
	}
	return errConstantOverflow{n, typ}
}

func (n ConstantNumber) ToInt() (int, error) {
	x, ok := constant.Int64Val(constant.ToInt(n.val))
	if !ok || int64(int(x)) != x {
		return 0, n.truncatedOrOverflow("int")
	}
	return int(x), nil
}

func (n ConstantNumber) ToInt64() (int64, error) {
	x, ok := constant.Int64Val(constant.ToInt(n.val))
	if !ok {
		return 0, n.truncatedOrOverflow("int64")
	}
	return x, nil
}

func (n ConstantNumber) ToInt32() (int32, error) {
	x, ok := constant.Int64Val(constant.ToInt(n.val))
	if !ok || int64(int32(x)) != x {
		return 0, n.truncatedOrOverflow("int32")
	}
	return int32(x), nil
}

func (n ConstantNumber) ToInt16() (int16, error) {
	x, ok := constant.Int64Val(constant.ToInt(n.val))
	if !ok || int64(int16(x)) != x {
		return 0, n.truncatedOrOverflow("int16")
	}
	return int16(x), nil
}

func (n ConstantNumber) ToInt8() (int8, error) {
	x, ok := constant.Int64Val(constant.ToInt(n.val))
	if !ok || int64(int8(x)) != x {
		return 0, n.truncatedOrOverflow("int8")
	}
	return int8(x), nil
}

func (n ConstantNumber) ToUint() (uint, error) {
	x, ok := constant.Uint64Val(constant.ToInt(n.val))
	if !ok || uint64(uint(x)) != x {
		return 0, n.truncatedOrOverflow("int")
	}
	return uint(x), nil
}

func (n ConstantNumber) ToUint64() (uint64, error) {
	x, ok := constant.Uint64Val(constant.ToInt(n.val))
	if !ok {
		return 0, n.truncatedOrOverflow("uint64")
	}
	return x, nil
}

func (n ConstantNumber) ToUint32() (uint32, error) {
	x, ok := constant.Uint64Val(constant.ToInt(n.val))
	if !ok || uint64(uint32(x)) != x {
		return 0, n.truncatedOrOverflow("uint32")
	}
	return uint32(x), nil
}

func (n ConstantNumber) ToUint16() (uint16, error) {
	x, ok := constant.Uint64Val(constant.ToInt(n.val))
	if !ok || uint64(uint16(x)) != x {
		return 0, n.truncatedOrOverflow("uint16")
	}
	return uint16(x), nil
}

func (n ConstantNumber) ToUint8() (uint8, error) {
	x, ok := constant.Uint64Val(constant.ToInt(n.val))
	if !ok || uint64(uint8(x)) != x {
		return 0, n.truncatedOrOverflow("uint8")
	}
	return uint8(x), nil
}

func (n ConstantNumber) ToFloat64() float64 {
	x, _ := constant.Float64Val(constant.ToFloat(n.val))
	return x
}

func (n ConstantNumber) ToFloat32() float32 {
	x, _ := constant.Float32Val(constant.ToFloat(n.val))
	return x
}

func (n ConstantNumber) ToTyped() (interface{}, error) {
	switch n.typ {
	case TypeInt:
		return n.ToInt()
	case TypeRune:
		return n.ToInt32()
	case TypeFloat64:
		return n.ToFloat64(), nil
	}
	panic("unknown type")
}

// ToType returns n converted to type typ. If it is not convertible to typ,
// returns an errConstantConversion error. If there is an overflow returns an
// errConstantOverflow error.
func (n ConstantNumber) ToType(typ reflect.Type) (interface{}, error) {
	switch typ.Kind() {
	case reflect.Int:
		return n.ToInt()
	case reflect.Int64:
		return n.ToInt64()
	case reflect.Int32:
		return n.ToInt32()
	case reflect.Int16:
		return n.ToInt16()
	case reflect.Int8:
		return n.ToInt8()
	case reflect.Uint:
		return n.ToUint()
	case reflect.Uint64:
		return n.ToUint64()
	case reflect.Uint32:
		return n.ToUint32()
	case reflect.Uint16:
		return n.ToUint16()
	case reflect.Uint8:
		return n.ToUint8()
	case reflect.Float64:
		return n.ToFloat64(), nil
	case reflect.Float32:
		return n.ToFloat32(), nil
	case reflect.Interface:
		if typ == emptyInterfaceType {
			return n.ToTyped()
		}
	}
	return nil, errConstantConversion{n, typ.String()}
}

func (n ConstantNumber) BinaryOp(op ast.OperatorType, y ConstantNumber) (interface{}, error) {
	var v interface{}
	switch op {
	case ast.OperatorEqual:
		v = constant.Compare(n.val, token.EQL, y.val)
	case ast.OperatorNotEqual:
		v = constant.Compare(n.val, token.NEQ, y.val)
	case ast.OperatorLess:
		v = constant.Compare(n.val, token.LSS, y.val)
	case ast.OperatorLessOrEqual:
		v = constant.Compare(n.val, token.LEQ, y.val)
	case ast.OperatorGreater:
		v = constant.Compare(n.val, token.GTR, y.val)
	case ast.OperatorGreaterOrEqual:
		v = constant.Compare(n.val, token.GEQ, y.val)
	case ast.OperatorAddition:
		v = constant.BinaryOp(n.val, token.ADD, y.val)
	case ast.OperatorSubtraction:
		v = constant.BinaryOp(n.val, token.SUB, y.val)
	case ast.OperatorMultiplication:
		v = constant.BinaryOp(n.val, token.MUL, y.val)
	case ast.OperatorDivision:
		if constant.Sign(y.val) == 0 {
			return nil, errDivisionByZero
		}
		if n.typ == TypeFloat64 || y.typ == TypeFloat64 {
			v = constant.BinaryOp(n.val, token.QUO, y.val)
		} else {
			a, _ := new(big.Int).SetString(n.val.ExactString(), 10)
			b, _ := new(big.Int).SetString(y.val.ExactString(), 10)
			v = constant.MakeFromLiteral(a.Quo(a, b).String(), token.INT, 0)
		}
	case ast.OperatorModulo:
		if n.typ == TypeFloat64 || y.typ == TypeFloat64 {
			return nil, errFloatModulo
		}
		if constant.Sign(y.val) == 0 {
			return nil, errDivisionByZero
		}
		v = constant.BinaryOp(n.val, token.REM, y.val)
	}
	if v, ok := v.(constant.Value); ok {
		typ := n.typ
		if typ < y.typ {
			typ = y.typ
		}
		return ConstantNumber{val: v, typ: typ}, nil
	}
	return v, nil
}

func (n ConstantNumber) Neg() ConstantNumber {
	return ConstantNumber{val: constant.UnaryOp(token.SUB, n.val, 0), typ: n.typ}
}

func (n ConstantNumber) Sign() int {
	return constant.Sign(n.val)
}

func (n ConstantNumber) Type() ConstantNumberType {
	return n.typ
}

func (n ConstantNumber) String() (string, error) {
	switch n.typ {
	case TypeInt:
		i, err := n.ToInt()
		if err != nil {
			return "", err
		}
		return strconv.Itoa(i), nil
	case TypeRune:
		r, err := n.ToInt32()
		if err != nil {
			return "", err
		}
		return strconv.QuoteRuneToASCII(r), nil
	case TypeFloat64:
		return strconv.FormatFloat(n.ToFloat64(), 'g', -1, 64), nil
	}
	panic("unknown type")
}
