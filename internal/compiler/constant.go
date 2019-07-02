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
	"strconv"
	"strings"

	"scriggo/internal/compiler/ast"
)

var errNotRepresentable = errors.New("not representable")
var errInvalidOperation = errors.New("invalid operation")
var errDivisionByZero = errors.New("division by zero")
var errComplexDivisionByZero = errors.New("complex division by zero")

// constant represents boolean, string, integer, floating point and complex
// constant values.
type constant interface {

	// String returns a string representation of the constant.
	String() string

	// bool returns the constant as a bool. If the constant is not
	// representable by a bool value, the result is undefined.
	bool() bool

	// string returns the constant as a string. If the constants is not
	// representable by a string value, the result is undefined.
	string() string

	// int64 returns the constant as an int64. If the constant is not
	// representable by an int64 value, the result is undefined.
	int64() int64

	// uint64 returns the constant as an uint64. If the constant is not
	// representable by an uint64 value, the result is undefined.
	uint64() uint64

	// float64 returns the constant as a float64. If the constant is not
	// representable by a float64 value, the result is undefined.
	float64() float64

	// complex128 returns the constant as a complex128. If the constant is not
	// representable by a complex128 value, the result is undefined.
	complex128() complex128

	// unaryOp executes the unary operation op c1 and returns the result.
	// Returns an error if the operation cannot be executed.
	unaryOp(op ast.OperatorType) (constant, error)

	// binaryOp executes the binary operation c1 op c2 and returns the result.
	// Returns an error if the operation cannot be executed. If c1 and c2 are
	// not both booleans, strings or numbers, the behaviour is undefined.
	binaryOp(op ast.OperatorType, c2 constant) (constant, error)

	// representedBy checks if the constant is representable by a value of a
	// given type and returns a constant representing that value. Otherwise
	// returns nil and the error.
	representedBy(typ reflect.Type) (constant, error)

	// equals reports whether the constant is equal to the constant c2.
	equals(c2 constant) bool

	// zero reports whether the constant is the zero value.
	zero() bool

	// real returns the real part of a complex number or the constant itself
	// if it is a number. Panics if the constant is a boolean or a string.
	real() constant

	// imag returns the imaginary part of a complex number or the number
	// constant zero if it is not a complex. Panics if the constant is a
	// boolean or a string.
	imag() constant
}

// boolConst represents a boolean constant.
type boolConst bool

func (c1 boolConst) String() string {
	if c1 {
		return "true"
	}
	return "false"
}

func (c1 boolConst) bool() bool             { return bool(c1) }
func (c1 boolConst) string() string         { return "" }
func (c1 boolConst) int64() int64           { return 0 }
func (c1 boolConst) uint64() uint64         { return 0 }
func (c1 boolConst) float64() float64       { return 0 }
func (c1 boolConst) complex128() complex128 { return 0 }

func (c1 boolConst) unaryOp(op ast.OperatorType) (constant, error) {
	if op != ast.OperatorNot {
		return nil, errInvalidOperation
	}
	return !c1, nil
}

func (c1 boolConst) binaryOp(op ast.OperatorType, c2 constant) (constant, error) {
	b1 := c1
	b2 := c2.(boolConst)
	switch op {
	case ast.OperatorEqual:
		return boolConst(b1 == b2), nil
	case ast.OperatorNotEqual:
		return boolConst(b1 != b2), nil
	case ast.OperatorAndAnd:
		return b1 && b2, nil
	case ast.OperatorOrOr:
		return b1 || b2, nil
	}
	return nil, errInvalidOperation
}

func (c1 boolConst) representedBy(typ reflect.Type) (constant, error) {
	if typ.Kind() == reflect.Bool {
		return c1, nil
	}
	return nil, errNotRepresentable
}

func (c1 boolConst) zero() bool {
	return bool(c1)
}

func (c1 boolConst) real() constant {
	panic("not number constant")
}

func (c1 boolConst) imag() constant {
	panic("not number constant")
}

func (c1 boolConst) equals(c2 constant) bool {
	if c2, ok := c2.(boolConst); ok {
		return c1 == c2
	}
	return false
}

// stringConst represents a string constant.
type stringConst string

func (c1 stringConst) String() string {
	return string(c1)
}

func (c1 stringConst) bool() bool             { return false }
func (c1 stringConst) string() string         { return string(c1) }
func (c1 stringConst) int64() int64           { return 0 }
func (c1 stringConst) uint64() uint64         { return 0 }
func (c1 stringConst) float64() float64       { return 0 }
func (c1 stringConst) complex128() complex128 { return 0 }

func (c1 stringConst) unaryOp(op ast.OperatorType) (constant, error) {
	return nil, errInvalidOperation
}

func (c1 stringConst) binaryOp(op ast.OperatorType, c2 constant) (constant, error) {
	s1 := c1
	s2 := c2.(stringConst)
	switch op {
	case ast.OperatorEqual:
		return boolConst(s1 == s2), nil
	case ast.OperatorNotEqual:
		return boolConst(s1 != s2), nil
	case ast.OperatorLess:
		return boolConst(s1 < s2), nil
	case ast.OperatorLessOrEqual:
		return boolConst(s1 <= s2), nil
	case ast.OperatorGreater:
		return boolConst(s1 > s2), nil
	case ast.OperatorGreaterOrEqual:
		return boolConst(s1 >= s2), nil
	case ast.OperatorAddition:
		return s1 + s2, nil
	}
	return nil, errInvalidOperation
}

