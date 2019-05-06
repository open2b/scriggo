// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"scrigo/ast"
)

func tierr(line, column int, text string) *Error {
	return &Error{Pos: ast.Position{Line: line, Column: column}, Err: errors.New(text)}
}

type definedBool bool
type definedString string
type definedInt int
type definedIntSlice []int
type definedIntSlice2 []int
type definedByteSlice []byte
type definedStringSlice []byte
type definedStringMap map[string]string

var checkerExprs = []struct {
	src   string
	ti    *TypeInfo
	scope map[string]*TypeInfo
}{
	// Untyped constant literals.
	{`true`, tiUntypedBoolConst(true), nil},
	{`false`, tiUntypedBoolConst(false), nil},
	{`""`, tiUntypedStringConst(""), nil},
	{`"abc"`, tiUntypedStringConst("abc"), nil},
	{`0`, tiUntypedIntConst("0"), nil},
	{`'a'`, tiUntypedRuneConst('a'), nil},
	{`0.0`, tiUntypedFloatConst("0"), nil},

	// Untyped constants.
	{`a`, tiUntypedBoolConst(true), map[string]*TypeInfo{"a": tiUntypedBoolConst(true)}},
	{`a`, tiUntypedBoolConst(false), map[string]*TypeInfo{"a": tiUntypedBoolConst(false)}},
	{`a`, tiUntypedStringConst("a"), map[string]*TypeInfo{"a": tiUntypedStringConst("a")}},
	{`a`, tiUntypedIntConst("0"), map[string]*TypeInfo{"a": tiUntypedIntConst("0")}},
	{`a`, tiUntypedRuneConst(0), map[string]*TypeInfo{"a": tiUntypedRuneConst(0)}},
	{`a`, tiUntypedFloatConst("0.0"), map[string]*TypeInfo{"a": tiUntypedFloatConst("0.0")}},

	// Typed constants
	{`a`, tiBoolConst(true), map[string]*TypeInfo{"a": tiBoolConst(true)}},
	{`a`, tiBoolConst(false), map[string]*TypeInfo{"a": tiBoolConst(false)}},
	{`a`, tiStringConst("a"), map[string]*TypeInfo{"a": tiStringConst("a")}},
	{`a`, tiIntConst(0), map[string]*TypeInfo{"a": tiIntConst(0)}},
	{`a`, tiInt64Const(0), map[string]*TypeInfo{"a": tiInt64Const(0)}},
	{`a`, tiInt32Const(0), map[string]*TypeInfo{"a": tiInt32Const(0)}},
	{`a`, tiInt16Const(0), map[string]*TypeInfo{"a": tiInt16Const(0)}},
	{`a`, tiInt8Const(0), map[string]*TypeInfo{"a": tiInt8Const(0)}},
	{`a`, tiUintConst(0), map[string]*TypeInfo{"a": tiUintConst(0)}},
	{`a`, tiUint64Const(0), map[string]*TypeInfo{"a": tiUint64Const(0)}},
	{`a`, tiUint32Const(0), map[string]*TypeInfo{"a": tiUint32Const(0)}},
	{`a`, tiUint16Const(0), map[string]*TypeInfo{"a": tiUint16Const(0)}},
	{`a`, tiUint8Const(0), map[string]*TypeInfo{"a": tiUint8Const(0)}},
	{`a`, tiFloat64Const(0.0), map[string]*TypeInfo{"a": tiFloat64Const(0.0)}},
	{`a`, tiFloat32Const(0.0), map[string]*TypeInfo{"a": tiFloat32Const(0.0)}},

	// Operations ( untyped )
	{`!true`, tiUntypedBoolConst(false), nil},
	{`!false`, tiUntypedBoolConst(true), nil},
	{`+5`, tiUntypedIntConst("5"), nil},
	{`+5.7`, tiUntypedFloatConst("5.7"), nil},
	{`+'a'`, tiUntypedRuneConst('a'), nil},
	{`-5`, tiUntypedIntConst("-5"), nil},
	{`-5.7`, tiUntypedFloatConst("-5.7"), nil},
	{`-'a'`, tiUntypedRuneConst(-'a'), nil},

	// Operations ( typed constant )
	{`!a`, tiBoolConst(false), map[string]*TypeInfo{"a": tiBoolConst(true)}},
	{`!a`, tiBoolConst(true), map[string]*TypeInfo{"a": tiBoolConst(false)}},
	{`+a`, tiIntConst(5), map[string]*TypeInfo{"a": tiIntConst(5)}},
	{`+a`, tiFloat64Const(5.7), map[string]*TypeInfo{"a": tiFloat64Const(5.7)}},
	{`+a`, tiInt32Const('a'), map[string]*TypeInfo{"a": tiInt32Const('a')}},
	{`-a`, tiIntConst(-5), map[string]*TypeInfo{"a": tiIntConst(5)}},
	{`-a`, tiFloat64Const(-5.7), map[string]*TypeInfo{"a": tiFloat64Const(5.7)}},
	{`-a`, tiInt32Const(-'a'), map[string]*TypeInfo{"a": tiInt32Const('a')}},

	// Operations ( typed )
	{`!a`, tiBool(), map[string]*TypeInfo{"a": tiBool()}},
	{`+a`, tiInt(), map[string]*TypeInfo{"a": tiInt()}},
	{`+a`, tiFloat64(), map[string]*TypeInfo{"a": tiFloat64()}},
	{`+a`, tiInt32(), map[string]*TypeInfo{"a": tiInt32()}},
	{`-a`, tiInt(), map[string]*TypeInfo{"a": tiInt()}},
	{`-a`, tiFloat64(), map[string]*TypeInfo{"a": tiFloat64()}},
	{`-a`, tiInt32(), map[string]*TypeInfo{"a": tiInt32()}},
	{`*a`, tiAddrInt(), map[string]*TypeInfo{"a": tiIntPtr()}},
	{`&a`, tiIntPtr(), map[string]*TypeInfo{"a": tiAddrInt()}},
	{`&[]int{}`, &TypeInfo{Type: reflect.PtrTo(reflect.SliceOf(intType))}, nil},
	{`&[...]int{}`, &TypeInfo{Type: reflect.PtrTo(reflect.ArrayOf(0, intType))}, nil},
	{`&map[int]int{}`, &TypeInfo{Type: reflect.PtrTo(reflect.MapOf(intType, intType))}, nil},

	// Operations ( untyped + untyped ).
	{`true && true`, tiUntypedBoolConst(true), nil},
	{`true || true`, tiUntypedBoolConst(true), nil},
	{`false && true`, tiUntypedBoolConst(false), nil},
	{`false || true`, tiUntypedBoolConst(true), nil},
	{`"a" + "b"`, tiUntypedStringConst("ab"), nil},
	{`1 + 2`, tiUntypedIntConst("3"), nil},
	{`1 + 'a'`, tiUntypedRuneConst('b'), nil},
	{`'a' + 'b'`, tiUntypedRuneConst(rune(195)), nil},
	{`1 + 1.2`, tiUntypedFloatConst("2.2"), nil},
	{`'a' + 1.2`, tiUntypedFloatConst("98.2"), nil},
	{`1.5 + 1.2`, tiUntypedFloatConst("2.7"), nil},
	{`"a" + "b"`, tiUntypedStringConst("ab"), nil},
	{`12 & 9`, tiUntypedIntConst("8"), nil},
	{`12 | 9`, tiUntypedIntConst("13"), nil},
	{`12 ^ 9`, tiUntypedIntConst("5"), nil},
	{`12 &^ 9`, tiUntypedIntConst("4"), nil},

	// Operations ( typed + untyped ).
	{`a && true`, tiBoolConst(true), map[string]*TypeInfo{"a": tiBoolConst(true)}},
	{`a || true`, tiBoolConst(true), map[string]*TypeInfo{"a": tiBoolConst(true)}},
	{`a && true`, tiBoolConst(false), map[string]*TypeInfo{"a": tiBoolConst(false)}},
	{`a || true`, tiBoolConst(true), map[string]*TypeInfo{"a": tiBoolConst(false)}},
	{`a + "b"`, tiStringConst("ab"), map[string]*TypeInfo{"a": tiStringConst("a")}},
	{`a + 2`, tiIntConst(3), map[string]*TypeInfo{"a": tiIntConst(1)}},
	{`a + 'a'`, tiIntConst(98), map[string]*TypeInfo{"a": tiIntConst(1)}},
	{`a + 2`, tiInt8Const(3), map[string]*TypeInfo{"a": tiInt8Const(1)}},
	{`a + 2`, tiFloat64Const(3.1), map[string]*TypeInfo{"a": tiFloat64Const(1.1)}},
	{`a + 'a'`, tiFloat64Const(98.1), map[string]*TypeInfo{"a": tiFloat64Const(1.1)}},
	{`a + 2.5`, tiFloat64Const(3.6), map[string]*TypeInfo{"a": tiFloat64Const(1.1)}},
	{`v + 2`, tiInt(), map[string]*TypeInfo{"v": tiInt()}},
	{`v + 2`, tiFloat64(), map[string]*TypeInfo{"v": tiFloat64()}},
	{`v + 2.5`, tiFloat32(), map[string]*TypeInfo{"v": tiFloat32()}},
	{`v & 9`, tiIntConst(8), map[string]*TypeInfo{"v": tiIntConst(12)}},
	{`v | 9`, tiIntConst(13), map[string]*TypeInfo{"v": tiIntConst(12)}},
	{`v ^ 9`, tiIntConst(5), map[string]*TypeInfo{"v": tiIntConst(12)}},
	{`v &^ 9`, tiIntConst(4), map[string]*TypeInfo{"v": tiIntConst(12)}},

	// Operations ( untyped + typed ).
	{`true && a`, tiBoolConst(true), map[string]*TypeInfo{"a": tiBoolConst(true)}},
	{`true || a`, tiBoolConst(true), map[string]*TypeInfo{"a": tiBoolConst(true)}},
	{`true && a`, tiBoolConst(false), map[string]*TypeInfo{"a": tiBoolConst(false)}},
	{`true || a`, tiBoolConst(true), map[string]*TypeInfo{"a": tiBoolConst(false)}},
	{`"b" + a`, tiStringConst("b" + "a"), map[string]*TypeInfo{"a": tiStringConst("a")}},
	{`2 + a`, tiIntConst(2 + int(1)), map[string]*TypeInfo{"a": tiIntConst(1)}},
	{`'a' + a`, tiIntConst('a' + int(1)), map[string]*TypeInfo{"a": tiIntConst(1)}},
	{`2 + a`, tiInt8Const(2 + int8(1)), map[string]*TypeInfo{"a": tiInt8Const(1)}},
	{`2 + a`, tiFloat64Const(2 + float64(1.1)), map[string]*TypeInfo{"a": tiFloat64Const(1.1)}},
	{`'a' + a`, tiFloat64Const('a' + float64(1.1)), map[string]*TypeInfo{"a": tiFloat64Const(1.1)}},
	{`2.5 + a`, tiFloat64Const(2.5 + float64(1.1)), map[string]*TypeInfo{"a": tiFloat64Const(1.1)}},
	{`5.3 / a`, tiFloat64Const(5.3 / float64(1.8)), map[string]*TypeInfo{"a": tiFloat64Const(1.8)}},
	{`2 + v`, tiInt(), map[string]*TypeInfo{"v": tiInt()}},
	{`2 + v`, tiFloat64(), map[string]*TypeInfo{"v": tiFloat64()}},
	{`2.5 + v`, tiFloat32(), map[string]*TypeInfo{"v": tiFloat32()}},
	{`12 & v`, tiIntConst(8), map[string]*TypeInfo{"v": tiIntConst(9)}},
	{`12 | v`, tiIntConst(13), map[string]*TypeInfo{"v": tiIntConst(9)}},
	{`12 ^ v`, tiIntConst(5), map[string]*TypeInfo{"v": tiIntConst(9)}},
	{`12 &^ v`, tiIntConst(4), map[string]*TypeInfo{"v": tiIntConst(9)}},

	// Operations ( typed + typed ).
	{`a && b`, tiBoolConst(true), map[string]*TypeInfo{"a": tiBoolConst(true), "b": tiBoolConst(true)}},
	{`a || b`, tiBoolConst(true), map[string]*TypeInfo{"a": tiBoolConst(true), "b": tiBoolConst(true)}},
	{`a && b`, tiBoolConst(false), map[string]*TypeInfo{"a": tiBoolConst(false), "b": tiBoolConst(true)}},
	{`a || b`, tiBoolConst(true), map[string]*TypeInfo{"a": tiBoolConst(false), "b": tiBoolConst(true)}},
	{`a + b`, tiStringConst("a" + "b"), map[string]*TypeInfo{"a": tiStringConst("a"), "b": tiStringConst("b")}},
	{`a + b`, tiIntConst(int(1) + int(2)), map[string]*TypeInfo{"a": tiIntConst(1), "b": tiIntConst(2)}},
	{`a + b`, tiInt16Const(int16(-3) + int16(5)), map[string]*TypeInfo{"a": tiInt16Const(-3), "b": tiInt16Const(5)}},
	{`a + b`, tiFloat64Const(float64(1.1) + float64(3.7)), map[string]*TypeInfo{"a": tiFloat64Const(1.1), "b": tiFloat64Const(3.7)}},
	{`a / b`, tiFloat64Const(float64(5.3) / float64(1.8)), map[string]*TypeInfo{"a": tiFloat64Const(5.3), "b": tiFloat64Const(1.8)}},
	{`a + b`, tiString(), map[string]*TypeInfo{"a": tiStringConst("a"), "b": tiString()}},
	{`a + b`, tiString(), map[string]*TypeInfo{"a": tiString(), "b": tiStringConst("b")}},
	{`a + b`, tiString(), map[string]*TypeInfo{"a": tiString(), "b": tiString()}},
	{`a & b`, tiIntConst(8), map[string]*TypeInfo{"a": tiIntConst(12), "b": tiIntConst(9)}},
	{`a | b`, tiIntConst(13), map[string]*TypeInfo{"a": tiIntConst(12), "b": tiIntConst(9)}},
	{`a ^ b`, tiIntConst(5), map[string]*TypeInfo{"a": tiIntConst(12), "b": tiIntConst(9)}},
	{`a &^ b`, tiIntConst(4), map[string]*TypeInfo{"a": tiIntConst(12), "b": tiIntConst(9)}},

	// Equality ( untyped + untyped )
	{`false == false`, tiUntypedBoolConst(false == false), nil},
	{`false == true`, tiUntypedBoolConst(false == true), nil},
	{`true == true`, tiUntypedBoolConst(true == true), nil},
	{`true == false`, tiUntypedBoolConst(true == false), nil},
	{`0 == 0`, tiUntypedBoolConst(0 == 0), nil},
	{`0 == 1`, tiUntypedBoolConst(0 == 1), nil},
	{`1 == 1`, tiUntypedBoolConst(1 == 1), nil},
	{`0.0 == 0`, tiUntypedBoolConst(0.0 == 0), nil},
	{`0.1 == 0`, tiUntypedBoolConst(0.1 == 0), nil},
	{`1 == 1.0`, tiUntypedBoolConst(1 == 1.0), nil},
	{`0 == 1.1`, tiUntypedBoolConst(0 == 1.1), nil},
	{`"a" == "a"`, tiUntypedBoolConst("a" == "a"), nil},
	{`"a" == "b"`, tiUntypedBoolConst("a" == "b"), nil},

	// Equality ( typed + untyped )
	{`a == false`, tiBoolConst(bool(false) == false), map[string]*TypeInfo{"a": tiBoolConst(false)}},
	{`a == true`, tiBoolConst(bool(false) == true), map[string]*TypeInfo{"a": tiBoolConst(false)}},
	{`a == 0`, tiBoolConst(int(0) == 0), map[string]*TypeInfo{"a": tiIntConst(0)}},
	{`a == 1`, tiBoolConst(int(1) == 1), map[string]*TypeInfo{"a": tiIntConst(1)}},
	{`a == 0`, tiBoolConst(float64(0.0) == 0), map[string]*TypeInfo{"a": tiFloat64Const(0.0)}},
	{`a == 0`, tiBoolConst(float32(1.0) == 0), map[string]*TypeInfo{"a": tiFloat32Const(1.0)}},
	{`a == 1.0`, tiBoolConst(int(1) == 1.0), map[string]*TypeInfo{"a": tiIntConst(1)}},
	{`a == "a"`, tiBoolConst(string("a") == "a"), map[string]*TypeInfo{"a": tiStringConst("a")}},
	{`a == "b"`, tiBoolConst(string("a") == "b"), map[string]*TypeInfo{"a": tiStringConst("a")}},
	{`a == 0`, tiUntypedBool(), map[string]*TypeInfo{"a": tiInt()}},

	// Shifts.
	{`1 << 1`, tiUntypedIntConst("2"), nil},
	{`a << 1`, tiUntypedIntConst("2"), map[string]*TypeInfo{"a": tiUntypedIntConst("1")}},
	{`a << 1`, tiInt8Const(2), map[string]*TypeInfo{"a": tiInt8Const(1)}},
	{`a << 1`, tiInt(), map[string]*TypeInfo{"a": tiInt()}},
	{`a << 1`, tiInt16(), map[string]*TypeInfo{"a": tiInt16()}},
	{`1 << a`, tiUntypedIntConst("2"), map[string]*TypeInfo{"a": tiUntypedIntConst("1")}},
	{`uint8(1) << a`, tiUint8Const(2), map[string]*TypeInfo{"a": tiUntypedIntConst("1")}},
	{`1 << 511`, tiUntypedIntConst("6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042048"), nil},

	// Index.
	{`"a"[0]`, tiByte(), nil},
	{`a[0]`, tiByte(), map[string]*TypeInfo{"a": tiUntypedStringConst("a")}},
	{`a[0]`, tiByte(), map[string]*TypeInfo{"a": tiStringConst("a")}},
	{`a[0]`, tiByte(), map[string]*TypeInfo{"a": tiAddrString()}},
	{`a[0]`, tiByte(), map[string]*TypeInfo{"a": tiString()}},
	{`"a"[0.0]`, tiByte(), nil},
	{`"ab"[1.0]`, tiByte(), nil},
	{`"abc"[1+1]`, tiByte(), nil},
	{`"abc"[i]`, tiByte(), map[string]*TypeInfo{"i": tiUntypedIntConst("1")}},
	{`"abc"[i]`, tiByte(), map[string]*TypeInfo{"i": tiIntConst(1)}},
	{`"abc"[i]`, tiByte(), map[string]*TypeInfo{"i": tiAddrInt()}},
	{`"abc"[i]`, tiByte(), map[string]*TypeInfo{"i": tiInt()}},
	{`[]int{0,1}[i]`, tiAddrInt(), map[string]*TypeInfo{"i": tiUntypedIntConst("1")}},
	{`[]int{0,1}[i]`, tiAddrInt(), map[string]*TypeInfo{"i": tiUntypedRuneConst('a')}},
	{`[]int{0,1}[i]`, tiAddrInt(), map[string]*TypeInfo{"i": tiUntypedFloatConst("1.0")}},
	{`[]int{0,1}[i]`, tiAddrInt(), map[string]*TypeInfo{"i": tiIntConst(1)}},
	{`[]int{0,1}[i]`, tiAddrInt(), map[string]*TypeInfo{"i": tiInt()}},
	{`[...]int{0,1}[i]`, tiInt(), map[string]*TypeInfo{"i": tiUntypedIntConst("1")}},
	{`[...]int{0,1}[i]`, tiInt(), map[string]*TypeInfo{"i": tiUntypedRuneConst(1)}},
	{`[...]int{0,1}[i]`, tiInt(), map[string]*TypeInfo{"i": tiUntypedFloatConst("1.0")}},
	{`[...]int{0,1}[i]`, tiInt(), map[string]*TypeInfo{"i": tiIntConst(1)}},
	{`[...]int{0,1}[i]`, tiInt(), map[string]*TypeInfo{"i": tiAddrInt()}},
	{`[...]int{0,1}[i]`, tiInt(), map[string]*TypeInfo{"i": tiInt()}},
	{`map[int]int{}[i]`, tiInt(), map[string]*TypeInfo{"i": tiUntypedIntConst("1")}},
	{`map[int]int{}[i]`, tiInt(), map[string]*TypeInfo{"i": tiUntypedRuneConst(1)}},
	{`map[int]int{}[i]`, tiInt(), map[string]*TypeInfo{"i": tiUntypedFloatConst("1.0")}},
	{`map[int]int{}[i]`, tiInt(), map[string]*TypeInfo{"i": tiIntConst(1)}},
	{`map[int]int{}[i]`, tiInt(), map[string]*TypeInfo{"i": tiAddrInt()}},
	{`map[int]int{}[i]`, tiInt(), map[string]*TypeInfo{"i": tiInt()}},
	{`p[1]`, tiAddrInt(), map[string]*TypeInfo{"p": &TypeInfo{Type: reflect.TypeOf(new([2]int))}}},
	{`a[1]`, tiByte(), map[string]*TypeInfo{"a": tiString()}},
	{`a[1]`, tiAddrInt(), map[string]*TypeInfo{"a": &TypeInfo{Type: reflect.TypeOf([]int{0, 1}), Properties: PropertyAddressable}}},
	{`a[1]`, tiAddrInt(), map[string]*TypeInfo{"a": &TypeInfo{Type: reflect.TypeOf([...]int{0, 1}), Properties: PropertyAddressable}}},
	{`a[1]`, tiInt(), map[string]*TypeInfo{"a": &TypeInfo{Type: reflect.TypeOf(map[int]int(nil)), Properties: PropertyAddressable}}},

	// Slicing.
	{`"a"[:]`, tiString(), nil},
	{`a[:]`, tiString(), map[string]*TypeInfo{"a": tiUntypedStringConst("a")}},
	{`a[:]`, tiString(), map[string]*TypeInfo{"a": tiStringConst("a")}},
	{`a[:]`, tiString(), map[string]*TypeInfo{"a": tiAddrString()}},
	{`a[:]`, tiString(), map[string]*TypeInfo{"a": tiString()}},
	{`"a"[1:]`, tiString(), nil},
	{`"a"[1.0:]`, tiString(), nil},
	{`"a"[:0]`, tiString(), nil},
	{`"a"[:0.0]`, tiString(), nil},
	{`"abc"[l:]`, tiString(), map[string]*TypeInfo{"l": tiUntypedIntConst("1")}},
	{`"abc"[l:]`, tiString(), map[string]*TypeInfo{"l": tiUntypedFloatConst("1.0")}},
	{`"abc"[l:]`, tiString(), map[string]*TypeInfo{"l": tiIntConst(1)}},
	{`"abc"[l:]`, tiString(), map[string]*TypeInfo{"l": tiAddrInt()}},
	{`"abc"[l:]`, tiString(), map[string]*TypeInfo{"l": tiInt()}},
	{`"abc"[:h]`, tiString(), map[string]*TypeInfo{"h": tiUntypedIntConst("1")}},
	{`"abc"[:h]`, tiString(), map[string]*TypeInfo{"h": tiUntypedFloatConst("1.0")}},
	{`"abc"[:h]`, tiString(), map[string]*TypeInfo{"h": tiIntConst(1)}},
	{`"abc"[:h]`, tiString(), map[string]*TypeInfo{"h": tiAddrInt()}},
	{`"abc"[:h]`, tiString(), map[string]*TypeInfo{"h": tiInt()}},
	{`"abc"[0:2]`, tiString(), nil},
	{`"abc"[2:2]`, tiString(), nil},
	{`"abc"[3:3]`, tiString(), nil},
	{`[]int{0,1,2}[:]`, tiIntSlice(), nil},
	{`new([3]int)[:]`, tiIntSlice(), nil},
	{`a[:]`, tiIntSlice(), map[string]*TypeInfo{"a": tiIntSlice()}},
	{`a[:]`, tiIntSlice(), map[string]*TypeInfo{"a": tiIntSlice()}},
	{`a[:]`, tiIntSlice(), map[string]*TypeInfo{"a": &TypeInfo{Type: reflect.TypeOf(new([3]int))}}},

	// Conversions ( untyped )
	{`int(5)`, tiIntConst(5), nil},
	{`int8(5)`, tiInt8Const(5), nil},
	{`int16(5)`, tiInt16Const(5), nil},
	{`int32(5)`, tiInt32Const(5), nil},
	{`int64(5)`, tiInt64Const(5), nil},
	{`uint(5)`, tiUintConst(5), nil},
	{`uint8(5)`, tiUint8Const(5), nil},
	{`uint16(5)`, tiUint16Const(5), nil},
	{`uint32(5)`, tiUint32Const(5), nil},
	{`uint64(5)`, tiUint64Const(5), nil},
	{`float32(5.3)`, tiFloat32Const(float32(5.3)), nil},
	{`float64(5.3)`, tiFloat64Const(5.3), nil},
	{`float64(15/3.5)`, tiFloat64Const(15 / 3.5), nil},
	{`int(5.0)`, tiIntConst(5), nil},
	{`int(15/3)`, tiIntConst(5), nil},
	{`string(5)`, tiStringConst(string(5)), nil},
	{`[]byte("abc")`, &TypeInfo{Type: reflect.SliceOf(uint8Type)}, nil},
	{`[]rune("abc")`, &TypeInfo{Type: reflect.SliceOf(int32Type)}, nil},

	// Conversions ( typed constants )
	{`int(a)`, tiIntConst(5), map[string]*TypeInfo{"a": tiIntConst(5)}},
	{`int8(a)`, tiInt8Const(5), map[string]*TypeInfo{"a": tiInt8Const(5)}},
	{`int16(a)`, tiInt16Const(5), map[string]*TypeInfo{"a": tiInt16Const(5)}},
	{`int32(a)`, tiInt32Const(5), map[string]*TypeInfo{"a": tiInt32Const(5)}},
	{`int64(a)`, tiInt64Const(5), map[string]*TypeInfo{"a": tiInt64Const(5)}},
	{`uint(a)`, tiUintConst(5), map[string]*TypeInfo{"a": tiIntConst(5)}},
	{`uint8(a)`, tiUint8Const(5), map[string]*TypeInfo{"a": tiUint8Const(5)}},
	{`uint16(a)`, tiUint16Const(5), map[string]*TypeInfo{"a": tiUint16Const(5)}},
	{`uint32(a)`, tiUint32Const(5), map[string]*TypeInfo{"a": tiUint32Const(5)}},
	{`uint64(a)`, tiUint64Const(5), map[string]*TypeInfo{"a": tiUint64Const(5)}},
	{`float32(a)`, tiFloat32Const(5.3), map[string]*TypeInfo{"a": tiFloat32Const(5.3)}},
	{`float64(a)`, tiFloat64Const(5.3), map[string]*TypeInfo{"a": tiFloat64Const(5.3)}},
	{`float64(a)`, tiFloat64Const(float64(float32(5.3))), map[string]*TypeInfo{"a": tiFloat32Const(5.3)}},
	{`float32(a)`, tiFloat32Const(float32(float64(5.3))), map[string]*TypeInfo{"a": tiFloat64Const(5.3)}},
	{`int(a)`, tiIntConst(5), map[string]*TypeInfo{"a": tiFloat64Const(5.0)}},
	{`[]byte(a)`, &TypeInfo{Type: reflect.SliceOf(uint8Type)}, map[string]*TypeInfo{"a": tiStringConst("abc")}},
	{`[]rune(a)`, &TypeInfo{Type: reflect.SliceOf(int32Type)}, map[string]*TypeInfo{"a": tiStringConst("abc")}},

	// Conversions ( not constants )
	{`int(a)`, tiInt(), map[string]*TypeInfo{"a": tiInt()}},
	{`int8(a)`, tiInt8(), map[string]*TypeInfo{"a": tiInt8()}},
	{`int16(a)`, tiInt16(), map[string]*TypeInfo{"a": tiInt16()}},
	{`int32(a)`, tiInt32(), map[string]*TypeInfo{"a": tiInt32()}},
	{`int64(a)`, tiInt64(), map[string]*TypeInfo{"a": tiInt64()}},
	{`uint(a)`, tiUint(), map[string]*TypeInfo{"a": tiInt()}},
	{`uint8(a)`, tiUint8(), map[string]*TypeInfo{"a": tiUint8()}},
	{`uint16(a)`, tiUint16(), map[string]*TypeInfo{"a": tiUint16()}},
	{`uint32(a)`, tiUint32(), map[string]*TypeInfo{"a": tiUint32()}},
	{`uint64(a)`, tiUint64(), map[string]*TypeInfo{"a": tiUint64()}},
	{`float32(a)`, tiFloat32(), map[string]*TypeInfo{"a": tiFloat32()}},
	{`float64(a)`, tiFloat64(), map[string]*TypeInfo{"a": tiFloat64()}},
	{`float32(a)`, tiFloat32(), map[string]*TypeInfo{"a": tiFloat64()}},
	{`int(a)`, tiInt(), map[string]*TypeInfo{"a": tiFloat64()}},
	{`[]byte(a)`, &TypeInfo{Type: reflect.SliceOf(uint8Type)}, map[string]*TypeInfo{"a": tiString()}},
	{`[]rune(a)`, &TypeInfo{Type: reflect.SliceOf(int32Type)}, map[string]*TypeInfo{"a": tiString()}},
	{`string([]byte{1,2,3})`, tiString(), nil},
	{`string([]rune{'a','b','c'})`, tiString(), nil},

	// append
	{`append([]byte{})`, tiByteSlice(), nil},
	{`append([]string{})`, tiStringSlice(), nil},
	{`append([]byte{}, 'a', 'b', 'c')`, tiByteSlice(), nil},
	{`append([]byte{}, "abc"...)`, tiByteSlice(), nil},
	{`append([]string{}, "a", "b", "c")`, tiStringSlice(), nil},
	{`append([]string{}, []string{"a", "b", "c"}...)`, tiStringSlice(), nil},
	{`append(s, 1, 2, 3)`, tiIntSlice(), map[string]*TypeInfo{"s": tiIntSlice()}},
	{`append(s, 1, 2, 3)`, tiDefinedIntSlice, map[string]*TypeInfo{"s": tiDefinedIntSlice}},
	{`append(s, 1.0, 2.0, 3.0)`, tiDefinedIntSlice, map[string]*TypeInfo{"s": tiDefinedIntSlice}},

	// make
	{`make([]int, 0)`, tiIntSlice(), nil},
	{`make([]int, 0, 0)`, tiIntSlice(), nil},
	{`make([]int, 2, 3)`, tiIntSlice(), nil},
	{`make([]int, 3, 3)`, tiIntSlice(), nil},
	{`make([]int, l, c)`, tiIntSlice(), map[string]*TypeInfo{"l": tiUntypedIntConst("1"), "c": tiUntypedIntConst("1")}},
	{`make([]int, l, c)`, tiIntSlice(), map[string]*TypeInfo{"l": tiIntConst(1), "c": tiIntConst(1)}},
	{`make([]int, l, c)`, tiIntSlice(), map[string]*TypeInfo{"l": tiInt(), "c": tiInt()}},
	{`make([]int, l, c)`, tiIntSlice(), map[string]*TypeInfo{"l": tiUntypedIntConst("1"), "c": tiIntConst(1)}},
	{`make([]int, l, c)`, tiIntSlice(), map[string]*TypeInfo{"l": tiInt(), "c": tiIntConst(1)}},
	{`make(map[string]string)`, tiStringMap(), nil},
	{`make(map[string]string, 0)`, tiStringMap(), nil},
	{`make(map[string]string, s)`, tiStringMap(), map[string]*TypeInfo{"s": tiUntypedIntConst("1")}},
	{`make(map[string]string, s)`, tiStringMap(), map[string]*TypeInfo{"s": tiIntConst(1)}},
	{`make(map[string]string, s)`, tiStringMap(), map[string]*TypeInfo{"s": tiInt()}},
	{`make(chan int)`, tiIntChan(reflect.BothDir), nil},
	{`make(chan<- int)`, tiIntChan(reflect.SendDir), nil},
	{`make(<-chan int)`, tiIntChan(reflect.RecvDir), nil},
	{`make(chan int, 0)`, tiIntChan(reflect.BothDir), nil},
	{`make(chan int, s)`, tiIntChan(reflect.BothDir), map[string]*TypeInfo{"s": tiUntypedIntConst("1")}},
	{`make(chan int, s)`, tiIntChan(reflect.BothDir), map[string]*TypeInfo{"s": tiIntConst(1)}},
	{`make(chan int, s)`, tiIntChan(reflect.BothDir), map[string]*TypeInfo{"s": tiInt()}},

	// cap
	{`cap([]int{})`, tiInt(), nil},
	{`cap([...]byte{})`, tiIntConst(0), nil},
	{`cap(s)`, tiInt(), map[string]*TypeInfo{"s": &TypeInfo{Type: reflect.TypeOf(definedIntSlice{})}}},
	// {`cap(new([1]byte))`, tiInt(), nil}, // TODO.

	// copy
	{`copy([]int{}, []int{})`, tiInt(), nil},
	{`copy([]interface{}{}, []interface{}{})`, tiInt(), nil},
	{`copy([]int{}, s)`, tiInt(), map[string]*TypeInfo{"s": &TypeInfo{Type: reflect.TypeOf(definedIntSlice{})}}},
	{`copy(s, []int{})`, tiInt(), map[string]*TypeInfo{"s": &TypeInfo{Type: reflect.TypeOf(definedIntSlice{})}}},
	{`copy(s1, s2)`, tiInt(), map[string]*TypeInfo{
		"s1": &TypeInfo{Type: reflect.TypeOf(definedIntSlice{})},
		"s2": &TypeInfo{Type: reflect.TypeOf(definedIntSlice2{})},
	}},
	{`copy([]byte{0}, "a")`, tiInt(), nil},
	{`copy(s1, s2)`, tiInt(), map[string]*TypeInfo{
		"s1": &TypeInfo{Type: reflect.TypeOf(definedByteSlice{})},
		"s2": &TypeInfo{Type: reflect.TypeOf(definedStringSlice{})},
	}},

	// new
	{`new(int)`, tiIntPtr(), nil},

	// len
	{`len("")`, tiIntConst(0), nil},
	{`len("a")`, tiIntConst(1), nil},
	{`len([]int{})`, tiInt(), nil},
	{`len(map[string]int{})`, tiInt(), nil},
	{`len([...]byte{})`, tiIntConst(0), nil},
	{`len(s)`, tiInt(), map[string]*TypeInfo{"s": &TypeInfo{Type: reflect.TypeOf(definedIntSlice{})}}},
	// {`len(new([1]byte))`, tiInt(), nil}, // TODO.

	// recover
	{`recover()`, tiInterface(), nil},
}

func TestCheckerExpressions(t *testing.T) {
	for _, expr := range checkerExprs {
		var lex = newLexer([]byte(expr.src), ast.ContextNone)
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*Error); ok {
						t.Errorf("source: %q, %s\n", expr.src, err)
					} else {
						panic(r)
					}
				}
			}()
			var p = &parsing{
				lex:       lex,
				ctx:       ast.ContextNone,
				ancestors: nil,
			}
			node, tok := p.parseExpr(token{}, false, false, false, false)
			if node == nil {
				t.Errorf("source: %q, unexpected %s, expecting expression\n", expr.src, tok)
				return
			}
			scope := make(typeCheckerScope, len(expr.scope))
			for k, v := range expr.scope {
				scope[k] = scopeElement{t: v}
			}
			var scopes []typeCheckerScope
			if expr.scope == nil {
				scopes = []typeCheckerScope{}
			} else {
				scopes = []typeCheckerScope{scope}
			}
			tc := newTypechecker("", false)
			tc.scopes = scopes
			tc.universe = universe
			tc.addScope()
			ti := tc.checkExpression(node)
			err := equalTypeInfo(expr.ti, ti)
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				if testing.Verbose() {
					t.Logf("\nUnexpected:\n%s\nExpected:\n%s\n", dumpTypeInfo(ti), dumpTypeInfo(expr.ti))
				}
			}
		}()
	}
}