func (c1 stringConst) representedBy(typ reflect.Type) (constant, error) {
	if typ.Kind() == reflect.String {
		return c1, nil
	}
	return nil, errNotRepresentable
}

func (c1 stringConst) zero() bool {
	return c1 == ""
}

func (c1 stringConst) real() constant {
	panic("not number constant")
}

func (c1 stringConst) imag() constant {
	panic("not number constant")
}

func (c1 stringConst) equals(c2 constant) bool {
	if c2, ok := c2.(stringConst); ok {
		return c1 == c2
	}
	return false
}

// int64Const represents an integer constant in the int64 range of values.
type int64Const int64

func (c1 int64Const) String() string {
	return strconv.FormatInt(int64(c1), 10)
}

func (c1 int64Const) bool() bool             { return false }
func (c1 int64Const) string() string         { return "" }
func (c1 int64Const) int64() int64           { return int64(c1) }
func (c1 int64Const) uint64() uint64         { return uint64(c1) }
func (c1 int64Const) float64() float64       { return float64(c1) }
func (c1 int64Const) complex128() complex128 { return complex(float64(c1), 0) }

func (c1 int64Const) unaryOp(op ast.OperatorType) (constant, error) {
	switch op {
	case ast.OperatorAddition:
		return c1, nil
	case ast.OperatorSubtraction:
		if c1 == math.MinInt64 {
			i := new(big.Int).SetInt64(int64(c1))
			return intConst{i: i.Neg(i)}, nil
		}
		return -c1, nil
	case ast.OperatorXor:
		// TODO(marco)
		panic("unary xor not yet implemented")
	}
	return nil, errInvalidOperation
}

func (c1 int64Const) binaryOp(op ast.OperatorType, c2 constant) (constant, error) {
	if op == ast.OperatorLeftShift || op == ast.OperatorRightShift {
		if err := shiftConstError(c2); err != nil {
			return nil, err
		}
		sc := uint(c2.uint64())
		if op == ast.OperatorLeftShift {
			i := big.NewInt(int64(c1))
			return intConst{i: i.Lsh(i, sc)}, nil
		}
		return c1 >> sc, nil
	}
	n1 := c1
	n2, ok := c2.(int64Const)
	if !ok {
		d1, d2 := toSameConstImpl(c1, c2)
		return d1.binaryOp(op, d2)
	}
	switch op {
	case ast.OperatorEqual:
		return boolConst(n1 == n2), nil
	case ast.OperatorNotEqual:
		return boolConst(n1 != n2), nil
	case ast.OperatorLess:
		return boolConst(n1 < n2), nil
	case ast.OperatorLessOrEqual:
		return boolConst(n1 <= n2), nil
	case ast.OperatorGreater:
		return boolConst(n1 > n2), nil
	case ast.OperatorGreaterOrEqual:
		return boolConst(n1 >= n2), nil
	case ast.OperatorAddition:
		n := n1 + n2
		if (n < n1) != (n2 < 0) {
			return n1.asInt().binaryOp(op, n2.asInt())
		}
		return n, nil
	case ast.OperatorSubtraction:
		n := n1 - n2
		if (n < n1) != (n2 > 0) {
			return n1.asInt().binaryOp(op, n2.asInt())
		}
		return n, nil
	case ast.OperatorMultiplication:
		n := n1 * n2
		if (n < 0) != ((n1 < 0) != (n2 < 0)) || n/n2 != n1 {
			return n1.asInt().binaryOp(op, n2.asInt())
		}
		return n, nil
	case ast.OperatorDivision:
		if n2 == 0 {
			return nil, errDivisionByZero
		}
		if n1%n2 == 0 && !(n1 == minInt64 && n2 == -1) {
			return n1 / n2, nil
		}
		return newRatConst(int64(n1), int64(n2)), nil
	case ast.OperatorModulo:
		if n2 == 0 {
			return nil, errDivisionByZero
		}
		return n1 % n2, nil
	case ast.OperatorAnd:
		return n1 & n2, nil
	case ast.OperatorOr:
		return n1 | n2, nil
	case ast.OperatorXor:
		return n1 ^ n2, nil
	case ast.OperatorAndNot:
		return n1 &^ n2, nil
	}
	return nil, errInvalidOperation
}

func (c1 int64Const) representedBy(typ reflect.Type) (constant, error) {
	n := int64(c1)
	switch typ.Kind() {
	case reflect.Int:
		if int64(minInt) <= n && n <= int64(maxInt) {
			return c1, nil
		}
	case reflect.Int8:
		if -1<<7 <= n && n <= 1<<7-1 {
			return c1, nil
		}
	case reflect.Int16:
		if -1<<15 <= n && n <= 1<<15-1 {
			return c1, nil
		}
	case reflect.Int32:
		if -1<<31 <= n && n <= 1<<31-1 {
			return c1, nil
		}
	case reflect.Int64:
		return c1, nil
	case reflect.Uint, reflect.Uintptr:
		if 0 <= n && uint64(n) <= uint64(maxUint) {
			return c1, nil
		}
	case reflect.Uint8:
		if 0 <= n && n <= 1<<8-1 {
			return c1, nil
		}
	case reflect.Uint16:
		if 0 <= n && n <= 1<<16-1 {
			return c1, nil
		}
	case reflect.Uint32:
		if 0 <= n && n <= 1<<32-1 {
			return c1, nil
		}
	case reflect.Uint64:
		if n >= 0 {
			return c1, nil
		}
	case reflect.Float32, reflect.Complex64:
		return float64Const(float32(c1)), nil
	case reflect.Float64, reflect.Complex128:
		return float64Const(c1), nil
	default:
		return nil, errNotRepresentable
	}
	return nil, fmt.Errorf("constant %s overflows %s", c1, typ)
}