var checkerExprErrors = []struct {
	src   string
	err   *Error
	scope map[string]*TypeInfo
}{
	// Index.
	{`"a"["i"]`, tierr(1, 5, `non-integer string index "i"`), nil},
	{`"a"[1.2]`, tierr(1, 5, `constant 1.2 truncated to integer`), nil},
	{`"a"[i]`, tierr(1, 5, `constant 1.2 truncated to integer`), map[string]*TypeInfo{"i": tiUntypedFloatConst("1.2")}},
	{`"a"[nil]`, tierr(1, 5, `non-integer string index nil`), nil},
	{`"a"[i]`, tierr(1, 5, `non-integer string index i`), map[string]*TypeInfo{"i": tiFloat32()}},
	{`5[1]`, tierr(1, 2, `invalid operation: 5[1] (type int does not support indexing)`), nil},
	{`"a"[-1]`, tierr(1, 5, `invalid string index -1 (index must be non-negative)`), nil},
	{`"a"[1]`, tierr(1, 5, `invalid string index 1 (out of bounds for 1-byte string)`), nil},
	{`nil[1]`, tierr(1, 4, `use of untyped nil`), nil},

	// Slicing.
	{`nil[:]`, tierr(1, 4, `use of untyped nil`), nil},
	{`"a"[nil:]`, tierr(1, 5, `invalid slice index nil (type nil)`), nil},
	{`"a"[:nil]`, tierr(1, 6, `invalid slice index nil (type nil)`), nil},
	{`"a"["":]`, tierr(1, 5, `invalid slice index "" (type untyped string)`), nil},
	{`"a"[:""]`, tierr(1, 6, `invalid slice index "" (type untyped string)`), nil},
	{`"a"[true:]`, tierr(1, 5, `invalid slice index true (type untyped bool)`), nil},
	{`"a"[:true]`, tierr(1, 6, `invalid slice index true (type untyped bool)`), nil},
	{`"a"[2:]`, tierr(1, 5, `invalid slice index 2 (out of bounds for 1-byte string)`), nil},
	{`"a"[:2]`, tierr(1, 6, `invalid slice index 2 (out of bounds for 1-byte string)`), nil},
	{`"a"[1:0]`, tierr(1, 4, `invalid slice index: 1 > 0`), nil},
}