func (c1 int64Const) zero() bool {
	return c1 == 0
}

func (c1 int64Const) real() constant {
	return c1
}

func (c1 int64Const) imag() constant {
	return int64Const(0)
}

func (c1 int64Const) equals(c2 constant) bool {
	n1 := c1
	n2, ok := c2.(int64Const)
	if !ok {
		d1, d2 := toSameConstImpl(c1, c2)
		return d1.equals(d2)
	}
	return n1 == n2
}

func (c1 int64Const) asInt() intConst {
	return intConst{i: new(big.Int).SetInt64(int64(c1))}
}

// intConst represents an integer constant.
type intConst struct {
	i *big.Int
}

func newIntConst(x int64) intConst {
	return intConst{i: big.NewInt(x)}
}

func (c1 intConst) String() string {
	return c1.i.Text(10)
}

func (c1 intConst) bool() bool     { return false }
func (c1 intConst) string() string { return "" }
func (c1 intConst) int64() int64   { return c1.i.Int64() }
func (c1 intConst) uint64() uint64 { return c1.i.Uint64() }
func (c1 intConst) float64() float64 {
	if c1.i.IsInt64() {
		return float64(c1.i.Int64())
	}
	if c1.i.IsUint64() {
		return float64(c1.i.Int64())
	}
	f, _ := new(big.Float).SetInt(c1.i).Float64()
	return f
}
func (c1 intConst) complex128() complex128 { return complex(c1.float64(), 0) }

func (c1 intConst) unaryOp(op ast.OperatorType) (constant, error) {
	switch op {
	case ast.OperatorAddition:
		return c1, nil
	case ast.OperatorSubtraction:
		i := new(big.Int).Set(c1.i)
		return intConst{i: i.Neg(i)}, nil
	case ast.OperatorXor:
		// TODO(marco)
		panic("unary xor not yet implemented")
	}
	return nil, errInvalidOperation
}

func (c1 intConst) binaryOp(op ast.OperatorType, c2 constant) (constant, error) {
	if op == ast.OperatorLeftShift || op == ast.OperatorRightShift {
		if err := shiftConstError(c2); err != nil {
			return nil, err
		}
		sc := uint(c2.uint64())
		i := new(big.Int).Set(c1.i)
		if op == ast.OperatorLeftShift {
			return intConst{i: i.Lsh(i, sc)}, nil
		}
		return intConst{i: i.Rsh(i, sc)}, nil
	}
	n1 := c1
	n2, ok := c2.(intConst)
	if !ok {
		d1, d2 := toSameConstImpl(c1, c2)
		return d1.binaryOp(op, d2)
	}
	switch op {
	default:
		cmp := n1.i.Cmp(n2.i)
		switch op {
		case ast.OperatorEqual:
			return boolConst(cmp == 0), nil
		case ast.OperatorNotEqual:
			return boolConst(cmp != 0), nil
		case ast.OperatorLess:
			return boolConst(cmp < 0), nil
		case ast.OperatorLessOrEqual:
			return boolConst(cmp <= 0), nil
		case ast.OperatorGreater:
			return boolConst(cmp > 0), nil
		case ast.OperatorGreaterOrEqual:
			return boolConst(cmp >= 0), nil
		}
	case ast.OperatorAddition:
		return intConst{i: new(big.Int).Add(n1.i, n2.i)}, nil
	case ast.OperatorSubtraction:
		return intConst{i: new(big.Int).Sub(n1.i, n2.i)}, nil
	case ast.OperatorMultiplication:
		return intConst{i: new(big.Int).Mul(n1.i, n2.i)}, nil
	case ast.OperatorDivision:
		if n2.i.Sign() == 0 {
			return nil, errDivisionByZero
		}
		return newRatConst(0, 0).setFrac(n1.i, n2.i), nil
	case ast.OperatorModulo:
		if n2.i.Sign() == 0 {
			return nil, errDivisionByZero
		}
		return newRatConst(0, 0).setFrac(n1.i, n2.i), nil
	case ast.OperatorAnd:
		return intConst{i: new(big.Int).And(n1.i, n2.i)}, nil
	case ast.OperatorOr:
		return intConst{i: new(big.Int).Or(n1.i, n2.i)}, nil
	case ast.OperatorXor:
		return intConst{i: new(big.Int).Xor(n1.i, n2.i)}, nil
	case ast.OperatorAndNot:
		return intConst{i: new(big.Int).AndNot(n1.i, n2.i)}, nil
	}
	return nil, errInvalidOperation
}

func (c1 intConst) representedBy(typ reflect.Type) (constant, error) {
	if c1.i.IsInt64() {
		return int64Const(c1.i.Int64()).representedBy(typ)
	}
	kind := typ.Kind()
	if c1.i.IsUint64() {
		if kind == reflect.Uint64 || kind == reflect.Uint && strconv.IntSize == 64 {
			return c1, nil
		}
	}
	if reflect.Int <= kind && kind <= reflect.Uintptr {
		return nil, fmt.Errorf("constant %s overflows %s", c1, typ)
	}
	if reflect.Float32 <= kind && kind <= reflect.Complex128 {
		return newFloatConst(0).setInt(c1.i).representedBy(typ)
	}
	return nil, errNotRepresentable
}

func (c1 intConst) zero() bool {
	return c1.i.Sign() == 0
}

func (c1 intConst) real() constant {
	return c1
}

func (c1 intConst) imag() constant {
	return int64Const(0)
}

func (c1 intConst) equals(c2 constant) bool {
	n1 := c1
	n2, ok := c2.(intConst)
	if !ok {
		d1, d2 := toSameConstImpl(c1, c2)
		return d1.equals(d2)
	}
	return n1.i.Cmp(n2.i) == 0
}

func (c1 intConst) setUint64(n uint64) constant { c1.i.SetUint64(n); return c1 }

func (c1 intConst) setInt(n *big.Int) constant { c1.i.Set(n); return c1 }

// float64Const represents a floating point constant in the float64 range of
// values.
type float64Const float64

func (c1 float64Const) String() string {
	return strconv.FormatFloat(float64(c1), 'f', -1, 64)
}

func (c1 float64Const) bool() bool             { return false }
func (c1 float64Const) string() string         { return "" }
func (c1 float64Const) int64() int64           { return int64(c1) }
func (c1 float64Const) uint64() uint64         { return uint64(c1) }
func (c1 float64Const) float64() float64       { return float64(c1) }
func (c1 float64Const) complex128() complex128 { return complex(float64(c1), 0) }

func (c1 float64Const) unaryOp(op ast.OperatorType) (constant, error) {
	switch op {
	case ast.OperatorAddition:
		return c1, nil
	case ast.OperatorSubtraction:
		return -c1, nil
	}
	return nil, errInvalidOperation
}

func (c1 float64Const) binaryOp(op ast.OperatorType, c2 constant) (constant, error) {
	if op == ast.OperatorLeftShift || op == ast.OperatorRightShift {
		return newFloatConst(float64(c1)).binaryOp(op, c2)
	}
	n1 := c1
	n2, ok := c2.(float64Const)
	if !ok {
		d1, d2 := toSameConstImpl(c1, c2)
		return d1.binaryOp(op, d2)
	}
	switch op {
	case ast.OperatorEqual:
		return boolConst(n1 == n2), nil
	case ast.OperatorNotEqual:
		return boolConst(n1 != n2), nil
	case ast.OperatorLess:
		return boolConst(n1 < n2), nil
	case ast.OperatorLessOrEqual:
		return boolConst(n1 <= n2), nil
	case ast.OperatorGreater:
		return boolConst(n1 > n2), nil
	case ast.OperatorGreaterOrEqual:
		return boolConst(n1 >= n2), nil
	case ast.OperatorAddition:
		if n1 == 0 {
			return n2, nil
		}
		if n2 == 0 {
			return n1, nil
		}
		return n1.asFloat().binaryOp(op, n2.asFloat())
	case ast.OperatorSubtraction:
		if n2 == 0 {
			return n1, nil
		}
		return n1.asFloat().binaryOp(op, n2.asFloat())
	case ast.OperatorMultiplication:
		if n1 == 0 || n2 == 0 {
			return float64Const(0), nil
		}
		if n1 == 1 {
			return n2, nil
		}
		if n2 == 1 {
			return n1, nil
		}
		return n1.asFloat().binaryOp(op, n2.asFloat())
	case ast.OperatorDivision:
		if n2 == 0 {
			return nil, errDivisionByZero
		}
		if n2 == 1 {
			return n1, nil
		}
		return n1.asFloat().binaryOp(op, n2.asFloat())
	}
	return nil, errInvalidOperation
}

func (c1 float64Const) representedBy(typ reflect.Type) (constant, error) {
	f := float64(c1)
	switch typ.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if -1<<63-1024 <= f && f <= 1<<63-513 && float64(int64(f)) == f {
			return int64Const(f).representedBy(typ)
		}
		return nil, fmt.Errorf("constant %s truncated to integer", c1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if 0 <= f && f <= 1<<64-1025 && float64(int64(f)) == f {
			return int64Const(f).representedBy(typ)
		}
		return nil, fmt.Errorf("constant %s truncated to integer", c1)
	case reflect.Float32, reflect.Complex64:
		if f := float32(c1); !math.IsInf(float64(f), 0) {
			return float64Const(f), nil
		}
		return nil, fmt.Errorf("constant %s overflow %s", c1, typ)
	case reflect.Float64, reflect.Complex128:
		return c1, nil
	}
	return nil, errNotRepresentable
}

func (c1 float64Const) zero() bool {
	return c1 == 0
}

func (c1 float64Const) real() constant {
	return c1
}

func (c1 float64Const) imag() constant {
	return int64Const(0)
}

func (c1 float64Const) equals(c2 constant) bool {
	n1 := c1
	n2, ok := c2.(float64Const)
	if !ok {
		d1, d2 := toSameConstImpl(c1, c2)
		return d1.equals(d2)
	}
	return n1 == n2
}

func bigFloat() *big.Float {
	return new(big.Float).SetPrec(512)
}

func (c1 float64Const) asFloat() floatConst {
	return floatConst{f: bigFloat().SetFloat64(float64(c1))}
}

// floatConst represents a floating point constant.
type floatConst struct {
	f *big.Float
}

func (c1 floatConst) String() string {
	return c1.f.String()
}

func newFloatConst(x float64) floatConst {
	return floatConst{f: bigFloat().SetFloat64(x)}
}