func TestCheckerExpressionErrors(t *testing.T) {
	for _, expr := range checkerExprErrors {
		var lex = newLexer([]byte(expr.src), ast.ContextNone)
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*Error); ok {
						err := sameTypeCheckError(err, expr.err)
						if err != nil {
							t.Errorf("source: %q, %s\n", expr.src, err)
							return
						}
					} else {
						panic(r)
					}
				}
			}()
			var p = &parsing{
				lex:       lex,
				ctx:       ast.ContextNone,
				ancestors: nil,
			}
			node, tok := p.parseExpr(token{}, false, false, false, false)
			if node == nil {
				t.Errorf("source: %q, unexpected %s, expecting error %q\n", expr.src, tok, expr.err)
				return
			}
			scope := make(typeCheckerScope, len(expr.scope))
			for k, v := range expr.scope {
				scope[k] = scopeElement{t: v}
			}
			var scopes []typeCheckerScope
			if expr.scope == nil {
				scopes = []typeCheckerScope{}
			} else {
				scopes = []typeCheckerScope{scope}
			}
			tc := newTypechecker("", false)
			tc.scopes = scopes
			tc.universe = universe
			tc.addScope()
			ti := tc.checkExpression(node)
			t.Errorf("source: %s, unexpected %s, expecting error %q\n", expr.src, ti, expr.err)
		}()
	}
}

const ok = ""
const missingReturn = "missing return at end of function"
const noNewVariables = "no new variables on left side of :="
const cannotUseBlankAsValue = "cannot use _ as value"

func declaredNotUsed(v string) string {
	return v + " declared and not used"
}

func redeclaredInThisBlock(v string) string {
	return v + " redeclared in this block"
}

func undefined(v string) string {
	return "undefined: " + v
}

func evaluatedButNotUsed(v string) string {
	return v + " evaluated but not used"
}