func (c1 floatConst) bool() bool             { return false }
func (c1 floatConst) string() string         { return "" }
func (c1 floatConst) int64() int64           { n, _ := c1.f.Int64(); return n }
func (c1 floatConst) uint64() uint64         { n, _ := c1.f.Uint64(); return n }
func (c1 floatConst) float64() float64       { f, _ := c1.f.Float64(); return f }
func (c1 floatConst) complex128() complex128 { return complex(c1.float64(), 0) }

func (c1 floatConst) unaryOp(op ast.OperatorType) (constant, error) {
	switch op {
	case ast.OperatorAddition:
		return c1, nil
	case ast.OperatorSubtraction:
		f := bigFloat().Set(c1.f)
		return floatConst{f: f.Neg(f)}, nil
	}
	return nil, errInvalidOperation
}

func (c1 floatConst) binaryOp(op ast.OperatorType, c2 constant) (constant, error) {
	if op == ast.OperatorLeftShift || op == ast.OperatorRightShift {
		if c1.f.IsInt() {
			i, _ := c1.f.Int(nil)
			return newIntConst(0).setInt(i).binaryOp(op, c2)
		}
		return nil, fmt.Errorf("constant %s truncated to integer", c1)
	}
	n1 := c1
	n2, ok := c2.(floatConst)
	if !ok {
		d1, d2 := toSameConstImpl(c1, c2)
		return d1.binaryOp(op, d2)
	}
	switch op {
	default:
		cmp := n1.f.Cmp(n2.f)
		switch op {
		case ast.OperatorEqual:
			return boolConst(cmp == 0), nil
		case ast.OperatorNotEqual:
			return boolConst(cmp != 0), nil
		case ast.OperatorLess:
			return boolConst(cmp < 0), nil
		case ast.OperatorLessOrEqual:
			return boolConst(cmp <= 0), nil
		case ast.OperatorGreater:
			return boolConst(cmp > 0), nil
		case ast.OperatorGreaterOrEqual:
			return boolConst(cmp >= 0), nil
		}
	case ast.OperatorAddition:
		return floatConst{f: bigFloat().Add(n1.f, n2.f)}, nil
	case ast.OperatorSubtraction:
		return floatConst{f: bigFloat().Sub(n1.f, n2.f)}, nil
	case ast.OperatorMultiplication:
		return floatConst{f: bigFloat().Mul(n1.f, n2.f)}, nil
	case ast.OperatorDivision:
		if n2.f.Sign() == 0 {
			return nil, errDivisionByZero
		}
		return floatConst{f: bigFloat().Quo(n1.f, n2.f)}, nil
	}
	return nil, errInvalidOperation
}

func (c1 floatConst) representedBy(typ reflect.Type) (constant, error) {
	kind := typ.Kind()
	if reflect.Int <= kind && kind <= reflect.Int64 {
		if n, acc := c1.f.Int64(); acc == big.Exact {
			return int64Const(n).representedBy(typ)
		}
		return nil, fmt.Errorf("constant %s truncated to integer", c1)
	}
	if reflect.Uint <= kind && kind <= reflect.Uintptr {
		if n, acc := c1.f.Uint64(); acc == big.Exact {
			if n <= maxInt64 {
				return int64Const(n).representedBy(typ)
			}
			return newIntConst(0).setUint64(n).representedBy(typ)
		}
		return nil, fmt.Errorf("constant %s truncated to integer", c1)
	}
	if f, _ := c1.f.Float64(); !math.IsInf(f, 1) {
		return float64Const(f).representedBy(typ)
	}
	if reflect.Float32 <= kind && kind <= reflect.Complex128 {
		return nil, fmt.Errorf("constant %s overflows %s", c1, typ)
	}
	return nil, errNotRepresentable
}

func (c1 floatConst) zero() bool {
	return c1.f.Sign() == 0
}

func (c1 floatConst) real() constant {
	return c1
}

func (c1 floatConst) imag() constant {
	return int64Const(0)
}

func (c1 floatConst) equals(c2 constant) bool {
	n1 := c1
	n2, ok := c2.(floatConst)
	if !ok {
		d1, d2 := toSameConstImpl(c1, c2)
		return d1.equals(d2)
	}
	return n1.f.Cmp(n2.f) == 0
}

func (c1 floatConst) setInt64(x int64) floatConst  { c1.f.SetInt64(x); return c1 }
func (c1 floatConst) setInt(x *big.Int) floatConst { c1.f.SetInt(x); return c1 }
func (c1 floatConst) setRat(x *big.Rat) floatConst { c1.f.SetRat(x); return c1 }

// ratConst represents a floating point constant.
type ratConst struct {
	r *big.Rat
}

func newRatConst(x, y int64) ratConst {
	return ratConst{r: big.NewRat(x, y)}
}

func (c1 ratConst) String() string {
	return newFloatConst(0).setRat(c1.r).String()
}

func (c1 ratConst) bool() bool             { return false }
func (c1 ratConst) string() string         { return "" }
func (c1 ratConst) int64() int64           { return c1.r.Num().Int64() }
func (c1 ratConst) uint64() uint64         { return c1.r.Num().Uint64() }
func (c1 ratConst) float64() float64       { f, _ := c1.r.Float64(); return f }
func (c1 ratConst) complex128() complex128 { return complex(c1.float64(), 0) }

func (c1 ratConst) unaryOp(op ast.OperatorType) (constant, error) {
	switch op {
	case ast.OperatorAddition:
		return c1, nil
	case ast.OperatorSubtraction:
		r := new(big.Rat).Set(c1.r)
		return ratConst{r: r.Neg(r)}, nil
	}
	return nil, errInvalidOperation
}