// checkerStmts contains some Scrigo snippets with expected type-checker error
// (or empty string if type-checking is valid). Error messages are based upon Go
// 1.12. Tests are subdivided for categories. Each category has a title
// (indicated by a comment), and it's split in two parts: correct source codes
// (which goes first) and bad ones. Correct source codes and bad source codes
// are, respectively, sorted by lexicographical order.
var checkerStmts = map[string]string{

	// Var declarations.
	`var a = 3; _ = a`:             ok,
	`var a int = 1; _ = a`:         ok,
	`var a int; _ = a`:             ok,
	`var a int; a = 3; _ = a`:      ok,
	`var a, b = 1, 2; _, _ = a, b`: ok,
	`var a int = "s"`:              `cannot use "s" (type string) as type int in assignment`,
	`var a, b = 1`:                 "assignment mismatch: 2 variable but 1 values",
	`var a, b int = 1, "2"`:        `cannot use "2" (type string) as type int in assignment`,
	`var a, b, c, d = 1, 2`:        "assignment mismatch: 4 variable but 2 values",
	`f := func() (int, int, int) { return 0, 0, 0 }; var a, b, c string = f()`: `cannot assign int to a (type string) in multiple assignment`,

	// Constant declarations.
	`const a = 2`:     ok,
	`const a int = 2`: ok,
	`const A = 0; B := A; const C = A;   _ = B`: ok,
	`const A = 0; B := A; const C = B;   _ = B`: `const initializer B is not a constant`,
	`const a string = 2`:                        `cannot use 2 (type int) as type string in assignment`, // TODO (Gianluca): Go returns error: cannot convert 2 (type untyped number) to type string

	// Constants - from https://golang.org/ref/spec#Constant_expressions
	`const a = 2 + 3.0`:                      ok,
	`const b = 15 / 4`:                       ok,
	`const c = 15 / 4.0`:                     ok,
	`const j = true`:                         ok,
	`const k = 'w' + 1; const m = string(k)`: ok,
	`const k = 'w' + 1`:                      ok,
	`const l = "hi"`:                         ok,
	`const Θ float64 = 3/2`:                  ok,
	`const Π float64 = 3/2.`:                 ok,
	`const a = 3.14 / 0.0`:                   `division by zero`,
	`const _ = uint(-1)`:                     `constant -1 overflows uint`,
	`const _ = int(3.14)`:                    `constant 3.14 truncated to integer`,
	// `const c = 15 / 4.0; const Θ float64 = 3/2; const ic = complex(0, c)`: ok, // TODO.
	// `const d = 1 << 3.0`:                         ok, // TODO.
	// `const e = 1.0 << 3`:                         ok, // TODO.
	// `const f = int32(1) << 33`:                   `constant 8589934592 overflows int32`, // TODO.
	// `const g = float64(2) >> 1`:                  `invalid operation: float64(2) >> 1 (shift of type float64)`, // TODO.
	// `const h = "foo" > "bar"`:                ok, // TODO.
	// `const Huge = 1 << 100; const Four int8 = Huge >> 98`: ok, // TODO.
	// `const Huge = 1 << 100`:                               ok, // TODO.
	// `const Θ float64 = 3/2; const iΘ = complex(0, Θ)`:                     ok, // TODO.
	// `const Σ = 1 - 0.707i; const Δ = Σ + 2.0e-4`: ok,  // TODO.
	// `const Φ = iota*1i - 1/1i`:                   ok, // TODO.

	// Identifiers.
	`a := 0; a`: evaluatedButNotUsed("a"),

	// Blank identifiers.
	`_ = 1`:                           ok,
	`_, b, c := 1, 2, 3; _, _ = b, c`: ok,
	`var _ = 0`:                       ok,
	`var _, _ = 0, 0`:                 ok,
	`_ := 1`:                          noNewVariables,
	`_, _, _ := 1, 2, 3`:              noNewVariables,
	`_ ++`:                            cannotUseBlankAsValue,
	`_ += 0`:                          cannotUseBlankAsValue,
	`_ = 4 + _`:                       cannotUseBlankAsValue,
	`_ = []_{}`:                       cannotUseBlankAsValue,

	// Assignments (= and :=).
	`(((map[int]string{}[0]))) = ""`:                                ok,
	`a := ((0)); var _ int = a`:                                     ok,
	`a := 0; -a = 1`:                                                `cannot assign to -a`,
	`f := func() (int, int) { return 0, 0 }; _, _ = f()`:            ok,
	`f := func() (int, int) { return 0, 0 }; _, b := f() ; _ = b`:   ok,
	`f := func() int { return 0 } ; var a int = f() ; _ = a`:        ok,
	`map[int]string{}[0] = ""`:                                      ok,
	`v := "s" + "s"; _ = v`:                                         ok,
	`v := 1 + 2; _ = v`:                                             ok,
	`v := 1 + 2; v = 3 + 4; _ = v`:                                  ok,
	`v := 1; _ = v`:                                                 ok,
	`v := 1; v := 2`:                                                noNewVariables,
	`v := 1; v = 2; _ = v`:                                          ok,
	`v1 := 0; v2 := 1; v3 := v2 + v1; _ = v3`:                       ok,
	`a := 0; *a = 1`:                                                `invalid indirect of a (type int)`,
	`a := 0; b := &a; b[0] = 2`:                                     `invalid operation: b[0] (type *int does not support indexing)`,
	`a := 1; a, a = 1, 2`:                                           declaredNotUsed("a"),
	`a, a := 1, 2`:                                                  `a repeated on left side of :=`,
	`f := func() (int, int) { return 0, 0 }; f() = 0`:               `multiple-value f() in single-value context`,
	`f := func() (int, int) { return 0, 0 }; var a, b string = f()`: `cannot assign int to a (type string) in multiple assignment`,
	`f := func() { }; f() = 0`:                                      `f() used as value`,
	`f := func() int { return 0 } ; var a string = f() ; _ = a`:     `cannot use f() (type int) as type string in assignment`,
	`f := func() int { return 0 }; f() = 1`:                         `cannot assign to f()`,
	`len = 0`:                                                       `use of builtin len not in function call`,
	`v = 1`:                                                         undefined("v"),
	`v1 := 1; v2 := "a"; v1 = v2`:                                   `cannot use v2 (type string) as type int in assignment`,

	// Receive.
	`<-aIntChan`:                        ok,
	`_ = <-aIntChan`:                    ok,
	`v := <-aIntChan; _ = v`:            ok,
	`v, ok := <-aIntChan; _, _ = v, ok`: ok,

	// Send.
	`aIntChan <- 5`:    ok,
	`aIntChan <- nil`:  `cannot convert nil to type int`,
	`aIntChan <- 1.34`: `cannot use 1.34 (type float64) as type int in send`,
	// TODO(marco): next test should fail with `cannot convert "a" (type untyped string) to type int`
	`aIntChan <- "a"`:          `cannot use "a" (type string) as type int in send`,
	`aSliceChan <- nil`:        ok,
	`aSliceChan <- []int(nil)`: ok,
	`aSliceChan <- []int{1}`:   ok,

	// Unary operators on untyped nil.
	`!nil`:  `invalid operation: ! nil`,
	`+nil`:  `invalid operation: + nil`,
	`-nil`:  `invalid operation: - nil`,
	`*nil`:  `invalid indirect of nil`,
	`&nil`:  `cannot take the address of nil`,
	`<-nil`: `use of untyped nil`,

	// Increments (++) and decrements (--).
	`a := 1; a++`:   ok,
	`a := ""; a++`:  `invalid operation: a++ (non-numeric type string)`,
	`b++`:           `undefined: b`,
	`a := 5.0; a--`: ok,
	`a := ""; a--`:  `invalid operation: a-- (non-numeric type string)`,
	`b--`:           `undefined: b`,

	// "Compact" assignments (+=, -=, *=, ...).
	`a := 1; a += 1; _ = a`: ok,
	`a := 1; a *= 2; _ = a`: ok,
	`a := ""; a /= 6`:       `cannot convert 6 (type untyped int) to type string`, // TODO (Gianluca): should be "number", not "int"
	`a := ""; a %= 2`:       `cannot convert 2 (type untyped int) to type string`, // TODO (Gianluca): should be "number", not "int"

	// Declarations with self-references.
	`a, b, c := 1, 2, a`:      undefined("a"),
	`const a, b, c = 1, 2, a`: undefined("a"),
	`var a, b, c = 1, 2, a`:   undefined("a"),

	// Interface assignments.
	`i := interface{}(0); _ = i`:                ok,
	`i := interface{}(0); i = 1; _ = i`:         ok,
	`i := interface{}(0); i = 1; i = ""; _ = i`: ok,
	`var i interface{} = interface{}(0); _ = i`: ok,
	`var i interface{}; i = 0; _ = i`:           ok,

	// Type assertions.
	`a := interface{}(3); n, ok := a.(int); var _ int = n; var _ bool = ok`: ok,
	`a := int(3); n, ok := a.(int); var _ int = n; var _ bool = ok`:         `invalid type assertion: a.(int) (non-interface type int on left)`,

	// Slices.
	`_ = [][]string{[]string{"a", "f"}, []string{"g", "h"}}`: ok,
	`_ = []int{}`:      ok,
	`_ = []int{1,2,3}`: ok,
	`_ = [][]int{[]string{"a", "f"}, []string{"g", "h"}}`: `cannot use []string literal (type []string) as type []int in array or slice literal`,
	`_ = []int{-3: 9}`:      `index must be non-negative integer constant`,
	`_ = []int{"a"}`:        `cannot convert "a" (type untyped string) to type int`,
	`_ = []int{1:10, 1:20}`: `duplicate index in array literal: 1`,

	// Arrays.
	`_ = [1]int{1}`:          ok,
	`_ = [5 + 6]int{}`:       ok,
	`_ = [5.0]int{}`:         ok,
	`_ = [-2]int{}`:          `array bound must be non-negative`,
	`_ = [0]int{1}`:          `array index 0 out of bounds [0:0]`,
	`_ = [1]int{10:2}`:       `array index 10 out of bounds [0:1]`,
	`_ = [3]int{1:10, 1:20}`: `duplicate index in array literal: 1`,
	`_ = [5.3]int{}`:         `constant 5.3 truncated to integer`,
	`a := 4; _ = [a]int{}`:   `non-constant array bound a`,

	// Maps.
	`_ = map[string]string{"k1": "v1"}`:    ok,
	`_ = map[string]string{}`:              ok,
	`_ = map[int]int{1: 3, 1: 4}  `:        `duplicate key 1 in map literal`,
	`_ = map[string]int{"a": 3, "a": 4}  `: `duplicate key "a" in map literal`,
	`_ = map[string]string{"k1": 2}`:       `cannot use 2 (type int) as type string in map value`,
	`_ = map[string]string{2: "v1"}`:       `cannot use 2 (type int) as type string in map key`,

	// Map keys.
	`a, ok := map[int]string{}[0]; var _ string = a; var _ bool = ok;`: ok,
	`a, ok := map[int]string{}[0]; var _ string = a; var _ int = ok;`:  `cannot use ok (type bool) as type int in assignment`,

	// Structs.
	`_ = pointInt{}`:           ok,
	`_ = pointInt{1,2}`:        ok,
	`_ = pointInt{1.0,2.0}`:    ok,
	`_ = pointInt{X: 1, Y: 2}`: ok,
	`_ = pointInt{_:0, _:1}`:   `invalid field name _ in struct initializer`,
	`_ = pointInt{"a", "b"}`:   `cannot use "a" (type string) as type int in field value`,
	`_ = pointInt{1, Y: 2}`:    `mixture of field:value and value initializers`,
	`_ = pointInt{1,2,3}`:      `too many values in parser.pointInt literal`,
	//`_ = pointInt{1.2,2.0}`:        `constant 1.2 truncated to integer`, // TODO
	`_ = pointInt{1}`:              `too few values in parser.pointInt literal`,
	`_ = pointInt{X: "a", Y: "b"}`: `cannot use "a" (type string) as type int in field value`,
	`_ = pointInt{X: 1, 2}`:        `mixture of field:value and value initializers`,
	`_ = pointInt{X: 2, X: 2}`:     `duplicate field name in struct literal: X`,

	// Struct fields and methods.
	`_ = (&pointInt{0,0}).X`:    ok,
	`_ = (pointInt{0,0}).X`:     ok,
	`(&pointInt{0,0}).SetX(10)`: ok,
	`_ = (&pointInt{0,0}).Z`:    `&parser.pointInt literal.Z undefined (type *parser.pointInt has no field or method Z)`,       // TODO (Gianluca): '&pointInt literal' should be '(&pointInt literal)'
	`(&pointInt{0,0}).SetZ(10)`: `&parser.pointInt literal.SetZ undefined (type *parser.pointInt has no field or method SetZ)`, // TODO (Gianluca): '&pointInt literal' should be '(&pointInt literal)'
	`(pointInt{0,0}).SetZ(10)`:  `parser.pointInt literal.SetZ undefined (type parser.pointInt has no field or method SetZ)`,   // TODO (Gianluca): '&pointInt literal' should be '(&pointInt literal)'

	// Expressions.
	`int + 2`: `type int is not an expression`,
	`0`:       evaluatedButNotUsed("0"),

	// Address operator.
	`var a int; _ = &((a))`:             ok,
	`var a int; _ = &a`:                 ok,
	`_ = &[]int{}`:                      ok,
	`_ = &[10]int{}`:                    ok,
	`_ = &map[int]string{}`:             ok,
	`var a int; &a`:                     evaluatedButNotUsed("&a"),
	`_ = &[]int{1}[0]`:                  ok,
	`var a int; var b *int = &a; _ = b`: ok,
	`_ = &(_)`:                          `cannot use _ as value`,
	`_ = &(0)`:                          `cannot take the address of 0`,

	// Pointer indirection operator.
	`var a int; b := &a; var c int = *b; _ = c`: ok,

	// Pointer types.
	`a := 0; var _ *a`:                    `*a is not a type`,
	`var _ *int`:                          ok,
	`var _ *map[string]interface{}`:       ok,
	`var _ *(((map[string]interface{})))`: ok,
	`var _ map[*int][]*string`:            ok,

	// Shifts.
	`_ = 1 << nil`:                     `cannot convert nil to type uint`,
	`_ = 1 << "s"`:                     `invalid operation: 1 << "s" (shift count type string, must be unsigned integer)`,
	`_ = 1 << 1.2`:                     `invalid operation: 1 << 1.2 (shift count type float64, must be unsigned integer)`, // NOTE: gc returns `constant 1.2 truncated to integer`
	`_ = 1 << -1`:                      `invalid negative shift count: -1`,
	`_ = 1 << 512`:                     `shift count too large: 512`,
	`const a string = "s"; _ = 1 << a`: `invalid operation: 1 << a (shift count type string, must be unsigned integer)`,
	`const a int = -1; _ = 1 << a`:     `invalid operation: 1 << a (shift count type int, must be unsigned integer)`,
	`var a = "s"; _ = 1 << a`:          `invalid operation: 1 << a (shift count type string, must be unsigned integer)`,
	`var a = 1.2; _ = 1 << a`:          `invalid operation: 1 << a (shift count type float64, must be unsigned integer)`,
	`var a = -1; _ = 1 << a`:           `invalid operation: 1 << a (shift count type int, must be unsigned integer)`,
	`_ = nil << 1`:                     `invalid operation: nil << 1 (shift of type nil)`,
	`_ = "a" << 1`:                     `invalid operation: "a" << 1 (shift of type untyped string)`,
	`_ = 1.2 << 1`:                     `invalid operation: 1.2 << 1 (shift of type untyped float64)`, // NOTE: gc returns `constant 1.2 truncated to integer`
	`_ = 1 << 1`:                       ok,
	`_ = 1 << 1.0`:                     ok,
	`_ = 1 << 511`:                     ok,
	`_ = -1 << 1`:                      ok,
	// `_ = 1.0 << 1`:                     ok, TODO(marco)

	// Blocks.
	`{ a := 1; a = 10; _ = a }`:            ok,
	`{ a := 1; { a = 10; _ = a }; _ = a }`: ok,
	`{ a := 1; a := 2}`:                    noNewVariables,
	`{ { { a := 1; a := 2 } } }`:           noNewVariables,

	// If statements.
	`if 1 == 1 { }`:                        ok,
	`if a := 1; a == 2 { }`:                ok,
	`if a := 1; a == 2 { b := a ; _ = b }`: ok,
	`if true { }`:                          ok,
	`if 1 { }`:                             "non-bool 1 (type int) used as if condition",
	`if 1 == 1 { a := 3 ; _ = a }; a = 1`:  "undefined: a",
	`if false { a }`:                       undefined("a"),
	`if x { }`:                             undefined("x"),

	// For statements with single condition.
	`for true { }`:    ok,
	`for 10 > 20 { }`: ok,
	`for 3 { }`:       "non-bool 3 (type int) used as for condition",

	// For statements with 'for' clause.
	`for i := 0; i < 10; i++ { }`:                                      ok,
	`for i := 0; i < 10; {}`:                                           ok,
	`for i := 0; i < 10; _ = 2 {}`:                                     ok,
	`s := []int{}; for i := range s { _ = i }`:                         ok,
	`s := []int{}; for i, v := range s { _, _ = i, v }`:                ok,
	`s := []int{0,1}; for i    := range s { _ = s[i] }`:                ok,
	`s := []int{0,1}; for _, i := range s { _ = s[i] }`:                ok,
	`s := []string{"a","b"}; for i := range s { _ = s[i] }`:            ok,
	`s := []string{"a","b"}; for i := 0; i < len(s); i++ { _ = s[i] }`: ok,
	`for i := 10; i; i++ { }`:                                          "non-bool i (type int) used as for condition",
	`for i := 0; i < 10; i = "" {}`:                                    `cannot use "" (type string) as type int in assignment`,
	`s := []string{"a","b"}; for _, i := range s { _ = s[i] }`:         `non-integer slice index i`,

	// For statements with 'range' clause.
	`for _, _ = range "abc" { }`:                                                     ok,
	`for _, _ = range []int{1,2,3} { }`:                                              ok,
	`for k, v := range ([...]int{}) { var _, _ int = k, v }`:                         ok,
	`for k, v := range map[float64]string{} { var _ float64 = k; var _ string = v }`: ok,
	`for _, _ = range (&[...]int{}) { }`:                                             ok,
	`for _, _ = range 0 { }`:                                                         `cannot range over 0 (type untyped int)`, // TODO (Gianluca): should be 'number', not int.
	`for _, _ = range (&[]int{}) { }`:                                                `cannot range over &[]int literal (type *[]int)`,
	`for a, b, c := range "" { }`:                                                    `too many variables in range`,

	// Switch (expression) statements.
	`switch 1 { case 1: }`:                  ok,
	`switch 1 + 2 { case 3: }`:              ok,
	`switch true { case true: }`:            ok,
	`a := false; switch a { case true: }`:   ok,
	`a := false; switch a { case 4 > 10: }`: ok,
	`a := false; switch a { case a: }`:      ok,
	`a := 3; switch a { case a: }`:          ok,
	`switch 1 + 2 { case "3": }`:            `invalid case "3" in switch on 1 + 2 (mismatched types string and int)`,
	`a := 3; switch a { case a > 2: }`:      `invalid case a > 2 in switch on a (mismatched types bool and int)`,
	`a := 3; switch 0.0 { case a: }`:        `invalid case a in switch on 0 (mismatched types int and float64)`,

	// Type-switch statements.
	`i := interface{}(int(0)); switch i.(type) { }`:                         ok,
	`i := interface{}(int(0)); switch i.(type) { case int: case float64: }`: ok,
	`i := interface{}(int(0)); switch i.(type) { case 2: case float64: }`:   `2 (type untyped number) is not a type`,
	`i := 0; switch i.(type) { }`:                                           `cannot type switch on non-interface value i (type int)`,

	// Function literals definitions.
	`_ = func(     )         {                                             }`: ok,
	`_ = func(     )         { return                                      }`: ok,
	`_ = func(int  )         {                                             }`: ok,
	`_ = func(     )         { if true { }; { a := 10; { _ = a } ; _ = a } }`: ok,
	`_ = func(     )         { a                                           }`: `undefined: a`,
	`_ = func(     )         { 7 == "hey"                                  }`: `invalid operation: 7 == "hey" (mismatched types int and string)`,
	`_ = func(     )         { if true { }; { a := 10; { _ = b } ; _ = a } }`: `undefined: b`,
	`_ = func(     ) (s int) { s := 0; return 0                            }`: `no new variables on left side of :=`,
	`_ = func(s int)         { s := 0; _ = s                               }`: `no new variables on left side of :=`,

	// Slice expressions.
	`a := [5]int{1, 2, 3, 4, 5}; var _ []int = a[:2]`:   ok,
	`a := [5]int{1, 2, 3, 4, 5}; var _ []int = a[1:]`:   ok,
	`a := [5]int{1, 2, 3, 4, 5}; var _ []int = a[1:4]`:  ok,
	`a := [5]int{1, 2, 3, 4, 5}; var _ [3]int = a[1:4]`: `cannot use a[1:4] (type []int) as type [3]int in assignment`,

	// Terminating statements - https://golang.org/ref/spec#Terminating_statements (misc)
	`_ = func() int { a := 2; _ = a                                     }`: missingReturn,
	`_ = func() int {                                                   }`: missingReturn,

	// Terminating statements - https://golang.org/ref/spec#Terminating_statements (1)
	`_ = func() int { return 1                                          }`: ok, // (1)

	// Terminating statements - https://golang.org/ref/spec#Terminating_statements (3)
	`_ = func() int { { return 0 }                                      }`: ok,
	`_ = func() int { { }                                               }`: missingReturn,

	// Terminating statements - https://golang.org/ref/spec#Terminating_statements (4)
	`_ = func() int { if true { return 1 } else { return 2 }            }`: ok,
	`_ = func() int { if true { return 1 } else { }                     }`: missingReturn,
	`_ = func() int { if true { } else { }                              }`: missingReturn,
	`_ = func() int { if true { } else { return 1 }                     }`: missingReturn,

	// Terminating statements - https://golang.org/ref/spec#Terminating_statements (5)
	`_ = func() int { for { }                                           }`: ok,
	`_ = func() int { for { break }                                     }`: missingReturn,
	`_ = func() int { for { { break } }                                 }`: missingReturn,
	`_ = func() int { for true { }                                      }`: missingReturn,
	`_ = func() int { for i := 0; i < 10; i++ { }                       }`: missingReturn,

	// Terminating statements - https://golang.org/ref/spec#Terminating_statements (6)
	`_ = func() int { switch { case true: return 0; default: return 0 } }`: ok,
	`_ = func() int { switch { case true: fallthrough; default: }       }`: ok,
	`_ = func() int { switch { }                                        }`: missingReturn,
	`_ = func() int { switch { case true: return 0; default:  }         }`: missingReturn,

	// Return statements with named result parameters.
	`_ = func() (a int)           { return             }`: ok,
	`_ = func() (a int, b string) { return             }`: ok,
	`_ = func() (a int, b string) { return 0, ""       }`: ok,
	`_ = func() (s int)           { { s := 0; return } }`: `s is shadowed during return`,
	`_ = func() (a int)           { return ""          }`: `cannot use "" (type string) as type int in return argument`,
	`_ = func() (a int, b string) { return "", ""      }`: `cannot use "" (type string) as type int in return argument`,
	`_ = func() (a int)           { return 0, 0        }`: "too many arguments to return\n\thave (number, number)\n\twant (int)",
	`_ = func() (a int, b string) { return 0           }`: "not enough arguments to return\n\thave (number)\n\twant (int, string)",

	// Result statements with non-named result parameters.
	`_ = func() int { return 0 }`:              ok,
	`_ = func() int { return "" }`:             `cannot use "" (type string) as type int in return argument`,
	`_ = func() (int, string) { return 0 }`:    "not enough arguments to return\n\thave (number)\n\twant (int, string)",
	`_ = func() (int, int) { return 0, 0, 0}`:  "too many arguments to return\n\thave (number, number, number)\n\twant (int, int)",
	`_ = func() (int, int) { return 0, "", 0}`: "too many arguments to return\n\thave (number, string, number)\n\twant (int, int)",

	// Return statements with functions as return value.
	`f := func () (int, int) { return 0, 0 }; _ = func() (int, int) { return f() }`:         ok,
	`f := func () (int, int, int) { return 0, 0, 0 }; _ = func() (int, int) { return f() }`: "too many arguments to return\n\thave (int, int, int)\n\twant (int, int)",
	`f := func () int { return 0 }; _ = func() (int, int) { return f() }`:                   "not enough arguments to return\n\thave (int)\n\twant (int, int)",
	`f := func () (string, string) { return "", "" }; _ = func() (int, int) { return f() }`: `cannot use f() (type string) as type int in return argument`, // TODO (Gianluca): should be cannot use string as type int in return argument
	// `var f func () (int, int); _ = func() (int, int) { return f() }`: ok, // TODO (Gianluca): parsing error.

	// Function literal calls.
	`f := func() { }; f()`:                                            ok,
	`f := func(int) { }; f(0)`:                                        ok,
	`f := func(a, b int) { }; f(0, 0)`:                                ok,
	`f := func(a string, b int) { }; f("", 0)`:                        ok,
	`f := func() (a, b int) { return 0, 0 }; f()`:                     ok,
	`var _, _ int = func(a, b int) (int, int) { return a, b }(0, 0)`:  ok,
	`f := func(a, b int) { }; f("", 0)`:                               `cannot use "" (type string) as type int in argument to f`,
	`f := func(string) { } ; f(0)`:                                    `cannot use 0 (type int) as type string in argument to f`,
	`f := func(string, int) { } ; f(0)`:                               "not enough arguments in call to f\n\thave (number)\n\twant (string, int)",
	`f := func(string, int) { } ; f(0, 0, 0)`:                         "too many arguments in call to f\n\thave (number, number, number)\n\twant (string, int)",
	`f := func() (a, b int) { return 0, "" }; f()`:                    `cannot use "" (type string) as type int in return argument`,
	`var _, _ int = func(a, b int) (int, int) { return a, b }("", 0)`: `cannot use "" (type string) as type int in argument to func literal`,
	// `f := func(n ...int) { for _ = range n { } }; f(1,2,3)`:           ok, // TODO.
	// `func(c int) { _ = c == 0 && c == 0 }(0)`:      ok, // TODO (Gianluca): panics.

	// Function literal calls with function call as argument.
	`f := func() (int, int) { return 0, 0 } ; g := func(int, int) { } ; g(f())`:         ok,
	`f := func() int { return 0 } ; g := func(int, int) { } ; g(f())`:                   "not enough arguments in call to g\n\thave (int)\n\twant (int, int)",
	`f := func() (string, int) { return "", 0 } ; g := func(int, int) { } ; g(f())`:     `cannot use string as type int in argument to g`,
	`f := func() (int, int, int) { return 0, 0, 0 } ; g := func(int, int) { } ; g(f())`: "too many arguments in call to g\n\thave (int, int, int)\n\twant (int, int)",

	// Variadic functions and calls.
	`f := func(a ...int) { } ; f(nil...)`:        ok,
	`f := func(a ...int) { } ; f([]int(nil)...)`: ok,
	`f := func(a ...int) { } ; f([]int{1,2}...)`: ok,

	// Variadic function literals.
	`f := func(a int, b...int)  { b[0] = 1 };  f(1);               f(1);  f(1,2,3)`: ok,
	`f := func(a... int)        { a[0] = 1 };  f([]int{1,2,3}...)`:                  ok,
	`f := func(a... int)        { a[0] = 1 };  f();                f(1);  f(1,2,3)`: ok,
	`f := func(a... int) { a[0] = 1 };  f([]string{"1","2","3"}...)`:                `cannot use []string literal (type []string) as type []int in argument to f`,
	`f := func(a... int) { a[0] = 1 };  var a int; f(a...)`:                         `cannot use a (type int) as type []int in argument to f`,
	`f := func(a, b, c int, d... int) {  };  f(1,2)`:                                "not enough arguments in call to f\n\thave (number, number)\n\twant (int, int, int, ...int)",

	// Conversions.
	`int()`:     `missing argument to conversion to int: int()`,
	`int(0, 0)`: `too many arguments to conversion to int: int(0, 0)`,
	// `int(nil)`:  `cannot convert nil to type int`, // TODO
	// `float64("a")`: `cannot convert "a" (type untyped string) to type float64`, // TODO

	// Function calls.
	`a := 0; a()`:                  `cannot call non-function a (type int)`,
	`a := []int{}; a()`:            `cannot call non-function a (type []int)`,
	`f := func(a int) {} ; f(nil)`: `cannot use nil as type int in argument to f`,
	// `nil.F()`:     `use of untyped nil`, // TODO

	// Variable declared and not used.
	`a := 0; { _ = a }`:          ok,
	`{ { a := 0 } }`:             declaredNotUsed("a"),
	`{ const A = 0; var B = 0 }`: declaredNotUsed("B"),
	`a := 0; { b := 0 }`:         declaredNotUsed("b"),
	`a := 0; a = 1`:              declaredNotUsed("a"),
	`a := 0`:                     declaredNotUsed("a"),

	// Redeclaration (variables and constants) in the same block.
	`{ const A = 0 }`:                 ok,
	`var A = 0; _ = A`:                ok,
	`{ const A = 0; var A = 0 }`:      redeclaredInThisBlock("A"),
	`A := 0; var A = 1`:               redeclaredInThisBlock("A"),
	`const A = 0; const A = 1; _ = A`: redeclaredInThisBlock("A"),
	`var A = 0; var A = 1; _ = A`:     redeclaredInThisBlock("A"),

	// Assignment of unsigned values.
	`var a bool = 1 == 1; _ = a`:        ok,
	`var a boolType = 1 == 1; _ = a`:    ok,
	`var a int = 5; _ = a`:              ok,
	`var a interface{} = 1 == 1; _ = a`: ok,
	`var a interface{} = 5; _ = a`:      ok,
	`var a stringType = "a"; _ = a`:     ok,

	// Types and expressions.
	`var _ int`:       ok,
	`a := 0; var _ a`: `a is not a type`,

	// Builtin functions 'print' and 'println'.
	`print()`:         ok,
	`print("a")`:      ok,
	`print("a", 5)`:   ok,
	`println()`:       ok,
	`println("a")`:    ok,
	`println("a", 5)`: ok,
	`println = 0`:     `use of builtin println not in function call`,

	// Builtin function 'append'.
	`_ = append([]int{}, 0)`:     ok,
	`append := 0; _ = append`:    ok,
	`_ = append + 3`:             `use of builtin append not in function call`,
	`a, b := append([]int{}, 0)`: `assignment mismatch: 2 variable but 1 values`,
	`append()`:                   `missing arguments to append`,
	`append([]int{}, 0)`:         evaluatedButNotUsed("append([]int literal, 0)"),
	`append(0)`:                  `first argument to append must be slice; have untyped number`,
	`append(nil)`:                `first argument to append must be typed slice; have untyped nil`,

	// Builtin function 'copy'.
	`_ = copy([]int{}, []int{})`:     ok,
	`copy([]int{}, []int{})`:         ok,
	`_ = copy + copy`:                `use of builtin copy not in function call`,
	`a, b := copy([]int{}, []int{})`: `assignment mismatch: 2 variable but 1 values`,
	`copy([]int{},[]string{})`:       `arguments to copy have different element types: []int and []string`,
	`copy([]int{},0)`:                `second argument to copy should be slice or string; have int`,
	`copy(0,[]int{})`:                `first argument to copy should be slice; have int`,
	`copy(0,0)`:                      `arguments to copy must be slices; have int, int`,

	// Builtin function 'delete'.
	`delete(aStringMap, "a")`:                  ok,
	`delete(map[string]string{}, "a")`:         ok,
	`delete(map[stringType]string{}, aString)`: ok,
	`delete(map[string]string{}, 10 + 2)`:      `cannot use 10 + 2 (type int) as type string in delete`, // TODO.
	`delete(map[string]string{}, nil)`:         `cannot use nil as type string in delete`,
	`delete(nil, 0)`:                           `first argument to delete must be map; have nil`,

	// Builtin function 'len'.
	`_ = len([]int{})`:           ok,
	`len()`:                      `missing argument to len: len()`,
	`len([]string{"", ""})`:      evaluatedButNotUsed("len([]string literal)"),
	`len(0)`:                     `invalid argument 0 (type int) for len`,
	`len(nil)`:                   `use of untyped nil`,
	`len := 0; _ = len`:          ok,
	`const _ = len("")`:          ok,
	`const _ = len([...]byte{})`: ok,
	// `const _ = len(new([1]byte))`: `const initializer len(new([1]byte)) is not a constant`, // TODO.

	// Builtin function 'cap'.
	`_ = cap([]int{})`:          ok,
	`const _ = cap([...]int{})`: ok,
	`const _ = cap([2]int{})`:   ok,
	`cap()`:                     `missing argument to cap: cap()`,
	`cap(0)`:                    `invalid argument 0 (type int) for cap`,
	`cap(nil)`:                  `use of untyped nil`,
	`cap([]int{})`:              evaluatedButNotUsed("cap([]int literal)"),
	`const _ = cap([]int{})`:    `const initializer cap([]int literal) is not a constant`,

	// Builtin function 'make'.
	`_ = make(map[int]int)`:   ok,
	`make()`:                  `missing argument to make`,
	`make([]int, nil)`:        `non-integer len argument in make([]int) - nil`,
	`make([]int, -1)`:         `negative len argument in make([]int)`,
	`make([]int, "")`:         `non-integer len argument in make([]int) - untyped string`,
	`make([]int, []int{})`:    `non-integer len argument in make([]int) - []int`,
	`make([]int, 0,0,0)`:      `too many arguments to make([]int)`,
	`make([]int, 1, -1)`:      `negative cap argument in make([]int)`,
	`make([]int, 1, "")`:      `non-integer cap argument in make([]int) - untyped string`,
	`make([]int, 1, 1.2)`:     `constant 1.2 truncated to integer`,
	`make([]int, 1.2)`:        `constant 1.2 truncated to integer`,
	`make([]int, 10, 1)`:      `len larger than cap in make([]int)`,
	`make([]int)`:             `missing len argument to make([]int)`,
	`make([2]int)`:            `cannot make type [2]int`,
	`make(map[int]int, nil)`:  `cannot convert nil to type int`,
	`make(map[int]int, -1)`:   `negative size argument in make(map[int]int)`,
	`make(map[int]int, "")`:   `non-integer size argument in make(map[int]int) - string`,
	`make(map[int]int, 0, 0)`: `too many arguments to make(map[int]int)`,
	`make(map[int]int)`:       evaluatedButNotUsed("make(map[int]int)"),
	`make(string)`:            `cannot make type string`,
	`make(chan int, nil)`:     `cannot convert nil to type int`,
	`make(chan int, -1)`:      `negative buffer argument in make(chan int)`,
	`make(chan int, "")`:      `non-integer buffer argument in make(chan int) - string`,
	`make(chan int, 0, 0)`:    `too many arguments to make(chan int)`,
	`make(chan int)`:          evaluatedButNotUsed("make(chan int)"),

	// Builtin function 'new'.
	`_ = new(int)`: ok,
	`new()`:        `missing argument to new`,
	`new(int)`:     evaluatedButNotUsed("new(int)"),

	// Builtin function 'recover'.
	`recover()`:                 ok,
	`recover(1)`:                `too many arguments to recover`,
	`recover := 0; _ = recover`: ok,

	// Type definitions.
	`type  ( T1 int ; T2 string; T3 map[T1]T2 ) ; _ = T3{0:"a"}`: ok,
	`type T int            ; var _ T = T(0)`:                     ok,
	`type T interface{}    ; var _ T`:                            ok,
	`type T map[string]int ; _ = T{"one": 1}`:                    ok,
	`type T string         ; _ = []T{"a", "b"}`:                  ok,
	`type T T2`: undefined("T2"),
	// `type T int            ; _ = []T{"a", "b"}`:    `cannot convert "a" (type untyped string) to type T`, // TODO.
	// `type T float64        ; _ = T("a")`:           `cannot convert "a" (type untyped string) to type T`, // TODO.
	// `type T float64        ; var _ T = float64(0)`: `cannot use float64(0) (type float64) as type T in assignment`, // TODO.

	// Alias declarations.
	`type T = float64 ; var _ T = float64(0)`: ok,
	`type T = float64 ; var _ T = int(0)`:     `cannot use int(0) (type int) as type float64 in assignment`,

	// Struct types.
	`_ = struct{ A int }{A: 10}`:          ok,
	`_ = struct{ A struct { A2 int } }{}`: ok,
	`type S struct { }`:                   ok,
	`type S struct{A,B int ; C,D float64} ; _ = S{A: 5, B: 10, C: 3.4, D: 1.1}`: ok,
	`type S struct{A,B int} ; _ = S{A: 5, B: 10}`:                               ok,
	`type S1 struct { A int ; B map[string][]int; *int }`:                       ok,
	`_ = struct{ A int }{C: 10}`:                                                `unknown field 'C' in struct literal of type struct { A int }`,
	`type S struct{A,B int ; C,D float64} ; _ = S{A: 5, B: 10, C: 3.4, D: ""}`:  `cannot use "" (type string) as type float64 in field value`,
}