func (c1 ratConst) binaryOp(op ast.OperatorType, c2 constant) (constant, error) {
	if op == ast.OperatorLeftShift || op == ast.OperatorRightShift {
		if c1.r.IsInt() {
			num := c1.r.Num()
			return newIntConst(0).setInt(num).binaryOp(op, c2)
		}
		return nil, fmt.Errorf("constant %s truncated to integer", c1)
	}
	n1 := c1
	n2, ok := c2.(ratConst)
	if !ok {
		d1, d2 := toSameConstImpl(c1, c2)
		return d1.binaryOp(op, d2)
	}
	switch op {
	default:
		cmp := n1.r.Cmp(n2.r)
		switch op {
		case ast.OperatorEqual:
			return boolConst(cmp == 0), nil
		case ast.OperatorNotEqual:
			return boolConst(cmp != 0), nil
		case ast.OperatorLess:
			return boolConst(cmp < 0), nil
		case ast.OperatorLessOrEqual:
			return boolConst(cmp <= 0), nil
		case ast.OperatorGreater:
			return boolConst(cmp > 0), nil
		case ast.OperatorGreaterOrEqual:
			return boolConst(cmp >= 0), nil
		}
	case ast.OperatorAddition:
		return ratConst{r: new(big.Rat).Add(n1.r, n2.r)}, nil
	case ast.OperatorSubtraction:
		return ratConst{r: new(big.Rat).Sub(n1.r, n2.r)}, nil
	case ast.OperatorMultiplication:
		return ratConst{r: new(big.Rat).Mul(n1.r, n2.r)}, nil
	case ast.OperatorDivision:
		if n2.r.Sign() == 0 {
			return nil, errDivisionByZero
		}
		return ratConst{r: new(big.Rat).Quo(n1.r, n2.r)}, nil
	}
	return nil, errInvalidOperation
}

func (c1 ratConst) representedBy(typ reflect.Type) (constant, error) {
	if c1.r.IsInt() {
		return intConst{i: c1.r.Num()}.representedBy(typ)
	}
	if f, ok := c1.r.Float64(); ok {
		return float64Const(f).representedBy(typ)
	}
	return newFloatConst(0).setRat(c1.r).representedBy(typ)
}

func (c1 ratConst) zero() bool {
	return c1.r.Sign() == 0
}

func (c1 ratConst) real() constant {
	return c1
}

func (c1 ratConst) imag() constant {
	return int64Const(0)
}

func (c1 ratConst) equals(c2 constant) bool {
	n1 := c1
	n2, ok := c2.(ratConst)
	if !ok {
		d1, d2 := toSameConstImpl(c1, c2)
		return d1.equals(d2)
	}
	return n1.r.Cmp(n2.r) == 0
}

func (c1 ratConst) setFrac(num, den *big.Int) ratConst { c1.r.SetFrac(num, den); return c1 }
func (c1 ratConst) setFloat64(x float64) ratConst      { c1.r.SetFloat64(x); return c1 }

// complexConst represents a complex constant.
type complexConst struct {
	r, i constant
}

func newComplexConst(re, im constant) complexConst {
	return complexConst{r: re, i: im}
}

func (c1 complexConst) String() string {
	re := c1.r.String()
	im := c1.i.String()
	if im[0] != '-' {
		im = "+" + im
	}
	return "(" + re + im + "i)"
}

func (c1 complexConst) shortString() string {
	if c1.r.zero() {
		return c1.i.String() + "i"
	}
	if c1.i.zero() {
		return c1.r.String()
	}
	re := c1.r.String()
	im := c1.i.String()
	if im[0] != '-' {
		im = "+" + im
	}
	return re + im + "i"
}

func (c1 complexConst) bool() bool             { return false }
func (c1 complexConst) string() string         { return "" }
func (c1 complexConst) int64() int64           { return c1.r.int64() }
func (c1 complexConst) uint64() uint64         { return c1.r.uint64() }
func (c1 complexConst) float64() float64       { return c1.r.float64() }
func (c1 complexConst) complex128() complex128 { return complex(c1.r.float64(), c1.i.float64()) }

func (c1 complexConst) unaryOp(op ast.OperatorType) (constant, error) {
	switch op {
	case ast.OperatorAddition:
		return c1, nil
	case ast.OperatorSubtraction:
		r, _ := c1.r.unaryOp(op)
		i, _ := c1.i.unaryOp(op)
		return complexConst{r: r, i: i}, nil
	}
	return nil, errInvalidOperation
}

func (c1 complexConst) binaryOp(op ast.OperatorType, c2 constant) (constant, error) {
	if op == ast.OperatorLeftShift || op == ast.OperatorRightShift {
		if c1.i.zero() {
			return c1.r.binaryOp(op, c2)
		}
		return nil, fmt.Errorf("constant %s truncated to integer", c1.shortString())
	}
	n1 := c1
	n2, ok := c2.(complexConst)
	if !ok {
		d1, d2 := toSameConstImpl(c1, c2)
		return d1.binaryOp(op, d2)
	}
	switch op {
	case ast.OperatorEqual:
		re, _ := n1.r.binaryOp(op, n2.r)
		im, _ := n1.i.binaryOp(op, n2.i)
		return re.(boolConst) && im.(boolConst), nil
	case ast.OperatorAddition, ast.OperatorSubtraction:
		re, _ := n1.r.binaryOp(op, n2.r)
		im, _ := n1.i.binaryOp(op, n2.i)
		return newComplexConst(re, im), nil
	case ast.OperatorMultiplication:
		ac, _ := n1.r.binaryOp(op, n2.r)
		bd, _ := n1.i.binaryOp(op, n2.i)
		bc, _ := n1.i.binaryOp(op, n2.r)
		ad, _ := n1.r.binaryOp(op, n2.i)
		c := complexConst{}
		c.r, _ = ac.binaryOp(ast.OperatorSubtraction, bd)
		c.i, _ = bc.binaryOp(ast.OperatorSubtraction, ad)
		return c, nil
	case ast.OperatorDivision:
		if n2.zero() {
			return nil, errComplexDivisionByZero
		}
		// s = cc + dd
		cc, _ := n2.r.binaryOp(ast.OperatorMultiplication, n2.r)
		dd, _ := n2.i.binaryOp(ast.OperatorMultiplication, n2.i)
		s, _ := cc.binaryOp(ast.OperatorAddition, dd)
		// z = (ac+bd)/s + i(bc-ad)/s
		ac, _ := n1.r.binaryOp(ast.OperatorMultiplication, n2.r)
		bd, _ := n1.i.binaryOp(ast.OperatorMultiplication, n2.i)
		bc, _ := n1.i.binaryOp(ast.OperatorMultiplication, n2.r)
		ad, _ := n1.r.binaryOp(ast.OperatorMultiplication, n2.i)
		re, _ := ac.binaryOp(ast.OperatorAddition, bd)
		im, _ := bc.binaryOp(ast.OperatorSubtraction, ad)
		c := complexConst{}
		c.r, _ = re.binaryOp(op, s)
		c.i, _ = im.binaryOp(op, s)
		return c, nil
	}
	return nil, errInvalidOperation
}

func (c1 complexConst) representedBy(typ reflect.Type) (constant, error) {
	if c1.i.zero() {
		return c1.r.representedBy(typ)
	}
	kind := typ.Kind()
	if kind == reflect.Complex64 || kind == reflect.Complex128 {
		t := float32Type
		if kind == reflect.Complex128 {
			t = float64Type
		}
		re, err := c1.r.representedBy(t)
		if err != nil {
			return nil, err
		}
		im, err := c1.i.representedBy(t)
		if err != nil {
			return nil, err
		}
		return complexConst{r: re, i: im}, nil
	}
	if reflect.Int <= kind && kind <= reflect.Uintptr {
		return nil, fmt.Errorf("constant %s truncated to integer", c1.shortString())
	}
	if kind == reflect.Float32 || kind == reflect.Float64 {
		return nil, fmt.Errorf("constant %s truncated to real", c1.shortString())
	}
	return nil, errNotRepresentable
}

func (c1 complexConst) zero() bool {
	return c1.r.zero() && c1.i.zero()
}

func (c1 complexConst) real() constant {
	return c1.r
}

func (c1 complexConst) imag() constant {
	return c1.i
}

func (c1 complexConst) equals(c2 constant) bool {
	n1 := c1
	n2, ok := c2.(complexConst)
	if !ok {
		d1, d2 := toSameConstImpl(c1, c2)
		return d1.equals(d2)
	}
	return n1.r.equals(n2.r) && n1.i.equals(n2.i)
}

// toSameConstImpl returns the two constants with the same implementation type
// without changing its represented values.
func toSameConstImpl(c1, c2 constant) (constant, constant) {
	switch n1 := c1.(type) {
	case int64Const:
		switch n2 := c2.(type) {
		case intConst:
			return newIntConst(int64(n1)), n2
		case float64Const:
			return newFloatConst(0).setInt64(int64(n1)), newFloatConst(float64(n2))
		case floatConst:
			return newFloatConst(0).setInt64(int64(n1)), n2
		case ratConst:
			return newRatConst(int64(n1), 1), n2
		case complexConst:
			return newComplexConst(n1, int64Const(0)), n2
		}
	case intConst:
		switch n2 := c2.(type) {
		case float64Const:
			return newFloatConst(0).setInt(n1.i), newFloatConst(float64(n2))
		case floatConst:
			return newFloatConst(0).setInt(n1.i), n2
		case ratConst:
			return newRatConst(1, 1).setFrac(n1.i, big.NewInt(1)), n2
		case complexConst:
			return newComplexConst(n1, int64Const(0)), n2
		}
	case float64Const:
		switch n2 := c2.(type) {
		case floatConst:
			return newFloatConst(float64(n1)), n2
		case ratConst:
			return newRatConst(1, 1).setFloat64(float64(n1)), n2
		case complexConst:
			return newComplexConst(n1, int64Const(0)), n2
		}
	case floatConst:
		switch n2 := c2.(type) {
		case ratConst:
			return n1, newFloatConst(0).setRat(n2.r)
		case complexConst:
			return newComplexConst(n1, int64Const(0)), n2
		}
	case ratConst:
		switch n2 := c2.(type) {
		case complexConst:
			return newComplexConst(n1, int64Const(0)), n2
		}
	}
	n2, n1 := toSameConstImpl(c2, c1)
	return n1, n2
}

var errNegativeShiftCount = errors.New("negative shift count")
var errShiftCountTooLarge = errors.New("shift count too large")
var errShiftCountTruncatedToInteger = errors.New("shift count truncated to integer")