type pointInt struct{ X, Y int }

func (p *pointInt) SetX(newX int) {
	p.X = newX
}

func TestCheckerStatements(t *testing.T) {
	scope := typeCheckerScope{
		"boolType":    {t: &TypeInfo{Properties: PropertyIsType, Type: reflect.TypeOf(definedBool(false))}},
		"aString":     {t: &TypeInfo{Type: reflect.TypeOf(definedString(""))}},
		"stringType":  {t: &TypeInfo{Properties: PropertyIsType, Type: reflect.TypeOf(definedString(""))}},
		"aStringMap":  {t: &TypeInfo{Type: reflect.TypeOf(definedStringMap{})}},
		"pointInt":    {t: &TypeInfo{Properties: PropertyIsType, Type: reflect.TypeOf(pointInt{})}},
		"interface{}": {t: &TypeInfo{Type: reflect.TypeOf(&[]interface{}{interface{}(nil)}[0]).Elem(), Properties: PropertyIsType}},
		"aIntChan":    {t: &TypeInfo{Type: reflect.TypeOf(make(chan int))}},
		"aSliceChan":  {t: &TypeInfo{Type: reflect.TypeOf(make(chan []int))}},
	}
	for src, expectedError := range checkerStmts {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*Error); ok {
						if expectedError == "" {
							t.Errorf("source: '%s' should be 'ok' but got error: %q", src, err)
						} else if !strings.Contains(err.Error(), expectedError) {
							t.Errorf("source: '%s' should return error: %q but got: %q", src, expectedError, err)
						}
					} else {
						panic(fmt.Errorf("source %q: %s", src, r))
					}
				} else {
					if expectedError != ok {
						t.Errorf("source: '%s' expecting error: %q, but no errors have been returned by type-checker", src, expectedError)
					}
				}
			}()
			tree, err := ParseSource([]byte(src), ast.ContextNone)
			if err != nil {
				t.Errorf("source: %s returned parser error: %s", src, err.Error())
				return
			}
			tc := newTypechecker("", false)
			tc.scopes = append(tc.scopes, scope)
			tc.universe = universe
			tc.addScope()
			tc.checkNodes(tree.Nodes)
			tc.removeCurrentScope()
		}()
	}
}

// tiEquals checks that t1 and t2 are identical.
func equalTypeInfo(t1, t2 *TypeInfo) error {
	if t1.Type == nil && t2.Type != nil {
		return fmt.Errorf("unexpected type %s, expecting untyped", t2.Type)
	}
	if t1.Type != nil && t2.Type == nil {
		return fmt.Errorf("unexpected untyped, expecting type %s", t1.Type)
	}
	if t1.Type != nil && t1.Type != t2.Type {
		return fmt.Errorf("unexpected type %s, expecting %s", t2.Type, t1.Type)
	}
	if t1.Nil() && !t2.Nil() {
		return fmt.Errorf("unexpected non-predeclared nil")
	}
	if !t1.Nil() && t2.Nil() {
		return fmt.Errorf("unexpected predeclared nil")
	}
	if t1.Untyped() && !t2.Untyped() {
		return fmt.Errorf("unexpected typed")
	}
	if !t1.Untyped() && t2.Untyped() {
		return fmt.Errorf("unexpected untyped")
	}
	if t1.IsConstant() && !t2.IsConstant() {
		return fmt.Errorf("unexpected non-constant")
	}
	if !t1.IsConstant() && t2.IsConstant() {
		return fmt.Errorf("unexpected constant")
	}
	if t1.IsType() && !t2.IsType() {
		return fmt.Errorf("unexpected non-type")
	}
	if !t1.IsType() && t2.IsType() {
		return fmt.Errorf("unexpected type")
	}
	if t1.IsBuiltin() && !t2.IsBuiltin() {
		return fmt.Errorf("unexpected non-builtin")
	}
	if !t1.IsBuiltin() && t2.IsBuiltin() {
		return fmt.Errorf("unexpected builtin")
	}
	if t1.Addressable() && !t2.Addressable() {
		return fmt.Errorf("unexpected not addressable")
	}
	if !t1.Addressable() && t2.Addressable() {
		return fmt.Errorf("unexpected addressable")
	}
	if t1.Value == nil && t2.Value != nil {
		return fmt.Errorf("unexpected value")
	}
	if t1.Value != nil && t2.Value == nil {
		return fmt.Errorf("unexpected nil value")
	}
	if t1.Value != nil {
		switch v1 := t1.Value.(type) {
		case int64:
			switch v2 := t2.Value.(type) {
			case int64:
				if v1 != v2 {
					return fmt.Errorf("unexpected integer %d, expecting %d", v2, v1)
				}
				return nil
			case *big.Int:
				if v2.Cmp(big.NewInt(v1)) != 0 {
					return fmt.Errorf("unexpected integer %s, expecting %d", v2, v1)
				}
				return nil
			}
		case *big.Int:
			switch v2 := t2.Value.(type) {
			case int64:
				if v1.Cmp(big.NewInt(v2)) != 0 {
					return fmt.Errorf("unexpected integer %d, expecting %s", v2, v1)
				}
				return nil
			case *big.Int:
				if v1.Cmp(v2) != 0 {
					return fmt.Errorf("unexpected integer %s, expecting %s", v2, v1)
				}
				return nil
			}
		case float64:
			switch v2 := t2.Value.(type) {
			case float64:
				if v1 != v2 {
					return fmt.Errorf("unexpected integer %f, expecting %f", v2, v1)
				}
				return nil
			case *big.Float:
				if v2.Cmp(big.NewFloat(v1)) != 0 {
					return fmt.Errorf("unexpected integer %s, expecting %f", v2, v1)
				}
				return nil
			}
		case *big.Float:
			switch v2 := t2.Value.(type) {
			case float64:
				if v1.Cmp(big.NewFloat(v2)) != 0 {
					return fmt.Errorf("unexpected floating-point %v, expecting %v", big.NewFloat(v2).Prec(), v1.Prec())
				}
				return nil
			case *big.Float:
				if v1.Cmp(v2) != 0 {
					return fmt.Errorf("unexpected floating-point %v, expecting %v",
						v2.Text('f', 53), v1.Text('f', 53))
				}
				return nil
			}
		case *big.Rat:
			v2 := t2.Value.(*big.Rat)
			if v1.Cmp(v2) != 0 {
				return fmt.Errorf("unexpected floating-point %v, expecting %v", v2, v1)
			}
		default:
			if t1.Value != t2.Value {
				return fmt.Errorf("unexpected value %v, expecting %v", t2.Value, t1.Value)
			}
			return nil
		}
		return fmt.Errorf("unexpected value type %T, expecting %T", t2.Value, t1.Value)
	}
	return nil
}

func dumpTypeInfo(ti *TypeInfo) string {
	s := "\tType:"
	if ti.Type != nil {
		s += " " + ti.Type.String()
	}
	s += "\n\tProperties:"
	if ti.Nil() {
		s += " nil"
	}
	if ti.Untyped() {
		s += " untyped"
	}
	if ti.IsConstant() {
		s += " constant"
	}
	if ti.IsType() {
		s += " isType"
	}
	if ti.IsBuiltin() {
		s += " isBuiltin"
	}
	if ti.Addressable() {
		s += " addressable"
	}
	s += "\n\tValue:"
	if ti.Value != nil {
		switch v := ti.Value.(type) {
		case *ast.Package:
			s += fmt.Sprintf(" %s (package)", v.Name)
		default:
			s += fmt.Sprintf(" %v (%T)", ti.Value, ti.Value)
		}
	}
	return s
}

// bool type infos.
func tiUntypedBoolConst(b bool) *TypeInfo {
	return &TypeInfo{Type: boolType, Value: b, Properties: PropertyUntyped | PropertyIsConstant}
}

func tiBool() *TypeInfo { return &TypeInfo{Type: boolType} }

func tiAddrBool() *TypeInfo {
	return &TypeInfo{Type: boolType, Properties: PropertyAddressable}
}

func tiBoolConst(b bool) *TypeInfo {
	return &TypeInfo{Type: boolType, Value: b, Properties: PropertyIsConstant}
}

func tiUntypedBool() *TypeInfo {
	return &TypeInfo{Type: boolType, Properties: PropertyUntyped}
}

// float type infos.

func tiUntypedFloatConst(lit string) *TypeInfo {
	value, ok := newFloat().SetString(lit)
	if !ok {
		panic("invalid floating-point literal value")
	}
	return &TypeInfo{
		Type:       float64Type,
		Value:      value,
		Properties: PropertyUntyped | PropertyIsConstant,
	}
}

func tiFloat32() *TypeInfo { return &TypeInfo{Type: universe["float32"].t.Type} }
func tiFloat64() *TypeInfo { return &TypeInfo{Type: float64Type} }

func tiAddrFloat32() *TypeInfo {
	return &TypeInfo{Type: universe["float32"].t.Type, Properties: PropertyAddressable}
}

func tiAddrFloat64() *TypeInfo {
	return &TypeInfo{Type: float64Type, Properties: PropertyAddressable}
}

func tiFloat32Const(n float32) *TypeInfo {
	return &TypeInfo{Type: universe["float32"].t.Type, Value: float64(n), Properties: PropertyIsConstant}
}

func tiFloat64Const(n float64) *TypeInfo {
	return &TypeInfo{Type: float64Type, Value: n, Properties: PropertyIsConstant}
}

// rune type infos.

func tiUntypedRuneConst(r rune) *TypeInfo {
	return &TypeInfo{
		Type:       int32Type,
		Value:      (&big.Int{}).SetInt64(int64(r)),
		Properties: PropertyUntyped | PropertyIsConstant,
	}
}

// string type infos.

func tiUntypedStringConst(s string) *TypeInfo {
	return &TypeInfo{
		Type:       stringType,
		Value:      s,
		Properties: PropertyUntyped | PropertyIsConstant,
	}
}

func tiString() *TypeInfo { return &TypeInfo{Type: stringType} }

func tiAddrString() *TypeInfo {
	return &TypeInfo{Type: stringType, Properties: PropertyAddressable}
}

func tiStringConst(s string) *TypeInfo {
	return &TypeInfo{Type: stringType, Value: s, Properties: PropertyIsConstant}
}

// int type infos.

func tiUntypedIntConst(lit string) *TypeInfo {
	value, ok := (&big.Int{}).SetString(lit, 0)
	if !ok {
		panic("invalid integer literal value")
	}
	return &TypeInfo{
		Type:       intType,
		Value:      value,
		Properties: PropertyUntyped | PropertyIsConstant,
	}
}