// shiftConstError returns an error that explain why c cannot be used as the
// right operand in a shift expression. Returns nil if c can be used.
func shiftConstError(c constant) error {
	if c, _ := c.representedBy(uintType); c != nil {
		if ok, _ := c.binaryOp(ast.OperatorGreaterOrEqual, int64Const(512)); ok.bool() {
			return errShiftCountTooLarge
		}
		return nil
	}
	switch n := c.(type) {
	case int64Const:
		if n < 0 {
			return errNegativeShiftCount
		}
		return errShiftCountTooLarge
	case intConst:
		if n.i.Sign() < 0 {
			return errNegativeShiftCount
		}
		return errShiftCountTooLarge
	}
	return errShiftCountTruncatedToInteger
}

// convertToConstant converts a value to a constant.
func convertToConstant(value interface{}) constant {
	v := reflect.ValueOf(value)
	k := v.Kind()
	switch {
	case k == reflect.Bool:
		return boolConst(v.Bool())
	case reflect.Int <= k && k <= reflect.Int64:
		return int64Const(v.Int())
	case reflect.Uint <= k && k <= reflect.Uintptr:
		n := v.Uint()
		if n <= maxInt64 {
			return int64Const(n)
		}
		return newIntConst(0).setUint64(n)
	case k == reflect.Float64 || k == reflect.Float32:
		return float64Const(v.Float())
	case k == reflect.String:
		return stringConst(v.String())
	case k == reflect.Complex64, k == reflect.Complex128:
		c := v.Complex()
		return newComplexConst(float64Const(real(c)), float64Const(imag(c)))
	}
	return nil
}

// parseConstant parses a constant and if the parsing was successful returns
// the constant kind and value. The string can be a basic literal, "true",
// "false" or "a/b" where a and b are integers.
//
// If the string cannot be parsed, it returns the error strconv.ErrSyntax.
func parseConstant(s string) (constant, reflect.Type, error) {
	if len(s) == 0 {
		return nil, nil, strconv.ErrSyntax
	}
	switch s[0] {
	case 't':
		if s == "true" {
			return boolConst(true), boolType, nil
		}
	case 'f':
		if s == "false" {
			return boolConst(false), boolType, nil
		}
	case '"', '`':
		str, err := strconv.Unquote(s)
		if err == nil {
			return stringConst(str), stringType, nil
		}
	case '\'':
		r, _, tail, err := strconv.UnquoteChar(s[1:], '\'')
		if err == nil && tail == "'" {
			return int64Const(r), int32Type, nil
		}
	default:
		if strings.Contains(s, "/") {
			r, ok := new(big.Rat).SetString(s)
			if ok {
				return ratConst{r: r}, float64Type, nil
			}
		} else if s[len(s)-1] == 'i' {
			if strings.ContainsAny(s, ".eEpP") {
				i, _, err := parseConstant(s[:len(s)-1])
				if err == nil {
					return complexConst{r: int64Const(0), i: i}, complex128Type, nil
				}
			} else if len(s) >= 2 {
				if s[0] == '0' && ('0' <= s[1] && s[1] <= '9' || s[1] == '_') {
					s = strings.TrimLeft(s, "0_")
					if s == "i" {
						return int64Const(0), intType, nil
					}
				}
				i, _, err := parseConstant(s[:len(s)-1])
				if err == nil {
					return complexConst{r: int64Const(0), i: i}, complex128Type, nil
				}
			}
		} else if strings.ContainsAny(s, ".eEpE") {
			n, ok := bigFloat().SetString(s[:len(s)-1])
			if ok {
				if f, _ := n.Float64(); !math.IsInf(f, 1) {
					return float64Const(f), float64Type, nil
				}
				return floatConst{f: n}, float64Type, nil
			}
		} else {
			n, ok := new(big.Int).SetString(s, 0)
			if ok {
				if n.IsInt64() {
					return int64Const(n.Int64()), intType, nil
				}
				return intConst{i: n}, intType, nil
			}
		}
	}
	return nil, nil, strconv.ErrSyntax
}

// parseBasicLiteral parses a basic literal and returns the represented
// constant. If the string is not a basic literal the behaviour is undefined.
func parseBasicLiteral(typ ast.LiteralType, s string) constant {
	switch typ {
	case ast.StringLiteral:
		return stringConst(unquoteString([]byte(s)))
	case ast.RuneLiteral:
		if len(s) == 3 {
			return int64Const(s[1])
		}
		r, _ := parseEscapedRune([]byte(s[1:]))
		return int64Const(r)
	case ast.IntLiteral:
		n, _ := new(big.Int).SetString(s, 0)
		if n.IsInt64() {
			return int64Const(n.Int64())
		}
		return intConst{i: n}
	case ast.FloatLiteral:
		n, _ := bigFloat().SetString(s)
		if n.MinPrec() < 53 {
			f, _ := n.Float64()
			return float64Const(f)
		}
		if !n.IsInf() {
			const maxExp = 4 << 10
			if e := n.MantExp(nil); -maxExp < e && e < maxExp {
				r, _ := new(big.Rat).SetString(s)
				return ratConst{r: r}
			}
		}
		return floatConst{f: n}
	case ast.ImaginaryLiteral:
		if strings.ContainsAny(s, ".eEpP") {
			return complexConst{r: int64Const(0), i: parseBasicLiteral(ast.FloatLiteral, s[:len(s)-1])}
		}
		if s[0] == '0' && ('0' <= s[1] && s[1] <= '9' || s[1] == '_') {
			s = strings.TrimLeft(s, "0_")
			if s == "i" {
				return int64Const(0)
			}
		}
		return complexConst{r: int64Const(0), i: parseBasicLiteral(ast.IntLiteral, s[:len(s)-1])}
	}
	panic("no basic literal")
}