func tiInt() *TypeInfo    { return &TypeInfo{Type: intType} }
func tiInt8() *TypeInfo   { return &TypeInfo{Type: universe["int8"].t.Type} }
func tiInt16() *TypeInfo  { return &TypeInfo{Type: universe["int16"].t.Type} }
func tiInt32() *TypeInfo  { return &TypeInfo{Type: universe["int32"].t.Type} }
func tiInt64() *TypeInfo  { return &TypeInfo{Type: universe["int64"].t.Type} }
func tiUint() *TypeInfo   { return &TypeInfo{Type: universe["uint"].t.Type} }
func tiUint8() *TypeInfo  { return &TypeInfo{Type: universe["uint8"].t.Type} }
func tiUint16() *TypeInfo { return &TypeInfo{Type: universe["uint16"].t.Type} }
func tiUint32() *TypeInfo { return &TypeInfo{Type: universe["uint32"].t.Type} }
func tiUint64() *TypeInfo { return &TypeInfo{Type: universe["uint64"].t.Type} }

func tiAddrInt() *TypeInfo {
	return &TypeInfo{Type: intType, Properties: PropertyAddressable}
}

func tiAddrInt8() *TypeInfo {
	return &TypeInfo{Type: universe["int8"].t.Type, Properties: PropertyAddressable}
}

func tiAddrInt16() *TypeInfo {
	return &TypeInfo{Type: universe["int16"].t.Type, Properties: PropertyAddressable}
}

func tiAddrInt32() *TypeInfo {
	return &TypeInfo{Type: universe["int32"].t.Type, Properties: PropertyAddressable}
}

func tiAddrInt64() *TypeInfo {
	return &TypeInfo{Type: universe["int64"].t.Type, Properties: PropertyAddressable}
}

func tiAddrUint() *TypeInfo {
	return &TypeInfo{Type: universe["uint"].t.Type, Properties: PropertyAddressable}
}

func tiAddrUint8() *TypeInfo {
	return &TypeInfo{Type: universe["uint8"].t.Type, Properties: PropertyAddressable}
}

func tiAddrUint16() *TypeInfo {
	return &TypeInfo{Type: universe["uint16"].t.Type, Properties: PropertyAddressable}
}

func tiAddrUint32() *TypeInfo {
	return &TypeInfo{Type: universe["uint32"].t.Type, Properties: PropertyAddressable}
}

func tiAddrUint64() *TypeInfo {
	return &TypeInfo{Type: universe["uint64"].t.Type, Properties: PropertyAddressable}
}

func tiIntConst(n int) *TypeInfo {
	return &TypeInfo{Type: intType, Value: big.NewInt(int64(n)), Properties: PropertyIsConstant}
}

func tiInt8Const(n int8) *TypeInfo {
	return &TypeInfo{Type: universe["int8"].t.Type, Value: big.NewInt(int64(n)), Properties: PropertyIsConstant}
}

func tiInt16Const(n int16) *TypeInfo {
	return &TypeInfo{Type: universe["int16"].t.Type, Value: big.NewInt(int64(n)), Properties: PropertyIsConstant}
}

func tiInt32Const(n int32) *TypeInfo {
	return &TypeInfo{Type: universe["int32"].t.Type, Value: big.NewInt(int64(n)), Properties: PropertyIsConstant}
}

func tiInt64Const(n int64) *TypeInfo {
	return &TypeInfo{Type: universe["int64"].t.Type, Value: big.NewInt(n), Properties: PropertyIsConstant}
}

func tiUintConst(n uint) *TypeInfo {
	return &TypeInfo{Type: universe["uint"].t.Type, Value: big.NewInt(0).SetUint64(uint64(n)), Properties: PropertyIsConstant}
}

func tiUint8Const(n uint8) *TypeInfo {
	return &TypeInfo{Type: universe["uint8"].t.Type, Value: big.NewInt(0).SetUint64(uint64(n)), Properties: PropertyIsConstant}
}

func tiUint16Const(n uint16) *TypeInfo {
	return &TypeInfo{Type: universe["uint16"].t.Type, Value: big.NewInt(0).SetUint64(uint64(n)), Properties: PropertyIsConstant}
}

func tiUint32Const(n uint32) *TypeInfo {
	return &TypeInfo{Type: universe["uint32"].t.Type, Value: big.NewInt(0).SetUint64(uint64(n)), Properties: PropertyIsConstant}
}

func tiUint64Const(n uint64) *TypeInfo {
	return &TypeInfo{Type: universe["uint64"].t.Type, Value: big.NewInt(0).SetUint64(n), Properties: PropertyIsConstant}
}

func tiIntPtr() *TypeInfo {
	return &TypeInfo{Type: reflect.PtrTo(intType)}
}

var tiDefinedIntSlice = &TypeInfo{Type: reflect.TypeOf(definedIntSlice{})}

// nil type info.

func tiNil() *TypeInfo { return &TypeInfo{Properties: PropertyNil} }

// byte type info.

func tiByte() *TypeInfo { return &TypeInfo{Type: universe["byte"].t.Type} }

// byte slice type info.

func tiByteSlice() *TypeInfo { return &TypeInfo{Type: reflect.TypeOf([]byte{})} }

// string slice type info.

func tiStringSlice() *TypeInfo { return &TypeInfo{Type: reflect.TypeOf([]string{})} }

// int slice type info.

func tiIntSlice() *TypeInfo { return &TypeInfo{Type: reflect.SliceOf(intType)} }

// string map type info.

func tiStringMap() *TypeInfo { return &TypeInfo{Type: reflect.TypeOf(map[string]string(nil))} }

// int chan type info.

func tiIntChan(dir reflect.ChanDir) *TypeInfo { return &TypeInfo{Type: reflect.ChanOf(dir, intType)} }

// interface{} type info.

func tiInterface() *TypeInfo { return &TypeInfo{Type: universe["interface{}"].t.Type} }

func TestTypechecker_MaxIndex(t *testing.T) {
	cases := map[string]int{
		"[]T{}":              noEllipses,
		"[]T{x}":             0,
		"[]T{x, x}":          1,
		"[]T{4:x}":           4,
		"[]T{3:x, x}":        4,
		"[]T{x, x, x, 9: x}": 9,
		"[]T{x, 9: x, x, x}": 11,
	}
	tc := newTypechecker("", false)
	for src, expected := range cases {
		tree, err := ParseSource([]byte(src), ast.ContextNone)
		if err != nil {
			t.Error(err)
		}
		got := tc.maxIndex(tree.Nodes[0].(*ast.CompositeLiteral))
		if got != expected {
			t.Errorf("src '%s': expected: %v, got: %v", src, expected, got)
		}
	}
}

func TestTypechecker_IsAssignableTo(t *testing.T) {
	stringType := universe["string"].t.Type
	float64Type := universe["float64"].t.Type
	intSliceType := reflect.TypeOf([]int{})
	stringSliceType := reflect.TypeOf([]string{})
	emptyInterfaceType := reflect.TypeOf(&[]interface{}{interface{}(nil)}[0]).Elem()
	weirdInterfaceType := reflect.TypeOf(&[]interface{ F() }{interface{ F() }(nil)}[0]).Elem()
	byteType := reflect.TypeOf(byte(0))
	type myInt int
	myIntType := reflect.TypeOf(myInt(0))
	type myIntSlice []int
	myIntSliceType := reflect.TypeOf(myIntSlice(nil))
	type myIntSlice2 []int
	myIntSliceType2 := reflect.TypeOf(myIntSlice2(nil))
	cases := []struct {
		x          *TypeInfo
		T          reflect.Type
		assignable bool
	}{
		// From https://golang.org/ref/spec#Assignability

		// «x's type is identical to T»
		{x: tiInt(), T: intType, assignable: true},
		{x: tiString(), T: stringType, assignable: true},
		{x: tiFloat64(), T: float64Type, assignable: true},
		{x: tiFloat64(), T: stringType, assignable: false},
		{x: &TypeInfo{Type: myIntType}, T: myIntType, assignable: true},

		// «x's type V and T have identical underlying types and at least one of
		// V or T is not a defined type.»
		{x: &TypeInfo{Type: intSliceType}, T: myIntSliceType, assignable: true},     // x is not a defined type, but T is
		{x: &TypeInfo{Type: myIntSliceType}, T: intSliceType, assignable: true},     // x is a defined type, but T is not
		{x: &TypeInfo{Type: myIntSliceType}, T: myIntSliceType2, assignable: false}, // x and T are both defined types
		{x: &TypeInfo{Type: intSliceType}, T: stringSliceType, assignable: false},   // underlying types are different

		// «T is an interface type and x implements T.»
		{x: tiInt(), T: emptyInterfaceType, assignable: true},
		{x: tiInt(), T: weirdInterfaceType, assignable: false},
		{x: tiString(), T: emptyInterfaceType, assignable: true},
		{x: tiString(), T: weirdInterfaceType, assignable: false},

		// «x is the predeclared identifier nil and T is a pointer, function,
		// slice, map, channel, or interface type»
		{x: tiNil(), T: intSliceType, assignable: true},
		{x: tiNil(), T: emptyInterfaceType, assignable: true},
		{x: tiNil(), T: weirdInterfaceType, assignable: true},
		{x: tiNil(), T: intType, assignable: false},

		// «x is an untyped constant representable by a value of type T.»
		{x: tiUntypedBoolConst(false), T: boolType, assignable: true},
		{x: tiUntypedIntConst("0"), T: boolType, assignable: false},
		{x: tiUntypedIntConst("0"), T: intType, assignable: true},
		{x: tiUntypedIntConst("10"), T: float64Type, assignable: true},
		{x: tiUntypedIntConst("10"), T: byteType, assignable: true},
		// {x: tiUntypedIntConst("300"), T: byteType, assignable: false},
	}
	for _, c := range cases {
		got := isAssignableTo(c.x, c.T)
		if c.assignable && !got {
			t.Errorf("%s should be assignable to %s, but isAssignableTo returned false", c.x, c.T)
		}
		if !c.assignable && got {
			t.Errorf("%s should not be assignable to %s, but isAssignableTo returned true", c.x, c.T)
		}
	}
}

func TestFunctionUpvalues(t *testing.T) {
	cases := map[string][]string{
		`_ = func() { }`:                              nil,           // no variables.
		`a := 1; _ = func() { }`:                      nil,           // a declared outside but not used.
		`a := 1; _ = func() { _ = a }`:                []string{"a"}, // a declared outside and used.
		`_ = func() { a := 1; _ = a }`:                nil,           // a declared inside and used.
		`a := 1; _ = a; _ = func() { a := 1; _ = a }`: nil,           // a declared both outside and inside, used.

		`a, b := 1, 1; _ = a + b; _ = func() { _ = a + b }`:               []string{"a", "b"},
		`a, b := 1, 1; _ = a + b; _ = func() { b := 1; _ = a + b }`:       []string{"a"},
		`a, b := 1, 1; _ = a + b; _ = func() { a, b := 1, 1; _ = a + b }`: nil,
	}
	for src, expected := range cases {
		tc := newTypechecker("", false)
		tc.addScope()
		tree, err := ParseSource([]byte(src), ast.ContextNone)
		if err != nil {
			t.Error(err)
		}
		tc.checkNodes(tree.Nodes)
		got := tree.Nodes[len(tree.Nodes)-1].(*ast.Assignment).Values[0].(*ast.Func).Upvalues
		if len(got) != len(expected) {
			t.Errorf("bad upvalues for src: '%s': expected: %s, got: %s", src, expected, got)
			continue
		}
		for i := range got {
			if got[i] != expected[i] {
				t.Errorf("bad upvalues for src: '%s': expected: %s, got: %s", src, expected, got)
			}
		}
	}
}

func sameTypeCheckError(err1, err2 *Error) error {
	if err1.Err.Error() != err2.Err.Error() {
		return fmt.Errorf("unexpected error %q, expecting error %q\n", err1.Err, err2.Err)
	}
	pos1 := err1.Pos
	pos2 := err2.Pos
	if pos1.Line != pos2.Line {
		return fmt.Errorf("unexpected line %d, expecting %d", pos1.Line, pos2.Line)
	}
	if pos1.Line == 1 {
		if pos1.Column != pos2.Column {
			return fmt.Errorf("unexpected column %d, expecting %d", pos1.Column, pos2.Column)
		}
	} else {
		if pos1.Column != pos2.Column {
			return fmt.Errorf("unexpected column %d, expecting %d", pos1.Column, pos2.Column)
		}
	}
	return nil
}
